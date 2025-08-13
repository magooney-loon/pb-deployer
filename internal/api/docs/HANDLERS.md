# API Handlers Documentation

Complete documentation for the PocketBase API handlers implementing the pb-deployer Phase 3 REST API.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         PocketBase Core App                         │
├─────────────────────────────────────────────────────────────────────┤
│                         API Handler Layer                          │
├──────────────┬──────────────┬──────────────┬──────────────────────┤
│   Servers    │     Apps     │   Versions   │     Deployments      │
│   /api/...   │   /api/...   │   /api/...   │      /api/...        │
└──────────────┴──────────────┴──────────────┴──────────────────────┘
       │              │              │                   │
       ▼              ▼              ▼                   ▼
┌──────────┐  ┌──────────┐  ┌──────────┐      ┌──────────────┐
│   SSH    │  │ Service  │  │   File   │      │   Process    │
│ Manager  │  │ Manager  │  │ Manager  │      │   Manager    │
└──────────┘  └──────────┘  └──────────┘      └──────────────┘
```

## Handler Registration

All handlers are registered through the main registration function:

```go
// pb-deployer/internal/handlers/handlers.go
package handlers

import (
    "github.com/pocketbase/pocketbase/core"
    "pb-deployer/internal/handlers/apps"
    "pb-deployer/internal/handlers/deployment"
    "pb-deployer/internal/handlers/server"
    "pb-deployer/internal/handlers/version"
)

// RegisterHandlers registers all API handlers with the application
func RegisterHandlers(app core.App) {
    app.OnServe().BindFunc(func(e *core.ServeEvent) error {
        // Create API group
        apiGroup := e.Router.Group("/api")

        // Register all handlers
        server.RegisterServerHandlers(app, apiGroup)
        apps.RegisterAppsHandlers(app, apiGroup)
        version.RegisterVersionHandlers(app, apiGroup)
        deployment.RegisterDeploymentHandlers(app, apiGroup)

        return e.Next()
    })
}
```

## Server Handlers

### Connection & Status Management

#### POST `/api/servers/{id}/test`
**Comprehensive Connection Testing**

```go
func testServerConnection(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    
    // Get server record from database
    record, err := app.FindRecordById("servers", serverID)
    if err != nil {
        return e.JSON(http.StatusNotFound, ConnectionTestResponse{
            Success: false,
            Error:   "Server not found",
        })
    }

    // Convert to models.Server
    server := recordToServer(record)
    
    // Test TCP connectivity with latency measurement
    tcpResult := testTCPConnectionEnhanced(server.Host, server.Port)
    
    // Test SSH connections based on security status
    var rootSSHResult, appSSHResult SSHTestResult
    
    if server.SecurityLocked {
        // Root SSH should be disabled after security lockdown
        rootSSHResult = SSHTestResult{
            Success:  false,
            Username: server.RootUsername,
            Error:    "Root SSH access disabled after security lockdown",
        }
        appSSHResult = testSSHConnectionEnhanced(server, false, app)
    } else {
        // Test both connections for non-locked servers
        rootSSHResult = testSSHConnectionEnhanced(server, true, app)
        appSSHResult = testSSHConnectionEnhanced(server, false, app)
    }

    // Determine overall status
    overallSuccess := tcpResult.Success && 
        (server.SecurityLocked ? appSSHResult.Success : 
         (rootSSHResult.Success && appSSHResult.Success))

    return e.JSON(http.StatusOK, ConnectionTestResponse{
        Success:           overallSuccess,
        TCPConnection:     tcpResult,
        RootSSHConnection: rootSSHResult,
        AppSSHConnection:  appSSHResult,
        OverallStatus:     determineOverallStatus(server, tcpResult, rootSSHResult, appSSHResult),
    })
}
```

**Response Format:**
```json
{
  "success": true,
  "tcp_connection": {
    "success": true,
    "latency": "15.23ms"
  },
  "root_ssh_connection": {
    "success": false,
    "username": "root",
    "error": "Root SSH access disabled after security lockdown"
  },
  "app_ssh_connection": {
    "success": true,
    "username": "pocketbase",
    "auth_method": "ssh_agent"
  },
  "overall_status": "healthy_secured"
}
```

#### GET `/api/servers/{id}/status`
**Quick Server Status Check**

```go
func getServerStatus(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    record, err := app.FindRecordById("servers", serverID)
    if err != nil {
        return e.JSON(http.StatusNotFound, map[string]string{
            "error": "Server not found",
        })
    }

    host := record.GetString("host")
    port := record.GetInt("port")
    
    // Quick TCP connection test with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    
    connected, connErr := testTCPConnectionWithContext(ctx, host, port)
    
    status := map[string]any{
        "server_id":       serverID,
        "setup_complete":  record.GetBool("setup_complete"),
        "security_locked": record.GetBool("security_locked"),
        "connection":      "offline",
        "timestamp":       time.Now().UTC().Format(time.RFC3339),
    }

    if connected {
        status["connection"] = "online"
    } else if connErr != nil {
        status["connection_error"] = connErr.Error()
    }

    return e.JSON(http.StatusOK, status)
}
```

#### GET `/api/servers/{id}/health`
**Connection Pool Health Monitoring**

```go
func getConnectionHealth(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    server := getServerFromDB(app, serverID)
    
    // Get SSH service for health monitoring
    sshService := ssh.GetSSHService()
    connectionStatus := sshService.GetConnectionStatus()
    healthMetrics := sshService.GetHealthMetrics()
    
    // Get connection keys for this server
    rootKey := sshService.GetConnectionKey(server, true)
    appKey := sshService.GetConnectionKey(server, false)
    
    return e.JSON(http.StatusOK, map[string]any{
        "server_id": serverID,
        "connections": map[string]any{
            "root": getConnectionDetails(connectionStatus, rootKey, server.SecurityLocked),
            "app":  getConnectionDetails(connectionStatus, appKey, false),
        },
        "overall_metrics": healthMetrics,
    })
}
```

### Server Setup & Security

#### POST `/api/servers/{id}/setup`
**Server Initial Setup**

```go
func runServerSetup(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    server := getServerFromDB(app, serverID)
    
    // Check if setup is already complete
    if server.SetupComplete {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server setup is already complete",
        })
    }

    // Start setup process in background with progress notifications
    go func() {
        progressChan := make(chan ssh.SetupStep, 10)
        
        // Monitor progress and send WebSocket updates
        go func() {
            for step := range progressChan {
                notifySetupProgress(app, serverID, step)
            }
        }()

        // Run actual setup
        sshService := ssh.GetSSHService()
        err := sshService.RunServerSetup(server, progressChan)
        
        if err != nil {
            notifySetupProgress(app, serverID, ssh.SetupStep{
                Step: "complete", Status: "failed",
                Message: "Server setup failed", Error: err.Error(),
            })
            return
        }

        // Update database
        updateServerSetupStatus(app, serverID, true)
        
        notifySetupProgress(app, serverID, ssh.SetupStep{
            Step: "complete", Status: "success",
            Message: "Server setup completed successfully",
        })
    }()

    return e.JSON(http.StatusOK, map[string]any{
        "message": "Server setup started",
        "server_id": serverID,
    })
}
```

#### POST `/api/servers/{id}/security`
**Security Lockdown**

```go
func applySecurityLockdown(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    server := getServerFromDB(app, serverID)
    
    // Validate prerequisites
    if !server.SetupComplete {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server setup must be completed before applying security lockdown",
        })
    }
    
    if server.SecurityLocked {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server security is already locked down",
        })
    }

    // Start security lockdown in background
    go func() {
        progressChan := make(chan ssh.SetupStep, 10)
        
        go func() {
            for step := range progressChan {
                notifySecurityProgress(app, serverID, step)
            }
        }()

        sshService := ssh.GetSSHService()
        err := sshService.ApplySecurityLockdown(server, progressChan)
        
        if err != nil {
            notifySecurityProgress(app, serverID, ssh.SetupStep{
                Step: "complete", Status: "failed",
                Message: "Security lockdown failed", Error: err.Error(),
            })
            return
        }

        // Update database - mark as security locked
        updateServerSecurityStatus(app, serverID, true)
        
        notifySecurityProgress(app, serverID, ssh.SetupStep{
            Step: "complete", Status: "success",
            Message: "Security lockdown completed successfully",
        })
    }()

    return e.JSON(http.StatusOK, map[string]any{
        "message": "Security lockdown started",
        "server_id": serverID,
    })
}
```

### Troubleshooting

#### POST `/api/servers/{id}/troubleshoot`
**Comprehensive SSH Troubleshooting**

```go
func troubleshootServerConnection(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    server := getServerFromDB(app, serverID)
    
    // Get query parameters
    enhanced := e.Request.URL.Query().Get("enhanced") == "true"
    autoFix := e.Request.URL.Query().Get("auto_fix") == "true"
    
    clientIP := getClientIP(e.Request)
    
    // Run appropriate diagnostics
    var diagnostics []ssh.ConnectionDiagnostic
    var err error
    
    if server.SecurityLocked {
        diagnostics, err = ssh.DiagnoseAppUserPostSecurity(server)
    } else {
        diagnostics, err = ssh.TroubleshootConnection(server, clientIP)
    }
    
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": fmt.Sprintf("Troubleshooting failed: %v", err),
        })
    }

    // Process diagnostics into response
    response := processDiagnostics(server, diagnostics, clientIP)
    
    // Auto-fix if requested and possible
    if autoFix && response.CanAutoFix {
        fixResults := ssh.FixCommonIssues(server)
        response.Diagnostics = append(response.Diagnostics, fixResults...)
        response = processDiagnostics(server, response.Diagnostics, clientIP)
    }

    if enhanced {
        enhancedResponse := enhanceResponse(response, server)
        return e.JSON(http.StatusOK, enhancedResponse)
    }

    return e.JSON(http.StatusOK, response)
}
```

## App Handlers

### CRUD Operations

#### GET `/api/apps`
**List Applications**

```go
func listApps(app core.App, e *core.RequestEvent) error {
    // Get optional server filter
    serverID := e.Request.URL.Query().Get("server_id")
    
    var records []*core.Record
    var err error
    
    if serverID != "" {
        records, err = app.FindRecordsByFilter("apps", 
            "server_id = {:server_id}", "", 0, 0, 
            map[string]any{"server_id": serverID})
    } else {
        records, err = app.FindRecordsByFilter("apps", 
            "", "-created", 0, 0, nil)
    }
    
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to fetch apps",
        })
    }

    // Convert records to response format
    apps := make([]map[string]any, len(records))
    for i, record := range records {
        apps[i] = recordToAppResponse(record)
    }

    return e.JSON(http.StatusOK, map[string]any{
        "apps":  apps,
        "count": len(apps),
    })
}
```

#### POST `/api/apps`
**Create Application**

```go
func createApp(app core.App, e *core.RequestEvent) error {
    var req AppCreateRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    // Validate required fields
    if err := validateAppCreateRequest(req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": err.Error(),
        })
    }

    // Verify server exists
    serverRecord, err := app.FindRecordById("servers", req.ServerID)
    if err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid server ID",
        })
    }

    // Check for duplicate app name on server
    if err := checkAppNameUnique(app, req.Name, req.ServerID); err != nil {
        return e.JSON(http.StatusConflict, map[string]string{
            "error": err.Error(),
        })
    }

    // Create app record
    record := createAppRecord(app, req)
    if err := app.Save(record); err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to create app",
        })
    }

    response := recordToAppResponse(record)
    response["server_name"] = serverRecord.GetString("name")

    return e.JSON(http.StatusCreated, response)
}
```

### Service Management

#### POST `/api/apps/{id}/start`
**Start Application Service**

```go
func startAppService(app core.App, e *core.RequestEvent) error {
    return handleServiceAction(app, e, "start")
}

func handleServiceAction(app core.App, e *core.RequestEvent, action string) error {
    appID := e.Request.PathValue("id")
    appRecord := getAppFromDB(app, appID)
    serverRecord := getServerFromDB(app, appRecord.GetString("server_id"))
    
    server := recordToServer(serverRecord)
    serviceName := appRecord.GetString("service_name")
    
    // Check server readiness
    if !server.SetupComplete {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server setup is not complete",
        })
    }

    // Create SSH manager based on security status
    var sshManager *ssh.SSHManager
    var err error
    
    if server.SecurityLocked {
        sshManager, err = ssh.NewSSHManager(server, false) // app user
    } else {
        sshManager, err = ssh.NewSSHManager(server, true)  // root user
    }
    
    if err != nil {
        return e.JSON(http.StatusInternalServerError, ServiceActionResponse{
            AppID: appID, ServiceName: serviceName, Action: action,
            Success: false, Error: err.Error(),
        })
    }
    defer sshManager.Close()

    // Perform service action
    var actionErr error
    switch action {
    case "start":
        actionErr = sshManager.StartService(serviceName)
    case "stop":
        actionErr = sshManager.StopService(serviceName)
    case "restart":
        actionErr = sshManager.RestartService(serviceName)
    }

    // Get service status after action
    status, _ := sshManager.GetServiceStatus(serviceName)
    
    response := ServiceActionResponse{
        AppID: appID, ServiceName: serviceName, Action: action,
        Success: actionErr == nil, Status: status,
        Timestamp: time.Now().UTC(),
    }
    
    if actionErr != nil {
        response.Error = actionErr.Error()
        response.Message = fmt.Sprintf("Failed to %s service", action)
    } else {
        response.Message = fmt.Sprintf("Service %s successful", action)
        
        // Update app status in database
        updateAppStatus(app, appID, action, status)
    }

    statusCode := http.StatusOK
    if actionErr != nil {
        statusCode = http.StatusInternalServerError
    }

    return e.JSON(statusCode, response)
}
```

### Deployment Operations

#### POST `/api/apps/{id}/deploy`
**Deploy Application**

```go
func deployApp(app core.App, e *core.RequestEvent) error {
    appID := e.Request.PathValue("id")
    
    var req DeploymentRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    // Validate deployment request
    if err := validateDeploymentRequest(req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": err.Error(),
        })
    }

    appRecord := getAppFromDB(app, appID)
    versionRecord := getVersionFromDB(app, req.VersionID)
    
    // Verify version belongs to app
    if versionRecord.GetString("app_id") != appID {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Version does not belong to this app",
        })
    }

    // Check if first deployment
    isFirstDeploy := appRecord.GetString("current_version") == ""
    
    if isFirstDeploy && (req.SuperuserEmail == "" || req.SuperuserPass == "") {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "First deployment requires superuser credentials",
        })
    }

    // Create deployment record
    deploymentRecord := createDeploymentRecord(app, appID, req.VersionID, req.Notes)
    
    // Start deployment in background
    go func() {
        deploymentErr := performDeployment(app, deploymentRecord.Id, 
            appRecord, versionRecord, req, isFirstDeploy)
        
        updateDeploymentResult(app, deploymentRecord.Id, deploymentErr, 
            appRecord, versionRecord)
    }()

    return e.JSON(http.StatusAccepted, map[string]any{
        "message": "Deployment started",
        "deployment_id": deploymentRecord.Id,
        "app_id": appID,
        "version_id": req.VersionID,
        "is_first_deploy": isFirstDeploy,
    })
}
```

#### POST `/api/apps/{id}/rollback`
**Rollback Application**

```go
func rollbackApp(app core.App, e *core.RequestEvent) error {
    appID := e.Request.PathValue("id")
    
    var req RollbackRequest
    if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Invalid request body",
        })
    }

    appRecord := getAppFromDB(app, appID)
    versionRecord := getVersionFromDB(app, req.VersionID)
    
    // Verify version belongs to app
    if versionRecord.GetString("app_id") != appID {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Version does not belong to this app",
        })
    }

    // Create rollback deployment record
    deploymentRecord := createDeploymentRecord(app, appID, req.VersionID, 
        "Rollback: " + req.Notes)
    
    // Start rollback in background
    go func() {
        rollbackErr := performRollback(app, deploymentRecord.Id, 
            appRecord, versionRecord, req)
        
        updateDeploymentResult(app, deploymentRecord.Id, rollbackErr, 
            appRecord, versionRecord)
    }()

    return e.JSON(http.StatusAccepted, map[string]any{
        "message": "Rollback started",
        "deployment_id": deploymentRecord.Id,
        "app_id": appID,
        "version_id": req.VersionID,
    })
}
```

## Version Handlers

### File Management

#### POST `/api/versions/{id}/upload`
**Upload Deployment Files**

```go
func uploadVersionZip(app core.App, e *core.RequestEvent) error {
    versionID := e.Request.PathValue("id")
    versionRecord := getVersionFromDB(app, versionID)
    
    // Parse multipart form (150MB max)
    if err := e.Request.ParseMultipartForm(157286400); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Failed to parse multipart form",
        })
    }

    // Get PocketBase binary file
    binaryFile, binaryHeader, err := e.Request.FormFile("pocketbase_binary")
    if err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "PocketBase binary file is required",
        })
    }
    defer binaryFile.Close()

    // Get public folder files
    publicFiles := e.Request.MultipartForm.File["pb_public_files"]
    if len(publicFiles) == 0 {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "pb_public folder files are required",
        })
    }

    // Validate file sizes
    if err := validateUploadSizes(binaryHeader, publicFiles); err != nil {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": err.Error(),
        })
    }

    // Create deployment ZIP in memory
    zipBuffer, err := createDeploymentZip(binaryFile, publicFiles)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to create deployment package",
        })
    }

    // Save ZIP file to PocketBase filesystem
    deploymentFilename := fmt.Sprintf("deployment_%s_%d.zip", 
        versionRecord.GetString("version_number"), time.Now().Unix())
    
    deploymentFile, err := filesystem.NewFileFromBytes(zipBuffer.Bytes(), 
        deploymentFilename)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to create deployment package file",
        })
    }

    // Update version record with deployment file
    versionRecord.Set("deployment_zip", deploymentFile)
    if err := app.Save(versionRecord); err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to save version record",
        })
    }

    return e.JSON(http.StatusOK, map[string]any{
        "message": "Version files uploaded successfully",
        "version_id": versionID,
        "deployment_file": deploymentFilename,
        "deployment_size": zipBuffer.Len(),
    })
}
```

#### GET `/api/versions/{id}/download`
**Download Deployment Package**

```go
func downloadVersionZip(app core.App, e *core.RequestEvent) error {
    versionID := e.Request.PathValue("id")
    versionRecord := getVersionFromDB(app, versionID)
    
    // Check if deployment zip exists
    deploymentZip := versionRecord.GetString("deployment_zip")
    if deploymentZip == "" {
        return e.JSON(http.StatusNotFound, map[string]string{
            "error": "No deployment zip found for this version",
        })
    }

    // Get PocketBase filesystem
    filesystem, err := app.NewFilesystem()
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to access file system",
        })
    }
    defer filesystem.Close()

    // Serve the file directly
    serveKey := versionRecord.BaseFilesPath() + "/" + deploymentZip
    return filesystem.Serve(e.Response, e.Request, serveKey, deploymentZip)
}
```

## Deployment Handlers

### Process Management

#### GET `/api/deployments`
**List Deployments**

```go
func listDeployments(app core.App, e *core.RequestEvent) error {
    // Get filters
    appID := e.Request.URL.Query().Get("app_id")
    status := e.Request.URL.Query().Get("status")
    limit := getIntQueryParam(e.Request, "limit", 50, 100)

    // Build filter conditions
    filter, params := buildDeploymentFilter(appID, status)

    // Get deployments
    records, err := app.FindRecordsByFilter("deployments", filter, 
        "-created", limit, 0, params)
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to fetch deployments",
        })
    }

    // Convert to response format
    deployments := make([]DeploymentResponse, len(records))
    for i, record := range records {
        deployments[i] = recordToDeploymentResponse(record, app)
    }

    return e.JSON(http.StatusOK, map[string]any{
        "deployments": deployments,
        "count": len(deployments),
        "filters": map[string]any{
            "app_id": appID,
            "status": status,
            "limit": limit,
        },
    })
}
```

#### POST `/api/deployments/{id}/cancel`
**Cancel Running Deployment**

```go
func cancelDeployment(app core.App, e *core.RequestEvent) error {
    deploymentID := e.Request.PathValue("id")
    deploymentRecord := getDeploymentFromDB(app, deploymentID)
    
    status := deploymentRecord.GetString("status")
    
    // Check if deployment can be canceled
    if status != "running" && status != "pending" {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Deployment cannot be canceled in current status: " + status,
        })
    }

    // Update deployment status to canceled
    now := time.Now()
    deploymentRecord.Set("status", "failed")
    deploymentRecord.Set("completed_at", now)
    
    currentLogs := deploymentRecord.GetString("logs")
    deploymentRecord.Set("logs", currentLogs + 
        "\nDeployment canceled by user at " + now.Format(time.RFC3339))

    if err := app.Save(deploymentRecord); err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to cancel deployment",
        })
    }

    return e.JSON(http.StatusOK, map[string]any{
        "message": "Deployment canceled successfully",
        "deployment_id": deploymentID,
        "canceled_at": now.UTC(),
    })
}
```

#### GET `/api/deployments/stats`
**Deployment Statistics**

```go
func getDeploymentStats(app core.App, e *core.RequestEvent) error {
    days := getIntQueryParam(e.Request, "days", 30, 365)
    since := time.Now().AddDate(0, 0, -days)

    // Get deployments in time range
    records, err := app.FindRecordsByFilter("deployments", 
        "created >= {:since}", "-created", 0, 0, 
        map[string]any{"since": since})
    if err != nil {
        return e.JSON(http.StatusInternalServerError, map[string]string{
            "error": "Failed to fetch deployment statistics",
        })
    }

    // Calculate statistics
    stats := calculateDeploymentStats(records, app)

    return e.JSON(http.StatusOK, stats)
}
```

## WebSocket Integration

### Real-time Progress Updates

All long-running operations support real-time progress via PocketBase's built-in realtime system:

```go
// Server setup progress
func notifySetupProgress(app core.App, serverID string, step ssh.SetupStep) error {
    subscription := fmt.Sprintf("server_setup_%s", serverID)
    return notifyClients(app, subscription, step)
}

// Security lockdown progress
func notifySecurityProgress(app core.App, serverID string, step ssh.SetupStep) error {
    subscription := fmt.Sprintf("server_security_%s", serverID)
    return notifyClients(app, subscription, step)
}

// Deployment progress
func notifyDeploymentProgress(app core.App, deploymentID string, step DeploymentStep) error {
    subscription := fmt.Sprintf("deployment_progress_%s", deploymentID)
    return notifyClients(app, subscription, step)
}

// Generic notification helper
func notifyClients(app core.App, subscription string, data any) error {
    rawData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal data: %w", err)
    }

    message := subscriptions.Message{
        Name: subscription,
        Data: rawData,
    }

    group := new(errgroup.Group)
    chunks := app.SubscriptionsBroker().ChunkedClients(300)

    for _, chunk := range chunks {
        group.Go(func() error {
            for _, client := range chunk {
                if client.HasSubscription(subscription) {
                    client.Send(message)
                }
            }
            return nil
        })
    }

    return group.Wait()
}
```

### WebSocket Endpoints

**GET `/api/servers/{id}/setup-ws`**
**GET `/api/servers/{id}/security-ws`**
**GET `/api/apps/{id}/deploy-ws`**
**GET `/api/deployments/{id}/ws`**

```go
func handleSetupWebSocket(app core.App, e *core.RequestEvent) error {
    serverID := e.Request.PathValue("id")
    
    return e.JSON(http.StatusOK, map[string]any{
        "message": "Setup progress available via PocketBase realtime",
        "subscription": fmt.Sprintf("server_setup_%s", serverID),
        "event_types": []string{"init", "create_user", "install_packages", 
                                 "setup_directories", "configure_firewall", "complete"},
    })
}
```

## Error Handling Patterns

### Consistent Error Response Format

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Details string `json:"details,omitempty"`
    Code    string `json:"code,omitempty"`
}

func handleError(e *core.RequestEvent, statusCode int, message string, details ...string) error {
    response := ErrorResponse{Error: message}
    if len(details) > 0 {
        response.Details = details[0]
    }
    
    return e.JSON(statusCode, response)
}
```

### Database Error Handling

```go
func handleDatabaseError(app core.App, e *core.RequestEvent, err error, operation string) error {
    app.Logger().Error("Database operation failed", 
        "operation", operation, 
        "error", err)
    
    if strings.Contains(err.Error(), "no rows") {
        return handleError(e, http.StatusNotFound, "Resource not found")
    }
    
    if strings.Contains(err.Error(), "UNIQUE constraint failed") {
        return handleError(e, http.StatusConflict, "Resource already exists")
    }
    
    return handleError(e, http.StatusInternalServerError, 
        "Database operation failed", err.Error())
}
```

### SSH Connection Error Handling

```go
func handleSSHError(e *core.RequestEvent, err error, operation string) error {
    if strings.Contains(err.Error(), "connection refused") {
        return handleError(e, http.StatusBadGateway, 
            "Cannot connect to server", 
            "Server may be down or SSH service not running")
    }
    
    if strings.Contains(err.Error(), "authentication failed") {
        return handleError(e, http.StatusUnauthorized, 
            "SSH authentication failed", 
            "Check SSH keys and user permissions")
    }
    
    if strings.Contains(err.Error(), "timeout") {
        return handleError(e, http.StatusRequestTimeout, 
            "SSH connection timeout", 
            "Server may be overloaded or network issues")
    }
    
    return handleError(e, http.StatusInternalServerError, 
        "SSH operation failed", err.Error())
}
```

## Utility Functions

### Record Conversion Helpers

```go
// Convert PocketBase record to models.Server
func recordToServer(record *core.Record) *models.Server {
    return &models.Server{
        ID:             record.Id,
        Name:           record.GetString("name"),
        Host:           record.GetString("host"),
        Port:           record.GetInt("port"),
        RootUsername:   record.GetString("root_username"),
        AppUsername:    record.GetString("app_username"),
        UseSSHAgent:    record.GetBool("use_ssh_agent"),
        ManualKeyPath:  record.GetString("manual_key_path"),
        SetupComplete:  record.GetBool("setup_complete"),
        SecurityLocked: record.GetBool("security_locked"),
    }
}

// Convert PocketBase record to app response
func recordToAppResponse(record *core.Record) map[string]any {
    return map[string]any{
        "id":              record.Id,
        "name":            record.GetString("name"),
        "server_id":       record.GetString("server_id"),
        "domain":          record.GetString("domain"),
        "remote_path":     record.GetString("remote_path"),
        "service_name":    record.GetString("service_name"),
        "current_version": record.GetString("current_version"),
        "status":          record.GetString("status"),
        "created":         record.GetDateTime("created"),
        "updated":         record.GetDateTime("updated"),
    }
}

// Convert PocketBase record to version response
func recordToVersionResponse(record *core.Record, app core.App) VersionResponse {
    response := VersionResponse{
        ID:            record.Id,
        AppID:         record.GetString("app_id"),
        VersionNumber: record.GetString("version_number"),
        Notes:         record.GetString("notes"),
        Created:       record.GetDateTime("created").Time(),
        Updated:       record.GetDateTime("updated").Time(),
    }
    
    // Check if deployment zip exists
    deploymentZip := record.GetString("deployment_zip")
    response.HasZip = deploymentZip != ""
    
    return response
}

// Convert PocketBase record to deployment response
func recordToDeploymentResponse(record *core.Record, app core.App) DeploymentResponse {
    response := DeploymentResponse{
        ID:        record.Id,
        AppID:     record.GetString("app_id"),
        VersionID: record.GetString("version_id"),
        Status:    record.GetString("status"),
        Logs:      record.GetString("logs"),
        Created:   record.GetDateTime("created").Time(),
        Updated:   record.GetDateTime("updated").Time(),
    }
    
    // Set timing information
    if startedAt := record.GetDateTime("started_at"); !startedAt.Time().IsZero() {
        response.StartedAt = startedAt.Time()
    }
    
    if completedAt := record.GetDateTime("completed_at"); !completedAt.Time().IsZero() {
        response.CompletedAt = completedAt.Time()
        if !response.StartedAt.IsZero() {
            duration := response.CompletedAt.Sub(response.StartedAt)
            response.Duration = utils.FormatDuration(duration)
        }
    }
    
    // Get version and app info
    if versionRecord, err := app.FindRecordById("versions", response.VersionID); err == nil {
        response.Version = versionRecord.GetString("version_number")
    }
    
    if appRecord, err := app.FindRecordById("apps", response.AppID); err == nil {
        response.AppName = appRecord.GetString("name")
    }
    
    return response
}
```

### Query Parameter Helpers

```go
// Get integer query parameter with default and max values
func getIntQueryParam(r *http.Request, param string, defaultVal, maxVal int) int {
    if str := r.URL.Query().Get(param); str != "" {
        if val, err := strconv.Atoi(str); err == nil && val > 0 && val <= maxVal {
            return val
        }
    }
    return defaultVal
}

// Get boolean query parameter
func getBoolQueryParam(r *http.Request, param string, defaultVal bool) bool {
    if str := r.URL.Query().Get(param); str != "" {
        if val, err := strconv.ParseBool(str); err == nil {
            return val
        }
    }
    return defaultVal
}

// Get client IP from request headers
func getClientIP(r *http.Request) string {
    // Check common proxy headers
    headers := []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}
    
    for _, header := range headers {
        if ip := r.Header.Get(header); ip != "" {
            if strings.Contains(ip, ",") {
                return strings.TrimSpace(strings.Split(ip, ",")[0])
            }
            return ip
        }
    }
    
    // Fall back to RemoteAddr
    if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
        return host
    }
    
    return r.RemoteAddr
}
```

### Database Helpers

```go
// Get server from database with error handling
func getServerFromDB(app core.App, serverID string) (*core.Record, error) {
    record, err := app.FindRecordById("servers", serverID)
    if err != nil {
        return nil, fmt.Errorf("server not found: %w", err)
    }
    return record, nil
}

// Get app from database with error handling
func getAppFromDB(app core.App, appID string) (*core.Record, error) {
    record, err := app.FindRecordById("apps", appID)
    if err != nil {
        return nil, fmt.Errorf("app not found: %w", err)
    }
    return record, nil
}

// Get version from database with error handling
func getVersionFromDB(app core.App, versionID string) (*core.Record, error) {
    record, err := app.FindRecordById("versions", versionID)
    if err != nil {
        return nil, fmt.Errorf("version not found: %w", err)
    }
    return record, nil
}

// Get deployment from database with error handling
func getDeploymentFromDB(app core.App, deploymentID string) (*core.Record, error) {
    record, err := app.FindRecordById("deployments", deploymentID)
    if err != nil {
        return nil, fmt.Errorf("deployment not found: %w", err)
    }
    return record, nil
}
```

## Validation Functions

### Request Validation

```go
// Validate app creation request
func validateAppCreateRequest(req AppCreateRequest) error {
    if req.Name == "" {
        return fmt.Errorf("app name is required")
    }
    
    if req.ServerID == "" {
        return fmt.Errorf("server ID is required")
    }
    
    if req.Domain == "" {
        return fmt.Errorf("domain is required")
    }
    
    if err := validateAppName(req.Name); err != nil {
        return err
    }
    
    if err := validateDomain(req.Domain); err != nil {
        return err
    }
    
    return nil
}

// Validate deployment request
func validateDeploymentRequest(req DeploymentRequest) error {
    if req.VersionID == "" {
        return fmt.Errorf("version ID is required")
    }
    
    return nil
}

// Validate app name format
func validateAppName(name string) error {
    if len(name) < 1 || len(name) > 50 {
        return fmt.Errorf("app name must be between 1 and 50 characters")
    }
    
    // Allow alphanumeric, dash, underscore
    for _, char := range name {
        if !((char >= 'a' && char <= 'z') ||
            (char >= 'A' && char <= 'Z') ||
            (char >= '0' && char <= '9') ||
            char == '-' || char == '_') {
            return fmt.Errorf("app name can only contain letters, numbers, hyphens, and underscores")
        }
    }
    
    // Cannot start or end with dash or underscore
    if strings.HasPrefix(name, "-") || strings.HasPrefix(name, "_") ||
        strings.HasSuffix(name, "-") || strings.HasSuffix(name, "_") {
        return fmt.Errorf("app name cannot start or end with dash or underscore")
    }
    
    return nil
}

// Validate domain format
func validateDomain(domain string) error {
    if len(domain) < 3 || len(domain) > 255 {
        return fmt.Errorf("domain must be between 3 and 255 characters")
    }
    
    if !strings.Contains(domain, ".") {
        return fmt.Errorf("domain must contain at least one dot")
    }
    
    return nil
}

// Validate version number format
func validateVersionNumber(version string) error {
    if len(version) < 1 || len(version) > 50 {
        return fmt.Errorf("version number must be between 1 and 50 characters")
    }
    
    if strings.TrimSpace(version) == "" {
        return fmt.Errorf("version number cannot be empty or just whitespace")
    }
    
    return nil
}
```

### File Upload Validation

```go
// Validate upload file sizes
func validateUploadSizes(binaryHeader *multipart.FileHeader, publicFiles []*multipart.FileHeader) error {
    // Check binary size (100MB max)
    if binaryHeader.Size > 104857600 {
        return fmt.Errorf("binary file size exceeds 100MB limit")
    }
    
    // Check total public files size (50MB max)
    var totalPublicSize int64
    for _, fileHeader := range publicFiles {
        totalPublicSize += fileHeader.Size
    }
    
    if totalPublicSize > 52428800 {
        return fmt.Errorf("public folder total size exceeds 50MB limit")
    }
    
    return nil
}

// Create deployment ZIP from uploaded files
func createDeploymentZip(binaryFile multipart.File, publicFiles []*multipart.FileHeader) (*bytes.Buffer, error) {
    var zipBuffer bytes.Buffer
    zipWriter := zip.NewWriter(&zipBuffer)
    defer zipWriter.Close()
    
    // Add binary file
    binaryWriter, err := zipWriter.Create("pocketbase")
    if err != nil {
        return nil, err
    }
    
    if _, err := io.Copy(binaryWriter, binaryFile); err != nil {
        return nil, err
    }
    
    // Add public folder files
    for _, fileHeader := range publicFiles {
        file, err := fileHeader.Open()
        if err != nil {
            return nil, err
        }
        defer file.Close()
        
        // Create file in ZIP under pb_public/
        deploymentPath := fmt.Sprintf("pb_public/%s", fileHeader.Filename)
        writer, err := zipWriter.Create(deploymentPath)
        if err != nil {
            return nil, err
        }
        
        if _, err := io.Copy(writer, file); err != nil {
            return nil, err
        }
    }
    
    return &zipBuffer, nil
}
```

## Background Process Management

### Deployment Process

```go
// Perform actual deployment
func performDeployment(app core.App, deploymentID string, appRecord, versionRecord *core.Record, 
    req DeploymentRequest, isFirstDeploy bool) error {
    
    app.Logger().Info("Starting deployment process", 
        "deployment_id", deploymentID,
        "app_id", appRecord.Id,
        "version_id", versionRecord.Id,
        "is_first_deploy", isFirstDeploy)

    // Get server information
    serverRecord, err := app.FindRecordById("servers", appRecord.GetString("server_id"))
    if err != nil {
        return fmt.Errorf("failed to get server: %w", err)
    }
    
    server := recordToServer(serverRecord)
    
    // Steps:
    // 1. Download deployment ZIP from PocketBase storage
    // 2. Extract files locally
    // 3. Stop existing service (if not first deploy)
    // 4. Rsync files to remote server
    // 5. Set up systemd service
    // 6. Create superuser (if first deploy)
    // 7. Start service
    // 8. Health check
    
    deploymentManager := ssh.GetDeploymentManager()
    return deploymentManager.Deploy(server, appRecord, versionRecord, req, isFirstDeploy)
}

// Perform rollback
func performRollback(app core.App, deploymentID string, appRecord, versionRecord *core.Record, 
    req RollbackRequest) error {
    
    app.Logger().Info("Starting rollback process", 
        "deployment_id", deploymentID,
        "app_id", appRecord.Id,
        "target_version_id", versionRecord.Id)

    // Steps:
    // 1. Stop current service
    // 2. Download target version ZIP
    // 3. Extract files locally
    // 4. Rsync to remote server
    // 5. Start service
    // 6. Health check
    
    deploymentManager := ssh.GetDeploymentManager()
    return deploymentManager.Rollback(server, appRecord, versionRecord, req)
}
```

### Status Updates

```go
// Update deployment result in database
func updateDeploymentResult(app core.App, deploymentID string, deploymentErr error, 
    appRecord, versionRecord *core.Record) {
    
    deploymentRecord, err := app.FindRecordById("deployments", deploymentID)
    if err != nil {
        app.Logger().Error("Failed to find deployment record for update", 
            "deployment_id", deploymentID, "error", err)
        return
    }
    
    now := time.Now()
    deploymentRecord.Set("completed_at", now)
    
    if deploymentErr != nil {
        deploymentRecord.Set("status", "failed")
        deploymentRecord.Set("logs", 
            deploymentRecord.GetString("logs") + "\nDeployment failed: " + deploymentErr.Error())
        
        app.Logger().Error("Deployment failed", 
            "deployment_id", deploymentID, 
            "error", deploymentErr)
    } else {
        deploymentRecord.Set("status", "success")
        deploymentRecord.Set("logs", 
            deploymentRecord.GetString("logs") + "\nDeployment completed successfully")
        
        // Update app's current version
        appRecord.Set("current_version", versionRecord.GetString("version_number"))
        appRecord.Set("status", "online")
        if saveErr := app.Save(appRecord); saveErr != nil {
            app.Logger().Error("Failed to update app version", 
                "app_id", appRecord.Id, "error", saveErr)
        }
        
        app.Logger().Info("Deployment completed successfully", 
            "deployment_id", deploymentID,
            "version", versionRecord.GetString("version_number"))
    }
    
    if saveErr := app.Save(deploymentRecord); saveErr != nil {
        app.Logger().Error("Failed to update deployment record", 
            "deployment_id", deploymentID, "error", saveErr)
    }
}

// Update app status after service action
func updateAppStatus(app core.App, appID, action, serviceStatus string) {
    appRecord, err := app.FindRecordById("apps", appID)
    if err != nil {
        app.Logger().Error("Failed to find app for status update", 
            "app_id", appID, "error", err)
        return
    }
    
    var newStatus string
    switch action {
    case "start", "restart":
        if serviceStatus == "active" {
            newStatus = "online"
        } else {
            newStatus = "offline"
        }
    case "stop":
        newStatus = "offline"
    default:
        return // No status change for other actions
    }
    
    appRecord.Set("status", newStatus)
    if err := app.Save(appRecord); err != nil {
        app.Logger().Error("Failed to update app status", 
            "app_id", appID, "new_status", newStatus, "error", err)
    }
}
```

## Performance Optimizations

### Connection Pooling

```go
// Use SSH connection pooling for better performance
func getSSHConnection(server *models.Server, asRoot bool) (*ssh.SSHManager, error) {
    connectionManager := ssh.GetConnectionManager()
    
    // Try to get existing connection from pool
    if conn := connectionManager.GetConnection(server, asRoot); conn != nil {
        return conn, nil
    }
    
    // Create new connection and add to pool
    conn, err := ssh.NewSSHManager(server, asRoot)
    if err != nil {
        return nil, err
    }
    
    connectionManager.AddConnection(server, asRoot, conn)
    return conn, nil
}
```

### Batch Operations

```go
// Process multiple deployments efficiently
func batchProcessDeployments(app core.App, deploymentIDs []string) error {
    // Group deployments by server to optimize SSH connections
    deploymentsByServer := make(map[string][]string)
    
    for _, deploymentID := range deploymentIDs {
        deployment, err := app.FindRecordById("deployments", deploymentID)
        if err != nil {
            continue
        }
        
        appRecord, err := app.FindRecordById("apps", deployment.GetString("app_id"))
        if err != nil {
            continue
        }
        
        serverID := appRecord.GetString("server_id")
        deploymentsByServer[serverID] = append(deploymentsByServer[serverID], deploymentID)
    }
    
    // Process each server's deployments using the same SSH connection
    for serverID, serverDeployments := range deploymentsByServer {
        processServerDeployments(app, serverID, serverDeployments)
    }
    
    return nil
}
```

## Testing Considerations

### Handler Testing

```go
// Example test for app creation handler
func TestCreateApp(t *testing.T) {
    app := setupTestApp(t)
    defer cleanupTestApp(app)
    
    // Create test server first
    serverRecord := createTestServer(app)
    
    // Test valid app creation
    reqBody := `{
        "name": "test-app",
        "server_id": "` + serverRecord.Id + `",
        "domain": "test.example.com"
    }`
    
    req := httptest.NewRequest("POST", "/api/apps", strings.NewReader(reqBody))
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    e := &core.RequestEvent{
        Request:  req,
        Response: w,
    }
    
    err := createApp(app, e)
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, w.Code)
    
    // Verify app was created in database
    var response map[string]any
    json.Unmarshal(w.Body.Bytes(), &response)
    
    appID := response["id"].(string)
    appRecord, err := app.FindRecordById("apps", appID)
    assert.NoError(t, err)
    assert.Equal(t, "test-app", appRecord.GetString("name"))
}
```

### Integration Testing

```go
// Test complete deployment flow
func TestFullDeploymentFlow(t *testing.T) {
    app := setupTestApp(t)
    defer cleanupTestApp(app)
    
    // 1. Create server
    server := createTestServer(app)
    
    // 2. Run server setup
    runServerSetup(app, server.Id)
    waitForSetupCompletion(app, server.Id)
    
    // 3. Create app
    appRecord := createTestApp(app, server.Id)
    
    // 4. Create version with deployment files
    versionRecord := createTestVersion(app, appRecord.Id)
    uploadTestDeploymentFiles(app, versionRecord.Id)
    
    // 5. Deploy app
    deploymentRecord := deployTestApp(app, appRecord.Id, versionRecord.Id)
    waitForDeploymentCompletion(app, deploymentRecord.Id)
    
    // 6. Verify deployment success
    verifyAppDeployment(app, appRecord.Id)
}
```

## Integration Examples

### Usage in Main Application

```go
// main.go
package main

import (
    "log"
    "pb-deployer/internal/handlers"
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/core"
)

func main() {
    app := pocketbase.New()
    
    // Register API handlers
    handlers.RegisterHandlers(app)
    
    // Start the application
    if err := app.Start(); err != nil {
        log.Fatal(err)
    }
}
```

### Frontend Integration

```javascript
// Example client usage
class PBDeployerClient {
    constructor(baseURL) {
        this.baseURL = baseURL;
    }
    
    // Test server connection
    async testConnection(serverId) {
        const response = await fetch(`${this.baseURL}/api/servers/${serverId}/test`, {
            method: 'POST'
        });
        return response.json();
    }
    
    // Deploy app with progress monitoring
    async deployApp(appId, versionId, credentials = {}) {
        // Start deployment
        const deployResponse = await fetch(`${this.baseURL}/api/apps/${appId}/deploy`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                version_id: versionId,
                ...credentials
            })
        });
        
        const deployment = await deployResponse.json();
        
        // Monitor progress via WebSocket
        const ws = new WebSocket(`ws://localhost:8090/api/realtime`);
        ws.onopen = () => {
            ws.send(JSON.stringify({
                subscriptions: [`deployment_progress_${deployment.deployment_id}`]
            }));
        };
        
        return new Promise((resolve, reject) => {
            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                if (data.name === `deployment_progress_${deployment.deployment_id}`) {
                    const progress = JSON.parse(data.data);
                    
                    if (progress.status === 'success') {
                        resolve(progress);
                        ws.close();
                    } else if (progress.status === 'failed') {
                        reject(new Error(progress.message));
                        ws.close();
                    }
                    
                    // Emit progress event
                    this.onProgress?.(progress);
                }
            };
        });
    }
}
```

## Security Considerations

Since this is a **local tool only**, authentication and authorization have been intentionally omitted. However, consider these security practices:

### Local Security

- **File Permissions**: Ensure deployment files have appropriate permissions
- **SSH Key Management**: Secure storage and handling of SSH keys
- **Input Validation**: Always validate and sanitize user inputs
- **Resource Limits**: Implement reasonable file size and request limits

### Network Security

- **Local Binding**: Bind to localhost only (`127.0.0.1`) for local-only access
- **Firewall Rules**: Configure firewall to block external access if needed
- **SSH Security**: Use secure SSH configurations and key-based authentication

### Data Security

- **Sensitive Data**: Never log passwords or SSH keys
- **Temporary Files**: Clean up temporary deployment files after use
- **Database**: Secure the PocketBase database file

## Conclusion

This handler implementation provides a complete REST API for the pb-deployer system with:

✅ **Full CRUD operations** for servers, apps, versions, and deployments  
✅ **Real-time progress updates** via PocketBase WebSocket integration  
✅ **Comprehensive error handling** with consistent response formats  
✅ **SSH connection management** with pooling and health monitoring  
✅ **File upload/download** with validation and size limits  
✅ **Background processing** for long-running operations  
✅ **Troubleshooting tools** for connection diagnostics  
✅ **Performance optimizations** with connection reuse and batch operations  

The handlers integrate seamlessly with PocketBase's core functionality while providing the specialized deployment management features needed for the pb-deployer system.