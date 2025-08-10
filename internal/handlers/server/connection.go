package server

import (
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
		"port", server.Port)

	// Test TCP connectivity first
	tcpResult := testTCPConnectionEnhanced(server.Host, server.Port)

	// Test SSH connections
	rootSSHResult := testSSHConnection(server, true, app) // as root
	appSSHResult := testSSHConnection(server, false, app) // as app user

	// Determine overall success
	overallSuccess := tcpResult.Success && rootSSHResult.Success && appSSHResult.Success
	overallStatus := "healthy"

	if !tcpResult.Success {
		overallStatus = "unreachable"
	} else if !rootSSHResult.Success && !appSSHResult.Success {
		overallStatus = "ssh_failed"
	} else if !rootSSHResult.Success {
		overallStatus = "root_ssh_failed"
	} else if !appSSHResult.Success {
		overallStatus = "app_ssh_failed"
	}

	response := ConnectionTestResponse{
		Success:           overallSuccess,
		TCPConnection:     tcpResult,
		RootSSHConnection: rootSSHResult,
		AppSSHConnection:  appSSHResult,
		OverallStatus:     overallStatus,
	}

	if !overallSuccess {
		response.Error = fmt.Sprintf("Connection issues detected: %s", overallStatus)
	}

	app.Logger().Info("Connection test completed",
		"server_id", serverID,
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

// testSSHConnection tests SSH connectivity for a specific user
func testSSHConnection(server *models.Server, asRoot bool, app core.App) SSHTestResult {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	result := SSHTestResult{
		Username: username,
	}

	// Attempt SSH connection
	sshManager, err := ssh.NewSSHManager(server, asRoot)
	if err != nil {
		result.Error = fmt.Sprintf("SSH connection failed: %v", err)
		app.Logger().Debug("SSH connection failed",
			"username", username,
			"host", server.Host,
			"error", err)
		return result
	}
	defer sshManager.Close()

	// Test a simple command to verify the connection works
	err = sshManager.RunCommand("echo 'connection_test'")
	if err != nil {
		result.Error = fmt.Sprintf("SSH command test failed: %v", err)
		app.Logger().Debug("SSH command test failed",
			"username", username,
			"host", server.Host,
			"error", err)
		return result
	}

	result.Success = true

	// Determine auth method used
	if server.UseSSHAgent {
		result.AuthMethod = "ssh_agent"
	} else if server.ManualKeyPath != "" {
		result.AuthMethod = "private_key"
	} else {
		result.AuthMethod = "default_keys"
	}

	app.Logger().Debug("SSH connection successful",
		"username", username,
		"host", server.Host,
		"auth_method", result.AuthMethod)

	return result
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
