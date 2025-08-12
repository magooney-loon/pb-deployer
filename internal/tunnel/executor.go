package tunnel

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// executor implements the Executor interface with dependency injection
type executor struct {
	pool   Pool
	tracer SSHTracer
	config ExecutorConfig
	mu     sync.RWMutex
}

// ExecutorConfig holds configuration for the executor
type ExecutorConfig struct {
	DefaultTimeout     time.Duration
	MaxConcurrentCmds  int
	RetryStrategy      RetryStrategy
	CommandWhitelist   []string
	CommandBlacklist   []string
	EnableAuditLogging bool
	DefaultWorkingDir  string
	DefaultEnvironment map[string]string
	BufferSize         int
	EnableSudoPassword bool
	SudoPasswordPrompt string
}

// DefaultExecutorConfig returns default executor configuration
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		DefaultTimeout:     DefaultCommandTimeout,
		MaxConcurrentCmds:  10,
		RetryStrategy:      DefaultRetryStrategy(),
		CommandWhitelist:   []string{},
		CommandBlacklist:   []string{"rm -rf /", "mkfs", "fdisk"},
		EnableAuditLogging: true,
		DefaultWorkingDir:  "",
		DefaultEnvironment: make(map[string]string),
		BufferSize:         DefaultBufferSize,
		EnableSudoPassword: false,
		SudoPasswordPrompt: "[sudo] password for",
	}
}

// NewExecutor creates a new executor with dependency injection
func NewExecutor(pool Pool, tracer SSHTracer) Executor {
	if pool == nil {
		panic("pool cannot be nil")
	}

	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &executor{
		pool:   pool,
		tracer: tracer,
		config: DefaultExecutorConfig(),
	}
}

// NewExecutorWithConfig creates a new executor with custom configuration
func NewExecutorWithConfig(pool Pool, tracer SSHTracer, config ExecutorConfig) Executor {
	if pool == nil {
		panic("pool cannot be nil")
	}

	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	executor := &executor{
		pool:   pool,
		tracer: tracer,
		config: config,
	}

	// Validate configuration
	if err := executor.validateConfig(); err != nil {
		panic(fmt.Sprintf("invalid executor config: %v", err))
	}

	return executor
}

// RunCommand executes a command with options
func (e *executor) RunCommand(ctx context.Context, cmd Command) (*Result, error) {
	span := e.tracer.TraceCommand(ctx, cmd.Cmd, false)
	defer span.End()

	span.SetFields(map[string]any{
		"command":     cmd.Cmd,
		"sudo":        cmd.Sudo,
		"timeout":     cmd.Timeout,
		"working_dir": cmd.WorkingDir,
		"user":        cmd.User,
	})

	// Validate command
	if err := e.validateCommand(cmd); err != nil {
		span.EndWithError(err)
		return nil, WrapCommandError(cmd.Cmd, 0, "", err)
	}

	// Validate execution context
	if err := e.validateExecutionContext(ctx); err != nil {
		span.EndWithError(err)
		return nil, err
	}

	// Set defaults
	cmd = e.setCommandDefaults(cmd)

	// Get connection key from context
	connectionKey, ok := GetConnectionKey(ctx)
	if !ok {
		err := fmt.Errorf("no connection key available in context")
		span.EndWithError(err)
		return nil, err
	}

	// Execute with retry logic
	var result *Result
	var lastErr error

	for attempt := 1; attempt <= e.config.RetryStrategy.MaxAttempts; attempt++ {
		span.Event("execution_attempt", map[string]any{
			"attempt":      attempt,
			"max_attempts": e.config.RetryStrategy.MaxAttempts,
		})

		result, lastErr = e.executeCommand(ctx, connectionKey, cmd)

		if lastErr == nil {
			span.Event("execution_success", map[string]any{
				"attempt":   attempt,
				"exit_code": result.ExitCode,
				"duration":  result.Duration,
			})
			return result, nil
		}

		// Check if we should retry
		if !e.config.RetryStrategy.ShouldRetry(attempt, lastErr) {
			break
		}

		// Don't sleep on the last attempt
		if attempt < e.config.RetryStrategy.MaxAttempts {
			delay := e.config.RetryStrategy.CalculateBackoff(attempt)
			span.Event("execution_retry", map[string]any{
				"attempt": attempt,
				"delay":   delay,
				"error":   lastErr.Error(),
			})

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				span.EndWithError(ctx.Err())
				return nil, ctx.Err()
			}
		}
	}

	span.EndWithError(lastErr)
	return result, lastErr
}

// RunScript executes a script on the remote server
func (e *executor) RunScript(ctx context.Context, script Script) (*Result, error) {
	span := e.tracer.TraceCommand(ctx, "script_execution", false)
	defer span.End()

	span.SetFields(map[string]any{
		"interpreter":  script.Interpreter,
		"script_size":  len(script.Content),
		"args_count":   len(script.Args),
		"timeout":      script.Timeout,
		"has_env_vars": len(script.Environment) > 0,
	})

	// Validate script
	if err := e.validateScript(script); err != nil {
		span.EndWithError(err)
		return nil, err
	}

	// Set defaults
	script = e.setScriptDefaults(script)

	// Prepare script command
	cmd, err := e.prepareScriptCommand(script)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to prepare script command: %w", err)
	}

	span.Event("script_prepared", map[string]any{
		"final_command": cmd.Cmd,
		"interpreter":   script.Interpreter,
	})

	// Execute the script command
	return e.RunCommand(ctx, cmd)
}

// TransferFile transfers a file to/from the remote server
func (e *executor) TransferFile(ctx context.Context, transfer Transfer) error {
	span := e.tracer.TraceCommand(ctx, "file_transfer", false)
	defer span.End()

	span.SetFields(map[string]any{
		"source":      transfer.Source,
		"destination": transfer.Destination,
		"direction":   int(transfer.Direction),
		"permissions": transfer.Permissions,
		"owner":       transfer.Owner,
		"group":       transfer.Group,
	})

	// Validate transfer
	if err := e.validateTransfer(transfer); err != nil {
		span.EndWithError(err)
		return err
	}

	// Validate execution context
	if err := e.validateExecutionContext(ctx); err != nil {
		span.EndWithError(err)
		return err
	}

	// Get connection key from context
	connectionKey, ok := GetConnectionKey(ctx)
	if !ok {
		err := fmt.Errorf("no connection key available in context")
		span.EndWithError(err)
		return err
	}

	// Execute transfer
	err := e.executeTransfer(ctx, connectionKey, transfer)
	if err != nil {
		span.EndWithError(err)
		return err
	}

	span.Event("transfer_completed")
	return nil
}

// executeCommand executes a single command
func (e *executor) executeCommand(ctx context.Context, connectionKey string, cmd Command) (*Result, error) {
	// Get connection from pool
	client, err := e.pool.Get(ctx, connectionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer e.pool.Release(connectionKey, client)

	// Create execution session
	session, err := NewSession(client, e.tracer)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Configure session
	sessionConfig := e.createSessionConfig(cmd)
	session.SetConfig(sessionConfig)

	// Execute command
	startTime := time.Now()
	result, err := session.Run(ctx, cmd.Cmd)
	if err != nil {
		return nil, err
	}

	// Add execution metadata
	if result != nil {
		result.Started = startTime
		result.Finished = time.Now()
		result.Duration = result.Finished.Sub(result.Started)
	}

	return result, nil
}

// executeTransfer executes a file transfer operation
func (e *executor) executeTransfer(ctx context.Context, connectionKey string, transfer Transfer) error {
	// Get connection from pool
	client, err := e.pool.Get(ctx, connectionKey)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer e.pool.Release(connectionKey, client)

	// Get underlying SSH connection
	sshClient, ok := client.(*sshClient)
	if !ok {
		return fmt.Errorf("unsupported client type for file transfer")
	}

	sshClient.mu.RLock()
	conn := sshClient.conn
	sshClient.mu.RUnlock()

	if conn == nil {
		return ErrClientNotConnected
	}

	switch transfer.Direction {
	case TransferUpload:
		return e.uploadFile(ctx, conn, transfer)
	case TransferDownload:
		return e.downloadFile(ctx, conn, transfer)
	default:
		return fmt.Errorf("unsupported transfer direction: %d", transfer.Direction)
	}
}

// uploadFile uploads a file to the remote server using SCP
func (e *executor) uploadFile(ctx context.Context, conn *ssh.Client, transfer Transfer) error {
	// Read local file
	localFile, err := os.Open(transfer.Source)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Get file info
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create SCP session
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SCP session: %w", err)
	}
	defer session.Close()

	// Set up SCP command
	scpCmd := fmt.Sprintf("scp -t %s", shellEscape(transfer.Destination))

	// Get session pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	defer stdin.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start SCP
	if err := session.Start(scpCmd); err != nil {
		return fmt.Errorf("failed to start SCP: %w", err)
	}

	// SCP protocol implementation
	go func() {
		defer stdin.Close()

		// Send file header
		header := fmt.Sprintf("C0644 %d %s\n", fileInfo.Size(), filepath.Base(transfer.Destination))
		stdin.Write([]byte(header))

		// Copy file content
		io.Copy(stdin, localFile)

		// Send termination
		stdin.Write([]byte{0})
	}()

	// Read any errors
	buffer := make([]byte, 1024)
	stdout.Read(buffer)

	// Wait for completion
	return session.Wait()
}

// downloadFile downloads a file from the remote server using SCP
func (e *executor) downloadFile(ctx context.Context, conn *ssh.Client, transfer Transfer) error {
	// Create local file
	localFile, err := os.Create(transfer.Destination)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Create SCP session
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SCP session: %w", err)
	}
	defer session.Close()

	// Set up SCP command
	scpCmd := fmt.Sprintf("scp -f %s", shellEscape(transfer.Source))

	// Get session pipes
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	defer stdin.Close()

	// Start SCP
	if err := session.Start(scpCmd); err != nil {
		return fmt.Errorf("failed to start SCP: %w", err)
	}

	// Send ready signal
	stdin.Write([]byte{0})

	// Read file content (simplified SCP protocol)
	io.Copy(localFile, stdout)

	// Wait for completion
	return session.Wait()
}

// validateCommand validates a command
func (e *executor) validateCommand(cmd Command) error {
	if cmd.Cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check blacklist
	for _, blocked := range e.config.CommandBlacklist {
		if strings.Contains(cmd.Cmd, blocked) {
			return fmt.Errorf("command contains blocked pattern: %s", blocked)
		}
	}

	// Check whitelist if configured
	if len(e.config.CommandWhitelist) > 0 {
		allowed := false
		for _, pattern := range e.config.CommandWhitelist {
			if strings.Contains(cmd.Cmd, pattern) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command not in whitelist")
		}
	}

	// Validate timeout
	if cmd.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	return nil
}

// validateScript validates a script
func (e *executor) validateScript(script Script) error {
	if script.Content == "" {
		return fmt.Errorf("script content cannot be empty")
	}

	if script.Interpreter == "" {
		return fmt.Errorf("script interpreter cannot be empty")
	}

	if script.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	return nil
}

// validateTransfer validates a transfer operation
func (e *executor) validateTransfer(transfer Transfer) error {
	if transfer.Source == "" {
		return fmt.Errorf("source path cannot be empty")
	}

	if transfer.Destination == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	if transfer.Direction != TransferUpload && transfer.Direction != TransferDownload {
		return fmt.Errorf("invalid transfer direction")
	}

	return nil
}

// setCommandDefaults sets default values for command
func (e *executor) setCommandDefaults(cmd Command) Command {
	if cmd.Timeout == 0 {
		cmd.Timeout = e.config.DefaultTimeout
	}

	if cmd.Environment == nil {
		cmd.Environment = make(map[string]string)
	}

	// Add default environment variables
	for k, v := range e.config.DefaultEnvironment {
		if _, exists := cmd.Environment[k]; !exists {
			cmd.Environment[k] = v
		}
	}

	if cmd.WorkingDir == "" && e.config.DefaultWorkingDir != "" {
		cmd.WorkingDir = e.config.DefaultWorkingDir
	}

	return cmd
}

// setScriptDefaults sets default values for script
func (e *executor) setScriptDefaults(script Script) Script {
	if script.Timeout == 0 {
		script.Timeout = e.config.DefaultTimeout
	}

	if script.Interpreter == "" {
		script.Interpreter = "/bin/bash"
	}

	if script.Environment == nil {
		script.Environment = make(map[string]string)
	}

	// Add default environment variables
	for k, v := range e.config.DefaultEnvironment {
		if _, exists := script.Environment[k]; !exists {
			script.Environment[k] = v
		}
	}

	return script
}

// prepareScriptCommand prepares a command to execute a script
func (e *executor) prepareScriptCommand(script Script) (Command, error) {
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

// createSessionConfig creates session configuration from command
func (e *executor) createSessionConfig(cmd Command) ExtendedSessionConfig {
	return ExtendedSessionConfig{
		SessionConfig: SessionConfig{
			Timeout:     cmd.Timeout,
			Environment: cmd.Environment,
			PTY:         false, // Usually not needed for command execution
		},
		WorkingDir: cmd.WorkingDir,
		Sudo:       cmd.Sudo,
		SudoUser:   cmd.User,
	}
}

// validateConfig validates executor configuration
func (e *executor) validateConfig() error {
	if e.config.DefaultTimeout <= 0 {
		return fmt.Errorf("DefaultTimeout must be positive")
	}

	if e.config.MaxConcurrentCmds <= 0 {
		return fmt.Errorf("MaxConcurrentCmds must be positive")
	}

	if e.config.BufferSize <= 0 {
		return fmt.Errorf("BufferSize must be positive")
	}

	return nil
}

// validateExecutionContext validates that the context has required values for execution
func (e *executor) validateExecutionContext(ctx context.Context) error {
	if _, ok := GetConnectionKey(ctx); !ok {
		return fmt.Errorf("connection key not found in context")
	}
	return nil
}

// SetConfig updates the executor configuration
func (e *executor) SetConfig(config ExecutorConfig) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate configuration
	tempExecutor := &executor{config: config}
	if err := tempExecutor.validateConfig(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	e.config = config
	return nil
}

// GetConfig returns the current executor configuration
func (e *executor) GetConfig() ExecutorConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// GetStats returns executor statistics
func (e *executor) GetStats() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]any{
		"max_concurrent_commands": e.config.MaxConcurrentCmds,
		"default_timeout":         e.config.DefaultTimeout,
		"audit_logging_enabled":   e.config.EnableAuditLogging,
		"whitelist_count":         len(e.config.CommandWhitelist),
		"blacklist_count":         len(e.config.CommandBlacklist),
		"retry_max_attempts":      e.config.RetryStrategy.MaxAttempts,
	}
}
