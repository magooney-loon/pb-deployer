# Apps Handlers Migration Guide

## Overview

Migrate apps handlers from direct SSH operations to the modern tunnel/tracer/models architecture.

## Current State Analysis

### Current Issues
- Direct SSH connection management in handlers
- No structured tracing or observability
- Basic error handling without retry logic
- Manual database record conversion
- No connection pooling
- Mixed concerns (SSH operations in HTTP handlers)

### Files to Migrate
- `handlers.go` - Handler registration (minimal changes)
- `management.go` - App CRUD operations and health checks
- `service.go` - Service management and deployment operations

## Migration Strategy

### Phase 1: Dependency Injection Setup
Replace direct SSH managers with injected tunnel services.

### Phase 2: Observability Integration
Add structured tracing throughout all operations.

### Phase 3: Manager Integration
Use specialized managers for different operation types.

## File-by-File Migration

### `handlers.go` - Handler Registration

**Current:**
```go
func RegisterAppsHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
    // Direct handler functions
}
```

**Target:**
```go
type AppHandlers struct {
    executor      tunnel.Executor
    serviceMgr    tunnel.ServiceManager
    deployMgr     tunnel.DeploymentManager
    tracer        tracer.ServiceTracer
}

func NewAppHandlers(
    executor tunnel.Executor,
    serviceMgr tunnel.ServiceManager, 
    deployMgr tunnel.DeploymentManager,
    tracer tracer.ServiceTracer,
) *AppHandlers {
    return &AppHandlers{
        executor:   executor,
        serviceMgr: serviceMgr,
        deployMgr:  deployMgr,
        tracer:     tracer,
    }
}

func (h *AppHandlers) RegisterRoutes(group *router.RouterGroup[*core.RequestEvent]) {
    // Handler methods with dependency injection
}
```

### `management.go` - App CRUD Operations

#### Current Issues
- Direct database record manipulation
- No tracing for operations
- Basic error handling
- Manual health check implementation

#### Migration Changes

**App Creation:**
```go
// BEFORE
func createApp(app core.App, e *core.RequestEvent) error {
    // Direct record creation
    record := core.NewRecord(collection)
    record.Set("name", req.Name)
    // ...
}

// AFTER  
func (h *AppHandlers) createApp(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "app", "create")
    defer span.End()
    
    // Use models package for data operations
    appModel := models.NewApp()
    appModel.Name = req.Name
    appModel.ServerID = req.ServerID
    
    // Save through models with validation
    if err := models.SaveApp(app, appModel); err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "Failed to create app")
    }
    
    span.SetField("app.id", appModel.ID)
    span.Event("app_created")
    return e.JSON(http.StatusCreated, appModel)
}
```

**Health Checks:**
```go
// BEFORE
func checkAppHealth(app core.App, e *core.RequestEvent) error {
    // Manual HTTP client and health check
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(healthURL)
    // ...
}

// AFTER
func (h *AppHandlers) checkAppHealth(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "app", "health_check")
    defer span.End()
    
    appModel, err := models.GetApp(app, appID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    // Use service manager for health checks
    result, err := h.serviceMgr.GetServiceStatus(e.Request.Context(), appModel.ServiceName)
    if err != nil {
        span.EndWithError(err)
        return handleServiceError(e, err, "Health check failed")
    }
    
    // Update app status through models
    appModel.Status = result.Status
    if err := models.SaveApp(app, appModel); err != nil {
        h.tracer.RecordError(span, err, "failed to update app status")
    }
    
    span.SetFields(tracer.Fields{
        "app.id":           appModel.ID,
        "service.name":     appModel.ServiceName,
        "service.status":   result.Status,
        "health.endpoint":  appModel.GetHealthURL(),
    })
    
    return e.JSON(http.StatusOK, result)
}
```

### `service.go` - Service Management & Deployment

#### Current Issues
- Manual SSH manager creation
- No connection pooling
- Basic deployment logic
- No structured error handling
- Missing deployment tracking

#### Migration Changes

**Service Actions:**
```go
// BEFORE
func handleServiceAction(app core.App, e *core.RequestEvent, action string) error {
    // Manual SSH manager creation
    var sshManager *ssh.SSHManager
    if server.SecurityLocked {
        sshManager, err = ssh.NewSSHManager(server, false)
    } else {
        sshManager, err = ssh.NewSSHManager(server, true)
    }
    defer sshManager.Close()
    
    switch action {
    case "start":
        actionErr = sshManager.StartService(serviceName)
    // ...
    }
}

// AFTER
func (h *AppHandlers) handleServiceAction(app core.App, e *core.RequestEvent, action string) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "service", action)
    defer span.End()
    
    appModel, err := models.GetApp(app, appID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    serverModel, err := models.GetServer(app, appModel.ServerID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Use service manager with connection pooling
    serviceAction := tunnel.ServiceAction{
        Action:      action,
        ServiceName: appModel.ServiceName,
        Timeout:     30 * time.Second,
    }
    
    result, err := h.serviceMgr.ManageService(e.Request.Context(), serviceAction, appModel.ServiceName)
    if err != nil {
        if tunnel.IsRetryable(err) {
            span.Event("retrying_service_action")
            // Implement retry logic
        }
        span.EndWithError(err)
        return handleServiceError(e, err, fmt.Sprintf("Failed to %s service", action))
    }
    
    // Update app status through models
    if action == "start" && result.Success {
        appModel.Status = "online"
    } else if action == "stop" && result.Success {
        appModel.Status = "offline"
    }
    
    if err := models.SaveApp(app, appModel); err != nil {
        h.tracer.RecordError(span, err, "failed to update app status")
    }
    
    span.SetFields(tracer.Fields{
        "app.id":         appModel.ID,
        "service.name":   appModel.ServiceName,
        "service.action": action,
        "service.result": result.Success,
    })
    
    return e.JSON(http.StatusOK, result)
}
```

**Deployment Operations:**
```go
// BEFORE
func deployApp(app core.App, e *core.RequestEvent) error {
    // Manual deployment record creation
    deploymentRecord := core.NewRecord(deploymentCollection)
    deploymentRecord.Set("app_id", appID)
    // ...
    
    // Background goroutine with manual logic
    go func() {
        deploymentErr := performDeployment(...)
        // Manual record updates
    }()
}

// AFTER
func (h *AppHandlers) deployApp(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceDeployment(e.Request.Context(), appModel.Name, req.VersionID)
    defer span.End()
    
    appModel, err := models.GetApp(app, appID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    versionModel, err := models.GetVersion(app, req.VersionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    // Create deployment through models
    deployment := models.NewDeployment()
    deployment.AppID = appID
    deployment.VersionID = req.VersionID
    deployment.MarkAsRunning()
    
    if err := models.SaveDeployment(app, deployment); err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to create deployment")
    }
    
    // Use deployment manager
    deployConfig := tunnel.DeployConfig{
        AppName:       appModel.Name,
        Version:       versionModel.VersionNumber,
        SourcePath:    versionModel.GetLocalPath(),
        TargetPath:    appModel.RemotePath,
        ServiceName:   appModel.ServiceName,
        IsFirstDeploy: appModel.CurrentVersion == "",
        Environment:   req.Environment,
    }
    
    // Start deployment in background
    go h.performDeployment(app, deployment, deployConfig, span.Context())
    
    span.SetFields(tracer.Fields{
        "deployment.id":     deployment.ID,
        "app.id":           appModel.ID,
        "version.id":       versionModel.ID,
        "deployment.first": deployConfig.IsFirstDeploy,
    })
    
    return e.JSON(http.StatusAccepted, deployment)
}

func (h *AppHandlers) performDeployment(app core.App, deployment *models.Deployment, config tunnel.DeployConfig, ctx context.Context) {
    span := h.tracer.TraceDeployment(ctx, config.AppName, config.Version)
    defer span.End()
    
    // Use deployment manager with progress tracking
    result, err := h.deployMgr.DeployApplication(ctx, config)
    if err != nil {
        deployment.MarkAsFailed()
        deployment.AppendLog(fmt.Sprintf("Deployment failed: %v", err))
        span.EndWithError(err)
    } else {
        deployment.MarkAsSuccess()
        deployment.AppendLog("Deployment completed successfully")
        span.Event("deployment_completed")
    }
    
    models.SaveDeployment(app, deployment)
}
```

## New Dependencies and Injection

### Constructor Function
```go
func NewAppHandlers(app core.App) (*AppHandlers, error) {
    // Setup tracing
    tracerFactory := tracer.SetupProductionTracing(os.Stdout)
    serviceTracer := tracerFactory.CreateServiceTracer()
    sshTracer := tracerFactory.CreateSSHTracer()
    
    // Setup tunnel components
    factory := tunnel.NewConnectionFactory(sshTracer)
    poolConfig := tunnel.PoolConfig{
        MaxConnections:  50,
        IdleTimeout:     30 * time.Minute,
        HealthCheckInt:  5 * time.Minute,
    }
    pool := tunnel.NewPool(factory, poolConfig, sshTracer)
    executor := tunnel.NewExecutor(pool, sshTracer)
    
    // Setup managers
    serviceMgr := tunnel.NewServiceManager(executor, serviceTracer)
    deployMgr := tunnel.NewDeploymentManager(executor, serviceTracer)
    
    return &AppHandlers{
        executor:   executor,
        serviceMgr: serviceMgr,
        deployMgr:  deployMgr,
        tracer:     serviceTracer,
    }, nil
}
```

### Error Handling Improvements

**Before:**
```go
if err != nil {
    app.Logger().Error("Failed to create app", "error", err)
    return e.JSON(http.StatusInternalServerError, map[string]string{
        "error": "Failed to create app",
    })
}
```

**After:**
```go
func handleAppError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsConnectionError(err) {
        return e.JSON(http.StatusBadGateway, map[string]string{
            "error":      message,
            "details":    "Server connection failed",
            "suggestion": "Check server status and try again",
        })
    }
    
    if tunnel.IsAuthError(err) {
        return e.JSON(http.StatusUnauthorized, map[string]string{
            "error":      message,
            "details":    "Authentication failed",
            "suggestion": "Check SSH credentials",
        })
    }
    
    return e.JSON(http.StatusInternalServerError, map[string]string{
        "error":   message,
        "details": err.Error(),
    })
}
```

## Step-by-Step Migration

### Step 1: Update Dependencies
```go
// Add imports
import (
    "pb-deployer/internal/tunnel"
    "pb-deployer/internal/tracer"
    "pb-deployer/internal/models"
)

// Remove old imports
// import "pb-deployer/internal/ssh" // Remove direct SSH usage
```

### Step 2: Refactor Handler Structure
```go
// Convert functions to methods on AppHandlers struct
type AppHandlers struct {
    executor      tunnel.Executor
    serviceMgr    tunnel.ServiceManager
    deployMgr     tunnel.DeploymentManager
    tracer        tracer.ServiceTracer
}
```

### Step 3: Update Database Operations
```go
// BEFORE: Manual record manipulation
record := core.NewRecord(collection)
record.Set("name", req.Name)
app.Save(record)

// AFTER: Use models package
appModel := models.NewApp()
appModel.Name = req.Name
appModel.ServerID = req.ServerID
models.SaveApp(app, appModel)
```

### Step 4: Replace SSH Operations
```go
// BEFORE: Direct SSH manager
sshManager, err := ssh.NewSSHManager(server, useRoot)
sshManager.StartService(serviceName)

// AFTER: Use service manager
serviceAction := tunnel.ServiceStart
result, err := h.serviceMgr.ManageService(ctx, serviceAction, serviceName)
```

### Step 5: Add Comprehensive Tracing
```go
// Add to every operation
span := h.tracer.TraceServiceAction(ctx, "app", operation)
defer span.End()

span.SetFields(tracer.Fields{
    "app.id":   appID,
    "app.name": appName,
})

// Record events and errors
span.Event("app_validation_complete")
if err != nil {
    span.EndWithError(err)
}
```

## Breaking Changes

### Function Signatures
- All handlers become methods on `AppHandlers` struct
- Context propagation throughout call chain
- Structured error responses

### Error Handling
- Replace simple error messages with structured error types
- Add retry logic for retryable errors
- Implement circuit breaker patterns

### Response Formats
- Standardize response structures
- Add operation metadata (timing, tracing IDs)
- Include actionable error suggestions

## New Features to Implement

### Enhanced Health Monitoring
```go
func (h *AppHandlers) enhancedHealthCheck(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "app", "health_check")
    defer span.End()
    
    appModel, _ := models.GetApp(app, appID)
    
    // Multiple health check types
    healthChecks := []HealthCheck{
        {Type: "http", URL: appModel.GetHealthURL()},
        {Type: "service", Name: appModel.ServiceName},
        {Type: "database", Path: appModel.GetDBPath()},
    }
    
    results := make([]HealthResult, len(healthChecks))
    for i, check := range healthChecks {
        results[i] = h.performHealthCheck(e.Request.Context(), check)
    }
    
    // Aggregate results
    overallHealth := aggregateHealthResults(results)
    
    // Update app status
    appModel.Status = overallHealth.Status
    models.SaveApp(app, appModel)
    
    return e.JSON(http.StatusOK, overallHealth)
}
```

### Deployment Progress Tracking
```go
func (h *AppHandlers) deployWithProgress(ctx context.Context, deployment *models.Deployment, config tunnel.DeployConfig) {
    span := h.tracer.TraceDeployment(ctx, config.AppName, config.Version)
    defer span.End()
    
    // Create progress channel
    progressChan := make(chan tunnel.DeploymentProgress, 10)
    
    // Start progress monitoring
    go h.monitorDeploymentProgress(ctx, deployment.ID, progressChan)
    
    // Use deployment manager 
    result, err := h.deployMgr.DeployApplication(ctx, config)
    
    if err != nil {
        deployment.MarkAsFailed()
        span.EndWithError(err)
    } else {
        deployment.MarkAsSuccess()
        span.Event("deployment_completed")
    }
    
    tracer.RecordDeploymentResult(span, result)
}
```

## Testing Migration

### Mock Dependencies
```go
func TestAppHandlers(t *testing.T) {
    // Setup test dependencies
    tracerFactory := tracer.SetupTestTracing(t)
    mockExecutor := &tunnel.MockExecutor{}
    mockServiceMgr := &tunnel.MockServiceManager{}
    mockDeployMgr := &tunnel.MockDeploymentManager{}
    
    handlers := &AppHandlers{
        executor:   mockExecutor,
        serviceMgr: mockServiceMgr,
        deployMgr:  mockDeployMgr,
        tracer:     tracerFactory.CreateServiceTracer(),
    }
    
    // Test with mocks
    mockServiceMgr.On("ManageService", mock.Anything, tunnel.ServiceStart, "test-service").
        Return(&tunnel.ServiceResult{Success: true}, nil)
    
    // Execute test
    err := handlers.startAppService(testApp, testEvent)
    assert.NoError(t, err)
}
```

## Validation Checklist

### ✅ Pre-Migration
- [ ] Identify all SSH operations in current handlers
- [ ] Map service operations to tunnel managers
- [ ] Plan tracing integration points
- [ ] Design error handling strategy

### ✅ During Migration
- [ ] Replace direct SSH with tunnel.Executor
- [ ] Add tracing to all operations
- [ ] Use models package for data operations
- [ ] Implement structured error handling
- [ ] Add retry logic for network operations

### ✅ Post-Migration
- [ ] All handlers use dependency injection
- [ ] No direct SSH connections in handlers
- [ ] Comprehensive tracing coverage
- [ ] Structured error responses
- [ ] Connection pooling active
- [ ] Health monitoring enhanced
- [ ] Deployment progress tracking
- [ ] Test coverage maintained

## Integration Points

### With Server Handlers
```go
// Apps need server information for operations
serverModel, err := models.GetServer(app, appModel.ServerID)
if !serverModel.IsReadyForDeployment() {
    return handleReadinessError(e, "Server not ready for deployment")
}
```

### With Version Handlers
```go
// Apps need version information for deployments
versionModel, err := models.GetVersion(app, req.VersionID)
if !versionModel.HasDeploymentZip() {
    return handleVersionError(e, "Version has no deployment package")
}
```

### With Deployment Handlers
```go
// Apps create deployment records for tracking
deployment := models.NewDeployment()
deployment.AppID = appModel.ID
deployment.VersionID = versionModel.ID
// Deployment handlers track progress
```

## Performance Improvements

### Connection Pooling
- Replace individual SSH connections with pooled connections
- 50-80% performance improvement for repeated operations
- Automatic health monitoring and recovery

### Async Operations
- Background deployment processing
- Real-time progress updates via WebSocket
- Non-blocking service management

### Caching
- Cache service status for short periods
- Cache health check results
- Pool connection metadata

## Rollback Plan

If migration issues arise:
1. Keep old handlers alongside new ones
2. Use feature flags to switch between implementations
3. Gradual migration app by app
4. Maintain both SSH and tunnel services temporarily

## Timeline

- **Day 1**: Update handler structure and dependencies
- **Day 2**: Migrate CRUD operations to use models package
- **Day 3**: Replace SSH operations with tunnel managers
- **Day 4**: Add comprehensive tracing
- **Day 5**: Testing and validation
- **Day 6**: Error handling and edge cases
- **Day 7**: Documentation and cleanup