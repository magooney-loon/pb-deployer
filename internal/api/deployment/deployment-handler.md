# Deployment Handler Alignment

HTTP handlers for deployment management aligned with tunnel package interfaces.

## Handler Functions

### DeployApplication
```http
POST /api/v1/deployments
```
**Calls:**
- `executor.TransferFile(ctx, artifactTransfer)`
- `deployMgr.DeployApplication(ctx, deployConfig)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)` (for verification)

### RollbackDeployment
```http
POST /api/v1/deployments/{name}/rollback
```
**Calls:**
- `deployMgr.RollbackDeployment(ctx, targetVersion)`
- `serviceMgr.ManageService(ctx, "restart", serviceName)`

### GetDeploymentStatus
```http
GET /api/v1/deployments/{name}
```
**Calls:**
- `deployMgr.GetDeploymentStatus(ctx)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- `executor.RunCommand(ctx, deploymentInfoCmd)`

### ListDeployments
```http
GET /api/v1/deployments
```
**Calls:**
- Database queries only (no tunnel calls for basic listing)
- `deployMgr.GetDeploymentStatus(ctx)` (per deployment if detailed)

### ValidateDeployment
```http
POST /api/v1/deployments/validate
```
**Calls:**
- `executor.RunCommand(ctx, validateArtifactCmd)`
- `executor.RunCommand(ctx, checkDependenciesCmd)`
- Handler-level validation logic (no PackageValidator interface)

### DeploymentHealthCheck
```http
GET /api/v1/deployments/{name}/health
```
**Calls:**
- `executor.RunCommand(ctx, healthCheckCmd)`
- `serviceMgr.GetServiceStatus(ctx, serviceName)`
- Handler-level health aggregation logic

## Progress Tracking Pattern

All deployment operations use handler-level progress tracking:

```go
func (h *DeploymentHandler) deployWithProgress(ctx context.Context, name string) error {
    progressChan := make(chan DeploymentProgress, 10)
    go h.monitorDeploymentProgress(ctx, name, progressChan)
    
    // Step 1: Validate artifact
    progressChan <- DeploymentProgress{Step: "validate", Status: "running"}
    err := h.validateArtifact(ctx, deployConfig.ArtifactPath)
    if err != nil {
        progressChan <- DeploymentProgress{Step: "validate", Status: "failed", Error: err}
        return err
    }
    progressChan <- DeploymentProgress{Step: "validate", Status: "completed"}
    
    // Step 2: Transfer artifact
    progressChan <- DeploymentProgress{Step: "transfer", Status: "running"}
    err = h.executor.TransferFile(ctx, transferConfig)
    if err != nil {
        progressChan <- DeploymentProgress{Step: "transfer", Status: "failed", Error: err}
        return err
    }
    progressChan <- DeploymentProgress{Step: "transfer", Status: "completed"}
    
    // Step 3: Deploy via manager
    progressChan <- DeploymentProgress{Step: "deploy", Status: "running"}
    err = h.deployMgr.DeployApplication(ctx, deployConfig)
    if err != nil {
        progressChan <- DeploymentProgress{Step: "deploy", Status: "failed", Error: err}
        return err
    }
    progressChan <- DeploymentProgress{Step: "deploy", Status: "completed"}
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewDeploymentHandler(
    executor tunnel.Executor,
    deployMgr tunnel.DeploymentManager,
    serviceMgr tunnel.ServiceManager,
) *DeploymentHandler {
    return &DeploymentHandler{
        executor:   executor,
        deployMgr:  deployMgr,
        serviceMgr: serviceMgr,
    }
}
```

## Error Handling Pattern

```go
result, err := h.deployMgr.DeployApplication(ctx, config)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryDeployment(ctx, config)
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
// Transfer deployment artifact
transferConfig := tunnel.FileTransfer{
    LocalPath:  localArtifactPath,
    RemotePath: remoteDeployPath,
    Direction:  tunnel.TransferUpload,
    Progress:   true,
}
err := h.executor.TransferFile(ctx, transferConfig)

// Get deployment info via shell commands
infoCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("ls -la %s && stat %s", deployPath, artifactPath),
    Timeout: 10 * time.Second,
}
result, err := h.executor.RunCommand(ctx, infoCmd)
```

## Key Alignments

- ✅ No FileManager interface used - all file ops via `Executor.TransferFile()`
- ✅ Uses `DeploymentManager.DeployApplication()` not `DeployApplicationWithProgress()`
- ✅ Uses `DeploymentManager.GetDeploymentStatus()` not `GetDeploymentStatus(deploymentID)`
- ✅ Uses `DeploymentManager.RollbackDeployment(version)` not `RollbackDeployment(config)`
- ✅ Progress tracking implemented at handler level
- ✅ Validation logic in handlers, no PackageValidator interface
- ✅ Error handling uses tunnel package utilities
- ✅ No scheduling methods - handled at database level
- ✅ No comparison methods - implemented as handler-level logic