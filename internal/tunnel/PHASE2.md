# Phase 2: Implement Core SSHClient Without Singletons

## Overview
Phase 2 focuses on implementing the core SSHClient interface and related components without using singletons. This phase establishes the foundation for all SSH operations with proper dependency injection, context support, and structured error handling.

## Goals
- Implement the core `SSHClient` interface with dependency injection
- Create `ConnectionFactory` for SSH client instantiation
- Establish proper authentication methods and host key verification
- Integrate with the tracer package for observability
- Provide comprehensive error handling with retry logic
- Support context cancellation and timeout handling

## Prerequisites
- Phase 1 must be completed (interfaces and types defined)
- Tracer package should be available and functional
- Go SSH package (`golang.org/x/crypto/ssh`) available

## Phase 2 Steps

### Step 2.1: Core Types and Constants
**File**: `internal/tunnel/types.go`
**Description**: Define all core types, configurations, and constants needed for SSH operations.

**Tasks**:
- [ ] Define `ConnectionConfig` struct with all SSH connection parameters
- [ ] Define `AuthMethod` struct supporting key/agent/password authentication
- [ ] Define `HostKeyMode` enum for host key verification strategies
- [ ] Define `Command` struct for command execution with options
- [ ] Define `Result` struct for command execution results
- [ ] Define `HealthReport` struct for connection health information
- [ ] Add timeout and retry configuration constants

**Key Components**:
```go
type ConnectionConfig struct {
    Host        string
    Port        int
    Username    string
    AuthMethod  AuthMethod
    Timeout     time.Duration
    HostKeyMode HostKeyMode
    MaxRetries  int
}

type AuthMethod struct {
    Type        string // "key", "agent", "password"
    PrivateKey  []byte
    KeyPath     string
    Password    string
    Passphrase  string
}

type HostKeyMode int

const (
    HostKeyStrict HostKeyMode = iota
    HostKeyAcceptNew
    HostKeyIgnore
)
```

### Step 2.2: Error Types and Handling
**File**: `internal/tunnel/errors.go`
**Description**: Implement structured error types with classification and retry logic.

**Tasks**:
- [ ] Define `SSHError` struct with operation context and retry information
- [ ] Create error classification functions (`IsAuthError`, `IsConnectionError`, etc.)
- [ ] Implement retry logic helpers (`IsRetryable`, `ShouldRetry`)
- [ ] Add error wrapping utilities for context preservation
- [ ] Create timeout and cancellation specific errors

**Key Components**:
```go
type SSHError struct {
    Op        string
    Host      string
    Port      int
    User      string
    Err       error
    Retryable bool
    Temporary bool
}

func IsAuthError(err error) bool
func IsConnectionError(err error) bool  
func IsRetryable(err error) bool
func NewConnectionError(op, host string, port int, err error) *SSHError
```

### Step 2.3: Core SSHClient Implementation
**File**: `internal/tunnel/client.go`
**Description**: Implement the core SSHClient interface with proper lifecycle management.

**Tasks**:
- [ ] Implement `sshClient` struct with connection state management
- [ ] Implement `Connect()` method with authentication and host key verification
- [ ] Implement `Execute()` method for synchronous command execution
- [ ] Implement `ExecuteStream()` method for streaming command output
- [ ] Implement `IsConnected()` method for connection health checking
- [ ] Implement `Close()` method with proper resource cleanup
- [ ] Add connection keepalive and timeout handling
- [ ] Integrate tracing throughout all operations

**Key Components**:
```go
type sshClient struct {
    config     ConnectionConfig
    conn       *ssh.Client
    tracer     tracer.SSHTracer
    mu         sync.RWMutex
    connected  bool
    lastUsed   time.Time
}

func (c *sshClient) Connect(ctx context.Context) error
func (c *sshClient) Execute(ctx context.Context, cmd string) (string, error)
func (c *sshClient) ExecuteStream(ctx context.Context, cmd string) (<-chan string, error)
func (c *sshClient) IsConnected() bool
func (c *sshClient) Close() error
```

### Step 2.4: Authentication Methods
**File**: `internal/tunnel/auth.go`
**Description**: Implement various SSH authentication methods with proper key handling.

**Tasks**:
- [ ] Implement private key authentication with passphrase support
- [ ] Implement SSH agent authentication
- [ ] Implement password authentication
- [ ] Add key file loading and parsing utilities
- [ ] Implement host key verification strategies
- [ ] Add authentication method auto-detection
- [ ] Include proper error handling for auth failures

**Key Components**:
```go
func createAuthMethod(auth AuthMethod) (ssh.AuthMethod, error)
func loadPrivateKey(keyData []byte, passphrase string) (ssh.Signer, error)
func loadPrivateKeyFromFile(keyPath, passphrase string) (ssh.Signer, error)
func connectToAgent() (ssh.AuthMethod, error)
func createHostKeyCallback(mode HostKeyMode) (ssh.HostKeyCallback, error)
```

### Step 2.5: ConnectionFactory Implementation
**File**: `internal/tunnel/factory.go`
**Description**: Implement the ConnectionFactory for creating SSH clients with dependency injection.

**Tasks**:
- [ ] Implement `connectionFactory` struct with tracer dependency
- [ ] Implement `Create()` method for SSH client instantiation
- [ ] Add configuration validation and preprocessing
- [ ] Include connection testing and validation
- [ ] Add factory-level error handling and recovery
- [ ] Implement connection caching strategies (optional)

**Key Components**:
```go
type connectionFactory struct {
    tracer tracer.SSHTracer
}

func NewConnectionFactory(tracer tracer.SSHTracer) ConnectionFactory
func (f *connectionFactory) Create(config ConnectionConfig) (SSHClient, error)
func (f *connectionFactory) validateConfig(config ConnectionConfig) error
```

### Step 2.6: Command Execution Enhancement
**File**: `internal/tunnel/execution.go`
**Description**: Enhance command execution with advanced features and proper session management.

**Tasks**:
- [ ] Implement session management with proper cleanup
- [ ] Add environment variable support for commands
- [ ] Implement sudo command execution patterns
- [ ] Add command timeout handling independent of connection timeout
- [ ] Implement proper signal handling and process termination
- [ ] Add support for interactive commands (stdin handling)
- [ ] Include exit code capture and interpretation

**Key Components**:
```go
type Session struct {
    session    *ssh.Session
    client     *sshClient
    cmd        string
    timeout    time.Duration
    env        map[string]string
    sudo       bool
}

func (s *Session) Run(ctx context.Context) (*Result, error)
func (s *Session) Start(ctx context.Context) error
func (s *Session) Wait() error
func (s *Session) Kill() error
```

### Step 2.7: Health Monitoring Integration
**File**: `internal/tunnel/health.go`
**Description**: Implement connection health monitoring without circular dependencies.

**Tasks**:
- [ ] Implement connection health checking mechanisms
- [ ] Add periodic health validation (ping-like operations)
- [ ] Create health report generation
- [ ] Implement connection recovery strategies
- [ ] Add health metrics collection for tracing
- [ ] Include connection quality assessment

**Key Components**:
```go
type HealthChecker struct {
    client SSHClient
    tracer tracer.SSHTracer
}

func (h *HealthChecker) CheckHealth(ctx context.Context) HealthReport
func (h *HealthChecker) IsHealthy(ctx context.Context) bool
func (h *HealthChecker) RecoverConnection(ctx context.Context) error
```

### Step 2.8: Testing Infrastructure
**File**: `internal/tunnel/client_test.go`
**Description**: Create comprehensive tests for all SSHClient functionality.

**Tasks**:
- [ ] Create mock SSH server for testing
- [ ] Implement unit tests for all SSHClient methods
- [ ] Add integration tests with real SSH connections
- [ ] Test all authentication methods
- [ ] Test error conditions and recovery scenarios
- [ ] Add performance and load testing
- [ ] Include tracing validation in tests

**Key Test Cases**:
- Connection establishment with various auth methods
- Command execution (sync and streaming)
- Error handling and retry logic
- Context cancellation and timeout behavior
- Connection health monitoring
- Resource cleanup and connection lifecycle

### Step 2.9: Documentation and Examples
**File**: `internal/tunnel/examples/`
**Description**: Create usage examples and documentation for the new SSHClient.

**Tasks**:
- [ ] Create basic connection example
- [ ] Add authentication method examples
- [ ] Include command execution patterns
- [ ] Add error handling examples
- [ ] Create troubleshooting guide
- [ ] Include performance tuning guide

## Success Criteria

### Functional Requirements
- [ ] SSHClient can successfully connect using key, agent, and password authentication
- [ ] Commands execute successfully with proper output capture
- [ ] Streaming command execution works for long-running processes
- [ ] Connection health monitoring provides accurate status
- [ ] All operations support context cancellation and timeout
- [ ] Error handling provides clear classification and retry guidance

### Quality Requirements
- [ ] No global state or singletons used anywhere
- [ ] All dependencies are injected through constructors
- [ ] Comprehensive test coverage (>90%)
- [ ] All operations are traced with structured events
- [ ] Memory leaks are prevented with proper resource cleanup
- [ ] Performance is comparable or better than existing implementation

### Integration Requirements
- [ ] ConnectionFactory creates working SSHClient instances
- [ ] Tracer integration works throughout all operations
- [ ] Error types are consistent and well-classified
- [ ] All interfaces are satisfied by concrete implementations

## Dependencies

### External Packages
- `golang.org/x/crypto/ssh` - Core SSH functionality
- `golang.org/x/crypto/ssh/agent` - SSH agent support
- Internal `tracer` package - Observability and tracing

### Internal Dependencies
- Must work with existing `tracer` package
- Should integrate with future Pool and Executor components
- Must be compatible with existing server configuration types

## Rollback Plan

If Phase 2 encounters critical issues:

1. **Partial Implementation**: Complete working components can be used individually
2. **Interface Compatibility**: New interfaces are additive, don't break existing code
3. **Testing Safety**: Comprehensive tests ensure stability before integration
4. **Gradual Migration**: Can coexist with legacy SSH code during transition

## Next Steps

Upon completion of Phase 2:
- Phase 3: Implement connection pooling with dependency injection
- Integration testing with Pool and Executor components
- Performance benchmarking against legacy SSH implementation
- Documentation updates for new SSH client usage patterns

## Estimated Timeline

- **Step 2.1-2.2**: 2-3 days (Types and errors foundation)
- **Step 2.3**: 3-4 days (Core SSHClient implementation)
- **Step 2.4-2.5**: 2-3 days (Authentication and factory)
- **Step 2.6-2.7**: 3-4 days (Command execution and health monitoring)
- **Step 2.8-2.9**: 2-3 days (Testing and documentation)

**Total Estimated Time**: 12-17 days

## Risk Mitigation

### Technical Risks
- **SSH Library Limitations**: Thoroughly test golang.org/x/crypto/ssh edge cases
- **Authentication Complexity**: Test all auth methods with real SSH servers
- **Memory Management**: Use race detector and memory profiling during development
- **Context Handling**: Validate all context cancellation scenarios

### Integration Risks
- **Tracer Dependency**: Ensure tracer package stability before integration
- **Interface Evolution**: Use semantic versioning for interface changes
- **Legacy Compatibility**: Maintain parallel implementation until full migration