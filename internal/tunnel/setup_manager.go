package tunnel

import (
	"fmt"
	"strings"
	"time"
)

type SetupManager struct {
	manager *Manager
}

func NewSetupManager(manager *Manager) *SetupManager {
	return &SetupManager{
		manager: manager,
	}
}

func (s *SetupManager) SetupPocketBaseServer(username string, publicKeys []string) error {
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

	return nil
}

func (s *SetupManager) CreatePocketBaseDirectories(username string) error {
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

	err = s.manager.CreateDirectory("/opt/pocketbase/scripts", "755", username, username)
	if err != nil {
		return err
	}

	return nil
}

func (s *SetupManager) UpdateSystem() error {
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
	essentials := []string{
		"curl",
		"wget",
		"unzip",
		"systemd",
		"logrotate",
	}

	return s.manager.InstallPackages(essentials...)
}

func (s *SetupManager) VerifySetup(username string) error {
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
		"/opt/pocketbase/scripts",
	}

	for _, dir := range directories {
		if result, err := s.manager.client.Execute(fmt.Sprintf("test -d %s", dir)); err != nil || result.ExitCode != 0 {
			return &Error{
				Type:    ErrorVerification,
				Message: fmt.Sprintf("directory %s does not exist", dir),
			}
		}
	}

	essentials := []string{"curl", "wget", "unzip"}
	for _, pkg := range essentials {
		if result, err := s.manager.client.Execute(fmt.Sprintf("which %s", pkg)); err != nil || result.ExitCode != 0 {
			return &Error{
				Type:    ErrorVerification,
				Message: fmt.Sprintf("package %s is not installed", pkg),
			}
		}
	}

	return nil
}

func (s *SetupManager) GetSetupInfo() (*SetupInfo, error) {
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

	return info, nil
}

type SetupInfo struct {
	OS              string
	Architecture    string
	Hostname        string
	PocketBaseSetup bool
	InstalledApps   []string
}
