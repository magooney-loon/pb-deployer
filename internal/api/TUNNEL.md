# Tunnel Package Interface Alignment

This document ensures all migration guides properly align with the actual tunnel package interfaces.

## Available Tunnel Interfaces

### Core Interfaces (from tunnel/README.md)

```go
// SSH Client
type SSHClient interface {
    Connect(ctx context.Context) error
    Execute(ctx context.Context, cmd string) (string, error)
    ExecuteStream(ctx context.Context, cmd string, handler OutputHandler) error
    Close() error
    IsHealthy() bool
}

// Connection Pool
type Pool interface {
    Get(ctx context.Context, key string) (SSHClient, error)
    Release(key string, client SSHClient)
    HealthCheck(ctx context.Context) HealthReport
    Close() error
}

// Command Executor
type Executor interface {
    RunCommand(ctx context.Context, cmd Command) (*Result, error)
    RunScript(ctx context.Context, script Script) (*Result, error)
    TransferFile(ctx context.Context, transfer FileTransfer) error
}
```

### Manager Interfaces

```go
// Setup Manager
type SetupManager interface {
    CreateUser(ctx context.Context, config UserConfig) error
    SetupSSHKeys(ctx context.Context, username string, keys []string) error
    CreateDirectory(ctx context.Context, path string, owner string) error
}

// Security Manager
type SecurityManager interface {
    ApplyLockdown(ctx context.Context, config SecurityConfig) error
    ConfigureFirewall(ctx context.Context, rules []FirewallRule) error
    SetupFail2ban(ctx context.Context, config Fail2banConfig) error
    AuditSecurity(ctx context.Context) (*SecurityAuditResult, error)  // From actual implementation
}

// Service Manager
type ServiceManager interface {
    ManageService(ctx context.Context, action ServiceAction, name string) error
    GetServiceStatus(ctx context.Context, name string) (*ServiceStatus, error)
    EnableService(ctx context.Context, name string) error
}

// Deployment Manager
type DeploymentManager interface {
    DeployApplication(ctx context.Context, config DeployConfig) error
    RollbackDeployment(ctx context.Context, version string) error
    GetDeploymentStatus(ctx context.Context) (*DeploymentStatus, error)
}
```

## Constructor Patterns

### Correct Service Container Integration

```go
// ✅ CORRECT - Following tunnel package patterns
func NewServerHandlers(
    executor tunnel.Executor,
    setupMgr tunnel.SetupManager,
    securityMgr tunnel.SecurityManager,
    serviceMgr tunnel.ServiceManager,
    pool tunnel.Pool,
    tracerFactory tracer.TracerFactory,
) *ServerHandlers {
    return &ServerHandlers{
        executor:       executor,
        setupMgr:       setupMgr,
        securityMgr:    securityMgr,
        serviceMgr:     serviceMgr,
        pool:           pool,
        sshTracer:      tracerFactory.CreateSSHTracer(),
        poolTracer:     tracerFactory.CreatePoolTracer(),
        securityTracer: tracerFactory.CreateSecurityTracer(),
    }
}
```

## Progress Tracking Implementation

Since tunnel interfaces don't include progress tracking, implement it at handler level:

```go
// ✅ CORRECT - Handler-level progress tracking
func (h *ServerHandlers) performServerSetup(ctx context.Context, config SetupConfig) error {
    // Create progress channel for UI updates
    progressChan := make(chan SetupProgress, 10)
    go h.monitorSetupProgress(ctx, server.ID, progressChan)

    // Use individual setup manager calls
    if err := h.setupMgr.CreateUser(ctx, config.UserConfig); err != nil {
        progressChan <- SetupProgress{Step: "user_creation", Status: "failed", Error: err}
        return err
    }
    progressChan <- SetupProgress{Step: "user_creation", Status: "completed"}

    if err := h.setupMgr.SetupSSHKeys(ctx, config.Username, config.SSHKeys); err != nil {
        progressChan <- SetupProgress{Step: "ssh_keys", Status: "failed", Error: err}
        return err
    }
    progressChan <- SetupProgress{Step: "ssh_keys", Status: "completed"}

    close(progressChan)
    return nil
}
```

## Error Handling Patterns

### Correct Error Handling (from tunnel package)

```go
// ✅ CORRECT - Using tunnel error utilities
result, err := h.executor.RunCommand(ctx, cmd)
if err != nil {
    if tunnel.IsRetryable(err) {
        // Implement retry logic
        return h.retryOperation(ctx, cmd)
    }
    if tunnel.IsAuthError(err) {
        // Handle authentication failure
        return handleAuthError(e, err)
    }
    if tunnel.IsConnectionError(err) {
        // Handle connection failure
        return handleConnectionError(e, err)
    }
    return handleGenericError(e, err)
}
```

## Key Types Alignment

### Available Types (from tunnel package)

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

type FileTransfer struct {
    LocalPath  string
    RemotePath string
    Direction  TransferDirection
    Progress   bool
}
```
