# Phase 4: High-Level Executor Implementation

## Overview
Phase 4 implements the high-level Executor that orchestrates SSH operations using the connection pool from Phase 3. The Executor provides a simple interface for command execution, script running, and file transfers while handling connection management, error recovery, and comprehensive tracing.

## Goals
- Implement the `Executor` interface with dependency injection
- Provide high-level SSH operation patterns (commands, scripts, transfers)
- Handle sudo operations and environment management
- Integrate comprehensive error handling and retry logic
- Support timeouts and context cancellation
- Provide rich tracing and observability

## Prerequisites
- Phase 3 must be completed (Connection Pool)
- Phase 2 components (SSHClient, ConnectionFactory) are functional
- Tracer package integration is working
- All previous phase tests are passing

## Phase 4 Implementation

### Step 4.1: Core Executor Implementation
**File**: `internal/tunnel/executor/executor.go`
**Description**: Main executor implementation with connection pool integration.

**Tasks**:
- [ ] Implement `Executor` interface with dependency injection
- [ ] Create connection key generation from command context
- [ ] Implement `RunCommand()` with sudo support and environment handling
- [ ] Implement `RunScript()` with interpreter selection
- [ ] Implement `TransferFile()` for file uploads/downloads
- [ ] Add comprehensive error handling with retry logic
- [ ] Integrate tracing throughout all operations

**Key Components**:
```go
type executor struct {
    pool   Pool
    tracer SSHTracer
    config ExecutorConfig
}

type ExecutorConfig struct {
    DefaultTimeout  time.Duration
    MaxRetries      int
    RetryBackoff    time.Duration
    SudoTimeout     time.Duration
}

func NewExecutor(pool Pool, config ExecutorConfig, sshTracer SSHTracer) Executor
func (e *executor) RunCommand(ctx context.Context, cmd Command) (*Result, error)
func (e *executor) RunScript(ctx context.Context, script Script) (*Result, error)
func (e *executor) TransferFile(ctx context.Context, transfer Transfer) error
```

### Step 4.2: Command Execution Logic
**File**: `internal/tunnel/executor/commands.go`
**Description**: Detailed command execution with sudo and environment handling.

**Tasks**:
- [ ] Implement command preparation with environment variables
- [ ] Add sudo wrapper logic with proper escaping
- [ ] Implement timeout handling per command
- [ ] Add exit code parsing and error classification
- [ ] Implement command output streaming support
- [ ] Add working directory support
- [ ] Create command validation and sanitization

**Key Components**:
```go
func (e *executor) executeCommand(ctx context.Context, client SSHClient, cmd Command) (*Result, error)
func (e *executor) prepareSudoCommand(cmd Command) string
func (e *executor) applyEnvironment(cmd Command) string
func (e *executor) parseCommandResult(output string, err error) (*Result, error)
func (e *executor) validateCommand(cmd Command) error
```

### Step 4.3: Script Execution
**File**: `internal/tunnel/executor/scripts.go`
**Description**: Script execution with interpreter support and file management.

**Tasks**:
- [ ] Implement script upload to temporary location
- [ ] Add interpreter detection and validation
- [ ] Support for script arguments and environment
- [ ] Implement script cleanup after execution
- [ ] Add support for different script types (bash, python, etc.)
- [ ] Implement script permissions handling

**Key Components**:
```go
func (e *executor) executeScript(ctx context.Context, client SSHClient, script Script) (*Result, error)
func (e *executor) uploadScript(ctx context.Context, client SSHClient, script Script) (string, error)
func (e *executor) detectInterpreter(script Script) string
func (e *executor) cleanupScript(ctx context.Context, client SSHClient, path string)
func (e *executor) validateScript(script Script) error
```

### Step 4.4: File Transfer Operations
**File**: `internal/tunnel/executor/transfer.go`
**Description**: File transfer implementation using SCP/SFTP protocols.

**Tasks**:
- [ ] Implement SCP-based file upload
- [ ] Implement SCP-based file download
- [ ] Add file permissions and ownership support
- [ ] Implement directory transfer support
- [ ] Add progress reporting for large files
- [ ] Handle file conflicts and overwrites
- [ ] Implement transfer validation (checksums)

**Key Components**:
```go
func (e *executor) transferFile(ctx context.Context, client SSHClient, transfer Transfer) error
func (e *executor) uploadFile(ctx context.Context, client SSHClient, transfer Transfer) error
func (e *executor) downloadFile(ctx context.Context, client SSHClient, transfer Transfer) error
func (e *executor) setFilePermissions(ctx context.Context, client SSHClient, path, permissions, owner, group string) error
func (e *executor) validateTransfer(transfer Transfer) error
```

### Step 4.5: Error Handling and Retry Logic
**File**: `internal/tunnel/executor/retry.go`
**Description**: Sophisticated error handling with retry strategies.

**Tasks**:
- [ ] Implement error classification (retryable vs. permanent)
- [ ] Add exponential backoff retry logic
- [ ] Implement connection health checking before retry
- [ ] Add circuit breaker pattern for failed hosts
- [ ] Implement retry statistics and reporting
- [ ] Create custom error types for different failure modes

**Key Components**:
```go
type RetryStrategy struct {
    MaxAttempts  int
    BaseDelay    time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func (e *executor) executeWithRetry(ctx context.Context, operation func() error, strategy RetryStrategy) error
func (e *executor) classifyError(err error) ErrorType
func (e *executor) shouldRetry(err error, attempt int) bool
func (e *executor) calculateBackoff(attempt int, strategy RetryStrategy) time.Duration
```

### Step 4.6: Connection Management
**File**: `internal/tunnel/executor/connection.go`
**Description**: Smart connection management and key generation.

**Tasks**:
- [ ] Implement connection key generation from command/script/transfer context
- [ ] Add connection affinity for related operations
- [ ] Implement connection health validation
- [ ] Add connection preference handling (prefer existing connections)
- [ ] Implement connection switching for failed operations
- [ ] Add connection usage analytics

**Key Components**:
```go
func (e *executor) generateConnectionKey(host, user string, port int) string
func (e *executor) getConnection(ctx context.Context, key string) (SSHClient, error)
func (e *executor) releaseConnection(key string, client SSHClient)
func (e *executor) validateConnection(ctx context.Context, client SSHClient) error
func (e *executor) switchConnection(ctx context.Context, oldKey string) (SSHClient, string, error)
```

### Step 4.7: Configuration and Utilities
**File**: `internal/tunnel/executor/config.go`
**Description**: Executor configuration with presets and validation.

**Tasks**:
- [ ] Define `ExecutorConfig` with all tuning options
- [ ] Implement configuration validation
- [ ] Provide configuration presets (dev, prod, etc.)
- [ ] Add configuration builder pattern
- [ ] Implement runtime configuration updates
- [ ] Create configuration documentation

**Key Components**:
```go
type ExecutorConfig struct {
    DefaultTimeout   time.Duration
    MaxRetries       int
    RetryBackoff     time.Duration
    SudoTimeout      time.Duration
    ScriptTimeout    time.Duration
    TransferTimeout  time.Duration
    ConnectionReuse  bool
    MaxConcurrent    int
}

func DefaultExecutorConfig() ExecutorConfig
func ProductionExecutorConfig() ExecutorConfig
func (c ExecutorConfig) Validate() error
func (c ExecutorConfig) WithTimeout(timeout time.Duration) ExecutorConfig
```



## Core Implementation Details

### Command Execution with Context
```go
func (e *executor) RunCommand(ctx context.Context, cmd Command) (*Result, error) {
    span := e.tracer.TraceCommand(ctx, cmd.Cmd, cmd.Sudo)
    defer span.End()

    // Apply defaults
    if cmd.Timeout == 0 {
        cmd.Timeout = e.config.DefaultTimeout
    }

    // Create connection key
    connKey := e.generateConnectionKey(cmd.Host, cmd.User, cmd.Port)
    
    span.SetFields(tracer.Fields{
        "command.cmd":         cmd.Cmd,
        "command.sudo":        cmd.Sudo,
        "command.timeout":     cmd.Timeout,
        "command.working_dir": cmd.WorkingDir,
        "command.env_count":   len(cmd.Environment),
        "connection.key":      connKey,
    })

    // Execute with retry logic
    var result *Result
    retryStrategy := RetryStrategy{
        MaxAttempts: e.config.MaxRetries,
        BaseDelay:   e.config.RetryBackoff,
        MaxDelay:    30 * time.Second,
        Multiplier:  2.0,
    }

    err := e.executeWithRetry(ctx, func() error {
        var execErr error
        result, execErr = e.executeCommandOnce(ctx, connKey, cmd)
        return execErr
    }, retryStrategy)

    if err != nil {
        span.EndWithError(err)
        return nil, err
    }

    span.SetFields(tracer.Fields{
        "result.exit_code": result.ExitCode,
        "result.duration":  result.Duration,
        "result.success":   result.ExitCode == 0,
    })

    span.Event("command_completed")
    return result, nil
}
```

### Script Execution
```go
func (e *executor) RunScript(ctx context.Context, script Script) (*Result, error) {
    span := e.tracer.TraceCommand(ctx, "script_execution", false)
    defer span.End()

    // Generate connection key
    connKey := e.generateConnectionKey(script.Host, script.User, script.Port)

    span.SetFields(tracer.Fields{
        "script.interpreter": script.Interpreter,
        "script.args_count":  len(script.Args),
        "script.size":        len(script.Content),
        "connection.key":     connKey,
    })

    // Get connection
    client, err := e.getConnection(ctx, connKey)
    if err != nil {
        span.EndWithError(err)
        return nil, fmt.Errorf("failed to get connection: %w", err)
    }
    defer e.releaseConnection(connKey, client)

    // Upload script
    scriptPath, err := e.uploadScript(ctx, client, script)
    if err != nil {
        span.EndWithError(err)
        return nil, fmt.Errorf("failed to upload script: %w", err)
    }
    defer e.cleanupScript(ctx, client, scriptPath)

    // Execute script
    cmd := Command{
        Cmd:         fmt.Sprintf("%s %s %s", script.Interpreter, scriptPath, strings.Join(script.Args, " ")),
        Timeout:     script.Timeout,
        Environment: script.Environment,
    }

    result, err := e.executeCommand(ctx, client, cmd)
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }

    span.Event("script_completed")
    return result, nil
}
```

### File Transfer
```go
func (e *executor) TransferFile(ctx context.Context, transfer Transfer) error {
    span := e.tracer.TraceFileTransfer(ctx, transfer.Source, transfer.Destination, 0)
    defer span.End()

    // Generate connection key
    connKey := e.generateConnectionKey(transfer.Host, transfer.User, transfer.Port)

    span.SetFields(tracer.Fields{
        "transfer.source":      transfer.Source,
        "transfer.destination": transfer.Destination,
        "transfer.direction":   transfer.Direction,
        "connection.key":       connKey,
    })

    // Get connection
    client, err := e.getConnection(ctx, connKey)
    if err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to get connection: %w", err)
    }
    defer e.releaseConnection(connKey, client)

    // Perform transfer
    switch transfer.Direction {
    case TransferUpload:
        err = e.uploadFile(ctx, client, transfer)
    case TransferDownload:
        err = e.downloadFile(ctx, client, transfer)
    default:
        err = fmt.Errorf("unknown transfer direction: %v", transfer.Direction)
    }

    if err != nil {
        span.EndWithError(err)
        return err
    }

    // Set permissions if specified
    if transfer.Permissions != "" || transfer.Owner != "" || transfer.Group != "" {
        targetPath := transfer.Destination
        if transfer.Direction == TransferUpload {
            targetPath = transfer.Destination
        }
        
        err = e.setFilePermissions(ctx, client, targetPath, transfer.Permissions, transfer.Owner, transfer.Group)
        if err != nil {
            span.EndWithError(err)
            return fmt.Errorf("failed to set permissions: %w", err)
        }
    }

    span.Event("transfer_completed")
    return nil
}
```

## Usage Examples

### Basic Command Execution
```go
// Create executor
tracerFactory := tracer.SetupProductionTracing(os.Stdout)
sshTracer := tracerFactory.CreateSSHTracer()
poolTracer := tracerFactory.CreatePoolTracer()

factory := tunnel.NewConnectionFactory(sshTracer)
pool := tunnel.NewPool(factory, tunnel.DefaultPoolConfig(), poolTracer)
executor := tunnel.NewExecutor(pool, tunnel.DefaultExecutorConfig(), sshTracer)
defer pool.Close()

// Execute command
cmd := tunnel.Command{
    Cmd:     "systemctl status nginx",
    Host:    "web-server-1",
    User:    "deploy",
    Port:    22,
    Sudo:    true,
    Timeout: 30 * time.Second,
    Environment: map[string]string{
        "LC_ALL": "C",
    },
}

result, err := executor.RunCommand(ctx, cmd)
if err != nil {
    return err
}

fmt.Printf("Exit code: %d\n", result.ExitCode)
fmt.Printf("Output: %s\n", result.Output)
```

### Script Execution
```go
script := tunnel.Script{
    Content: `#!/bin/bash
set -e
echo "Starting deployment..."
systemctl stop myapp
cp /tmp/myapp-new /usr/local/bin/myapp
systemctl start myapp
echo "Deployment completed"
`,
    Interpreter: "/bin/bash",
    Host:        "app-server-1",
    User:        "deploy",
    Port:        22,
    Timeout:     5 * time.Minute,
    Environment: map[string]string{
        "DEPLOY_ENV": "production",
    },
}

result, err := executor.RunScript(ctx, script)
```

### File Transfer
```go
transfer := tunnel.Transfer{
    Source:      "/local/app/myapp",
    Destination: "/tmp/myapp-new",
    Direction:   tunnel.TransferUpload,
    Host:        "app-server-1",
    User:        "deploy",
    Port:        22,
    Permissions: "0755",
    Owner:       "deploy",
    Group:       "deploy",
}

err := executor.TransferFile(ctx, transfer)
```

## Success Criteria

### Functional Requirements
- [ ] Executor successfully orchestrates SSH operations using pool
- [ ] Command execution works with sudo and environment variables
- [ ] Script execution supports multiple interpreters
- [ ] File transfers work in both directions with permissions
- [ ] Error handling and retry logic work correctly
- [ ] All operations support context cancellation and timeouts
- [ ] Connection management is efficient and leak-free

### Quality Requirements
- [ ] No singleton patterns used
- [ ] All dependencies injected through constructors
- [ ] Comprehensive test coverage (>90%)
- [ ] All operations traced with rich metadata
- [ ] Memory usage is bounded and predictable
- [ ] Performance is suitable for production workloads

## File Structure

```
internal/tunnel/executor/
├── executor.go              # Main executor implementation
├── commands.go              # Command execution logic
├── scripts.go               # Script execution logic
├── transfer.go              # File transfer operations
├── retry.go                 # Error handling and retry logic
├── connection.go            # Connection management
└── config.go                # Configuration and utilities
```

## Estimated Timeline

- **Step 4.1**: 2-3 days (Core executor implementation)
- **Step 4.2**: 2-3 days (Command execution logic)
- **Step 4.3**: 2-3 days (Script execution)
- **Step 4.4**: 3-4 days (File transfer operations)
- **Step 4.5**: 2-3 days (Error handling and retry logic)
- **Step 4.6**: 2 days (Connection management)
- **Step 4.7**: 1 day (Configuration and utilities)

**Total Estimated Time**: 14-21 days

## Next Steps

Upon completion of Phase 4:
- Phase 5: Implement specialized managers (SetupManager, SecurityManager, ServiceManager)
- Integration testing between all components
- Performance benchmarking
- Load testing with realistic workloads
- Documentation and examples

## Key Design Decisions

### Connection Reuse
- Executor generates stable connection keys for operation affinity
- Pool handles connection lifecycle and health
- Failed connections trigger automatic retry with new connections

### Error Classification
- Retryable errors: network issues, temporary SSH problems
- Permanent errors: authentication failures, permission issues
- Circuit breaker prevents repeated failures to same host

### Sudo Handling
- Automatic sudo wrapper with proper command escaping
- Separate timeout for sudo operations
- Environment preservation across sudo boundary

### Script Management
- Temporary script upload with automatic cleanup
- Interpreter detection and validation
- Argument passing with proper escaping

This simplified approach focuses on the essential executor functionality while maintaining clean architecture and comprehensive error handling.