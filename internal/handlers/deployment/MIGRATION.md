# Deployment Handlers Migration Guide

## Overview

Migrate deployment handlers from basic database operations to the modern tunnel/tracer/models architecture with enhanced deployment management, progress tracking, and observability.

## Current State Analysis

### Current Issues
- Direct database record manipulation
- No deployment orchestration or rollback capabilities
- Basic progress tracking without real-time updates
- Limited deployment validation
- No retry logic for failed deployments
- Manual deployment process simulation
- Basic error handling without categorization
- No deployment health monitoring
- Missing deployment analytics and insights

### Files to Migrate
- `handlers.go` - Handler registration (needs dependency injection)
- `management.go` - Deployment CRUD, status, logs, and cleanup operations

## Migration Strategy

### Phase 1: Deployment Manager Integration
Replace manual deployment logic with tunnel.DeploymentManager.

### Phase 2: Enhanced Progress Tracking
Implement real-time deployment progress with WebSocket updates.

### Phase 3: Advanced Deployment Features
Add rollback capabilities, health monitoring, and analytics.

## File-by-File Migration

### `handlers.go` - Handler Registration

**Current:**
```go
func RegisterDeploymentHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
    // Direct handler functions
}
```

**Target:**
```go
type DeploymentHandlers struct {
    executor     tunnel.Executor
    deployMgr    tunnel.DeploymentManager
    serviceMgr   tunnel.ServiceManager
    tracer       tracer.ServiceTracer
    deployTracer tracer.DeploymentTracer
    pool         tunnel.Pool
}

func NewDeploymentHandlers(
    executor tunnel.Executor,
    deployMgr tunnel.DeploymentManager,
    serviceMgr tunnel.ServiceManager,
    pool tunnel.Pool,
    tracerFactory tracer.TracerFactory,
) *DeploymentHandlers {
    return &DeploymentHandlers{
        executor:     executor,
        deployMgr:    deployMgr,
        serviceMgr:   serviceMgr,
        tracer:       tracerFactory.CreateServiceTracer(),
        deployTracer: tracerFactory.CreateDeploymentTracer(),
        pool:         pool,
    }
}

func (h *DeploymentHandlers) RegisterRoutes(group *router.RouterGroup[*core.RequestEvent]) {
    // Handler methods with dependency injection
}
```

### `management.go` - Deployment Operations

#### Current Issues
- Manual deployment record creation and updates
- Simulated deployment process (TODO comments)
- Basic status tracking without real-time updates
- No deployment validation or health checks
- Limited cleanup capabilities
- No deployment analytics or insights

#### Migration Changes

**Deployment Listing:**
```go
// BEFORE
func listDeployments(app core.App, e *core.RequestEvent) error {
    records, err := app.FindRecordsByFilter("deployments", filter, "-created", limit, 0, params)
    // Manual record conversion
    deployments := make([]DeploymentResponse, len(records))
    for i, record := range records {
        deployments[i] = recordToDeploymentResponse(record, app)
    }
}

// AFTER
func (h *DeploymentHandlers) listDeployments(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "deployment", "list")
    defer span.End()
    
    // Parse filters and pagination
    filters := parseDeploymentFilters(e.Request.URL.Query())
    
    // Use models package for data operations
    deployments, err := models.GetDeployments(app, filters)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to fetch deployments")
    }
    
    // Enhance with real-time status updates
    for i := range deployments {
        h.enrichDeploymentWithRealTimeStatus(&deployments[i])
    }
    
    span.SetFields(tracer.Fields{
        "deployments.count":     len(deployments),
        "deployments.app_id":    filters.AppID,
        "deployments.status":    filters.Status,
    })
    
    return e.JSON(http.StatusOK, DeploymentListResponse{
        Deployments: deployments,
        Count:       len(deployments),
        Filters:     filters,
        Metadata:    h.getDeploymentMetadata(deployments),
    })
}
```

**Deployment Status with Real-time Updates:**
```go
// BEFORE
func getDeploymentStatus(app core.App, e *core.RequestEvent) error {
    record, err := app.FindRecordById("deployments", deploymentID)
    // Basic status return
    response := DeploymentStatusResponse{
        ID:     deploymentID,
        Status: record.GetString("status"),
    }
}

// AFTER
func (h *DeploymentHandlers) getDeploymentStatus(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "status_check")
    defer span.End()
    
    deploymentModel, err := models.GetDeployment(app, deploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Deployment not found")
    }
    
    // Get real-time deployment status from manager
    liveStatus, err := h.deployMgr.GetDeploymentStatus(e.Request.Context(), deploymentModel.ID)
    if err != nil && !errors.Is(err, tunnel.ErrDeploymentNotActive) {
        h.tracer.RecordError(span, err, "failed to get live status")
    }
    
    // Combine stored status with live status
    status := h.mergeDeploymentStatus(deploymentModel, liveStatus)
    
    // Add deployment health check
    healthStatus := h.checkDeploymentHealth(e.Request.Context(), deploymentModel)
    
    response := EnhancedDeploymentStatusResponse{
        ID:               deploymentModel.ID,
        Status:           status.Status,
        Progress:         status.Progress,
        CurrentStep:      status.CurrentStep,
        EstimatedRemaining: status.EstimatedRemaining,
        Health:           healthStatus,
        LiveMetrics:      liveStatus,
        CanCancel:        status.CanCancel,
        CanRetry:         status.CanRetry,
        CanRollback:      status.CanRollback,
        Timestamp:        time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "deployment.id":       deploymentModel.ID,
        "deployment.status":   status.Status,
        "deployment.progress": status.Progress,
        "deployment.health":   healthStatus.Overall,
    })
    
    return e.JSON(http.StatusOK, response)
}
```

**Advanced Deployment Operations:**
```go
// NEW: Start deployment with comprehensive management
func (h *DeploymentHandlers) startDeployment(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "start")
    defer span.End()
    
    var req DeploymentStartRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return handleValidationError(e, err, "Invalid request body")
    }
    
    // Validate deployment request
    if err := h.validateDeploymentRequest(req); err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Deployment validation failed")
    }
    
    // Get models
    appModel, err := models.GetApp(app, req.AppID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    versionModel, err := models.GetVersion(app, req.VersionID)
    if err != nil {
        span.EndWithError(err)
        return handleVersionError(e, err, "Version not found")
    }
    
    serverModel, err := models.GetServer(app, appModel.ServerID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Validate deployment prerequisites
    prereqs, err := h.validateDeploymentPrerequisites(e.Request.Context(), appModel, versionModel, serverModel)
    if err != nil {
        span.EndWithError(err)
        return handlePrerequisiteError(e, err, "Prerequisite check failed")
    }
    
    if !prereqs.Valid {
        return e.JSON(http.StatusBadRequest, ValidationErrorResponse{
            Error:   "Deployment prerequisites not met",
            Issues:  prereqs.Issues,
            Suggestions: prereqs.Suggestions,
        })
    }
    
    // Create deployment record
    deployment := models.NewDeployment()
    deployment.AppID = appModel.ID
    deployment.VersionID = versionModel.ID
    deployment.Strategy = req.Strategy
    deployment.Environment = req.Environment
    deployment.MarkAsRunning()
    
    if err := models.SaveDeployment(app, deployment); err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to create deployment record")
    }
    
    // Configure deployment
    deployConfig := tunnel.DeployConfig{
        DeploymentID:     deployment.ID,
        AppName:          appModel.Name,
        Version:          versionModel.VersionNumber,
        SourcePath:       versionModel.GetDeploymentZipPath(),
        TargetPath:       appModel.RemotePath,
        ServiceName:      appModel.ServiceName,
        Strategy:         req.Strategy,
        HealthCheckURL:   appModel.GetHealthURL(),
        Environment:      req.Environment,
        PreDeployHooks:   req.PreDeployHooks,
        PostDeployHooks:  req.PostDeployHooks,
        RollbackOnFailure: req.RollbackOnFailure,
    }
    
    // Start deployment with progress tracking
    go h.executeDeployment(e.Request.Context(), deployment, deployConfig, span)
    
    span.SetFields(tracer.Fields{
        "deployment.id":       deployment.ID,
        "app.id":             appModel.ID,
        "version.id":         versionModel.ID,
        "server.id":          serverModel.ID,
        "deployment.strategy": req.Strategy,
        "deployment.first":    appModel.CurrentVersion == "",
    })
    
    return e.JSON(http.StatusAccepted, DeploymentStartResponse{
        DeploymentID: deployment.ID,
        AppID:        appModel.ID,
        VersionID:    versionModel.ID,
        Strategy:     req.Strategy,
        Status:       "running",
        StartedAt:    deployment.StartedAt,
    })
}
```

**Enhanced Deployment Execution:**
```go
func (h *DeploymentHandlers) executeDeployment(ctx context.Context, deployment *models.Deployment, config tunnel.DeployConfig, parentSpan tracer.Span) {
    span := parentSpan.StartChild("deployment_execution")
    defer span.End()
    
    // Create progress channel for real-time updates
    progressChan := make(chan tunnel.DeploymentProgress, 20)
    
    // Start progress monitoring
    go h.monitorDeploymentProgress(ctx, deployment.ID, progressChan)
    
    // Execute deployment with comprehensive management
    result, err := h.deployMgr.DeployApplicationWithProgress(ctx, config, progressChan)
    close(progressChan)
    
    // Update deployment record based on result
    if err != nil {
        deployment.MarkAsFailed()
        deployment.AppendLog(fmt.Sprintf("Deployment failed: %v", err))
        
        // Attempt automatic rollback if configured
        if config.RollbackOnFailure && deployment.CanRollback() {
            h.performAutomaticRollback(ctx, deployment, config, span)
        }
        
        span.EndWithError(err)
    } else {
        deployment.MarkAsSuccess()
        deployment.AppendLog("Deployment completed successfully")
        
        // Update app current version
        appModel, _ := models.GetApp(app, deployment.AppID)
        appModel.CurrentVersion = config.Version
        appModel.Status = "online"
        models.SaveApp(app, appModel)
        
        // Run post-deployment verification
        h.verifyDeployment(ctx, deployment, config, span)
        
        span.Event("deployment_completed")
    }
    
    // Save final deployment state
    models.SaveDeployment(app, deployment)
    
    // Record deployment metrics
    tracer.RecordDeploymentResult(span, result)
    
    // Send final notification
    h.notifyDeploymentComplete(deployment.ID, deployment.Status == "success")
}
```

**Rollback Operations:**
```go
// BEFORE
func retryDeployment(app core.App, e *core.RequestEvent) error {
    // Create new deployment record manually
    newRecord := core.NewRecord(collection)
    newRecord.Set("app_id", appID)
    // Basic retry logic
}

// AFTER
func (h *DeploymentHandlers) rollbackDeployment(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "rollback")
    defer span.End()
    
    var req RollbackRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return handleValidationError(e, err, "Invalid rollback request")
    }
    
    deploymentModel, err := models.GetDeployment(app, deploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Deployment not found")
    }
    
    appModel, err := models.GetApp(app, deploymentModel.AppID)
    if err != nil {
        span.EndWithError(err)
        return handleAppError(e, err, "App not found")
    }
    
    // Validate rollback target
    targetVersion, err := h.validateRollbackTarget(app, req.TargetVersion, appModel)
    if err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Invalid rollback target")
    }
    
    // Check rollback eligibility
    if !h.canRollback(deploymentModel, appModel) {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error":      "Rollback not available",
            "suggestion": "Ensure deployment is in a rollback-eligible state",
        })
    }
    
    // Create rollback deployment
    rollbackDeployment := models.NewDeployment()
    rollbackDeployment.AppID = appModel.ID
    rollbackDeployment.VersionID = targetVersion.ID
    rollbackDeployment.Type = "rollback"
    rollbackDeployment.OriginalDeploymentID = deploymentModel.ID
    rollbackDeployment.Reason = req.Reason
    rollbackDeployment.MarkAsRunning()
    
    if err := models.SaveDeployment(app, rollbackDeployment); err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to create rollback deployment")
    }
    
    // Configure rollback
    rollbackConfig := tunnel.DeployConfig{
        DeploymentID:  rollbackDeployment.ID,
        AppName:       appModel.Name,
        Version:       targetVersion.VersionNumber,
        SourcePath:    targetVersion.GetDeploymentZipPath(),
        TargetPath:    appModel.RemotePath,
        ServiceName:   appModel.ServiceName,
        Strategy:      "rollback",
        IsRollback:    true,
        PreviousVersion: appModel.CurrentVersion,
        Force:         req.Force,
    }
    
    // Execute rollback
    go h.executeRollback(e.Request.Context(), rollbackDeployment, rollbackConfig, span)
    
    span.SetFields(tracer.Fields{
        "rollback.deployment_id":  rollbackDeployment.ID,
        "rollback.app_id":        appModel.ID,
        "rollback.target_version": targetVersion.VersionNumber,
        "rollback.from_version":   appModel.CurrentVersion,
        "rollback.reason":        req.Reason,
    })
    
    return e.JSON(http.StatusAccepted, RollbackResponse{
        DeploymentID:   rollbackDeployment.ID,
        AppID:         appModel.ID,
        TargetVersion: targetVersion.VersionNumber,
        FromVersion:   appModel.CurrentVersion,
        Status:        "running",
        StartedAt:     rollbackDeployment.StartedAt,
    })
}
```

**Enhanced Deployment Status:**
```go
// BEFORE
func getDeploymentStatus(app core.App, e *core.RequestEvent) error {
    record, err := app.FindRecordById("deployments", deploymentID)
    // Basic status fields
    response := DeploymentStatusResponse{
        ID:     deploymentID,
        Status: record.GetString("status"),
    }
}

// AFTER
func (h *DeploymentHandlers) getDeploymentStatus(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "status_check")
    defer span.End()
    
    deploymentModel, err := models.GetDeployment(app, deploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Deployment not found")
    }
    
    // Get comprehensive deployment status
    status := h.getComprehensiveDeploymentStatus(e.Request.Context(), deploymentModel)
    
    // Add real-time metrics if deployment is active
    if deploymentModel.IsRunning() {
        liveMetrics, err := h.deployMgr.GetLiveDeploymentMetrics(e.Request.Context(), deploymentModel.ID)
        if err == nil {
            status.LiveMetrics = liveMetrics
            status.EstimatedRemaining = h.calculateRemainingTime(liveMetrics, deploymentModel)
        }
    }
    
    // Add deployment health assessment
    healthStatus := h.assessDeploymentHealth(e.Request.Context(), deploymentModel)
    status.Health = healthStatus
    
    // Add rollback capability assessment
    rollbackStatus := h.assessRollbackCapability(deploymentModel)
    status.RollbackStatus = rollbackStatus
    
    span.SetFields(tracer.Fields{
        "deployment.id":         deploymentModel.ID,
        "deployment.status":     status.Status,
        "deployment.progress":   status.Progress,
        "deployment.health":     healthStatus.Overall,
        "deployment.can_rollback": rollbackStatus.CanRollback,
    })
    
    return e.JSON(http.StatusOK, status)
}
```

**Deployment Analytics:**
```go
// BEFORE
func getDeploymentStats(app core.App, e *core.RequestEvent) error {
    // Basic statistics calculation
    stats := DeploymentStatsResponse{
        Total:   int64(len(records)),
        Success: stats.Success,
    }
}

// AFTER
func (h *DeploymentHandlers) getDeploymentAnalytics(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "deployment", "analytics")
    defer span.End()
    
    // Parse analytics parameters
    params := parseAnalyticsParams(e.Request.URL.Query())
    
    // Get comprehensive deployment analytics
    analytics, err := models.GetDeploymentAnalytics(app, params)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to get deployment analytics")
    }
    
    // Enhance with real-time insights
    insights := h.generateDeploymentInsights(analytics)
    
    // Add performance trends
    trends := h.calculateDeploymentTrends(analytics)
    
    // Add health trends
    healthTrends := h.calculateHealthTrends(analytics)
    
    response := DeploymentAnalyticsResponse{
        Analytics:    analytics,
        Insights:     insights,
        Trends:       trends,
        HealthTrends: healthTrends,
        Recommendations: h.generateDeploymentRecommendations(analytics),
        TimeRange:   params.TimeRange,
        GeneratedAt: time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "analytics.total_deployments": analytics.TotalDeployments,
        "analytics.success_rate":      analytics.SuccessRate,
        "analytics.avg_duration":      analytics.AverageDuration,
        "analytics.trend":            trends.Overall,
    })
    
    return e.JSON(http.StatusOK, response)
}
```

**Advanced Cleanup Operations:**
```go
// BEFORE
func cleanupOldDeployments(app core.App, e *core.RequestEvent) error {
    // Basic record deletion
    for _, record := range oldRecords {
        app.Delete(record)
    }
}

// AFTER
func (h *DeploymentHandlers) cleanupDeployments(app core.App, e *core.RequestEvent) error {
    span := h.tracer.TraceServiceAction(e.Request.Context(), "deployment", "cleanup")
    defer span.End()
    
    // Parse cleanup parameters
    cleanupConfig := parseCleanupConfig(e.Request.URL.Query())
    
    // Get deployments eligible for cleanup
    candidates, err := models.GetCleanupCandidates(app, cleanupConfig)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to identify cleanup candidates")
    }
    
    // Validate cleanup safety
    safetyCheck := h.validateCleanupSafety(candidates)
    if !safetyCheck.Safe {
        return e.JSON(http.StatusBadRequest, CleanupSafetyErrorResponse{
            Error:   "Cleanup operation not safe",
            Issues:  safetyCheck.Issues,
            Suggestions: safetyCheck.Suggestions,
        })
    }
    
    // Execute cleanup with progress tracking
    progressChan := make(chan tunnel.CleanupProgress, 10)
    go h.monitorCleanupProgress(e.Request.Context(), progressChan)
    
    result, err := h.performCleanupOperation(e.Request.Context(), candidates, cleanupConfig, progressChan)
    close(progressChan)
    
    if err != nil {
        span.EndWithError(err)
        return handleCleanupError(e, err, "Cleanup operation failed")
    }
    
    span.SetFields(tracer.Fields{
        "cleanup.candidates":  len(candidates),
        "cleanup.removed":     result.RemovedCount,
        "cleanup.kept":        result.KeptCount,
        "cleanup.freed_space": result.FreedSpace,
        "cleanup.dry_run":     cleanupConfig.DryRun,
    })
    
    return e.JSON(http.StatusOK, result)
}
```

## New Features to Implement

### Deployment Health Monitoring
```go
func (h *DeploymentHandlers) getDeploymentHealth(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "health_check")
    defer span.End()
    
    deploymentModel, err := models.GetDeployment(app, deploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Deployment not found")
    }
    
    // Comprehensive health check
    healthChecks := []HealthCheck{
        {Type: "service", Name: "Service Status"},
        {Type: "http", Name: "HTTP Health Endpoint"},
        {Type: "database", Name: "Database Connectivity"},
        {Type: "filesystem", Name: "File System Health"},
        {Type: "resources", Name: "Resource Utilization"},
    }
    
    healthResults := make([]HealthCheckResult, len(healthChecks))
    for i, check := range healthChecks {
        healthResults[i] = h.performHealthCheck(e.Request.Context(), deploymentModel, check)
    }
    
    // Calculate overall health
    overallHealth := h.calculateOverallHealth(healthResults)
    
    response := DeploymentHealthResponse{
        DeploymentID:   deploymentModel.ID,
        OverallHealth:  overallHealth,
        HealthChecks:   healthResults,
        LastChecked:    time.Now().UTC(),
        NextCheckDue:   time.Now().Add(5 * time.Minute),
        HealthHistory:  h.getHealthHistory(deploymentModel.ID),
    }
    
    span.SetFields(tracer.Fields{
        "deployment.id":     deploymentModel.ID,
        "health.overall":    overallHealth.Status,
        "health.score":      overallHealth.Score,
        "health.checks":     len(healthResults),
    })
    
    return e.JSON(http.StatusOK, response)
}
```

### Deployment Comparison and Diff
```go
func (h *DeploymentHandlers) compareDeployments(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "comparison")
    defer span.End()
    
    fromDeploymentID := e.Request.URL.Query().Get("from")
    toDeploymentID := e.Request.URL.Query().Get("to")
    
    if fromDeploymentID == "" || toDeploymentID == "" {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Both 'from' and 'to' deployment IDs are required",
        })
    }
    
    fromDeployment, err := models.GetDeployment(app, fromDeploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "From deployment not found")
    }
    
    toDeployment, err := models.GetDeployment(app, toDeploymentID)
    if err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "To deployment not found")
    }
    
    // Generate deployment comparison
    comparison, err := h.deployMgr.CompareDeployments(e.Request.Context(), tunnel.DeploymentComparison{
        FromDeployment: fromDeployment.ID,
        ToDeployment:   toDeployment.ID,
        IncludeFiles:   e.Request.URL.Query().Get("include_files") == "true",
        IncludeConfig:  e.Request.URL.Query().Get("include_config") == "true",
        IncludeMetrics: e.Request.URL.Query().Get("include_metrics") == "true",
    })
    
    if err != nil {
        span.EndWithError(err)
        return handleComparisonError(e, err, "Deployment comparison failed")
    }
    
    span.SetFields(tracer.Fields{
        "comparison.from":     fromDeploymentID,
        "comparison.to":       toDeploymentID,
        "comparison.changes":  len(comparison.Changes),
        "comparison.type":     comparison.Type,
    })
    
    return e.JSON(http.StatusOK, comparison)
}
```

### Deployment Scheduling
```go
func (h *DeploymentHandlers) scheduleDeployment(app core.App, e *core.RequestEvent) error {
    span := h.deployTracer.TraceDeployment(e.Request.Context(), "", "schedule")
    defer span.End()
    
    var req ScheduleDeploymentRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return handleValidationError(e, err, "Invalid schedule request")
    }
    
    // Validate schedule time
    if req.ScheduledAt.Before(time.Now().Add(5 * time.Minute)) {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Scheduled time must be at least 5 minutes in the future",
        })
    }
    
    // Create scheduled deployment
    deployment := models.NewDeployment()
    deployment.AppID = req.AppID
    deployment.VersionID = req.VersionID
    deployment.Status = "scheduled"
    deployment.ScheduledAt = req.ScheduledAt
    deployment.AutoDeploy = req.AutoDeploy
    
    if err := models.SaveDeployment(app, deployment); err != nil {
        span.EndWithError(err)
        return handleDeploymentError(e, err, "Failed to schedule deployment")
    }
    
    // Register with scheduler
    h.deployMgr.ScheduleDeployment(deployment.ID, req.ScheduledAt, tunnel.ScheduleConfig{
        AutoCancel:     req.AutoCancel,
        NotifyBefore:   req.NotifyBefore,
        ValidateFirst:  req.ValidateFirst,
    })
    
    span.SetFields(tracer.Fields{
        "deployment.id":        deployment.ID,
        "deployment.scheduled": req.ScheduledAt,
        "deployment.auto":      req.AutoDeploy,
    })
    
    return e.JSON(http.StatusCreated, ScheduleResponse{
        DeploymentID: deployment.ID,
        ScheduledAt:  req.ScheduledAt,
        Status:       "scheduled",
    })
}
```

## Enhanced Error Handling

### Deployment-Specific Error Types
```go
func handleDeploymentError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsDeploymentError(err) {
        return e.JSON(http.StatusUnprocessableEntity, DeploymentErrorResponse{
            Error:      message,
            Details:    err.Error(),
            Phase:      tunnel.GetDeploymentPhase(err),
            Suggestion: tunnel.GetDeploymentSuggestion(err),
            Code:       "DEPLOYMENT_FAILED",
            Retryable:  tunnel.IsRetryable(err),
        })
    }
    
    if tunnel.IsRollbackError(err) {
        return e.JSON(http.StatusConflict, RollbackErrorResponse{
            Error:      message,
            Details:    err.Error(),
            Suggestion: "Check app status and deployment history",
            Code:       "ROLLBACK_FAILED",
            CanRetry:   tunnel.CanRetryRollback(err),
        })
    }
    
    if tunnel.IsValidationError(err) {
        return e.JSON(http.StatusBadRequest, ValidationErrorResponse{
            Error:      message,
            Details:    err.Error(),
            Field:      tunnel.GetValidationField(err),
            Suggestion: tunnel.GetValidationSuggestion(err),
            Code:       "VALIDATION_FAILED",
        })
    }
    
    return handleGenericError(e, err, message)
}

func handleCleanupError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsPermissionError(err) {
        return e.JSON(http.StatusForbidden, PermissionErrorResponse{
            Error:      message,
            Details:    "Insufficient permissions for cleanup operation",
            Suggestion: "Check file system permissions and user access",
            Code:       "PERMISSION_DENIED",
        })
    }
    
    return handleDeploymentError(e, err, message)
}