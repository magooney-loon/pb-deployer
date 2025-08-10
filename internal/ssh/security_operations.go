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
		{"validate_app_user", sm.validateAppUserConnectionWithProgress},
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

	// Try common SSH service names in order of likelihood
	serviceNames := []string{"ssh", "sshd", "openssh-server", "openssh"}
	var detectionErrors []string

	for _, serviceName := range serviceNames {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Testing service name: %s", serviceName), 82)
		}

		// Method 1: Check with and without .service suffix
		for _, variant := range []string{serviceName + ".service", serviceName} {
			statusCmd := fmt.Sprintf("systemctl status %s 2>/dev/null", variant)
			if output, err := sm.ExecuteCommand(statusCmd); err == nil || !strings.Contains(output, "not found") {
				if progressChan != nil {
					sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service: %s", serviceName), 83)
				}
				return serviceName, nil
			}
		}

		// Method 2: Check if service unit file exists
		unitFileCmd := fmt.Sprintf("systemctl list-unit-files 2>/dev/null | grep -E '^%s\\.service|^%s\\s'", serviceName, serviceName)
		if output, err := sm.ExecuteCommand(unitFileCmd); err == nil && strings.TrimSpace(output) != "" {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found SSH service unit file: %s", serviceName), 83)
			}
			return serviceName, nil
		}

		detectionErrors = append(detectionErrors, fmt.Sprintf("%s: not found in systemctl", serviceName))
	}

	// Method 3: Check for running SSH daemon process
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Checking for running SSH daemon process", 84)
	}
	if output, err := sm.ExecuteCommand("ps aux | grep '[s]shd' | head -1"); err == nil && strings.TrimSpace(output) != "" {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Found running SSH daemon, checking system type", 85)
		}

		// Determine likely service name based on OS
		if _, err := sm.ExecuteCommand("test -f /etc/debian_version"); err == nil {
			// Debian/Ubuntu systems typically use 'ssh'
			return "ssh", nil
		}
		if _, err := sm.ExecuteCommand("test -f /etc/redhat-release"); err == nil {
			// RHEL/CentOS systems typically use 'sshd'
			return "sshd", nil
		}

		// Default for other systems
		return "ssh", nil
	}

	// Method 4: Check for SSH configuration file
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Checking SSH config file", 86)
	}
	if _, err := sm.ExecuteCommand("test -f /etc/ssh/sshd_config"); err == nil {
		// Try to determine service name from OS type
		if _, err := sm.ExecuteCommand("which apt-get"); err == nil {
			return "ssh", nil // Debian/Ubuntu
		}
		if _, err := sm.ExecuteCommand("which yum"); err == nil {
			return "sshd", nil // RHEL/CentOS
		}
		return "ssh", nil // Default for modern systems
	}

	// If we can't detect anything, warn but continue with 'ssh' as default
	if progressChan != nil {
		errorDetails := strings.Join(detectionErrors, "; ")
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Could not detect SSH service. Will try 'ssh' as default. Errors: %s", errorDetails), 87)
	}
	return "ssh", nil
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

	// Apply PermitRootLogin setting last, after ensuring app user works
	rootLoginSettings := map[string]string{
		"PermitRootLogin": "no",
	}

	totalSettings := len(sshSettings) + len(rootLoginSettings)
	i := 0

	// Apply general SSH hardening settings first
	for setting, value := range sshSettings {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Configuring SSH setting: %s = %s", setting, value), 10+(i*60)/totalSettings)
		}

		if err := sm.applySSHSetting(setting, value); err != nil {
			return fmt.Errorf("failed to apply SSH setting %s: %w", setting, err)
		}
		i++
	}

	// Test SSH service reload before applying root login restriction
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Testing SSH configuration before disabling root login", 70)
	}

	serviceName, err := sm.detectSSHServiceName(progressChan)
	if err != nil {
		return fmt.Errorf("failed to detect SSH service name: %w", err)
	}

	if err := sm.testSSHConfigAndReload(serviceName, progressChan); err != nil {
		return fmt.Errorf("SSH configuration test failed: %w", err)
	}

	// Apply root login restriction last
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Applying final security settings (disabling root login)", 80)
	}

	for setting, value := range rootLoginSettings {
		if err := sm.applySSHSetting(setting, value); err != nil {
			return fmt.Errorf("failed to apply SSH setting %s: %w", setting, err)
		}
		i++
	}

	// Final SSH service reload
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Final SSH service reload: %s", serviceName), 90)
	}

	if err := sm.reloadSSHService(serviceName, progressChan); err != nil {
		return fmt.Errorf("failed to reload SSH service: %w", err)
	}

	// Warn about root access being disabled
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "SSH hardening complete - root login now disabled", 95)
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
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Attempting to reload SSH service: %s", serviceName), 93)
	}

	// Try different service names and methods
	serviceVariants := []string{serviceName + ".service", serviceName}

	for _, variant := range serviceVariants {
		// First verify the service exists
		checkCmd := fmt.Sprintf("systemctl status %s 2>/dev/null", variant)
		if output, err := sm.ExecuteCommand(checkCmd); err == nil || !strings.Contains(output, "not found") {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("Found service as: %s", variant), 94)
			}

			// Try reload first (graceful)
			reloadCmd := fmt.Sprintf("systemctl reload %s", variant)
			if _, err := sm.ExecuteCommand(reloadCmd); err == nil {
				// Verify service is still running
				statusCmd := fmt.Sprintf("systemctl is-active %s", variant)
				if output, err := sm.ExecuteCommand(statusCmd); err == nil && strings.Contains(output, "active") {
					if progressChan != nil {
						sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH service %s successfully reloaded", variant), 98)
					}
					return nil
				}
			}

			// If reload fails, try restart
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Reload failed, attempting restart", 95)
			}
			restartCmd := fmt.Sprintf("systemctl restart %s", variant)
			if _, err := sm.ExecuteCommand(restartCmd); err == nil {
				// Verify service is still running
				statusCmd := fmt.Sprintf("systemctl is-active %s", variant)
				if output, err := sm.ExecuteCommand(statusCmd); err == nil && strings.Contains(output, "active") {
					if progressChan != nil {
						sm.SendProgressUpdate(progressChan, "harden_ssh", "running", fmt.Sprintf("SSH service %s successfully restarted", variant), 98)
					}
					return nil
				}
			}
		}
	}

	// If systemctl methods fail, try manual reload
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Systemctl methods failed, trying manual reload", 96)
	}

	return sm.reloadSSHConfigManually(progressChan)
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

// validateAppUserConnectionWithProgress validates that app user can connect before disabling root
func (sm *SSHManager) validateAppUserConnectionWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "validate_app_user", "running", "Testing app user SSH connection", 10)
	}

	// Create a test SSH connection as the app user
	testManager, err := NewSSHManager(sm.server, false)
	if err != nil {
		return fmt.Errorf("failed to create app user connection: %w", err)
	}
	defer testManager.Close()

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "validate_app_user", "running", "Verifying app user can execute commands", 50)
	}

	// Test basic command execution
	if err := testManager.TestConnection(); err != nil {
		return fmt.Errorf("app user connection test failed: %w", err)
	}

	// Test sudo access for deployment commands
	testSudoCmd := "sudo -n systemctl --version"
	if _, err := testManager.ExecuteCommand(testSudoCmd); err != nil {
		return fmt.Errorf("app user sudo access test failed: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "validate_app_user", "running", "App user connection validated successfully", 90)
	}

	return nil
}

// applySSHSetting applies a single SSH configuration setting
func (sm *SSHManager) applySSHSetting(setting, value string) error {
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
	return nil
}

// testSSHConfigAndReload tests SSH configuration and reloads if valid
func (sm *SSHManager) testSSHConfigAndReload(serviceName string, progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Testing SSH configuration syntax", 75)
	}

	// Test SSH configuration syntax
	testCmd := "sshd -t"
	if _, err := sm.ExecuteCommand(testCmd); err != nil {
		return fmt.Errorf("SSH configuration syntax test failed: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "harden_ssh", "running", "Reloading SSH service to apply changes", 78)
	}

	// Reload SSH service to apply changes
	if err := sm.reloadSSHService(serviceName, progressChan); err != nil {
		return fmt.Errorf("failed to reload SSH service: %w", err)
	}

	return nil
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
