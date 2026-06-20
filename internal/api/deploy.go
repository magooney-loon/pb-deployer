package api

// API_SOURCE

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/tunnel"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Deploy a PocketBase app version to a server
// API_TAGS Deployment
func handleDeploy(c *core.RequestEvent) error {
	app := c.App
	log := logger.GetAPILogger()
	log.Info("Starting deployment process")

	type deployRequest struct {
		AppID          string `json:"app_id"`
		VersionID      string `json:"version_id"`
		DeploymentID   string `json:"deployment_id"`
		SuperuserEmail string `json:"superuser_email,omitempty"`
		SuperuserPass  string `json:"superuser_pass,omitempty"`
	}

	var req deployRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Info("Received deployment request: app_id=%s, version_id=%s, deployment_id=%s", req.AppID, req.VersionID, req.DeploymentID)

	// Validate required fields
	if req.AppID == "" || req.VersionID == "" || req.DeploymentID == "" {
		log.Error("Validation failed: Missing required fields")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "app_id, version_id, and deployment_id are required",
		})
	}

	// Get deployment record
	deploymentRecord, err := app.FindRecordById("deployments", req.DeploymentID)
	if err != nil {
		log.Error("Failed to find deployment record: %v", err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Deployment not found",
		})
	}

	// Get app record
	appRecord, err := app.FindRecordById("apps", req.AppID)
	if err != nil {
		log.Error("Failed to find app record: %v", err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "App not found",
		})
	}

	// Get version record
	versionRecord, err := app.FindRecordById("versions", req.VersionID)
	if err != nil {
		log.Error("Failed to find version record: %v", err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Version not found",
		})
	}

	// Get server record
	serverID := appRecord.GetString("server_id")
	serverRecord, err := app.FindRecordById("servers", serverID)
	if err != nil {
		log.Error("Failed to find server record: %v", err)
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Server not found",
		})
	}

	// Check if server is ready for deployment
	if !serverRecord.GetBool("setup_complete") {
		log.Error("Server not ready for deployment: setup_complete=%v",
			serverRecord.GetBool("setup_complete"))
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Server is not ready for deployment. Please complete server setup first.",
		})
	}

	// Reject apps that need proxy migration
	httpPort := appRecord.GetInt("http_port")
	if httpPort == 0 {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "App has no http_port assigned. Delete and recreate the app.",
		})
	}

	// Warn if server is not security locked but allow deployment
	if !serverRecord.GetBool("security_locked") {
		warningMsg := "⚠️  SECURITY WARNING: Deploying to non-secured server! Server has setup_complete=true but security_locked=false. Consider completing security configuration for production use."
		log.Warning(warningMsg)
		// Also add to deployment logs for visibility
		appendDeploymentLog(app, deploymentRecord, warningMsg)
	}

	// Check if version has deployment zip
	if versionRecord.GetString("deployment_zip") == "" {
		log.Error("Version has no deployment package")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Version has no deployment package",
		})
	}

	// Determine if this is an initial deployment based on presence of superuser credentials
	isInitialDeploy := req.SuperuserEmail != "" && req.SuperuserPass != ""

	// Build deployment ZIP URL
	zipURL := fmt.Sprintf("%s/api/files/versions/%s/%s",
		getBaseURL(c.Request), req.VersionID, versionRecord.GetString("deployment_zip"))

	// Update deployment status to running
	now := time.Now()
	deploymentRecord.Set("status", "running")
	deploymentRecord.Set("started_at", now)
	deploymentRecord.Set("logs", "Starting deployment...\n")

	if err := app.Save(deploymentRecord); err != nil {
		log.Error("Failed to update deployment status: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Failed to update deployment status",
		})
	}

	// Start deployment in goroutine
	go func() {
		// A panic here (e.g. an unexpected nil deref in the deploy steps) must not
		// crash the whole process — contain it and mark the deployment failed.
		defer func() {
			if r := recover(); r != nil {
				log.Error("Deployment panicked: %v", r)
				updateDeploymentStatus(app, deploymentRecord, "failed", fmt.Sprintf("Deployment panicked: %v", r))
			}
		}()

		err := performDeployment(app, &deploymentDeploymentContext{
			AppRecord:        appRecord,
			VersionRecord:    versionRecord,
			DeploymentRecord: deploymentRecord,
			ServerRecord:     serverRecord,
			ZipURL:           zipURL,
			IsInitialDeploy:  isInitialDeploy,
			SuperuserEmail:   req.SuperuserEmail,
			SuperuserPass:    req.SuperuserPass,
		})

		if err != nil {
			// Status was already marked failed by performDeployment (pre-Deploy
			// errors) or the deployment manager (in-Deploy errors); just log here.
			log.Error("Deployment failed: %v", err)
		}
	}()

	log.Success("Deployment started successfully")
	return c.JSON(http.StatusOK, map[string]any{
		"success":       true,
		"message":       "Deployment started",
		"deployment_id": req.DeploymentID,
	})
}

type deploymentDeploymentContext struct {
	AppRecord        *core.Record
	VersionRecord    *core.Record
	DeploymentRecord *core.Record
	ServerRecord     *core.Record
	ZipURL           string
	IsInitialDeploy  bool
	SuperuserEmail   string
	SuperuserPass    string
}

func performDeployment(app core.App, ctx *deploymentDeploymentContext) error {
	log := logger.GetAPILogger()

	// Create SSH client. Failures here happen *before* the deployment manager
	// runs, so the manager won't mark the deployment failed — the API must.
	client, err := createSSHClient(
		ctx.ServerRecord.GetString("host"),
		ctx.ServerRecord.GetInt("port"),
		ctx.ServerRecord.GetString("root_username"),
	)
	if err != nil {
		updateDeploymentStatus(app, ctx.DeploymentRecord, "failed", fmt.Sprintf("Failed to create SSH client: %v", err))
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	cleanup := tunnel.NewCleanupManager()
	defer cleanup.Close()
	cleanup.AddCloser(client)

	if err := client.Connect(); err != nil {
		updateDeploymentStatus(app, ctx.DeploymentRecord, "failed", fmt.Sprintf("Failed to connect to server: %v", err))
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Create managers
	manager := tunnel.NewManager(client)
	cleanup.AddCloser(manager)

	deploymentManager := tunnel.NewDeploymentManager(manager, app)
	cleanup.AddCloser(deploymentManager)

	// Build deployment request. From here on the deployment manager is the sole
	// owner of the deployment record (status + logs) and the app record (status +
	// current_version); the API must not write those concurrently or it will
	// clobber the manager's saves with a stale in-memory copy.
	deployReq := &tunnel.DeploymentRequest{
		AppName:              ctx.AppRecord.GetString("name"),
		AppID:                ctx.AppRecord.Id,
		VersionID:            ctx.VersionRecord.Id,
		VersionNum:           ctx.VersionRecord.GetString("version_num"),
		DeploymentID:         ctx.DeploymentRecord.Id,
		Domain:               ctx.AppRecord.GetString("domain"),
		ServiceName:          ctx.AppRecord.GetString("service_name"),
		RemotePath:           ctx.AppRecord.GetString("remote_path"),
		ZipDownloadURL:       ctx.ZipURL,
		HTTPPort:             ctx.AppRecord.GetInt("http_port"),
		IsInitialDeploy:      ctx.IsInitialDeploy,
		SuperuserEmail:       ctx.SuperuserEmail,
		SuperuserPass:        ctx.SuperuserPass,
		AppUsername:          ctx.ServerRecord.GetString("app_username"),
		ServerSecurityLocked: ctx.ServerRecord.GetBool("security_locked"),
		ProgressCallback: func(step int, total int, message string) {
			log.Step(step, total, message)
		},
	}

	// Perform deployment. Deploy() sets the terminal deployment status and the
	// app record itself, so we just propagate the error for the caller to log.
	if err := deploymentManager.Deploy(context.Background(), deployReq); err != nil {
		return err
	}

	log.Success("Deployment completed successfully")
	return nil
}

func updateDeploymentStatus(app core.App, deploymentRecord *core.Record, status string, message string) {
	log := logger.GetAPILogger()

	// Reload to avoid clobbering logs the deployment manager may have written to
	// this record on the server side with our stale in-memory copy.
	if fresh, err := app.FindRecordById("deployments", deploymentRecord.Id); err == nil {
		deploymentRecord = fresh
	}

	deploymentRecord.Set("status", status)

	if status == "success" || status == "failed" {
		now := time.Now()
		deploymentRecord.Set("completed_at", now)
	}

	if message != "" {
		appendDeploymentLog(app, deploymentRecord, message)
	}

	if err := app.Save(deploymentRecord); err != nil {
		log.Error("Failed to update deployment status: %v", err)
	}
}

func appendDeploymentLog(app core.App, deploymentRecord *core.Record, message string) {
	currentLogs := deploymentRecord.GetString("logs")
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newLog := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Limit log size to prevent database bloat
	maxLogSize := 50000 // 50KB
	updatedLogs := currentLogs + newLog
	if len(updatedLogs) > maxLogSize {
		// Trim from the beginning, keeping the most recent logs
		trimmed := updatedLogs[len(updatedLogs)-maxLogSize:]
		// Find the first newline to avoid cutting mid-line
		if idx := strings.Index(trimmed, "\n"); idx > 0 {
			updatedLogs = trimmed[idx+1:]
		} else {
			updatedLogs = trimmed
		}
	}

	deploymentRecord.Set("logs", updatedLogs)
}

func getBaseURL(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil || req.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, req.Host)
}
