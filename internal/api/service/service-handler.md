# Service Handler Alignment

HTTP handlers for service management aligned with tunnel package interfaces.

## Handler Functions

### ManageService
```http
POST /api/v1/services/{name}/action
```
**Calls:**
- `serviceMgr.ManageService(ctx, action, serviceName)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### GetServiceStatus
```http
GET /api/v1/services/{name}
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- `executor.RunCommand(ctx, systemStatsCmd)` (for memory/CPU usage)

### GetServiceLogs
```http
GET /api/v1/services/{name}/logs
```
**Calls:**
- `executor.RunCommand(ctx, journalctlCmd)`

### EnableService
```http
POST /api/v1/services/{name}/enable
```
**Calls:**
- `serviceMgr.EnableService(ctx, serviceName)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### DisableService
```http
POST /api/v1/services/{name}/disable
```
**Calls:**
- `executor.RunCommand(ctx, systemctlDisableCmd)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### CreateServiceFile
```http
POST /api/v1/services
```
**Calls:**
- `executor.RunCommand(ctx, createServiceFileCmd)`
- `executor.RunCommand(ctx, systemctlDaemonReloadCmd)`
- `serviceMgr.EnableService(ctx, serviceName)` (if enabled in request)

### WaitForService
```http
POST /api/v1/services/{name}/wait
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (polling until desired state)
- Handler-level timeout and retry logic

### ListServices
```http
GET /api/v1/services
```
**Calls:**
- `executor.RunCommand(ctx, systemctlListCmd)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (per service if detailed)

### DeleteServiceFile
```http
DELETE /api/v1/services/{name}
```
**Calls:**
- `serviceMgr.ManageService(ctx, "stop", serviceName)` (if stop_first)
- `executor.RunCommand(ctx, removeServiceFileCmd)`
- `executor.RunCommand(ctx, systemctlDaemonReloadCmd)`

### GetServiceConfiguration
```http
GET /api/v1/services/config
```
**Calls:**
- Database queries only (no tunnel calls for handler config)

### UpdateServiceConfiguration
```http
PUT /api/v1/services/config
```
**Calls:**
- Database update operations only

## Progress Tracking Pattern

Service operations with progress tracking:

```go
func (h *ServiceHandler) manageServiceWithProgress(ctx context.Context, name, action string) error {
    progressChan := make(chan ServiceProgress, 10)
    go h.monitorServiceProgress(ctx, name, progressChan)
    
    // Step 1: Execute service action
    progressChan <- ServiceProgress{Step: action, Status: "running"}
    err := h.serviceMgr.ManageService(ctx, action, name)
    if err != nil {
        progressChan <- ServiceProgress{Step: action, Status: "failed", Error: err}
        return err
    }
    progressChan <- ServiceProgress{Step: action, Status: "completed"}
    
    // Step 2: Verify service state
    progressChan <- ServiceProgress{Step: "verify", Status: "running"}
    status, err := h.serviceMgr.GetServiceStatus(ctx, name)
    if err != nil {
        progressChan <- ServiceProgress{Step: "verify", Status: "failed", Error: err}
        return err
    }
    progressChan <- ServiceProgress{Step: "verify", Status: "completed"}
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewServiceHandler(
    executor tunnel.Executor,
    serviceMgr tunnel.ServiceManager,
) *ServiceHandler {
    return &ServiceHandler{
        executor:   executor,
        serviceMgr: serviceMgr,
    }
}
```

## Error Handling Pattern

```go
result, err := h.serviceMgr.ManageService(ctx, action, name)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryServiceAction(ctx, action, name)
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

## File Operations Pattern

```go
// Create systemd service file
serviceContent := generateServiceFileContent(serviceConfig)
createCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("cat > /etc/systemd/system/%s.service", serviceName),
    Input:   serviceContent,
    Sudo:    true,
    Timeout: 10 * time.Second,
}
err := h.executor.RunCommand(ctx, createCmd)

// Reload systemd daemon
reloadCmd := tunnel.Command{
    Cmd:     "systemctl daemon-reload",
    Sudo:    true,
    Timeout: 30 * time.Second,
}
err = h.executor.RunCommand(ctx, reloadCmd)
```

## Log Retrieval Pattern

```go
// Get service logs via journalctl
logCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("journalctl -u %s -n %d --no-pager", serviceName, lines),
    Timeout: 30 * time.Second,
}
result, err := h.executor.RunCommand(ctx, logCmd)
```

## Wait Pattern

```go
func (h *ServiceHandler) waitForServiceState(ctx context.Context, name, desiredState string, timeout time.Duration) error {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return fmt.Errorf("timeout waiting for service %s to reach state %s", name, desiredState)
        case <-ticker.C:
            status, err := h.serviceMgr.GetServiceStatus(ctx, name)
            if err != nil {
                continue // Keep trying
            }
            if status.State == desiredState {
                return nil
            }
        }
    }
}
```

## Key Alignments

- ✅ Uses `ServiceManager.ManageService(ctx, action, name)` not `ManageServiceWithProgress()`
- ✅ Uses `ServiceManager.GetServiceStatus(ctx, name)` for status checks
- ✅ Uses `ServiceManager.EnableService(ctx, name)` for enabling
- ✅ Service file operations via `Executor.RunCommand()` not FileManager
- ✅ Log retrieval via `Executor.RunCommand()` with journalctl
- ✅ Progress tracking implemented at handler level
- ✅ Wait operations use polling with handler-level timeout logic
- ✅ Error handling uses tunnel package utilities
- ✅ No non-existent service management methods