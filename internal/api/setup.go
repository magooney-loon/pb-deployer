package api

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"pb-deployer/internal/models"
	"pb-deployer/internal/tunnel"

	"github.com/pocketbase/pocketbase/core"
)

// DeploymentMode represents different deployment environments
type DeploymentMode string

const (
	DeploymentModeDevelopment DeploymentMode = "development"
	DeploymentModeStaging     DeploymentMode = "staging"
	DeploymentModeProduction  DeploymentMode = "production"
)

// getDeploymentMode returns the deployment mode from environment or defaults to development
func getDeploymentMode() DeploymentMode {
	mode := os.Getenv("DEPLOYMENT_MODE")
	switch mode {
	case "staging":
		return DeploymentModeStaging
	case "production":
		return DeploymentModeProduction
	default:
		return DeploymentModeDevelopment
	}
}

func RegisterSetupHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Server setup endpoint
		e.Router.POST("/api/servers/{id}/setup", func(c *core.RequestEvent) error {
			return handleServerSetup(c, app)
		})

		// Server security endpoint
		e.Router.POST("/api/servers/{id}/security", func(c *core.RequestEvent) error {
			return handleServerSecurity(c, app)
		})

		// Server connection validation endpoint
		e.Router.POST("/api/servers/{id}/validate", func(c *core.RequestEvent) error {
			return handleServerValidation(c, app)
		})

		return e.Next()
	})
}

func handleServerSetup(c *core.RequestEvent, app core.App) error {
	serverID := c.Request.PathValue("id")
	if serverID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Server not found",
		})
	}

	// Convert to server model
	server := &models.Server{
		ID:            record.Id,
		Name:          record.GetString("name"),
		Host:          record.GetString("host"),
		Port:          record.GetInt("port"),
		RootUsername:  record.GetString("root_username"),
		AppUsername:   record.GetString("app_username"),
		UseSSHAgent:   record.GetBool("use_ssh_agent"),
		ManualKeyPath: record.GetString("manual_key_path"),
		SetupComplete: record.GetBool("setup_complete"),
	}

	// Check if already setup
	if server.SetupComplete {
		return c.JSON(http.StatusConflict, map[string]any{
			"error": "Server setup already completed",
		})
	}

	// Create SSH client optimized for setup
	client, err := createSSHClientForSetup(server)
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

	// Create managers
	mgr := tunnel.NewManager(client)
	setupMgr := tunnel.NewSetupManager(mgr)

	// Get public keys for setup
	publicKeys, err := getPublicKeysForSetup(server)
	if err != nil {
		app.Logger().Warn("Failed to get public keys", "error", err)
		publicKeys = []string{} // Continue without keys
	}

	if len(publicKeys) > 0 {
		app.Logger().Info("Found public keys for setup", "count", len(publicKeys))
	} else {
		app.Logger().Info("No public keys found, setup will continue without SSH key configuration")
	}

	// Run the setup
	err = setupMgr.SetupPocketBaseServer(server.AppUsername, publicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Server setup failed: %v", err),
		})
	}

	// Verify setup
	err = setupMgr.VerifySetup(server.AppUsername)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Setup verification failed: %v", err),
		})
	}

	// Update server status in database
	record.Set("setup_complete", true)
	record.Set("updated", time.Now())

	if err := app.Save(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to update server status: %v", err),
		})
	}

	// Get setup info
	setupInfo, err := setupMgr.GetSetupInfo()
	if err != nil {
		app.Logger().Warn("Failed to get setup info", "error", err)
		setupInfo = &tunnel.SetupInfo{}
	}

	return c.JSON(http.StatusOK, map[string]any{
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

func handleServerSecurity(c *core.RequestEvent, app core.App) error {
	serverID := c.Request.PathValue("id")
	if serverID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Server not found",
		})
	}

	// Convert to server model
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

	// Check if setup is complete
	if !server.SetupComplete {
		return c.JSON(http.StatusPreconditionFailed, map[string]any{
			"error": "Server setup must be completed before security lockdown",
		})
	}

	// Check if already secured
	if server.SecurityLocked {
		return c.JSON(http.StatusConflict, map[string]any{
			"error": "Server security already locked down",
		})
	}

	// Create SSH client with strict security for lockdown
	client, err := createSSHClientSecure(server)
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

	// Create managers
	mgr := tunnel.NewManager(client)
	securityMgr := tunnel.NewSecurityManager(mgr)

	// Configure security
	securityConfig := tunnel.SecurityConfig{
		FirewallRules:  securityMgr.GetDefaultPocketBaseRules(),
		HardenSSH:      true,
		SSHConfig:      securityMgr.GetDefaultSSHConfig(),
		EnableFail2ban: true,
	}

	// Add app username to SSH allowed users
	securityConfig.SSHConfig.AllowUsers = []string{server.AppUsername}

	// Apply security configuration
	err = securityMgr.SecureServer(securityConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Security lockdown failed: %v", err),
		})
	}

	// Update server status in database
	record.Set("security_locked", true)
	record.Set("updated", time.Now())

	if err := app.Save(record); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to update server status: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Server security lockdown completed successfully",
		"security_config": map[string]any{
			"firewall_rules": len(securityConfig.FirewallRules),
			"ssh_hardened":   securityConfig.HardenSSH,
			"fail2ban":       securityConfig.EnableFail2ban,
			"allowed_users":  securityConfig.SSHConfig.AllowUsers,
		},
	})
}

func handleServerValidation(c *core.RequestEvent, app core.App) error {
	serverID := c.Request.PathValue("id")
	if serverID == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Server ID is required",
		})
	}

	// Get server from database
	record, err := app.FindRecordById("servers", serverID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error": "Server not found",
		})
	}

	// Convert to server model
	server := &models.Server{
		ID:            record.Id,
		Name:          record.GetString("name"),
		Host:          record.GetString("host"),
		Port:          record.GetInt("port"),
		RootUsername:  record.GetString("root_username"),
		UseSSHAgent:   record.GetBool("use_ssh_agent"),
		ManualKeyPath: record.GetString("manual_key_path"),
	}

	// Validate SSH connection
	err = validateSSHConnection(server)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error":    "SSH connection validation failed",
			"details":  err.Error(),
			"host":     server.Host,
			"port":     server.Port,
			"username": server.RootUsername,
			"auth_method": func() string {
				if server.UseSSHAgent {
					return "ssh_agent"
				}
				return "private_key"
			}(),
		})
	}

	// Get additional connection info
	client, err := createSSHClient(server)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]any{
			"error": fmt.Sprintf("Failed to connect: %v", err),
		})
	}

	hostInfo, err := client.HostInfo()
	if err != nil {
		app.Logger().Warn("Failed to get host info", "error", err)
		hostInfo = "Unknown"
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "SSH connection validation successful",
		"connection_info": map[string]any{
			"host":            server.Host,
			"port":            server.Port,
			"username":        server.RootUsername,
			"host_info":       hostInfo,
			"deployment_mode": getDeploymentMode(),
			"auth_method": func() string {
				if server.UseSSHAgent {
					if tunnel.IsAgentAvailable() {
						return "ssh_agent_available"
					}
					return "ssh_agent_unavailable"
				}
				return "private_key"
			}(),
		},
	})
}

func createSSHClient(server *models.Server) (*tunnel.Client, error) {
	return createSSHClientWithMode(server, getDeploymentMode())
}

// createSSHClientWithMode creates an SSH client configured for specific deployment mode
func createSSHClientWithMode(server *models.Server, mode DeploymentMode) (*tunnel.Client, error) {
	var authConfig tunnel.AuthConfig

	// Configure host key verification based on deployment mode and environment
	hostKeyConfig := getHostKeyConfigForMode(mode)

	// Check for environment variable overrides
	if knownHostsFile := os.Getenv("SSH_KNOWN_HOSTS_FILE"); knownHostsFile != "" {
		hostKeyConfig.KnownHostsFile = knownHostsFile
	}

	if strictHostCheck := os.Getenv("SSH_STRICT_HOST_KEY_CHECKING"); strictHostCheck != "" {
		if strictHostCheck == "yes" || strictHostCheck == "true" {
			hostKeyConfig.Mode = tunnel.HostKeyModeStrict
			hostKeyConfig.StrictHostKeyChecking = true
			hostKeyConfig.AcceptNewKeys = false
		} else if strictHostCheck == "no" || strictHostCheck == "false" {
			hostKeyConfig.Mode = tunnel.HostKeyModeAcceptNew
			hostKeyConfig.StrictHostKeyChecking = false
			hostKeyConfig.AcceptNewKeys = true
		}
	}

	if server.UseSSHAgent {
		// Use SSH agent for authentication
		if !tunnel.IsAgentAvailable() {
			return nil, fmt.Errorf("SSH agent not available")
		}
		authConfig = tunnel.AuthConfigWithAgent()
		authConfig.HostKeyVerification = hostKeyConfig
	} else if server.ManualKeyPath != "" {
		// Use manual key path
		if err := tunnel.ValidateKeyFile(server.ManualKeyPath); err != nil {
			return nil, fmt.Errorf("invalid key file: %w", err)
		}

		// Check for passphrase in environment
		passphrase := os.Getenv("SSH_KEY_PASSPHRASE")
		if passphrase != "" {
			authConfig = tunnel.AuthConfigFromKeyPath(server.ManualKeyPath, passphrase)
		} else {
			authConfig = tunnel.AuthConfigFromKeyPath(server.ManualKeyPath)
		}
		authConfig.HostKeyVerification = hostKeyConfig
	} else {
		return nil, fmt.Errorf("no authentication method configured")
	}

	// Configure connection timeouts based on deployment mode
	timeout := 30 * time.Second
	retryCount := 3
	retryDelay := 5 * time.Second

	if mode == DeploymentModeProduction {
		// More conservative timeouts for production
		timeout = 45 * time.Second
		retryCount = 5
		retryDelay = 10 * time.Second
	} else if mode == DeploymentModeDevelopment {
		// Faster timeouts for development
		timeout = 15 * time.Second
		retryCount = 2
		retryDelay = 3 * time.Second
	}

	// Allow environment override of timeouts
	if envTimeout := os.Getenv("SSH_TIMEOUT_SECONDS"); envTimeout != "" {
		if parsedTimeout, err := time.ParseDuration(envTimeout + "s"); err == nil {
			timeout = parsedTimeout
		}
	}

	config := tunnel.Config{
		Host:       server.Host,
		Port:       server.Port,
		User:       server.RootUsername,
		Auth:       authConfig,
		Timeout:    timeout,
		RetryCount: retryCount,
		RetryDelay: retryDelay,
	}

	return tunnel.NewClient(config)
}

// getHostKeyConfigForMode returns appropriate host key configuration for deployment mode
func getHostKeyConfigForMode(mode DeploymentMode) tunnel.HostKeyConfig {
	switch mode {
	case DeploymentModeProduction:
		// Production: Strict host key checking
		return tunnel.HostKeyConfig{
			Mode:                  tunnel.HostKeyModeStrict,
			StrictHostKeyChecking: true,
			AcceptNewKeys:         false,
			KnownHostsFile:        "", // Use default
		}
	case DeploymentModeStaging:
		// Staging: Accept new keys but log warnings
		return tunnel.HostKeyConfig{
			Mode:                  tunnel.HostKeyModeAcceptNew,
			StrictHostKeyChecking: false,
			AcceptNewKeys:         true,
			KnownHostsFile:        "", // Use default
		}
	default: // Development
		// Development: Accept new keys for ease of use
		return tunnel.HostKeyConfig{
			Mode:                  tunnel.HostKeyModeAcceptNew,
			StrictHostKeyChecking: false,
			AcceptNewKeys:         true,
			KnownHostsFile:        "", // Use default
		}
	}
}

// createSSHClientSecure creates an SSH client with strict host key verification
// Use this for post-setup connections when security is paramount
func createSSHClientSecure(server *models.Server) (*tunnel.Client, error) {
	return createSSHClientWithMode(server, DeploymentModeProduction)
}

// createSSHClientForSetup creates an SSH client optimized for initial server setup
func createSSHClientForSetup(server *models.Server) (*tunnel.Client, error) {
	// For setup, we typically want to accept new host keys
	return createSSHClientWithMode(server, DeploymentModeDevelopment)
}

// validateSSHConnection tests the SSH connection without performing setup
func validateSSHConnection(server *models.Server) error {
	client, err := createSSHClient(server)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if err := client.Ping(); err != nil {
		return fmt.Errorf("connection ping failed: %w", err)
	}

	return nil
}

// Helper function to extract public keys for setup
func getPublicKeysForSetup(server *models.Server) ([]string, error) {
	if server.UseSSHAgent {
		// For SSH agent, we'll extract keys using the auth system
		if !tunnel.IsAgentAvailable() {
			return []string{}, fmt.Errorf("SSH agent not available")
		}
		return []string{}, nil

	} else if server.ManualKeyPath != "" {
		// Try to read corresponding .pub file
		pubKeyPath := server.ManualKeyPath + ".pub"

		// Validate the private key first
		if err := tunnel.ValidateKeyFile(server.ManualKeyPath); err != nil {
			return []string{}, fmt.Errorf("invalid private key: %w", err)
		}

		// Try to read the public key file
		pubKeyData, err := os.ReadFile(pubKeyPath)
		if err != nil {
			// If .pub file doesn't exist, we can't extract the public key easily
			// Log warning but continue without public key
			return []string{}, fmt.Errorf("public key file not found: %s", pubKeyPath)
		}

		// Return the public key content as a single-item slice
		pubKeyContent := strings.TrimSpace(string(pubKeyData))
		if pubKeyContent == "" {
			return []string{}, fmt.Errorf("public key file is empty")
		}

		return []string{pubKeyContent}, nil
	}

	return []string{}, fmt.Errorf("no authentication method configured")
}
