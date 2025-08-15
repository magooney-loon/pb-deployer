# Tunnel Package

Modern SSH client library with dependency injection, connection pooling, and specialized managers.

## Features

- **Dependency Injection**: No singletons, clean architecture
- **Connection Pooling**: Efficient connection reuse and health monitoring
- **Specialized Managers**: Domain-specific operations (setup, security, services, deployment)
- **Advanced File Transfer**: Parallel transfers, resume support, integrity checks
- **Health Monitoring**: Real-time connection health and automatic recovery
- **Context Support**: Proper cancellation and timeout handling
- **Structured Errors**: Typed errors with retry logic
- **Comprehensive Tracing**: Full observability with structured tracing

## Core Interfaces

```go
// Core SSH client
type SSHClient interface {
    Connect(ctx context.Context) error
    Execute(ctx context.Context, cmd string) (string, error)
    ExecuteStream(ctx context.Context, cmd string, handler OutputHandler) error
    Close() error
    IsHealthy() bool
}

// Connection management
type Pool interface {
    Get(ctx context.Context, key string) (SSHClient, error)
    Release(key string, client SSHClient)
    HealthCheck(ctx context.Context) HealthReport
    Close() error
}

// Command execution
type Executor interface {
    RunCommand(ctx context.Context, cmd Command) (*Result, error)
    RunScript(ctx context.Context, script Script) (*Result, error)
    TransferFile(ctx context.Context, transfer FileTransfer) error
}
```

## Managers

```go
// Server setup
type SetupManager interface {
    CreateUser(ctx context.Context, config UserConfig) error
    SetupSSHKeys(ctx context.Context, username string, keys []string) error
    CreateDirectory(ctx context.Context, path string, owner string) error
}

// Security operations
type SecurityManager interface {
    ApplyLockdown(ctx context.Context, config SecurityConfig) error
    ConfigureFirewall(ctx context.Context, rules []FirewallRule) error
    SetupFail2ban(ctx context.Context, config Fail2banConfig) error
}

// Service management
type ServiceManager interface {
    ManageService(ctx context.Context, action ServiceAction, name string) error
    GetServiceStatus(ctx context.Context, name string) (*ServiceStatus, error)
    EnableService(ctx context.Context, name string) error
}

// Deployment operations
type DeploymentManager interface {
    DeployApplication(ctx context.Context, config DeployConfig) error
    RollbackDeployment(ctx context.Context, version string) error
    GetDeploymentStatus(ctx context.Context) (*DeploymentStatus, error)
}
```

## Quick Start

```go
// Setup
tracerFactory := tracer.SetupProductionTracing(os.Stdout)
sshTracer := tracerFactory.CreateSSHTracer()
factory := tunnel.NewConnectionFactory(sshTracer)
pool := tunnel.NewPool(factory, poolConfig, sshTracer)
executor := tunnel.NewExecutor(pool, sshTracer)

// Basic command execution
result, err := executor.RunCommand(ctx, tunnel.Command{
    Cmd:     "systemctl status nginx",
    Sudo:    true,
    Timeout: 30 * time.Second,
})

// User setup
setupMgr := tunnel.NewSetupManager(executor, setupTracer)
err := setupMgr.CreateUser(ctx, tunnel.UserConfig{
    Username:   "appuser",
    Groups:     []string{"sudo", "docker"},
    CreateHome: true,
})

// Security lockdown
securityMgr := tunnel.NewSecurityManager(executor, securityTracer)
err := securityMgr.ApplyLockdown(ctx, tunnel.SecurityConfig{
    DisableRootLogin:    true,
    DisablePasswordAuth: true,
    AllowedPorts:        []int{22, 80, 443},
})

// Service management
serviceMgr := tunnel.NewServiceManager(executor, serviceTracer)
err := serviceMgr.ManageService(ctx, tunnel.ServiceRestart, "nginx")

// Deployment
deployMgr := tunnel.NewDeploymentManager(executor, deployTracer)
err := deployMgr.DeployApplication(ctx, tunnel.DeployConfig{
    AppName:    "myapp",
    Version:    "v1.2.3",
    SourcePath: "/local/app",
    TargetPath: "/opt/myapp",
})
```

## Key Types

```go
type ConnectionConfig struct {
    Host       string
    Port       int
    Username   string
    AuthMethod AuthMethod
    Timeout    time.Duration
}

type Command struct {
    Cmd         string
    Sudo        bool
    Timeout     time.Duration
    Environment map[string]string
}

type UserConfig struct {
    Username   string
    HomeDir    string
    Shell      string
    Groups     []string
    CreateHome bool
}

type SecurityConfig struct {
    DisableRootLogin    bool
    DisablePasswordAuth bool
    AllowedPorts        []int
    Fail2banConfig      Fail2banConfig
}
```

## Error Handling

```go
result, err := executor.RunCommand(ctx, cmd)
if err != nil {
    if tunnel.IsRetryable(err) {
        // Retry logic
    }
    if tunnel.IsAuthError(err) {
        // Handle auth failure
    }
    if tunnel.IsConnectionError(err) {
        // Handle connection failure
    }
}
```

## Testing

```go
func TestExecutor(t *testing.T) {
    mockPool := &mockPool{}
    mockClient := &mockClient{}
    tracerFactory := tracer.SetupTestTracing(t)

    mockPool.On("Get", mock.Anything, "test").Return(mockClient, nil)
    mockClient.On("Execute", mock.Anything, "echo test").Return("test", nil)

    executor := tunnel.NewExecutor(mockPool, tracerFactory.CreateSSHTracer())
    result, err := executor.RunCommand(ctx, tunnel.Command{Cmd: "echo test"})

    assert.NoError(t, err)
    assert.Equal(t, "test", result.Output)
}
```
