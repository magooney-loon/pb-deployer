package tunnel

import (
	"fmt"
	"strings"
	"time"
)

// Manager handles server operations through a single interface
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

	sshDir := fmt.Sprintf("%s/.ssh", homeDir)
	authKeysFile := fmt.Sprintf("%s/authorized_keys", sshDir)

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

// CreateDirectory creates a directory with specified permissions
func (m *Manager) CreateDirectory(path, permissions, owner, group string) error {
	// Create directory
	cmd := fmt.Sprintf("mkdir -p '%s'", path)
	result, err := m.client.ExecuteSudo(cmd)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return &Error{
			Type:    ErrorExecution,
			Message: fmt.Sprintf("failed to create directory: %s", result.Stderr),
		}
	}

	// Set permissions if specified
	if permissions != "" {
		cmd = fmt.Sprintf("chmod %s '%s'", permissions, path)
		m.client.ExecuteSudo(cmd)
	}

	// Set ownership if specified
	if owner != "" && group != "" {
		cmd = fmt.Sprintf("chown %s:%s '%s'", owner, group, path)
		m.client.ExecuteSudo(cmd)
	}

	return nil
}

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

// InstallPackages installs system packages
func (m *Manager) InstallPackages(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Detect package manager and install
	result, err := m.client.Execute("which apt", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		// Debian/Ubuntu
		cmd := fmt.Sprintf("apt update && apt install -y %s", strings.Join(packages, " "))
		result, err = m.client.ExecuteSudo(cmd, WithTimeout(5*time.Minute))
	} else {
		result, err = m.client.Execute("which yum", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			// RHEL/CentOS
			cmd := fmt.Sprintf("yum install -y %s", strings.Join(packages, " "))
			result, err = m.client.ExecuteSudo(cmd, WithTimeout(5*time.Minute))
		} else {
			result, err = m.client.Execute("which dnf", WithTimeout(5*time.Second))
			if err == nil && result.ExitCode == 0 {
				// Fedora
				cmd := fmt.Sprintf("dnf install -y %s", strings.Join(packages, " "))
				result, err = m.client.ExecuteSudo(cmd, WithTimeout(5*time.Minute))
			} else {
				return &Error{
					Type:    ErrorNotFound,
					Message: "no supported package manager found",
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
			Message: fmt.Sprintf("failed to install packages: %s", result.Stderr),
		}
	}

	return nil
}

// SystemInfo returns basic system information
func (m *Manager) SystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get OS information
	result, err := m.client.Execute("lsb_release -a 2>/dev/null || cat /etc/os-release")
	if err == nil {
		for _, line := range strings.Split(result.Stdout, "\n") {
			if strings.Contains(line, "Description:") {
				if _, after, found := strings.Cut(line, ":"); found {
					info.OS = strings.TrimSpace(after)
					break
				}
			} else if strings.Contains(line, "PRETTY_NAME=") {
				if _, after, found := strings.Cut(line, "="); found {
					info.OS = strings.Trim(after, "\"")
					break
				}
			}
		}
	}

	// Get hostname
	result, err = m.client.Execute("hostname")
	if err == nil {
		info.Hostname = strings.TrimSpace(result.Stdout)
	}

	// Get architecture
	result, err = m.client.Execute("uname -m")
	if err == nil {
		info.Architecture = strings.TrimSpace(result.Stdout)
	}

	return info, nil
}
