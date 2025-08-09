package ssh

import (
	"fmt"
	"strings"
	"time"
)

// ApplySecurityLockdown applies all security hardening measures to the server
func (sm *SSHManager) ApplySecurityLockdown(progressChan chan<- SetupStep) error {
	if !sm.isRoot {
		return fmt.Errorf("security lockdown requires root access")
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"setup_firewall", sm.setupFirewall},
		{"setup_fail2ban", sm.setupFail2ban},
		{"harden_ssh", sm.hardenSSH},
		{"verify_security", sm.verifySecurityLockdown},
	}

	totalSteps := len(steps)

	for i, step := range steps {
		// Send running status
		progressChan <- SetupStep{
			Step:        step.name,
			Status:      "running",
			Message:     fmt.Sprintf("Executing %s", step.name),
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: (i * 100) / totalSteps,
		}

		if err := step.fn(); err != nil {
			// Send failure status
			progressChan <- SetupStep{
				Step:        step.name,
				Status:      "failed",
				Message:     fmt.Sprintf("Failed to execute %s", step.name),
				Details:     err.Error(),
				Timestamp:   time.Now().Format(time.RFC3339),
				ProgressPct: (i * 100) / totalSteps,
			}
			return fmt.Errorf("security step %s failed: %w", step.name, err)
		}

		// Send success status
		progressChan <- SetupStep{
			Step:        step.name,
			Status:      "success",
			Message:     fmt.Sprintf("Successfully completed %s", step.name),
			Timestamp:   time.Now().Format(time.RFC3339),
			ProgressPct: ((i + 1) * 100) / totalSteps,
		}
	}

	return nil
}

// setupFirewall configures UFW firewall with essential ports
func (sm *SSHManager) setupFirewall() error {
	// Install UFW if not already installed
	installCmd := "which ufw || (apt-get update && apt-get install -y ufw)"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install UFW: %w", err)
	}

	// Reset UFW to default state
	if _, err := sm.ExecuteCommand("ufw --force reset"); err != nil {
		return fmt.Errorf("failed to reset UFW: %w", err)
	}

	// Set default policies
	if _, err := sm.ExecuteCommand("ufw default deny incoming"); err != nil {
		return fmt.Errorf("failed to set default deny incoming: %w", err)
	}

	if _, err := sm.ExecuteCommand("ufw default allow outgoing"); err != nil {
		return fmt.Errorf("failed to set default allow outgoing: %w", err)
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
		cmd := fmt.Sprintf("ufw allow %s comment '%s'", portRule.port, portRule.description)
		if _, err := sm.ExecuteCommand(cmd); err != nil {
			return fmt.Errorf("failed to allow port %s: %w", portRule.port, err)
		}
	}

	// Enable UFW
	if _, err := sm.ExecuteCommand("ufw --force enable"); err != nil {
		return fmt.Errorf("failed to enable UFW: %w", err)
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
	// Install fail2ban
	installCmd := "which fail2ban-client || (apt-get update && apt-get install -y fail2ban)"
	if _, err := sm.ExecuteCommand(installCmd); err != nil {
		return fmt.Errorf("failed to install fail2ban: %w", err)
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

	// Restart and enable fail2ban service
	if _, err := sm.ExecuteCommand("systemctl restart fail2ban"); err != nil {
		return fmt.Errorf("failed to restart fail2ban: %w", err)
	}

	if _, err := sm.ExecuteCommand("systemctl enable fail2ban"); err != nil {
		return fmt.Errorf("failed to enable fail2ban: %w", err)
	}

	// Verify fail2ban is running
	status, err := sm.ExecuteCommand("systemctl is-active fail2ban")
	if err != nil || !strings.Contains(status, "active") {
		return fmt.Errorf("fail2ban is not running after configuration")
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
	// Backup original SSH config
	backupCmd := "cp /etc/ssh/sshd_config /etc/ssh/sshd_config.backup.$(date +%Y%m%d_%H%M%S)"
	if _, err := sm.ExecuteCommand(backupCmd); err != nil {
		return fmt.Errorf("failed to backup SSH config: %w", err)
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

	// Write the hardened configuration
	writeConfigCmd := fmt.Sprintf("cat > /etc/ssh/sshd_config << 'EOF'\n%s\nEOF", modifiedConfig)
	if _, err := sm.ExecuteCommand(writeConfigCmd); err != nil {
		return fmt.Errorf("failed to write hardened SSH config: %w", err)
	}

	// Test SSH configuration syntax
	if _, err := sm.ExecuteCommand("sshd -t"); err != nil {
		// Restore backup if config is invalid
		sm.ExecuteCommand("cp /etc/ssh/sshd_config.backup.* /etc/ssh/sshd_config")
		return fmt.Errorf("invalid SSH configuration, restored backup: %w", err)
	}

	// Reload SSH service (don't restart to avoid losing connection)
	if _, err := sm.ExecuteCommand("systemctl reload sshd"); err != nil {
		return fmt.Errorf("failed to reload SSH service: %w", err)
	}

	return nil
}

// verifySecurityLockdown verifies that all security measures are properly applied
func (sm *SSHManager) verifySecurityLockdown() error {
	// Verify UFW is active
	ufwStatus, err := sm.ExecuteCommand("ufw status")
	if err != nil || !strings.Contains(ufwStatus, "Status: active") {
		return fmt.Errorf("UFW firewall is not active")
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

	// Verify fail2ban is running
	fail2banStatus, err := sm.ExecuteCommand("systemctl is-active fail2ban")
	if err != nil || !strings.Contains(fail2banStatus, "active") {
		return fmt.Errorf("fail2ban is not running")
	}

	// Verify SSH jail is active
	sshJailStatus, err := sm.ExecuteCommand("fail2ban-client status sshd")
	if err != nil || !strings.Contains(sshJailStatus, "Status for the jail: sshd") {
		return fmt.Errorf("SSH jail is not active in fail2ban")
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
