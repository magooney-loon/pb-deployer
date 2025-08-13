package deployment

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pb-deployer/internal/utils"

	"github.com/pocketbase/pocketbase/core"
)

// DeploymentResponse represents a deployment response
type DeploymentResponse struct {
	ID          string    `json:"id"`
	AppID       string    `json:"app_id"`
	AppName     string    `json:"app_name,omitempty"`
	VersionID   string    `json:"version_id"`
	Version     string    `json:"version,omitempty"`
	Status      string    `json:"status"`
	Logs        string    `json:"logs"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
}

// DeploymentStatusResponse represents a deployment status response
type DeploymentStatusResponse struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"`
	Progress    int       `json:"progress,omitempty"`
	Message     string    `json:"message,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Duration    string    `json:"duration,omitempty"`
	IsRunning   bool      `json:"is_running"`
	CanCancel   bool      `json:"can_cancel"`
	CanRetry    bool      `json:"can_retry"`
}

// DeploymentStatsResponse represents deployment statistics
type DeploymentStatsResponse struct {
	Total       int64                `json:"total"`
	Pending     int64                `json:"pending"`
	Running     int64                `json:"running"`
	Success     int64                `json:"success"`
	Failed      int64                `json:"failed"`
	SuccessRate float64              `json:"success_rate"`
	AvgDuration string               `json:"avg_duration"`
	Recent      []DeploymentResponse `json:"recent"`
	ByApp       map[string]any       `json:"by_app"`
	ByStatus    map[string]int64     `json:"by_status"`
}

// listDeployments handles the list deployments endpoint
func listDeployments(app core.App, e *core.RequestEvent) error {
	// Get optional filters
	appID := e.Request.URL.Query().Get("app_id")
	status := e.Request.URL.Query().Get("status")
	limit := 50 // Default limit

	if limitStr := e.Request.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Build filter conditions
	var filter string
	var params map[string]any

	if appID != "" && status != "" {
		filter = "app_id = {:app_id} && status = {:status}"
		params = map[string]any{
			"app_id": appID,
			"status": status,
		}
	} else if appID != "" {
		filter = "app_id = {:app_id}"
		params = map[string]any{
			"app_id": appID,
		}
	} else if status != "" {
		filter = "status = {:status}"
		params = map[string]any{
			"status": status,
		}
	}

	// Get deployments
	records, err := app.FindRecordsByFilter("deployments", filter, "-created", limit, 0, params)
	if err != nil {
		app.Logger().Error("Failed to fetch deployments", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch deployments",
		})
	}

	// Convert records to response format
	deployments := make([]DeploymentResponse, len(records))
	for i, record := range records {
		deployments[i] = recordToDeploymentResponse(record, app)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"deployments": deployments,
		"count":       len(deployments),
		"filters": map[string]any{
			"app_id": appID,
			"status": status,
			"limit":  limit,
		},
	})
}

// getDeployment handles the get single deployment endpoint
func getDeployment(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// Get deployment record
	record, err := app.FindRecordById("deployments", deploymentID)
	if err != nil {
		app.Logger().Error("Failed to find deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Deployment not found",
		})
	}

	response := recordToDeploymentResponse(record, app)

	return e.JSON(http.StatusOK, response)
}

// getDeploymentStatus handles the get deployment status endpoint
func getDeploymentStatus(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// Get deployment record
	record, err := app.FindRecordById("deployments", deploymentID)
	if err != nil {
		app.Logger().Error("Failed to find deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Deployment not found",
		})
	}

	status := record.GetString("status")
	startedAt := record.GetDateTime("started_at").Time()
	completedAt := record.GetDateTime("completed_at").Time()

	response := DeploymentStatusResponse{
		ID:        deploymentID,
		Status:    status,
		StartedAt: startedAt,
		IsRunning: status == "running" || status == "pending",
		CanCancel: status == "running" || status == "pending",
		CanRetry:  status == "failed",
	}

	// Calculate duration
	if !completedAt.IsZero() {
		response.CompletedAt = completedAt
		duration := completedAt.Sub(startedAt)
		response.Duration = formatDuration(duration)
	} else if response.IsRunning {
		duration := time.Since(startedAt)
		response.Duration = formatDuration(duration)
	}

	// Set progress based on status
	switch status {
	case "pending":
		response.Progress = 0
		response.Message = "Deployment queued"
	case "running":
		response.Progress = 50
		response.Message = "Deployment in progress"
	case "success":
		response.Progress = 100
		response.Message = "Deployment completed successfully"
	case "failed":
		response.Progress = 0
		response.Message = "Deployment failed"
	}

	return e.JSON(http.StatusOK, response)
}

// getDeploymentLogs handles the get deployment logs endpoint
func getDeploymentLogs(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// Get deployment record
	record, err := app.FindRecordById("deployments", deploymentID)
	if err != nil {
		app.Logger().Error("Failed to find deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Deployment not found",
		})
	}

	logs := record.GetString("logs")
	status := record.GetString("status")

	return e.JSON(http.StatusOK, map[string]any{
		"deployment_id": deploymentID,
		"status":        status,
		"logs":          logs,
		"retrieved_at":  time.Now().UTC(),
	})
}

// cancelDeployment handles canceling a running deployment
func cancelDeployment(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// Get deployment record
	record, err := app.FindRecordById("deployments", deploymentID)
	if err != nil {
		app.Logger().Error("Failed to find deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Deployment not found",
		})
	}

	status := record.GetString("status")

	// Check if deployment can be canceled
	if status != "running" && status != "pending" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment cannot be canceled in current status: " + status,
		})
	}

	// Update deployment status to failed with cancellation note
	now := time.Now()
	record.Set("status", "failed")
	record.Set("completed_at", now)

	currentLogs := record.GetString("logs")
	record.Set("logs", currentLogs+"\nDeployment canceled by user at "+now.Format(time.RFC3339))

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to cancel deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to cancel deployment",
		})
	}

	app.Logger().Info("Deployment canceled", "deployment_id", deploymentID)

	return e.JSON(http.StatusOK, map[string]any{
		"message":       "Deployment canceled successfully",
		"deployment_id": deploymentID,
		"status":        "failed",
		"canceled_at":   now.UTC(),
	})
}

// retryDeployment handles retrying a failed deployment
func retryDeployment(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// Get deployment record
	record, err := app.FindRecordById("deployments", deploymentID)
	if err != nil {
		app.Logger().Error("Failed to find deployment", "id", deploymentID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Deployment not found",
		})
	}

	status := record.GetString("status")

	// Check if deployment can be retried
	if status != "failed" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Only failed deployments can be retried",
		})
	}

	// Create new deployment record for retry
	appID := record.GetString("app_id")
	versionID := record.GetString("version_id")

	collection, err := app.FindCollectionByNameOrId("deployments")
	if err != nil {
		app.Logger().Error("Failed to find deployments collection", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	newRecord := core.NewRecord(collection)
	newRecord.Set("app_id", appID)
	newRecord.Set("version_id", versionID)
	newRecord.Set("status", "pending")
	newRecord.Set("logs", "Retry of deployment "+deploymentID)
	now := time.Now()
	newRecord.Set("started_at", now)

	if err := app.Save(newRecord); err != nil {
		app.Logger().Error("Failed to create retry deployment", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create retry deployment",
		})
	}

	app.Logger().Info("Deployment retry created",
		"original_deployment_id", deploymentID,
		"new_deployment_id", newRecord.Id)

	// TODO: Trigger actual deployment process here

	return e.JSON(http.StatusCreated, map[string]any{
		"message":             "Deployment retry created",
		"original_deployment": deploymentID,
		"new_deployment_id":   newRecord.Id,
		"status":              "pending",
	})
}

// listAppDeployments handles listing deployments for a specific app
func listAppDeployments(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("app_id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Verify app exists
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get optional status filter
	status := e.Request.URL.Query().Get("status")
	limit := 20 // Default limit for app-specific queries

	if limitStr := e.Request.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	var filter string
	var params map[string]any

	if status != "" {
		filter = "app_id = {:app_id} && status = {:status}"
		params = map[string]any{
			"app_id": appID,
			"status": status,
		}
	} else {
		filter = "app_id = {:app_id}"
		params = map[string]any{
			"app_id": appID,
		}
	}

	// Get deployments for this app
	records, err := app.FindRecordsByFilter("deployments", filter, "-created", limit, 0, params)
	if err != nil {
		app.Logger().Error("Failed to fetch app deployments", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch deployments",
		})
	}

	// Convert records to response format
	deployments := make([]DeploymentResponse, len(records))
	for i, record := range records {
		deployments[i] = recordToDeploymentResponse(record, app)
		deployments[i].AppName = appRecord.GetString("name")
	}

	return e.JSON(http.StatusOK, map[string]any{
		"app_id":      appID,
		"app_name":    appRecord.GetString("name"),
		"deployments": deployments,
		"count":       len(deployments),
		"status":      status,
	})
}

// getLatestAppDeployment handles getting the latest deployment for an app
func getLatestAppDeployment(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("app_id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Verify app exists
	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get latest deployment for this app
	records, err := app.FindRecordsByFilter("deployments", "app_id = {:app_id}", "-created", 1, 0, map[string]any{
		"app_id": appID,
	})
	if err != nil || len(records) == 0 {
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "No deployments found for this app",
		})
	}

	record := records[0]

	response := recordToDeploymentResponse(record, app)
	response.AppName = appRecord.GetString("name")

	return e.JSON(http.StatusOK, response)
}

// handleDeploymentProgressWebSocket handles WebSocket connections for deployment progress
func handleDeploymentProgressWebSocket(app core.App, e *core.RequestEvent) error {
	deploymentID := e.Request.PathValue("id")
	if deploymentID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Deployment ID is required",
		})
	}

	// For now, return a simple response since we're using PocketBase realtime
	// The actual WebSocket connection is handled by PocketBase's realtime system
	return e.JSON(http.StatusOK, map[string]any{
		"message":      "Deployment progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("deployment_progress_%s", deploymentID),
	})
}

// getDeploymentStats handles getting deployment statistics
func getDeploymentStats(app core.App, e *core.RequestEvent) error {
	// Get time range filter
	days := 30 // Default to last 30 days
	if daysStr := e.Request.URL.Query().Get("days"); daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}

	since := time.Now().AddDate(0, 0, -days)

	// Get all deployments in time range
	records, err := app.FindRecordsByFilter("deployments", "created >= {:since}", "-created", 0, 0, map[string]any{
		"since": since,
	})
	if err != nil {
		app.Logger().Error("Failed to fetch deployment stats", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch deployment statistics",
		})
	}

	// Calculate statistics
	stats := DeploymentStatsResponse{
		Total:    int64(len(records)),
		ByStatus: make(map[string]int64),
		ByApp:    make(map[string]any),
	}

	var totalDuration time.Duration
	var completedCount int
	appStats := make(map[string]map[string]int)

	for _, record := range records {
		status := record.GetString("status")
		appID := record.GetString("app_id")

		// Count by status
		stats.ByStatus[status]++

		switch status {
		case "pending":
			stats.Pending++
		case "running":
			stats.Running++
		case "success":
			stats.Success++
		case "failed":
			stats.Failed++
		}

		// Calculate duration for completed deployments
		startedAt := record.GetDateTime("started_at").Time()
		completedAt := record.GetDateTime("completed_at").Time()
		if !completedAt.IsZero() {
			duration := completedAt.Sub(startedAt)
			totalDuration += duration
			completedCount++
		}

		// Count by app
		if appStats[appID] == nil {
			appStats[appID] = make(map[string]int)
		}
		appStats[appID][status]++
		appStats[appID]["total"]++
	}

	// Calculate success rate
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Success) / float64(stats.Total) * 100
	}

	// Calculate average duration
	if completedCount > 0 {
		avgDuration := totalDuration / time.Duration(completedCount)
		stats.AvgDuration = formatDuration(avgDuration)
	}

	// Get recent deployments (last 10)
	recentLimit := 10
	if len(records) < recentLimit {
		recentLimit = len(records)
	}
	stats.Recent = make([]DeploymentResponse, recentLimit)
	for i := 0; i < recentLimit; i++ {
		stats.Recent[i] = recordToDeploymentResponse(records[i], app)
	}

	// Convert app stats
	for appID, appStat := range appStats {
		if appRecord, err := app.FindRecordById("apps", appID); err == nil {
			stats.ByApp[appRecord.GetString("name")] = appStat
		}
	}

	return e.JSON(http.StatusOK, stats)
}

// cleanupOldDeployments handles cleaning up old deployment records
func cleanupOldDeployments(app core.App, e *core.RequestEvent) error {
	// Get cleanup parameters
	keepDays := 90 // Default to keep 90 days
	if keepDaysStr := e.Request.URL.Query().Get("keep_days"); keepDaysStr != "" {
		if parsed, err := strconv.Atoi(keepDaysStr); err == nil && parsed > 0 {
			keepDays = parsed
		}
	}

	dryRun := e.Request.URL.Query().Get("dry_run") == "true"

	cutoffDate := time.Now().AddDate(0, 0, -keepDays)

	// Find old deployments
	oldRecords, err := app.FindRecordsByFilter("deployments", "created < {:cutoff}", "-created", 0, 0, map[string]any{
		"cutoff": cutoffDate,
	})
	if err != nil {
		app.Logger().Error("Failed to fetch old deployments", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch old deployments",
		})
	}

	result := map[string]any{
		"found":     len(oldRecords),
		"kept_days": keepDays,
		"cutoff":    cutoffDate,
		"dry_run":   dryRun,
	}

	if !dryRun && len(oldRecords) > 0 {
		// Actually delete the records
		deleted := 0
		for _, record := range oldRecords {
			if err := app.Delete(record); err != nil {
				app.Logger().Error("Failed to delete old deployment", "id", record.Id, "error", err)
			} else {
				deleted++
			}
		}
		result["deleted"] = deleted

		app.Logger().Info("Cleaned up old deployments",
			"total_found", len(oldRecords),
			"deleted", deleted,
			"kept_days", keepDays)
	}

	return e.JSON(http.StatusOK, result)
}

// Helper functions

// recordToDeploymentResponse converts a deployment record to response format
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

	// Set started at
	if startedAt := record.GetDateTime("started_at"); !startedAt.Time().IsZero() {
		response.StartedAt = startedAt.Time()
	}

	// Set completed at and duration
	if completedAt := record.GetDateTime("completed_at"); !completedAt.Time().IsZero() {
		response.CompletedAt = completedAt.Time()
		if !response.StartedAt.IsZero() {
			duration := response.CompletedAt.Sub(response.StartedAt)
			response.Duration = formatDuration(duration)
		}
	}

	// Get version info if available
	if versionRecord, err := app.FindRecordById("versions", response.VersionID); err == nil {
		response.Version = versionRecord.GetString("version_number")
	}

	// Get app info if available
	if appRecord, err := app.FindRecordById("apps", response.AppID); err == nil {
		response.AppName = appRecord.GetString("name")
	}

	return response
}

// formatDuration formats a duration in a human-readable way
// formatDuration is now available from utils package
// Keeping this as a wrapper for backward compatibility
func formatDuration(d time.Duration) string {
	return utils.FormatDuration(d)
}
