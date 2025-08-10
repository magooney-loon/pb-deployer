package apps

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// AppCreateRequest represents the request for creating a new app
type AppCreateRequest struct {
	Name       string `json:"name"`
	ServerID   string `json:"server_id"`
	Domain     string `json:"domain"`
	RemotePath string `json:"remote_path,omitempty"`
}

// AppUpdateRequest represents the request for updating an app
type AppUpdateRequest struct {
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

// AppStatusResponse represents the response for app status
type AppStatusResponse struct {
	AppID          string    `json:"app_id"`
	Name           string    `json:"app_name"`
	Status         string    `json:"status"`
	Domain         string    `json:"domain"`
	CurrentVersion string    `json:"current_version"`
	HealthURL      string    `json:"health_url"`
	LastChecked    time.Time `json:"last_checked"`
	ServerName     string    `json:"server_name"`
	ServiceName    string    `json:"service_name"`
	Error          string    `json:"error,omitempty"`
}

// listApps handles the list apps endpoint
func listApps(app core.App, e *core.RequestEvent) error {
	// Get optional server filter
	serverID := e.Request.URL.Query().Get("server_id")

	var records []*core.Record
	var err error

	if serverID != "" {
		// Filter by server ID
		records, err = app.FindRecordsByFilter("apps", "server_id = {:server_id}", "", 0, 0, map[string]any{
			"server_id": serverID,
		})
	} else {
		// Get all apps
		records, err = app.FindRecordsByFilter("apps", "", "-created", 0, 0, nil)
	}

	if err != nil {
		app.Logger().Error("Failed to fetch apps", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch apps",
		})
	}

	// Convert records to response format
	apps := make([]map[string]interface{}, len(records))
	for i, record := range records {
		apps[i] = map[string]interface{}{
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

	return e.JSON(http.StatusOK, map[string]interface{}{
		"apps":  apps,
		"count": len(apps),
	})
}

// createApp handles the create app endpoint
func createApp(app core.App, e *core.RequestEvent) error {
	var req AppCreateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App name is required",
		})
	}

	if req.ServerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	if req.Domain == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Domain is required",
		})
	}

	// Verify server exists
	serverRecord, err := app.FindRecordById("servers", req.ServerID)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid server ID",
		})
	}

	// Check if app name is unique for this server
	existing, err := app.FindFirstRecordByFilter("apps", "name = {:name} && server_id = {:server_id}", map[string]any{
		"name":      req.Name,
		"server_id": req.ServerID,
	})
	if err == nil && existing != nil {
		return e.JSON(http.StatusConflict, map[string]string{
			"error": "App with this name already exists on the server",
		})
	}

	// Generate default values
	serviceName := fmt.Sprintf("pocketbase-%s", req.Name)
	remotePath := req.RemotePath
	if remotePath == "" {
		remotePath = fmt.Sprintf("/opt/pocketbase/apps/%s", req.Name)
	}

	// Create new app record
	collection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		app.Logger().Error("Failed to find apps collection", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Internal server error",
		})
	}

	record := core.NewRecord(collection)
	record.Set("name", req.Name)
	record.Set("server_id", req.ServerID)
	record.Set("domain", req.Domain)
	record.Set("remote_path", remotePath)
	record.Set("service_name", serviceName)
	record.Set("status", "offline")

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to create app", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create app",
		})
	}

	app.Logger().Info("App created successfully",
		"app_id", record.Id,
		"name", req.Name,
		"server_id", req.ServerID,
		"domain", req.Domain)

	return e.JSON(http.StatusCreated, map[string]interface{}{
		"id":              record.Id,
		"name":            record.GetString("name"),
		"server_id":       record.GetString("server_id"),
		"server_name":     serverRecord.GetString("name"),
		"domain":          record.GetString("domain"),
		"remote_path":     record.GetString("remote_path"),
		"service_name":    record.GetString("service_name"),
		"current_version": record.GetString("current_version"),
		"status":          record.GetString("status"),
		"created":         record.GetDateTime("created"),
		"updated":         record.GetDateTime("updated"),
	})
}

// getApp handles the get single app endpoint
func getApp(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Get app record
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get server information
	serverRecord, err := app.FindRecordById("servers", record.GetString("server_id"))
	if err != nil {
		app.Logger().Error("Failed to find server for app", "app_id", appID, "server_id", record.GetString("server_id"), "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load server information",
		})
	}

	return e.JSON(http.StatusOK, map[string]interface{}{
		"id":              record.Id,
		"name":            record.GetString("name"),
		"server_id":       record.GetString("server_id"),
		"server_name":     serverRecord.GetString("name"),
		"server_host":     serverRecord.GetString("host"),
		"domain":          record.GetString("domain"),
		"remote_path":     record.GetString("remote_path"),
		"service_name":    record.GetString("service_name"),
		"current_version": record.GetString("current_version"),
		"status":          record.GetString("status"),
		"created":         record.GetDateTime("created"),
		"updated":         record.GetDateTime("updated"),
	})
}

// updateApp handles the update app endpoint
func updateApp(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	var req AppUpdateRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get existing app record
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Update fields if provided
	if req.Name != "" {
		// Check if new name is unique for this server
		existing, err := app.FindFirstRecordByFilter("apps", "name = {:name} && server_id = {:server_id} && id != {:id}", map[string]any{
			"name":      req.Name,
			"server_id": record.GetString("server_id"),
			"id":        appID,
		})
		if err == nil && existing != nil {
			return e.JSON(http.StatusConflict, map[string]string{
				"error": "App with this name already exists on the server",
			})
		}

		record.Set("name", req.Name)
		// Update service name if name changed
		record.Set("service_name", fmt.Sprintf("pocketbase-%s", req.Name))
	}

	if req.Domain != "" {
		record.Set("domain", req.Domain)
	}

	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to update app", "id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update app",
		})
	}

	app.Logger().Info("App updated successfully", "app_id", appID)

	return e.JSON(http.StatusOK, map[string]interface{}{
		"id":              record.Id,
		"name":            record.GetString("name"),
		"server_id":       record.GetString("server_id"),
		"domain":          record.GetString("domain"),
		"remote_path":     record.GetString("remote_path"),
		"service_name":    record.GetString("service_name"),
		"current_version": record.GetString("current_version"),
		"status":          record.GetString("status"),
		"updated":         record.GetDateTime("updated"),
	})
}

// deleteApp handles the delete app endpoint
func deleteApp(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Get app record to ensure it exists
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	appName := record.GetString("name")
	serverID := record.GetString("server_id")

	// Delete the app record
	if err := app.Delete(record); err != nil {
		app.Logger().Error("Failed to delete app", "id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete app",
		})
	}

	app.Logger().Info("App deleted successfully",
		"app_id", appID,
		"name", appName,
		"server_id", serverID)

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message": "App deleted successfully",
		"app_id":  appID,
	})
}

// getAppStatus handles the get app status endpoint
func getAppStatus(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Get app record
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	// Get server information
	serverRecord, err := app.FindRecordById("servers", record.GetString("server_id"))
	if err != nil {
		app.Logger().Error("Failed to find server for app", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load server information",
		})
	}

	domain := record.GetString("domain")
	healthURL := ""
	if domain != "" {
		healthURL = fmt.Sprintf("https://%s/api/health", domain)
	}

	response := AppStatusResponse{
		AppID:          appID,
		Name:           record.GetString("name"),
		Status:         record.GetString("status"),
		Domain:         domain,
		CurrentVersion: record.GetString("current_version"),
		HealthURL:      healthURL,
		LastChecked:    time.Now().UTC(),
		ServerName:     serverRecord.GetString("name"),
		ServiceName:    record.GetString("service_name"),
	}

	app.Logger().Info("App status retrieved",
		"app_id", appID,
		"status", response.Status)

	return e.JSON(http.StatusOK, response)
}

// checkAppHealth handles the app health check endpoint
func checkAppHealth(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Get app record
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	domain := record.GetString("domain")
	if domain == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App domain not configured",
		})
	}

	// Perform health check
	healthURL := fmt.Sprintf("https://%s/api/health", domain)
	isHealthy, err := performHealthCheck(healthURL)

	status := "offline"
	errorMsg := ""

	if err != nil {
		errorMsg = err.Error()
		app.Logger().Debug("Health check failed", "app_id", appID, "url", healthURL, "error", err)
	} else if isHealthy {
		status = "online"
		app.Logger().Debug("Health check successful", "app_id", appID, "url", healthURL)
	}

	// Update app status in database
	record.Set("status", status)
	if err := app.Save(record); err != nil {
		app.Logger().Error("Failed to update app status", "app_id", appID, "error", err)
	}

	response := map[string]interface{}{
		"app_id":     appID,
		"status":     status,
		"health_url": healthURL,
		"healthy":    isHealthy,
		"checked_at": time.Now().UTC(),
	}

	if errorMsg != "" {
		response["error"] = errorMsg
	}

	return e.JSON(http.StatusOK, response)
}

// performHealthCheck performs an HTTP health check on the given URL
func performHealthCheck(healthURL string) (bool, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return false, fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	return false, fmt.Errorf("health check returned status %d", resp.StatusCode)
}

// getAppLogs handles the get app logs endpoint
func getAppLogs(app core.App, e *core.RequestEvent) error {
	appID := e.Request.PathValue("id")
	if appID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App ID is required",
		})
	}

	// Get lines parameter (default to 100)
	linesStr := e.Request.URL.Query().Get("lines")
	lines := 100
	if linesStr != "" {
		if parsed, err := strconv.Atoi(linesStr); err == nil && parsed > 0 {
			lines = parsed
		}
	}

	// Get app record
	record, err := app.FindRecordById("apps", appID)
	if err != nil {
		app.Logger().Error("Failed to find app", "id", appID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "App not found",
		})
	}

	serviceName := record.GetString("service_name")
	if serviceName == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "App service name not configured",
		})
	}

	// Get server information
	serverRecord, err := app.FindRecordById("servers", record.GetString("server_id"))
	if err != nil {
		app.Logger().Error("Failed to find server for app", "app_id", appID, "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load server information",
		})
	}

	// For now, return a placeholder response
	// TODO: Implement actual log retrieval via SSH
	response := map[string]interface{}{
		"app_id":       appID,
		"service":      serviceName,
		"server":       serverRecord.GetString("name"),
		"lines":        lines,
		"logs":         []string{"Log retrieval not yet implemented"},
		"retrieved_at": time.Now().UTC(),
	}

	return e.JSON(http.StatusOK, response)
}

// validateAppName validates app name format
func validateAppName(name string) error {
	if len(name) < 1 || len(name) > 50 {
		return fmt.Errorf("app name must be between 1 and 50 characters")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
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

// validateDomain validates domain format
func validateDomain(domain string) error {
	if len(domain) < 3 || len(domain) > 255 {
		return fmt.Errorf("domain must be between 3 and 255 characters")
	}

	// Basic domain validation
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("domain must contain at least one dot")
	}

	return nil
}
