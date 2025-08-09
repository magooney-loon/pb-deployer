package ssh

import (
	"fmt"
	"strings"
	"time"
)

// RunServerSetup performs the complete server setup process
func (sm *SSHManager) RunServerSetup(progressChan chan<- SetupStep) error {
	if !sm.isRoot {
		return fmt.Errorf("server setup requires root access")
	}

	steps := []struct {
		name string
		fn   func() error
	}{
		{"create_user", sm.createPocketbaseUser},
		{"setup_ssh_keys", sm.setupSSHKeys},
		{"create_directories", sm.createDirectories},
		{"test_connection", sm.testUserConnection},
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
			return fmt.Errorf("setup step %s failed: %w", step.name, err)
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

// createPocketbaseUser creates the pocketbase user account on the server
func (sm *SSHManager) createPocketbaseUser() error {
	username := sm.server.AppUsername

	// Check if user already exists
	checkUserCmd := fmt.Sprintf("id %s", username)
	output, err := sm.ExecuteCommand(checkUserCmd)
	if err == nil {
		// User already exists, check if it's properly configured
		if strings.Contains(output, "uid=") {
			return nil // User exists and is valid
		}
	}

	// Create the user with home directory
	createUserCmd := fmt.Sprintf("useradd -m -s /bin/bash %s", username)
	if _, err := sm.ExecuteCommand(createUserCmd); err != nil {
		return fmt.Errorf("failed to create user %s: %w", username, err)
	}

	// Add user to sudo group for deployment operations
	addSudoCmd := fmt.Sprintf("usermod -aG sudo %s", username)
	if _, err := sm.ExecuteCommand(addSudoCmd); err != nil {
		return fmt.Errorf("failed to add user to sudo group: %w", err)
	}

	// Configure sudo access for specific commands without password
	sudoersContent := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: /bin/systemctl, /usr/bin/systemctl, /bin/mkdir, /usr/bin/mkdir, /bin/chown, /usr/bin/chown, /bin/chmod, /usr/bin/chmod", username)
	sudoersCmd := fmt.Sprintf("echo '%s' > /etc/sudoers.d/%s", sudoersContent, username)
	if _, err := sm.ExecuteCommand(sudoersCmd); err != nil {
		return fmt.Errorf("failed to configure sudo access: %w", err)
	}

	return nil
}

// setupSSHKeys configures SSH key authentication for the pocketbase user
func (sm *SSHManager) setupSSHKeys() error {
	username := sm.server.AppUsername

	// Create .ssh directory for the user
	sshDirCmd := fmt.Sprintf("mkdir -p /home/%s/.ssh", username)
	if _, err := sm.ExecuteCommand(sshDirCmd); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Copy root's authorized_keys to the new user (assuming root has the keys we need)
	copyKeysCmd := fmt.Sprintf("cp /root/.ssh/authorized_keys /home/%s/.ssh/authorized_keys", username)
	if _, err := sm.ExecuteCommand(copyKeysCmd); err != nil {
		// If root doesn't have authorized_keys, try to get the current session's public key
		return fmt.Errorf("failed to copy SSH keys: %w", err)
	}

	// Set proper permissions
	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s/.ssh", username, username, username)
	if _, err := sm.ExecuteCommand(chownCmd); err != nil {
		return fmt.Errorf("failed to set SSH directory ownership: %w", err)
	}

	chmodDirCmd := fmt.Sprintf("chmod 700 /home/%s/.ssh", username)
	if _, err := sm.ExecuteCommand(chmodDirCmd); err != nil {
		return fmt.Errorf("failed to set SSH directory permissions: %w", err)
	}

	chmodKeysCmd := fmt.Sprintf("chmod 600 /home/%s/.ssh/authorized_keys", username)
	if _, err := sm.ExecuteCommand(chmodKeysCmd); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w", err)
	}

	return nil
}

// createDirectories creates the required directory structure for PocketBase apps
func (sm *SSHManager) createDirectories() error {
	directories := []string{
		"/opt/pocketbase",
		"/opt/pocketbase/apps",
		"/var/log/pocketbase",
	}

	for _, dir := range directories {
		// Create directory
		mkdirCmd := fmt.Sprintf("mkdir -p %s", dir)
		if _, err := sm.ExecuteCommand(mkdirCmd); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Set ownership to pocketbase user
		chownCmd := fmt.Sprintf("chown %s:%s %s", sm.server.AppUsername, sm.server.AppUsername, dir)
		if _, err := sm.ExecuteCommand(chownCmd); err != nil {
			return fmt.Errorf("failed to set ownership for %s: %w", dir, err)
		}

		// Set appropriate permissions
		chmodCmd := fmt.Sprintf("chmod 755 %s", dir)
		if _, err := sm.ExecuteCommand(chmodCmd); err != nil {
			return fmt.Errorf("failed to set permissions for %s: %w", dir, err)
		}
	}

	return nil
}

// testUserConnection tests SSH connection as the pocketbase user
func (sm *SSHManager) testUserConnection() error {
	// We need to create a new SSH connection as the pocketbase user to test
	testManager, err := NewSSHManager(sm.server, false)
	if err != nil {
		return fmt.Errorf("failed to create test connection as %s: %w", sm.server.AppUsername, err)
	}
	defer testManager.Close()

	// Test basic command execution
	if err := testManager.TestConnection(); err != nil {
		return fmt.Errorf("user connection test failed: %w", err)
	}

	// Test sudo access for required commands
	testSudoCmd := "sudo -n systemctl --version"
	if _, err := testManager.ExecuteCommand(testSudoCmd); err != nil {
		return fmt.Errorf("sudo access test failed: %w", err)
	}

	// Test directory access
	testDirCmd := "ls -la /opt/pocketbase"
	if _, err := testManager.ExecuteCommand(testDirCmd); err != nil {
		return fmt.Errorf("directory access test failed: %w", err)
	}

	return nil
}

// VerifySetupComplete checks if the server setup has been completed successfully
func (sm *SSHManager) VerifySetupComplete() error {
	// Check if pocketbase user exists
	if _, err := sm.ExecuteCommand(fmt.Sprintf("id %s", sm.server.AppUsername)); err != nil {
		return fmt.Errorf("pocketbase user does not exist: %w", err)
	}

	// Check if directories exist
	directories := []string{"/opt/pocketbase", "/opt/pocketbase/apps", "/var/log/pocketbase"}
	for _, dir := range directories {
		if _, err := sm.ExecuteCommand(fmt.Sprintf("test -d %s", dir)); err != nil {
			return fmt.Errorf("required directory %s does not exist: %w", dir, err)
		}
	}

	// Test connection as pocketbase user
	return sm.testUserConnection()
}

// GetSetupStatus returns the current setup status of the server
func (sm *SSHManager) GetSetupStatus() (map[string]bool, error) {
	status := map[string]bool{
		"user_exists":       false,
		"ssh_configured":    false,
		"directories_exist": false,
		"sudo_configured":   false,
	}

	// Check if user exists
	if _, err := sm.ExecuteCommand(fmt.Sprintf("id %s", sm.server.AppUsername)); err == nil {
		status["user_exists"] = true
	}

	// Check SSH configuration
	sshKeyPath := fmt.Sprintf("/home/%s/.ssh/authorized_keys", sm.server.AppUsername)
	if _, err := sm.ExecuteCommand(fmt.Sprintf("test -f %s", sshKeyPath)); err == nil {
		status["ssh_configured"] = true
	}

	// Check directories
	if _, err := sm.ExecuteCommand("test -d /opt/pocketbase && test -d /opt/pocketbase/apps"); err == nil {
		status["directories_exist"] = true
	}

	// Check sudo configuration
	sudoersPath := fmt.Sprintf("/etc/sudoers.d/%s", sm.server.AppUsername)
	if _, err := sm.ExecuteCommand(fmt.Sprintf("test -f %s", sudoersPath)); err == nil {
		status["sudo_configured"] = true
	}

	return status, nil
}
