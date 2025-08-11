package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

// TroubleshootResponse represents the response for troubleshooting
type TroubleshootResponse struct {
	Success      bool                       `json:"success"`
	ServerID     string                     `json:"server_id"`
	ServerName   string                     `json:"server_name"`
	Host         string                     `json:"host"`
	Port         int                        `json:"port"`
	Timestamp    string                     `json:"timestamp"`
	Diagnostics  []ssh.ConnectionDiagnostic `json:"diagnostics"`
	Summary      string                     `json:"summary"`
	HasErrors    bool                       `json:"has_errors"`
	HasWarnings  bool                       `json:"has_warnings"`
	ErrorCount   int                        `json:"error_count"`
	WarningCount int                        `json:"warning_count"`
	SuccessCount int                        `json:"success_count"`
	Suggestions  []string                   `json:"suggestions"`
}

// troubleshootServerConnection handles the troubleshooting endpoint
func troubleshootServerConnection(app core.App, e *core.RequestEvent) error {
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
		return e.JSON(http.StatusNotFound, TroubleshootResponse{
			Success:  false,
			ServerID: serverID,
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

	app.Logger().Info("Starting server troubleshooting",
		"server_id", serverID,
		"host", server.Host,
		"port", server.Port,
		"security_locked", server.SecurityLocked)

	// Run appropriate troubleshooting based on server status
	var diagnostics []ssh.ConnectionDiagnostic
	var summary string

	if server.SecurityLocked {
		// For security-locked servers, use post-security diagnostics
		app.Logger().Debug("Running post-security troubleshooting", "server_id", serverID)
		diagnostics, err = ssh.DiagnoseAppUserPostSecurity(server)
		if err != nil {
			app.Logger().Error("Failed to run post-security diagnostics", "server_id", serverID, "error", err)
			return e.JSON(http.StatusInternalServerError, TroubleshootResponse{
				Success:  false,
				ServerID: serverID,
			})
		}
		summary, _ = ssh.GetPostSecurityTroubleshootingSummary(server)
	} else {
		// For regular servers, run comprehensive troubleshooting
		app.Logger().Debug("Running comprehensive troubleshooting", "server_id", serverID)
		diagnostics, err = ssh.TroubleshootConnection(server, false) // Start with app user
		if err != nil {
			app.Logger().Error("Failed to run troubleshooting", "server_id", serverID, "error", err)
			return e.JSON(http.StatusInternalServerError, TroubleshootResponse{
				Success:  false,
				ServerID: serverID,
			})
		}
		summary, _ = ssh.GetConnectionSummary(server, false)
	}

	// Analyze diagnostic results
	errorCount := 0
	warningCount := 0
	successCount := 0
	suggestions := make([]string, 0)

	for _, diag := range diagnostics {
		switch diag.Status {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "success":
			successCount++
		}

		// Collect unique suggestions
		if diag.Suggestion != "" {
			suggestions = appendUnique(suggestions, diag.Suggestion)
		}
	}

	// Check for specific connection issues and add fail2ban diagnostics if needed
	hasConnectionRefused := false
	for _, diag := range diagnostics {
		if diag.Step == "network_connectivity" && diag.Status == "error" {
			if contains(diag.Details, "connection refused") {
				hasConnectionRefused = true
				// Add fail2ban diagnostic
				fail2banDiag := ssh.DiagnoseConnectionRefused(server)
				diagnostics = append(diagnostics, fail2banDiag)
				if fail2banDiag.Suggestion != "" {
					suggestions = appendUnique(suggestions, fail2banDiag.Suggestion)
				}
				if fail2banDiag.Status == "error" {
					errorCount++
				} else if fail2banDiag.Status == "warning" {
					warningCount++
				}
				break
			}
		}
	}

	// Add general connection refused guidance if detected
	if hasConnectionRefused {
		suggestions = appendUnique(suggestions, "If you have console access, check: sudo fail2ban-client status sshd")
		suggestions = appendUnique(suggestions, "Try connecting from a different IP address (mobile hotspot, VPN)")
		suggestions = appendUnique(suggestions, "Wait 10-15 minutes as fail2ban bans are often temporary")
	}

	response := TroubleshootResponse{
		Success:      errorCount == 0,
		ServerID:     serverID,
		ServerName:   server.Name,
		Host:         server.Host,
		Port:         server.Port,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Diagnostics:  diagnostics,
		Summary:      summary,
		HasErrors:    errorCount > 0,
		HasWarnings:  warningCount > 0,
		ErrorCount:   errorCount,
		WarningCount: warningCount,
		SuccessCount: successCount,
		Suggestions:  suggestions,
	}

	app.Logger().Info("Server troubleshooting completed",
		"server_id", serverID,
		"success", response.Success,
		"errors", errorCount,
		"warnings", warningCount,
		"diagnostics_count", len(diagnostics))

	return e.JSON(http.StatusOK, response)
}

// quickTroubleshoot handles quick troubleshooting for immediate feedback
func quickTroubleshoot(app core.App, e *core.RequestEvent) error {
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

	server := &models.Server{
		ID:   record.Id,
		Name: record.GetString("name"),
		Host: record.GetString("host"),
		Port: record.GetInt("port"),
	}

	app.Logger().Info("Running quick troubleshooting", "server_id", serverID, "host", server.Host)

	// Run quick fail2ban check
	err = ssh.QuickFail2banCheck(server.Host, server.Port)

	var status string
	var message string
	var suggestion string

	if err != nil {
		if contains(err.Error(), "connection refused") {
			status = "error"
			message = "Connection refused - likely fail2ban IP ban"
			suggestion = "Check if your IP is banned by fail2ban and unban if necessary"
		} else if contains(err.Error(), "timeout") {
			status = "warning"
			message = "Connection timeout - network or firewall issue"
			suggestion = "Check network connectivity and firewall settings"
		} else {
			status = "error"
			message = fmt.Sprintf("Connection failed: %v", err)
			suggestion = "Check server status and network connectivity"
		}
	} else {
		status = "success"
		message = "Connection successful - SSH port is reachable"
		suggestion = ""
	}

	response := map[string]interface{}{
		"success":    status == "success",
		"server_id":  serverID,
		"host":       server.Host,
		"port":       server.Port,
		"status":     status,
		"message":    message,
		"suggestion": suggestion,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	return e.JSON(http.StatusOK, response)
}

// Helper functions
func contains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr ||
		len(str) > len(substr) && str[len(str)-len(substr):] == substr ||
		(len(str) > len(substr) &&
			func() bool {
				for i := 0; i <= len(str)-len(substr); i++ {
					if str[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())
}

func appendUnique(slice []string, item string) []string {
	for _, existing := range slice {
		if existing == item {
			return slice
		}
	}
	return append(slice, item)
}
