# Tunnel Package

SSH automation library for PocketBase production deployment with agent authentication.

## Core Interfaces

```go
type SSHClient interface {
    Connect() error
    Execute(cmd string, opts ...ExecOption) (*Result, error)
    Upload(localPath, remotePath string) error
}

type Manager struct {
    CreateUser(username string, opts ...UserOption) error
    InstallPackages(packages ...string) error
    SystemInfo() (*SystemInfo, error)
}

type SetupManager struct {
    SetupPocketBaseServer(username string, publicKeys []string) error
    CreatePocketBaseDirectories(username string) error
}

type SecurityManager struct {
    SecureServer(config SecurityConfig) error
    SetupFirewall(rules []FirewallRule) error
    HardenSSH(config SSHConfig) error
}
```

## Files

**auth.go** - SSH agent authentication, host key verification, known_hosts cleanup  
**client.go** - SSH connection management, command execution, file transfer  
**manager.go** - System operations (users, packages, services, directories)  
**setup_manager.go** - PocketBase server setup and verification  
**security_manager.go** - Firewall, SSH hardening, fail2ban configuration  
**types.go** - Core interfaces, structs, options, errors

## Quick Usage

```go
client, _ := tunnel.NewClient(tunnel.Config{Host: "server.com", User: "root"})
defer client.Close() // Always cleanup

client.Connect()

mgr := tunnel.NewManager(client)
defer mgr.Close() // Cleanup manager

setup := tunnel.NewSetupManager(mgr)
defer setup.Close() // Cleanup setup manager

setup.SetupPocketBaseServer("pocketbase", publicKeys)

security := tunnel.NewSecurityManager(mgr)
defer security.Close() // Cleanup security manager

security.SecureServer(tunnel.SecurityConfig{...})
```

## Resource Management & Cleanup

All components support proper cleanup to prevent resource leaks:

```go
// Basic cleanup pattern
client, err := tunnel.NewClient(config)
if err != nil {
    return err
}
defer client.Close() // SSH connection + agent socket cleanup

// Using cleanup manager for complex scenarios
cleanup := tunnel.NewCleanupManager()
defer cleanup.Close()

client, _ := tunnel.NewClient(config)
cleanup.AddCloser(client)

mgr := tunnel.NewManager(client)
cleanup.AddCloser(mgr)

// Add custom cleanup functions
cleanup.Add(func() {
    fmt.Println("Custom cleanup executed")
})

// Signal handling (automatic in Client)
// Client handles SIGINT/SIGTERM gracefully
```

## Error Handling with Cleanup

```go
func deployWithCleanup() error {
    client, err := tunnel.NewClient(config)
    if err != nil {
        return err
    }
    
    return tunnel.WithCleanup(client.Close, func() error {
        if err := client.Connect(); err != nil {
            return err
        }
        
        // Your deployment logic here
        return performDeployment(client)
    })
}
```
