# Tunnel Package - PocketBase Deployment Automation

A streamlined SSH client library specifically designed for automating PocketBase server setup and deployment.

## Core Purpose

Automates the lifecycle of deploying PocketBase apps to production servers through:

1. **Server Setup**: Automated user creation and directory structure (`/opt/pocketbase/apps/`)
2. **Security Lockdown**: Firewall, fail2ban, disable root SSH
3. **Deployment**: SFTP transfer protocol & systemd service management (coming soon)

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
    "github.com/magooney-loon/pb-deployer/internal/tunnel"
)

func main() {
    // 1. Connect to server
    client, err := tunnel.NewClient(tunnel.Config{
        Host:    "your-server.com",
        Port:    22,
        User:    "root",
        Auth:    tunnel.AuthConfigFromKeyPath("~/.ssh/id_rsa"),
        Timeout: 30 * time.Second,
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    if err := client.Connect(); err != nil {
        panic(err)
    }

    // 2. Create managers
    mgr := tunnel.NewManager(client)
    setupMgr := tunnel.NewSetupManager(mgr)
    securityMgr := tunnel.NewSecurityManager(mgr)

    // 3. Setup PocketBase server
    err = setupMgr.SetupPocketBaseServer("pocketbase", []string{publicKey})
    if err != nil {
        panic(err)
    }

    // 4. Secure the server
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

    fmt.Println("PocketBase server setup completed!")
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
    Host       string        // Server hostname/IP
    Port       int           // SSH port (default: 22)
    User       string        // SSH username
    Auth       AuthConfig    // Authentication configuration
    Timeout    time.Duration // Connection timeout
    RetryCount int           // Connection retries
    RetryDelay time.Duration // Retry delay
}
```

### Authentication and Security

The tunnel package supports two authentication methods with comprehensive security features:

#### SSH Agent (Recommended)
```go
// Use SSH agent if available (auto-detected)
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfigWithAgent(),
}

// Or with fallback to auto-detected keys
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.DefaultAuthConfig(), // Uses agent if available, falls back to ~/.ssh keys
}
```

#### Private Key File
```go
// Specific key file
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfigFromKeyPath("~/.ssh/id_rsa"),
}

// Key file with passphrase
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfigFromKeyPath("~/.ssh/id_rsa", "passphrase"),
}

// Custom auth configuration
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfig{
        UseAgent:      IsAgentAvailable(),
        KeyPath:       "~/.ssh/custom_key",
        KeyPassphrase: "secret",
        PreferAgent:   true, // Try agent first, fallback to key file
        HostKeyVerification: tunnel.HostKeyConfig{
            Mode:                  tunnel.HostKeyModeAcceptNew,
            StrictHostKeyChecking: false,
            AcceptNewKeys:         true,
            KnownHostsFile:        "~/.ssh/known_hosts",
        },
    },
}
```

#### Host Key Verification

The tunnel package provides secure host key verification to prevent man-in-the-middle attacks:

```go
// Strict host key checking (production recommended)
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfig{
        UseAgent: true,
        HostKeyVerification: tunnel.StrictHostKeyConfig(),
    },
}

// Accept new keys and add to known_hosts (development friendly)
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfig{
        KeyPath: "~/.ssh/id_rsa",
        HostKeyVerification: tunnel.DefaultHostKeyConfig(), // Accepts new keys
    },
}

// Custom known_hosts file location
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfig{
        UseAgent: true,
        HostKeyVerification: tunnel.HostKeyConfig{
            Mode: tunnel.HostKeyModeStrict,
            KnownHostsFile: "/custom/path/known_hosts",
        },
    },
}

// Insecure mode (NOT RECOMMENDED for production)
config := tunnel.Config{
    Host: "server.com",
    User: "user",
    Auth: tunnel.AuthConfig{
        KeyPath: "~/.ssh/id_rsa",
        HostKeyVerification: tunnel.InsecureHostKeyConfig(),
    },
}
```

#### Host Key Validation

```go
// Validate a host key against known_hosts
err := tunnel.ValidateHostKey("server.com", publicKey)
if err != nil {
    fmt.Printf("Host key validation failed: %v\n", err)
}

// Validate against custom known_hosts file
err = tunnel.ValidateHostKey("server.com", publicKey, "/custom/known_hosts")
```

#### Authentication Validation
```go
// Check if SSH agent is available
if tunnel.IsAgentAvailable() {
    fmt.Println("SSH agent detected")
}

// Validate a key file before use
if err := tunnel.ValidateKeyFile("~/.ssh/id_rsa"); err != nil {
    fmt.Printf("Key validation failed: %v\n", err)
}

// Get key fingerprint
fingerprint, err := tunnel.GetKeyFingerprint("~/.ssh/id_rsa")
if err == nil {
    fmt.Printf("Key fingerprint: %s\n", fingerprint)
}

// Validate host key verification setup
hostKeyConfig := tunnel.DefaultHostKeyConfig()
callback, err := tunnel.GetHostKeyCallback(hostKeyConfig)
if err == nil {
    fmt.Println("✓ Host key verification configured")
}
```

### Host Key Verification Modes

```go
// Strict mode - requires keys to exist in known_hosts
type HostKeyConfig struct {
    Mode                  HostKeyMode // HostKeyModeStrict
    StrictHostKeyChecking bool        // true
    AcceptNewKeys         bool        // false
    KnownHostsFile        string      // "~/.ssh/known_hosts"
}

// Accept new mode - adds unknown keys to known_hosts
type HostKeyConfig struct {
    Mode                  HostKeyMode // HostKeyModeAcceptNew
    StrictHostKeyChecking bool        // false
    AcceptNewKeys         bool        // true
    KnownHostsFile        string      // "~/.ssh/known_hosts"
}

// Insecure mode - no host key verification (NOT RECOMMENDED)
type HostKeyConfig struct {
    Mode HostKeyMode // HostKeyModeInsecure
}

// Custom mode - use your own callback function
type HostKeyConfig struct {
    Mode            HostKeyMode         // HostKeyModeCustom
    HostKeyCallback ssh.HostKeyCallback // Your custom function
}
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

1. **PocketBase Focus**: Built specifically for PocketBase deployment workflows
2. **Single Connection**: One server at a time, no connection pooling complexity
3. **Simple Operations**: Direct API without excessive abstraction
4. **Practical Defaults**: Sensible security defaults for production servers
5. **Clear Errors**: Simple error types that are easy to understand
6. **Minimal Dependencies**: Keep it simple and maintainable
7. **Security First**: Proper host key verification and authentication by default

## Dependencies

- `golang.org/x/crypto/ssh`: SSH client implementation, agent support, and host key verification
- `github.com/pkg/sftp`: SFTP file transfer support

## Future Features

- **Deployment Manager**: PocketBase app deployment and version management
- **Service Templates**: Systemd service file generation
- **SSL/TLS Setup**: Automatic certificate management
- **Monitoring**: Basic health checks and log management

This package is specifically designed for the PocketBase deployment workflow described in the main project README.
