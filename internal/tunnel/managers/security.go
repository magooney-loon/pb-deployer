package managers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pb-deployer/internal/tunnel"
)

// securityManager implements the SecurityManager interface
type securityManager struct {
	executor tunnel.Executor
	tracer   tunnel.ServiceTracer
	config   tunnel.SecurityConfig
}

// NewSecurityManager creates a new security manager with default configuration
func NewSecurityManager(executor tunnel.Executor, tracer tunnel.ServiceTracer) tunnel.SecurityManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &securityManager{
		executor: executor,
		tracer:   tracer,
		config:   defaultSecurityConfig(),
	}
}

// NewSecurityManagerWithConfig creates a new security manager with custom configuration
func NewSecurityManagerWithConfig(executor tunnel.Executor, tracer tunnel.ServiceTracer, config tunnel.SecurityConfig) tunnel.SecurityManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &securityManager{
		executor: executor,
		tracer:   tracer,
		config:   config,
	}
}

// ApplyLockdown applies comprehensive security lockdown configuration
func (sm *securityManager) ApplyLockdown(ctx context.Context, config tunnel.SecurityConfig) error {
	span := sm.tracer.TraceSecurityOperation(ctx, "apply_lockdown", "system")
	defer span.End()

	span.SetFields(map[string]any{
		"disable_root_login":    config.DisableRootLogin,
		"disable_password_auth": config.DisablePasswordAuth,
		"firewall_rules":        len(config.FirewallRules),
		"allowed_ports":         config.AllowedPorts,
		"allowed_users":         config.AllowedUsers,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "running",
		Message:     "Starting security lockdown",
		ProgressPct: 5,
		Timestamp:   time.Now(),
	})

	// 1. Harden SSH configuration
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "running",
		Message:     "Hardening SSH configuration",
		ProgressPct: 20,
		Timestamp:   time.Now(),
	})

	if err := sm.HardenSSH(ctx, config.SSHHardeningConfig); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("harden_ssh", "ssh", err)
	}

	// 2. Configure firewall
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "running",
		Message:     "Configuring firewall",
		ProgressPct: 50,
		Timestamp:   time.Now(),
	})

	if err := sm.SetupFirewall(ctx, config.FirewallRules); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("setup_firewall", "firewall", err)
	}

	// 3. Setup fail2ban
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "running",
		Message:     "Setting up fail2ban",
		ProgressPct: 75,
		Timestamp:   time.Now(),
	})

	if config.Fail2banConfig.Enabled {
		if err := sm.SetupFail2ban(ctx, config.Fail2banConfig); err != nil {
			span.EndWithError(err)
			return tunnel.WrapSecurityError("setup_fail2ban", "fail2ban", err)
		}
	}

	// 4. Apply additional security measures
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "running",
		Message:     "Applying additional security measures",
		ProgressPct: 90,
		Timestamp:   time.Now(),
	})

	if err := sm.applyAdditionalSecurity(ctx, config); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("additional_security", "system", err)
	}

	span.Event("lockdown_completed", map[string]any{
		"ssh_hardened":   true,
		"firewall_setup": true,
		"fail2ban_setup": config.Fail2banConfig.Enabled,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "apply_lockdown",
		Status:      "success",
		Message:     "Security lockdown completed successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// SetupFirewall configures firewall rules using UFW or iptables
func (sm *securityManager) SetupFirewall(ctx context.Context, rules []tunnel.FirewallRule) error {
	span := sm.tracer.TraceSecurityOperation(ctx, "setup_firewall", "firewall")
	defer span.End()

	span.SetFields(map[string]any{
		"rule_count": len(rules),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_firewall",
		Status:      "running",
		Message:     "Detecting firewall system",
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Detect available firewall system
	firewallType, err := sm.detectFirewallSystem(ctx)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("detect_firewall", "firewall", err)
	}

	span.Event("firewall_detected", map[string]any{
		"type": firewallType,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_firewall",
		Status:      "running",
		Message:     fmt.Sprintf("Configuring %s firewall", firewallType),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	switch firewallType {
	case "ufw":
		return sm.setupUFWFirewall(ctx, rules, span)
	case "iptables":
		return sm.setupIPTablesFirewall(ctx, rules, span)
	case "firewalld":
		return sm.setupFirewalldFirewall(ctx, rules, span)
	default:
		err := fmt.Errorf("unsupported firewall system: %s", firewallType)
		span.EndWithError(err)
		return tunnel.WrapSecurityError("unsupported_firewall", "firewall", err)
	}
}

// SetupFail2ban configures fail2ban for intrusion prevention
func (sm *securityManager) SetupFail2ban(ctx context.Context, config tunnel.Fail2banConfig) error {
	span := sm.tracer.TraceSecurityOperation(ctx, "setup_fail2ban", "fail2ban")
	defer span.End()

	span.SetFields(map[string]any{
		"enabled":     config.Enabled,
		"max_retries": config.MaxRetries,
		"ban_time":    config.BanTime.Seconds(),
		"find_time":   config.FindTime.Seconds(),
		"services":    config.Services,
	})

	if !config.Enabled {
		span.Event("fail2ban_disabled")
		return nil
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_fail2ban",
		Status:      "running",
		Message:     "Installing fail2ban",
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Install fail2ban if not present
	if err := sm.installFail2ban(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("install_fail2ban", "fail2ban", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_fail2ban",
		Status:      "running",
		Message:     "Configuring fail2ban",
		ProgressPct: 40,
		Timestamp:   time.Now(),
	})

	// Create jail.local configuration
	if err := sm.createFail2banConfig(ctx, config); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("create_config", "fail2ban", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_fail2ban",
		Status:      "running",
		Message:     "Starting fail2ban service",
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	// Start and enable fail2ban service
	if err := sm.startFail2banService(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("start_service", "fail2ban", err)
	}

	span.Event("fail2ban_configured", map[string]any{
		"config_created":  true,
		"service_started": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_fail2ban",
		Status:      "success",
		Message:     "Fail2ban configured successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// HardenSSH applies SSH security hardening configuration
func (sm *securityManager) HardenSSH(ctx context.Context, settings tunnel.SSHHardeningConfig) error {
	span := sm.tracer.TraceSecurityOperation(ctx, "harden_ssh", "ssh")
	defer span.End()

	span.SetFields(map[string]any{
		"password_auth":         settings.PasswordAuthentication,
		"pubkey_auth":           settings.PubkeyAuthentication,
		"permit_root_login":     settings.PermitRootLogin,
		"max_auth_tries":        settings.MaxAuthTries,
		"client_alive_interval": settings.ClientAliveInterval,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "harden_ssh",
		Status:      "running",
		Message:     "Backing up SSH configuration",
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Create backup of existing configuration
	if err := sm.backupSSHConfig(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("backup_config", "ssh", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "harden_ssh",
		Status:      "running",
		Message:     "Applying SSH hardening configuration",
		ProgressPct: 50,
		Timestamp:   time.Now(),
	})

	// Apply hardening configuration
	if err := sm.applySSHHardeningConfig(ctx, settings); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("apply_config", "ssh", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "harden_ssh",
		Status:      "running",
		Message:     "Validating SSH configuration",
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	// Validate configuration
	if err := sm.validateSSHConfig(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("validate_config", "ssh", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "harden_ssh",
		Status:      "running",
		Message:     "Restarting SSH service",
		ProgressPct: 90,
		Timestamp:   time.Now(),
	})

	// Restart SSH service to apply changes
	if err := sm.restartSSHService(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("restart_service", "ssh", err)
	}

	span.Event("ssh_hardened", map[string]any{
		"config_applied":    true,
		"service_restarted": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "harden_ssh",
		Status:      "success",
		Message:     "SSH hardening completed successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// Helper methods for firewall configuration

func (sm *securityManager) detectFirewallSystem(ctx context.Context) (string, error) {
	// Check for UFW
	cmd := tunnel.Command{
		Cmd:     "which ufw",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}
	result, err := sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return "ufw", nil
	}

	// Check for firewalld
	cmd = tunnel.Command{
		Cmd:     "which firewall-cmd",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return "firewalld", nil
	}

	// Check for iptables
	cmd = tunnel.Command{
		Cmd:     "which iptables",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return "iptables", nil
	}

	return "", fmt.Errorf("no supported firewall system found")
}

func (sm *securityManager) setupUFWFirewall(ctx context.Context, rules []tunnel.FirewallRule, span tunnel.ServiceSpan) error {
	// Reset UFW to default state
	cmd := tunnel.Command{
		Cmd:     "ufw --force reset",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to reset UFW: %s", result.Output)
	}

	// Set default policies
	cmd = tunnel.Command{
		Cmd:     "ufw default deny incoming",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	cmd = tunnel.Command{
		Cmd:     "ufw default allow outgoing",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Apply rules
	for i, rule := range rules {
		progress := int(float64(i+1)/float64(len(rules))*40) + 30
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "setup_firewall",
			Status:      "running",
			Message:     fmt.Sprintf("Applying rule %d/%d", i+1, len(rules)),
			ProgressPct: progress,
			Timestamp:   time.Now(),
		})

		if err := sm.applyUFWRule(ctx, rule); err != nil {
			return err
		}
	}

	// Enable UFW
	cmd = tunnel.Command{
		Cmd:     "ufw --force enable",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	span.Event("ufw_configured", map[string]any{
		"rules_applied": len(rules),
		"enabled":       true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_firewall",
		Status:      "success",
		Message:     "UFW firewall configured successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

func (sm *securityManager) applyUFWRule(ctx context.Context, rule tunnel.FirewallRule) error {
	var cmdStr string

	if rule.Source != "" {
		cmdStr = fmt.Sprintf("ufw %s from %s to any port %d proto %s",
			rule.Action, rule.Source, rule.Port, rule.Protocol)
	} else {
		cmdStr = fmt.Sprintf("ufw %s %d/%s", rule.Action, rule.Port, rule.Protocol)
	}

	cmd := tunnel.Command{
		Cmd:     cmdStr,
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to apply UFW rule: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) setupIPTablesFirewall(ctx context.Context, rules []tunnel.FirewallRule, span tunnel.ServiceSpan) error {
	// Flush existing rules
	cmd := tunnel.Command{
		Cmd:     "iptables -F",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Set default policies
	cmd = tunnel.Command{
		Cmd:     "iptables -P INPUT DROP",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	cmd = tunnel.Command{
		Cmd:     "iptables -P FORWARD DROP",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	cmd = tunnel.Command{
		Cmd:     "iptables -P OUTPUT ACCEPT",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Allow loopback
	cmd = tunnel.Command{
		Cmd:     "iptables -A INPUT -i lo -j ACCEPT",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Allow established connections
	cmd = tunnel.Command{
		Cmd:     "iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Apply custom rules
	for _, rule := range rules {
		if err := sm.applyIPTablesRule(ctx, rule); err != nil {
			return err
		}
	}

	span.Event("iptables_configured", map[string]any{
		"rules_applied": len(rules),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_firewall",
		Status:      "success",
		Message:     "IPTables firewall configured successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

func (sm *securityManager) applyIPTablesRule(ctx context.Context, rule tunnel.FirewallRule) error {
	var cmdStr string
	action := "ACCEPT"
	if rule.Action == "deny" {
		action = "DROP"
	}

	if rule.Source != "" {
		cmdStr = fmt.Sprintf("iptables -A INPUT -p %s --dport %d -s %s -j %s",
			rule.Protocol, rule.Port, rule.Source, action)
	} else {
		cmdStr = fmt.Sprintf("iptables -A INPUT -p %s --dport %d -j %s",
			rule.Protocol, rule.Port, action)
	}

	cmd := tunnel.Command{
		Cmd:     cmdStr,
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to apply iptables rule: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) setupFirewalldFirewall(ctx context.Context, rules []tunnel.FirewallRule, span tunnel.ServiceSpan) error {
	// Start firewalld service
	cmd := tunnel.Command{
		Cmd:     "systemctl start firewalld",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Set default zone to drop
	cmd = tunnel.Command{
		Cmd:     "firewall-cmd --set-default-zone=drop",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Apply rules
	for _, rule := range rules {
		if err := sm.applyFirewalldRule(ctx, rule); err != nil {
			return err
		}
	}

	// Reload to apply changes
	cmd = tunnel.Command{
		Cmd:     "firewall-cmd --reload",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	_, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	span.Event("firewalld_configured", map[string]any{
		"rules_applied": len(rules),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_firewall",
		Status:      "success",
		Message:     "Firewalld configured successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

func (sm *securityManager) applyFirewalldRule(ctx context.Context, rule tunnel.FirewallRule) error {
	var cmdStr string

	if rule.Action == "allow" {
		if rule.Source != "" {
			cmdStr = fmt.Sprintf("firewall-cmd --permanent --add-rich-rule='rule family=\"ipv4\" source address=\"%s\" port protocol=\"%s\" port=\"%d\" accept'",
				rule.Source, rule.Protocol, rule.Port)
		} else {
			cmdStr = fmt.Sprintf("firewall-cmd --permanent --add-port=%d/%s", rule.Port, rule.Protocol)
		}
	}

	if cmdStr != "" {
		cmd := tunnel.Command{
			Cmd:     cmdStr,
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, cmd)
		if err != nil {
			return err
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to apply firewalld rule: %s", result.Output)
		}
	}

	return nil
}

// Helper methods for fail2ban configuration

func (sm *securityManager) installFail2ban(ctx context.Context) error {
	// Check if fail2ban is already installed
	cmd := tunnel.Command{
		Cmd:     "which fail2ban-client",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}
	result, err := sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return nil // Already installed
	}

	// Try to install fail2ban using package manager
	packageManagers := []struct {
		check   string
		install string
	}{
		{"which apt", "apt update && apt install -y fail2ban"},
		{"which yum", "yum install -y fail2ban"},
		{"which dnf", "dnf install -y fail2ban"},
		{"which pacman", "pacman -S --noconfirm fail2ban"},
		{"which zypper", "zypper install -y fail2ban"},
	}

	for _, pm := range packageManagers {
		cmd = tunnel.Command{
			Cmd:     pm.check,
			Sudo:    false,
			Timeout: 10 * time.Second,
		}
		result, err = sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 {
			// Found package manager, install fail2ban
			cmd = tunnel.Command{
				Cmd:     pm.install,
				Sudo:    true,
				Timeout: 5 * time.Minute,
			}
			result, err = sm.executor.RunCommand(ctx, cmd)
			if err != nil {
				return err
			}
			if result.ExitCode != 0 {
				return fmt.Errorf("failed to install fail2ban: %s", result.Output)
			}
			return nil
		}
	}

	return fmt.Errorf("could not install fail2ban: no supported package manager found")
}

func (sm *securityManager) createFail2banConfig(ctx context.Context, config tunnel.Fail2banConfig) error {
	jailConfig := sm.buildFail2banJailConfig(config)
	jailPath := filepath.Join(tunnel.Fail2banConfigPath, tunnel.Fail2banJailLocal)

	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", shellEscape(jailPath), jailConfig),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to create fail2ban configuration: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) buildFail2banJailConfig(config tunnel.Fail2banConfig) string {
	var lines []string

	lines = append(lines, "[DEFAULT]")
	lines = append(lines, fmt.Sprintf("bantime = %d", int(config.BanTime.Seconds())))
	lines = append(lines, fmt.Sprintf("findtime = %d", int(config.FindTime.Seconds())))
	lines = append(lines, fmt.Sprintf("maxretry = %d", config.MaxRetries))
	lines = append(lines, "")

	// SSH jail
	if containsService(config.Services, "ssh") || containsService(config.Services, "sshd") {
		lines = append(lines, "[sshd]")
		lines = append(lines, "enabled = true")
		lines = append(lines, "port = ssh")
		lines = append(lines, "logpath = /var/log/auth.log")
		lines = append(lines, "backend = systemd")
		lines = append(lines, "")
	}

	// Add other services as needed
	for _, service := range config.Services {
		if service != "ssh" && service != "sshd" {
			lines = append(lines, fmt.Sprintf("[%s]", service))
			lines = append(lines, "enabled = true")
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

func (sm *securityManager) startFail2banService(ctx context.Context) error {
	// Enable fail2ban service
	cmd := tunnel.Command{
		Cmd:     "systemctl enable fail2ban",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	// Start fail2ban service
	cmd = tunnel.Command{
		Cmd:     "systemctl start fail2ban",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}
	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start fail2ban service: %s", result.Output)
	}

	return nil
}

// Helper methods for SSH hardening

func (sm *securityManager) backupSSHConfig(ctx context.Context) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cp %s %s", tunnel.SSHDConfigPath, tunnel.SSHDConfigBackupPath),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to backup SSH config: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) applySSHHardeningConfig(ctx context.Context, settings tunnel.SSHHardeningConfig) error {
	config := sm.buildSSHDConfig(settings)

	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", tunnel.SSHDConfigPath, config),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to apply SSH config: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) buildSSHDConfig(settings tunnel.SSHHardeningConfig) string {
	var lines []string

	lines = append(lines, "# SSH Daemon Configuration")
	lines = append(lines, "# Generated by pb-deployer security manager")
	lines = append(lines, fmt.Sprintf("# Generated at %s", time.Now().Format(time.RFC3339)))
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("Protocol %d", settings.Protocol))
	lines = append(lines, fmt.Sprintf("PasswordAuthentication %s", boolToYesNo(settings.PasswordAuthentication)))
	lines = append(lines, fmt.Sprintf("PubkeyAuthentication %s", boolToYesNo(settings.PubkeyAuthentication)))
	lines = append(lines, fmt.Sprintf("PermitRootLogin %s", boolToYesNo(settings.PermitRootLogin)))
	lines = append(lines, fmt.Sprintf("X11Forwarding %s", boolToYesNo(settings.X11Forwarding)))
	lines = append(lines, fmt.Sprintf("AllowAgentForwarding %s", boolToYesNo(settings.AllowAgentForwarding)))
	lines = append(lines, fmt.Sprintf("AllowTcpForwarding %s", boolToYesNo(settings.AllowTcpForwarding)))
	lines = append(lines, fmt.Sprintf("ClientAliveInterval %d", settings.ClientAliveInterval))
	lines = append(lines, fmt.Sprintf("ClientAliveCountMax %d", settings.ClientAliveCountMax))
	lines = append(lines, fmt.Sprintf("MaxAuthTries %d", settings.MaxAuthTries))
	lines = append(lines, fmt.Sprintf("MaxSessions %d", settings.MaxSessions))
	lines = append(lines, fmt.Sprintf("IgnoreRhosts %s", boolToYesNo(settings.IgnoreRhosts)))
	lines = append(lines, fmt.Sprintf("HostbasedAuthentication %s", boolToYesNo(settings.HostbasedAuthentication)))
	lines = append(lines, fmt.Sprintf("PermitEmptyPasswords %s", boolToYesNo(settings.PermitEmptyPasswords)))
	lines = append(lines, fmt.Sprintf("ChallengeResponseAuthentication %s", boolToYesNo(settings.ChallengeResponseAuthentication)))
	lines = append(lines, fmt.Sprintf("KerberosAuthentication %s", boolToYesNo(settings.KerberosAuthentication)))
	lines = append(lines, fmt.Sprintf("GSSAPIAuthentication %s", boolToYesNo(settings.GSSAPIAuthentication)))

	// Additional security settings
	lines = append(lines, "")
	lines = append(lines, "# Additional security settings")
	lines = append(lines, "PrintMotd no")
	lines = append(lines, "PrintLastLog yes")
	lines = append(lines, "TCPKeepAlive yes")
	lines = append(lines, "Compression delayed")
	lines = append(lines, "LogLevel INFO")
	lines = append(lines, "SyslogFacility AUTH")
	lines = append(lines, "StrictModes yes")
	lines = append(lines, "MaxStartups 10:30:60")
	lines = append(lines, "LoginGraceTime 60")
	lines = append(lines, "PermitUserEnvironment no")
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

func (sm *securityManager) validateSSHConfig(ctx context.Context) error {
	cmd := tunnel.Command{
		Cmd:     "sshd -t",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("SSH configuration validation failed: %s", result.Output)
	}

	return nil
}

func (sm *securityManager) restartSSHService(ctx context.Context) error {
	// Try different service names
	serviceNames := []string{"ssh", "sshd"}

	for _, serviceName := range serviceNames {
		cmd := tunnel.Command{
			Cmd:     fmt.Sprintf("systemctl restart %s", serviceName),
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 {
			return nil
		}
	}

	return fmt.Errorf("failed to restart SSH service")
}

func (sm *securityManager) applyAdditionalSecurity(ctx context.Context, config tunnel.SecurityConfig) error {
	// Disable unnecessary services
	if err := sm.disableUnnecessaryServices(ctx); err != nil {
		return err
	}

	// Set kernel parameters for security
	if err := sm.setSecurityKernelParameters(ctx); err != nil {
		return err
	}

	// Configure automatic updates (optional)
	if err := sm.configureAutoUpdates(ctx); err != nil {
		// Log but don't fail - this is not critical
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "apply_lockdown",
			Status:      "warning",
			Message:     "Auto-updates configuration failed but continuing",
			ProgressPct: 95,
			Timestamp:   time.Now(),
		})
	}

	return nil
}

func (sm *securityManager) disableUnnecessaryServices(ctx context.Context) error {
	unnecessaryServices := []string{
		"telnet",
		"rsh",
		"rlogin",
		"vsftpd",
		"xinetd",
		"avahi-daemon",
		"cups",
	}

	for _, service := range unnecessaryServices {
		// Check if service exists and is active
		cmd := tunnel.Command{
			Cmd:     fmt.Sprintf("systemctl is-active %s", service),
			Sudo:    true,
			Timeout: 10 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Output) == "active" {
			// Stop and disable the service
			cmd = tunnel.Command{
				Cmd:     fmt.Sprintf("systemctl stop %s && systemctl disable %s", service, service),
				Sudo:    true,
				Timeout: 30 * time.Second,
			}
			sm.executor.RunCommand(ctx, cmd) // Ignore errors
		}
	}

	return nil
}

func (sm *securityManager) setSecurityKernelParameters(ctx context.Context) error {
	kernelParams := []string{
		"# IP spoofing protection",
		"net.ipv4.conf.default.rp_filter = 1",
		"net.ipv4.conf.all.rp_filter = 1",
		"",
		"# IP source routing protection",
		"net.ipv4.conf.default.accept_source_route = 0",
		"net.ipv4.conf.all.accept_source_route = 0",
		"net.ipv6.conf.default.accept_source_route = 0",
		"net.ipv6.conf.all.accept_source_route = 0",
		"",
		"# ICMP redirect protection",
		"net.ipv4.conf.default.accept_redirects = 0",
		"net.ipv4.conf.all.accept_redirects = 0",
		"net.ipv6.conf.default.accept_redirects = 0",
		"net.ipv6.conf.all.accept_redirects = 0",
		"",
		"# Log martian packets",
		"net.ipv4.conf.default.log_martians = 1",
		"net.ipv4.conf.all.log_martians = 1",
		"",
		"# Ignore ICMP pings",
		"net.ipv4.icmp_echo_ignore_all = 1",
		"",
		"# Ignore broadcast pings",
		"net.ipv4.icmp_echo_ignore_broadcasts = 1",
		"",
		"# SYN flood protection",
		"net.ipv4.tcp_syncookies = 1",
		"net.ipv4.tcp_max_syn_backlog = 2048",
		"net.ipv4.tcp_synack_retries = 2",
		"net.ipv4.tcp_syn_retries = 5",
	}

	securityConf := strings.Join(kernelParams, "\n")

	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > /etc/sysctl.d/99-security.conf << 'EOF'\n%s\nEOF", securityConf),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to set kernel security parameters: %s", result.Output)
	}

	// Apply the parameters
	cmd = tunnel.Command{
		Cmd:     "sysctl -p /etc/sysctl.d/99-security.conf",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	sm.executor.RunCommand(ctx, cmd) // Ignore errors - parameters will be applied on reboot

	return nil
}

func (sm *securityManager) configureAutoUpdates(ctx context.Context) error {
	// Try to install and configure unattended-upgrades (Ubuntu/Debian)
	cmd := tunnel.Command{
		Cmd:     "which apt",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return sm.configureAptAutoUpdates(ctx)
	}

	// Try yum-cron (RHEL/CentOS)
	cmd = tunnel.Command{
		Cmd:     "which yum",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		return sm.configureYumAutoUpdates(ctx)
	}

	return fmt.Errorf("no supported package manager for auto-updates")
}

func (sm *securityManager) configureAptAutoUpdates(ctx context.Context) error {
	// Install unattended-upgrades
	cmd := tunnel.Command{
		Cmd:     "apt update && apt install -y unattended-upgrades",
		Sudo:    true,
		Timeout: 5 * time.Minute,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to install unattended-upgrades: %s", result.Output)
	}

	// Configure automatic updates
	cmd = tunnel.Command{
		Cmd:     "dpkg-reconfigure -f noninteractive unattended-upgrades",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	sm.executor.RunCommand(ctx, cmd) // Ignore errors

	return nil
}

func (sm *securityManager) configureYumAutoUpdates(ctx context.Context) error {
	// Install yum-cron
	cmd := tunnel.Command{
		Cmd:     "yum install -y yum-cron",
		Sudo:    true,
		Timeout: 5 * time.Minute,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to install yum-cron: %s", result.Output)
	}

	// Enable and start yum-cron
	cmd = tunnel.Command{
		Cmd:     "systemctl enable yum-cron && systemctl start yum-cron",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	sm.executor.RunCommand(ctx, cmd) // Ignore errors

	return nil
}

// Utility methods

func (sm *securityManager) reportProgress(ctx context.Context, update tunnel.ProgressUpdate) {
	if reporter, ok := tunnel.GetProgressReporter(ctx); ok {
		reporter.Report(update)
	}
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func containsService(services []string, service string) bool {
	for _, s := range services {
		if s == service {
			return true
		}
	}
	return false
}

// defaultSecurityConfig returns default security configuration
func defaultSecurityConfig() tunnel.SecurityConfig {
	return tunnel.SecurityConfig{
		DisableRootLogin:    true,
		DisablePasswordAuth: true,
		FirewallRules: []tunnel.FirewallRule{
			{Port: 22, Protocol: "tcp", Action: "allow", Description: "SSH access"},
			{Port: 80, Protocol: "tcp", Action: "allow", Description: "HTTP access"},
			{Port: 443, Protocol: "tcp", Action: "allow", Description: "HTTPS access"},
		},
		Fail2banConfig: tunnel.Fail2banConfig{
			Enabled:    true,
			MaxRetries: 5,
			BanTime:    1 * time.Hour,
			FindTime:   10 * time.Minute,
			Services:   []string{"ssh", "apache", "nginx"},
		},
		SSHHardeningConfig: tunnel.SSHHardeningConfig{
			PasswordAuthentication:          false,
			PubkeyAuthentication:            true,
			PermitRootLogin:                 false,
			X11Forwarding:                   false,
			AllowAgentForwarding:            false,
			AllowTcpForwarding:              false,
			ClientAliveInterval:             300,
			ClientAliveCountMax:             2,
			MaxAuthTries:                    3,
			MaxSessions:                     10,
			Protocol:                        2,
			IgnoreRhosts:                    true,
			HostbasedAuthentication:         false,
			PermitEmptyPasswords:            false,
			ChallengeResponseAuthentication: false,
			KerberosAuthentication:          false,
			GSSAPIAuthentication:            false,
		},
		AllowedPorts: []int{22, 80, 443},
		AllowedUsers: []string{},
	}
}

// SetConfig updates the security manager configuration
func (sm *securityManager) SetConfig(config tunnel.SecurityConfig) {
	sm.config = config
}

// GetConfig returns the current security manager configuration
func (sm *securityManager) GetConfig() tunnel.SecurityConfig {
	return sm.config
}

// ConfigureAutoUpdates configures automatic security updates
func (sm *securityManager) ConfigureAutoUpdates(ctx context.Context) error {
	span := sm.tracer.TraceSecurityOperation(ctx, "configure_auto_updates", "system")
	defer span.End()

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "configure_auto_updates",
		Status:      "running",
		Message:     "Configuring automatic security updates",
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	if err := sm.configureAutoUpdates(ctx); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSecurityError("configure_auto_updates", "system", err)
	}

	span.Event("auto_updates_configured", map[string]any{
		"configured": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "configure_auto_updates",
		Status:      "success",
		Message:     "Automatic security updates configured successfully",
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// AuditSecurity performs comprehensive security audit and compliance checking
func (sm *securityManager) AuditSecurity(ctx context.Context) (*tunnel.SecurityReport, error) {
	span := sm.tracer.TraceSecurityOperation(ctx, "audit_security", "system")
	defer span.End()

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "running",
		Message:     "Starting security audit",
		ProgressPct: 5,
		Timestamp:   time.Now(),
	})

	report := &tunnel.SecurityReport{
		Timestamp:       time.Now(),
		Overall:         "unknown",
		Checks:          make([]tunnel.SecurityCheck, 0),
		Recommendations: make([]string, 0),
	}

	// Audit SSH configuration
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "running",
		Message:     "Auditing SSH configuration",
		ProgressPct: 20,
		Timestamp:   time.Now(),
	})

	sshCheck := sm.auditSSHConfiguration(ctx)
	report.Checks = append(report.Checks, sshCheck)

	// Audit firewall configuration
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "running",
		Message:     "Auditing firewall configuration",
		ProgressPct: 40,
		Timestamp:   time.Now(),
	})

	firewallCheck := sm.auditFirewallConfiguration(ctx)
	report.Checks = append(report.Checks, firewallCheck)

	// Audit fail2ban configuration
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "running",
		Message:     "Auditing fail2ban configuration",
		ProgressPct: 60,
		Timestamp:   time.Now(),
	})

	fail2banCheck := sm.auditFail2banConfiguration(ctx)
	report.Checks = append(report.Checks, fail2banCheck)

	// Audit system security
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "running",
		Message:     "Auditing system security",
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	systemCheck := sm.auditSystemSecurity(ctx)
	report.Checks = append(report.Checks, systemCheck)

	// Calculate overall score and status
	sm.calculateOverallSecurity(report)

	span.Event("security_audit_completed", map[string]any{
		"overall_status":        report.Overall,
		"checks_count":          len(report.Checks),
		"recommendations_count": len(report.Recommendations),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "audit_security",
		Status:      "success",
		Message:     fmt.Sprintf("Security audit completed - Overall status: %s", report.Overall),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return report, nil
}

func (sm *securityManager) auditSSHConfiguration(ctx context.Context) tunnel.SecurityCheck {
	check := tunnel.SecurityCheck{
		Name:     "SSH Configuration",
		Category: "ssh",
		Status:   "pass",
		Score:    100,
		Issues:   make([]string, 0),
		Details:  make(map[string]any),
	}

	// Check if SSH is running
	cmd := tunnel.Command{
		Cmd:     "systemctl is-active ssh || systemctl is-active sshd",
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil || result.ExitCode != 0 {
		check.Status = "fail"
		check.Score = 0
		check.Issues = append(check.Issues, "SSH service is not running")
		return check
	}

	// Check SSH configuration file
	cmd = tunnel.Command{
		Cmd:     fmt.Sprintf("cat %s", tunnel.SSHDConfigPath),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		check.Status = "warning"
		check.Score = 50
		check.Issues = append(check.Issues, "Could not read SSH configuration file")
		return check
	}

	config := result.Output
	issues := 0

	// Check various security settings
	if !strings.Contains(config, "PasswordAuthentication no") {
		check.Issues = append(check.Issues, "Password authentication is enabled")
		issues++
	}

	if !strings.Contains(config, "PermitRootLogin no") {
		check.Issues = append(check.Issues, "Root login is permitted")
		issues++
	}

	if !strings.Contains(config, "Protocol 2") {
		check.Issues = append(check.Issues, "SSH protocol version 2 is not enforced")
		issues++
	}

	if strings.Contains(config, "X11Forwarding yes") {
		check.Issues = append(check.Issues, "X11 forwarding is enabled")
		issues++
	}

	// Calculate score based on issues
	if issues > 0 {
		check.Status = "warning"
		check.Score = max(0, 100-(issues*25))
	}

	check.Details["issues_found"] = issues
	check.Details["ssh_active"] = true

	return check
}

func (sm *securityManager) auditFirewallConfiguration(ctx context.Context) tunnel.SecurityCheck {
	check := tunnel.SecurityCheck{
		Name:     "Firewall Configuration",
		Category: "firewall",
		Status:   "pass",
		Score:    100,
		Issues:   make([]string, 0),
		Details:  make(map[string]any),
	}

	// Check if any firewall is active
	firewallActive := false

	// Check UFW
	cmd := tunnel.Command{
		Cmd:     "ufw status",
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 && strings.Contains(result.Output, "Status: active") {
		firewallActive = true
		check.Details["firewall_type"] = "ufw"
	}

	// Check iptables if UFW is not active
	if !firewallActive {
		cmd = tunnel.Command{
			Cmd:     "iptables -L",
			Sudo:    true,
			Timeout: 10 * time.Second,
		}

		result, err = sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 && !strings.Contains(result.Output, "ACCEPT     all  --  anywhere             anywhere") {
			firewallActive = true
			check.Details["firewall_type"] = "iptables"
		}
	}

	// Check firewalld
	if !firewallActive {
		cmd = tunnel.Command{
			Cmd:     "firewall-cmd --state",
			Sudo:    true,
			Timeout: 10 * time.Second,
		}

		result, err = sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 {
			firewallActive = true
			check.Details["firewall_type"] = "firewalld"
		}
	}

	if !firewallActive {
		check.Status = "fail"
		check.Score = 0
		check.Issues = append(check.Issues, "No active firewall detected")
	}

	check.Details["firewall_active"] = firewallActive

	return check
}

func (sm *securityManager) auditFail2banConfiguration(ctx context.Context) tunnel.SecurityCheck {
	check := tunnel.SecurityCheck{
		Name:     "Fail2ban Configuration",
		Category: "intrusion_prevention",
		Status:   "pass",
		Score:    100,
		Issues:   make([]string, 0),
		Details:  make(map[string]any),
	}

	// Check if fail2ban is installed
	cmd := tunnel.Command{
		Cmd:     "which fail2ban-client",
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil || result.ExitCode != 0 {
		check.Status = "warning"
		check.Score = 50
		check.Issues = append(check.Issues, "Fail2ban is not installed")
		check.Details["fail2ban_installed"] = false
		return check
	}

	check.Details["fail2ban_installed"] = true

	// Check if fail2ban is running
	cmd = tunnel.Command{
		Cmd:     "systemctl is-active fail2ban",
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil || result.ExitCode != 0 {
		check.Status = "warning"
		check.Score = 75
		check.Issues = append(check.Issues, "Fail2ban service is not running")
		check.Details["fail2ban_active"] = false
		return check
	}

	check.Details["fail2ban_active"] = true

	// Check fail2ban status
	cmd = tunnel.Command{
		Cmd:     "fail2ban-client status",
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 {
		check.Details["fail2ban_status"] = result.Output
	}

	return check
}

func (sm *securityManager) auditSystemSecurity(ctx context.Context) tunnel.SecurityCheck {
	check := tunnel.SecurityCheck{
		Name:     "System Security",
		Category: "system",
		Status:   "pass",
		Score:    100,
		Issues:   make([]string, 0),
		Details:  make(map[string]any),
	}

	issues := 0

	// Check for unnecessary services
	unnecessaryServices := []string{"telnet", "rsh", "rlogin", "vsftpd"}
	for _, service := range unnecessaryServices {
		cmd := tunnel.Command{
			Cmd:     fmt.Sprintf("systemctl is-active %s", service),
			Sudo:    true,
			Timeout: 10 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Output) == "active" {
			check.Issues = append(check.Issues, fmt.Sprintf("Unnecessary service %s is running", service))
			issues++
		}
	}

	// Check kernel security parameters
	cmd := tunnel.Command{
		Cmd:     "sysctl net.ipv4.ip_forward",
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err == nil && result.ExitCode == 0 && strings.Contains(result.Output, " = 1") {
		check.Issues = append(check.Issues, "IP forwarding is enabled")
		issues++
	}

	// Calculate score based on issues
	if issues > 0 {
		check.Status = "warning"
		check.Score = max(0, 100-(issues*20))
	}

	check.Details["issues_found"] = issues

	return check
}

func (sm *securityManager) calculateOverallSecurity(report *tunnel.SecurityReport) {
	if len(report.Checks) == 0 {
		report.Overall = "unknown"
		return
	}

	totalScore := 0
	criticalIssues := 0

	for _, check := range report.Checks {
		totalScore += check.Score
		if check.Status == "fail" {
			criticalIssues++
		}
	}

	averageScore := totalScore / len(report.Checks)

	// Generate recommendations based on findings
	if criticalIssues > 0 {
		report.Overall = "critical"
		report.Recommendations = append(report.Recommendations, "Address critical security issues immediately")
	} else if averageScore < 70 {
		report.Overall = "poor"
		report.Recommendations = append(report.Recommendations, "Multiple security improvements needed")
	} else if averageScore < 85 {
		report.Overall = "fair"
		report.Recommendations = append(report.Recommendations, "Some security improvements recommended")
	} else if averageScore < 95 {
		report.Overall = "good"
		report.Recommendations = append(report.Recommendations, "Minor security improvements available")
	} else {
		report.Overall = "excellent"
		report.Recommendations = append(report.Recommendations, "Security configuration is excellent")
	}

	// Add specific recommendations based on check results
	for _, check := range report.Checks {
		if len(check.Issues) > 0 {
			for _, issue := range check.Issues {
				report.Recommendations = append(report.Recommendations, fmt.Sprintf("Fix %s: %s", check.Category, issue))
			}
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
