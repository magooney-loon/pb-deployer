package ssh

import (
	"fmt"
	"strings"
	"time"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/models"
)

// PostSecurityManager handles SSH operations after security lockdown
type PostSecurityManager struct {
	server       *models.Server
	rootManager  *SSHManager // May be nil after security lockdown
	appManager   *SSHManager // Primary connection after security lockdown
	securityMode bool        // True if security lockdown has been applied
}

// NewPostSecurityManager creates a manager for post-security-lockdown operations
func NewPostSecurityManager(server *models.Server, securityLocked bool) (*PostSecurityManager, error) {
	if server == nil {
		return nil, fmt.Errorf("server cannot be nil")
	}

	psm := &PostSecurityManager{
		server:       server,
		securityMode: securityLocked,
	}

	// If security is not locked, we can still use root connections
	if !securityLocked {
		rootManager, err := NewSSHManager(server, true)
		if err != nil {
			return nil, fmt.Errorf("failed to create root SSH manager: %w", err)
		}
		psm.rootManager = rootManager
	}

	// Always try to create app user connection
	appManager, err := NewSSHManager(server, false)
	if err != nil {
		// Ensure cleanup of any successfully created managers
		if psm.rootManager != nil {
			if closeErr := psm.rootManager.Close(); closeErr != nil {
				// Log but don't override the original error
				logger.WithFields(map[string]interface{}{
					"host": server.Host,
					"port": server.Port,
				}).WithError(closeErr).Warn("Failed to close root manager during cleanup")
			}
		}
		return nil, fmt.Errorf("failed to create app user SSH manager: %w", err)
	}
	psm.appManager = appManager

	return psm, nil
}

// GetActiveManager returns the appropriate SSH manager based on security status
func (psm *PostSecurityManager) GetActiveManager() *SSHManager {
	if psm.securityMode || psm.rootManager == nil {
		return psm.appManager
	}
	return psm.rootManager
}

// GetAppManager returns the app user SSH manager
func (psm *PostSecurityManager) GetAppManager() *SSHManager {
	return psm.appManager
}

// GetRootManager returns the root SSH manager (may be nil if security is locked)
func (psm *PostSecurityManager) GetRootManager() *SSHManager {
	return psm.rootManager
}

// ExecuteCommand executes a command using the appropriate manager
func (psm *PostSecurityManager) ExecuteCommand(command string) (string, error) {
	manager := psm.GetActiveManager()
	if manager == nil {
		logger.WithFields(map[string]interface{}{
			"host":            psm.server.Host,
			"security_locked": psm.securityMode,
		}).Error("No active SSH manager available for command execution")
		return "", fmt.Errorf("no active SSH manager available")
	}

	// Validate connection before executing command
	if !manager.IsConnected() {
		logger.WithFields(map[string]interface{}{
			"host":     psm.server.Host,
			"username": manager.GetUsername(),
			"command":  command,
		}).Error("SSH connection is not active for command execution")
		return "", fmt.Errorf("SSH connection is not active")
	}

	logger.WithFields(map[string]interface{}{
		"host":            psm.server.Host,
		"username":        manager.GetUsername(),
		"command":         command,
		"security_locked": psm.securityMode,
	}).Debug("Executing command via post-security manager")

	return manager.ExecuteCommand(command)
}

// ExecuteCommandStream executes a command with streaming output using the appropriate manager
func (psm *PostSecurityManager) ExecuteCommandStream(command string, output chan<- string) error {
	manager := psm.GetActiveManager()
	if manager == nil {
		return fmt.Errorf("no active SSH manager available")
	}

	// Validate connection before executing command
	if !manager.IsConnected() {
		return fmt.Errorf("SSH connection is not active")
	}

	return manager.ExecuteCommandStream(command, output)
}

// ExecutePrivilegedCommand executes a command that requires elevated privileges
// Uses sudo through the app user if security is locked, otherwise uses root
func (psm *PostSecurityManager) ExecutePrivilegedCommand(command string) (string, error) {
	if psm.securityMode || psm.rootManager == nil {
		// Use sudo through app user
		if psm.appManager == nil {
			logger.WithFields(map[string]interface{}{
				"host":    psm.server.Host,
				"command": command,
			}).Error("App manager not available for privileged command execution")
			return "", fmt.Errorf("app manager not available for privileged command")
		}
		if !psm.appManager.IsConnected() {
			logger.WithFields(map[string]interface{}{
				"host":     psm.server.Host,
				"username": psm.appManager.GetUsername(),
				"command":  command,
			}).Error("App SSH connection is not active for privileged command")
			return "", fmt.Errorf("app SSH connection is not active")
		}
		sudoCommand := fmt.Sprintf("sudo %s", command)
		logger.WithFields(map[string]interface{}{
			"host":         psm.server.Host,
			"username":     psm.appManager.GetUsername(),
			"command":      command,
			"sudo_command": sudoCommand,
		}).Debug("Executing privileged command via sudo")
		return psm.appManager.ExecuteCommand(sudoCommand)
	} else {
		// Use root manager directly
		if !psm.rootManager.IsConnected() {
			logger.WithFields(map[string]interface{}{
				"host":     psm.server.Host,
				"username": psm.rootManager.GetUsername(),
				"command":  command,
			}).Error("Root SSH connection is not active for privileged command")
			return "", fmt.Errorf("root SSH connection is not active")
		}
		logger.WithFields(map[string]interface{}{
			"host":     psm.server.Host,
			"username": psm.rootManager.GetUsername(),
			"command":  command,
		}).Debug("Executing privileged command as root")
		return psm.rootManager.ExecuteCommand(command)
	}
}

// TestConnections tests all available SSH connections
func (psm *PostSecurityManager) TestConnections() error {
	var errors []string

	// Test app user connection
	if psm.appManager != nil {
		if err := psm.appManager.TestConnection(); err != nil {
			errors = append(errors, fmt.Sprintf("app user connection failed: %v", err))
		}
	} else {
		errors = append(errors, "app user manager is not available")
	}

	// Test root connection if available and security is not locked
	if psm.rootManager != nil && !psm.securityMode {
		if err := psm.rootManager.TestConnection(); err != nil {
			errors = append(errors, fmt.Sprintf("root connection failed: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("connection test failures: %s", strings.Join(errors, "; "))
	}

	return nil
}

// SwitchToSecurityMode transitions the manager to security-locked mode
func (psm *PostSecurityManager) SwitchToSecurityMode() error {
	if psm.securityMode {
		return fmt.Errorf("already in security mode")
	}

	// Test app user connection before switching
	if psm.appManager == nil {
		return fmt.Errorf("app user manager not available")
	}

	if err := psm.appManager.TestConnection(); err != nil {
		return fmt.Errorf("app user connection test failed: %w", err)
	}

	// Close root connection if it exists
	if psm.rootManager != nil {
		if err := psm.rootManager.Close(); err != nil {
			// Log but don't fail the switch
			logger.WithFields(map[string]interface{}{
				"host": psm.server.Host,
				"port": psm.server.Port,
			}).WithError(err).Warn("Failed to close root manager during security mode switch")
		}
		psm.rootManager = nil
	}

	psm.securityMode = true
	logger.WithFields(map[string]interface{}{
		"host": psm.server.Host,
		"port": psm.server.Port,
	}).Info("Successfully switched post-security manager to security mode")
	return nil
}

// VerifyPostSecurityAccess verifies that required access is available after security lockdown
func (psm *PostSecurityManager) VerifyPostSecurityAccess() error {
	if psm.appManager == nil {
		return fmt.Errorf("app user manager not available")
	}

	// Test basic connectivity
	if err := psm.appManager.TestConnection(); err != nil {
		return fmt.Errorf("basic connectivity test failed: %w", err)
	}

	// Test sudo access for common deployment operations
	testCommands := []string{
		"sudo -n systemctl --version",
		"sudo -n mkdir -p /tmp/test_access",
		"sudo -n rm -rf /tmp/test_access",
	}

	for _, cmd := range testCommands {
		if _, err := psm.appManager.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("sudo access test failed for command '%s': %w", cmd, err)
		}
	}

	// Verify that root login is actually disabled
	if err := psm.verifyRootLoginDisabled(); err != nil {
		return fmt.Errorf("root login verification failed: %w", err)
	}

	return nil
}

// verifyRootLoginDisabled checks that root login is properly disabled
func (psm *PostSecurityManager) verifyRootLoginDisabled() error {
	// Check SSH configuration
	checkCmd := "grep -q '^PermitRootLogin no' /etc/ssh/sshd_config"
	if _, err := psm.appManager.ExecuteCommand(fmt.Sprintf("sudo %s", checkCmd)); err != nil {
		return fmt.Errorf("PermitRootLogin setting not found or incorrect")
	}

	return nil
}

// GetConnectionStatus returns detailed status of all connections
func (psm *PostSecurityManager) GetConnectionStatus() map[string]interface{} {
	status := map[string]interface{}{
		"security_mode": psm.securityMode,
		"server_host":   psm.server.Host,
		"server_port":   psm.server.Port,
	}

	// App user connection status
	if psm.appManager != nil {
		appInfo := psm.appManager.GetConnectionInfo()
		status["app_user"] = appInfo
	} else {
		status["app_user"] = map[string]interface{}{
			"connected": false,
			"error":     "app user manager not available",
		}
	}

	// Root connection status
	if psm.rootManager != nil && !psm.securityMode {
		rootInfo := psm.rootManager.GetConnectionInfo()
		status["root_user"] = rootInfo
	} else {
		reason := "security mode enabled"
		if psm.rootManager == nil {
			reason = "root manager not available"
		}
		status["root_user"] = map[string]interface{}{
			"connected": false,
			"reason":    reason,
		}
	}

	return status
}

// RestartService restarts a systemd service using appropriate privileges
func (psm *PostSecurityManager) RestartService(serviceName string) error {
	manager := psm.GetActiveManager()
	if manager == nil {
		return fmt.Errorf("no active SSH manager available")
	}

	if !manager.IsConnected() {
		return fmt.Errorf("SSH connection is not active")
	}

	if psm.securityMode || psm.rootManager == nil {
		// Use sudo
		cmd := fmt.Sprintf("sudo systemctl restart %s", serviceName)
		_, err := manager.ExecuteCommand(cmd)
		return err
	} else {
		// Use root manager directly
		return psm.rootManager.RestartService(serviceName)
	}
}

// StartService starts a systemd service using appropriate privileges
func (psm *PostSecurityManager) StartService(serviceName string) error {
	manager := psm.GetActiveManager()
	if manager == nil {
		return fmt.Errorf("no active SSH manager available")
	}

	if !manager.IsConnected() {
		return fmt.Errorf("SSH connection is not active")
	}

	if psm.securityMode || psm.rootManager == nil {
		// Use sudo
		cmd := fmt.Sprintf("sudo systemctl start %s", serviceName)
		_, err := manager.ExecuteCommand(cmd)
		if err != nil {
			return fmt.Errorf("failed to start service %s: %w", serviceName, err)
		}

		// Verify service started
		time.Sleep(2 * time.Second)
		statusCmd := fmt.Sprintf("systemctl is-active %s", serviceName)
		output, err := manager.ExecuteCommand(statusCmd)
		if err != nil || !strings.Contains(output, "active") {
			return fmt.Errorf("service %s did not start properly", serviceName)
		}

		return nil
	} else {
		// Use root manager directly
		return psm.rootManager.StartService(serviceName)
	}
}

// StopService stops a systemd service using appropriate privileges
func (psm *PostSecurityManager) StopService(serviceName string) error {
	manager := psm.GetActiveManager()
	if manager == nil {
		return fmt.Errorf("no active SSH manager available")
	}

	if !manager.IsConnected() {
		return fmt.Errorf("SSH connection is not active")
	}

	if psm.securityMode || psm.rootManager == nil {
		// Use sudo
		cmd := fmt.Sprintf("sudo systemctl stop %s", serviceName)
		_, err := manager.ExecuteCommand(cmd)
		return err
	} else {
		// Use root manager directly
		return psm.rootManager.StopService(serviceName)
	}
}

// ExecutePrivilegedCommandWithDetails executes a privileged command with detailed error reporting
func (psm *PostSecurityManager) ExecutePrivilegedCommandWithDetails(command string) (string, error) {
	manager := psm.GetActiveManager()
	if manager == nil {
		return "", fmt.Errorf("no active SSH manager available")
	}

	var fullCommand string
	if psm.securityMode || psm.rootManager == nil {
		fullCommand = fmt.Sprintf("sudo %s", command)
	} else {
		fullCommand = command
	}

	output, err := manager.ExecuteCommand(fullCommand)
	if err != nil {
		return output, fmt.Errorf("privileged command failed (security_mode: %v, command: %s): %w",
			psm.securityMode, fullCommand, err)
	}

	return output, nil
}

// Close closes all SSH connections
func (psm *PostSecurityManager) Close() error {
	var errors []string

	// Close app manager
	if psm.appManager != nil {
		if err := psm.appManager.Close(); err != nil {
			logger.WithFields(map[string]interface{}{
				"host": psm.server.Host,
				"port": psm.server.Port,
			}).WithError(err).Warn("Failed to close app manager during post-security manager shutdown")
			errors = append(errors, fmt.Sprintf("app manager close error: %v", err))
		}
		psm.appManager = nil
	}

	// Close root manager
	if psm.rootManager != nil {
		if err := psm.rootManager.Close(); err != nil {
			logger.WithFields(map[string]interface{}{
				"host": psm.server.Host,
				"port": psm.server.Port,
			}).WithError(err).Warn("Failed to close root manager during post-security manager shutdown")
			errors = append(errors, fmt.Sprintf("root manager close error: %v", err))
		}
		psm.rootManager = nil
	}

	// Reset state
	psm.securityMode = false

	logger.WithFields(map[string]interface{}{
		"host": psm.server.Host,
		"port": psm.server.Port,
	}).Info("Post-security manager closed successfully")

	if len(errors) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// IsSecurityMode returns true if the manager is in security-locked mode
func (psm *PostSecurityManager) IsSecurityMode() bool {
	return psm.securityMode
}

// GetServer returns the server associated with this manager
func (psm *PostSecurityManager) GetServer() *models.Server {
	return psm.server
}

// ValidateDeploymentCapabilities checks if the manager can perform deployment operations
func (psm *PostSecurityManager) ValidateDeploymentCapabilities() error {
	// Ensure we have an active manager
	manager := psm.GetActiveManager()
	if manager == nil {
		return fmt.Errorf("no active SSH manager available for validation")
	}

	if !manager.IsConnected() {
		return fmt.Errorf("SSH connection is not active for validation")
	}

	// Test file operations
	testDir := "/tmp/pb_deploy_test"

	// Create test directory
	if _, err := psm.ExecutePrivilegedCommand(fmt.Sprintf("mkdir -p %s", testDir)); err != nil {
		return fmt.Errorf("failed to create test directory: %w", err)
	}

	// Ensure cleanup happens even if tests fail
	defer func() {
		if _, cleanupErr := psm.ExecutePrivilegedCommand(fmt.Sprintf("rm -rf %s", testDir)); cleanupErr != nil {
			logger.WithFields(map[string]interface{}{
				"host":     psm.server.Host,
				"test_dir": testDir,
			}).WithError(cleanupErr).Warn("Failed to cleanup test directory after deployment capability validation")
		}
	}()

	// Test file permissions
	if _, err := psm.ExecutePrivilegedCommand(fmt.Sprintf("touch %s/test_file", testDir)); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	// Test ownership changes
	appUser := psm.server.AppUsername
	if _, err := psm.ExecutePrivilegedCommand(fmt.Sprintf("chown %s:%s %s/test_file", appUser, appUser, testDir)); err != nil {
		return fmt.Errorf("failed to change file ownership: %w", err)
	}

	// Test service management
	if _, err := psm.ExecutePrivilegedCommand("systemctl --version"); err != nil {
		return fmt.Errorf("systemctl access test failed: %w", err)
	}

	return nil
}
