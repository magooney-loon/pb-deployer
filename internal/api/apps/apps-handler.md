# Apps Handler Alignment

HTTP handlers for apps management aligned with tunnel package interfaces.

## Handler Functions

### CreateApplication
```http
POST /api/apps
```
**Calls:**
- `setupMgr.CreateDirectory(ctx, appPath, username)`
- `serviceMgr.EnableService(ctx, serviceName)` (if auto_start)
- `executor.RunCommand(ctx, createAppDirCmd)`

### GetApplication
```http
GET /api/apps/{id}
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- `executor.RunCommand(ctx, diskUsageCmd)`
- `deployMgr.GetDeploymentStatus(ctx)`

### ListApplications
```http
GET /api/apps
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (per app)
- Database queries only (no tunnel calls for basic listing)

### UpdateApplication
```http
PUT /api/apps/{id}
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- `executor.RunCommand(ctx, updateConfigCmd)` (if config changes)

### DeleteApplication
```http
DELETE /api/apps/{id}
```
**Calls:**
- `serviceMgr.ManageService(ctx, "stop", serviceName)`
- `executor.RunCommand(ctx, removeFilesCmd)` (if cleanup_files)
- `executor.RunCommand(ctx, removeServiceFileCmd)`

### DeployApplication
```http
POST /api/apps/{id}/deploy
```
**Calls:**
- `executor.TransferFile(ctx, deploymentPackageTransfer)`
- `deployMgr.DeployApplication(ctx, deployConfig)`
- `serviceMgr.ManageService(ctx, "start", serviceName)`

### RollbackApplication
```http
POST /api/apps/{id}/rollback
```
**Calls:**
- `deployMgr.RollbackDeployment(ctx, targetVersion)`
- `serviceMgr.ManageService(ctx, "restart", serviceName)`

### StartApplicationService
```http
POST /api/apps/{id}/start
```
**Calls:**
- `serviceMgr.ManageService(ctx, "start", serviceName)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### StopApplicationService
```http
POST /api/apps/{id}/stop
```
**Calls:**
- `serviceMgr.ManageService(ctx, "stop", serviceName)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### RestartApplicationService
```http
POST /api/apps/{id}/restart
```
**Calls:**
- `serviceMgr.ManageService(ctx, "restart", serviceName)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### GetApplicationStatus
```http
GET /api/apps/{id}/status
```
**Calls:**
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- `deployMgr.GetDeploymentStatus(ctx)`
- `executor.RunCommand(ctx, healthCheckCmd)` (if health_check_url)

### PerformHealthCheck
```http
POST /api/apps/{id}/health-check
```
**Calls:**
- `executor.RunCommand(ctx, curlHealthCmd)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)`

### GetApplicationLogs
```http
GET /api/apps/{id}/logs
```
**Calls:**
- `executor.RunCommand(ctx, journalctlCmd)`

### GetApplicationMetrics
```http
GET /api/apps/{id}/metrics
```
**Calls:**
- `executor.RunCommand(ctx, systemStatsCmd)`
- `executor.RunCommand(ctx, processStatsCmd)`

### GetApplicationConfiguration
```http
GET /api/apps/{id}/config
```
**Calls:**
- `executor.RunCommand(ctx, readConfigCmd)`
- Database queries for app settings

### UpdateApplicationConfiguration
```http
PUT /api/apps/{id}/config
```
**Calls:**
- `executor.RunCommand(ctx, writeConfigCmd)`
- `serviceMgr.ManageService(ctx, "restart", serviceName)` (if restart_service)

## Progress Tracking Pattern

All async operations use handler-level progress tracking:

```go
func (h *AppsHandler) deployWithProgress(ctx context.Context, appID string) error {
    progressChan := make(chan DeployProgress, 10)
    go h.monitorDeployProgress(ctx, appID, progressChan)
    
    // Step 1: Transfer files
    progressChan <- DeployProgress{Step: "transfer", Status: "running"}
    err := h.executor.TransferFile(ctx, transferConfig)
    if err != nil {
        progressChan <- DeployProgress{Step: "transfer", Status: "failed", Error: err}
        return err
    }
    progressChan <- DeployProgress{Step: "transfer", Status: "completed"}
    
    // Step 2: Deploy via manager  
    progressChan <- DeployProgress{Step: "deploy", Status: "running"}
    err = h.deployMgr.DeployApplication(ctx, deployConfig)
    if err != nil {
        progressChan <- DeployProgress{Step: "deploy", Status: "failed", Error: err}
        return err
    }
    progressChan <- DeployProgress{Step: "deploy", Status: "completed"}
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewAppsHandler(
    executor tunnel.Executor,
    deployMgr tunnel.DeploymentManager,
    serviceMgr tunnel.ServiceManager,
    setupMgr tunnel.SetupManager,
) *AppsHandler {
    return &AppsHandler{
        executor:   executor,
        deployMgr:  deployMgr,
        serviceMgr: serviceMgr,
        setupMgr:   setupMgr,
    }
}
```

## Error Handling Pattern

```go
result, err := h.executor.RunCommand(ctx, cmd)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryOperation(ctx, cmd)
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

## Key Alignments

- ✅ No FileManager interface used - all file ops via `Executor.TransferFile()`
- ✅ Individual manager method calls, no non-existent `*WithProgress()` methods
- ✅ Progress tracking implemented at handler level
- ✅ Error handling uses tunnel package utilities
- ✅ Service operations use `ServiceManager.ManageService()`
- ✅ Deployments use `DeploymentManager.DeployApplication()`
- ✅ Security audits use `SecurityManager.AuditSecurity()` (no config param)