package tunnel

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// StreamingExecutor extends the basic executor with streaming capabilities
type StreamingExecutor struct {
	executor Executor
	tracer   SSHTracer
	config   StreamingConfig
	mu       sync.RWMutex
}

// StreamingConfig holds configuration for streaming operations
type StreamingConfig struct {
	BufferSize          int
	FlushInterval       time.Duration
	MaxStreams          int
	EnableLineBuffering bool
	OutputFormat        OutputFormat
	ProgressInterval    time.Duration
}

// OutputFormat defines how streaming output is formatted
type OutputFormat int

const (
	// OutputFormatRaw outputs raw command output
	OutputFormatRaw OutputFormat = iota
	// OutputFormatLines outputs line-by-line with timestamps
	OutputFormatLines
	// OutputFormatJSON outputs structured JSON with metadata
	OutputFormatJSON
)

// StreamResult represents the result of a streaming command
type StreamResult struct {
	Command    string
	StartTime  time.Time
	EndTime    time.Time
	ExitCode   int
	TotalBytes int64
	LineCount  int64
	Error      error
	Metadata   map[string]any
}

// StreamEvent represents an event during streaming
type StreamEvent struct {
	Type      StreamEventType
	Timestamp time.Time
	Data      string
	Metadata  map[string]any
}

// StreamEventType defines types of streaming events
type StreamEventType int

const (
	StreamEventOutput StreamEventType = iota
	StreamEventError
	StreamEventProgress
	StreamEventCompletion
	StreamEventCancellation
)

// DefaultStreamingConfig returns default streaming configuration
func DefaultStreamingConfig() StreamingConfig {
	return StreamingConfig{
		BufferSize:          4096,
		FlushInterval:       100 * time.Millisecond,
		MaxStreams:          5,
		EnableLineBuffering: true,
		OutputFormat:        OutputFormatLines,
		ProgressInterval:    1 * time.Second,
	}
}

// NewStreamingExecutor creates a new streaming executor
func NewStreamingExecutor(executor Executor, tracer SSHTracer) *StreamingExecutor {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &StreamingExecutor{
		executor: executor,
		tracer:   tracer,
		config:   DefaultStreamingConfig(),
	}
}

// NewStreamingExecutorWithConfig creates a streaming executor with custom config
func NewStreamingExecutorWithConfig(executor Executor, tracer SSHTracer, config StreamingConfig) *StreamingExecutor {
	se := NewStreamingExecutor(executor, tracer)
	se.config = config
	return se
}

// StreamCommand executes a command and streams output in real-time
func (se *StreamingExecutor) StreamCommand(ctx context.Context, cmd Command) (<-chan StreamEvent, error) {
	span := se.tracer.TraceCommand(ctx, cmd.Cmd, true)

	span.SetFields(map[string]any{
		"command":           cmd.Cmd,
		"streaming":         true,
		"output_format":     int(se.config.OutputFormat),
		"line_buffering":    se.config.EnableLineBuffering,
		"progress_interval": se.config.ProgressInterval,
	})

	// Validate context and command
	if err := se.validateStreamingContext(ctx, cmd); err != nil {
		span.EndWithError(err)
		return nil, err
	}

	// Get connection key
	connectionKey, ok := GetConnectionKey(ctx)
	if !ok {
		err := fmt.Errorf("connection key not found in context")
		span.EndWithError(err)
		return nil, err
	}

	// Create event channel
	eventCh := make(chan StreamEvent, se.config.BufferSize)

	// Start streaming in goroutine
	go func() {
		defer func() {
			close(eventCh)
			span.End()
		}()

		result := se.executeStreamingCommand(ctx, connectionKey, cmd, eventCh)

		// Send completion event
		eventCh <- StreamEvent{
			Type:      StreamEventCompletion,
			Timestamp: time.Now(),
			Metadata: map[string]any{
				"exit_code":   result.ExitCode,
				"total_bytes": result.TotalBytes,
				"line_count":  result.LineCount,
				"duration":    result.EndTime.Sub(result.StartTime),
				"success":     result.Error == nil,
			},
		}

		if result.Error != nil {
			span.EndWithError(result.Error)
		} else {
			span.Event("streaming_completed", map[string]any{
				"total_bytes": result.TotalBytes,
				"line_count":  result.LineCount,
				"duration":    result.EndTime.Sub(result.StartTime),
			})
		}
	}()

	span.Event("streaming_started", map[string]any{
		"command": cmd.Cmd,
	})

	return eventCh, nil
}

// StreamScript executes a script and streams output
func (se *StreamingExecutor) StreamScript(ctx context.Context, script Script) (<-chan StreamEvent, error) {
	span := se.tracer.TraceCommand(ctx, "script_streaming", true)
	defer span.End()

	// Convert script to command
	cmd, err := se.prepareScriptCommand(script)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to prepare script command: %w", err)
	}

	span.Event("script_prepared_for_streaming", map[string]any{
		"interpreter": script.Interpreter,
		"script_size": len(script.Content),
		"has_args":    len(script.Args) > 0,
	})

	return se.StreamCommand(ctx, cmd)
}

// StreamMultipleCommands executes multiple commands sequentially with streaming
func (se *StreamingExecutor) StreamMultipleCommands(ctx context.Context, commands []Command) (<-chan StreamEvent, error) {
	span := se.tracer.TraceCommand(ctx, "multi_command_streaming", true)

	eventCh := make(chan StreamEvent, se.config.BufferSize)

	go func() {
		defer func() {
			close(eventCh)
			span.End()
		}()

		totalCommands := len(commands)
		span.SetFields(map[string]any{
			"total_commands": totalCommands,
			"streaming":      true,
		})

		for i, cmd := range commands {
			// Send progress event
			eventCh <- StreamEvent{
				Type:      StreamEventProgress,
				Timestamp: time.Now(),
				Metadata: map[string]any{
					"command_index":   i,
					"total_commands":  totalCommands,
					"progress_pct":    int((float64(i) / float64(totalCommands)) * 100),
					"current_command": cmd.Cmd,
				},
			}

			// Execute command
			commandEventCh, err := se.StreamCommand(ctx, cmd)
			if err != nil {
				eventCh <- StreamEvent{
					Type:      StreamEventError,
					Timestamp: time.Now(),
					Data:      fmt.Sprintf("Failed to start command %d: %v", i+1, err),
					Metadata: map[string]any{
						"command_index": i,
						"command":       cmd.Cmd,
						"error":         err.Error(),
					},
				}
				span.EndWithError(err)
				return
			}

			// Forward events from command
			for event := range commandEventCh {
				// Add command context to events
				if event.Metadata == nil {
					event.Metadata = make(map[string]any)
				}
				event.Metadata["command_index"] = i
				event.Metadata["total_commands"] = totalCommands

				eventCh <- event

				// Stop on error if needed
				if event.Type == StreamEventCompletion {
					if exitCode, ok := event.Metadata["exit_code"].(int); ok && exitCode != 0 {
						eventCh <- StreamEvent{
							Type:      StreamEventError,
							Timestamp: time.Now(),
							Data:      fmt.Sprintf("Command %d failed with exit code %d", i+1, exitCode),
							Metadata: map[string]any{
								"command_index": i,
								"exit_code":     exitCode,
								"command":       cmd.Cmd,
							},
						}
						return
					}
				}
			}
		}

		span.Event("multi_command_completed", map[string]any{
			"commands_executed": totalCommands,
		})
	}()

	return eventCh, nil
}

// executeStreamingCommand executes a single command with streaming
func (se *StreamingExecutor) executeStreamingCommand(ctx context.Context, connectionKey string, cmd Command, eventCh chan<- StreamEvent) StreamResult {
	result := StreamResult{
		Command:   cmd.Cmd,
		StartTime: time.Now(),
		Metadata:  make(map[string]any),
	}

	// Get connection from pool
	pool, ok := se.executor.(*executor)
	if !ok {
		result.Error = fmt.Errorf("unsupported executor type")
		result.EndTime = time.Now()
		return result
	}

	client, err := pool.pool.Get(ctx, connectionKey)
	if err != nil {
		result.Error = fmt.Errorf("failed to get connection: %w", err)
		result.EndTime = time.Now()
		return result
	}
	defer pool.pool.Release(connectionKey, client)

	// Create streaming session
	outputCh, err := client.ExecuteStream(ctx, cmd.Cmd)
	if err != nil {
		result.Error = fmt.Errorf("failed to start streaming: %w", err)
		result.EndTime = time.Now()
		return result
	}

	// Process streaming output
	se.processStreamingOutput(ctx, outputCh, eventCh, &result)

	result.EndTime = time.Now()
	return result
}

// processStreamingOutput processes the streaming output and sends events
func (se *StreamingExecutor) processStreamingOutput(ctx context.Context, outputCh <-chan string, eventCh chan<- StreamEvent, result *StreamResult) {
	var lineBuffer strings.Builder
	lastProgressTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			// Context canceled
			eventCh <- StreamEvent{
				Type:      StreamEventCancellation,
				Timestamp: time.Now(),
				Data:      "Command execution canceled",
				Metadata: map[string]any{
					"reason": ctx.Err().Error(),
				},
			}
			result.Error = ctx.Err()
			return

		case output, ok := <-outputCh:
			if !ok {
				// Channel closed, command completed
				return
			}

			result.TotalBytes += int64(len(output))

			if se.config.EnableLineBuffering {
				se.processLineBufferedOutput(output, &lineBuffer, eventCh, result)
			} else {
				// Send raw output
				eventCh <- StreamEvent{
					Type:      StreamEventOutput,
					Timestamp: time.Now(),
					Data:      output,
					Metadata: map[string]any{
						"bytes": len(output),
						"raw":   true,
					},
				}
			}

			// Send progress updates
			now := time.Now()
			if now.Sub(lastProgressTime) >= se.config.ProgressInterval {
				eventCh <- StreamEvent{
					Type:      StreamEventProgress,
					Timestamp: now,
					Metadata: map[string]any{
						"bytes_received": result.TotalBytes,
						"lines_received": result.LineCount,
						"duration":       now.Sub(result.StartTime),
					},
				}
				lastProgressTime = now
			}
		}
	}
}

// processLineBufferedOutput processes output line by line
func (se *StreamingExecutor) processLineBufferedOutput(output string, lineBuffer *strings.Builder, eventCh chan<- StreamEvent, result *StreamResult) {
	lineBuffer.WriteString(output)
	content := lineBuffer.String()

	// Split by newlines
	lines := strings.Split(content, "\n")

	// Process complete lines (all but the last)
	for i := 0; i < len(lines)-1; i++ {
		line := lines[i]
		result.LineCount++

		event := StreamEvent{
			Type:      StreamEventOutput,
			Timestamp: time.Now(),
			Data:      line,
			Metadata: map[string]any{
				"line_number": result.LineCount,
				"bytes":       len(line),
				"complete":    true,
			},
		}

		if se.config.OutputFormat == OutputFormatJSON {
			event.Metadata["formatted"] = true
		}

		eventCh <- event
	}

	// Keep the last partial line in buffer
	lineBuffer.Reset()
	if len(lines) > 0 {
		lineBuffer.WriteString(lines[len(lines)-1])
	}
}

// prepareScriptCommand converts a script to a command suitable for streaming
func (se *StreamingExecutor) prepareScriptCommand(script Script) (Command, error) {
	if script.Interpreter == "" {
		script.Interpreter = "/bin/bash"
	}

	// Build command arguments
	args := strings.Join(script.Args, " ")

	// Create command that pipes script content to interpreter
	var cmdParts []string

	if args != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("cat << 'EOF' | %s %s", script.Interpreter, args))
	} else {
		cmdParts = append(cmdParts, fmt.Sprintf("cat << 'EOF' | %s", script.Interpreter))
	}

	cmdParts = append(cmdParts, script.Content)
	cmdParts = append(cmdParts, "EOF")

	cmdString := strings.Join(cmdParts, "\n")

	return Command{
		Cmd:         cmdString,
		Timeout:     script.Timeout,
		Environment: script.Environment,
	}, nil
}

// validateStreamingContext validates context for streaming operations
func (se *StreamingExecutor) validateStreamingContext(ctx context.Context, cmd Command) error {
	if _, ok := GetConnectionKey(ctx); !ok {
		return fmt.Errorf("connection key not found in context")
	}

	if cmd.Cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	return nil
}

// SetConfig updates the streaming configuration
func (se *StreamingExecutor) SetConfig(config StreamingConfig) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.config = config
}

// GetConfig returns the current streaming configuration
func (se *StreamingExecutor) GetConfig() StreamingConfig {
	se.mu.RLock()
	defer se.mu.RUnlock()
	return se.config
}

// CollectStreamOutput collects all streaming output into a single result
func (se *StreamingExecutor) CollectStreamOutput(ctx context.Context, cmd Command) (*Result, error) {
	eventCh, err := se.StreamCommand(ctx, cmd)
	if err != nil {
		return nil, err
	}

	var output strings.Builder
	var result *Result
	startTime := time.Now()

	for event := range eventCh {
		switch event.Type {
		case StreamEventOutput:
			output.WriteString(event.Data)
			if se.config.EnableLineBuffering {
				output.WriteString("\n")
			}

		case StreamEventCompletion:
			exitCode := 0
			if code, ok := event.Metadata["exit_code"].(int); ok {
				exitCode = code
			}

			result = &Result{
				Output:   output.String(),
				ExitCode: exitCode,
				Duration: time.Since(startTime),
				Started:  startTime,
				Finished: time.Now(),
			}

			if !event.Metadata["success"].(bool) {
				result.Error = fmt.Errorf("command failed with exit code %d", exitCode)
			}

		case StreamEventError:
			if result == nil {
				result = &Result{
					Output:   output.String(),
					ExitCode: 1,
					Duration: time.Since(startTime),
					Started:  startTime,
					Finished: time.Now(),
					Error:    fmt.Errorf("streaming error: %s", event.Data),
				}
			}

		case StreamEventCancellation:
			return &Result{
				Output:   output.String(),
				ExitCode: 130, // SIGINT
				Duration: time.Since(startTime),
				Started:  startTime,
				Finished: time.Now(),
				Error:    ctx.Err(),
			}, ctx.Err()
		}
	}

	if result == nil {
		return &Result{
			Output:   output.String(),
			ExitCode: 0,
			Duration: time.Since(startTime),
			Started:  startTime,
			Finished: time.Now(),
		}, nil
	}

	return result, result.Error
}

// StreamToWriter streams command output directly to an io.Writer
func (se *StreamingExecutor) StreamToWriter(ctx context.Context, cmd Command, writer io.Writer) error {
	eventCh, err := se.StreamCommand(ctx, cmd)
	if err != nil {
		return err
	}

	for event := range eventCh {
		switch event.Type {
		case StreamEventOutput:
			_, err := writer.Write([]byte(event.Data))
			if err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}

			if se.config.EnableLineBuffering {
				writer.Write([]byte("\n"))
			}

		case StreamEventError:
			return fmt.Errorf("streaming error: %s", event.Data)

		case StreamEventCompletion:
			if !event.Metadata["success"].(bool) {
				exitCode := event.Metadata["exit_code"].(int)
				return fmt.Errorf("command failed with exit code %d", exitCode)
			}
			return nil

		case StreamEventCancellation:
			return ctx.Err()
		}
	}

	return nil
}
