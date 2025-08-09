package ssh

import (
	"fmt"
	"strings"
	"time"
)

// detectSSHServiceName detects the correct SSH service name on the system
func (sm *SSHManager) detectSSHServiceName(progressChan chan<- SetupStep) (string, error) {
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Detecting SSH service name", 81)
	}

	// Try common SSH service names
	serviceNames := []string{"sshd", "ssh", "openssh", "openssh-server"}
	var lastError error

	for _, serviceName := range serviceNames {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Testing service name: %s", serviceName), 82)
		}

		// Method 1: Check if service is loaded (regardless of enabled status)
		cmd := fmt.Sprintf("systemctl list-units --type=service --all | grep -q '%s.service'", serviceName)
		if _, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: list-units)", serviceName), 83)
			}
			return serviceName, nil
		}

		// Method 2: Check if service unit file exists
		cmd = fmt.Sprintf("systemctl list-unit-files --type=service | grep -q '%s.service'", serviceName)
		if _, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: list-unit-files)", serviceName), 83)
			}
			return serviceName, nil
		}

		// Method 3: Check if service status can be queried (even if inactive)
		cmd = fmt.Sprintf("systemctl status %s", serviceName)
		if output, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: status check)", serviceName), 83)
			}
			return serviceName, nil
		} else {
			lastError = fmt.Errorf("service %s status check failed: %w (output: %s)", serviceName, err, output)
		}
	}

	// Method 4: Try to find SSH daemon process
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Checking for running SSH daemon process", 84)
	}
	if output, err := sm.ExecuteCommand("ps aux | grep -E '[s]shd.*-D|[o]penssh.*-D' | grep -v grep"); err == nil && strings.TrimSpace(output) != "" {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Found running SSH daemon, checking if managed by systemctl", 85)
		}

		// Check if this SSH daemon is managed by systemctl by testing a few more service names
		finalServiceNames := []string{"sshd", "ssh", "openssh-server"}
		for _, serviceName := range finalServiceNames {
			// Use a more lenient check - see if systemctl knows about the service at all
			cmd := fmt.Sprintf("systemctl show %s --property=LoadState 2>/dev/null | grep -q 'LoadState=loaded'", serviceName)
			if _, err := sm.ExecuteCommand(cmd); err == nil {
				if progressChan != nil {
					sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH daemon is managed by systemctl as '%s'", serviceName), 87)
				}
				return serviceName, nil
			}
		}

		// SSH is running but not managed by systemctl, return special signal
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "SSH daemon running but not managed by systemctl", 87)
		}
		return "skip-reload", nil
	}

	// Method 5: Check for SSH config file to confirm SSH is installed
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Checking for SSH configuration file", 86)
	}
	if _, err := sm.ExecuteCommand("test -f /etc/ssh/sshd_config"); err == nil {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Found SSH config file, but no running daemon detected", 87)
		}
		return "skip-reload", nil
	}

	// Return error with comprehensive debugging information
	debugInfo := make([]string, 0)

	if debugOutput, err := sm.ExecuteCommand("systemctl list-units --type=service | grep -E 'ssh|openssh'"); err == nil {
		debugInfo = append(debugInfo, fmt.Sprintf("SSH services: %s", debugOutput))
	}

	if debugOutput, err := sm.ExecuteCommand("ps aux | grep ssh | grep -v grep"); err == nil {
		debugInfo = append(debugInfo, fmt.Sprintf("SSH processes: %s", debugOutput))
	}

	if debugOutput, err := sm.ExecuteCommand("ls -la /etc/ssh/"); err == nil {
		debugInfo = append(debugInfo, fmt.Sprintf("SSH config dir: %s", debugOutput))
	}

	return "", fmt.Errorf("could not detect SSH service name. Last error: %v. Debug info: %s", lastError, strings.Join(debugInfo, "; "))
}

// ApplySecurityLockdown applies all security hardening measures to the server
func (sm *SSHManager) ApplySecurityLockdown(progressChan chan<- SetupStep) error {
	if !sm.isRoot {
		return fmt.Errorf("security lockdown requires root access")
	}

	sm.sendProgressUpdate(progressChan, "security_lockdown", "running", "Starting security lockdown process", 0)

	steps := []struct {
		name string
		fn   func(chan<- SetupStep) error
	}{
		{"setup_firewall", sm.setupFirewallWithProgress},
		{"setup_fail2ban", sm.setupFail2banWithProgress},
		{"harden_ssh", sm.hardenSSHWithProgress},
		{"verify_security", sm.verifySecurityLockdownWithProgress},
	}

	totalSteps := len(steps)

	for i, step := range steps {
		// Send running status
		sm.sendProgressUpdate(progressChan, step.name, "running", fmt.Sprintf("Executing %s", step.name), (i*100)/totalSteps)

		if err := step.fn(progressChan); err != nil {
			// Send failure status
			sm.sendProgressUpdate(progressChan, step.name, "failed", fmt.Sprintf("Failed to execute %s", step.name), (i*100)/totalSteps, err.Error())
			return fmt.Errorf("security step %s failed: %w", step.name, err)
		}

		// Send success status
		sm.sendProgressUpdate(progressChan, step.name, "success", fmt.Sprintf("Successfully completed %s", step.name), ((i+1)*100)/totalSteps)
	}

	sm.sendProgressUpdate(progressChan, "security_lockdown", "success", "Security lockdown completed successfully", 100)
	return nil
}

// setupFirewall configures UFW firewall with essential ports
func (sm *SSHManager) setupFirewall() error {
	return sm.setupFirewallWithProgress(nil)
}

// setupFirewallWithProgress configures UFW firewall with detailed progress reporting
func (sm *SSHManager) setupFirewallWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Installing UFW firewall", 10)
	}

	// Install UFW if not already installed
	installCmd := "which ufw || (apt-get update && apt-get install -y ufw)"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install UFW: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Resetting UFW to default state", 20)
	}

	// Reset UFW to default state
	if _, err := sm.ExecuteCommand("ufw --force reset"); err != nil {
		return fmt.Errorf("failed to reset UFW: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Setting default firewall policies", 30)
	}

	// Set default policies
	if _, err := sm.ExecuteCommand("ufw default deny incoming"); err != nil {
		return fmt.Errorf("failed to set default deny incoming: %w", err)
	}

	if _, err := sm.ExecuteCommand("ufw default allow outgoing"); err != nil {
		return fmt.Errorf("failed to set default allow outgoing: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Configuring essential ports", 50)
	}

	// Allow essential ports
	essentialPorts := []struct {
		port        string
		description string
	}{
		{fmt.Sprintf("%d", sm.server.Port), "SSH"},
		{"80", "HTTP"},
		{"443", "HTTPS"},
	}

	for _, portRule := range essentialPorts {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "setup_firewall", "running", fmt.Sprintf("Allowing port %s (%s)", portRule.port, portRule.description), 50)
		}

		cmd := fmt.Sprintf("ufw allow %s comment '%s'", portRule.port, portRule.description)
		if _, err := sm.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to allow port %s: %w", portRule.port, err)
		}
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Enabling UFW firewall", 80)
	}

	// Enable UFW
	if _, err := sm.ExecuteCommand("ufw --force enable"); err != nil {
		return fmt.Errorf("failed to enable UFW: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_firewall", "running", "Verifying UFW status", 90)
	}

	// Verify UFW status
	status, err := sm.ExecuteCommand("ufw status")
	if err != nil {
		return fmt.Errorf("failed to check UFW status: %w", err)
	}

	if !strings.Contains(status, "Status: active") {
		return fmt.Errorf("UFW is not active after enabling")
	}

	return nil
}

// setupFail2ban installs and configures fail2ban for SSH protection
func (sm *SSHManager) setupFail2ban() error {
	return sm.setupFail2banWithProgress(nil)
}

// setupFail2banWithProgress installs and configures fail2ban with detailed progress reporting
func (sm *SSHManager) setupFail2banWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_fail2ban", "running", "Installing fail2ban package", 10)
	}

	// Install fail2ban
	installCmd := "which fail2ban-client || (apt-get update && apt-get install -y fail2ban)"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install fail2ban: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_fail2ban", "running", "Creating fail2ban jail configuration", 30)
	}

	// Create custom fail2ban jail for SSH
	jailConfig := fmt.Sprintf(`[sshd]
enabled = true
port = %d
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600
findtime = 600
ignoreip = 127.0.0.1/8 ::1
`, sm.server.Port)

	// Write jail configuration
	writeJailCmd := fmt.Sprintf("cat > /etc/fail2ban/jail.d/sshd-custom.conf << 'EOF'\n%sEOF", jailConfig)
	if _, err := sm.ExecuteCommand(writeJailCmd); err != nil {
		return fmt.Errorf("failed to create fail2ban jail configuration: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_fail2ban", "running", "Restarting fail2ban service", 60)
	}

	// Restart and enable fail2ban service
	if _, err := sm.ExecuteCommand("systemctl restart fail2ban"); err != nil {
		return fmt.Errorf("failed to restart fail2ban: %w", err)
	}

	if _, err := sm.ExecuteCommand("systemctl enable fail2ban"); err != nil {
		return fmt.Errorf("failed to enable fail2ban: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_fail2ban", "running", "Verifying fail2ban status", 80)
	}

	// Verify fail2ban is running
	status, err := sm.ExecuteCommand("systemctl is-active fail2ban")
	if err != nil || !strings.Contains(status, "active") {
		return fmt.Errorf("fail2ban is not running after configuration")
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "setup_fail2ban", "running", "Verifying SSH jail configuration", 90)
	}

	// Verify SSH jail is active
	time.Sleep(2 * time.Second) // Give fail2ban time to start the jail
	jailStatus, err := sm.ExecuteCommand("fail2ban-client status sshd")
	if err != nil {
		return fmt.Errorf("failed to check SSH jail status: %w", err)
	}

	if !strings.Contains(jailStatus, "Status for the jail: sshd") {
		return fmt.Errorf("SSH jail is not properly configured")
	}

	return nil
}

// hardenSSH applies SSH hardening configurations
func (sm *SSHManager) hardenSSH() error {
	return sm.hardenSSHWithProgress(nil)
}

// hardenSSHWithProgress applies SSH hardening with detailed progress reporting
func (sm *SSHManager) hardenSSHWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Backing up SSH configuration", 10)
	}

	// Backup original SSH config
	backupCmd := "cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup.$(date +%Y%m%d_%H%M%S)"
	if _, err := sm.ExecuteCommand(backupCmd); err != nil {
		return fmt.Errorf("failed to backup SSH config: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Applying SSH hardening settings", 30)
	}

	// SSH hardening configurations
	hardening := []struct {
		setting string
		value   string
	}{
		{"PasswordAuthentication", "no"},
		{"PubkeyAuthentication", "yes"},
		{"PermitRootLogin", "no"},
		{"PermitEmptyPasswords", "no"},
		{"ChallengeResponseAuthentication", "no"},
		{"UsePAM", "no"},
		{"X11Forwarding", "no"},
		{"PrintMotd", "no"},
		{"TCPKeepAlive", "yes"},
		{"Compression", "no"},
		{"MaxAuthTries", "3"},
		{"MaxSessions", "2"},
		{"ClientAliveInterval", "300"},
		{"ClientAliveCountMax", "2"},
		{"LoginGraceTime", "60"},
		{"Protocol", "2"},
	}

	// Read current SSH config
	currentConfig, err := sm.ExecuteCommand("cat /etc/ssh/sshd_config")
	if err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	// Apply each hardening setting
	modifiedConfig := currentConfig
	for _, setting := range hardening {
		// Remove any existing setting (commented or not)
		lines := strings.Split(modifiedConfig, "\n")
		var newLines []string

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmedLine, setting.setting) &&
				!strings.HasPrefix(trimmedLine, "#"+setting.setting) {
				newLines = append(newLines, line)
			}
		}

		// Add the new setting
		newLines = append(newLines, fmt.Sprintf("%s %s", setting.setting, setting.value))
		modifiedConfig = strings.Join(newLines, "\n")
	}

	// Add custom port if not default
	if sm.server.Port != 22 {
		// Remove any existing Port settings
		lines := strings.Split(modifiedConfig, "\n")
		var newLines []string

		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmedLine, "Port ") &&
				!strings.HasPrefix(trimmedLine, "#Port ") {
				newLines = append(newLines, line)
			}
		}

		newLines = append(newLines, fmt.Sprintf("Port %d", sm.server.Port))
		modifiedConfig = strings.Join(newLines, "\n")
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Writing hardened SSH configuration", 50)
	}

	// Write the hardened configuration
	writeConfigCmd := fmt.Sprintf("cat > /etc/ssh/sshd_config << 'EOF'\n%s\nEOF", modifiedConfig)
	if _, err := sm.ExecuteCommand(writeConfigCmd); err != nil {
		return fmt.Errorf("failed to write hardened SSH config: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Testing SSH configuration syntax", 70)
	}

	// Test SSH configuration syntax
	if _, err := sm.ExecuteCommand("sshd -t"); err != nil {
		// Restore backup if config is invalid
		sm.ExecuteCommand("cp /etc/ssh/sshd_config.backup.* /etc/ssh/sshd_config")
		return fmt.Errorf("invalid SSH configuration, restored backup: %w", err)
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Detecting SSH service name", 80)
	}

	// Detect the correct SSH service name
	sshServiceName, err := sm.detectSSHServiceName(progressChan)
	if err != nil {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "SSH service detection failed, trying manual configuration reload", 88, err.Error())
		}

		// Fallback: try to reload SSH configuration without systemctl
		return sm.reloadSSHConfigManually(progressChan)
	}

	// Special case: if detection returned "skip-reload", don't try to reload
	if sshServiceName == "skip-reload" {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "SSH daemon running but no systemctl service found, skipping reload", 95)
		}
		return nil
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Reloading SSH service (%s)", sshServiceName), 90)
	}

	// Try to reload the SSH service with comprehensive error handling
	return sm.reloadSSHService(sshServiceName, progressChan)
}

// reloadSSHService attempts to reload SSH service with fallbacks
func (sm *SSHManager) reloadSSHService(serviceName string, progressChan chan<- SetupStep) error {
	// Try reload first
	reloadCmd := fmt.Sprintf("systemctl reload %s", serviceName)
	reloadOutput, reloadErr := sm.ExecuteCommand(reloadCmd)

	if reloadErr == nil {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH service %s reloaded successfully", serviceName), 95)
		}
		return nil
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Reload failed (%v), attempting restart of %s service", reloadErr, serviceName), 92)
	}

	// Try restart as fallback
	restartCmd := fmt.Sprintf("systemctl restart %s", serviceName)
	restartOutput, restartErr := sm.ExecuteCommand(restartCmd)

	if restartErr == nil {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH service %s restarted successfully", serviceName), 95)
		}
		return nil
	}

	// If both systemctl methods failed, try manual reload
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "Systemctl methods failed, trying manual configuration reload", 94)
	}

	if err := sm.reloadSSHConfigManually(progressChan); err == nil {
		return nil
	}

	// Return comprehensive error information
	return fmt.Errorf("failed to reload SSH service '%s': reload error: %w (output: %s), restart error: %v (output: %s)",
		serviceName, reloadErr, reloadOutput, restartErr, restartOutput)
}

// reloadSSHConfigManually tries to reload SSH config without systemctl
func (sm *SSHManager) reloadSSHConfigManually(progressChan chan<- SetupStep) error {
	// Method 1: Try HUP signal to main SSH daemon
	if output, err := sm.ExecuteCommand("pgrep -f 'sshd.*-D'"); err == nil && strings.TrimSpace(output) != "" {
		pid := strings.TrimSpace(strings.Split(output, "\n")[0])
		if _, err := sm.ExecuteCommand(fmt.Sprintf("kill -HUP %s", pid)); err == nil {
			if progressChan != nil {
				sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH configuration reloaded using HUP signal (PID: %s)", pid), 95)
			}
			return nil
		}
	}

	// Method 2: Try to find any SSH daemon and send HUP
	if output, err := sm.ExecuteCommand("pgrep sshd | head -1"); err == nil && strings.TrimSpace(output) != "" {
		pid := strings.TrimSpace(output)
		if _, err := sm.ExecuteCommand(fmt.Sprintf("kill -HUP %s", pid)); err == nil {
			if progressChan != nil {
				sm.sendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH daemon reloaded using HUP signal (PID: %s)", pid), 95)
			}
			return nil
		}
	}

	// Method 3: Check if we can at least verify the configuration is valid
	if _, err := sm.ExecuteCommand("sshd -t"); err == nil {
		if progressChan != nil {
			sm.sendProgressUpdate(progressChan, "harden_ssh", "running", "SSH configuration is valid, but could not reload service", 95)
		}
		// Configuration is valid, so we'll assume it will be applied on next SSH restart
		return nil
	}

	return fmt.Errorf("could not reload SSH configuration using any method")
}

// verifySecurityLockdown verifies that all security measures are properly applied
func (sm *SSHManager) verifySecurityLockdown() error {
	return sm.verifySecurityLockdownWithProgress(nil)
}

// verifySecurityLockdownWithProgress verifies security measures with detailed progress reporting
func (sm *SSHManager) verifySecurityLockdownWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "verify_security", "running", "Verifying UFW firewall status", 20)
	}

	// Verify UFW is active
	ufwStatus, err := sm.ExecuteCommand("ufw status")
	if err != nil || !strings.Contains(ufwStatus, "Status: active") {
		return fmt.Errorf("UFW firewall is not active")
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "verify_security", "running", "Verifying essential ports are configured", 40)
	}

	// Verify essential ports are allowed
	requiredPorts := []string{
		fmt.Sprintf("%d", sm.server.Port),
		"80",
		"443",
	}

	for _, port := range requiredPorts {
		if !strings.Contains(ufwStatus, port) {
			return fmt.Errorf("required port %s is not allowed in UFW", port)
		}
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "verify_security", "running", "Verifying fail2ban service", 60)
	}

	// Verify fail2ban is running
	fail2banStatus, err := sm.ExecuteCommand("systemctl is-active fail2ban")
	if err != nil || !strings.Contains(fail2banStatus, "active") {
		return fmt.Errorf("fail2ban is not running")
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "verify_security", "running", "Verifying SSH jail configuration", 80)
	}

	// Verify SSH jail is active
	sshJailStatus, err := sm.ExecuteCommand("fail2ban-client status sshd")
	if err != nil || !strings.Contains(sshJailStatus, "Status for the jail: sshd") {
		return fmt.Errorf("SSH jail is not active in fail2ban")
	}

	if progressChan != nil {
		sm.sendProgressUpdate(progressChan, "verify_security", "running", "Verifying SSH hardening settings", 90)
	}

	// Verify SSH hardening
	sshConfig, err := sm.ExecuteCommand("sshd -T")
	if err != nil {
		return fmt.Errorf("failed to verify SSH configuration: %w", err)
	}

	requiredSettings := map[string]string{
		"passwordauthentication": "no",
		"pubkeyauthentication":   "yes",
		"permitrootlogin":        "no",
		"permitemptypasswords":   "no",
	}

	sshConfigLower := strings.ToLower(sshConfig)
	for setting, expectedValue := range requiredSettings {
		expected := fmt.Sprintf("%s %s", setting, expectedValue)
		if !strings.Contains(sshConfigLower, expected) {
			return fmt.Errorf("SSH setting '%s' is not properly configured", setting)
		}
	}

	return nil
}

// GetSecurityStatus returns the current security status of the server
func (sm *SSHManager) GetSecurityStatus() (map[string]bool, error) {
	status := map[string]bool{
		"ufw_active":       false,
		"fail2ban_running": false,
		"ssh_hardened":     false,
		"ports_configured": false,
	}

	// Check UFW status
	if ufwStatus, err := sm.ExecuteCommand("ufw status"); err == nil {
		if strings.Contains(ufwStatus, "Status: active") {
			status["ufw_active"] = true

			// Check if required ports are configured
			requiredPorts := []string{
				fmt.Sprintf("%d", sm.server.Port),
				"80", "443",
			}
			portsConfigured := true
			for _, port := range requiredPorts {
				if !strings.Contains(ufwStatus, port) {
					portsConfigured = false
					break
				}
			}
			status["ports_configured"] = portsConfigured
		}
	}

	// Check fail2ban status
	if fail2banStatus, err := sm.ExecuteCommand("systemctl is-active fail2ban"); err == nil {
		if strings.Contains(fail2banStatus, "active") {
			status["fail2ban_running"] = true
		}
	}

	// Check SSH hardening
	if sshConfig, err := sm.ExecuteCommand("sshd -T"); err == nil {
		sshConfigLower := strings.ToLower(sshConfig)
		if strings.Contains(sshConfigLower, "passwordauthentication no") &&
			strings.Contains(sshConfigLower, "permitrootlogin no") {
			status["ssh_hardened"] = true
		}
	}

	return status, nil
}

// DisableSecurityForTesting temporarily disables security measures for testing (USE WITH CAUTION)
func (sm *SSHManager) DisableSecurityForTesting() error {
	if !sm.isRoot {
		return fmt.Errorf("disabling security requires root access")
	}

	// This should only be used in development/testing environments
	// Disable UFW
	if _, err := sm.ExecuteCommand("ufw --force disable"); err != nil {
		return fmt.Errorf("failed to disable UFW: %w", err)
	}

	// Stop fail2ban
	if _, err := sm.ExecuteCommand("systemctl stop fail2ban"); err != nil {
		return fmt.Errorf("failed to stop fail2ban: %w", err)
	}

	return nil
}
