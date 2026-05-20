package tunnel

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"pb-deployer/internal/logger"
)

type SetupManager struct {
	manager *Manager
	logger  *logger.Logger
	cleanup []func()
	mu      sync.Mutex
	closed  bool
}

func NewSetupManager(manager *Manager) *SetupManager {
	return &SetupManager{
		manager: manager,
		logger:  logger.GetTunnelLogger(),
	}
}

func (s *SetupManager) SetupPocketBaseServer(username string, publicKeys []string, proxyEmail string) error {
	s.logger.SystemOperation(fmt.Sprintf("Setting up PocketBase server for user: %s", username))

	err := s.manager.CreateUser(username,
		WithHome(fmt.Sprintf("/home/%s", username)),
		WithShell("/bin/bash"),
		WithGroups("sudo"),
		WithSudoAccess(),
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if len(publicKeys) > 0 {
		err = s.manager.SetupSSHKeys(username, publicKeys)
		if err != nil {
			return fmt.Errorf("failed to setup SSH keys: %w", err)
		}
	}

	err = s.CreatePocketBaseDirectories(username)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	err = s.UpdateSystem()
	if err != nil {
		return fmt.Errorf("failed to update system: %w", err)
	}

	err = s.InstallEssentials()
	if err != nil {
		return fmt.Errorf("failed to install essentials: %w", err)
	}

	err = s.InstallCaddy()
	if err != nil {
		return fmt.Errorf("failed to install caddy: %w", err)
	}

	err = s.WriteBaseCaddyfile(proxyEmail)
	if err != nil {
		return fmt.Errorf("failed to write base Caddyfile: %w", err)
	}

	s.logger.Success("PocketBase server setup completed successfully")
	return nil
}

func (s *SetupManager) CreatePocketBaseDirectories(username string) error {
	s.logger.SystemOperation("Creating PocketBase directory structure")

	err := s.manager.CreateDirectory("/opt/pocketbase", "755", "root", "root")
	if err != nil {
		return err
	}

	err = s.manager.CreateDirectory("/opt/pocketbase/apps", "755", username, username)
	if err != nil {
		return err
	}

	err = s.manager.CreateDirectory("/opt/pocketbase/backups", "755", username, username)
	if err != nil {
		return err
	}

	err = s.manager.CreateDirectory("/opt/pocketbase/logs", "755", username, username)
	if err != nil {
		return err
	}

	err = s.manager.CreateDirectory("/opt/pocketbase/staging", "755", username, username)
	if err != nil {
		return err
	}

	return nil
}

func (s *SetupManager) UpdateSystem() error {
	s.logger.SystemOperation("Updating system packages")

	result, err := s.manager.client.Execute("which apt", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		// Debian/Ubuntu
		cmd := "apt update && apt upgrade -y && apt autoremove -y"
		result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
	} else {
		result, err = s.manager.client.Execute("which yum", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			// RHEL/CentOS
			cmd := "yum update -y"
			result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
		} else {
			result, err = s.manager.client.Execute("which dnf", WithTimeout(5*time.Second))
			if err == nil && result.ExitCode == 0 {
				// Fedora
				cmd := "dnf update -y"
				result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(15*time.Minute))
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
			Message: fmt.Sprintf("system update failed: %s", result.Stderr),
		}
	}

	return nil
}

func (s *SetupManager) InstallEssentials() error {
	s.logger.SystemOperation("Installing essential packages")

	essentials := []string{
		"curl",
		"wget",
		"unzip",
		"systemd",
		"logrotate",
	}

	return s.manager.InstallPackages(essentials...)
}

func (s *SetupManager) InstallCaddy() error {
	s.logger.SystemOperation("Installing Caddy")

	// Check if caddy is already installed
	result, err := s.manager.client.Execute("which caddy", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		s.logger.SystemOperation("Caddy already installed, skipping")
		return nil
	}

	// Detect package manager and install Caddy
	result, err = s.manager.client.Execute("which apt", WithTimeout(5*time.Second))
	if err == nil && result.ExitCode == 0 {
		// Debian/Ubuntu — add official Caddy apt repo
		cmds := []string{
			"apt install -y debian-keyring debian-archive-keyring apt-transport-https curl",
			"curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg",
			"curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list",
			"apt update",
			"apt install -y caddy",
		}
		for _, cmd := range cmds {
			result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(10*time.Minute))
			if err != nil || result.ExitCode != 0 {
				return fmt.Errorf("caddy install step failed (%s): %s", cmd, result.Stderr)
			}
		}
	} else {
		result, err = s.manager.client.Execute("which dnf", WithTimeout(5*time.Second))
		if err == nil && result.ExitCode == 0 {
			// Fedora
			cmds := []string{
				"dnf install -y 'dnf-command(copr)'",
				"dnf copr enable -y @caddy/caddy",
				"dnf install -y caddy",
			}
			for _, cmd := range cmds {
				result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(10*time.Minute))
				if err != nil || result.ExitCode != 0 {
					return fmt.Errorf("caddy install step failed (%s): %s", cmd, result.Stderr)
				}
			}
		} else {
			result, err = s.manager.client.Execute("which yum", WithTimeout(5*time.Second))
			if err == nil && result.ExitCode == 0 {
				// RHEL/CentOS
				cmds := []string{
					"yum install -y yum-plugin-copr",
					"yum copr enable -y @caddy/caddy",
					"yum install -y caddy",
				}
				for _, cmd := range cmds {
					result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(10*time.Minute))
					if err != nil || result.ExitCode != 0 {
						return fmt.Errorf("caddy install step failed (%s): %s", cmd, result.Stderr)
					}
				}
			} else {
				return &Error{Type: ErrorNotFound, Message: "no supported package manager found for Caddy install"}
			}
		}
	}

	// Verify installation
	result, err = s.manager.client.Execute("which caddy", WithTimeout(5*time.Second))
	if err != nil || result.ExitCode != 0 {
		return &Error{Type: ErrorVerification, Message: "caddy binary not found after install"}
	}

	s.logger.Success("Caddy installed successfully")
	return nil
}

func (s *SetupManager) WriteBaseCaddyfile(email string) error {
	s.logger.SystemOperation("Writing base Caddyfile")

	// Create conf.d directory
	result, err := s.manager.client.ExecuteSudo("mkdir -p /etc/caddy/conf.d", WithTimeout(10*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create /etc/caddy/conf.d: %s", result.Stderr)
	}

	// Build Caddyfile content
	var emailLine string
	if email != "" {
		emailLine = "\n\temail " + email
	}
	caddyfileContent := fmt.Sprintf(`{%s
	admin off
}

import /etc/caddy/conf.d/*.caddy
`, emailLine)

	// Write idempotently — only if content differs
	writeCmd := fmt.Sprintf("cat > /etc/caddy/Caddyfile << 'CADDYEOF'\n%sCaddyEOF", caddyfileContent)
	result, err = s.manager.client.ExecuteSudo(writeCmd, WithTimeout(10*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write Caddyfile: %s", result.Stderr)
	}

	// Enable caddy (do not start yet — configureReverseProxy starts/reloads it per-deploy)
	result, err = s.manager.client.ExecuteSudo("systemctl enable caddy", WithTimeout(15*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to enable caddy service: %s", result.Stderr)
	}

	s.logger.Success("Base Caddyfile written and caddy enabled")
	return nil
}

type MigrateProxyRequest struct {
	AppName     string
	Domain      string
	ServiceName string
	AppUsername string
	HTTPPort    int
}

func (s *SetupManager) MigrateAppToProxy(req MigrateProxyRequest) error {
	s.logger.SystemOperation(fmt.Sprintf("Migrating %s to Caddy proxy (port %d)", req.AppName, req.HTTPPort))

	binaryPath := fmt.Sprintf("/opt/pocketbase/apps/%s/%s", req.AppName, req.AppName)
	workingDir := fmt.Sprintf("/opt/pocketbase/apps/%s", req.AppName)
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", req.ServiceName)

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s PocketBase Server
After=network.target caddy.service

[Service]
Type=simple
User=%s
Group=%s
LimitNOFILE=4096
Restart=always
RestartSec=5s
StandardOutput=append:/opt/pocketbase/logs/%s.log
StandardError=append:/opt/pocketbase/logs/%s.log
WorkingDirectory=%s
ExecStart=%s serve %s --http=127.0.0.1:%d

[Install]
WantedBy=multi-user.target
`, req.AppName, req.AppUsername, req.AppUsername, req.AppName, req.AppName, workingDir, binaryPath, req.Domain, req.HTTPPort)

	writeServiceCmd := fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", servicePath, serviceContent)
	result, err := s.manager.client.ExecuteSudo(writeServiceCmd, WithTimeout(15*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to rewrite systemd unit: %s", result.Stderr)
	}

	fragmentPath := fmt.Sprintf("/etc/caddy/conf.d/%s.caddy", req.AppName)
	fragmentContent := fmt.Sprintf(`%s {
	encode zstd gzip
	reverse_proxy 127.0.0.1:%d {
		header_up Host {host}
		header_up X-Real-IP {remote_host}
		header_up X-Forwarded-For {remote_host}
		header_up X-Forwarded-Proto {scheme}
	}

	log {
		output file /opt/pocketbase/logs/%s.access.log {
			roll_size 10mb
			roll_keep 5
		}
	}
}
`, req.Domain, req.HTTPPort, req.AppName)

	writeFragmentCmd := fmt.Sprintf("cat > %s << 'FRAGMENTEOF'\n%sFRAGMENTEOF", fragmentPath, fragmentContent)
	result, err = s.manager.client.ExecuteSudo(writeFragmentCmd, WithTimeout(15*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to write Caddy fragment: %s", result.Stderr)
	}

	cmds := []string{
		"systemctl daemon-reload",
		fmt.Sprintf("systemctl restart %s", req.ServiceName),
		"systemctl reload caddy 2>/dev/null || systemctl restart caddy",
	}
	for _, cmd := range cmds {
		result, err = s.manager.client.ExecuteSudo(cmd, WithTimeout(30*time.Second))
		if err != nil || result.ExitCode != 0 {
			s.logger.Warning("migrate-proxy cmd %q failed: %s", cmd, result.Stderr)
		}
	}

	s.logger.Success("App %s migrated to Caddy proxy", req.AppName)
	return nil
}

type CleanupAppRequest struct {
	AppName     string
	ServiceName string
}

func (s *SetupManager) CleanupApp(req CleanupAppRequest) error {
	s.logger.SystemOperation(fmt.Sprintf("Cleaning up remote resources for app: %s", req.AppName))

	cmds := []string{
		fmt.Sprintf("systemctl stop %s 2>/dev/null || true", req.ServiceName),
		fmt.Sprintf("systemctl disable %s 2>/dev/null || true", req.ServiceName),
		fmt.Sprintf("rm -f /etc/systemd/system/%s.service", req.ServiceName),
		fmt.Sprintf("rm -f /etc/caddy/conf.d/%s.caddy", req.AppName),
		"systemctl daemon-reload",
		"systemctl reload caddy 2>/dev/null || systemctl restart caddy 2>/dev/null || true",
	}

	for _, cmd := range cmds {
		result, err := s.manager.client.ExecuteSudo(cmd, WithTimeout(30*time.Second))
		if err != nil || result.ExitCode != 0 {
			s.logger.Warning("cleanup cmd %q failed: %s", cmd, result.Stderr)
		}
	}

	s.logger.Success("Remote cleanup for app %s completed", req.AppName)
	return nil
}

func (s *SetupManager) VerifySetup(username string) error {
	s.logger.SystemOperation(fmt.Sprintf("Verifying setup for user: %s", username))

	result, err := s.manager.client.Execute(fmt.Sprintf("id %s", username))
	if err != nil || result.ExitCode != 0 {
		return &Error{
			Type:    ErrorVerification,
			Message: fmt.Sprintf("user %s does not exist", username),
		}
	}

	result, err = s.manager.client.Execute(fmt.Sprintf("sudo -l -U %s", username))
	if err != nil || result.ExitCode != 0 {
		return &Error{
			Type:    ErrorVerification,
			Message: fmt.Sprintf("user %s does not have sudo access", username),
		}
	}

	directories := []string{
		"/opt/pocketbase",
		"/opt/pocketbase/apps",
		"/opt/pocketbase/backups",
		"/opt/pocketbase/logs",
		"/opt/pocketbase/staging",
		"/etc/caddy",
		"/etc/caddy/conf.d",
	}

	for _, dir := range directories {
		if result, err := s.manager.client.Execute(fmt.Sprintf("test -d %s", dir)); err != nil || result.ExitCode != 0 {
			return &Error{
				Type:    ErrorVerification,
				Message: fmt.Sprintf("directory %s does not exist", dir),
			}
		}
	}

	if result, err := s.manager.client.Execute("test -f /etc/caddy/Caddyfile"); err != nil || result.ExitCode != 0 {
		return &Error{Type: ErrorVerification, Message: "/etc/caddy/Caddyfile does not exist"}
	}

	essentials := []string{"curl", "wget", "unzip", "caddy"}
	for _, pkg := range essentials {
		if result, err := s.manager.client.Execute(fmt.Sprintf("which %s", pkg)); err != nil || result.ExitCode != 0 {
			return &Error{
				Type:    ErrorVerification,
				Message: fmt.Sprintf("package %s is not installed", pkg),
			}
		}
	}

	s.logger.Success("Setup verification completed successfully")
	return nil
}

func (s *SetupManager) GetSetupInfo() (*SetupInfo, error) {
	s.logger.SystemOperation("Gathering setup information")

	info := &SetupInfo{}

	sysInfo, err := s.manager.SystemInfo()
	if err == nil {
		info.OS = sysInfo.OS
		info.Architecture = sysInfo.Architecture
		info.Hostname = sysInfo.Hostname
	}

	result, err := s.manager.client.Execute("test -d /opt/pocketbase")
	info.PocketBaseSetup = (err == nil && result.ExitCode == 0)

	if info.PocketBaseSetup {
		if result, err := s.manager.client.Execute("ls -1 /opt/pocketbase/apps"); err == nil && result.ExitCode == 0 {
			for _, app := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
				if app != "" {
					info.InstalledApps = append(info.InstalledApps, app)
				}
			}
		}
	}

	result, err = s.manager.client.Execute("which caddy")
	info.CaddyInstalled = (err == nil && result.ExitCode == 0)

	return info, nil
}

type SetupInfo struct {
	OS              string
	Architecture    string
	Hostname        string
	PocketBaseSetup bool
	CaddyInstalled  bool
	InstalledApps   []string
}

// Close performs cleanup and closes the setup manager
func (s *SetupManager) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	s.logger.SystemOperation("Shutting down setup manager")

	// Run all cleanup functions in reverse order
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		if s.cleanup[i] != nil {
			s.cleanup[i]()
		}
	}
	s.cleanup = nil

	return nil
}

// AddCleanup adds a cleanup function to be called when the setup manager is closed
func (s *SetupManager) AddCleanup(cleanup func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.cleanup = append(s.cleanup, cleanup)
	}
}

// IsClosed returns true if the setup manager has been closed
func (s *SetupManager) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
