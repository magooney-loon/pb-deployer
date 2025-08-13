package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

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

		// Get SSH service for security operations
		sshService := ssh.GetSSHService()

		// Create progress channel
		progressChan := make(chan ssh.SetupStep, 10)
		securityDone := make(chan error, 1)

		// Start progress monitoring
		go func() {
			for step := range progressChan {
				notifySecurityProgress(app, serverID, step)
			}
		}()

		// Run security lockdown using SSH service
		go func() {
			defer close(progressChan)
			err := sshService.ApplySecurityLockdown(server, progressChan)
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

		// Note: After security lockdown, connections will automatically use app user
		app.Logger().Info("Security lockdown completed - future connections will use app user", "server_id", serverID)

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

	return e.JSON(http.StatusOK, map[string]any{
		"message":   "Security lockdown started",
		"server_id": serverID,
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
	return e.JSON(http.StatusOK, map[string]any{
		"message":      "Security progress available via PocketBase realtime",
		"subscription": fmt.Sprintf("server_security_%s", serverID),
	})
}
