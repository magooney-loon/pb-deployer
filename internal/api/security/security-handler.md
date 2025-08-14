# Security Handler Alignment

HTTP handlers for security management aligned with tunnel package interfaces.

## Handler Functions

### ApplySecurityLockdown
```http
POST /api/v1/security/lockdown
```
**Calls:**
- `securityMgr.ApplyLockdown(ctx, securityConfig)`
- `securityMgr.ConfigureFirewall(ctx, firewallRules)`
- `securityMgr.SetupFail2ban(ctx, fail2banConfig)`
- `executor.RunCommand(ctx, sshRestartCmd)`

### ConfigureFirewall
```http
POST /api/v1/security/firewall
```
**Calls:**
- `securityMgr.ConfigureFirewall(ctx, firewallRules)`
- `executor.RunCommand(ctx, firewallStatusCmd)` (for verification)

### SetupFail2ban
```http
POST /api/v1/security/fail2ban
```
**Calls:**
- `securityMgr.SetupFail2ban(ctx, fail2banConfig)`
- `serviceMgr.GetServiceStatus(ctx, "fail2ban")`

### HardenSSHConfiguration
```http
POST /api/v1/security/ssh/harden
```
**Calls:**
- `executor.RunCommand(ctx, backupSSHConfigCmd)`
- `executor.RunCommand(ctx, updateSSHConfigCmd)`
- `serviceMgr.ManageService(ctx, "restart", "ssh")`

### ConfigureAutoUpdates
```http
POST /api/v1/security/auto-updates
```
**Calls:**
- `executor.RunCommand(ctx, setupAutoUpdatesCmd)`
- `executor.RunCommand(ctx, configureUnattendedUpgradesCmd)`

### SecurityAudit
```http
GET /api/v1/security/audit
```
**Calls:**
- `securityMgr.AuditSecurity(ctx)` (no config parameter)
- `serviceMgr.GetServiceStatus(ctx, "ssh")`
- `serviceMgr.GetServiceStatus(ctx, "fail2ban")`
- `executor.RunCommand(ctx, firewallStatusCmd)`

### GetSecurityStatus
```http
GET /api/v1/security/status
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, "ssh")`
- `serviceMgr.GetServiceStatus(ctx, "fail2ban")`
- `executor.RunCommand(ctx, firewallStatusCmd)`
- `executor.RunCommand(ctx, sshConfigCheckCmd)`

### GetSecurityConfiguration
```http
GET /api/v1/security/config
```
**Calls:**
- `executor.RunCommand(ctx, readConfigCmd)`
- Database queries for security settings

### UpdateSecurityConfiguration
```http
PUT /api/v1/security/config
```
**Calls:**
- `executor.RunCommand(ctx, updateConfigCmd)`
- Database update operations

## Progress Tracking Pattern

All security operations use handler-level progress tracking:

```go
func (h *SecurityHandler) lockdownWithProgress(ctx context.Context, config SecurityConfig) error {
    progressChan := make(chan SecurityProgress, 10)
    go h.monitorSecurityProgress(ctx, progressChan)
    
    // Step 1: Apply basic lockdown
    progressChan <- SecurityProgress{Step: "lockdown", Status: "running"}
    err := h.securityMgr.ApplyLockdown(ctx, config)
    if err != nil {
        progressChan <- SecurityProgress{Step: "lockdown", Status: "failed", Error: err}
        return err
    }
    progressChan <- SecurityProgress{Step: "lockdown", Status: "completed"}
    
    // Step 2: Configure firewall
    progressChan <- SecurityProgress{Step: "firewall", Status: "running"}
    err = h.securityMgr.ConfigureFirewall(ctx, config.FirewallRules)
    if err != nil {
        progressChan <- SecurityProgress{Step: "firewall", Status: "failed", Error: err}
        return err
    }
    progressChan <- SecurityProgress{Step: "firewall", Status: "completed"}
    
    // Step 3: Setup fail2ban
    progressChan <- SecurityProgress{Step: "fail2ban", Status: "running"}
    err = h.securityMgr.SetupFail2ban(ctx, config.Fail2banConfig)
    if err != nil {
        progressChan <- SecurityProgress{Step: "fail2ban", Status: "failed", Error: err}
        return err
    }
    progressChan <- SecurityProgress{Step: "fail2ban", Status: "completed"}
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewSecurityHandler(
    executor tunnel.Executor,
    securityMgr tunnel.SecurityManager,
    serviceMgr tunnel.ServiceManager,
) *SecurityHandler {
    return &SecurityHandler{
        executor:    executor,
        securityMgr: securityMgr,
        serviceMgr:  serviceMgr,
    }
}
```

## Error Handling Pattern

```go
result, err := h.securityMgr.ApplyLockdown(ctx, config)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryLockdown(ctx, config)
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

## Configuration Management Pattern

```go
// Read SSH configuration
readCmd := tunnel.Command{
    Cmd:     "cat /etc/ssh/sshd_config | grep -E '^(PasswordAuthentication|PermitRootLogin)'",
    Timeout: 10 * time.Second,
}
result, err := h.executor.RunCommand(ctx, readCmd)

// Update SSH configuration
updateCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config"),
    Sudo:    true,
    Timeout: 10 * time.Second,
}
err = h.executor.RunCommand(ctx, updateCmd)
```

## Key Alignments

- ✅ Uses individual manager methods: `ApplyLockdown()`, `ConfigureFirewall()`, `SetupFail2ban()`
- ✅ No non-existent methods like `ApplyLockdownWithProgress()`
- ✅ `AuditSecurity(ctx)` called with context only, no config parameter
- ✅ Progress tracking implemented at handler level
- ✅ Error handling uses tunnel package utilities
- ✅ SSH operations use `Executor.RunCommand()` and `ServiceManager.ManageService()`
- ✅ Configuration management via shell commands, not file interfaces
- ✅ Service status checks use `ServiceManager.GetServiceStatus()`
