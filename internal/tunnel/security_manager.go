package tunnel

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"pb-deployer/internal/logger"
)

type SecurityManager struct {
	manager *Manager
	logger  *logger.Logger
	cleanup []func()
	mu      sync.Mutex
	closed  bool
}

func NewSecurityManager(manager *Manager) *SecurityManager {
	return &SecurityManager{
		manager: manager,
		logger:  logger.GetTunnelLogger(),
	}
}

func (s *SecurityManager) SecureServer(config SecurityConfig) error {
	s.logger.SystemOperation("Starting server security hardening")

	if len(config.FirewallRules) > 0 {
		err := s.SetupFirewall(config.FirewallRules)
		if err != nil {
			return fmt.Errorf("failed to setup firewall: %w", err)
		}
	}

	if config.HardenSSH {
		err := s.HardenSSH(config.SSHConfig)
		if err != nil {
			return fmt.Errorf("failed to harden SSH: %w", err)
		}
	}

	if config.EnableFail2ban {
		err := s.SetupFail2ban()
		if err != nil {
			return fmt.Errorf("failed to setup fail2ban: %w", err)
		}
	}

	s.logger.Success("Server security hardening completed")
	return nil
}

func (s *SecurityManager) SetupFirewall(rules []FirewallRule) error {
	s.logger.SystemOperation(fmt.Sprintf("Setting up firewall with %d rules", len(rules)))
	var firewallCmd string

	result, err := s.manager.client.Execute("which ufw", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		firewallCmd = "ufw"
	} else {
		result, err = s.manager.client.Execute("which firewall-cmd", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			firewallCmd = "firewalld"
		} else {
			firewallCmd = "iptables"
		}
	}

	switch firewallCmd {
	case "ufw":
		return s.setupUFW(rules)
	case "firewalld":
		return s.setupFirewalld(rules)
	default:
		return s.setupIPTables(rules)
	}
}

func (s *SecurityManager) setupUFW(rules []FirewallRule) error {
	s.logger.SystemOperation("Configuring UFW firewall")
	s.manager.InstallPackages("ufw")

	cmds := []string{
		"ufw --force reset",
		"ufw default deny incoming",
		"ufw default allow outgoing",
	}

	for _, cmd := range cmds {
		result, err := s.manager.client.ExecuteSudo(cmd)
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

	for _, rule := range rules {
		var cmd string
		if rule.Source != "" {
			cmd = fmt.Sprintf("ufw %s from %s to any port %d proto %s",
				rule.Action, rule.Source, rule.Port, rule.Protocol)
		} else {
			cmd = fmt.Sprintf("ufw %s %d/%s", rule.Action, rule.Port, rule.Protocol)
		}

		result, err := s.manager.client.ExecuteSudo(cmd)
		if err != nil {
			return err
		}
		if result.ExitCode != 0 {
			return &Error{
				Type:    ErrorExecution,
				Message: fmt.Sprintf("failed to add UFW rule: %s", result.Stderr),
			}
		}
	}

	result, err := s.manager.client.ExecuteSudo("ufw --force enable")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to enable UFW: %s", result.Stderr),
		}
	}

	return nil
}

func (s *SecurityManager) setupFirewalld(rules []FirewallRule) error {
	s.logger.SystemOperation("Configuring firewalld")
	s.manager.ServiceStart("firewalld")

	for _, rule := range rules {
		var cmd string
		if rule.Action == "allow" {
			if rule.Source != "" {
				cmd = fmt.Sprintf("firewall-cmd --permanent --add-rich-rule='rule family=\"ipv4\" source address=\"%s\" port protocol=\"%s\" port=\"%d\" accept'",
					rule.Source, rule.Protocol, rule.Port)
			} else {
				cmd = fmt.Sprintf("firewall-cmd --permanent --add-port=%d/%s", rule.Port, rule.Protocol)
			}

			result, err := s.manager.client.ExecuteSudo(cmd)
			if err != nil {
				return err
			}
			if result.ExitCode != 0 {
				return &Error{
					Type:    ErrorExecution,
					Message: fmt.Sprintf("failed to add firewalld rule: %s", result.Stderr),
				}
			}
		}
	}

	result, err := s.manager.client.ExecuteSudo("firewall-cmd --reload")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to reload firewalld: %s", result.Stderr),
		}
	}

	return nil
}

func (s *SecurityManager) setupIPTables(rules []FirewallRule) error {
	s.logger.SystemOperation("Configuring iptables")
	s.manager.InstallPackages("iptables-persistent")

	cmds := []string{
		"iptables -F",
		"iptables -P INPUT DROP",
		"iptables -P FORWARD DROP",
		"iptables -P OUTPUT ACCEPT",
		"iptables -A INPUT -i lo -j ACCEPT",
		"iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT",
	}

	for _, cmd := range cmds {
		s.manager.client.ExecuteSudo(cmd)
	}

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

		s.manager.client.ExecuteSudo(cmd)
	}

	s.manager.client.ExecuteSudo("iptables-save > /etc/iptables/rules.v4")

	return nil
}

func (s *SecurityManager) HardenSSH(config SSHConfig) error {
	s.logger.SystemOperation("Hardening SSH configuration")
	s.manager.client.ExecuteSudo("cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak")

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

	configContent := strings.Join(configLines, "\n")
	cmd := fmt.Sprintf("echo '%s' > /etc/ssh/sshd_config.d/99-hardening.conf", configContent)
	result, err := s.manager.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to write SSH config: %s", result.Stderr),
		}
	}

	result, err = s.manager.client.ExecuteSudo("sshd -t")
	if err != nil || result.ExitCode != 0 {
		s.manager.client.ExecuteSudo("rm /etc/ssh/sshd_config.d/99-hardening.conf")
		return &Error{
			Type:    ErrorExecution,
			Message: "SSH configuration test failed",
		}
	}

	s.manager.ServiceRestart("sshd")

	return nil
}

func (s *SecurityManager) SetupFail2ban() error {
	s.logger.SystemOperation("Setting up fail2ban intrusion detection")
	err := s.manager.InstallPackages("fail2ban")
	if err != nil {
		return err
	}

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
	result, err := s.manager.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to configure fail2ban: %s", result.Stderr),
		}
	}

	s.manager.ServiceEnable("fail2ban")
	s.manager.ServiceRestart("fail2ban")

	return nil
}

func (s *SecurityManager) GetDefaultPocketBaseRules() []FirewallRule {
	return []FirewallRule{
		{Port: 22, Protocol: "tcp", Action: "allow", Description: "SSH"},
		{Port: 80, Protocol: "tcp", Action: "allow", Description: "HTTP"},
		{Port: 443, Protocol: "tcp", Action: "allow", Description: "HTTPS"},
	}
}

func (s *SecurityManager) GetDefaultSSHConfig() SSHConfig {
	return SSHConfig{
		PasswordAuth:        false,
		RootLogin:           false,
		PubkeyAuth:          true,
		MaxAuthTries:        3,
		ClientAliveInterval: 300,
		ClientAliveCountMax: 2,
	}
}

type SecurityConfig struct {
	FirewallRules  []FirewallRule
	HardenSSH      bool
	SSHConfig      SSHConfig
	EnableFail2ban bool
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// Close performs cleanup and closes the security manager
func (s *SecurityManager) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.logger.SystemOperation("Shutting down security manager")

	// Run all cleanup functions in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		if s.cleanup[i] != nil {
			s.cleanup[i]()
		}
	}
	s.cleanup = nil

	return nil
}

// AddCleanup adds a cleanup function to be called when the security manager is closed
func (s *SecurityManager) AddCleanup(cleanup func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.cleanup = append(s.cleanup, cleanup)
	}
}

// IsClosed returns true if the security manager has been closed
func (s *SecurityManager) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
