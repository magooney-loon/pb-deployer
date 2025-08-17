package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pb-deployer/internal/tunnel"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterSetupHandlers registers server setup API endpoints
func RegisterSetupHandlers(e *core.ServeEvent, app core.App) {
	e.Router.POST("/api/setup/server", func(c *core.RequestEvent) error {
		return handleServerSetup(c, app)
	})

	e.Router.POST("/api/setup/security", func(c *core.RequestEvent) error {
		return handleServerSecurity(c, app)
	})

	e.Router.POST("/api/setup/validate", func(c *core.RequestEvent) error {
		return handleServerValidation(c, app)
	})
}

// handleServerSetup sets up a PocketBase server
func handleServerSetup(c *core.RequestEvent, app core.App) error {
	type setupRequest struct {
		Host       string   `json:"host"`
		Port       int      `json:"port"`
		User       string   `json:"user"`
		Username   string   `json:"username"`
		PublicKeys []string `json:"public_keys"`
	}

	var req setupRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	if !tunnel.IsAgentAvailable() {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}

	// Create SSH client
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()

	// Connect to server
	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}

	// Validate SSH connection
	if err := validateSSHConnection(client); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("SSH connection validation failed: %v", err),
		})
	}

	// Create managers
	manager := tunnel.NewManager(client)
	setupManager := tunnel.NewSetupManager(manager)

	// Setup PocketBase server
	err = setupManager.SetupPocketBaseServer(req.Username, req.PublicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Server setup failed: %v", err),
		})
	}

	// Verify setup
	if err := setupManager.VerifySetup(req.Username); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Setup verification failed: %v", err),
		})
	}

	// Get setup info
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to get setup info: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": "Server setup completed successfully",
		"setup_info": map[string]any{
			"os":               setupInfo.OS,
			"architecture":     setupInfo.Architecture,
			"hostname":         setupInfo.Hostname,
			"pocketbase_setup": setupInfo.PocketBaseSetup,
			"installed_apps":   setupInfo.InstalledApps,
		},
	})
}

// handleServerSecurity applies security hardening to the server
func handleServerSecurity(c *core.RequestEvent, app core.App) error {
	type securityRequest struct {
		Host           string                `json:"host"`
		Port           int                   `json:"port"`
		User           string                `json:"user"`
		FirewallRules  []tunnel.FirewallRule `json:"firewall_rules"`
		SSHConfig      *tunnel.SSHConfig     `json:"ssh_config"`
		EnableFail2ban bool                  `json:"enable_fail2ban"`
	}

	var req securityRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	if !tunnel.IsAgentAvailable() {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}

	// Create SSH client
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()

	// Connect to server
	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}

	// Create security manager
	manager := tunnel.NewManager(client)
	securityManager := tunnel.NewSecurityManager(manager)

	// Use default firewall rules if none provided
	if len(req.FirewallRules) == 0 {
		req.FirewallRules = securityManager.GetDefaultPocketBaseRules()
	}

	// Use default SSH config if none provided
	var sshConfig tunnel.SSHConfig
	if req.SSHConfig != nil {
		sshConfig = *req.SSHConfig
	} else {
		sshConfig = securityManager.GetDefaultSSHConfig()
	}

	// Apply security configuration
	securityConfig := tunnel.SecurityConfig{
		FirewallRules:  req.FirewallRules,
		HardenSSH:      true,
		SSHConfig:      sshConfig,
		EnableFail2ban: req.EnableFail2ban,
	}

	err = securityManager.SecureServer(securityConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Security hardening failed: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"message": "Server security hardening completed successfully",
		"applied_config": map[string]any{
			"firewall_rules":   req.FirewallRules,
			"ssh_hardened":     true,
			"fail2ban_enabled": req.EnableFail2ban,
		},
	})
}

// handleServerValidation validates server setup and configuration
func handleServerValidation(c *core.RequestEvent, app core.App) error {
	type validationRequest struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Username string `json:"username"`
	}

	var req validationRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Host == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	if !tunnel.IsAgentAvailable() {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}

	// Create SSH client
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()

	// Connect to server
	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}

	// Validate SSH connection
	if err := validateSSHConnection(client); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("SSH connection validation failed: %v", err),
		})
	}

	// Create setup manager
	manager := tunnel.NewManager(client)
	setupManager := tunnel.NewSetupManager(manager)

	// Verify setup
	if err := setupManager.VerifySetup(req.Username); err != nil {
		return c.JSON(http.StatusOK, map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Setup validation failed: %v", err),
		})
	}

	// Get setup info
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to get setup info: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"valid":   true,
		"message": "Server setup validation passed",
		"setup_info": map[string]any{
			"os":               setupInfo.OS,
			"architecture":     setupInfo.Architecture,
			"hostname":         setupInfo.Hostname,
			"pocketbase_setup": setupInfo.PocketBaseSetup,
			"installed_apps":   setupInfo.InstalledApps,
		},
	})
}

// createSSHClient creates and configures an SSH client
func createSSHClient(host string, port int, user string) (*tunnel.Client, error) {
	config := tunnel.Config{
		Host:       host,
		Port:       port,
		User:       user,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 5 * time.Second,
	}

	return tunnel.NewClient(config)
}

// validateSSHConnection validates the SSH connection is working
func validateSSHConnection(client *tunnel.Client) error {
	// Test connection with a simple ping
	if err := client.Ping(); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Get host info to verify we can execute commands
	hostInfo, err := client.HostInfo()
	if err != nil {
		return fmt.Errorf("failed to get host info: %w", err)
	}

	if strings.TrimSpace(hostInfo) == "" {
		return fmt.Errorf("empty host info response")
	}

	return nil
}

// getPublicKeysForSetup extracts public keys from the request
func getPublicKeysForSetup(publicKeys []string) []string {
	var validKeys []string

	for _, key := range publicKeys {
		key = strings.TrimSpace(key)
		if key != "" && (strings.HasPrefix(key, "ssh-") || strings.HasPrefix(key, "ecdsa-") || strings.HasPrefix(key, "ssh-ed25519")) {
			validKeys = append(validKeys, key)
		}
	}

	return validKeys
}
