# Tunnel Package Interface Alignment

This document ensures all migration guides properly align with the actual tunnel package interfaces.

## Available Tunnel Interfaces

### Core Interfaces (from tunnel package)

```go
// SSH Client (actual interface from tunnel package)
type SSHClient interface {
    Connect() error
    Close() error
    IsConnected() bool
    Execute(cmd string, opts ...ExecOption) (*Result, error)
    ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error)
    Upload(localPath, remotePath string, opts ...FileOption) error
    Download(remotePath, localPath string, opts ...FileOption) error
    Ping() error
    HostInfo() (string, error)
    SetTracer(tracer Tracer)
}

// Authentication Configuration
type AuthConfig struct {
    UseAgent      bool
    KeyPath       string
    KeyPassphrase string
    PreferAgent   bool
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
// ✅ CORRECT - Following actual tunnel package patterns
func NewServerHandlers(
    app core.App,
) *ServerHandlers {
    return &ServerHandlers{
        app: app,
    }
}

// ✅ CORRECT - Creating SSH client with new auth system
func createSSHClient(server *models.Server) (*tunnel.Client, error) {
    var authConfig tunnel.AuthConfig
    
    if server.UseSSHAgent {
        authConfig = tunnel.AuthConfigWithAgent()
    } else if server.ManualKeyPath != "" {
        authConfig = tunnel.AuthConfigFromKeyPath(server.ManualKeyPath)
    } else {
        return nil, fmt.Errorf("no authentication method configured")
    }

    config := tunnel.Config{
        Host: server.Host,
        Port: server.Port,
        User: server.RootUsername,
        Auth: authConfig,
        Timeout: 30 * time.Second,
    }

    return tunnel.NewClient(config)
}
```

## Progress Tracking Implementation

Since tunnel interfaces don't include progress tracking, implement it at handler level:

```go
// ✅ CORRECT - Handler-level progress tracking with actual tunnel methods
func handleServerSetup(c *core.RequestEvent, app core.App) error {
    // Create SSH client
    client, err := createSSHClient(server)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]any{
            "error": fmt.Sprintf("Failed to create SSH client: %v", err),
        })
    }
    defer client.Close()

    // Connect to server
    if err := client.Connect(); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]any{
            "error": fmt.Sprintf("Failed to connect: %v", err),
        })
    }

    // Create managers using actual tunnel constructors
    mgr := tunnel.NewManager(client)
    setupMgr := tunnel.NewSetupManager(mgr)

    // Run setup with actual method signatures
    err = setupMgr.SetupPocketBaseServer(server.AppUsername, publicKeys)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]any{
            "error": fmt.Sprintf("Setup failed: %v", err),
        })
    }

    return c.JSON(http.StatusOK, map[string]any{
        "message": "Setup completed successfully",
    })
}
```

## Error Handling Patterns

### Correct Error Handling (from tunnel package)

```go
// ✅ CORRECT - Using actual tunnel error handling
result, err := client.Execute(cmd)
if err != nil {
    if sshErr, ok := err.(*tunnel.Error); ok {
        switch sshErr.Type {
        case tunnel.ErrorConnection:
            return handleConnectionError(c, err)
        case tunnel.ErrorAuth:
            return handleAuthError(c, err)
        case tunnel.ErrorTimeout:
            return handleTimeoutError(c, err)
        case tunnel.ErrorExecution:
            return handleExecutionError(c, err, result)
        }
    }
    return handleGenericError(c, err)
}
```

## Key Types Alignment

### Available Types (from actual tunnel package)

```go
// Connection Configuration
type Config struct {
    Host       string
    Port       int
    User       string
    Auth       AuthConfig
    Timeout    time.Duration
    RetryCount int
    RetryDelay time.Duration
}

// Authentication Configuration
type AuthConfig struct {
    UseAgent      bool
    KeyPath       string
    KeyPassphrase string
    PreferAgent   bool
}

// Security Configuration
type SecurityConfig struct {
    FirewallRules  []FirewallRule
    HardenSSH      bool
    SSHConfig      SSHConfig
    EnableFail2ban bool
}

// SSH Configuration
type SSHConfig struct {
    PasswordAuth        bool
    RootLogin           bool
    PubkeyAuth          bool
    MaxAuthTries        int
    ClientAliveInterval int
    ClientAliveCountMax int
    AllowUsers          []string
    AllowGroups         []string
}

// Firewall Rule
type FirewallRule struct {
    Port        int
    Protocol    string
    Source      string
    Action      string
    Description string
}
```
