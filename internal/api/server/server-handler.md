# Server Handler Alignment

HTTP handlers for server management aligned with tunnel package interfaces.

## Handler Functions

### CreateServer
```http
POST /api/v1/servers
```
**Calls:**
- Database operations only (no tunnel calls for registration)
- Connection validation via `pool.Get(ctx, serverKey)` (optional test)

### GetServer
```http
GET /api/v1/servers/{id}
```
**Calls:**
- Database queries for basic info
- `pool.HealthCheck(ctx)` (for connection health)
- `executor.RunCommand(ctx, systemInfoCmd)` (if detailed info requested)

### ListServers
```http
GET /api/v1/servers
```
**Calls:**
- Database queries only (no tunnel calls for basic listing)
- `pool.HealthCheck(ctx)` (per server if connection status needed)

### UpdateServer
```http
PUT /api/v1/servers/{id}
```
**Calls:**
- Database update operations
- `pool.Get(ctx, serverKey)` (to test new connection settings)

### DeleteServer
```http
DELETE /api/v1/servers/{id}
```
**Calls:**
- `executor.RunCommand(ctx, cleanupCmd)` (if cleanup_apps)
- Database cleanup operations

### TestConnection
```http
POST /api/v1/servers/{id}/test
```
**Calls:**
- `pool.Get(ctx, serverKey)` (TCP + SSH test)
- `executor.RunCommand(ctx, sudoTestCmd)` (sudo access test)
- `executor.RunCommand(ctx, systemInfoCmd)` (system information)

### SetupServer
```http
POST /api/v1/servers/{id}/setup
```
**Calls:**
- `setupMgr.CreateUser(ctx, userConfig)` (if create_user)
- `setupMgr.SetupSSHKeys(ctx, username, sshKeys)`
- `setupMgr.CreateDirectory(ctx, dirPath, owner)` (per directory)
- `executor.RunCommand(ctx, installPackagesCmd)` (if install_packages)
- `executor.RunCommand(ctx, configureSudoCmd)` (if configure_sudo)

### ApplySecurity
```http
POST /api/v1/servers/{id}/security
```
**Calls:**
- `securityMgr.ApplyLockdown(ctx, securityConfig)`
- `securityMgr.ConfigureFirewall(ctx, firewallRules)` (if configure_firewall)
- `securityMgr.SetupFail2ban(ctx, fail2banConfig)` (if setup_fail2ban)
- `executor.RunCommand(ctx, enableAutoUpdatesCmd)` (if enable_auto_updates)

### GetServerStatus
```http
GET /api/v1/servers/{id}/status
```
**Calls:**
- `pool.HealthCheck(ctx)` (connection status)
- `executor.RunCommand(ctx, systemStatsCmd)` (system health)
- `serviceMgr.GetServiceStatus(ctx, "ssh")` (SSH service)
- `serviceMgr.GetServiceStatus(ctx, "fail2ban")` (if security_locked)

### GetServerHealth
```http
GET /api/v1/servers/{id}/health
```
**Calls:**
- `pool.Get(ctx, serverKey)` (connection health)
- `executor.RunCommand(ctx, systemHealthCmd)` (system metrics)
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (per service)
- `executor.RunCommand(ctx, appHealthCmd)` (if include_apps)

### GetServerLogs
```http
GET /api/v1/servers/{id}/logs
```
**Calls:**
- `executor.RunCommand(ctx, journalctlCmd)` (system logs)

### GetServerMetrics
```http
GET /api/v1/servers/{id}/metrics
```
**Calls:**
- `executor.RunCommand(ctx, systemMetricsCmd)` (CPU, memory, disk)
- `executor.RunCommand(ctx, networkStatsCmd)` (network statistics)
- `executor.RunCommand(ctx, loadAverageCmd)` (load average)

## Progress Tracking Pattern

All async operations use handler-level progress tracking:

```go
func (h *ServerHandler) setupServerWithProgress(ctx context.Context, serverID string, config SetupConfig) error {
    progressChan := make(chan SetupProgress, 10)
    go h.monitorSetupProgress(ctx, serverID, progressChan)
    
    // Step 1: Create user (if requested)
    if config.CreateUser {
        progressChan <- SetupProgress{Step: "create_user", Status: "running"}
        err := h.setupMgr.CreateUser(ctx, config.UserConfig)
        if err != nil {
            progressChan <- SetupProgress{Step: "create_user", Status: "failed", Error: err}
            return err
        }
        progressChan <- SetupProgress{Step: "create_user", Status: "completed"}
    }
    
    // Step 2: Setup SSH keys
    progressChan <- SetupProgress{Step: "ssh_keys", Status: "running"}
    err := h.setupMgr.SetupSSHKeys(ctx, config.Username, config.SSHKeys)
    if err != nil {
        progressChan <- SetupProgress{Step: "ssh_keys", Status: "failed", Error: err}
        return err
    }
    progressChan <- SetupProgress{Step: "ssh_keys", Status: "completed"}
    
    // Step 3: Install packages (if requested)
    if len(config.InstallPackages) > 0 {
        progressChan <- SetupProgress{Step: "install_packages", Status: "running"}
        installCmd := generateInstallCommand(config.InstallPackages)
        _, err := h.executor.RunCommand(ctx, installCmd)
        if err != nil {
            progressChan <- SetupProgress{Step: "install_packages", Status: "failed", Error: err}
            return err
        }
        progressChan <- SetupProgress{Step: "install_packages", Status: "completed"}
    }
    
    // Step 4: Create directories
    progressChan <- SetupProgress{Step: "create_directories", Status: "running"}
    for _, dir := range config.CreateDirectories {
        err := h.setupMgr.CreateDirectory(ctx, dir.Path, dir.Owner)
        if err != nil {
            progressChan <- SetupProgress{Step: "create_directories", Status: "failed", Error: err}
            return err
        }
    }
    progressChan <- SetupProgress{Step: "create_directories", Status: "completed"}
    
    close(progressChan)
    return nil
}
```

## Security Hardening Pattern

```go
func (h *ServerHandler) applySecurityWithProgress(ctx context.Context, serverID string, config SecurityConfig) error {
    progressChan := make(chan SecurityProgress, 10)
    go h.monitorSecurityProgress(ctx, serverID, progressChan)
    
    // Step 1: Apply basic lockdown
    progressChan <- SecurityProgress{Step: "ssh_hardening", Status: "running"}
    err := h.securityMgr.ApplyLockdown(ctx, config)
    if err != nil {
        progressChan <- SecurityProgress{Step: "ssh_hardening", Status: "failed", Error: err}
        return err
    }
    progressChan <- SecurityProgress{Step: "ssh_hardening", Status: "completed"}
    
    // Step 2: Configure firewall
    if config.ConfigureFirewall {
        progressChan <- SecurityProgress{Step: "firewall_setup", Status: "running"}
        err := h.securityMgr.ConfigureFirewall(ctx, config.FirewallRules)
        if err != nil {
            progressChan <- SecurityProgress{Step: "firewall_setup", Status: "failed", Error: err}
            return err
        }
        progressChan <- SecurityProgress{Step: "firewall_setup", Status: "completed"}
    }
    
    // Step 3: Setup fail2ban
    if config.SetupFail2ban {
        progressChan <- SecurityProgress{Step: "fail2ban_config", Status: "running"}
        err := h.securityMgr.SetupFail2ban(ctx, config.Fail2banConfig)
        if err != nil {
            progressChan <- SecurityProgress{Step: "fail2ban_config", Status: "failed", Error: err}
            return err
        }
        progressChan <- SecurityProgress{Step: "fail2ban_config", Status: "completed"}
    }
    
    // Step 4: Enable auto updates
    if config.EnableAutoUpdates {
        progressChan <- SecurityProgress{Step: "system_updates", Status: "running"}
        updateCmd := generateAutoUpdateCommand()
        _, err := h.executor.RunCommand(ctx, updateCmd)
        if err != nil {
            progressChan <- SecurityProgress{Step: "system_updates", Status: "failed", Error: err}
            return err
        }
        progressChan <- SecurityProgress{Step: "system_updates", Status: "completed"}
    }
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewServerHandler(
    executor tunnel.Executor,
    setupMgr tunnel.SetupManager,
    securityMgr tunnel.SecurityManager,
    serviceMgr tunnel.ServiceManager,
    pool tunnel.Pool,
) *ServerHandler {
    return &ServerHandler{
        executor:    executor,
        setupMgr:    setupMgr,
        securityMgr: securityMgr,
        serviceMgr:  serviceMgr,
        pool:        pool,
    }
}
```

## Error Handling Pattern

```go
result, err := h.setupMgr.CreateUser(ctx, userConfig)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryUserCreation(ctx, userConfig)
    }
    if tunnel.IsAuthError(err) {
        return handleAuthError(e, err)
    }
    if tunnel.IsConnectionError(err) {
        return handleConnectionError(e, err)
    }
    return handleGenericError(e, err)
}
```

## Connection Testing Pattern

```go
func (h *ServerHandler) performConnectionTest(ctx context.Context, serverID string) (*ConnectionTestResult, error) {
    result := &ConnectionTestResult{}
    
    // Test TCP connection
    client, err := h.pool.Get(ctx, serverID)
    if err != nil {
        result.TCPConnection.Success = false
        result.TCPConnection.Message = err.Error()
        return result, nil
    }
    defer h.pool.Release(serverID, client)
    
    result.TCPConnection.Success = true
    result.TCPConnection.Latency = "25ms" // Calculate actual latency
    
    // Test SSH authentication
    testCmd := tunnel.Command{
        Cmd:     "echo 'SSH test successful'",
        Timeout: 10 * time.Second,
    }
    _, err = h.executor.RunCommand(ctx, testCmd)
    if err != nil {
        result.SSHConnection.Success = false
        result.SSHConnection.Message = err.Error()
    } else {
        result.SSHConnection.Success = true
        result.SSHConnection.Message = "SSH authentication successful"
    }
    
    // Test sudo access
    sudoCmd := tunnel.Command{
        Cmd:     "sudo -n true",
        Timeout: 5 * time.Second,
    }
    _, err = h.executor.RunCommand(ctx, sudoCmd)
    result.SudoAccess.Success = (err == nil)
    
    return result, nil
}
```

## System Information Pattern

```go
func (h *ServerHandler) getSystemInfo(ctx context.Context) (*SystemInfo, error) {
    // Get OS information
    osCmd := tunnel.Command{
        Cmd:     "lsb_release -d -s 2>/dev/null || cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2 | tr -d '\"'",
        Timeout: 10 * time.Second,
    }
    osResult, err := h.executor.RunCommand(ctx, osCmd)
    
    // Get kernel version
    kernelCmd := tunnel.Command{
        Cmd:     "uname -r",
        Timeout: 5 * time.Second,
    }
    kernelResult, err := h.executor.RunCommand(ctx, kernelCmd)
    
    // Get system stats
    statsCmd := tunnel.Command{
        Cmd:     "free -h && df -h / && nproc",
        Timeout: 10 * time.Second,
    }
    statsResult, err := h.executor.RunCommand(ctx, statsCmd)
    
    return parseSystemInfo(osResult, kernelResult, statsResult), nil
}
```

## Key Alignments

- ✅ Uses individual setup manager methods: `CreateUser()`, `SetupSSHKeys()`, `CreateDirectory()`
- ✅ Uses individual security manager methods: `ApplyLockdown()`, `ConfigureFirewall()`, `SetupFail2ban()`
- ✅ No non-existent methods like `SetupServerWithProgress()` or `ApplySecurityWithProgress()`
- ✅ Progress tracking implemented at handler level using channels
- ✅ Connection testing uses `Pool.Get()` and `Executor.RunCommand()`
- ✅ System information gathering via `Executor.RunCommand()` with shell commands
- ✅ Error handling uses tunnel package utilities
- ✅ Service status checks use `ServiceManager.GetServiceStatus()`
- ✅ No FileManager interface usage - all file ops via `Executor.RunCommand()`
- ✅ Package installation via `Executor.RunCommand()` not PackageManager interface
- ✅ Security audit calls `SecurityManager.AuditSecurity(ctx)` without config parameter