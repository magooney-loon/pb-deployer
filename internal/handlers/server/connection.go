package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

// ConnectionTestResponse represents the response for connection test
type ConnectionTestResponse struct {
	Success           bool          `json:"success"`
	Error             string        `json:"error,omitempty"`
	TCPConnection     TCPTestResult `json:"tcp_connection"`
	RootSSHConnection SSHTestResult `json:"root_ssh_connection"`
	AppSSHConnection  SSHTestResult `json:"app_ssh_connection"`
	OverallStatus     string        `json:"overall_status"`
}

// TCPTestResult represents TCP connectivity test results
type TCPTestResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// SSHTestResult represents SSH connection test results
type SSHTestResult struct {
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	Username   string `json:"username"`
	AuthMethod string `json:"auth_method,omitempty"`
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

	app.Logger().Info("Starting comprehensive connection test",
		"server_id", serverID,
		"host", server.Host,
		"port", server.Port,
		"security_locked", server.SecurityLocked)

	// Create context with timeout for the entire test
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test TCP connectivity first
	tcpResult := testTCPConnectionEnhanced(server.Host, server.Port)

	// Test SSH connections based on security status
	var rootSSHResult SSHTestResult
	var appSSHResult SSHTestResult

	if server.SecurityLocked {
		// For security-locked servers, root SSH should be disabled
		rootSSHResult = SSHTestResult{
			Success:  false,
			Username: server.RootUsername,
			Error:    "Root SSH access disabled after security lockdown",
		}
		// Focus on app user connection for security-locked servers
		appSSHResult = testSSHConnectionEnhancedWithContext(ctx, server, false, app)
	} else {
		// For non-security-locked servers, test both connections using enhanced version
		rootSSHResult = testSSHConnectionEnhancedWithContext(ctx, server, true, app) // as root
		appSSHResult = testSSHConnectionEnhancedWithContext(ctx, server, false, app) // as app user
	}

	// Determine overall success based on security status
	var overallSuccess bool
	var overallStatus string

	if server.SecurityLocked {
		// For security-locked servers, only app user connection matters
		overallSuccess = tcpResult.Success && appSSHResult.Success
		if !tcpResult.Success {
			overallStatus = "unreachable"
		} else if !appSSHResult.Success {
			overallStatus = "app_ssh_failed"
		} else {
			overallStatus = "healthy_secured"
		}
	} else {
		// For non-security-locked servers, both connections should work
		overallSuccess = tcpResult.Success && rootSSHResult.Success && appSSHResult.Success
		if !tcpResult.Success {
			overallStatus = "unreachable"
		} else if !rootSSHResult.Success && !appSSHResult.Success {
			overallStatus = "ssh_failed"
		} else if !rootSSHResult.Success {
			overallStatus = "root_ssh_failed"
		} else if !appSSHResult.Success {
			overallStatus = "app_ssh_failed"
		} else {
			overallStatus = "healthy"
		}
	}

	response := ConnectionTestResponse{
		Success:           overallSuccess,
		TCPConnection:     tcpResult,
		RootSSHConnection: rootSSHResult,
		AppSSHConnection:  appSSHResult,
		OverallStatus:     overallStatus,
	}

	if !overallSuccess {
		if server.SecurityLocked {
			response.Error = fmt.Sprintf("Connection issues detected: %s (Note: Root SSH is expected to be disabled on security-locked servers)", overallStatus)
		} else {
			response.Error = fmt.Sprintf("Connection issues detected: %s", overallStatus)
		}
	}

	app.Logger().Info("Connection test completed",
		"server_id", serverID,
		"security_locked", server.SecurityLocked,
		"tcp_success", tcpResult.Success,
		"root_ssh_success", rootSSHResult.Success,
		"app_ssh_success", appSSHResult.Success,
		"overall_status", overallStatus)

	return e.JSON(http.StatusOK, response)
}

// testTCPConnectionEnhanced performs a TCP connection test with latency measurement
func testTCPConnectionEnhanced(host string, port int) TCPTestResult {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	timeout := 5 * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	latency := time.Since(start)

	if err != nil {
		return TCPTestResult{
			Success: false,
			Error:   fmt.Sprintf("connection failed: %v", err),
		}
	}
	defer conn.Close()

	return TCPTestResult{
		Success: true,
		Latency: fmt.Sprintf("%.2fms", float64(latency.Nanoseconds())/1e6),
	}
}

// testSSHConnectionEnhanced tests SSH connectivity for a specific user with enhanced error handling
func testSSHConnectionEnhanced(server *models.Server, asRoot bool, app core.App) SSHTestResult {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return testSSHConnectionEnhancedWithContext(ctx, server, asRoot, app)
}

// testSSHConnectionEnhancedWithContext tests SSH connectivity with context for timeout control
func testSSHConnectionEnhancedWithContext(ctx context.Context, server *models.Server, asRoot bool, app core.App) SSHTestResult {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	// Use SSH service for connection testing
	sshService := ssh.GetSSHService()

	// Test connection with context timeout
	testResult := sshService.TestConnectionWithContext(ctx, server, asRoot)

	// Convert SSH service result to handler result format
	result := SSHTestResult{
		Username:   username,
		Success:    testResult.Success,
		Error:      testResult.Error,
		AuthMethod: testResult.AuthMethod,
	}

	if testResult.Success {
		app.Logger().Debug("SSH connection successful",
			"username", username,
			"host", server.Host,
			"auth_method", result.AuthMethod,
			"response_time", testResult.ResponseTime)
	} else {
		app.Logger().Debug("SSH connection failed",
			"username", username,
			"host", server.Host,
			"error", testResult.Error)
	}

	return result
}

// testSSHConnection tests SSH connectivity for a specific user (legacy function - deprecated)
// Use testSSHConnectionEnhanced instead for better reliability
func testSSHConnection(server *models.Server, asRoot bool, app core.App) SSHTestResult {
	app.Logger().Debug("Using legacy SSH connection test - consider upgrading to enhanced version",
		"host", server.Host,
		"as_root", asRoot)

	// Delegate to enhanced version for consistency
	return testSSHConnectionEnhanced(server, asRoot, app)
}

// testTCPConnection performs a simple TCP connection test to check if the server is reachable
// Kept for compatibility with other parts of the code
func testTCPConnection(host string, port int) (bool, error) {
	result := testTCPConnectionEnhanced(host, port)
	if result.Success {
		return true, nil
	}
	return false, errors.New(result.Error)
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

	// Test connection for current status with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test connection with context timeout
	connected := false
	var connErr error

	connectionDone := make(chan bool, 1)
	connectionErrChan := make(chan error, 1)

	go func() {
		conn, err := testTCPConnection(host, port)
		connectionDone <- conn
		connectionErrChan <- err
	}()

	status := map[string]interface{}{
		"server_id":       serverID,
		"setup_complete":  setupComplete,
		"security_locked": securityLocked,
		"connection":      "offline",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	select {
	case connected = <-connectionDone:
		select {
		case connErr = <-connectionErrChan:
		default:
		}
	case <-ctx.Done():
		status["connection"] = "timeout"
		status["connection_error"] = "Connection test timed out"
		app.Logger().Info("Server status checked - timeout",
			"server_id", serverID,
			"host", host,
			"port", port)
		return e.JSON(http.StatusOK, status)
	}

	if connected {
		status["connection"] = "online"
	} else if connErr != nil {
		status["connection_error"] = connErr.Error()
	}

	app.Logger().Info("Server status checked",
		"server_id", serverID,
		"connected", connected,
		"host", host,
		"port", port)

	return e.JSON(http.StatusOK, status)
}

// getConnectionHealth handles the connection health endpoint
func getConnectionHealth(app core.App, e *core.RequestEvent) error {
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

	// Get SSH service for health monitoring
	sshService := ssh.GetSSHService()

	// Get connection health status
	connectionStatus := sshService.GetConnectionStatus()
	healthMetrics := sshService.GetHealthMetrics()

	// Get specific connection keys for this server
	rootKey := sshService.GetConnectionKey(server, true)
	appKey := sshService.GetConnectionKey(server, false)

	response := map[string]interface{}{
		"server_id":       serverID,
		"server_name":     server.Name,
		"host":            server.Host,
		"port":            server.Port,
		"security_locked": server.SecurityLocked,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"connections": map[string]interface{}{
			"root": getConnectionDetails(connectionStatus, rootKey, server.SecurityLocked),
			"app":  getConnectionDetails(connectionStatus, appKey, false),
		},
		"overall_metrics": map[string]interface{}{
			"total_connections":     healthMetrics.TotalConnections,
			"healthy_connections":   healthMetrics.HealthyConnections,
			"unhealthy_connections": healthMetrics.UnhealthyConnections,
			"average_response_time": healthMetrics.AverageResponseTime.String(),
			"error_rate":            fmt.Sprintf("%.2f%%", healthMetrics.ErrorRate*100),
			"last_update":           healthMetrics.LastUpdate.Format(time.RFC3339),
		},
	}

	app.Logger().Info("Connection health status retrieved",
		"server_id", serverID,
		"host", server.Host,
		"security_locked", server.SecurityLocked)

	return e.JSON(http.StatusOK, response)
}

// getConnectionDetails extracts connection details from status map
func getConnectionDetails(connectionStatus map[string]ssh.ConnectionHealthStatus, key string, expectDisabled bool) map[string]interface{} {
	if status, exists := connectionStatus[key]; exists {
		return map[string]interface{}{
			"exists":        true,
			"healthy":       status.Healthy,
			"last_used":     status.LastUsed.Format(time.RFC3339),
			"age":           status.Age.String(),
			"use_count":     status.UseCount,
			"response_time": status.ResponseTime.String(),
			"last_error":    status.LastError,
		}
	} else if expectDisabled {
		return map[string]interface{}{
			"exists":   false,
			"healthy":  false,
			"disabled": true,
			"reason":   "Root connections are disabled after security lockdown",
		}
	} else {
		return map[string]interface{}{
			"exists":  false,
			"healthy": false,
			"reason":  "Connection not established or not in pool",
		}
	}
}
