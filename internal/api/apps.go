package api

// API_SOURCE

import (
	"encoding/json"
	"fmt"
	"net/http"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/tunnel"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Create a new app with auto-allocated loopback port (must use this instead of direct PocketBase collection API)
// API_TAGS Apps
func handleCreateApp(c *core.RequestEvent) error {
	app := c.App
	log := logger.GetAPILogger()

	type createAppRequest struct {
		ServerID string `json:"server_id"`
		Name     string `json:"name"`
		Domain   string `json:"domain"`
	}

	var req createAppRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "Invalid request body"})
	}

	if req.ServerID == "" || req.Name == "" || req.Domain == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "server_id, name, and domain are required",
		})
	}

	serviceName := "pocketbase-" + req.Name
	remotePath := "/opt/pocketbase/apps/" + req.Name

	if _, err := app.FindRecordById("servers", req.ServerID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "Server not found"})
	}

	httpPort, err := allocatePort(app, req.ServerID)
	if err != nil {
		log.Error("Port allocation failed: %v", err)
		return c.JSON(http.StatusConflict, map[string]any{"error": err.Error()})
	}

	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Apps collection not found"})
	}

	record := core.NewRecord(appsCollection)
	record.Set("server_id", req.ServerID)
	record.Set("name", req.Name)
	record.Set("domain", req.Domain)
	record.Set("service_name", serviceName)
	record.Set("remote_path", remotePath)
	record.Set("http_port", httpPort)
	record.Set("status", "offline")

	if err := app.Save(record); err != nil {
		log.Error("Failed to save app record: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": fmt.Sprintf("Failed to create app: %v", err)})
	}

	log.Success("App created: %s (port %d)", req.Name, httpPort)
	return c.JSON(http.StatusOK, map[string]any{
		"success":   true,
		"id":        record.Id,
		"http_port": httpPort,
	})
}

// API_DESC Migrate a legacy app (no http_port) to the Caddy reverse proxy
// API_TAGS Apps
func handleMigrateProxy(c *core.RequestEvent) error {
	app := c.App
	log := logger.GetAPILogger()

	appID := c.Request.PathValue("id")
	if appID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "App ID required"})
	}

	appRecord, err := app.FindRecordById("apps", appID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "App not found"})
	}

	serverID := appRecord.GetString("server_id")
	serverRecord, err := app.FindRecordById("servers", serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "Server not found"})
	}

	httpPort := appRecord.GetInt("http_port")
	if httpPort == 0 {
		httpPort, err = allocatePort(app, serverID)
		if err != nil {
			return c.JSON(http.StatusConflict, map[string]any{"error": err.Error()})
		}
	}

	client, err := createSSHClient(
		serverRecord.GetString("host"),
		serverRecord.GetInt("port"),
		serverRecord.GetString("root_username"),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "Failed to create SSH client"})
	}

	cleanup := tunnel.NewCleanupManager()
	defer cleanup.Close()
	cleanup.AddCloser(client)

	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": "SSH connection failed"})
	}

	manager := tunnel.NewManager(client)
	cleanup.AddCloser(manager)

	setupManager := tunnel.NewSetupManager(manager)
	cleanup.AddCloser(setupManager)

	migrateReq := tunnel.MigrateProxyRequest{
		AppName:     appRecord.GetString("name"),
		Domain:      appRecord.GetString("domain"),
		ServiceName: appRecord.GetString("service_name"),
		AppUsername: serverRecord.GetString("app_username"),
		HTTPPort:    httpPort,
	}

	if err := setupManager.MigrateAppToProxy(migrateReq); err != nil {
		log.Error("Migration failed for app %s: %v", migrateReq.AppName, err)
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": fmt.Sprintf("Migration failed: %v", err)})
	}

	appRecord.Set("http_port", httpPort)
	appRecord.Set("status", "online")
	if err := app.Save(appRecord); err != nil {
		log.Warning("Failed to save app record after migration: %v", err)
	}

	log.Success("App %s migrated to Caddy proxy (port %d)", migrateReq.AppName, httpPort)
	return c.JSON(http.StatusOK, map[string]any{
		"success":   true,
		"http_port": httpPort,
	})
}

func RegisterAppDeleteHook(pbApp core.App) {
	pbApp.OnRecordDelete("apps").BindFunc(func(e *core.RecordEvent) error {
		log := logger.GetAPILogger()

		appRecord := e.Record
		serverID := appRecord.GetString("server_id")
		appName := appRecord.GetString("name")

		serverRecord, err := pbApp.FindRecordById("servers", serverID)
		if err != nil {
			log.Warning("App delete hook: server %s not found, skipping remote cleanup", serverID)
			return e.Next()
		}

		client, err := createSSHClient(
			serverRecord.GetString("host"),
			serverRecord.GetInt("port"),
			serverRecord.GetString("root_username"),
		)
		if err != nil {
			log.Warning("App delete hook: SSH client creation failed for %s: %v", appName, err)
			return e.Next()
		}

		cleanup := tunnel.NewCleanupManager()
		defer cleanup.Close()
		cleanup.AddCloser(client)

		if err := client.Connect(); err != nil {
			log.Warning("App delete hook: SSH connection failed for %s: %v", appName, err)
			return e.Next()
		}

		manager := tunnel.NewManager(client)
		cleanup.AddCloser(manager)

		setupManager := tunnel.NewSetupManager(manager)
		cleanup.AddCloser(setupManager)

		cleanupReq := tunnel.CleanupAppRequest{
			AppName:     appName,
			ServiceName: appRecord.GetString("service_name"),
		}

		if err := setupManager.CleanupApp(cleanupReq); err != nil {
			log.Warning("App delete hook: remote cleanup failed for %s: %v", appName, err)
		}

		return e.Next()
	})
}

// allocatePort picks the lowest free port in [8090, 8999] not yet used by another app on the same server.
func allocatePort(app core.App, serverID string) (int, error) {
	existingApps, err := app.FindRecordsByFilter(
		"apps",
		"server_id = {:server_id}",
		"", 0, 0,
		map[string]any{"server_id": serverID},
	)
	if err != nil {
		return 0, fmt.Errorf("failed to query existing apps: %w", err)
	}

	used := make(map[int]bool)
	for _, a := range existingApps {
		if p := a.GetInt("http_port"); p > 0 {
			used[p] = true
		}
	}

	for port := 8090; port <= 8999; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports available in [8090, 8999] for server %s", serverID)
}
