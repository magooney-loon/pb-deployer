# Tunnel Package - PocketBase Production Deployment

A streamlined SSH client library specifically designed for automating PocketBase production server setup and deployment.

## Core Purpose

Automates the lifecycle of deploying PocketBase apps to production servers through:

1. **Server Setup**: Automated user creation and directory structure (`/opt/pocketbase/apps/`)
2. **Security Hardening**: Firewall, fail2ban, SSH hardening
3. **Deployment**: SFTP transfer protocol & systemd service management (coming soon)

**Production Focus**: This tool is designed exclusively for production deployment with SSH agent authentication and strict security practices.

## Architecture

### 1. SSH Client (Single Connection)

```go
// Client represents a single SSH connection to a server
type Client struct {
    config   Config
    conn     *ssh.Client
    sftp     *sftp.Client
    tracer   Tracer
}

// Basic operations
func NewClient(config Config) (*Client, error)
func (c *Client) Connect() error
func (c *Client) Close() error
func (c *Client) Execute(cmd string, opts ...ExecOption) (*Result, error)
func (c *Client) ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error)
func (c *Client) Upload(localPath, remotePath string) error
func (c *Client) Download(remotePath, localPath string) error
```

### 2. Setup Manager (PocketBase Server Setup)

```go
// SetupManager handles server setup operations for PocketBase deployment
type SetupManager struct {
    manager *Manager
}

func NewSetupManager(manager *Manager) *SetupManager

// Main setup function
func (s *SetupManager) SetupPocketBaseServer(username string, publicKeys []string) error

// Core setup operations
func (s *SetupManager) CreatePocketBaseDirectories(username string) error
func (s *SetupManager) UpdateSystem() error
func (s *SetupManager) InstallEssentials() error
func (s *SetupManager) VerifySetup(username string) error
func (s *SetupManager) GetSetupInfo() (*SetupInfo, error)
```

### 3. Security Manager (Server Hardening)

```go
// SecurityManager handles server security and hardening operations
type SecurityManager struct {
    manager *Manager
}

func NewSecurityManager(manager *Manager) *SecurityManager

// Main security function
func (s *SecurityManager) SecureServer(config SecurityConfig) error

// Security operations
func (s *SecurityManager) SetupFirewall(rules []FirewallRule) error
func (s *SecurityManager) HardenSSH(config SSHConfig) error
func (s *SecurityManager) SetupFail2ban() error
func (s *SecurityManager) GetDefaultPocketBaseRules() []FirewallRule
func (s *SecurityManager) GetDefaultSSHConfig() SSHConfig
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"
    "github.com/magooney-loon/pb-deployer/internal/tunnel"
)

func main() {
    // 1. Ensure SSH agent is available
    if !tunnel.IsAgentAvailable() {
        panic("SSH agent is required but not available")
    }

    // 2. Connect to server (SSH agent auth only)
    client, err := tunnel.NewClient(tunnel.Config{
        Host:    "your-server.com",
        Port:    22,
        User:    "root",
        Timeout: 30 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    if err := client.Connect(); err != nil {
        panic(err)
    }

    // 3. Create managers
    mgr := tunnel.NewManager(client)
    setupMgr := tunnel.NewSetupManager(mgr)
    securityMgr := tunnel.NewSecurityManager(mgr)

    // 4. Setup PocketBase server
    publicKeys := []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB..."}
    err = setupMgr.SetupPocketBaseServer("pocketbase", publicKeys)
    if err != nil {
        panic(err)
    }

    // 5. Secure the server
    securityConfig := tunnel.SecurityConfig{
        FirewallRules:  securityMgr.GetDefaultPocketBaseRules(),
        HardenSSH:      true,
        SSHConfig:      securityMgr.GetDefaultSSHConfig(),
        EnableFail2ban: true,
    }

    err = securityMgr.SecureServer(securityConfig)
    if err != nil {
        panic(err)
    }

    fmt.Println("PocketBase production server setup completed!")
}
```

## Core Workflow

### Server Setup

```go
// Setup PocketBase server with user and directory structure
err := setupMgr.SetupPocketBaseServer("pocketbase", []string{
    "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB...", // your public key
})

// Verify setup was successful
err = setupMgr.VerifySetup("pocketbase")

// Get setup information
info, err := setupMgr.GetSetupInfo()
fmt.Printf("OS: %s, Apps: %v\n", info.OS, info.InstalledApps)
```

Creates:
- User `pocketbase` with sudo access
- Directory structure: `/opt/pocketbase/{apps,backups,logs,scripts}/`
- Installs essentials: `curl`, `wget`, `unzip`, `systemd`, `logrotate`

### Security Hardening

```go
// Get default PocketBase firewall rules
rules := securityMgr.GetDefaultPocketBaseRules()
// Returns: SSH(22), HTTP(80), HTTPS(443), PocketBase Admin(8080), API(8090)

// Apply security configuration
config := tunnel.SecurityConfig{
    FirewallRules:  rules,
    HardenSSH:      true,
    SSHConfig: tunnel.SSHConfig{
        PasswordAuth: false,  // Disable password auth
        RootLogin:    false,  // Disable root login
        PubkeyAuth:   true,   // Enable key auth only
        MaxAuthTries: 3,
    },
    EnableFail2ban: true,
}

err := securityMgr.SecureServer(config)
```

## Directory Structure Created

```
/opt/pocketbase/
├── apps/          # PocketBase application instances
├── backups/       # Backup storage for rollbacks
├── logs/          # Application logs
└── scripts/       # Utility scripts
```

## Firewall Rules (Default)

- **SSH (22/tcp)**: Remote access
- **HTTP (80/tcp)**: Web traffic
- **HTTPS (443/tcp)**: Secure web traffic
- **PocketBase Admin (8080/tcp)**: Admin dashboard
- **PocketBase API (8090/tcp)**: API endpoints

## SSH Hardening (Default)

- Disable password authentication
- Disable root login
- Enable public key authentication only
- Limit max auth tries to 3
- Configure connection timeouts

## Error Handling

```go
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
        case tunnel.ErrorVerification:
            // Setup verification failed
        }
    }
}
```

## Configuration Options

### Connection Config
```go
type Config struct {
    Host           string        // Server hostname/IP
    Port           int           // SSH port (default: 22)
    User           string        // SSH username
    KnownHostsFile string        // Optional custom known_hosts file path
    Timeout        time.Duration // Connection timeout
    RetryCount     int           // Connection retries
    RetryDelay     time.Duration // Retry delay
}
```

### Authentication and Security

The tunnel package uses SSH agent authentication exclusively with strict host key verification for production security.

#### SSH Agent Authentication (Required)

```go
// Check if SSH agent is available
if !tunnel.IsAgentAvailable() {
    log.Fatal("SSH agent is required but not available")
}

// Create client - automatically uses SSH agent
config := tunnel.Config{
    Host: "server.com",
    Port: 22,
    User: "root",
    Timeout: 30 * time.Second,
}

client, err := tunnel.NewClient(config)
if err != nil {
    log.Fatal(err)
}
```

#### Host Key Verification

Production deployment uses strict host key verification with known_hosts:

```go
// Default configuration uses ~/.ssh/known_hosts
config := tunnel.Config{
    Host: "server.com",
    User: "root",
    // KnownHostsFile: "", // Uses ~/.ssh/known_hosts by default
}

// Custom known_hosts file location
config := tunnel.Config{
    Host: "server.com",
    User: "root",
    KnownHostsFile: "/custom/path/known_hosts",
}
```

#### Prerequisites

Before using the tunnel package:

1. **SSH Agent**: Ensure SSH agent is running with your keys loaded
2. **Known Hosts**: Add server host keys to your known_hosts file
3. **Server Access**: Ensure you have root or sudo access on target server

```bash
# Start SSH agent and add keys
eval $(ssh-agent)
ssh-add ~/.ssh/id_rsa

# Add server to known_hosts (first connection)
ssh-keyscan your-server.com >> ~/.ssh/known_hosts

# Verify SSH agent has keys
ssh-add -l
```

### Execution Options
```go
func WithTimeout(d time.Duration) ExecOption
func WithEnv(key, value string) ExecOption
func WithWorkDir(dir string) ExecOption
func WithSudo() ExecOption
```

### User Creation Options
```go
func WithHome(path string) UserOption
func WithShell(shell string) UserOption
func WithGroups(groups ...string) UserOption
func WithSudoAccess() UserOption
func WithSystemUser() UserOption
```

## Design Principles

1. **Production Focus**: Built exclusively for production PocketBase deployment
2. **SSH Agent Only**: Simplified authentication using SSH agent exclusively
3. **Security First**: Strict host key verification and security hardening by default
4. **Single Connection**: One server at a time, no connection pooling complexity
5. **Simple Operations**: Direct API without excessive abstraction
6. **Clear Errors**: Simple error types that are easy to understand
7. **Minimal Dependencies**: Keep it simple and maintainable

## Dependencies

- `golang.org/x/crypto/ssh`: SSH client implementation, agent support, and host key verification
- `github.com/pkg/sftp`: SFTP file transfer support

## Future Features

- **Deployment Manager**: PocketBase app deployment and version management
- **Service Templates**: Systemd service file generation
- **SSL/TLS Setup**: Automatic certificate management
- **Monitoring**: Basic health checks and log management

This package is specifically designed for production PocketBase deployment with enterprise-grade security practices.
