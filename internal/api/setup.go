package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	log.Printf("[SetupAPI] Starting server setup process")

	type setupRequest struct {
		Host       string   `json:"host"`
		Port       int      `json:"port"`
		User       string   `json:"user"`
		Username   string   `json:"username"`
		PublicKeys []string `json:"public_keys"`
	}

	var req setupRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Printf("[SetupAPI] Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Printf("[SetupAPI] Received setup request for host: %s, port: %d, user: %s, username: %s", req.Host, req.Port, req.User, req.Username)

	// Validate required fields
	if req.Host == "" {
		log.Printf("[SetupAPI] Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Printf("[SetupAPI] Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		log.Printf("[SetupAPI] Validation failed: Username is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	log.Printf("[SetupAPI] Checking SSH agent availability")
	if !tunnel.IsAgentAvailable() {
		log.Printf("[SetupAPI] SSH agent is not available")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}
	log.Printf("[SetupAPI] SSH agent is available")

	// Create SSH client
	log.Printf("[SetupAPI] Creating SSH client for %s@%s:%d", req.User, req.Host, req.Port)
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		log.Printf("[SetupAPI] Failed to create SSH client: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()
	log.Printf("[SetupAPI] SSH client created successfully")

	// Connect to server
	log.Printf("[SetupAPI] Attempting to connect to server...")
	if err := client.Connect(); err != nil {
		// Check if this is a host key unknown error
		if strings.Contains(err.Error(), "key is unknown") {
			log.Printf("[SetupAPI] Host key unknown, attempting to add manually: %v", err)

			// Try to add the host key manually
			if addErr := addHostKeyManually(req.Host, req.Port); addErr != nil {
				log.Printf("[SetupAPI] Failed to add host key manually: %v", addErr)
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error":   fmt.Sprintf("Host key unknown and failed to add automatically: %v", addErr),
					"details": err.Error(),
				})
			}

			// Retry connection after adding host key
			log.Printf("[SetupAPI] Host key added, retrying connection...")
			if retryErr := client.Connect(); retryErr != nil {
				log.Printf("[SetupAPI] Connection still failed after adding host key: %v", retryErr)
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error":   fmt.Sprintf("Connection failed even after adding host key: %v", retryErr),
					"details": err.Error(),
				})
			}
			log.Printf("[SetupAPI] Successfully connected after adding host key")
		} else if strings.Contains(err.Error(), "illegal base64 data") || strings.Contains(err.Error(), "knownhosts:") {
			log.Printf("[SetupAPI] Connection failed due to known_hosts corruption: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error":   "SSH connection failed due to corrupted known_hosts file. Please check line 14 in ~/.ssh/known_hosts or remove the corrupted entry.",
				"details": err.Error(),
			})
		} else {
			log.Printf("[SetupAPI] Failed to connect to server: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": fmt.Sprintf("Failed to connect to server: %v", err),
			})
		}
	}
	log.Printf("[SetupAPI] Successfully connected to server")

	// Validate SSH connection
	log.Printf("[SetupAPI] Validating SSH connection...")
	if err := validateSSHConnection(client); err != nil {
		log.Printf("[SetupAPI] SSH connection validation failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("SSH connection validation failed: %v", err),
		})
	}
	log.Printf("[SetupAPI] SSH connection validation successful")

	// Create managers
	log.Printf("[SetupAPI] Creating tunnel manager and setup manager")
	manager := tunnel.NewManager(client)
	setupManager := tunnel.NewSetupManager(manager)

	// Setup PocketBase server
	log.Printf("[SetupAPI] Starting PocketBase server setup for user: %s", req.Username)
	err = setupManager.SetupPocketBaseServer(req.Username, req.PublicKeys)
	if err != nil {
		log.Printf("[SetupAPI] Server setup failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Server setup failed: %v", err),
		})
	}
	log.Printf("[SetupAPI] PocketBase server setup completed")

	// Verify setup
	log.Printf("[SetupAPI] Verifying setup for user: %s", req.Username)
	if err := setupManager.VerifySetup(req.Username); err != nil {
		log.Printf("[SetupAPI] Setup verification failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Setup verification failed: %v", err),
		})
	}
	log.Printf("[SetupAPI] Setup verification successful")

	// Get setup info
	log.Printf("[SetupAPI] Getting setup info...")
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		log.Printf("[SetupAPI] Failed to get setup info: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to get setup info: %v", err),
		})
	}
	log.Printf("[SetupAPI] Setup info retrieved successfully")

	log.Printf("[SetupAPI] Server setup completed successfully for %s@%s", req.User, req.Host)
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
	log.Printf("[SecurityAPI] Starting server security hardening process")

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
		log.Printf("[SecurityAPI] Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Printf("[SecurityAPI] Received security request for host: %s, port: %d, user: %s", req.Host, req.Port, req.User)

	// Validate required fields
	if req.Host == "" {
		log.Printf("[SecurityAPI] Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Printf("[SecurityAPI] Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	log.Printf("[SecurityAPI] Checking SSH agent availability")
	if !tunnel.IsAgentAvailable() {
		log.Printf("[SecurityAPI] SSH agent is not available")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}
	log.Printf("[SecurityAPI] SSH agent is available")

	// Create SSH client
	log.Printf("[SecurityAPI] Creating SSH client for %s@%s:%d", req.User, req.Host, req.Port)
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		log.Printf("[SecurityAPI] Failed to create SSH client: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()
	log.Printf("[SecurityAPI] SSH client created successfully")

	// Connect to server
	log.Printf("[SecurityAPI] Attempting to connect to server...")
	if err := client.Connect(); err != nil {
		log.Printf("[SecurityAPI] Failed to connect to server: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}
	log.Printf("[SecurityAPI] Successfully connected to server")

	// Create security manager
	log.Printf("[SecurityAPI] Creating tunnel manager and security manager")
	manager := tunnel.NewManager(client)
	securityManager := tunnel.NewSecurityManager(manager)

	// Use default firewall rules if none provided
	if len(req.FirewallRules) == 0 {
		log.Printf("[SecurityAPI] No firewall rules provided, using defaults")
		req.FirewallRules = securityManager.GetDefaultPocketBaseRules()
	} else {
		log.Printf("[SecurityAPI] Using %d custom firewall rules", len(req.FirewallRules))
	}

	// Use default SSH config if none provided
	var sshConfig tunnel.SSHConfig
	if req.SSHConfig != nil {
		log.Printf("[SecurityAPI] Using custom SSH config")
		sshConfig = *req.SSHConfig
	} else {
		log.Printf("[SecurityAPI] Using default SSH config")
		sshConfig = securityManager.GetDefaultSSHConfig()
	}

	// Apply security configuration
	log.Printf("[SecurityAPI] Applying security configuration (fail2ban: %v)", req.EnableFail2ban)
	securityConfig := tunnel.SecurityConfig{
		FirewallRules:  req.FirewallRules,
		HardenSSH:      true,
		SSHConfig:      sshConfig,
		EnableFail2ban: req.EnableFail2ban,
	}

	err = securityManager.SecureServer(securityConfig)
	if err != nil {
		log.Printf("[SecurityAPI] Security hardening failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Security hardening failed: %v", err),
		})
	}
	log.Printf("[SecurityAPI] Security hardening completed successfully")

	log.Printf("[SecurityAPI] Server security hardening completed successfully for %s@%s", req.User, req.Host)
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
	log.Printf("[ValidationAPI] Starting server validation process")

	type validationRequest struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Username string `json:"username"`
	}

	var req validationRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Printf("[ValidationAPI] Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Printf("[ValidationAPI] Received validation request for host: %s, port: %d, user: %s, username: %s", req.Host, req.Port, req.User, req.Username)

	// Validate required fields
	if req.Host == "" {
		log.Printf("[ValidationAPI] Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Printf("[ValidationAPI] Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		log.Printf("[ValidationAPI] Validation failed: Username is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	// Set default port
	if req.Port == 0 {
		req.Port = 22
	}

	// Check SSH agent availability
	log.Printf("[ValidationAPI] Checking SSH agent availability")
	if !tunnel.IsAgentAvailable() {
		log.Printf("[ValidationAPI] SSH agent is not available")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}
	log.Printf("[ValidationAPI] SSH agent is available")

	// Create SSH client
	log.Printf("[ValidationAPI] Creating SSH client for %s@%s:%d", req.User, req.Host, req.Port)
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		log.Printf("[ValidationAPI] Failed to create SSH client: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}
	defer client.Close()
	log.Printf("[ValidationAPI] SSH client created successfully")

	// Connect to server
	log.Printf("[ValidationAPI] Attempting to connect to server...")
	if err := client.Connect(); err != nil {
		log.Printf("[ValidationAPI] Failed to connect to server: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}
	log.Printf("[ValidationAPI] Successfully connected to server")

	// Validate SSH connection
	log.Printf("[ValidationAPI] Validating SSH connection...")
	if err := validateSSHConnection(client); err != nil {
		log.Printf("[ValidationAPI] SSH connection validation failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("SSH connection validation failed: %v", err),
		})
	}
	log.Printf("[ValidationAPI] SSH connection validation successful")

	// Create setup manager
	log.Printf("[ValidationAPI] Creating tunnel manager and setup manager")
	manager := tunnel.NewManager(client)
	setupManager := tunnel.NewSetupManager(manager)

	// Verify setup
	log.Printf("[ValidationAPI] Verifying setup for user: %s", req.Username)
	if err := setupManager.VerifySetup(req.Username); err != nil {
		log.Printf("[ValidationAPI] Setup validation failed: %v", err)
		return c.JSON(http.StatusOK, map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Setup validation failed: %v", err),
		})
	}
	log.Printf("[ValidationAPI] Setup verification successful")

	// Get setup info
	log.Printf("[ValidationAPI] Getting setup info...")
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		log.Printf("[ValidationAPI] Failed to get setup info: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to get setup info: %v", err),
		})
	}
	log.Printf("[ValidationAPI] Setup info retrieved successfully")

	log.Printf("[ValidationAPI] Server validation completed successfully for %s@%s", req.User, req.Host)
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

// addHostKeyManually adds a host key using ssh-keyscan
func addHostKeyManually(host string, port int) error {
	log.Printf("[SetupAPI] Adding host key manually for %s:%d", host, port)

	// Use ssh-keyscan to get the host key
	cmd := fmt.Sprintf("ssh-keyscan -p %d %s", port, host)

	// Execute ssh-keyscan locally
	result, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("failed to scan host key: %w", err)
	}

	if len(result) == 0 {
		return fmt.Errorf("no host key found for %s", host)
	}

	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Append to known_hosts
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	// Write the host key
	_, err = file.Write(result)
	if err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}

	log.Printf("[SetupAPI] Successfully added host key for %s to known_hosts", host)
	return nil
}

// createSSHClient creates and configures an SSH client
func createSSHClient(host string, port int, user string) (*tunnel.Client, error) {
	log.Printf("[SetupAPI] Creating SSH client config: host=%s, port=%d, user=%s", host, port, user)
	config := tunnel.Config{
		Host:       host,
		Port:       port,
		User:       user,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 5 * time.Second,
	}

	client, err := tunnel.NewClient(config)
	if err != nil {
		// Check if this is a known_hosts corruption error
		if strings.Contains(err.Error(), "illegal base64 data") || strings.Contains(err.Error(), "knownhosts:") {
			log.Printf("[SetupAPI] Detected known_hosts corruption, attempting to clean: %v", err)

			// Try to clean the known_hosts file
			if cleanErr := tunnel.CleanKnownHostsFile(""); cleanErr != nil {
				log.Printf("[SetupAPI] Failed to clean known_hosts file: %v", cleanErr)
				return nil, fmt.Errorf("known_hosts file corrupted and cleanup failed: %w", err)
			}

			log.Printf("[SetupAPI] Successfully cleaned known_hosts file, retrying client creation")

			// Retry creating the client after cleaning
			client, err = tunnel.NewClient(config)
			if err != nil {
				log.Printf("[SetupAPI] Failed to create tunnel client after cleanup: %v", err)
				return nil, err
			}
		} else {
			log.Printf("[SetupAPI] Failed to create tunnel client: %v", err)
			return nil, err
		}
	}

	log.Printf("[SetupAPI] SSH client created with config successfully")
	return client, nil
}

// validateSSHConnection validates the SSH connection is working
func validateSSHConnection(client *tunnel.Client) error {
	// Test connection with a simple ping
	log.Printf("[SetupAPI] Testing SSH connection with ping...")
	if err := client.Ping(); err != nil {
		log.Printf("[SetupAPI] SSH ping failed: %v", err)
		return fmt.Errorf("ping failed: %w", err)
	}
	log.Printf("[SetupAPI] SSH ping successful")

	// Get host info to verify we can execute commands
	log.Printf("[SetupAPI] Getting host info to verify command execution...")
	hostInfo, err := client.HostInfo()
	if err != nil {
		log.Printf("[SetupAPI] Failed to get host info: %v", err)
		return fmt.Errorf("failed to get host info: %w", err)
	}

	if strings.TrimSpace(hostInfo) == "" {
		log.Printf("[SetupAPI] Received empty host info response")
		return fmt.Errorf("empty host info response")
	}

	log.Printf("[SetupAPI] Host info received: %s", strings.TrimSpace(hostInfo))
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
