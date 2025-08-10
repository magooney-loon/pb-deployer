package ssh

import (
	"fmt"
	"strings"
	"time"
)

// ApplySecurityLockdown performs the complete security lockdown process
func (sm *SSHManager) ApplySecurityLockdown(progressChan chan<- SetupStep) error {
	if !sm.isRoot {
		return fmt.Errorf("security lockdown requires root access")
	}

	sm.SendProgressUpdate(progressChan, "security_lockdown", "running", "Starting security lockdown process", 0)

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
		sm.SendProgressUpdate(progressChan, step.name, "running", fmt.Sprintf("Executing %s", step.name), (i*100)/totalSteps)

		if err := step.fn(progressChan); err != nil {
			// Send failure status
			sm.SendProgressUpdate(progressChan, step.name, "failed", fmt.Sprintf("Failed to execute %s", step.name), (i*100)/totalSteps, err.Error())
			return fmt.Errorf("security step %s failed: %w", step.name, err)
		}

		// Send success status
		sm.SendProgressUpdate(progressChan, step.name, "success", fmt.Sprintf("Successfully completed %s", step.name), ((i+1)*100)/totalSteps)
	}

	sm.SendProgressUpdate(progressChan, "security_lockdown", "success", "Security lockdown completed successfully", 100)
	return nil
}

// detectSSHServiceName detects the correct SSH service name on the system
func (sm *SSHManager) detectSSHServiceName(progressChan chan<- SetupStep) (string, error) {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Detecting SSH service name", 81)
	}

	// Try common SSH service names
	serviceNames := []string{"sshd", "ssh", "openssh", "openssh-server"}
	var lastError error

	for _, serviceName := range serviceNames {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Testing service name: %s", serviceName), 82)
		}

		// Method 1: Check if service is loaded (regardless of enabled status)
		cmd := fmt.Sprintf("systemctl list-units --type=service --all | grep -q '%s.service'", serviceName)
		if _, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: list-units)", serviceName), 83)
			}
			return serviceName, nil
		}

		// Method 2: Check if service unit file exists
		cmd = fmt.Sprintf("systemctl list-unit-files --type=service | grep -q '%s.service'", serviceName)
		if _, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: list-unit-files)", serviceName), 83)
			}
			return serviceName, nil
		}

		// Method 3: Check if service status can be queried (even if inactive)
		cmd = fmt.Sprintf("systemctl status %s", serviceName)
		if output, err := sm.ExecuteCommand(cmd); err == nil {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s (method: status check)", serviceName), 83)
			}
			return serviceName, nil
		} else {
			lastError = fmt.Errorf("service %s status check failed: %w (output: %s)", serviceName, err, output)
		}
	}

	// Method 4: Check for running SSH daemon process
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Checking for running SSH daemon process", 84)
	}
	if output, err := sm.ExecuteCommand("ps aux | grep sshd | grep -v grep"); err == nil && strings.TrimSpace(output) != "" {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Found running SSH daemon, checking if managed by systemctl", 85)
		}

		// Check if any of the service names can control this daemon
		for _, serviceName := range serviceNames {
			testCmd := fmt.Sprintf("systemctl is-active %s || systemctl is-failed %s || systemctl is-enabled %s", serviceName, serviceName, serviceName)
			if _, err := sm.ExecuteCommand(testCmd); err == nil {
				if progressChan != nil {
					sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH daemon is managed by systemctl as '%s'", serviceName), 87)
				}
				return serviceName, nil
			}
		}

		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "SSH daemon running but not managed by systemctl", 87)
		}
	}

	// Method 5: Check for SSH configuration file existence
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Checking for SSH configuration file", 86)
	}
	if _, err := sm.ExecuteCommand("test -f /etc/ssh/sshd_config"); err == nil {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Found SSH config file, but no running daemon detected", 87)
		}
		return "sshd", nil // Default to sshd if config exists
	}

	return "", fmt.Errorf("could not detect SSH service name. Last error: %v", lastError)
}

// setupFirewall configures UFW firewall
func (sm *SSHManager) setupFirewall() error {
	return sm.setupFirewallWithProgress(nil)
}

// setupFirewallWithProgress configures UFW firewall with detailed progress reporting
func (sm *SSHManager) setupFirewallWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Installing UFW firewall", 10)
	}

	// Install UFW if not already installed
	installCmd := "apt-get update && apt-get install -y ufw"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install UFW: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Resetting UFW to default state", 20)
	}

	// Reset UFW to ensure clean state
	resetCmd := "ufw --force reset"
	if _, err := sm.ExecuteCommand(resetCmd); err != nil {
		return fmt.Errorf("failed to reset UFW: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Setting default firewall policies", 30)
	}

	// Set default policies
	defaultCmds := []string{
		"ufw default deny incoming",
		"ufw default allow outgoing",
	}

	for _, cmd := range defaultCmds {
		if _, err := sm.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to set default policy: %w", err)
		}
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Configuring essential ports", 50)
	}

	// Configure essential ports
	portRules := []struct {
		port        string
		description string
	}{
		{fmt.Sprintf("%d", sm.server.Port), "SSH"},
		{"80", "HTTP"},
		{"443", "HTTPS"},
	}

	for _, portRule := range portRules {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "setup_firewall", "running", fmt.Sprintf("Allowing port %s (%s)", portRule.port, portRule.description), 50)
		}

		allowCmd := fmt.Sprintf("ufw allow %s", portRule.port)
		if _, err := sm.ExecuteCommand(allowCmd); err != nil {
			return fmt.Errorf("failed to allow port %s: %w", portRule.port, err)
		}
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Enabling UFW firewall", 80)
	}

	// Enable UFW
	enableCmd := "ufw --force enable"
	if _, err := sm.ExecuteCommand(enableCmd); err != nil {
		return fmt.Errorf("failed to enable UFW: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_firewall", "running", "Verifying firewall status", 90)
	}

	// Verify UFW is active
	statusCmd := "ufw status"
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil {
		return fmt.Errorf("failed to check UFW status: %w", err)
	}

	if !strings.Contains(output, "Status: active") {
		return fmt.Errorf("UFW is not active after enabling")
	}

	return nil
}

// setupFail2ban configures fail2ban intrusion prevention
func (sm *SSHManager) setupFail2ban() error {
	return sm.setupFail2banWithProgress(nil)
}

// setupFail2banWithProgress configures fail2ban with detailed progress reporting
func (sm *SSHManager) setupFail2banWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_fail2ban", "running", "Installing fail2ban", 10)
	}

	// Install fail2ban
	installCmd := "apt-get update && apt-get install -y fail2ban"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install fail2ban: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_fail2ban", "running", "Configuring fail2ban for SSH protection", 30)
	}

	// Create jail.local configuration
	jailConfig := `[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 3

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600
`

	configCmd := fmt.Sprintf("cat > /etc/fail2ban/jail.local << 'EOF'\n%sEOF", jailConfig)
	if _, err := sm.ExecuteCommand(configCmd); err != nil {
		return fmt.Errorf("failed to create fail2ban configuration: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_fail2ban", "running", "Starting fail2ban service", 60)
	}

	// Enable and start fail2ban
	enableCmd := "systemctl enable fail2ban"
	if _, err := sm.ExecuteCommand(enableCmd); err != nil {
		return fmt.Errorf("failed to enable fail2ban: %w", err)
	}

	startCmd := "systemctl start fail2ban"
	if _, err := sm.ExecuteCommand(startCmd); err != nil {
		return fmt.Errorf("failed to start fail2ban: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_fail2ban", "running", "Verifying fail2ban status", 80)
	}

	// Verify fail2ban is running
	statusCmd := "systemctl is-active fail2ban"
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil || !strings.Contains(output, "active") {
		return fmt.Errorf("fail2ban is not running after setup")
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_fail2ban", "running", "Checking fail2ban SSH jail status", 90)
	}

	// Check if SSH jail is active
	jailCmd := "fail2ban-client status sshd"
	if _, err := sm.ExecuteCommand(jailCmd); err != nil {
		return fmt.Errorf("SSH jail is not active in fail2ban: %w", err)
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
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Starting SSH hardening process", 5)
	}

	// SSH hardening settings
	sshSettings := map[string]string{
		"PermitRootLogin":                 "no",
		"PasswordAuthentication":          "no",
		"PubkeyAuthentication":            "yes",
		"X11Forwarding":                   "no",
		"AllowAgentForwarding":            "no",
		"AllowTcpForwarding":              "no",
		"ClientAliveInterval":             "300",
		"ClientAliveCountMax":             "2",
		"MaxAuthTries":                    "3",
		"MaxSessions":                     "2",
		"Protocol":                        "2",
		"IgnoreRhosts":                    "yes",
		"HostbasedAuthentication":         "no",
		"PermitEmptyPasswords":            "no",
		"ChallengeResponseAuthentication": "no",
		"KerberosAuthentication":          "no",
		"GSSAPIAuthentication":            "no",
	}

	totalSettings := len(sshSettings)
	i := 0

	for setting, value := range sshSettings {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Configuring SSH setting: %s = %s", setting, value), 10+(i*70)/totalSettings)
		}

		// Check if setting already exists in config
		checkCmd := fmt.Sprintf("grep -q '^%s' /etc/ssh/sshd_config", setting)
		if _, err := sm.ExecuteCommand(checkCmd); err == nil {
			// Setting exists, update it
			updateCmd := fmt.Sprintf("sed -i 's/^%s.*/%s %s/' /etc/ssh/sshd_config", setting, setting, value)
			if _, err := sm.ExecuteCommand(updateCmd); err != nil {
				return fmt.Errorf("failed to update SSH setting %s: %w", setting, err)
			}
		} else {
			// Setting doesn't exist, add it
			addCmd := fmt.Sprintf("echo '%s %s' >> /etc/ssh/sshd_config", setting, value)
			if _, err := sm.ExecuteCommand(addCmd); err != nil {
				return fmt.Errorf("failed to add SSH setting %s: %w", setting, err)
			}
		}
		i++
	}

	// Detect SSH service name
	serviceName, err := sm.detectSSHServiceName(progressChan)
	if err != nil {
		return fmt.Errorf("failed to detect SSH service name: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Reloading SSH service: %s", serviceName), 90)
	}

	// Reload SSH service to apply changes
	if err := sm.reloadSSHService(serviceName, progressChan); err != nil {
		return fmt.Errorf("failed to reload SSH service: %w", err)
	}

	return nil
}

// reloadSSHService reloads the SSH service configuration
func (sm *SSHManager) reloadSSHService(serviceName string, progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Testing SSH configuration syntax", 91)
	}

	// Test SSH configuration syntax before reloading
	testCmd := "sshd -t"
	if _, err := sm.ExecuteCommand(testCmd); err != nil {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "SSH config test failed, attempting manual reload", 92)
		}
		// If syntax test fails, try manual config reload approach
		return sm.reloadSSHConfigManually(progressChan)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Reloading SSH service: %s", serviceName), 93)
	}

	// Reload the SSH service
	reloadCmd := fmt.Sprintf("systemctl reload %s", serviceName)
	if _, err := sm.ExecuteCommand(reloadCmd); err != nil {
		// If reload fails, try restart
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Reload failed, attempting restart", 94)
		}
		restartCmd := fmt.Sprintf("systemctl restart %s", serviceName)
		if _, err := sm.ExecuteCommand(restartCmd); err != nil {
			return fmt.Errorf("failed to restart SSH service: %w", err)
		}
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Verifying SSH service is running", 96)
	}

	// Verify service is still running
	statusCmd := fmt.Sprintf("systemctl is-active %s", serviceName)
	output, err := sm.ExecuteCommand(statusCmd)
	if err != nil || !strings.Contains(output, "active") {
		return fmt.Errorf("SSH service is not active after reload")
	}

	return nil
}

// reloadSSHConfigManually attempts to reload SSH config without systemctl
func (sm *SSHManager) reloadSSHConfigManually(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Attempting manual SSH config reload", 93)
	}

	// Try to find SSH daemon process and send SIGHUP
	findPidCmd := "pgrep sshd | head -1"
	pidOutput, err := sm.ExecuteCommand(findPidCmd)
	if err != nil {
		return fmt.Errorf("failed to find SSH daemon process: %w", err)
	}

	pid := strings.TrimSpace(pidOutput)
	if pid == "" {
		return fmt.Errorf("no SSH daemon process found")
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Sending SIGHUP to SSH daemon (PID: %s)", pid), 95)
	}

	// Send SIGHUP to reload configuration
	reloadCmd := fmt.Sprintf("kill -HUP %s", pid)
	if _, err := sm.ExecuteCommand(reloadCmd); err != nil {
		return fmt.Errorf("failed to send SIGHUP to SSH daemon: %w", err)
	}

	// Wait a moment for reload
	time.Sleep(2 * time.Second)

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Verifying SSH daemon is still running", 97)
	}

	// Verify daemon is still running
	checkCmd := fmt.Sprintf("kill -0 %s", pid)
	if _, err := sm.ExecuteCommand(checkCmd); err != nil {
		return fmt.Errorf("SSH daemon stopped after configuration reload")
	}

	return nil
}

// verifySecurityLockdown verifies all security measures are in place
func (sm *SSHManager) verifySecurityLockdown() error {
	return sm.verifySecurityLockdownWithProgress(nil)
}

// verifySecurityLockdownWithProgress verifies security with detailed progress reporting
func (sm *SSHManager) verifySecurityLockdownWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "verify_security", "running", "Verifying UFW firewall status", 20)
	}

	// Check UFW status
	ufwCmd := "ufw status"
	ufwOutput, err := sm.ExecuteCommand(ufwCmd)
	if err != nil {
		return fmt.Errorf("failed to check UFW status: %w", err)
	}

	if !strings.Contains(ufwOutput, "Status: active") {
		return fmt.Errorf("UFW firewall is not active")
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "verify_security", "running", "Verifying fail2ban status", 40)
	}

	// Check fail2ban status
	fail2banCmd := "systemctl is-active fail2ban"
	fail2banOutput, err := sm.ExecuteCommand(fail2banCmd)
	if err != nil || !strings.Contains(fail2banOutput, "active") {
		return fmt.Errorf("fail2ban is not active")
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "verify_security", "running", "Verifying SSH hardening configuration", 60)
	}

	// Check key SSH hardening settings
	criticalSettings := []string{
		"PermitRootLogin no",
		"PasswordAuthentication no",
		"PubkeyAuthentication yes",
	}

	for _, setting := range criticalSettings {
		checkCmd := fmt.Sprintf("grep -q '^%s' /etc/ssh/sshd_config", setting)
		if _, err := sm.ExecuteCommand(checkCmd); err != nil {
			return fmt.Errorf("critical SSH setting not found: %s", setting)
		}
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "verify_security", "running", "Verifying SSH service is running", 80)
	}

	// Verify SSH service is running
	serviceName, err := sm.detectSSHServiceName(nil)
	if err != nil {
		return fmt.Errorf("failed to detect SSH service: %w", err)
	}

	statusCmd := fmt.Sprintf("systemctl is-active %s", serviceName)
	statusOutput, err := sm.ExecuteCommand(statusCmd)
	if err != nil || !strings.Contains(statusOutput, "active") {
		return fmt.Errorf("SSH service is not active")
	}

	return nil
}

// GetSecurityStatus returns the current security status of the server
func (sm *SSHManager) GetSecurityStatus() (map[string]bool, error) {
	status := map[string]bool{
		"firewall_active":     false,
		"fail2ban_active":     false,
		"ssh_hardened":        false,
		"root_login_disabled": false,
	}

	// Check UFW status
	if ufwOutput, err := sm.ExecuteCommand("ufw status"); err == nil {
		status["firewall_active"] = strings.Contains(ufwOutput, "Status: active")
	}

	// Check fail2ban status
	if fail2banOutput, err := sm.ExecuteCommand("systemctl is-active fail2ban"); err == nil {
		status["fail2ban_active"] = strings.Contains(fail2banOutput, "active")
	}

	// Check SSH hardening
	if _, err := sm.ExecuteCommand("grep -q '^PermitRootLogin no' /etc/ssh/sshd_config"); err == nil {
		status["root_login_disabled"] = true
	}

	if _, err := sm.ExecuteCommand("grep -q '^PasswordAuthentication no' /etc/ssh/sshd_config"); err == nil {
		status["ssh_hardened"] = true
	}

	return status, nil
}

// DisableSecurityForTesting temporarily disables security measures for testing purposes
func (sm *SSHManager) DisableSecurityForTesting() error {
	if !sm.isRoot {
		return fmt.Errorf("disabling security requires root access")
	}

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
