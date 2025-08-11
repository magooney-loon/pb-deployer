package ssh

import (
	"context"
	"fmt"
	"time"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/models"
)

// SSHService provides a high-level service interface for SSH operations using connection pooling
type SSHService struct {
	connectionManager *ConnectionManager
	healthMonitor     *HealthMonitor
}

// NewSSHService creates a new SSH service instance
func NewSSHService() *SSHService {
	return &SSHService{
		connectionManager: GetConnectionManager(),
		healthMonitor:     GetHealthMonitor(),
	}
}

// GetGlobalSSHService returns the global SSH service instance
var globalSSHService *SSHService

func GetSSHService() *SSHService {
	if globalSSHService == nil {
		globalSSHService = NewSSHService()
	}
	return globalSSHService
}

// ConnectionTestResult represents the result of a connection test
type ConnectionTestResult struct {
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	Username     string        `json:"username"`
	AuthMethod   string        `json:"auth_method,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
}

// TestConnection tests SSH connectivity for a server
func (s *SSHService) TestConnection(server *models.Server, asRoot bool) *ConnectionTestResult {
	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"as_root": asRoot,
	}).Debug("Starting SSH connection test")

	return s.TestConnectionWithContext(context.Background(), server, asRoot)
}

// TestConnectionWithContext tests SSH connectivity with context timeout
func (s *SSHService) TestConnectionWithContext(ctx context.Context, server *models.Server, asRoot bool) *ConnectionTestResult {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	result := &ConnectionTestResult{
		Username: username,
	}

	// For security-locked servers attempting root connection, return expected failure
	if server.SecurityLocked && asRoot {
		result.Error = "Root SSH access disabled by security lockdown"
		logger.WithFields(map[string]interface{}{
			"host":            server.Host,
			"username":        username,
			"security_locked": true,
		}).Debug("Root SSH access blocked due to security lockdown")
		return result
	}

	// Pre-accept host key to avoid verification issues
	if err := AcceptHostKey(server); err != nil {
		logger.WithFields(map[string]interface{}{
			"host": server.Host,
			"port": server.Port,
		}).WithError(err).Warn("Could not pre-accept host key, continuing anyway")
	}

	// Test connection using connection manager
	start := time.Now()
	err := s.connectionManager.TestConnection(server, asRoot)
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Error = fmt.Sprintf("SSH connection test failed: %v", err)
		logger.WithFields(map[string]interface{}{
			"host":          server.Host,
			"port":          server.Port,
			"username":      username,
			"response_time": result.ResponseTime.String(),
		}).WithError(err).Error("SSH connection test failed")
		return result
	}

	result.Success = true
	logger.WithFields(map[string]interface{}{
		"host":          server.Host,
		"port":          server.Port,
		"username":      username,
		"response_time": result.ResponseTime.String(),
	}).Info("SSH connection test successful")

	// Determine auth method used
	if server.UseSSHAgent {
		result.AuthMethod = "ssh_agent"
	} else if server.ManualKeyPath != "" {
		result.AuthMethod = "private_key"
	} else {
		result.AuthMethod = "default_keys"
	}

	return result
}

// ExecuteCommand executes a command on the server using the connection pool
func (s *SSHService) ExecuteCommand(server *models.Server, asRoot bool, command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"username": s.getUsername(server, asRoot),
		"command":  command,
		"as_root":  asRoot,
	}).Debug("Executing SSH command via service")

	output, err := s.connectionManager.ExecuteCommand(server, asRoot, command)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"host":     server.Host,
			"username": s.getUsername(server, asRoot),
			"command":  command,
		}).WithError(err).Error("SSH command execution failed via service")
	}

	return output, err
}

// ExecuteCommandStream executes a command with streaming output using the connection pool
func (s *SSHService) ExecuteCommandStream(server *models.Server, asRoot bool, command string, output chan<- string) error {
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"username": s.getUsername(server, asRoot),
		"command":  command,
		"as_root":  asRoot,
	}).Debug("Starting streaming SSH command execution via service")

	err := s.connectionManager.ExecuteCommandStream(server, asRoot, command, output)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"host":     server.Host,
			"username": s.getUsername(server, asRoot),
			"command":  command,
		}).WithError(err).Error("SSH streaming command execution failed via service")
	}

	return err
}

// RunServerSetup performs complete server setup using connection pooling
func (s *SSHService) RunServerSetup(server *models.Server, progressChan chan<- SetupStep) error {
	if !s.isValidForRootOperations(server) {
		logger.WithFields(map[string]interface{}{
			"host":            server.Host,
			"security_locked": server.SecurityLocked,
		}).Error("Server setup cannot proceed - invalid configuration for root operations")
		return fmt.Errorf("server setup requires root access and server cannot be security-locked")
	}

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Starting server setup process via SSH service")

	// Get pooled connection for setup
	conn, err := s.connectionManager.pool.GetOrCreateConnection(server, true)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection for setup: %w", err)
	}

	// Get the underlying SSH manager for setup operations
	manager := conn.GetManager()
	if manager == nil {
		return fmt.Errorf("failed to get SSH manager from pooled connection")
	}

	// Run the setup process
	return manager.RunServerSetup(progressChan)
}

// ApplySecurityLockdown applies security lockdown using connection pooling
func (s *SSHService) ApplySecurityLockdown(server *models.Server, progressChan chan<- SetupStep) error {
	if !s.isValidForRootOperations(server) {
		logger.WithFields(map[string]interface{}{
			"host":            server.Host,
			"security_locked": server.SecurityLocked,
		}).Error("Security lockdown cannot proceed - invalid configuration for root operations")
		return fmt.Errorf("security lockdown requires root access and server cannot already be security-locked")
	}

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Starting security lockdown process via SSH service")

	// Get pooled connection for security operations
	conn, err := s.connectionManager.pool.GetOrCreateConnection(server, true)
	if err != nil {
		return fmt.Errorf("failed to get SSH connection for security lockdown: %w", err)
	}

	// Get the underlying SSH manager for security operations
	manager := conn.GetManager()
	if manager == nil {
		return fmt.Errorf("failed to get SSH manager from pooled connection")
	}

	// Run the security lockdown process
	return manager.ApplySecurityLockdown(progressChan)
}

// CreatePostSecurityManager creates a manager for handling post-security-lockdown operations
func (s *SSHService) CreatePostSecurityManager(server *models.Server) (*PostSecurityManager, error) {
	return NewPostSecurityManager(server, server.SecurityLocked)
}

// GetConnectionStatus returns the status of all pooled connections
func (s *SSHService) GetConnectionStatus() map[string]ConnectionHealthStatus {
	return s.connectionManager.GetConnectionStatus()
}

// GetHealthMetrics returns overall health metrics for all connections
func (s *SSHService) GetHealthMetrics() *HealthMetrics {
	return s.healthMonitor.GetHealthMetrics()
}

// CleanupConnections removes stale connections from the pool
func (s *SSHService) CleanupConnections() int {
	return s.connectionManager.CleanupConnections()
}

// RestartService restarts a systemd service using appropriate privileges
func (s *SSHService) RestartService(server *models.Server, serviceName string) error {
	if server.SecurityLocked {
		// Use PostSecurityManager for security-locked servers
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			return fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()
		return psm.RestartService(serviceName)
	} else {
		// Use regular connection manager for non-security-locked servers
		cmd := fmt.Sprintf("systemctl restart %s", serviceName)
		_, err := s.ExecuteCommand(server, true, cmd)
		return err
	}
}

// StartService starts a systemd service using appropriate privileges
func (s *SSHService) StartService(server *models.Server, serviceName string) error {
	logger.WithFields(map[string]interface{}{
		"host":            server.Host,
		"service":         serviceName,
		"security_locked": server.SecurityLocked,
	}).Info("Starting service via SSH service")

	if server.SecurityLocked {
		// Use PostSecurityManager for security-locked servers
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"host":    server.Host,
				"service": serviceName,
			}).WithError(err).Error("Failed to create post-security manager for service start")
			return fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()
		return psm.StartService(serviceName)
	} else {
		// Use regular connection manager for non-security-locked servers
		cmd := fmt.Sprintf("systemctl start %s", serviceName)
		_, err := s.ExecuteCommand(server, true, cmd)
		return err
	}
}

// StopService stops a systemd service using appropriate privileges
func (s *SSHService) StopService(server *models.Server, serviceName string) error {
	logger.WithFields(map[string]interface{}{
		"host":            server.Host,
		"service":         serviceName,
		"security_locked": server.SecurityLocked,
	}).Info("Stopping service via SSH service")

	if server.SecurityLocked {
		// Use PostSecurityManager for security-locked servers
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"host":    server.Host,
				"service": serviceName,
			}).WithError(err).Error("Failed to create post-security manager for service stop")
			return fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()
		return psm.StopService(serviceName)
	} else {
		// Use regular connection manager for non-security-locked servers
		cmd := fmt.Sprintf("systemctl stop %s", serviceName)
		_, err := s.ExecuteCommand(server, true, cmd)
		return err
	}
}

// ExecutePrivilegedCommand executes a command that requires elevated privileges
func (s *SSHService) ExecutePrivilegedCommand(server *models.Server, command string) (string, error) {
	logger.WithFields(map[string]interface{}{
		"host":            server.Host,
		"command":         command,
		"security_locked": server.SecurityLocked,
	}).Debug("Executing privileged command via SSH service")

	if server.SecurityLocked {
		// Use PostSecurityManager for security-locked servers (uses sudo)
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"host":    server.Host,
				"command": command,
			}).WithError(err).Error("Failed to create post-security manager for privileged command")
			return "", fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()
		return psm.ExecutePrivilegedCommand(command)
	} else {
		// Use root connection for non-security-locked servers
		return s.ExecuteCommand(server, true, command)
	}
}

// ValidateDeploymentCapabilities checks if the server can perform deployment operations
func (s *SSHService) ValidateDeploymentCapabilities(server *models.Server) error {
	logger.WithFields(map[string]interface{}{
		"host":            server.Host,
		"security_locked": server.SecurityLocked,
	}).Info("Validating deployment capabilities via SSH service")

	if server.SecurityLocked {
		// Use PostSecurityManager for validation
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			logger.WithFields(map[string]interface{}{
				"host": server.Host,
			}).WithError(err).Error("Failed to create post-security manager for deployment validation")
			return fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()
		return psm.ValidateDeploymentCapabilities()
	} else {
		// For non-security-locked servers, test basic connectivity
		testResult := s.TestConnection(server, true)
		if !testResult.Success {
			logger.WithFields(map[string]interface{}{
				"host":  server.Host,
				"error": testResult.Error,
			}).Error("Deployment validation failed - connection test failed")
			return fmt.Errorf("deployment validation failed: %s", testResult.Error)
		}

		// Test basic command execution
		_, err := s.ExecuteCommand(server, true, "echo 'deployment_test'")
		return err
	}
}

// GetServerSetupStatus returns the current setup status of the server
func (s *SSHService) GetServerSetupStatus(server *models.Server) (map[string]bool, error) {
	// Get pooled connection
	conn, err := s.connectionManager.pool.GetOrCreateConnection(server, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH connection: %w", err)
	}

	// Get the underlying SSH manager
	manager := conn.GetManager()
	if manager == nil {
		return nil, fmt.Errorf("failed to get SSH manager from pooled connection")
	}

	return manager.GetSetupStatus()
}

// GetServerSecurityStatus returns the current security status of the server
func (s *SSHService) GetServerSecurityStatus(server *models.Server) (map[string]bool, error) {
	if server.SecurityLocked {
		// For security-locked servers, use PostSecurityManager
		psm, err := s.CreatePostSecurityManager(server)
		if err != nil {
			return nil, fmt.Errorf("failed to create post-security manager: %w", err)
		}
		defer psm.Close()

		// Create a basic security status for locked servers
		return map[string]bool{
			"firewall_active":     true, // Assume security measures are in place
			"fail2ban_active":     true,
			"ssh_hardened":        true,
			"root_login_disabled": true,
			"security_locked":     true,
		}, nil
	} else {
		// Get pooled connection for non-security-locked servers
		conn, err := s.connectionManager.pool.GetOrCreateConnection(server, true)
		if err != nil {
			return nil, fmt.Errorf("failed to get SSH connection: %w", err)
		}

		// Get the underlying SSH manager
		manager := conn.GetManager()
		if manager == nil {
			return nil, fmt.Errorf("failed to get SSH manager from pooled connection")
		}

		status, err := manager.GetSecurityStatus()
		if err != nil {
			return nil, err
		}

		// Add security_locked status
		status["security_locked"] = false
		return status, nil
	}
}

// PerformDiagnostics runs comprehensive diagnostics on the server
func (s *SSHService) PerformDiagnostics(server *models.Server, asRoot bool) ([]ConnectionDiagnostic, error) {
	if server.SecurityLocked && !asRoot {
		// Use specialized post-security diagnostics for app user on locked servers
		return DiagnoseAppUserPostSecurity(server)
	} else {
		// Use general troubleshooting for other cases
		return TroubleshootConnection(server, "")
	}
}

// AutoFixCommonIssues attempts to automatically fix common SSH issues
func (s *SSHService) AutoFixCommonIssues(server *models.Server) []ConnectionDiagnostic {
	return FixCommonIssues(server)
}

// Shutdown gracefully shuts down the SSH service and all connections
func (s *SSHService) Shutdown() {
	logger.Info("Shutting down SSH service")

	if s.connectionManager != nil {
		logger.Debug("Shutting down connection manager")
		s.connectionManager.Shutdown()
	}
	if s.healthMonitor != nil {
		logger.Debug("Shutting down health monitor")
		s.healthMonitor.Stop()
	}

	logger.Info("SSH service shutdown complete")
}

// getUsername helper to get the appropriate username for logging
func (s *SSHService) getUsername(server *models.Server, asRoot bool) string {
	if asRoot {
		return server.RootUsername
	}
	return server.AppUsername
}

// Helper methods

// isValidForRootOperations checks if a server is valid for root operations
func (s *SSHService) isValidForRootOperations(server *models.Server) bool {
	if server == nil {
		return false
	}

	// Root operations should not be performed on security-locked servers
	return !server.SecurityLocked
}

// GetConnectionKey generates a connection key for a server
func (s *SSHService) GetConnectionKey(server *models.Server, asRoot bool) string {
	userType := "app"
	if asRoot {
		userType = "root"
	}
	return fmt.Sprintf("%s:%d:%s:%s", server.Host, server.Port, server.ID, userType)
}

// IsConnectionHealthy checks if a specific connection is healthy
func (s *SSHService) IsConnectionHealthy(server *models.Server, asRoot bool) bool {
	key := s.GetConnectionKey(server, asRoot)
	result, err := s.healthMonitor.CheckConnectionHealth(key)
	if err != nil {
		return false
	}
	return result.Status == StatusHealthy
}

// RecoverConnection attempts to recover a failed connection
func (s *SSHService) RecoverConnection(server *models.Server, asRoot bool) error {
	key := s.GetConnectionKey(server, asRoot)
	return s.healthMonitor.RecoverConnection(key)
}

// GetConnectionInfo returns detailed information about a connection
func (s *SSHService) GetConnectionInfo(server *models.Server, asRoot bool) (map[string]interface{}, error) {
	conn, err := s.connectionManager.pool.GetOrCreateConnection(server, asRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	manager := conn.GetManager()
	if manager == nil {
		return nil, fmt.Errorf("SSH manager not available")
	}

	info := manager.GetConnectionInfo()

	// Add additional connection pool information
	status := conn.GetHealthStatus()
	info["pool_health"] = map[string]interface{}{
		"healthy":       status.Healthy,
		"last_used":     status.LastUsed,
		"age":           status.Age.String(),
		"use_count":     status.UseCount,
		"response_time": status.ResponseTime.String(),
	}

	return info, nil
}

// ValidateServerConfig validates server configuration before operations
func (s *SSHService) ValidateServerConfig(server *models.Server) error {
	if server == nil {
		return fmt.Errorf("server cannot be nil")
	}

	if server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", server.Port)
	}

	if server.AppUsername == "" {
		return fmt.Errorf("app username cannot be empty")
	}

	if server.RootUsername == "" {
		return fmt.Errorf("root username cannot be empty")
	}

	return nil
}
