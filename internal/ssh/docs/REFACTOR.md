# SSH Package Refactor Plan

## Overview
Complete architectural redesign to eliminate singletons, circular dependencies, and overlapping responsibilities. Focus on dependency injection, clear interfaces, and proper separation of concerns.

## Core Architecture

### 1. Interfaces & Types

```go
// Core SSH interface - single responsibility
type SSHClient interface {
    Connect(ctx context.Context) error
    Execute(ctx context.Context, cmd string) (string, error)
    ExecuteStream(ctx context.Context, cmd string) (<-chan string, error)
    Close() error
    IsConnected() bool
}

// Connection factory - creates SSH clients
type ConnectionFactory interface {
    Create(config ConnectionConfig) (SSHClient, error)
}

// Connection pool - manages multiple connections
type Pool interface {
    Get(ctx context.Context, key string) (SSHClient, error)
    Release(key string, client SSHClient)
    Close() error
    HealthCheck(ctx context.Context) HealthReport
}

// Operations executor - high-level operations
type Executor interface {
    RunCommand(ctx context.Context, cmd Command) (*Result, error)
    RunScript(ctx context.Context, script Script) (*Result, error)
    TransferFile(ctx context.Context, transfer Transfer) error
}

// Configuration types
type ConnectionConfig struct {
    Host        string
    Port        int
    Username    string
    AuthMethod  AuthMethod
    Timeout     time.Duration
    HostKeyMode HostKeyMode
}

type AuthMethod struct {
    Type       string // "key", "agent", "password"
    PrivateKey []byte
    Password   string
}

type Command struct {
    Cmd         string
    Sudo        bool
    Timeout     time.Duration
    Environment map[string]string
}
```

### 2. Core Components

```go
// sshclient.go - Basic SSH client implementation
type sshClient struct {
    config     ConnectionConfig
    conn       *ssh.Client
    logger     *logger.SSHLogger
    mu         sync.RWMutex
    lastUsed   time.Time
}

func (c *sshClient) Connect(ctx context.Context) error {
    // 1. Create SSH config
    // 2. Dial with timeout
    // 3. Store connection
    // 4. Log with logger.SSHConnect()
}

func (c *sshClient) Execute(ctx context.Context, cmd string) (string, error) {
    // 1. Check connection
    // 2. Create session
    // 3. Run command
    // 4. Log with logger.SSHCommand()
}
```

### 3. Connection Pool

```go
// pool.go - Connection pooling without singletons
type connectionPool struct {
    factory     ConnectionFactory
    connections map[string]*poolEntry
    config      PoolConfig
    logger      *logger.SSHLogger
    mu          sync.RWMutex
}

type poolEntry struct {
    client      SSHClient
    key         string
    createdAt   time.Time
    lastUsed    time.Time
    useCount    int64
    healthy     bool
}

type PoolConfig struct {
    MaxConnections  int
    MaxIdleTime     time.Duration
    HealthInterval  time.Duration
    CleanupInterval time.Duration
}

func NewPool(factory ConnectionFactory, config PoolConfig, log *logger.SSHLogger) Pool {
    // Create pool with injected dependencies
}

func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error) {
    // 1. Check for existing healthy connection
    // 2. Create new if needed
    // 3. Update metrics
    // 4. Log with logger.PoolConnection()
}
```

### 4. High-Level Executor

```go
// executor.go - High-level operations executor
type executor struct {
    pool   Pool
    logger *logger.SSHLogger
}

func NewExecutor(pool Pool, log *logger.SSHLogger) Executor {
    // Create executor with injected dependencies
}

func (e *executor) RunCommand(ctx context.Context, cmd Command) (*Result, error) {
    // 1. Get connection from pool
    // 2. Apply sudo if needed
    // 3. Execute command
    // 4. Process result
    // 5. Log with logger.SSHOperation()
}
```

### 5. Specialized Managers

```go
// setup_manager.go - Server setup operations
type SetupManager struct {
    executor Executor
    logger   *logger.SSHLogger
}

func (m *SetupManager) CreateUser(ctx context.Context, user UserConfig) error {
    steps := []SetupStep{
        {Name: "check_user", Fn: m.checkUserExists},
        {Name: "create_user", Fn: m.createUser},
        {Name: "setup_sudo", Fn: m.setupSudo},
    }
    // Execute steps with progress logging
}

// security_manager.go - Security operations
type SecurityManager struct {
    executor Executor
    logger   *logger.SSHLogger
}

func (m *SecurityManager) ApplyLockdown(ctx context.Context, config SecurityConfig) error {
    // 1. Setup firewall
    // 2. Configure fail2ban
    // 3. Harden SSH
    // Log each step with logger.SecurityStep()
}

// service_manager.go - Systemd service operations
type ServiceManager struct {
    executor Executor
    logger   *logger.SSHLogger
}

func (m *ServiceManager) ManageService(ctx context.Context, action, service string) error {
    // Execute systemctl commands
    // Log with logger.ServiceOperation()
}
```

### 6. Health Monitoring

```go
// health.go - Health monitoring without circular dependencies
type HealthMonitor struct {
    pool     Pool
    logger   *logger.SSHLogger
    interval time.Duration
    stopCh   chan struct{}
}

func (h *HealthMonitor) Start(ctx context.Context) {
    ticker := time.NewTicker(h.interval)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            report := h.pool.HealthCheck(ctx)
            h.logger.PoolHealthMetrics(report)
        }
    }
}
```

### 7. Troubleshooting

```go
// troubleshoot.go - Diagnostic operations
type Troubleshooter struct {
    logger *logger.SSHLogger
}

type DiagnosticResult struct {
    Step       string
    Status     string
    Message    string
    Suggestion string
    Duration   time.Duration
}

func (t *Troubleshooter) Diagnose(ctx context.Context, config ConnectionConfig) []DiagnosticResult {
    checks := []DiagnosticCheck{
        t.checkNetwork,
        t.checkSSHService,
        t.checkAuthentication,
        t.checkPermissions,
    }
    // Run checks and log with logger.DiagnosticStep()
}
```

## Usage Pattern

```go
// main.go or service initialization
func initializeSSH() (*SSHService, error) {
    // 1. Create logger
    sshLogger := logger.NewSSHLogger(server, false)

    // 2. Create factory
    factory := NewConnectionFactory(sshLogger)

    // 3. Create pool
    poolConfig := PoolConfig{
        MaxConnections: 10,
        MaxIdleTime:    15 * time.Minute,
    }
    pool := NewPool(factory, poolConfig, sshLogger)

    // 4. Create executor
    executor := NewExecutor(pool, sshLogger)

    // 5. Create specialized managers
    setupMgr := NewSetupManager(executor, sshLogger)
    securityMgr := NewSecurityManager(executor, sshLogger)
    serviceMgr := NewServiceManager(executor, sshLogger)

    // 6. Create service facade
    return &SSHService{
        pool:        pool,
        executor:    executor,
        setup:       setupMgr,
        security:    securityMgr,
        service:     serviceMgr,
        logger:      sshLogger,
    }, nil
}

// Usage
func deployApp(ctx context.Context, svc *SSHService) error {
    // High-level operations
    result, err := svc.executor.RunCommand(ctx, Command{
        Cmd:  "systemctl restart myapp",
        Sudo: true,
    })

    if err != nil {
        svc.logger.SSHError("deployment", server.Host, "app", err)
        return err
    }

    svc.logger.DeploymentComplete("myapp", "v1.0", true, time.Since(start))
    return nil
}
```

## Key Improvements

### 1. Dependency Injection
- No singletons, all dependencies injected
- Testable with mock implementations
- Clear ownership and lifecycle

### 2. Single Responsibility
- SSHClient: Basic SSH operations only
- Pool: Connection management only
- Executor: Command execution patterns
- Managers: Specialized operations

### 3. Proper Logging
- Consistent use of logger.SSHLogger
- Structured logging at appropriate levels
- Operation tracking with logger.StartOperation()

### 4. Error Handling
```go
type SSHError struct {
    Op      string
    Server  string
    User    string
    Err     error
    Retryable bool
}

func (e *SSHError) Error() string {
    return logger.FormatConnectionError(e.Err, e.Server, e.User)
}
```

### 5. Context Support
- All operations accept context
- Proper cancellation and timeout support
- Resource cleanup on context cancellation

### 6. Testing Strategy
```go
// Mockable interfaces
type MockSSHClient struct {
    mock.Mock
}

func TestExecutor_RunCommand(t *testing.T) {
    mockPool := &MockPool{}
    mockClient := &MockSSHClient{}
    logger := logger.NewTestLogger()

    executor := NewExecutor(mockPool, logger)
    // Test without real SSH connections
}
```

## Migration Steps

1. **Phase 1**: Create new interfaces and types
2. **Phase 2**: Implement core SSHClient without singletons
3. **Phase 3**: Build new Pool with dependency injection
4. **Phase 4**: Create Executor for high-level operations
5. **Phase 5**: Migrate specialized managers one by one
6. **Phase 6**: Update all callers to use new API
7. **Phase 7**: Remove old singleton-based code

## Benefits

- **Testability**: Mock any component
- **Maintainability**: Clear boundaries and responsibilities
- **Performance**: Better connection reuse and pooling
- **Observability**: Structured logging throughout
- **Flexibility**: Easy to extend with new managers
- **Reliability**: Proper error handling and recovery

## File Structure
```
internal/ssh-v2/
├── client.go          # Core SSHClient implementation
├── factory.go         # Connection factory
├── pool.go           # Connection pool
├── executor.go       # Command executor
├── managers/
│   ├── setup.go      # Setup operations
│   ├── security.go   # Security operations
│   └── service.go    # Service management
├── health.go         # Health monitoring
├── troubleshoot.go   # Diagnostics
├── errors.go         # Error types
└── service.go        # Public API facade
```
