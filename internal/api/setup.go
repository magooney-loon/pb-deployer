package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/tunnel"

	"github.com/pocketbase/pocketbase/core"
)

func handleServerSetup(c *core.RequestEvent, app core.App) error {
	log := logger.GetAPILogger()
	log.Info("Starting server setup process")

	type setupRequest struct {
		Host       string   `json:"host"`
		Port       int      `json:"port"`
		User       string   `json:"user"`
		Username   string   `json:"username"`
		PublicKeys []string `json:"public_keys"`
	}

	sendStep := func(step int, message string) {
		log.Step(step, 6, message)
	}

	var req setupRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Info("Received setup request for host: %s, port: %d, user: %s, username: %s", req.Host, req.Port, req.User, req.Username)

	if req.Host == "" {
		log.Error("Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Error("Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		log.Error("Validation failed: Username is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	if req.Port == 0 {
		req.Port = 22
	}

	sendStep(1, "Checking SSH agent and creating connection")

	if !tunnel.IsAgentAvailable() {
		log.Error("SSH agent is not available")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent required",
		})
	}

	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		log.Error("Failed to create SSH client: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}

	// Setup cleanup manager for proper resource management
	cleanup := tunnel.NewCleanupManager()
	defer cleanup.Close()
	cleanup.AddCloser(client)

	// Add panic recovery for safer cleanup
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic during server setup: %v", r)
			c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "Internal server error during setup",
			})
		}
	}()

	sendStep(2, "Connecting to server")
	if err := client.Connect(); err != nil {
		// Handle host key unknown errors
		if strings.Contains(err.Error(), "key is unknown") {
			if addErr := addHostKeyManually(req.Host, req.Port); addErr != nil {
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error": "Host key verification failed",
				})
			}
			if retryErr := client.Connect(); retryErr != nil {
				return c.JSON(http.StatusInternalServerError, map[string]any{
					"error": "Connection failed after host key addition",
				})
			}
		} else if strings.Contains(err.Error(), "known_hosts") {
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "Corrupted known_hosts file",
			})
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "Connection failed",
			})
		}
	}
	sendStep(3, "Validating SSH connection")
	if err := validateSSHConnection(client); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "SSH validation failed",
		})
	}

	sendStep(4, "Setting up PocketBase server environment")
	manager := tunnel.NewManager(client)
	cleanup.AddCloser(manager)

	setupManager := tunnel.NewSetupManager(manager)
	cleanup.AddCloser(setupManager)
	err = setupManager.SetupPocketBaseServer(req.Username, req.PublicKeys)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Server setup failed",
		})
	}

	sendStep(5, "Verifying setup and gathering system info")
	if err := setupManager.VerifySetup(req.Username); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Setup verification failed",
		})
	}
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Failed to get setup info",
		})
	}

	sendStep(6, "Updating database and finalizing")
	err = updateServerSetupStatus(app, req.Host, true, false)
	if err != nil {
		log.Warning("Failed to update server setup status: %v", err)
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

func handleServerSecurity(c *core.RequestEvent, app core.App) error {
	log := logger.GetAPILogger()
	log.Info("Starting server security hardening process")

	type securityRequest struct {
		Host           string                `json:"host"`
		Port           int                   `json:"port"`
		User           string                `json:"user"`
		FirewallRules  []tunnel.FirewallRule `json:"firewall_rules"`
		SSHConfig      *tunnel.SSHConfig     `json:"ssh_config"`
		EnableFail2ban bool                  `json:"enable_fail2ban"`
	}

	// Step tracking for security process
	sendStep := func(step int, message string) {
		log.Step(step, 4, message)
	}

	var req securityRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Info("Received security request for host: %s, port: %d, user: %s", req.Host, req.Port, req.User)

	if req.Host == "" {
		log.Error("Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Error("Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}

	if req.Port == 0 {
		req.Port = 22
	}

	sendStep(1, "Connecting to server")
	if !tunnel.IsAgentAvailable() {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent required",
		})
	}
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Failed to create SSH client",
		})
	}

	// Setup cleanup manager for proper resource management
	cleanup := tunnel.NewCleanupManager()
	defer cleanup.Close()
	cleanup.AddCloser(client)

	// Add panic recovery for safer cleanup
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic during server security: %v", r)
			c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "Internal server error during security setup",
			})
		}
	}()

	if err := client.Connect(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Failed to connect to server",
		})
	}

	sendStep(2, "Configuring security settings")
	manager := tunnel.NewManager(client)
	cleanup.AddCloser(manager)

	securityManager := tunnel.NewSecurityManager(manager)
	cleanup.AddCloser(securityManager)

	if len(req.FirewallRules) == 0 {
		req.FirewallRules = securityManager.GetDefaultPocketBaseRules()
	}

	var sshConfig tunnel.SSHConfig
	if req.SSHConfig != nil {
		sshConfig = *req.SSHConfig
	} else {
		sshConfig = securityManager.GetDefaultSSHConfig()
	}

	sendStep(3, "Applying firewall, SSH hardening, and fail2ban")
	securityConfig := tunnel.SecurityConfig{
		FirewallRules:  req.FirewallRules,
		HardenSSH:      true,
		SSHConfig:      sshConfig,
		EnableFail2ban: req.EnableFail2ban,
	}

	err = securityManager.SecureServer(securityConfig)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": "Security hardening failed",
		})
	}

	sendStep(4, "Updating database")
	err = updateServerSetupStatus(app, req.Host, false, true)
	if err != nil {
		log.Warning("Failed to update server security status: %v", err)
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

func handleServerValidation(c *core.RequestEvent) error {
	log := logger.GetAPILogger()
	log.Info("Starting server validation process")

	type validationRequest struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Username string `json:"username"`
	}

	var req validationRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		log.Error("Failed to decode request body: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Invalid request body",
		})
	}

	log.Info("Received validation request for host: %s, port: %d, user: %s, username: %s", req.Host, req.Port, req.User, req.Username)

	if req.Host == "" {
		log.Error("Validation failed: Host is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Host is required",
		})
	}
	if req.User == "" {
		log.Error("Validation failed: User is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "User is required",
		})
	}
	if req.Username == "" {
		log.Error("Validation failed: Username is required")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "Username is required",
		})
	}

	if req.Port == 0 {
		req.Port = 22
	}

	log.Debug("Checking SSH agent availability")
	if !tunnel.IsAgentAvailable() {
		log.Error("SSH agent is not available")
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "SSH agent is required but not available",
		})
	}
	log.Debug("SSH agent is available")

	log.Debug("Creating SSH client for %s@%s:%d", req.User, req.Host, req.Port)
	client, err := createSSHClient(req.Host, req.Port, req.User)
	if err != nil {
		log.Error("Failed to create SSH client: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to create SSH client: %v", err),
		})
	}

	// Setup cleanup manager for proper resource management
	cleanup := tunnel.NewCleanupManager()
	defer cleanup.Close()
	cleanup.AddCloser(client)

	// Add panic recovery for safer cleanup
	defer func() {
		if r := recover(); r != nil {
			log.Error("Panic during server validation: %v", r)
			c.JSON(http.StatusInternalServerError, map[string]any{
				"error": "Internal server error during validation",
			})
		}
	}()

	log.Debug("SSH client created successfully")

	log.Debug("Attempting to connect to server...")
	if err := client.Connect(); err != nil {
		log.Error("Failed to connect to server: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to connect to server: %v", err),
		})
	}
	log.Success("Successfully connected to server")

	log.Debug("Validating SSH connection...")
	if err := validateSSHConnection(client); err != nil {
		log.Error("SSH connection validation failed: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("SSH connection validation failed: %v", err),
		})
	}
	log.Success("SSH connection validation successful")

	log.Debug("Creating tunnel manager and setup manager")
	manager := tunnel.NewManager(client)
	cleanup.AddCloser(manager)

	setupManager := tunnel.NewSetupManager(manager)
	cleanup.AddCloser(setupManager)

	log.Debug("Verifying setup for user: %s", req.Username)
	if err := setupManager.VerifySetup(req.Username); err != nil {
		log.Warning("Setup validation failed: %v", err)
		return c.JSON(http.StatusOK, map[string]any{
			"valid": false,
			"error": fmt.Sprintf("Setup validation failed: %v", err),
		})
	}
	log.Success("Setup verification successful")

	log.Debug("Getting setup info...")
	setupInfo, err := setupManager.GetSetupInfo()
	if err != nil {
		log.Error("Failed to get setup info: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"error": fmt.Sprintf("Failed to get setup info: %v", err),
		})
	}
	log.Debug("Setup info retrieved successfully")

	log.Success("Server validation completed successfully for %s@%s", req.User, req.Host)
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

func addHostKeyManually(host string, port int) error {
	log := logger.GetAPILogger()
	log.Info("Adding host key manually for %s:%d", host, port)

	cmd := fmt.Sprintf("ssh-keyscan -p %d %s", port, host)

	result, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("failed to scan host key: %w", err)
	}

	if len(result) == 0 {
		return fmt.Errorf("no host key found for %s", host)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(result)
	if err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}

	log.Success("Successfully added host key for %s to known_hosts", host)
	return nil
}

func updateServerSetupStatus(app core.App, host string, setupComplete, securityLocked bool) error {

	serverRecord, err := app.FindFirstRecordByFilter(
		"servers",
		"host = {:host}",
		map[string]any{"host": host},
	)
	if err != nil {
		return fmt.Errorf("failed to find server record: %w", err)
	}

	log := logger.GetAPILogger()
	if setupComplete {
		serverRecord.Set("setup_complete", true)
		log.Success("Marking server %s as setup complete", host)
	}
	if securityLocked {
		serverRecord.Set("security_locked", true)
		log.Success("Marking server %s as security locked", host)
	}

	if err := app.Save(serverRecord); err != nil {
		return fmt.Errorf("failed to save server record: %w", err)
	}

	log.Success("Successfully updated server status in database")
	return nil
}

func createSSHClient(host string, port int, user string) (*tunnel.Client, error) {
	log := logger.GetAPILogger()
	log.Debug("Creating SSH client config: host=%s, port=%d, user=%s", host, port, user)

	config := tunnel.Config{
		Host:       host,
		Port:       port,
		User:       user,
		Timeout:    30 * time.Second,
		RetryCount: 3,
		RetryDelay: 5 * time.Second,
	}

	// Function to attempt client creation with proper error handling
	createClient := func() (*tunnel.Client, error) {
		client, err := tunnel.NewClient(config)
		if err != nil {
			// Handle known_hosts corruption with automatic cleanup
			if strings.Contains(err.Error(), "illegal base64 data") ||
				strings.Contains(err.Error(), "knownhosts:") ||
				strings.Contains(err.Error(), "no matching host key") {

				log.Warning("Detected known_hosts issue, attempting to clean: %v", err)

				tunnel.WithCleanup(func() {
					// This cleanup runs regardless of success/failure
					log.Debug("Cleanup function called for known_hosts handling")
				}, func() error {
					if cleanErr := tunnel.CleanKnownHostsFile(""); cleanErr != nil {
						log.Error("Failed to clean known_hosts file: %v", cleanErr)
						return fmt.Errorf("known_hosts file corrupted and cleanup failed: %w", err)
					}

					log.Success("Successfully cleaned known_hosts file, retrying client creation")

					// Retry after cleanup
					retryClient, retryErr := tunnel.NewClient(config)
					if retryErr != nil {
						log.Error("Failed to create tunnel client after cleanup: %v", retryErr)
						return fmt.Errorf("client creation failed after known_hosts cleanup: %w", retryErr)
					}

					client = retryClient
					return nil
				})
			}

			log.Error("Failed to create tunnel client: %v", err)
			return nil, fmt.Errorf("SSH client creation failed: %w", err)
		}

		return client, nil
	}

	// Attempt to create client with error recovery
	client, err := createClient()
	if err != nil {
		return nil, err
	}

	if client == nil {
		return nil, fmt.Errorf("client creation returned nil without error")
	}

	log.Debug("SSH client created with config successfully")
	return client, nil
}

func validateSSHConnection(client *tunnel.Client) error {
	log := logger.GetAPILogger()

	if client == nil {
		return fmt.Errorf("client is nil")
	}

	log.Debug("Testing SSH connection with ping...")
	if err := client.Ping(); err != nil {
		log.Error("SSH ping failed: %v", err)
		return fmt.Errorf("ping failed: %w", err)
	}
	log.Success("SSH ping successful")

	log.Debug("Getting host info to verify command execution...")
	hostInfo, err := client.HostInfo()
	if err != nil {
		log.Error("Failed to get host info: %v", err)
		return fmt.Errorf("failed to get host info: %w", err)
	}

	if strings.TrimSpace(hostInfo) == "" {
		log.Error("Received empty host info response")
		return fmt.Errorf("empty host info response")
	}

	log.Debug("Host info received: %s", strings.TrimSpace(hostInfo))
	return nil
}

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
