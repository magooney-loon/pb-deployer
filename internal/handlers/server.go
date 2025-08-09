package handlers

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
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

	// TODO: Implement server setup logic
	app.Logger().Info("Server setup requested", "server_id", serverID)

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

	// TODO: Implement security lockdown logic
	app.Logger().Info("Security lockdown requested", "server_id", serverID)

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
