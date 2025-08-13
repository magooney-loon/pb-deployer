package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

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

		// Get SSH service for setup operations
		sshService := ssh.GetSSHService()

		// Create progress channel
		progressChan := make(chan ssh.SetupStep, 10)
		setupDone := make(chan error, 1)

		// Start progress monitoring
		go func() {
			for step := range progressChan {
				notifySetupProgress(app, serverID, step)
			}
		}()

		// Run server setup using SSH service
		go func() {
			defer close(progressChan)
			err := sshService.RunServerSetup(server, progressChan)
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

	return e.JSON(http.StatusOK, map[string]any{
		"message":   "Server setup started",
		"server_id": serverID,
	})
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
	return e.JSON(http.StatusOK, map[string]any{
		"message":      "Setup progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("server_setup_%s", serverID),
	})
}
