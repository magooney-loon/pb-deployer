package ssh

import (
	"fmt"
	"time"
)

// startService starts a systemd service
func (sm *SSHManager) startService(serviceName string) error {
	startCmd := fmt.Sprintf("sudo systemctl start %s", serviceName)
	if _, err := sm.ExecuteCommand(startCmd); err != nil {
		return fmt.Errorf("failed to start service %s: %w", serviceName, err)
	}

	// Wait a moment for service to start
	time.Sleep(2 * time.Second)

	// Verify service is running
	status, err := sm.getServiceStatus(serviceName)
	if err != nil {
		return fmt.Errorf("failed to verify service %s started: %w", serviceName, err)
	}

	if !contains(status, "active (running)") {
		return fmt.Errorf("service %s did not start properly", serviceName)
	}

	return nil
}

// stopService stops a systemd service
func (sm *SSHManager) stopService(serviceName string) error {
	stopCmd := fmt.Sprintf("sudo systemctl stop %s", serviceName)
	if _, err := sm.ExecuteCommand(stopCmd); err != nil {
		return fmt.Errorf("failed to stop service %s: %w", serviceName, err)
	}

	// Wait a moment for service to stop
	time.Sleep(2 * time.Second)

	return nil
}

// restartService restarts a systemd service
func (sm *SSHManager) restartService(serviceName string) error {
	restartCmd := fmt.Sprintf("sudo systemctl restart %s", serviceName)
	if _, err := sm.ExecuteCommand(restartCmd); err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}

	// Wait a moment for service to restart
	time.Sleep(3 * time.Second)

	// Verify service is running
	status, err := sm.getServiceStatus(serviceName)
	if err != nil {
		return fmt.Errorf("failed to verify service %s restarted: %w", serviceName, err)
	}

	if !contains(status, "active (running)") {
		return fmt.Errorf("service %s did not restart properly", serviceName)
	}

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
	enableCmd := fmt.Sprintf("sudo systemctl enable %s", serviceName)
	if _, err := sm.ExecuteCommand(enableCmd); err != nil {
		return fmt.Errorf("failed to enable service %s: %w", serviceName, err)
	}
	return nil
}

// disableService disables a systemd service from starting on boot
func (sm *SSHManager) disableService(serviceName string) error {
	disableCmd := fmt.Sprintf("sudo systemctl disable %s", serviceName)
	if _, err := sm.ExecuteCommand(disableCmd); err != nil {
		return fmt.Errorf("failed to disable service %s: %w", serviceName, err)
	}
	return nil
}

// reloadSystemdDaemon reloads the systemd daemon to pick up service file changes
func (sm *SSHManager) reloadSystemdDaemon() error {
	reloadCmd := "sudo systemctl daemon-reload"
	if _, err := sm.ExecuteCommand(reloadCmd); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
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
	logsCmd := fmt.Sprintf("sudo journalctl -u %s --no-pager -n %d", serviceName, lines)
	output, err := sm.ExecuteCommand(logsCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get service logs: %w", err)
	}
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
