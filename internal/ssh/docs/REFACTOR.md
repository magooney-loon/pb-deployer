# SSH Package Refactor Plan

## Overview
Complete architectural redesign to eliminate singletons, circular dependencies, and overlapping responsibilities. Focus on dependency injection, clean interfaces, and proper separation of concerns using the modern tracer package for observability.

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
    tracer     tracer.SSHTracer
    mu         sync.RWMutex
    lastUsed   time.Time
}

func (c *sshClient) Connect(ctx context.Context) error {
    span := c.tracer.TraceConnection(ctx, c.config.Host, c.config.Port, c.config.Username)
    defer span.End()
    
    // 1. Create SSH config
    // 2. Dial with timeout
    // 3. Store connection
    
    span.Event("connection_established", 
        tracer.String("host", c.config.Host),
        tracer.Int("port", c.config.Port),
        tracer.String("user", c.config.Username),
    )
    
    return nil
}

func (c *sshClient) Execute(ctx context.Context, cmd string) (string, error) {
    span := c.tracer.TraceCommand(ctx, cmd, false)
    defer span.End()
    
    // 1. Check connection
    // 2. Create session
    // 3. Run command
    
    span.SetFields(tracer.Fields{
        "command": cmd,
        "host": c.config.Host,
        "user": c.config.Username,
    })
    
    if err != nil {
        span.EndWithError(err)
        return "", err
    }
    
    span.Event("command_completed", 
        tracer.String("output", output),
        tracer.Duration("duration", time.Since(start)),
    )
    
    return output, nil
}
```

### 3. Connection Pool

```go
// pool.go - Connection pooling without singletons
type connectionPool struct {
    factory     ConnectionFactory
    connections map[string]*poolEntry
    config      PoolConfig
    tracer      tracer.PoolTracer
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

func NewPool(factory ConnectionFactory, config PoolConfig, poolTracer tracer.PoolTracer) Pool {
    return &connectionPool{
        factory:     factory,
        connections: make(map[string]*poolEntry),
        config:      config,
        tracer:      poolTracer,
    }
}

func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error) {
    span := p.tracer.TraceGet(ctx, key)
    defer span.End()
    
    // 1. Check for existing healthy connection
    // 2. Create new if needed
    // 3. Update metrics
    
    span.SetFields(tracer.Fields{
        "pool.key": key,
        "pool.size": len(p.connections),
    })
    
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    span.Event("connection_acquired")
    return client, nil
}
```

### 4. High-Level Executor

```go
// executor.go - High-level operations executor
type executor struct {
    pool   Pool
    tracer tracer.SSHTracer
}

func NewExecutor(pool Pool, sshTracer tracer.SSHTracer) Executor {
    return &executor{
        pool:   pool,
        tracer: sshTracer,
    }
}

func (e *executor) RunCommand(ctx context.Context, cmd Command) (*Result, error) {
    span := e.tracer.TraceCommand(ctx, cmd.Cmd, cmd.Sudo)
    defer span.End()
    
    // 1. Get connection from pool
    // 2. Apply sudo if needed
    // 3. Execute command
    // 4. Process result
    
    span.SetFields(tracer.Fields{
        "command.sudo": cmd.Sudo,
        "command.timeout": cmd.Timeout,
        "command.env_vars": len(cmd.Environment),
    })
    
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    span.Event("command_executed",
        tracer.Duration("duration", result.Duration),
        tracer.Int("exit_code", result.ExitCode),
    )
    
    return result, nil
}
```

### 5. Specialized Managers

```go
// setup_manager.go - Server setup operations
type SetupManager struct {
    executor Executor
    tracer   tracer.ServiceTracer
}

func (m *SetupManager) CreateUser(ctx context.Context, user UserConfig) error {
    span := m.tracer.TraceDeployment(ctx, "user_setup", user.Username)
    defer span.End()
    
    steps := []SetupStep{
        {Name: "check_user", Fn: m.checkUserExists},
        {Name: "create_user", Fn: m.createUser},
        {Name: "setup_sudo", Fn: m.setupSudo},
    }
    
    for i, step := range steps {
        stepSpan := span.StartChild(step.Name)
        stepSpan.SetField("step", fmt.Sprintf("%d/%d", i+1, len(steps)))
        
        err := step.Fn(ctx, user)
        if err != nil {
            stepSpan.EndWithError(err)
            span.EndWithError(err)
            return err
        }
        
        stepSpan.End()
        span.Event("step_completed", tracer.String("step", step.Name))
    }
    
    return nil
}

// security_manager.go - Security operations
type SecurityManager struct {
    executor Executor
    tracer   tracer.SecurityTracer
}

func (m *SecurityManager) ApplyLockdown(ctx context.Context, config SecurityConfig) error {
    span := m.tracer.TraceSecurityOperation(ctx, "system_lockdown")
    defer span.End()
    
    // Setup firewall
    fwSpan := m.tracer.TraceFirewallRule(ctx, "lockdown", "apply")
    err := m.setupFirewall(ctx, config)
    if err != nil {
        fwSpan.EndWithError(err)
        return err
    }
    fwSpan.End()
    
    // Configure fail2ban
    f2bSpan := m.tracer.TraceSecurityOperation(ctx, "fail2ban_setup")
    err = m.setupFail2ban(ctx, config.Fail2banConfig)
    if err != nil {
        f2bSpan.EndWithError(err)
        return err
    }
    f2bSpan.End()
    
    // Harden SSH
    sshSpan := m.tracer.TraceSecurityOperation(ctx, "ssh_hardening")
    err = m.hardenSSH(ctx, config)
    if err != nil {
        sshSpan.EndWithError(err)
        return err
    }
    sshSpan.End()
    
    span.Event("lockdown_completed")
    return nil
}

// service_manager.go - Systemd service operations
type ServiceManager struct {
    executor Executor
    tracer   tracer.ServiceTracer
}

func (m *ServiceManager) ManageService(ctx context.Context, action, service string) error {
    span := m.tracer.TraceServiceAction(ctx, service, action)
    defer span.End()
    
    cmd := fmt.Sprintf("systemctl %s %s", action, service)
    result, err := m.executor.RunCommand(ctx, Command{
        Cmd:  cmd,
        Sudo: true,
    })
    
    if err != nil {
        span.EndWithError(err)
        return err
    }
    
    span.SetFields(tracer.Fields{
        "service.name": service,
        "service.action": action,
        "service.exit_code": result.ExitCode,
    })
    
    return nil
}
```

### 6. Health Monitoring

```go
// health.go - Health monitoring without circular dependencies
type HealthMonitor struct {
    pool     Pool
    tracer   tracer.PoolTracer
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
            healthSpan := h.tracer.TraceHealthCheck(ctx)
            report := h.pool.HealthCheck(ctx)
            
            tracer.RecordPoolHealth(healthSpan, 
                report.TotalConnections,
                report.HealthyConnections, 
                report.FailedConnections)
            
            healthSpan.SetFields(tracer.Fields{
                "pool.total": report.TotalConnections,
                "pool.healthy": report.HealthyConnections,
                "pool.failed": report.FailedConnections,
            })
            
            healthSpan.End()
        }
    }
}
```

### 7. Troubleshooting

```go
// troubleshoot.go - Diagnostic operations
type Troubleshooter struct {
    tracer tracer.SSHTracer
}

type DiagnosticResult struct {
    Step       string
    Status     string
    Message    string
    Suggestion string
    Duration   time.Duration
}

func (t *Troubleshooter) Diagnose(ctx context.Context, config ConnectionConfig) []DiagnosticResult {
    span := t.tracer.TraceDiagnostic(ctx, config.Host)
    defer span.End()
    
    checks := []DiagnosticCheck{
        t.checkNetwork,
        t.checkSSHService,
        t.checkAuthentication,
        t.checkPermissions,
    }
    
    var results []DiagnosticResult
    
    for _, check := range checks {
        checkSpan := span.StartChild(check.Name)
        
        start := time.Now()
        result := check.Fn(ctx, config)
        result.Duration = time.Since(start)
        
        checkSpan.SetFields(tracer.Fields{
            "check.name": result.Step,
            "check.status": result.Status,
            "check.duration": result.Duration,
        })
        
        if result.Status == "error" {
            checkSpan.SetStatus(tracer.StatusError)
            checkSpan.SetField("error.message", result.Message)
        }
        
        checkSpan.End()
        results = append(results, result)
    }
    
    return results
}
```

## Usage Pattern

```go
// main.go or service initialization
func initializeSSH() (*SSHService, error) {
    // 1. Setup tracing
    tracerFactory := tracer.SetupProductionTracing(os.Stdout)
    
    // 2. Create specialized tracers
    sshTracer := tracerFactory.CreateSSHTracer()
    poolTracer := tracerFactory.CreatePoolTracer()
    securityTracer := tracerFactory.CreateSecurityTracer()
    serviceTracer := tracerFactory.CreateServiceTracer()

    // 3. Create factory
    factory := NewConnectionFactory(sshTracer)

    // 4. Create pool
    poolConfig := PoolConfig{
        MaxConnections: 10,
        MaxIdleTime:    15 * time.Minute,
    }
    pool := NewPool(factory, poolConfig, poolTracer)

    // 5. Create executor
    executor := NewExecutor(pool, sshTracer)

    // 6. Create specialized managers
    setupMgr := NewSetupManager(executor, serviceTracer)
    securityMgr := NewSecurityManager(executor, securityTracer)
    serviceMgr := NewServiceManager(executor, serviceTracer)

    // 7. Create service facade
    return &SSHService{
        pool:        pool,
        executor:    executor,
        setup:       setupMgr,
        security:    securityMgr,
        service:     serviceMgr,
        tracer:      sshTracer,
    }, nil
}

// Usage
func deployApp(ctx context.Context, svc *SSHService) error {
    // Trace the deployment
    deploySpan := svc.tracer.TraceDeployment(ctx, "myapp", "v1.0")
    defer deploySpan.End()
    
    // High-level operations
    result, err := svc.executor.RunCommand(ctx, Command{
        Cmd:  "systemctl restart myapp",
        Sudo: true,
    })

    if err != nil {
        deploySpan.EndWithError(err)
        return err
    }

    deploySpan.Event("deployment_completed", 
        tracer.String("app", "myapp"),
        tracer.String("version", "v1.0"),
        tracer.Duration("total_time", time.Since(start)),
    )
    
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

### 3. Modern Tracing
- Structured tracing with spans and events
- Context propagation across operations
- Rich metadata and error tracking
- Performance monitoring built-in

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
    return fmt.Sprintf("ssh %s failed for %s@%s: %v", e.Op, e.User, e.Server, e.Err)
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
    tracerFactory := tracer.SetupTestTracing(t)
    defer tracerFactory.Shutdown(context.Background())
    
    sshTracer := tracerFactory.CreateSSHTracer()
    executor := NewExecutor(mockPool, sshTracer)
    
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
- **Observability**: Rich tracing throughout all operations
- **Flexibility**: Easy to extend with new managers
- **Reliability**: Proper error handling and recovery

## File Structure
```
internal/tunnel/
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
