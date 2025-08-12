package ssh

import (
	"fmt"
	"time"

	"pb-deployer/internal/logger"
)

// startService starts a systemd service
func (sm *SSHManager) startService(serviceName string) error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"action":   "start",
	}).Info("Starting systemd service")

	startCmd := fmt.Sprintf("sudo systemctl start %s", serviceName)
	if _, err := sm.ExecuteCommand(startCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  startCmd,
		}).WithError(err).Error("Failed to start service")
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}

	// Wait a moment for service to start
	time.Sleep(2 * time.Second)

	// Verify service is running
	status, err := sm.getServiceStatus(serviceName)
	if err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
		}).WithError(err).Error("Failed to verify service started")
		return fmt.Errorf("failed to verify service %s started: %w", serviceName, err)
	}

	if !contains(status, "active (running)") {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"status":   status,
		}).Error("Service did not start properly")
		return fmt.Errorf("service %s did not start properly", serviceName)
	}

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
	}).Info("Service started successfully")

	return nil
}

// stopService stops a systemd service
func (sm *SSHManager) stopService(serviceName string) error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"action":   "stop",
	}).Info("Stopping systemd service")

	stopCmd := fmt.Sprintf("sudo systemctl stop %s", serviceName)
	if _, err := sm.ExecuteCommand(stopCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  stopCmd,
		}).WithError(err).Error("Failed to stop service")
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}

	// Wait a moment for service to stop
	time.Sleep(2 * time.Second)

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
	}).Info("Service stopped successfully")

	return nil
}

// restartService restarts a systemd service
func (sm *SSHManager) restartService(serviceName string) error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"action":   "restart",
	}).Info("Restarting systemd service")

	restartCmd := fmt.Sprintf("sudo systemctl restart %s", serviceName)
	if _, err := sm.ExecuteCommand(restartCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  restartCmd,
		}).WithError(err).Error("Failed to restart service")
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}

	// Wait a moment for service to restart
	time.Sleep(3 * time.Second)

	// Verify service is running
	status, err := sm.getServiceStatus(serviceName)
	if err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
		}).WithError(err).Error("Failed to verify service restarted")
		return fmt.Errorf("failed to verify service %s restarted: %w", serviceName, err)
	}

	if !contains(status, "active (running)") {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"status":   status,
		}).Error("Service did not restart properly")
		return fmt.Errorf("service %s did not restart properly", serviceName)
	}

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
	}).Info("Service restarted successfully")

	return nil
}

// getServiceStatus returns the status of a systemd service
func (sm *SSHManager) getServiceStatus(serviceName string) (string, error) {
	statusCmd := fmt.Sprintf("sudo systemctl status %s", serviceName)
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil {
		return output, fmt.Errorf("failed to get service status: %w", err)
	}
	return output, nil
}

// enableService enables a systemd service to start on boot
func (sm *SSHManager) enableService(serviceName string) error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"action":   "enable",
	}).Info("Enabling systemd service")

	enableCmd := fmt.Sprintf("sudo systemctl enable %s", serviceName)
	if _, err := sm.ExecuteCommand(enableCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  enableCmd,
		}).WithError(err).Error("Failed to enable service")
		return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
	}

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
	}).Info("Service enabled successfully")

	return nil
}

// disableService disables a systemd service from starting on boot
func (sm *SSHManager) disableService(serviceName string) error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"action":   "disable",
	}).Info("Disabling systemd service")

	disableCmd := fmt.Sprintf("sudo systemctl disable %s", serviceName)
	if _, err := sm.ExecuteCommand(disableCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  disableCmd,
		}).WithError(err).Error("Failed to disable service")
		return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
	}

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
	}).Info("Service disabled successfully")

	return nil
}

// reloadSystemdDaemon reloads the systemd daemon to pick up service file changes
func (sm *SSHManager) reloadSystemdDaemon() error {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"action":   "daemon-reload",
	}).Info("Reloading systemd daemon")

	reloadCmd := "sudo systemctl daemon-reload"
	if _, err := sm.ExecuteCommand(reloadCmd); err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"command":  reloadCmd,
		}).WithError(err).Error("Failed to reload systemd daemon")
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
	}).Info("Systemd daemon reloaded successfully")

	return nil
}

// isServiceActive checks if a service is currently active
func (sm *SSHManager) isServiceActive(serviceName string) (bool, error) {
	statusCmd := fmt.Sprintf("systemctl is-active %s", serviceName)
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil {
		return false, nil // Service might not exist or be inactive
	}
	return contains(output, "active"), nil
}

// isServiceEnabled checks if a service is enabled to start on boot
func (sm *SSHManager) isServiceEnabled(serviceName string) (bool, error) {
	statusCmd := fmt.Sprintf("systemctl is-enabled %s", serviceName)
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil {
		return false, nil // Service might not exist or be disabled
	}
	return contains(output, "enabled"), nil
}

// getServiceLogs returns recent logs for a service
func (sm *SSHManager) getServiceLogs(serviceName string, lines int) (string, error) {
	logger.WithFields(map[string]any{
		"host":     sm.server.Host,
		"username": sm.username,
		"service":  serviceName,
		"lines":    lines,
		"action":   "get_logs",
	}).Debug("Retrieving service logs")

	logsCmd := fmt.Sprintf("sudo journalctl -u %s --no-pager -n %d", serviceName, lines)
	output, err := sm.ExecuteCommand(logsCmd)
	if err != nil {
		logger.WithFields(map[string]any{
			"host":     sm.server.Host,
			"username": sm.username,
			"service":  serviceName,
			"command":  logsCmd,
		}).WithError(err).Error("Failed to get service logs")
		return "", fmt.Errorf("failed to get service logs: %w", err)
	}

	logger.WithFields(map[string]any{
		"host":            sm.server.Host,
		"username":        sm.username,
		"service":         serviceName,
		"lines_requested": lines,
		"output_length":   len(output),
	}).Debug("Service logs retrieved successfully")

	return output, nil
}

// Public API methods for service management

// GetServiceStatus returns the status of a systemd service (public API)
func (sm *SSHManager) GetServiceStatus(serviceName string) (string, error) {
	return sm.getServiceStatus(serviceName)
}

// StartService starts a systemd service (public API)
func (sm *SSHManager) StartService(serviceName string) error {
	return sm.startService(serviceName)
}

// StopService stops a systemd service (public API)
func (sm *SSHManager) StopService(serviceName string) error {
	return sm.stopService(serviceName)
}

// RestartService restarts a systemd service (public API)
func (sm *SSHManager) RestartService(serviceName string) error {
	return sm.restartService(serviceName)
}

// EnableService enables a systemd service to start on boot (public API)
func (sm *SSHManager) EnableService(serviceName string) error {
	return sm.enableService(serviceName)
}

// DisableService disables a systemd service from starting on boot (public API)
func (sm *SSHManager) DisableService(serviceName string) error {
	return sm.disableService(serviceName)
}

// ReloadSystemdDaemon reloads the systemd daemon (public API)
func (sm *SSHManager) ReloadSystemdDaemon() error {
	return sm.reloadSystemdDaemon()
}

// IsServiceActive checks if a service is currently active (public API)
func (sm *SSHManager) IsServiceActive(serviceName string) (bool, error) {
	return sm.isServiceActive(serviceName)
}

// IsServiceEnabled checks if a service is enabled to start on boot (public API)
func (sm *SSHManager) IsServiceEnabled(serviceName string) (bool, error) {
	return sm.isServiceEnabled(serviceName)
}

// GetServiceLogs returns recent logs for a service (public API)
func (sm *SSHManager) GetServiceLogs(serviceName string, lines int) (string, error) {
	return sm.getServiceLogs(serviceName, lines)
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring returns the index of substr in s, or -1 if not found
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
