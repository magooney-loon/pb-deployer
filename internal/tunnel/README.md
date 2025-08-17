# Tunnel Package - Simplified Architecture

A streamlined SSH client library for server management.

## Core Principles

- **Single Connection**: One server at a time, no connection pooling needed
- **Direct Access**: Simple, direct API without excessive abstraction
- **Practical Defaults**: Sensible defaults with optional configuration
- **Clear Errors**: Simple error types that are easy to understand
- **Minimal Dependencies**: Keep it simple and maintainable

## Core Components

### 1. SSH Client (Single Connection)

```go
// Client represents a single SSH connection to a server
type Client struct {
    config   Config
    conn     *ssh.Client
    sftp     *sftp.Client
    tracer   Tracer
}

// Config holds connection configuration
type Config struct {
    Host        string
    Port        int
    User        string
    Password    string        // Optional: password auth
    PrivateKey  string        // Optional: key auth
    Timeout     time.Duration
    RetryCount  int
}

// Basic operations
func NewClient(config Config) (*Client, error)
func (c *Client) Connect() error
func (c *Client) Close() error
func (c *Client) IsConnected() bool
func (c *Client) Execute(cmd string, opts ...ExecOption) (*Result, error)
func (c *Client) ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error)
func (c *Client) Upload(localPath, remotePath string) error
func (c *Client) Download(remotePath, localPath string) error
```

### 2. Manager (All-in-One)

```go
// Manager handles all server operations through a single interface
type Manager struct {
    client *Client
    tracer Tracer
}

func NewManager(client *Client) *Manager

// User Management
func (m *Manager) CreateUser(username string, opts ...UserOption) error
func (m *Manager) DeleteUser(username string) error
func (m *Manager) SetupSSHKeys(username string, keys []string) error

// Service Management
func (m *Manager) ServiceStart(name string) error
func (m *Manager) ServiceStop(name string) error
func (m *Manager) ServiceRestart(name string) error
func (m *Manager) ServiceStatus(name string) (*ServiceStatus, error)
func (m *Manager) ServiceLogs(name string, lines int) (string, error)

// Security
func (m *Manager) SetupFirewall(rules []FirewallRule) error
func (m *Manager) HardenSSH(config SSHConfig) error
func (m *Manager) SetupFail2ban() error

// Deployment
func (m *Manager) Deploy(app AppConfig) error
func (m *Manager) Rollback(app string, version string) error

// Package Management
func (m *Manager) InstallPackages(packages ...string) error
func (m *Manager) UpdateSystem() error
```

## Simple Usage

```go
// Connect to server
client, err := tunnel.NewClient(tunnel.Config{
    Host:       "example.com",
    Port:       22,
    User:       "root",
    PrivateKey: privateKeyContent,
    Timeout:    30 * time.Second,
})
if err != nil {
    return err
}
defer client.Close()

if err := client.Connect(); err != nil {
    return err
}

// Create manager
mgr := tunnel.NewManager(client)

// Execute operations
err = mgr.CreateUser("appuser",
    tunnel.WithHome("/home/appuser"),
    tunnel.WithGroups("sudo", "docker"),
)

err = mgr.SetupSSHKeys("appuser", []string{publicKey})

err = mgr.InstallPackages("nginx", "docker", "git")

err = mgr.ServiceRestart("nginx")

// Deploy application
err = mgr.Deploy(tunnel.AppConfig{
    Name:       "myapp",
    Version:    "v1.2.3",
    Source:     "/local/app.tar.gz",
    Target:     "/opt/myapp",
    Service:    "myapp",
    PreDeploy:  []string{"systemctl stop myapp"},
    PostDeploy: []string{"systemctl start myapp"},
})
```

## Command Execution

```go
// Simple command
result, err := client.Execute("ls -la /var/log")
fmt.Println(result.Stdout)

// With sudo
result, err = client.ExecuteSudo("apt update")

// With options
result, err = client.Execute("./long-running-script.sh",
    tunnel.WithTimeout(5*time.Minute),
    tunnel.WithEnv("NODE_ENV", "production"),
    tunnel.WithWorkDir("/opt/app"),
)

// Stream output
err = client.Execute("tail -f /var/log/syslog",
    tunnel.WithStream(func(line string) {
        fmt.Println("LOG:", line)
    }),
)
```

## File Operations

```go
// Upload file
err := client.Upload("/local/config.yml", "/etc/app/config.yml")

// Download file
err := client.Download("/var/log/app.log", "/local/logs/app.log")

// Upload with progress
err := client.Upload("/local/large-file.tar.gz", "/tmp/large-file.tar.gz",
    tunnel.WithProgress(func(percent int) {
        fmt.Printf("Upload progress: %d%%\n", percent)
    }),
)
```

## Error Handling

```go
// Simple error types
type Error struct {
    Type    ErrorType // Connection, Auth, Execution, Timeout
    Message string
    Cause   error
}

// Usage
result, err := client.Execute("some-command")
if err != nil {
    if sshErr, ok := err.(*tunnel.Error); ok {
        switch sshErr.Type {
        case tunnel.ErrorConnection:
            // Handle connection error
        case tunnel.ErrorTimeout:
            // Handle timeout
        case tunnel.ErrorExecution:
            // Command failed, check result.ExitCode
        }
    }
}
```

## Configuration Options

```go
// User options
type UserOption func(*userConfig)

func WithHome(path string) UserOption
func WithShell(shell string) UserOption
func WithGroups(groups ...string) UserOption
func WithSudoAccess() UserOption

// Execution options
type ExecOption func(*execConfig)

func WithTimeout(d time.Duration) ExecOption
func WithEnv(key, value string) ExecOption
func WithWorkDir(dir string) ExecOption
func WithStream(handler func(string)) ExecOption

// App deployment options
type AppConfig struct {
    Name        string
    Version     string
    Source      string   // Local path or URL
    Target      string   // Remote path
    Service     string   // Service name to restart
    Backup      bool     // Backup before deploy
    PreDeploy   []string // Commands to run before
    PostDeploy  []string // Commands to run after
    HealthCheck string   // URL or command to verify
}
```

## Advanced Features (Optional)

### Batch Operations

```go
// Execute multiple commands in sequence
results, err := mgr.Batch(
    tunnel.Cmd("apt update"),
    tunnel.Cmd("apt upgrade -y"),
    tunnel.Cmd("apt autoremove -y"),
)

// Transaction-like operations with rollback
err := mgr.Transaction(func(tx *Transaction) error {
    if err := tx.Execute("stop-service.sh"); err != nil {
        return err // Will trigger rollback
    }
    if err := tx.Upload("new-config.yml", "/etc/app/"); err != nil {
        return err // Will trigger rollback
    }
    if err := tx.Execute("start-service.sh"); err != nil {
        return err // Will trigger rollback
    }
    return nil // Success, no rollback
})
```

### Templates

```go
// Common server setups as templates
err := mgr.ApplyTemplate(tunnel.TemplateWebServer{
    Domain:     "example.com",
    SSL:        true,
    PHP:        true,
    Database:   "mysql",
})

err := mgr.ApplyTemplate(tunnel.TemplateDocker{
    ComposeVersion: true,
    Swarm:          false,
})
```

## Testing

```go
// Mock client for testing
func TestDeployment(t *testing.T) {
    client := tunnel.NewMockClient()
    client.OnExecute("systemctl status myapp").Return(&tunnel.Result{
        Stdout:   "active (running)",
        ExitCode: 0,
    })

    mgr := tunnel.NewManager(client)
    err := mgr.ServiceStatus("myapp")
    assert.NoError(t, err)
}
```

## Benefits of Simplification

1. **No Connection Pool**: Single connection per operation, no pool management overhead
2. **Unified Manager**: One manager instead of multiple specialized ones
3. **Simple Options**: Use functional options pattern for flexibility
4. **Direct API**: Methods do what they say without layers of abstraction
5. **Clear Errors**: Simple error types that are easy to handle
6. **Less Code**: Easier to maintain and understand
7. **Practical Focus**: Built for real-world server management tasks

## Migration from Complex Version

```go
// Old (complex)
factory := tunnel.NewConnectionFactory(tracer)
pool := tunnel.NewPool(factory, poolConfig, tracer)
executor := tunnel.NewExecutor(pool, tracer)
setupMgr := tunnel.NewSetupManager(executor, tracer)
err := setupMgr.CreateUser(ctx, userConfig)

// New (simple)
client, _ := tunnel.NewClient(config)
client.Connect()
mgr := tunnel.NewManager(client)
err := mgr.CreateUser("username", tunnel.WithGroups("sudo"))
```

## Design Decisions

- **No context.Context everywhere**: Use timeouts in options instead
- **No spans/tracing by default**: Add simple hooks if needed
- **No complex error wrapping**: Simple error types with clear messages
- **No progress reporters**: Use simple callback functions
- **No dependency injection**: Direct instantiation with New functions
- **Single manager**: All operations through one interface
- **Functional options**: Clean API with sensible defaults
