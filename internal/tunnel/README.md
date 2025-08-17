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
client.Connect()

mgr := tunnel.NewManager(client)
setup := tunnel.NewSetupManager(mgr)
setup.SetupPocketBaseServer("pocketbase", publicKeys)

security := tunnel.NewSecurityManager(mgr)
security.SecureServer(tunnel.SecurityConfig{...})
```
