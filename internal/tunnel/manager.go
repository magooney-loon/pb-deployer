package tunnel

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Manager handles all server operations through a single interface
type Manager struct {
	client SSHClient
	tracer Tracer
}

// NewManager creates a new manager instance
func NewManager(client SSHClient) *Manager {
	if client == nil {
		panic("client cannot be nil")
	}
	return &Manager{
		client: client,
		tracer: &NoOpTracer{},
	}
}

// SetTracer sets the tracer for the manager
func (m *Manager) SetTracer(tracer Tracer) {
	m.tracer = tracer
	if m.client != nil {
		m.client.SetTracer(tracer)
	}
}

// User Management

// CreateUser creates a new system user
func (m *Manager) CreateUser(username string, opts ...UserOption) error {
	// Apply options
	cfg := &userConfig{
		shell: "/bin/bash",
		home:  fmt.Sprintf("/home/%s", username),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Check if user exists
	result, err := m.client.Execute(fmt.Sprintf("id %s", username), WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		// User already exists
		return nil
	}

	// Build useradd command
	cmd := "useradd"
	if cfg.systemUser {
		cmd += " -r"
	}
	if cfg.home != "" {
		cmd += fmt.Sprintf(" -d '%s' -m", cfg.home)
	}
	if cfg.shell != "" {
		cmd += fmt.Sprintf(" -s '%s'", cfg.shell)
	}
	cmd += fmt.Sprintf(" '%s'", username)

	// Create user
	result, err = m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to create user: %s", result.Stderr),
		}
	}

	// Add to groups
	if len(cfg.groups) > 0 {
		groupList := strings.Join(cfg.groups, ",")
		cmd = fmt.Sprintf("usermod -aG '%s' '%s'", groupList, username)
		result, err = m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("failed to add user to groups: %s", result.Stderr),
			}
		}
	}

	// Setup sudo access
	if cfg.sudoAccess {
		sudoLine := fmt.Sprintf("%s ALL=(ALL:ALL) NOPASSWD:ALL", username)
		cmd = fmt.Sprintf("echo '%s' > /etc/sudoers.d/%s", sudoLine, username)
		result, err = m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("failed to setup sudo access: %s", result.Stderr),
			}
		}

		// Set correct permissions on sudoers file
		cmd = fmt.Sprintf("chmod 0440 /etc/sudoers.d/%s", username)
		m.client.ExecuteSudo(cmd)
	}

	return nil
}

// DeleteUser deletes a system user
func (m *Manager) DeleteUser(username string) error {
	cmd := fmt.Sprintf("userdel -r '%s'", username)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 && !strings.Contains(result.Stderr, "does not exist") {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to delete user: %s", result.Stderr),
		}
	}
	return nil
}

// SetupSSHKeys configures SSH keys for a user
func (m *Manager) SetupSSHKeys(username string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	// Get user home directory
	result, err := m.client.Execute(fmt.Sprintf("getent passwd %s | cut -d: -f6", username))
	if err != nil {
		return err
	}
	homeDir := strings.TrimSpace(result.Stdout)
	if homeDir == "" {
		homeDir = fmt.Sprintf("/home/%s", username)
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	authKeysFile := filepath.Join(sshDir, "authorized_keys")

	// Create .ssh directory
	cmd := fmt.Sprintf("mkdir -p '%s' && chmod 700 '%s' && chown '%s:%s' '%s'",
		sshDir, sshDir, username, username, sshDir)
	result, err = m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}

	// Write authorized_keys file
	keysContent := strings.Join(keys, "\n")
	cmd = fmt.Sprintf("echo '%s' > '%s' && chmod 600 '%s' && chown '%s:%s' '%s'",
		keysContent, authKeysFile, authKeysFile, username, username, authKeysFile)
	result, err = m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to setup SSH keys: %s", result.Stderr),
		}
	}

	return nil
}

// Service Management

// ServiceStart starts a system service
func (m *Manager) ServiceStart(name string) error {
	cmd := fmt.Sprintf("systemctl start %s", name)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to start service: %s", result.Stderr),
		}
	}
	return nil
}

// ServiceStop stops a system service
func (m *Manager) ServiceStop(name string) error {
	cmd := fmt.Sprintf("systemctl stop %s", name)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to stop service: %s", result.Stderr),
		}
	}
	return nil
}

// ServiceRestart restarts a system service
func (m *Manager) ServiceRestart(name string) error {
	cmd := fmt.Sprintf("systemctl restart %s", name)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to restart service: %s", result.Stderr),
		}
	}
	return nil
}

// ServiceStatus gets the status of a system service
func (m *Manager) ServiceStatus(name string) (*ServiceStatus, error) {
	cmd := fmt.Sprintf("systemctl show %s --no-page", name)
	result, err := m.client.Execute(cmd)
	if err != nil {
		return nil, err
	}

	status := &ServiceStatus{
		Name: name,
	}

	// Parse systemctl output
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "ActiveState":
			status.Active = (value == "active")
			status.Running = (value == "active")
		case "UnitFileState":
			status.Enabled = (value == "enabled")
		case "Description":
			status.Description = value
		case "MainPID":
			fmt.Sscanf(value, "%d", &status.MainPID)
		}
	}

	return status, nil
}

// ServiceLogs retrieves service logs
func (m *Manager) ServiceLogs(name string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	cmd := fmt.Sprintf("journalctl -u %s -n %d --no-pager", name, lines)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

// ServiceEnable enables a service to start on boot
func (m *Manager) ServiceEnable(name string) error {
	cmd := fmt.Sprintf("systemctl enable %s", name)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to enable service: %s", result.Stderr),
		}
	}
	return nil
}

// ServiceDisable disables a service from starting on boot
func (m *Manager) ServiceDisable(name string) error {
	cmd := fmt.Sprintf("systemctl disable %s", name)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to disable service: %s", result.Stderr),
		}
	}
	return nil
}

// Security Management

// SetupFirewall configures firewall rules
func (m *Manager) SetupFirewall(rules []FirewallRule) error {
	// Detect firewall system
	var firewallCmd string

	// Check for ufw
	result, err := m.client.Execute("which ufw", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		firewallCmd = "ufw"
	} else {
		// Check for firewalld
		result, err = m.client.Execute("which firewall-cmd", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			firewallCmd = "firewalld"
		} else {
			// Default to iptables
			firewallCmd = "iptables"
		}
	}

	switch firewallCmd {
	case "ufw":
		return m.setupUFW(rules)
	case "firewalld":
		return m.setupFirewalld(rules)
	default:
		return m.setupIPTables(rules)
	}
}

func (m *Manager) setupUFW(rules []FirewallRule) error {
	// Reset and configure UFW
	cmds := []string{
		"ufw --force reset",
		"ufw default deny incoming",
		"ufw default allow outgoing",
	}

	for _, cmd := range cmds {
		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("UFW setup failed: %s", result.Stderr),
			}
		}
	}

	// Add rules
	for _, rule := range rules {
		var cmd string
		if rule.Source != "" {
			cmd = fmt.Sprintf("ufw %s from %s to any port %d proto %s",
				rule.Action, rule.Source, rule.Port, rule.Protocol)
		} else {
			cmd = fmt.Sprintf("ufw %s %d/%s", rule.Action, rule.Port, rule.Protocol)
		}

		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Failed to add UFW rule: %s", result.Stderr),
			}
		}
	}

	// Enable UFW
	result, err := m.client.ExecuteSudo("ufw --force enable")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to enable UFW: %s", result.Stderr),
		}
	}

	return nil
}

func (m *Manager) setupFirewalld(rules []FirewallRule) error {
	// Start firewalld
	m.ServiceStart("firewalld")

	// Add rules
	for _, rule := range rules {
		var cmd string
		if rule.Action == "allow" {
			if rule.Source != "" {
				cmd = fmt.Sprintf("firewall-cmd --permanent --add-rich-rule='rule family=\"ipv4\" source address=\"%s\" port protocol=\"%s\" port=\"%d\" accept'",
					rule.Source, rule.Protocol, rule.Port)
			} else {
				cmd = fmt.Sprintf("firewall-cmd --permanent --add-port=%d/%s", rule.Port, rule.Protocol)
			}

			result, err := m.client.ExecuteSudo(cmd)
			if err != nil {
				return err
			}
			if result.ExitCode != 0 {
				return &Error{
					Type:    ErrorExecution,
					Message: fmt.Sprintf("Failed to add firewalld rule: %s", result.Stderr),
				}
			}
		}
	}

	// Reload firewalld
	result, err := m.client.ExecuteSudo("firewall-cmd --reload")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to reload firewalld: %s", result.Stderr),
		}
	}

	return nil
}

func (m *Manager) setupIPTables(rules []FirewallRule) error {
	// Basic iptables setup
	cmds := []string{
		"iptables -F",
		"iptables -P INPUT DROP",
		"iptables -P FORWARD DROP",
		"iptables -P OUTPUT ACCEPT",
		"iptables -A INPUT -i lo -j ACCEPT",
		"iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT",
	}

	for _, cmd := range cmds {
		m.client.ExecuteSudo(cmd)
	}

	// Add rules
	for _, rule := range rules {
		action := "ACCEPT"
		if rule.Action == "deny" {
			action = "DROP"
		}

		var cmd string
		if rule.Source != "" {
			cmd = fmt.Sprintf("iptables -A INPUT -p %s --dport %d -s %s -j %s",
				rule.Protocol, rule.Port, rule.Source, action)
		} else {
			cmd = fmt.Sprintf("iptables -A INPUT -p %s --dport %d -j %s",
				rule.Protocol, rule.Port, action)
		}

		m.client.ExecuteSudo(cmd)
	}

	// Save iptables rules
	m.client.ExecuteSudo("iptables-save > /etc/iptables/rules.v4")

	return nil
}

// HardenSSH applies SSH hardening configuration
func (m *Manager) HardenSSH(config SSHConfig) error {
	// Backup current SSH config
	m.client.ExecuteSudo("cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak")

	// Build SSH configuration
	var configLines []string
	configLines = append(configLines, "# SSH Hardening Configuration")
	configLines = append(configLines, fmt.Sprintf("PasswordAuthentication %s", boolToYesNo(config.PasswordAuth)))
	configLines = append(configLines, fmt.Sprintf("PermitRootLogin %s", boolToYesNo(config.RootLogin)))
	configLines = append(configLines, fmt.Sprintf("PubkeyAuthentication %s", boolToYesNo(config.PubkeyAuth)))
	configLines = append(configLines, fmt.Sprintf("MaxAuthTries %d", config.MaxAuthTries))
	configLines = append(configLines, fmt.Sprintf("ClientAliveInterval %d", config.ClientAliveInterval))
	configLines = append(configLines, fmt.Sprintf("ClientAliveCountMax %d", config.ClientAliveCountMax))

	if len(config.AllowUsers) > 0 {
		configLines = append(configLines, fmt.Sprintf("AllowUsers %s", strings.Join(config.AllowUsers, " ")))
	}
	if len(config.AllowGroups) > 0 {
		configLines = append(configLines, fmt.Sprintf("AllowGroups %s", strings.Join(config.AllowGroups, " ")))
	}

	// Write new SSH config
	configContent := strings.Join(configLines, "\n")
	cmd := fmt.Sprintf("echo '%s' > /etc/ssh/sshd_config.d/99-hardening.conf", configContent)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to write SSH config: %s", result.Stderr),
		}
	}

	// Test SSH config
	result, err = m.client.ExecuteSudo("sshd -t")
	if err != nil || result.ExitCode != 0 {
		// Restore backup
		m.client.ExecuteSudo("rm /etc/ssh/sshd_config.d/99-hardening.conf")
		return &Error{
			Type:    ErrorExecution,
			Message: "SSH configuration test failed",
		}
	}

	// Restart SSH service
	m.ServiceRestart("sshd")

	return nil
}

// SetupFail2ban installs and configures fail2ban
func (m *Manager) SetupFail2ban() error {
	// Install fail2ban
	err := m.InstallPackages("fail2ban")
	if err != nil {
		return err
	}

	// Basic fail2ban configuration
	jailConfig := `[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = ssh
logpath = /var/log/auth.log
backend = systemd`

	cmd := fmt.Sprintf("echo '%s' > /etc/fail2ban/jail.local", jailConfig)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to configure fail2ban: %s", result.Stderr),
		}
	}

	// Enable and start fail2ban
	m.ServiceEnable("fail2ban")
	m.ServiceRestart("fail2ban")

	return nil
}

// Deployment

// Deploy deploys an application
func (m *Manager) Deploy(app AppConfig) error {
	// Backup if requested
	if app.Backup && app.Target != "" {
		backupPath := fmt.Sprintf("%s.bak.%d", app.Target, time.Now().Unix())
		cmd := fmt.Sprintf("cp -r '%s' '%s'", app.Target, backupPath)
		m.client.ExecuteSudo(cmd)
	}

	// Run pre-deploy commands
	for _, cmd := range app.PreDeploy {
		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Pre-deploy command failed: %s", result.Stderr),
			}
		}
	}

	// Deploy application
	if strings.HasPrefix(app.Source, "http://") || strings.HasPrefix(app.Source, "https://") {
		// Download from URL
		cmd := fmt.Sprintf("wget -O /tmp/deploy.tar.gz '%s'", app.Source)
		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Failed to download application: %s", result.Stderr),
			}
		}

		// Extract
		cmd = fmt.Sprintf("mkdir -p '%s' && tar -xzf /tmp/deploy.tar.gz -C '%s'", app.Target, app.Target)
		result, err = m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Failed to extract application: %s", result.Stderr),
			}
		}
	} else {
		// Upload from local file
		err := m.client.Upload(app.Source, "/tmp/deploy.tar.gz")
		if err != nil {
			return err
		}

		// Extract
		cmd := fmt.Sprintf("mkdir -p '%s' && tar -xzf /tmp/deploy.tar.gz -C '%s'", app.Target, app.Target)
		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Failed to extract application: %s", result.Stderr),
			}
		}
	}

	// Run post-deploy commands
	for _, cmd := range app.PostDeploy {
		result, err := m.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("Post-deploy command failed: %s", result.Stderr),
			}
		}
	}

	// Restart service if specified
	if app.Service != "" {
		m.ServiceRestart(app.Service)
	}

	// Health check
	if app.HealthCheck != "" {
		time.Sleep(5 * time.Second)

		if strings.HasPrefix(app.HealthCheck, "http://") || strings.HasPrefix(app.HealthCheck, "https://") {
			cmd := fmt.Sprintf("curl -f -s '%s'", app.HealthCheck)
			result, err := m.client.Execute(cmd, WithTimeout(30*time.Second))
			if err != nil || result.ExitCode != 0 {
				return &Error{
					Type:    ErrorExecution,
					Message: "Health check failed",
				}
			}
		} else {
			// Execute as command
			result, err := m.client.Execute(app.HealthCheck, WithTimeout(30*time.Second))
			if err != nil || result.ExitCode != 0 {
				return &Error{
					Type:    ErrorExecution,
					Message: "Health check failed",
				}
			}
		}
	}

	return nil
}

// Rollback rolls back a deployment
func (m *Manager) Rollback(app string, version string) error {
	backupPath := fmt.Sprintf("/opt/%s.%s", app, version)
	targetPath := fmt.Sprintf("/opt/%s", app)

	// Check if backup exists
	result, err := m.client.Execute(fmt.Sprintf("test -d '%s'", backupPath))
	if err != nil || result.ExitCode != 0 {
		return &Error{
			Type:    ErrorNotFound,
			Message: fmt.Sprintf("Backup version %s not found", version),
		}
	}

	// Stop service
	m.ServiceStop(app)

	// Restore backup
	cmd := fmt.Sprintf("rm -rf '%s' && cp -r '%s' '%s'", targetPath, backupPath, targetPath)
	result, err = m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to restore backup: %s", result.Stderr),
		}
	}

	// Start service
	m.ServiceStart(app)

	return nil
}

// Package Management

// InstallPackages installs system packages
func (m *Manager) InstallPackages(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Detect package manager
	var installCmd string

	// Check for apt
	result, err := m.client.Execute("which apt", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		installCmd = fmt.Sprintf("apt update && apt install -y %s", strings.Join(packages, " "))
	} else {
		// Check for yum
		result, err = m.client.Execute("which yum", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			installCmd = fmt.Sprintf("yum install -y %s", strings.Join(packages, " "))
		} else {
			// Check for dnf
			result, err = m.client.Execute("which dnf", WithTimeout(5*time.Second))
			if err == nil && result.ExitCode == 0 {
				installCmd = fmt.Sprintf("dnf install -y %s", strings.Join(packages, " "))
			} else {
				return &Error{
					Type:    ErrorNotFound,
					Message: "No supported package manager found",
				}
			}
		}
	}

	// Install packages
	result, err = m.client.ExecuteSudo(installCmd, WithTimeout(5*time.Minute))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to install packages: %s", result.Stderr),
		}
	}

	return nil
}

// UpdateSystem updates all system packages
func (m *Manager) UpdateSystem() error {
	// Detect package manager and run update
	result, err := m.client.Execute("which apt", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		// Debian/Ubuntu
		cmd := "apt update && apt upgrade -y && apt autoremove -y"
		result, err = m.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
	} else {
		result, err = m.client.Execute("which yum", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			// RHEL/CentOS
			cmd := "yum update -y"
			result, err = m.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
		} else {
			result, err = m.client.Execute("which dnf", WithTimeout(5*time.Second))
			if err == nil && result.ExitCode == 0 {
				// Fedora
				cmd := "dnf update -y"
				result, err = m.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
			} else {
				return &Error{
					Type:    ErrorNotFound,
					Message: "No supported package manager found",
				}
			}
		}
	}

	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("System update failed: %s", result.Stderr),
		}
	}

	return nil
}

// Advanced Operations

// Batch executes multiple commands in sequence
func (m *Manager) Batch(commands ...Command) (*BatchResult, error) {
	result := &BatchResult{
		Results: make([]Result, 0, len(commands)),
		Errors:  make([]error, 0),
	}

	for _, cmd := range commands {
		res, err := m.client.Execute(cmd.Cmd, cmd.Opts...)
		if res != nil {
			result.Results = append(result.Results, *res)
		}
		if err != nil {
			result.Errors = append(result.Errors, err)
			// Continue executing remaining commands
		}
	}

	if len(result.Errors) > 0 {
		return result, result.Errors[0]
	}

	return result, nil
}

// Transaction executes commands in a transaction with rollback support
func (m *Manager) Transaction(fn func(*Transaction) error) error {
	tx := &Transaction{
		client:   m.client,
		rollback: make([]func() error, 0),
	}

	err := fn(tx)
	if err != nil {
		// Execute rollback functions in reverse order
		for i := len(tx.rollback) - 1; i >= 0; i-- {
			tx.rollback[i]()
		}
		return err
	}

	return nil
}

// Helper functions

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Transaction methods

// Execute runs a command in the transaction
func (tx *Transaction) Execute(cmd string, opts ...ExecOption) error {
	result, err := tx.client.Execute(cmd, opts...)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Command failed: %s", result.Stderr),
		}
	}
	// No rollback for execute commands
	return nil
}

// Upload uploads a file in the transaction
func (tx *Transaction) Upload(local, remote string, opts ...FileOption) error {
	err := tx.client.Upload(local, remote, opts...)
	if err != nil {
		return err
	}

	// Add rollback function
	tx.rollback = append(tx.rollback, func() error {
		cmd := fmt.Sprintf("rm -f '%s'", remote)
		tx.client.ExecuteSudo(cmd)
		return nil
	})

	return nil
}

// CreateFile creates a file in the transaction
func (tx *Transaction) CreateFile(path, content string) error {
	cmd := fmt.Sprintf("echo '%s' > '%s'", content, path)
	result, err := tx.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to create file: %s", result.Stderr),
		}
	}

	// Add rollback function
	tx.rollback = append(tx.rollback, func() error {
		cmd := fmt.Sprintf("rm -f '%s'", path)
		tx.client.ExecuteSudo(cmd)
		return nil
	})

	return nil
}

// SystemInfo returns system information
func (m *Manager) SystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get OS information
	result, err := m.client.Execute("lsb_release -a 2>/dev/null || cat /etc/os-release")
	if err == nil {
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Description:") || strings.Contains(line, "PRETTY_NAME=") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.OS = strings.TrimSpace(parts[1])
				} else if strings.Contains(line, "=") {
					parts = strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						info.OS = strings.Trim(parts[1], "\"")
					}
				}
			}
		}
	}

	// Get kernel version
	result, err = m.client.Execute("uname -r")
	if err == nil {
		info.Kernel = strings.TrimSpace(result.Stdout)
	}

	// Get architecture
	result, err = m.client.Execute("uname -m")
	if err == nil {
		info.Architecture = strings.TrimSpace(result.Stdout)
	}

	// Get hostname
	result, err = m.client.Execute("hostname")
	if err == nil {
		info.Hostname = strings.TrimSpace(result.Stdout)
	}

	// Get CPU count
	result, err = m.client.Execute("nproc")
	if err == nil {
		fmt.Sscanf(result.Stdout, "%d", &info.CPUCount)
	}

	// Get memory
	result, err = m.client.Execute("free -m | grep '^Mem:' | awk '{print $2}'")
	if err == nil {
		fmt.Sscanf(result.Stdout, "%d", &info.MemoryMB)
	}

	// Get disk space
	result, err = m.client.Execute("df -BG / | tail -1 | awk '{print $2}'")
	if err == nil {
		var diskGB string
		fmt.Sscanf(result.Stdout, "%sG", &diskGB)
		fmt.Sscanf(diskGB, "%d", &info.DiskGB)
	}

	// Get uptime
	result, err = m.client.Execute("cat /proc/uptime | awk '{print $1}'")
	if err == nil {
		var seconds float64
		fmt.Sscanf(result.Stdout, "%f", &seconds)
		info.Uptime = time.Duration(seconds) * time.Second
	}

	return info, nil
}

// CreateDirectory creates a directory with specified permissions
func (m *Manager) CreateDirectory(dir Directory) error {
	// Create directory
	cmd := fmt.Sprintf("mkdir -p '%s'", dir.Path)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to create directory: %s", result.Stderr),
		}
	}

	// Set permissions if specified
	if dir.Permissions != "" {
		cmd = fmt.Sprintf("chmod %s '%s'", dir.Permissions, dir.Path)
		m.client.ExecuteSudo(cmd)
	}

	// Set ownership if specified
	if dir.Owner != "" && dir.Group != "" {
		cmd = fmt.Sprintf("chown %s:%s '%s'", dir.Owner, dir.Group, dir.Path)
		m.client.ExecuteSudo(cmd)
	}

	return nil
}

// ApplyTemplate applies a server configuration template
func (m *Manager) ApplyTemplate(template Template) error {
	return template.Apply(m)
}

// Template implementations

// Apply implements Template for TemplateWebServer
func (t TemplateWebServer) Apply(mgr *Manager) error {
	// Install web server packages
	packages := []string{"nginx"}

	if t.PHP {
		packages = append(packages, "php-fpm", "php-mysql", "php-gd", "php-xml", "php-mbstring")
	}

	if t.Database == "mysql" {
		packages = append(packages, "mysql-server")
	} else if t.Database == "postgres" {
		packages = append(packages, "postgresql", "postgresql-contrib")
	}

	if t.SSL {
		packages = append(packages, "certbot", "python3-certbot-nginx")
	}

	err := mgr.InstallPackages(packages...)
	if err != nil {
		return err
	}

	// Configure nginx
	nginxConfig := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root /var/www/%s;
    index index.html index.php;

    location / {
        try_files $uri $uri/ =404;
    }`, t.Domain, t.Domain)

	if t.PHP {
		nginxConfig += `
    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/var/run/php/php-fpm.sock;
    }`
	}

	nginxConfig += `
}`

	// Write nginx config
	configPath := fmt.Sprintf("/etc/nginx/sites-available/%s", t.Domain)
	cmd := fmt.Sprintf("echo '%s' > '%s'", nginxConfig, configPath)
	mgr.client.ExecuteSudo(cmd)

	// Enable site
	cmd = fmt.Sprintf("ln -sf '%s' '/etc/nginx/sites-enabled/%s'", configPath, t.Domain)
	mgr.client.ExecuteSudo(cmd)

	// Create web root
	mgr.CreateDirectory(Directory{
		Path:        fmt.Sprintf("/var/www/%s", t.Domain),
		Permissions: "755",
		Owner:       "www-data",
		Group:       "www-data",
	})

	// Restart nginx
	mgr.ServiceRestart("nginx")

	// Setup SSL if requested
	if t.SSL {
		cmd = fmt.Sprintf("certbot --nginx -d %s --non-interactive --agree-tos --email admin@%s", t.Domain, t.Domain)
		mgr.client.ExecuteSudo(cmd)
	}

	// Setup firewall
	if t.Firewall {
		rules := []FirewallRule{
			{Port: 22, Protocol: "tcp", Action: "allow", Description: "SSH"},
			{Port: 80, Protocol: "tcp", Action: "allow", Description: "HTTP"},
			{Port: 443, Protocol: "tcp", Action: "allow", Description: "HTTPS"},
		}
		mgr.SetupFirewall(rules)
	}

	// Start services
	mgr.ServiceEnable("nginx")
	if t.PHP {
		mgr.ServiceEnable("php-fpm")
		mgr.ServiceRestart("php-fpm")
	}
	if t.Database == "mysql" {
		mgr.ServiceEnable("mysql")
		mgr.ServiceRestart("mysql")
	} else if t.Database == "postgres" {
		mgr.ServiceEnable("postgresql")
		mgr.ServiceRestart("postgresql")
	}

	return nil
}

// Apply implements Template for TemplateDocker
func (t TemplateDocker) Apply(mgr *Manager) error {
	// Install Docker
	installScript := `
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm get-docker.sh
`
	result, err := mgr.client.ExecuteSudo(installScript, WithTimeout(10*time.Minute))
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("Failed to install Docker: %s", result.Stderr),
		}
	}

	// Install Docker Compose if requested
	if t.ComposeVersion {
		cmd := `curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose && chmod +x /usr/local/bin/docker-compose`
		mgr.client.ExecuteSudo(cmd)
	}

	// Initialize Docker Swarm if requested
	if t.Swarm {
		cmd := "docker swarm init"
		mgr.client.ExecuteSudo(cmd)
	}

	// Login to registry if provided
	if t.Registry != "" {
		cmd := fmt.Sprintf("docker login %s", t.Registry)
		mgr.client.Execute(cmd)
	}

	// Enable and start Docker
	mgr.ServiceEnable("docker")
	mgr.ServiceRestart("docker")

	return nil
}

// CheckPort checks if a port is open
func (m *Manager) CheckPort(port int) (bool, error) {
	cmd := fmt.Sprintf("ss -tuln | grep ':%d '", port)
	result, err := m.client.Execute(cmd, WithTimeout(5*time.Second))
	if err != nil {
		return false, err
	}
	return result.ExitCode == 0, nil
}

// GetInstalledPackages returns list of installed packages
func (m *Manager) GetInstalledPackages() ([]Package, error) {
	packages := make([]Package, 0)

	// Try dpkg first (Debian/Ubuntu)
	result, err := m.client.Execute("dpkg -l | grep '^ii'", WithTimeout(30*time.Second))
	if err == nil && result.ExitCode == 0 {
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				packages = append(packages, Package{
					Name:    fields[1],
					Version: fields[2],
					Status:  "installed",
				})
			}
		}
		return packages, nil
	}

	// Try rpm (RHEL/CentOS)
	result, err = m.client.Execute("rpm -qa --qf '%{NAME}|%{VERSION}-%{RELEASE}\n'", WithTimeout(30*time.Second))
	if err == nil && result.ExitCode == 0 {
		lines := strings.Split(result.Stdout, "\n")
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				packages = append(packages, Package{
					Name:    parts[0],
					Version: parts[1],
					Status:  "installed",
				})
			}
		}
		return packages, nil
	}

	return packages, &Error{
		Type:    ErrorNotFound,
		Message: "Could not determine package manager",
	}
}

// RunScript uploads and executes a script
func (m *Manager) RunScript(localScript string, args ...string) (*Result, error) {
	// Generate temporary script name
	remotePath := fmt.Sprintf("/tmp/script_%d.sh", time.Now().Unix())

	// Upload script
	err := m.client.Upload(localScript, remotePath)
	if err != nil {
		return nil, err
	}

	// Make executable
	m.client.ExecuteSudo(fmt.Sprintf("chmod +x '%s'", remotePath))

	// Execute script
	cmd := remotePath
	if len(args) > 0 {
		cmd = fmt.Sprintf("%s %s", remotePath, strings.Join(args, " "))
	}

	result, err := m.client.ExecuteSudo(cmd, WithTimeout(10*time.Minute))

	// Clean up
	m.client.ExecuteSudo(fmt.Sprintf("rm -f '%s'", remotePath))

	return result, err
}
