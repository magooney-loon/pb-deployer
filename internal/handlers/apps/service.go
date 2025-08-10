package apps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

// ServiceActionRequest represents a request for service actions
type ServiceActionRequest struct {
	Force bool `json:"force,omitempty"` // Force action even if status check fails
}

// ServiceActionResponse represents the response for service actions
type ServiceActionResponse struct {
	AppID       string    `json:"app_id"`
	ServiceName string    `json:"service_name"`
	Action      string    `json:"action"`
	Success     bool      `json:"success"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DeploymentRequest represents a deployment request
type DeploymentRequest struct {
	VersionID      string `json:"version_id"`
	SuperuserEmail string `json:"superuser_email,omitempty"`    // For first deploy only
	SuperuserPass  string `json:"superuser_password,omitempty"` // For first deploy only
	Notes          string `json:"notes,omitempty"`
}

// RollbackRequest represents a rollback request
type RollbackRequest struct {
	VersionID string `json:"version_id"`
	Notes     string `json:"notes,omitempty"`
}

// startAppService handles starting an app service
func startAppService(app core.App, e *core.RequestEvent) error {
	return handleServiceAction(app, e, "start")
}

// stopAppService handles stopping an app service
func stopAppService(app core.App, e *core.RequestEvent) error {
	return handleServiceAction(app, e, "stop")
}

// restartAppService handles restarting an app service
func restartAppService(app core.App, e *core.RequestEvent) error {
	return handleServiceAction(app, e, "restart")
}

// handleServiceAction handles generic service actions (start/stop/restart)
func handleServiceAction(app core.App, e *core.RequestEvent, action string) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	var req ServiceActionRequest
	if e.Request.Body != nil {
		json.NewDecoder(e.Request.Body).Decode(&req)
	}

	// Get app record
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get server record
	serverRecord, err := app.FindRecordById("servers", appRecord.GetString("server_id"))
	if err != nil {
		app.Logger().Error("Failed to find server for app", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load server information",
		})
	}

	// Convert to models.Server
	server := &models.Server{
		ID:             serverRecord.Id,
		Name:           serverRecord.GetString("name"),
		Host:           serverRecord.GetString("host"),
		Port:           serverRecord.GetInt("port"),
		RootUsername:   serverRecord.GetString("root_username"),
		AppUsername:    serverRecord.GetString("app_username"),
		UseSSHAgent:    serverRecord.GetBool("use_ssh_agent"),
		ManualKeyPath:  serverRecord.GetString("manual_key_path"),
		SetupComplete:  serverRecord.GetBool("setup_complete"),
		SecurityLocked: serverRecord.GetBool("security_locked"),
	}

	serviceName := appRecord.GetString("service_name")
	if serviceName == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App service name not configured",
		})
	}

	// Check if server is ready
	if !server.SetupComplete {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server setup is not complete",
		})
	}

	app.Logger().Info("Starting service action",
		"app_id", appID,
		"action", action,
		"service", serviceName,
		"server", server.Host)

	// Create SSH connection manager
	var sshManager *ssh.SSHManager
	if server.SecurityLocked {
		// Use app user for security-locked servers
		sshManager, err = ssh.NewSSHManager(server, false)
	} else {
		// Use root for non-security-locked servers
		sshManager, err = ssh.NewSSHManager(server, true)
	}

	if err != nil {
		app.Logger().Error("Failed to create SSH manager", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, ServiceActionResponse{
			AppID:       appID,
			ServiceName: serviceName,
			Action:      action,
			Success:     false,
			Message:     "Failed to connect to server",
			Error:       err.Error(),
			Timestamp:   time.Now().UTC(),
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
	default:
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid action",
		})
	}

	response := ServiceActionResponse{
		AppID:       appID,
		ServiceName: serviceName,
		Action:      action,
		Success:     actionErr == nil,
		Timestamp:   time.Now().UTC(),
	}

	if actionErr != nil {
		response.Error = actionErr.Error()
		response.Message = fmt.Sprintf("Failed to %s service", action)
		app.Logger().Error("Service action failed",
			"app_id", appID,
			"action", action,
			"service", serviceName,
			"error", actionErr)
	} else {
		response.Message = fmt.Sprintf("Service %s successful", action)
		app.Logger().Info("Service action completed",
			"app_id", appID,
			"action", action,
			"service", serviceName)
	}

	// Get service status after action
	status, statusErr := sshManager.GetServiceStatus(serviceName)
	if statusErr == nil {
		response.Status = status
	}

	// Update app status in database if start/restart
	if actionErr == nil && (action == "start" || action == "restart") {
		// Wait a moment for service to fully start
		time.Sleep(2 * time.Second)

		// Try to update status to online, but don't fail if it doesn't work
		appRecord.Set("status", "online")
		if saveErr := app.Save(appRecord); saveErr != nil {
			app.Logger().Debug("Failed to update app status", "app_id", appID, "error", saveErr)
		}
	} else if actionErr == nil && action == "stop" {
		// Update status to offline
		appRecord.Set("status", "offline")
		if saveErr := app.Save(appRecord); saveErr != nil {
			app.Logger().Debug("Failed to update app status", "app_id", appID, "error", saveErr)
		}
	}

	statusCode := http.StatusOK
	if actionErr != nil {
		statusCode = http.StatusInternalServerError
	}

	return e.JSON(statusCode, response)
}

// deployApp handles app deployment
func deployApp(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	var req DeploymentRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.VersionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get app record
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get version record
	versionRecord, err := app.FindRecordById("versions", req.VersionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", req.VersionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Verify version belongs to this app
	if versionRecord.GetString("app_id") != appID {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version does not belong to this app",
		})
	}

	// Check if this is the first deployment (no current version)
	isFirstDeploy := appRecord.GetString("current_version") == ""

	if isFirstDeploy {
		// First deploy requires superuser credentials
		if req.SuperuserEmail == "" || req.SuperuserPass == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "First deployment requires superuser email and password",
			})
		}
	}

	// Create deployment record
	deploymentCollection, err := app.FindCollectionByNameOrId("deployments")
	if err != nil {
		app.Logger().Error("Failed to find deployments collection", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	deploymentRecord := core.NewRecord(deploymentCollection)
	deploymentRecord.Set("app_id", appID)
	deploymentRecord.Set("version_id", req.VersionID)
	deploymentRecord.Set("status", "pending")
	deploymentRecord.Set("logs", "Deployment initiated")
	now := time.Now()
	deploymentRecord.Set("started_at", now)

	if err := app.Save(deploymentRecord); err != nil {
		app.Logger().Error("Failed to create deployment record", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create deployment record",
		})
	}

	app.Logger().Info("Deployment started",
		"app_id", appID,
		"version_id", req.VersionID,
		"deployment_id", deploymentRecord.Id,
		"is_first_deploy", isFirstDeploy)

	// Start deployment in background
	go func() {
		deploymentErr := performDeployment(app, deploymentRecord.Id, appRecord, versionRecord, req, isFirstDeploy)

		// Update deployment record with result
		deploymentRecord, err := app.FindRecordById("deployments", deploymentRecord.Id)
		if err != nil {
			app.Logger().Error("Failed to find deployment record for update", "deployment_id", deploymentRecord.Id, "error", err)
			return
		}

		now := time.Now()
		deploymentRecord.Set("completed_at", now)

		if deploymentErr != nil {
			deploymentRecord.Set("status", "failed")
			deploymentRecord.Set("logs", deploymentRecord.GetString("logs")+"\nDeployment failed: "+deploymentErr.Error())
			app.Logger().Error("Deployment failed",
				"app_id", appID,
				"deployment_id", deploymentRecord.Id,
				"error", deploymentErr)
		} else {
			deploymentRecord.Set("status", "success")
			deploymentRecord.Set("logs", deploymentRecord.GetString("logs")+"\nDeployment completed successfully")

			// Update app's current version
			appRecord.Set("current_version", versionRecord.GetString("version_number"))
			appRecord.Set("status", "online")
			if saveErr := app.Save(appRecord); saveErr != nil {
				app.Logger().Error("Failed to update app version", "app_id", appID, "error", saveErr)
			}

			app.Logger().Info("Deployment completed successfully",
				"app_id", appID,
				"deployment_id", deploymentRecord.Id,
				"version", versionRecord.GetString("version_number"))
		}

		if saveErr := app.Save(deploymentRecord); saveErr != nil {
			app.Logger().Error("Failed to update deployment record", "deployment_id", deploymentRecord.Id, "error", saveErr)
		}
	}()

	return e.JSON(http.StatusAccepted, map[string]interface{}{
		"message":         "Deployment started",
		"deployment_id":   deploymentRecord.Id,
		"app_id":          appID,
		"version_id":      req.VersionID,
		"is_first_deploy": isFirstDeploy,
	})
}

// rollbackApp handles app rollback
func rollbackApp(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	var req RollbackRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.VersionID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version ID is required",
		})
	}

	// Get app record
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get version record
	versionRecord, err := app.FindRecordById("versions", req.VersionID)
	if err != nil {
		app.Logger().Error("Failed to find version", "id", req.VersionID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Version not found",
		})
	}

	// Verify version belongs to this app
	if versionRecord.GetString("app_id") != appID {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Version does not belong to this app",
		})
	}

	// Create deployment record for rollback
	deploymentCollection, err := app.FindCollectionByNameOrId("deployments")
	if err != nil {
		app.Logger().Error("Failed to find deployments collection", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	deploymentRecord := core.NewRecord(deploymentCollection)
	deploymentRecord.Set("app_id", appID)
	deploymentRecord.Set("version_id", req.VersionID)
	deploymentRecord.Set("status", "pending")
	deploymentRecord.Set("logs", "Rollback initiated")
	now := time.Now()
	deploymentRecord.Set("started_at", now)

	if err := app.Save(deploymentRecord); err != nil {
		app.Logger().Error("Failed to create rollback deployment record", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create rollback record",
		})
	}

	app.Logger().Info("Rollback started",
		"app_id", appID,
		"version_id", req.VersionID,
		"deployment_id", deploymentRecord.Id)

	// Start rollback in background (similar to deployment but different process)
	go func() {
		rollbackErr := performRollback(app, deploymentRecord.Id, appRecord, versionRecord, req)

		// Update deployment record with result
		deploymentRecord, err := app.FindRecordById("deployments", deploymentRecord.Id)
		if err != nil {
			app.Logger().Error("Failed to find deployment record for rollback update", "deployment_id", deploymentRecord.Id, "error", err)
			return
		}

		now := time.Now()
		deploymentRecord.Set("completed_at", now)

		if rollbackErr != nil {
			deploymentRecord.Set("status", "failed")
			deploymentRecord.Set("logs", deploymentRecord.GetString("logs")+"\nRollback failed: "+rollbackErr.Error())
			app.Logger().Error("Rollback failed",
				"app_id", appID,
				"deployment_id", deploymentRecord.Id,
				"error", rollbackErr)
		} else {
			deploymentRecord.Set("status", "success")
			deploymentRecord.Set("logs", deploymentRecord.GetString("logs")+"\nRollback completed successfully")

			// Update app's current version
			appRecord.Set("current_version", versionRecord.GetString("version_number"))
			appRecord.Set("status", "online")
			if saveErr := app.Save(appRecord); saveErr != nil {
				app.Logger().Error("Failed to update app version after rollback", "app_id", appID, "error", saveErr)
			}

			app.Logger().Info("Rollback completed successfully",
				"app_id", appID,
				"deployment_id", deploymentRecord.Id,
				"version", versionRecord.GetString("version_number"))
		}

		if saveErr := app.Save(deploymentRecord); saveErr != nil {
			app.Logger().Error("Failed to update rollback deployment record", "deployment_id", deploymentRecord.Id, "error", saveErr)
		}
	}()

	return e.JSON(http.StatusAccepted, map[string]interface{}{
		"message":       "Rollback started",
		"deployment_id": deploymentRecord.Id,
		"app_id":        appID,
		"version_id":    req.VersionID,
	})
}

// handleDeploymentWebSocket handles WebSocket connections for deployment progress
func handleDeploymentWebSocket(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// For now, return a simple response since we're using PocketBase realtime
	// The actual WebSocket connection is handled by PocketBase's realtime system
	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Deployment progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("app_deployment_%s", appID),
	})
}

// performDeployment performs the actual deployment process
func performDeployment(app core.App, deploymentID string, appRecord, versionRecord *core.Record, req DeploymentRequest, isFirstDeploy bool) error {
	// TODO: Implement actual deployment logic
	// This is a placeholder that will be implemented with:
	// 1. Download deployment zip from PocketBase storage
	// 2. Extract files locally
	// 3. rsync to remote server
	// 4. Create/update systemd service
	// 5. Setup superuser (if first deploy)
	// 6. Start service
	// 7. Health check

	app.Logger().Info("Deployment process started", "deployment_id", deploymentID)

	// Simulate deployment time
	time.Sleep(2 * time.Second)

	app.Logger().Info("Deployment process completed", "deployment_id", deploymentID)
	return nil
}

// performRollback performs the actual rollback process
func performRollback(app core.App, deploymentID string, appRecord, versionRecord *core.Record, req RollbackRequest) error {
	// TODO: Implement actual rollback logic
	// This is a placeholder that will be implemented with:
	// 1. Stop current service
	// 2. Download previous version zip from PocketBase storage
	// 3. Extract files locally
	// 4. rsync to remote server
	// 5. Start service
	// 6. Health check

	app.Logger().Info("Rollback process started", "deployment_id", deploymentID)

	// Simulate rollback time
	time.Sleep(2 * time.Second)

	app.Logger().Info("Rollback process completed", "deployment_id", deploymentID)
	return nil
}
