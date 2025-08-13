package tunnel

import (
	"context"
	"fmt"
	"io"
	"pb-deployer/internal/utils"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Session represents an SSH session with advanced execution capabilities
type Session struct {
	client      SSHClient
	session     *ssh.Session
	tracer      SSHTracer
	config      ExtendedSessionConfig
	mu          sync.RWMutex
	started     bool
	finished    bool
	startTime   time.Time
	endTime     time.Time
	exitCode    int
	output      strings.Builder
	errorOutput strings.Builder
}

// NewSession creates a new SSH session
func NewSession(client SSHClient, tracer SSHTracer) (*Session, error) {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &Session{
		client: client,
		tracer: tracer,
		config: *DefaultExtendedSessionConfig(),
	}, nil
}

// SetConfig sets the session configuration
func (s *Session) SetConfig(config ExtendedSessionConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// Run executes a command and waits for completion
func (s *Session) Run(ctx context.Context, cmd string) (*Result, error) {
	span := s.tracer.TraceCommand(ctx, cmd, false)
	defer span.End()

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		err := fmt.Errorf("session already started")
		span.EndWithError(err)
		return nil, err
	}
	s.mu.Unlock()

	if err := s.Start(ctx, cmd); err != nil {
		span.EndWithError(err)
		return nil, err
	}

	result, err := s.Wait(ctx)
	if err != nil {
		span.EndWithError(err)
		return nil, err
	}

	span.Event("command_completed", map[string]any{
		"exit_code": result.ExitCode,
		"duration":  result.Duration,
	})

	return result, nil
}

// Start begins command execution without waiting
func (s *Session) Start(ctx context.Context, cmd string) error {
	span := s.tracer.TraceCommand(ctx, cmd, false)
	defer span.End()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		err := fmt.Errorf("session already started")
		span.EndWithError(err)
		return err
	}

	// Get underlying SSH connection
	sshClient, ok := s.client.(*sshClient)
	if !ok {
		err := fmt.Errorf("unsupported client type")
		span.EndWithError(err)
		return err
	}

	sshClient.mu.RLock()
	conn := sshClient.conn
	sshClient.mu.RUnlock()

	if conn == nil {
		err := ErrClientNotConnected
		span.EndWithError(err)
		return err
	}

	// Create SSH session
	session, err := conn.NewSession()
	if err != nil {
		span.EndWithError(err)
		return WrapCommandError(cmd, 0, "", err)
	}

	s.session = session

	// Configure session environment
	if err := s.configureSession(); err != nil {
		s.session.Close()
		s.session = nil
		span.EndWithError(err)
		return err
	}

	// Set up output capturing
	if err := s.setupOutputCapture(); err != nil {
		s.session.Close()
		s.session = nil
		span.EndWithError(err)
		return err
	}

	// Prepare command with modifications
	finalCmd := s.prepareCommand(cmd)

	span.SetFields(map[string]any{
		"original_command": cmd,
		"final_command":    finalCmd,
		"environment_vars": len(s.config.Environment),
		"use_pty":          s.config.PTY,
	})

	// Start command execution
	s.startTime = time.Now()
	if err := s.session.Start(finalCmd); err != nil {
		s.session.Close()
		s.session = nil
		span.EndWithError(err)
		return WrapCommandError(cmd, 0, "", err)
	}

	s.started = true

	span.Event("command_started", map[string]any{
		"start_time": s.startTime,
	})

	return nil
}

// Wait waits for the command to complete and returns the result
func (s *Session) Wait(ctx context.Context) (*Result, error) {
	s.mu.RLock()
	if !s.started || s.session == nil {
		s.mu.RUnlock()
		return nil, fmt.Errorf("session not started")
	}
	session := s.session
	s.mu.RUnlock()

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- session.Wait()
	}()

	var waitErr error
	select {
	case <-ctx.Done():
		// Context canceled, try to kill the session
		s.Kill()
		waitErr = ctx.Err()
	case waitErr = <-done:
		// Command completed normally
	case <-time.After(s.config.Timeout):
		// Timeout reached
		s.Kill()
		waitErr = ErrTimeout
	}

	s.mu.Lock()
	s.finished = true
	s.endTime = time.Now()
	s.mu.Unlock()

	// Extract exit code
	if exitErr, ok := waitErr.(*ssh.ExitError); ok {
		s.exitCode = exitErr.ExitStatus()
		waitErr = nil // Non-zero exit codes are not errors in this context
	}

	// Create result
	result := &Result{
		Output:   s.output.String(),
		Error:    waitErr,
		ExitCode: s.exitCode,
		Duration: s.endTime.Sub(s.startTime),
		Started:  s.startTime,
		Finished: s.endTime,
	}

	// Include stderr in output if there were errors
	if errorOutput := s.errorOutput.String(); errorOutput != "" {
		if result.Output != "" {
			result.Output += "\n" + errorOutput
		} else {
			result.Output = errorOutput
		}
	}

	return result, waitErr
}

// Kill terminates the running command
func (s *Session) Kill() error {
	s.mu.RLock()
	session := s.session
	s.mu.RUnlock()

	if session == nil {
		return fmt.Errorf("no active session")
	}

	// Try graceful termination first
	if err := session.Signal(ssh.SIGTERM); err != nil {
		// If graceful termination fails, force kill
		session.Signal(ssh.SIGKILL)
	}

	return nil
}

// Close closes the session and cleans up resources
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session != nil {
		err := s.session.Close()
		s.session = nil
		return err
	}

	return nil
}

// configureSession configures the SSH session with environment and PTY settings
func (s *Session) configureSession() error {
	// Set environment variables
	for key, value := range s.config.Environment {
		if err := s.session.Setenv(key, value); err != nil {
			// Some SSH servers don't allow setting environment variables
			// Log the error but don't fail the session
			continue
		}
	}

	// Configure PTY if requested
	if s.config.PTY {
		ptyConfig := s.config.PTYConfig
		if ptyConfig == nil {
			ptyConfig = DefaultPTYConfig()
		}

		if err := s.session.RequestPty(ptyConfig.Term, ptyConfig.Height, ptyConfig.Width, ptyConfig.Modes); err != nil {
			return fmt.Errorf("failed to request PTY: %w", err)
		}
	}

	return nil
}

// setupOutputCapture sets up output capturing for the session
func (s *Session) setupOutputCapture() error {
	// Capture stdout
	stdout, err := s.session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Capture stderr
	stderr, err := s.session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start output readers
	go s.readOutput(stdout, &s.output)
	go s.readOutput(stderr, &s.errorOutput)

	return nil
}

// readOutput reads from a pipe and writes to a string builder
func (s *Session) readOutput(reader io.Reader, builder *strings.Builder) {
	buffer := make([]byte, DefaultBufferSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			s.mu.Lock()
			builder.Write(buffer[:n])
			s.mu.Unlock()
		}
		if err != nil {
			break
		}
	}
}

// prepareCommand prepares the command with modifications (sudo, working directory, etc.)
func (s *Session) prepareCommand(cmd string) string {
	var parts []string

	// Change working directory if specified
	if s.config.WorkingDir != "" {
		parts = append(parts, fmt.Sprintf("cd %s", utils.ShellEscape(s.config.WorkingDir)))
	}

	// Add environment variables that couldn't be set via Setenv
	for key, value := range s.config.Environment {
		parts = append(parts, fmt.Sprintf("export %s=%s", key, utils.ShellEscape(value)))
	}

	// Add the actual command
	parts = append(parts, cmd)

	// Join with && to ensure they execute in sequence
	finalCmd := strings.Join(parts, " && ")

	// Wrap with sudo if needed
	if s.config.Sudo {
		if s.config.SudoUser != "" {
			finalCmd = fmt.Sprintf("sudo -u %s %s", utils.ShellEscape(s.config.SudoUser), finalCmd)
		} else {
			finalCmd = fmt.Sprintf("sudo %s", finalCmd)
		}
	}

	return finalCmd
}

// ExecutorSession provides high-level command execution with session management
type ExecutorSession struct {
	client   SSHClient
	tracer   SSHTracer
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewExecutorSession creates a new executor session
func NewExecutorSession(client SSHClient, tracer SSHTracer) *ExecutorSession {
	return &ExecutorSession{
		client:   client,
		tracer:   tracer,
		sessions: make(map[string]*Session),
	}
}

// ExecuteCommand executes a command with the given execution options
func (es *ExecutorSession) ExecuteCommand(ctx context.Context, cmd string, opts *ExecutionOptions) (*Result, error) {
	span := es.tracer.TraceCommand(ctx, cmd, false)
	defer span.End()

	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	// Create session
	session, err := NewSession(es.client, es.tracer)
	if err != nil {
		span.EndWithError(err)
		return nil, err
	}
	defer session.Close()

	// Configure session
	sessionConfig := ExtendedSessionConfig{
		SessionConfig: SessionConfig{
			Timeout:     opts.Timeout,
			Environment: opts.Environment,
			PTY:         false, // Usually not needed for command execution
		},
		WorkingDir: opts.WorkingDir,
		Sudo:       opts.Sudo,
		SudoUser:   opts.SudoUser,
	}
	session.SetConfig(sessionConfig)

	span.SetFields(map[string]any{
		"command":     cmd,
		"sudo":        opts.Sudo,
		"timeout":     opts.Timeout,
		"working_dir": opts.WorkingDir,
	})

	// Execute command
	result, err := session.Run(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return nil, WrapCommandError(cmd, 0, "", err)
	}

	span.Event("execution_completed", map[string]any{
		"exit_code":   result.ExitCode,
		"duration":    result.Duration,
		"output_size": len(result.Output),
	})

	return result, nil
}

// ExecuteScript executes a script with the given execution options
func (es *ExecutorSession) ExecuteScript(ctx context.Context, script string, interpreter string, opts *ExecutionOptions) (*Result, error) {
	span := es.tracer.TraceCommand(ctx, "script_execution", false)
	defer span.End()

	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	if interpreter == "" {
		interpreter = "/bin/bash"
	}

	// Create a command that pipes the script to the interpreter
	cmd := fmt.Sprintf("cat << 'EOF' | %s\n%s\nEOF", interpreter, script)

	span.SetFields(map[string]any{
		"interpreter": interpreter,
		"script_size": len(script),
		"sudo":        opts.Sudo,
		"timeout":     opts.Timeout,
		"working_dir": opts.WorkingDir,
	})

	return es.ExecuteCommand(ctx, cmd, opts)
}

// ExecuteCommandStream executes a command and streams output
func (es *ExecutorSession) ExecuteCommandStream(ctx context.Context, cmd string, opts *ExecutionOptions) (<-chan string, error) {
	span := es.tracer.TraceCommand(ctx, cmd, true)

	if opts == nil {
		opts = DefaultExecutionOptions()
	}

	// For streaming, we need to use the client's ExecuteStream method
	outputCh, err := es.client.ExecuteStream(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return nil, err
	}

	span.Event("stream_started", map[string]any{
		"command": cmd,
	})

	// Wrap the output channel to add span completion
	wrappedCh := make(chan string, cap(outputCh))
	go func() {
		defer func() {
			close(wrappedCh)
			span.End()
		}()

		for output := range outputCh {
			select {
			case wrappedCh <- output:
			case <-ctx.Done():
				span.Event("stream_canceled")
				return
			}
		}

		span.Event("stream_completed")
	}()

	return wrappedCh, nil
}

// CreateNamedSession creates a named session for reuse
func (es *ExecutorSession) CreateNamedSession(name string) (*Session, error) {
	es.mu.Lock()
	defer es.mu.Unlock()

	if _, exists := es.sessions[name]; exists {
		return nil, fmt.Errorf("session '%s' already exists", name)
	}

	session, err := NewSession(es.client, es.tracer)
	if err != nil {
		return nil, err
	}

	es.sessions[name] = session
	return session, nil
}

// GetNamedSession retrieves a named session
func (es *ExecutorSession) GetNamedSession(name string) (*Session, bool) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	session, exists := es.sessions[name]
	return session, exists
}

// CloseNamedSession closes and removes a named session
func (es *ExecutorSession) CloseNamedSession(name string) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	session, exists := es.sessions[name]
	if !exists {
		return fmt.Errorf("session '%s' not found", name)
	}

	err := session.Close()
	delete(es.sessions, name)
	return err
}

// CloseAllSessions closes all named sessions
func (es *ExecutorSession) CloseAllSessions() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	var errors []error
	for name, session := range es.sessions {
		if err := session.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close session '%s': %w", name, err))
		}
	}

	es.sessions = make(map[string]*Session)

	if len(errors) > 0 {
		return fmt.Errorf("errors closing sessions: %v", errors)
	}

	return nil
}

// ExtendedSessionConfig extends the base SessionConfig with additional execution options
type ExtendedSessionConfig struct {
	SessionConfig
	WorkingDir string
	Sudo       bool
	SudoUser   string
}

// DefaultExtendedSessionConfig returns default extended session configuration
func DefaultExtendedSessionConfig() *ExtendedSessionConfig {
	return &ExtendedSessionConfig{
		SessionConfig: SessionConfig{
			Timeout:     DefaultCommandTimeout,
			Environment: make(map[string]string),
			PTY:         false,
		},
		WorkingDir: "",
		Sudo:       false,
		SudoUser:   "",
	}
}
