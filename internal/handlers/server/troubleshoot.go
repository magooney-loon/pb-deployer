package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

// TroubleshootResponse represents the response for SSH troubleshooting
type TroubleshootResponse struct {
	Success        bool                       `json:"success"`
	ServerID       string                     `json:"server_id"`
	ServerName     string                     `json:"server_name"`
	Host           string                     `json:"host"`
	Port           int                        `json:"port"`
	Timestamp      string                     `json:"timestamp"`
	Diagnostics    []ssh.ConnectionDiagnostic `json:"diagnostics"`
	Summary        string                     `json:"summary"`
	HasErrors      bool                       `json:"has_errors"`
	HasWarnings    bool                       `json:"has_warnings"`
	ErrorCount     int                        `json:"error_count"`
	WarningCount   int                        `json:"warning_count"`
	SuccessCount   int                        `json:"success_count"`
	Suggestions    []string                   `json:"suggestions"`
	ClientIP       string                     `json:"client_ip"`
	ConnectionTime *int                       `json:"connection_time_ms,omitempty"`
	CanAutoFix     bool                       `json:"can_auto_fix"`
	NextSteps      []string                   `json:"next_steps"`
	Severity       string                     `json:"severity"` // critical, high, medium, low, info
}

// EnhancedTroubleshootResponse provides enhanced troubleshooting with analysis and recommendations
type EnhancedTroubleshootResponse struct {
	TroubleshootResponse
	Analysis              map[string]interface{}   `json:"analysis"`
	RecoveryPlan          map[string]interface{}   `json:"recovery_plan"`
	ActionableSuggestions []map[string]interface{} `json:"actionable_suggestions"`
	EstimatedDuration     string                   `json:"estimated_duration"`
	RequiresAccess        []string                 `json:"requires_access"`
	AutoFixAvailable      bool                     `json:"auto_fix_available"`
}

// troubleshootServerConnection performs comprehensive SSH connection troubleshooting
func troubleshootServerConnection(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	// Get query parameters
	enhanced := e.Request.URL.Query().Get("enhanced") == "true"
	autoFix := e.Request.URL.Query().Get("auto_fix") == "true"
	postSecurity := e.Request.URL.Query().Get("post_security") == "true"

	app.Logger().Info("Starting SSH connection troubleshooting",
		"server_id", serverID,
		"enhanced", enhanced,
		"auto_fix", autoFix,
		"post_security", postSecurity,
		"client_ip", getClientIP(e.Request))

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to retrieve server", "server_id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
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

	clientIP := getClientIP(e.Request)
	startTime := time.Now()

	// Choose appropriate diagnostic function
	var diagnostics []ssh.ConnectionDiagnostic
	if postSecurity {
		diagnostics, err = ssh.DiagnoseAppUserPostSecurity(server)
	} else {
		diagnostics, err = ssh.TroubleshootConnection(server, clientIP)
	}

	if err != nil {
		app.Logger().Error("SSH troubleshooting failed",
			"server_id", serverID,
			"host", server.Host,
			"port", server.Port,
			"error", err)

		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Troubleshooting failed: %v", err),
		})
	}

	connectionTime := int(time.Since(startTime).Milliseconds())

	// Process diagnostics
	response := processDiagnostics(server, diagnostics, clientIP, &connectionTime)

	// Handle auto-fix if requested
	if autoFix && response.CanAutoFix {
		fixResults := ssh.FixCommonIssues(server)
		response.Diagnostics = append(response.Diagnostics, fixResults...)
		response = processDiagnostics(server, response.Diagnostics, clientIP, &connectionTime)
	}

	// Return enhanced response if requested
	if enhanced {
		enhancedResponse := enhanceResponse(response, server)
		return e.JSON(http.StatusOK, enhancedResponse)
	}

	return e.JSON(http.StatusOK, response)
}

// quickTroubleshoot performs a quick SSH connection test
func quickTroubleshoot(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	app.Logger().Info("Starting quick SSH troubleshooting",
		"server_id", serverID,
		"client_ip", getClientIP(e.Request))

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to retrieve server", "server_id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
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

	clientIP := getClientIP(e.Request)
	startTime := time.Now()

	// Use connection manager for quick test
	connectionManager := ssh.GetConnectionManager()
	err = connectionManager.TestConnection(server, false)

	connectionTime := int(time.Since(startTime).Milliseconds())

	if err != nil {
		app.Logger().Warn("Quick SSH connection test failed",
			"server_id", serverID,
			"host", server.Host,
			"port", server.Port,
			"connection_time", connectionTime,
			"error", err)

		// If quick test fails, provide basic diagnostic
		diagnostic := ssh.ConnectionDiagnostic{
			Step:       "quick_connection_test",
			Status:     "error",
			Message:    "SSH connection failed",
			Details:    fmt.Sprintf("Connection test failed: %v", err),
			Suggestion: "Run full diagnostics for detailed analysis",
			Duration:   time.Duration(connectionTime) * time.Millisecond,
			Timestamp:  startTime,
		}

		response := TroubleshootResponse{
			Success:        false,
			ServerID:       serverID,
			ServerName:     server.Name,
			Host:           server.Host,
			Port:           server.Port,
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
			Diagnostics:    []ssh.ConnectionDiagnostic{diagnostic},
			Summary:        "Quick connection test failed",
			HasErrors:      true,
			ErrorCount:     1,
			ClientIP:       clientIP,
			ConnectionTime: &connectionTime,
			Severity:       "high",
			NextSteps:      []string{"Run full diagnostics", "Check server status", "Verify credentials"},
		}

		return e.JSON(http.StatusOK, response)
	}

	app.Logger().Info("Quick SSH connection test successful",
		"server_id", serverID,
		"host", server.Host,
		"port", server.Port,
		"connection_time", connectionTime)

	// Success response
	diagnostic := ssh.ConnectionDiagnostic{
		Step:      "quick_connection_test",
		Status:    "success",
		Message:   "SSH connection successful",
		Duration:  time.Duration(connectionTime) * time.Millisecond,
		Timestamp: startTime,
	}

	response := TroubleshootResponse{
		Success:        true,
		ServerID:       serverID,
		ServerName:     server.Name,
		Host:           server.Host,
		Port:           server.Port,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Diagnostics:    []ssh.ConnectionDiagnostic{diagnostic},
		Summary:        "SSH connection is working properly",
		SuccessCount:   1,
		ClientIP:       clientIP,
		ConnectionTime: &connectionTime,
		Severity:       "info",
	}

	return e.JSON(http.StatusOK, response)
}

// enhancedTroubleshoot performs enhanced troubleshooting with detailed analysis
func enhancedTroubleshoot(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	timeoutStr := e.Request.URL.Query().Get("timeout")
	timeout := 2 * time.Minute // default
	if timeoutStr != "" {
		if t, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = t
		}
	}

	app.Logger().Info("Starting enhanced SSH troubleshooting",
		"server_id", serverID,
		"timeout", timeout.String(),
		"client_ip", getClientIP(e.Request))

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to retrieve server", "server_id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
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

	clientIP := getClientIP(e.Request)
	startTime := time.Now()

	// Use context-aware diagnostics with timeout
	diagnostics, err := ssh.DiagnoseWithContext(server, false, timeout)
	if err != nil {
		app.Logger().Error("Enhanced SSH troubleshooting failed",
			"server_id", serverID,
			"host", server.Host,
			"port", server.Port,
			"timeout", timeout.String(),
			"error", err)

		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Enhanced troubleshooting failed: %v", err),
		})
	}

	connectionTime := int(time.Since(startTime).Milliseconds())

	// Process and enhance response
	response := processDiagnostics(server, diagnostics, clientIP, &connectionTime)
	enhancedResponse := enhanceResponse(response, server)

	return e.JSON(http.StatusOK, enhancedResponse)
}

// autoFixIssues attempts to automatically fix detected SSH issues
func autoFixIssues(app core.App, e *core.RequestEvent) error {
	serverID := e.Request.PathValue("id")
	if serverID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "Server ID is required",
		})
	}

	app.Logger().Info("Starting automatic SSH issue fixing",
		"server_id", serverID,
		"client_ip", getClientIP(e.Request))

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		app.Logger().Error("Failed to retrieve server", "server_id", serverID, "error", err)
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "Server not found",
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

	clientIP := getClientIP(e.Request)
	startTime := time.Now()

	// Attempt auto-fixes
	fixResults := ssh.FixCommonIssues(server)
	connectionTime := int(time.Since(startTime).Milliseconds())

	// Process fix results
	response := processDiagnostics(server, fixResults, clientIP, &connectionTime)
	response.Summary = "Auto-fix attempt completed"

	successCount := 0
	for _, diag := range fixResults {
		if diag.Status == "success" {
			successCount++
		}
	}

	if successCount > 0 {
		response.Summary = fmt.Sprintf("Successfully applied %d fixes", successCount)
		response.NextSteps = []string{
			"Test SSH connection again",
			"Run full diagnostics to verify fixes",
		}
	} else {
		response.Summary = "Auto-fix could not resolve issues automatically"
		response.NextSteps = []string{
			"Manual intervention required",
			"Check server console access",
			"Contact system administrator",
		}
	}

	app.Logger().Info("Auto-fix completed",
		"server_id", serverID,
		"fixes_applied", len(fixResults),
		"success_count", successCount)

	return e.JSON(http.StatusOK, response)
}

// processDiagnostics processes diagnostic results into a response structure
func processDiagnostics(server *models.Server, diagnostics []ssh.ConnectionDiagnostic, clientIP string, connectionTime *int) TroubleshootResponse {
	var summary strings.Builder
	var suggestions []string
	var nextSteps []string

	errorCount := 0
	warningCount := 0
	successCount := 0
	canAutoFix := false

	for _, diag := range diagnostics {
		switch diag.Status {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "success":
			successCount++
		}

		if diag.Suggestion != "" {
			suggestions = appendUnique(suggestions, diag.Suggestion)
		}

		// Check if this is auto-fixable
		if diag.Status == "warning" &&
			(strings.Contains(diag.Suggestion, "chmod") ||
				strings.Contains(diag.Suggestion, "mkdir") ||
				strings.Contains(diag.Suggestion, "ssh-keyscan")) {
			canAutoFix = true
		}
	}

	// Generate summary
	if errorCount > 0 {
		summary.WriteString(fmt.Sprintf("❌ %d critical issues found", errorCount))
		if warningCount > 0 {
			summary.WriteString(fmt.Sprintf(" and %d warnings", warningCount))
		}
	} else if warningCount > 0 {
		summary.WriteString(fmt.Sprintf("⚠️ %d warnings found", warningCount))
	} else {
		summary.WriteString("✅ All checks passed")
	}

	// Determine severity
	severity := "info"
	if errorCount > 0 {
		severity = "critical"
	} else if warningCount > 2 {
		severity = "high"
	} else if warningCount > 0 {
		severity = "medium"
	}

	// Generate next steps
	if errorCount > 0 {
		nextSteps = []string{
			"Address critical errors first",
			"Check server console access if connection fails",
			"Verify server is running and accessible",
		}

		// Add specific steps based on error types
		for _, diag := range diagnostics {
			if diag.Status == "error" {
				if diag.Step == "network_connectivity" && strings.Contains(diag.Details, "connection refused") {
					nextSteps = appendUnique(nextSteps, "Check fail2ban status - IP might be banned")
				}
				if diag.Step == "ssh_connection" {
					nextSteps = appendUnique(nextSteps, "Verify SSH key configuration and authorized_keys")
				}
			}
		}
	} else if warningCount > 0 {
		nextSteps = []string{
			"Connection should work but consider fixing warnings",
			"Test actual SSH connection to verify",
		}
	} else {
		nextSteps = []string{
			"SSH connection is ready",
			"Proceed with deployment or server management",
		}
	}

	return TroubleshootResponse{
		Success:        errorCount == 0,
		ServerID:       server.ID,
		ServerName:     server.Name,
		Host:           server.Host,
		Port:           server.Port,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		Diagnostics:    diagnostics,
		Summary:        summary.String(),
		HasErrors:      errorCount > 0,
		HasWarnings:    warningCount > 0,
		ErrorCount:     errorCount,
		WarningCount:   warningCount,
		SuccessCount:   successCount,
		Suggestions:    suggestions,
		ClientIP:       clientIP,
		ConnectionTime: connectionTime,
		CanAutoFix:     canAutoFix,
		NextSteps:      nextSteps,
		Severity:       severity,
	}
}

// enhanceResponse creates an enhanced response with analysis and recovery plans
func enhanceResponse(response TroubleshootResponse, server *models.Server) EnhancedTroubleshootResponse {
	// Analyze diagnostic patterns
	analysis := ssh.AnalyzeDiagnosticPatterns(response.Diagnostics)

	// Generate actionable suggestions
	actionableSuggestions := ssh.GenerateActionableSuggestions(response.Diagnostics, server)

	// Generate recovery plan
	recoveryPlan := ssh.GenerateRecoveryPlan(response.Diagnostics, server)

	// Estimate duration
	estimatedDuration := ssh.EstimateDiagnosticDuration(server, true)

	// Determine access requirements
	var requiresAccess []string
	if response.HasErrors {
		for _, diag := range response.Diagnostics {
			if diag.Status == "error" {
				if strings.Contains(diag.Suggestion, "fail2ban") {
					requiresAccess = appendUnique(requiresAccess, "server_console")
				}
				if strings.Contains(diag.Suggestion, "systemctl") {
					requiresAccess = appendUnique(requiresAccess, "sudo_access")
				}
			}
		}
	}

	return EnhancedTroubleshootResponse{
		TroubleshootResponse:  response,
		Analysis:              analysis,
		RecoveryPlan:          recoveryPlan,
		ActionableSuggestions: actionableSuggestions,
		EstimatedDuration:     estimatedDuration.String(),
		RequiresAccess:        requiresAccess,
		AutoFixAvailable:      response.CanAutoFix,
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header (alternative reverse proxy header)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Check CF-Connecting-IP header (Cloudflare)
	cfIP := r.Header.Get("CF-Connecting-IP")
	if cfIP != "" {
		return cfIP
	}

	// Fall back to RemoteAddr
	if r.RemoteAddr != "" {
		// RemoteAddr format is "IP:port", so we need to split
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			return host
		}
		return r.RemoteAddr
	}

	return "unknown"
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// appendUnique appends a string to a slice only if it's not already present
func appendUnique(slice []string, item string) []string {
	if !contains(slice, item) {
		slice = append(slice, item)
	}
	return slice
}
