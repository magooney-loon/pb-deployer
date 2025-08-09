package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"golang.org/x/sync/errgroup"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

// ConnectionTestResponse represents the response for connection test
type ConnectionTestResponse struct {
	Success           bool            `json:"success"`
	Error             string          `json:"error,omitempty"`
	ConnectionInfo    *ConnectionInfo `json:"connection_info,omitempty"`
	AppUserConnection string          `json:"app_user_connection,omitempty"`
}

// ConnectionInfo represents the connection details
type ConnectionInfo struct {
	ServerHost string `json:"server_host"`
	Username   string `json:"username"`
}

// RegisterServerHandlers registers all server-related HTTP handlers
func RegisterServerHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
	group.POST("/servers/{id}/test", func(e *core.RequestEvent) error {
		return testServerConnection(app, e)
	})

	group.POST("/servers/{id}/setup", func(e *core.RequestEvent) error {
		return runServerSetup(app, e)
	})

	group.POST("/servers/{id}/security", func(e *core.RequestEvent) error {
		return applySecurityLockdown(app, e)
	})

	group.GET("/servers/{id}/status", func(e *core.RequestEvent) error {
		return getServerStatus(app, e)
	})

	// WebSocket endpoint for setup progress
	group.GET("/servers/{id}/setup-ws", func(e *core.RequestEvent) error {
		return handleSetupWebSocket(app, e)
	})

	// WebSocket endpoint for security progress
	group.GET("/servers/{id}/security-ws", func(e *core.RequestEvent) error {
		return handleSecurityWebSocket(app, e)
	})
}

// testServerConnection handles the connection test endpoint
func testServerConnection(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to find server", "id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, ConnectionTestResponse{
			Success: false,
			Error:   "Server not found",
		})
	}

	// Extract server data from record
	host := record.GetString("host")
	port := record.GetInt("port")
	rootUsername := record.GetString("root_username")
	appUsername := record.GetString("app_username")

	// Test basic connectivity (TCP connection)
	success, connErr := testTCPConnection(host, port)

	response := ConnectionTestResponse{
		Success: success,
	}

	if success {
		response.ConnectionInfo = &ConnectionInfo{
			ServerHost: host,
			Username:   rootUsername,
		}
		response.AppUserConnection = fmt.Sprintf("App user: %s", appUsername)
		app.Logger().Info("Server connection test successful",
			"server_id", serverID,
			"host", host,
			"port", port)
	} else {
		response.Error = connErr.Error()
		app.Logger().Warn("Server connection test failed",
			"server_id", serverID,
			"host", host,
			"port", port,
			"error", connErr)
	}

	return e.JSON(http.StatusOK, response)
}

// testTCPConnection performs a simple TCP connection test to check if the server is reachable
func testTCPConnection(host string, port int) (bool, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))

	// Set a reasonable timeout for the connection test
	timeout := 5 * time.Second

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false, fmt.Errorf("connection failed: %v", err)
	}

	defer conn.Close()
	return true, nil
}

// runServerSetup handles the server setup endpoint
func runServerSetup(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to find server", "id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	// Check if setup is already complete
	if record.GetBool("setup_complete") {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server setup is already complete",
		})
	}

	// Convert PocketBase record to models.Server
	server := &models.Server{
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

	// Start setup process in background
	go func() {
		app.Logger().Info("Starting server setup process", "server_id", serverID)

		// Send initial setup started notification
		notifySetupProgress(app, serverID, ssh.SetupStep{
			Step:        "init",
			Status:      "running",
			Message:     "Initializing server setup",
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: 0,
		})

		// Create SSH manager as root for setup
		sshManager, err := ssh.NewSSHManager(server, true)
		if err != nil {
			app.Logger().Error("Failed to create SSH manager", "server_id", serverID, "error", err)
			notifySetupProgress(app, serverID, ssh.SetupStep{
				Step:        "init",
				Status:      "failed",
				Message:     "Failed to establish SSH connection",
				Details:     err.Error(),
				Timestamp:   time.Now().Format(time.RFC3339),
				ProgressPct: 0,
			})
			return
		}
		defer sshManager.Close()

		// Create progress channel
		progressChan := make(chan ssh.SetupStep, 10)
		setupDone := make(chan error, 1)

		// Start progress monitoring
		go func() {
			for step := range progressChan {
				notifySetupProgress(app, serverID, step)
			}
		}()

		// Run server setup
		go func() {
			defer close(progressChan)
			err := sshManager.RunServerSetup(progressChan)
			setupDone <- err
		}()

		// Wait for setup completion
		if setupErr := <-setupDone; setupErr != nil {
			app.Logger().Error("Server setup failed", "server_id", serverID, "error", setupErr)
			notifySetupProgress(app, serverID, ssh.SetupStep{
				Step:        "complete",
				Status:      "failed",
				Message:     "Server setup failed",
				Details:     setupErr.Error(),
				Timestamp:   time.Now().Format(time.RFC3339),
				ProgressPct: 100,
			})
			return
		}

		// Update database to mark setup as complete
		record.Set("setup_complete", true)
		if err := app.Save(record); err != nil {
			app.Logger().Error("Failed to update server setup status", "server_id", serverID, "error", err)
		}

		// Send completion notification
		notifySetupProgress(app, serverID, ssh.SetupStep{
			Step:        "complete",
			Status:      "success",
			Message:     "Server setup completed successfully",
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: 100,
		})

		app.Logger().Info("Server setup completed successfully", "server_id", serverID)
	}()

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Server setup started",
		"server_id": serverID,
	})
}

// applySecurityLockdown handles the security lockdown endpoint
func applySecurityLockdown(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to find server", "id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	// Check if setup is complete
	if !record.GetBool("setup_complete") {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server setup must be completed before applying security lockdown",
		})
	}

	// Check if security is already locked
	if record.GetBool("security_locked") {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server security is already locked down",
		})
	}

	// Convert PocketBase record to models.Server
	server := &models.Server{
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

	// Start security lockdown process in background
	go func() {
		app.Logger().Info("Starting security lockdown process", "server_id", serverID)

		// Send initial security started notification
		notifySecurityProgress(app, serverID, ssh.SetupStep{
			Step:        "init",
			Status:      "running",
			Message:     "Initializing security lockdown",
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: 0,
		})

		// Create SSH manager as root for security operations
		sshManager, err := ssh.NewSSHManager(server, true)
		if err != nil {
			app.Logger().Error("Failed to create SSH manager", "server_id", serverID, "error", err)
			notifySecurityProgress(app, serverID, ssh.SetupStep{
				Step:        "init",
				Status:      "failed",
				Message:     "Failed to establish SSH connection",
				Details:     err.Error(),
				Timestamp:   time.Now().Format(time.RFC3339),
				ProgressPct: 0,
			})
			return
		}
		defer sshManager.Close()

		// Create progress channel
		progressChan := make(chan ssh.SetupStep, 10)
		securityDone := make(chan error, 1)

		// Start progress monitoring
		go func() {
			for step := range progressChan {
				notifySecurityProgress(app, serverID, step)
			}
		}()

		// Run security lockdown
		go func() {
			defer close(progressChan)
			err := sshManager.ApplySecurityLockdown(progressChan)
			securityDone <- err
		}()

		// Wait for security completion
		if securityErr := <-securityDone; securityErr != nil {
			app.Logger().Error("Security lockdown failed", "server_id", serverID, "error", securityErr)
			notifySecurityProgress(app, serverID, ssh.SetupStep{
				Step:        "complete",
				Status:      "failed",
				Message:     "Security lockdown failed",
				Details:     securityErr.Error(),
				Timestamp:   time.Now().Format(time.RFC3339),
				ProgressPct: 100,
			})
			return
		}

		// Update database to mark security as locked
		record.Set("security_locked", true)
		if err := app.Save(record); err != nil {
			app.Logger().Error("Failed to update server security status", "server_id", serverID, "error", err)
		}

		// Send completion notification
		notifySecurityProgress(app, serverID, ssh.SetupStep{
			Step:        "complete",
			Status:      "success",
			Message:     "Security lockdown completed successfully",
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: 100,
		})

		app.Logger().Info("Security lockdown completed successfully", "server_id", serverID)
	}()

	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":   "Security lockdown started",
		"server_id": serverID,
	})
}

// getServerStatus handles the server status endpoint
func getServerStatus(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to find server", "id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
		})
	}

	// Extract server data from record
	host := record.GetString("host")
	port := record.GetInt("port")
	setupComplete := record.GetBool("setup_complete")
	securityLocked := record.GetBool("security_locked")

	// Test connection for current status
	connected, connErr := testTCPConnection(host, port)

	status := map[string]interface{}{
		"server_id":       serverID,
		"setup_complete":  setupComplete,
		"security_locked": securityLocked,
		"connection":      "offline",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	if connected {
		status["connection"] = "online"
	} else {
		status["connection_error"] = connErr.Error()
	}

	app.Logger().Info("Server status checked", "server_id", serverID, "connected", connected)

	return e.JSON(http.StatusOK, status)
}

// handleSetupWebSocket handles WebSocket connections for setup progress
func handleSetupWebSocket(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// For now, return a simple response since we're using PocketBase realtime
	// The actual WebSocket connection is handled by PocketBase's realtime system
	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Setup progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("server_setup_%s", serverID),
	})
}

// handleSecurityWebSocket handles WebSocket connections for security progress
func handleSecurityWebSocket(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// For now, return a simple response since we're using PocketBase realtime
	// The actual WebSocket connection is handled by PocketBase's realtime system
	return e.JSON(http.StatusOK, map[string]interface{}{
		"message":      "Security progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("server_security_%s", serverID),
	})
}

// notifySetupProgress sends setup progress updates to all subscribed clients
func notifySetupProgress(app core.App, serverID string, step ssh.SetupStep) error {
	subscription := fmt.Sprintf("server_setup_%s", serverID)
	return notifyClients(app, subscription, step)
}

// notifySecurityProgress sends security progress updates to all subscribed clients
func notifySecurityProgress(app core.App, serverID string, step ssh.SetupStep) error {
	subscription := fmt.Sprintf("server_security_%s", serverID)
	return notifyClients(app, subscription, step)
}

// notifyClients sends a message to all clients subscribed to a specific topic
func notifyClients(app core.App, subscription string, data interface{}) error {
	// Add debugging to see what we're sending
	app.Logger().Debug("Sending realtime message",
		"subscription", subscription,
		"data", data)

	rawData, err := json.Marshal(data)
	if err != nil {
		app.Logger().Error("Failed to marshal data for realtime",
			"subscription", subscription,
			"data", data,
			"error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	app.Logger().Debug("Marshaled data",
		"subscription", subscription,
		"raw_data", string(rawData))

	message := subscriptions.Message{
		Name: subscription,
		Data: rawData,
	}

	group := new(errgroup.Group)
	chunks := app.SubscriptionsBroker().ChunkedClients(300)

	clientCount := 0
	for _, chunk := range chunks {
		group.Go(func() error {
			for _, client := range chunk {
				if !client.HasSubscription(subscription) {
					continue
				}
				clientCount++
				client.Send(message)
			}
			return nil
		})
	}

	app.Logger().Debug("Sent realtime message to clients",
		"subscription", subscription,
		"client_count", clientCount)

	return group.Wait()
}
