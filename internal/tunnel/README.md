# Tunnel Package

A modern, dependency-injection based SSH client library for Go with connection pooling, health monitoring, and specialized operation managers.

## Overview

The `tunnel` package is a complete redesign of SSH connection management, addressing the architectural issues of the legacy SSH package through:

- **Dependency Injection**: No singletons, all dependencies are injected
- **Single Responsibility**: Each component has a clear, focused purpose
- **Interface-Driven**: Clean interfaces enable testing and modularity
- **Context Support**: Proper cancellation and timeout handling
- **Structured Error Handling**: Typed errors with retry logic

## Architecture

### Core Components

#### 1. SSHClient Interface
The fundamental SSH connection interface providing basic operations:
- Connect/Disconnect
- Command execution (sync and streaming)
- Connection health checks

#### 2. ConnectionFactory
Creates SSH clients with specified configurations:
- Authentication methods (key, agent, password)
- Host key verification modes
- Connection timeouts and retries

#### 3. Connection Pool
Manages multiple SSH connections efficiently:
- Connection reuse and lifecycle management
- Health monitoring and automatic recovery
- Cleanup of stale connections
- Thread-safe operations

#### 4. Executor
High-level command execution patterns:
- Command execution with sudo support
- Script execution
- File transfers
- Environment and timeout management

#### 5. Specialized Managers
Domain-specific operation managers:
- **SetupManager**: User creation, SSH keys, directories
- **SecurityManager**: Firewall, fail2ban, SSH hardening
- **ServiceManager**: Systemd service operations

## Key Improvements Over Legacy SSH Package

### Before (Legacy SSH Package)
```go
// Singleton pattern with tight coupling
sshManager := ssh.GetGlobalSSHManager()
pool := ssh.GetConnectionPool()
health := ssh.GetHealthMonitor()

// Circular dependencies
// pool → health → pool
```

### After (Tunnel Package)
```go
// Dependency injection with clear ownership
tracerFactory := tracer.SetupProductionTracing(os.Stdout)
sshTracer := tracerFactory.CreateSSHTracer()
factory := tunnel.NewConnectionFactory(sshTracer)
pool := tunnel.NewPool(factory, poolConfig, sshTracer)
executor := tunnel.NewExecutor(pool, sshTracer)

// No circular dependencies
// Clear hierarchy: executor → pool → factory
```

## Usage Examples

### Basic Connection
```go
// Create configuration
config := tunnel.ConnectionConfig{
    Host:       "example.com",
    Port:       22,
    Username:   "deploy",
    AuthMethod: tunnel.AuthMethod{
        Type:    "key",
        KeyPath: "/home/user/.ssh/id_rsa",
    },
    Timeout:     30 * time.Second,
    HostKeyMode: tunnel.HostKeyAcceptNew,
}

// Create client
tracerFactory := tracer.SetupDevelopmentTracing()
sshTracer := tracerFactory.CreateSSHTracer()
factory := tunnel.NewConnectionFactory(sshTracer)
client, err := factory.Create(config)
if err != nil {
    return err
}
defer client.Close()

// Execute command
output, err := client.Execute(ctx, "ls -la")
```

### Connection Pool
```go
// Create pool configuration
poolConfig := tunnel.PoolConfig{
    MaxConnections:  10,
    MaxIdleTime:     15 * time.Minute,
    HealthInterval:  30 * time.Second,
    CleanupInterval: 5 * time.Minute,
}

// Create pool
pool := tunnel.NewPool(factory, poolConfig, sshTracer)
defer pool.Close()

// Get connection
client, err := pool.Get(ctx, "server-1")
if err != nil {
    return err
}
defer pool.Release("server-1", client)

// Use client
output, err := client.Execute(ctx, "uptime")
```

### High-Level Executor
```go
// Create executor
executor := tunnel.NewExecutor(pool, sshTracer)

// Run command with options
cmd := tunnel.Command{
    Cmd:     "systemctl restart nginx",
    Sudo:    true,
    Timeout: 30 * time.Second,
    Environment: map[string]string{
        "DEBIAN_FRONTEND": "noninteractive",
    },
}

result, err := executor.RunCommand(ctx, cmd)
if err != nil {
    return err
}

fmt.Printf("Command completed in %v\n", result.Duration)
```

### Server Setup Operations
```go
// Create setup manager
setupTracer := tracerFactory.CreateServiceTracer()
setupMgr := tunnel.NewSetupManager(executor, setupTracer)

// Create user
userConfig := tunnel.UserConfig{
    Username:   "appuser",
    HomeDir:    "/home/appuser",
    Shell:      "/bin/bash",
    Groups:     []string{"sudo", "docker"},
    CreateHome: true,
}

err := setupMgr.CreateUser(ctx, userConfig)
if err != nil {
    return err
}

// Setup SSH keys
keys := []string{
    "ssh-rsa AAAAB3NzaC1yc2EA...",
    "ssh-ed25519 AAAAC3NzaC1l...",
}

err = setupMgr.SetupSSHKeys(ctx, "appuser", keys)
```

### Security Lockdown
```go
// Create security manager
securityTracer := tracerFactory.CreateSecurityTracer()
securityMgr := tunnel.NewSecurityManager(executor, securityTracer)

// Apply lockdown
config := tunnel.SecurityConfig{
    DisableRootLogin:    true,
    DisablePasswordAuth: true,
    AllowedPorts:        []int{22, 80, 443},
    Fail2banConfig: tunnel.Fail2banConfig{
        Enabled:    true,
        MaxRetries: 5,
        BanTime:    3600 * time.Second,
        Services:   []string{"sshd"},
    },
}

err := securityMgr.ApplyLockdown(ctx, config)
```

### Service Management
```go
// Create service manager
serviceTracer := tracerFactory.CreateServiceTracer()
serviceMgr := tunnel.NewServiceManager(executor, serviceTracer)

// Manage service
err := serviceMgr.ManageService(ctx, tunnel.ServiceRestart, "nginx")
if err != nil {
    return err
}

// Get service status
status, err := serviceMgr.GetServiceStatus(ctx, "nginx")
if err != nil {
    return err
}

fmt.Printf("Service %s is %s\n", status.Name, status.State)
```

## Error Handling

The package provides structured error types with classification and retry logic:

```go
result, err := executor.RunCommand(ctx, cmd)
if err != nil {
    // Trace the error
    span := tracer.StartSpan(ctx, "error_handling")
    defer span.EndWithError(err)
    
    // Check if error is retryable
    if tunnel.IsRetryable(err) {
        span.Event("retry_attempted", tracer.Int("attempt", 1))
        // Implement retry logic
        return retryWithBackoff(func() error {
            return executor.RunCommand(ctx, cmd)
        })
    }

    // Check error type
    if tunnel.IsAuthError(err) {
        span.SetField("error.type", "authentication")
        return fmt.Errorf("authentication failed, check credentials")
    }

    if tunnel.IsConnectionError(err) {
        span.SetField("error.type", "connection")
        return fmt.Errorf("connection failed, check network")
    }

    return err
}
```

## Health Monitoring

Built-in health monitoring with tracing:

```go
// Get health report with tracing
poolTracer := tracerFactory.CreatePoolTracer()
healthSpan := poolTracer.TraceHealthCheck(ctx)
defer healthSpan.End()

report := pool.HealthCheck(ctx)

// Record health metrics
tracer.RecordPoolHealth(healthSpan, report.TotalConnections, 
    report.HealthyConnections, report.FailedConnections)

healthSpan.SetFields(tracer.Fields{
    "pool.total": report.TotalConnections,
    "pool.healthy": report.HealthyConnections,
    "pool.failed": report.FailedConnections,
})

fmt.Printf("Total connections: %d\n", report.TotalConnections)
fmt.Printf("Healthy: %d\n", report.HealthyConnections)
fmt.Printf("Failed: %d\n", report.FailedConnections)
```

## Progress Reporting

Track long-running operations with tracing:

```go
// Create progress span
setupTracer := tracerFactory.CreateServiceTracer()
progressSpan := setupTracer.TraceDeployment(ctx, "user_setup", "v1.0")
defer progressSpan.End()

// Add progress events
progressSpan.Event("setup_started")
progressSpan.Event("validation_completed", tracer.Int("progress", 25))
progressSpan.Event("user_created", tracer.Int("progress", 50))
progressSpan.Event("keys_configured", tracer.Int("progress", 75))
progressSpan.Event("setup_completed", tracer.Int("progress", 100))

// Execute with progress tracking
err := setupMgr.CreateUserWithProgress(ctx, userConfig, progressSpan)
```

## Testing

The interface-driven design enables comprehensive testing:

```go
// Mock SSH client
type mockClient struct {
    mock.Mock
}

func (m *mockClient) Execute(ctx context.Context, cmd string) (string, error) {
    args := m.Called(ctx, cmd)
    return args.String(0), args.Error(1)
}

// Test with mock and tracer
func TestExecutor_RunCommand(t *testing.T) {
    mockPool := &mockPool{}
    mockClient := &mockClient{}
    tracerFactory := tracer.SetupTestTracing(t)
    defer tracerFactory.Shutdown(context.Background())
    
    sshTracer := tracerFactory.CreateSSHTracer()

    mockPool.On("Get", mock.Anything, "test-key").Return(mockClient, nil)
    mockClient.On("Execute", mock.Anything, "echo test").Return("test", nil)

    executor := tunnel.NewExecutor(mockPool, sshTracer)

    result, err := executor.RunCommand(context.Background(), tunnel.Command{
        Cmd: "echo test",
    })

    assert.NoError(t, err)
    assert.Equal(t, "test", result.Output)
    mockClient.AssertExpectations(t)
}
```

## Migration from Legacy SSH Package

### Phase 1: Use New Interfaces
```go
// Old code
manager := ssh.NewSSHManager(server, true)

// New code
tracerFactory := tracer.SetupProductionTracing(os.Stdout)
sshTracer := tracerFactory.CreateSSHTracer()
config := tunnel.ConnectionConfig{
    Host:     server.Host,
    Port:     server.Port,
    Username: server.RootUsername,
    // ... other config
}
factory := tunnel.NewConnectionFactory(sshTracer)
client, err := factory.Create(config)
```

### Phase 2: Replace Connection Pool
```go
// Old code
pool := ssh.GetConnectionPool()
conn, err := pool.GetOrCreateConnection(server, asRoot)

// New code
pool := tunnel.NewPool(factory, poolConfig, sshTracer)
client, err := pool.Get(ctx, connectionKey)
```

### Phase 3: Update Operations
```go
// Old code
err := manager.RunServerSetup(progressChan)

// New code
setupTracer := tracerFactory.CreateServiceTracer()
setupMgr := tunnel.NewSetupManager(executor, setupTracer)
err := setupMgr.CreateUser(ctx, userConfig)
```

## Configuration

### Connection Configuration
- Host, port, username
- Authentication methods
- Timeout and retry settings
- Host key verification mode

### Pool Configuration
- Maximum connections
- Idle timeout
- Health check interval
- Cleanup interval

### Security Configuration
- SSH hardening settings
- Firewall rules
- Fail2ban configuration
- Allowed users and ports

## Performance Considerations

- **Connection Reuse**: Pool maintains healthy connections for reuse
- **Concurrent Operations**: Thread-safe pool and client operations
- **Resource Cleanup**: Automatic cleanup of stale connections
- **Rate Limiting**: Built-in rate limiter for operation throttling

## Best Practices

1. **Always use context**: Pass context for cancellation and timeouts
2. **Release connections**: Always return connections to the pool
3. **Handle errors properly**: Check error types and retry when appropriate
4. **Monitor health**: Use health reports to track connection status
5. **Log appropriately**: Use structured logging with proper levels

## File Structure

```
internal/tunnel/
├── interfaces.go      # Core interfaces and contracts
├── types.go          # Type definitions
├── constants.go      # Constants and defaults
├── errors.go         # Error types and handling
├── client.go         # SSH client implementation
├── factory.go        # Connection factory
├── pool.go          # Connection pool
├── executor.go      # Command executor
├── health.go        # Health monitoring
├── troubleshoot.go  # Diagnostics
├── managers/
│   ├── setup.go     # Setup operations
│   ├── security.go  # Security operations
│   └── service.go   # Service management
└── README.md        # This file
```
