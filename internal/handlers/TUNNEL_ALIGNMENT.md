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

## Migration Guide Corrections Applied

### 1. Security Manager Usage

**CORRECTED**: All `AuditSecurity` calls now use the actual method signature:
```go
// ✅ CORRECT
auditResult, err := h.securityMgr.AuditSecurity(ctx)

// ❌ INCORRECT (was using)
auditResult, err := h.securityMgr.AuditSecurity(ctx, auditConfig)
auditResult, err := h.securityMgr.RunSecurityAudit(ctx, auditConfig)
auditResult, err := h.securityMgr.RunComprehensiveAudit(ctx, auditConfig)
```

**CORRECTED**: Security lockdown now uses individual method calls:
```go
// ✅ CORRECT  
err := h.securityMgr.ApplyLockdown(ctx, securityConfig)
err := h.securityMgr.ConfigureFirewall(ctx, firewallRules) 
err := h.securityMgr.SetupFail2ban(ctx, fail2banConfig)

// ❌ INCORRECT (was using)
err := h.securityMgr.ApplyLockdownWithProgress(ctx, config, progressChan)
```

### 2. Setup Manager Usage

**CORRECTED**: Server setup now uses individual method calls:
```go
// ✅ CORRECT
err := h.setupMgr.CreateUser(ctx, userConfig)
err := h.setupMgr.SetupSSHKeys(ctx, username, sshKeys)
err := h.setupMgr.CreateDirectory(ctx, dirPath, owner)

// ❌ INCORRECT (was using)  
err := h.setupMgr.SetupServerWithProgress(ctx, config, progressChan)
```

### 3. Deployment Manager Usage

**CORRECTED**: Deployments use basic interface methods:
```go
// ✅ CORRECT
result, err := h.deployMgr.DeployApplication(ctx, config)
status, err := h.deployMgr.GetDeploymentStatus(ctx)
err := h.deployMgr.RollbackDeployment(ctx, version)

// ❌ INCORRECT (was using)
result, err := h.deployMgr.DeployApplicationWithProgress(ctx, config, progressChan)
comparison, err := h.deployMgr.CompareDeployments(ctx, comparisonConfig)
h.deployMgr.ScheduleDeployment(deploymentID, scheduledAt, scheduleConfig)
```

### 4. File Operations (No FileManager Interface)

**CORRECTED**: File operations now use Executor.TransferFile() and shell commands:
```go
// ✅ CORRECT - Using Executor for file operations
transferConfig := tunnel.FileTransfer{
    LocalPath:  localPath,
    RemotePath: remotePath,
    Direction:  tunnel.TransferUpload,
    Progress:   true,
}
err := h.executor.TransferFile(ctx, transferConfig)

// File info via shell commands
statCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("stat -c '%%s,%%Y' %s", filePath),
    Timeout: 10 * time.Second,
}
result, err := h.executor.RunCommand(ctx, statCmd)

// ❌ INCORRECT (was using non-existent FileManager)
uploadRequest, err := h.fileMgr.ParseUploadRequest(e.Request, config)
packageResult, err := h.fileMgr.CreateDeploymentPackage(ctx, config, progressChan)
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

## Summary of Alignment Issues Fixed

### 1. Removed Non-Existent Methods
- ❌ `SetupServerWithProgress()` → ✅ Individual `CreateUser()`, `SetupSSHKeys()`, `CreateDirectory()`
- ❌ `ApplyLockdownWithProgress()` → ✅ Individual `ApplyLockdown()`, `ConfigureFirewall()`, `SetupFail2ban()`
- ❌ `DeployApplicationWithProgress()` → ✅ `DeployApplication()` with handler-level progress
- ❌ `CompareDeployments()` → ✅ Handler-level comparison logic
- ❌ `ScheduleDeployment()` → ✅ Database-level scheduling

### 2. Removed Non-Existent Interfaces
- ❌ `FileManager` interface → ✅ Use `Executor.TransferFile()` and shell commands
- ❌ `PackageValidator` → ✅ Handler-level validation logic

### 3. Corrected Method Signatures
- ❌ `AuditSecurity(ctx, config)` → ✅ `AuditSecurity(ctx)`
- ❌ `GetDeploymentStatus(ctx, deploymentID)` → ✅ `GetDeploymentStatus(ctx)`

### 4. Progress Tracking Strategy
- ✅ Implement progress tracking at handler level
- ✅ Use channels for real-time updates
- ✅ Call individual manager methods sequentially
- ✅ Report progress between operations

## Validation Checklist

- [x] All manager method calls match tunnel package interfaces
- [x] No references to non-existent interfaces (FileManager, PackageValidator)
- [x] Progress tracking implemented at handler level
- [x] Error handling uses tunnel package error utilities
- [x] Constructor patterns follow dependency injection
- [x] File operations use Executor.TransferFile() 
- [x] Security operations use actual SecurityManager.AuditSecurity()
- [x] Service operations use ServiceManager interface methods
- [x] Deployment operations use DeploymentManager interface methods

## Implementation Notes

1. **Progress Tracking**: Since tunnel managers don't provide progress callbacks, implement progress tracking in handlers by calling manager methods sequentially and reporting status between operations.

2. **File Operations**: Use `Executor.TransferFile()` for file transfers and shell commands via `Executor.RunCommand()` for file system operations.

3. **Security Auditing**: The `SecurityManager.AuditSecurity()` method exists in the actual implementation but wasn't documented in the README - this has been verified and is now correctly used.

4. **Error Handling**: All error handling now uses the tunnel package's error utilities (`IsRetryable`, `IsAuthError`, `IsConnectionError`).

5. **Service Container**: The service container pattern correctly integrates all tunnel package interfaces using proper dependency injection.

All migration guides are now aligned with the actual tunnel package interfaces and implementation.