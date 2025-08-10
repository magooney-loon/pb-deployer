package ssh

import (
	"fmt"
	"strings"
)

// RunServerSetup performs the complete server setup process
func (sm *SSHManager) RunServerSetup(progressChan chan<- SetupStep) error {
	if !sm.isRoot {
		return fmt.Errorf("server setup requires root access")
	}

	sm.SendProgressUpdate(progressChan, "server_setup", "running", "Starting server setup process", 0)

	steps := []struct {
		name string
		fn   func(chan<- SetupStep) error
	}{
		{"create_user", sm.createPocketbaseUserWithProgress},
		{"setup_ssh_keys", sm.setupSSHKeysWithProgress},
		{"create_directories", sm.createDirectoriesWithProgress},
		{"test_connection", sm.testUserConnectionWithProgress},
	}

	totalSteps := len(steps)

	for i, step := range steps {
		// Send running status
		sm.SendProgressUpdate(progressChan, step.name, "running", fmt.Sprintf("Executing %s", step.name), (i*100)/totalSteps)

		if err := step.fn(progressChan); err != nil {
			// Send failure status
			sm.SendProgressUpdate(progressChan, step.name, "failed", fmt.Sprintf("Failed to execute %s", step.name), (i*100)/totalSteps, err.Error())
			return fmt.Errorf("setup step %s failed: %w", step.name, err)
		}

		// Send success status
		sm.SendProgressUpdate(progressChan, step.name, "success", fmt.Sprintf("Successfully completed %s", step.name), ((i+1)*100)/totalSteps)
	}

	sm.SendProgressUpdate(progressChan, "server_setup", "success", "Server setup completed successfully", 100)
	return nil
}

// createPocketbaseUser creates the pocketbase user account on the server
func (sm *SSHManager) createPocketbaseUser() error {
	return sm.createPocketbaseUserWithProgress(nil)
}

// createPocketbaseUserWithProgress creates the pocketbase user account with detailed progress reporting
func (sm *SSHManager) createPocketbaseUserWithProgress(progressChan chan<- SetupStep) error {
	username := sm.server.AppUsername

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "create_user", "running", fmt.Sprintf("Checking if user %s exists", username), 10)
	}

	// Check if user already exists
	checkUserCmd := fmt.Sprintf("id %s", username)
	output, err := sm.ExecuteCommand(checkUserCmd)
	if err == nil {
		// User already exists, check if it's properly configured
		if strings.Contains(output, "uid=") {
			if progressChan != nil {
				sm.SendProgressUpdate(progressChan, "create_user", "running", fmt.Sprintf("User %s already exists, skipping creation", username), 100)
			}
			return nil // User exists and is valid
		}
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "create_user", "running", fmt.Sprintf("Creating user %s with home directory", username), 30)
	}

	// Create the user with home directory
	createUserCmd := fmt.Sprintf("useradd -m -s /bin/bash %s", username)
	if _, err := sm.ExecuteCommand(createUserCmd); err != nil {
		return fmt.Errorf("failed to create user %s: %w", username, err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "create_user", "running", fmt.Sprintf("Adding user %s to sudo group", username), 60)
	}

	// Add user to sudo group for deployment operations
	addSudoCmd := fmt.Sprintf("usermod -aG sudo %s", username)
	if _, err := sm.ExecuteCommand(addSudoCmd); err != nil {
		return fmt.Errorf("failed to add user to sudo group: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "create_user", "running", "Configuring passwordless sudo access", 80)
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
	return sm.setupSSHKeysWithProgress(nil)
}

// setupSSHKeysWithProgress configures SSH key authentication with detailed progress reporting
func (sm *SSHManager) setupSSHKeysWithProgress(progressChan chan<- SetupStep) error {
	username := sm.server.AppUsername

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_ssh_keys", "running", fmt.Sprintf("Creating .ssh directory for user %s", username), 20)
	}

	// Create .ssh directory for the user
	sshDirCmd := fmt.Sprintf("mkdir -p /home/%s/.ssh", username)
	if _, err := sm.ExecuteCommand(sshDirCmd); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_ssh_keys", "running", "Copying SSH keys from root user", 40)
	}

	// Copy root's authorized_keys to the new user (assuming root has the keys we need)
	copyKeysCmd := fmt.Sprintf("cp /root/.ssh/authorized_keys /home/%s/.ssh/authorized_keys", username)
	if _, err := sm.ExecuteCommand(copyKeysCmd); err != nil {
		// If root doesn't have authorized_keys, try to get the current session's public key
		return fmt.Errorf("failed to copy SSH keys: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_ssh_keys", "running", "Setting SSH directory ownership", 60)
	}

	// Set proper permissions
	chownCmd := fmt.Sprintf("chown -R %s:%s /home/%s/.ssh", username, username, username)
	if _, err := sm.ExecuteCommand(chownCmd); err != nil {
		return fmt.Errorf("failed to set SSH directory ownership: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "setup_ssh_keys", "running", "Setting SSH directory permissions", 80)
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
	return sm.createDirectoriesWithProgress(nil)
}

// createDirectoriesWithProgress creates directories with detailed progress reporting
func (sm *SSHManager) createDirectoriesWithProgress(progressChan chan<- SetupStep) error {
	directories := []string{
		"/opt/pocketbase",
		"/opt/pocketbase/apps",
		"/var/log/pocketbase",
	}

	totalDirs := len(directories)

	for i, dir := range directories {
		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "create_directories", "running", fmt.Sprintf("Creating directory %s", dir), (i*100)/totalDirs)
		}

		// Create directory
		mkdirCmd := fmt.Sprintf("mkdir -p %s", dir)
		if _, err := sm.ExecuteCommand(mkdirCmd); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "create_directories", "running", fmt.Sprintf("Setting ownership for %s", dir), (i*100)/totalDirs)
		}

		// Set ownership to pocketbase user
		chownCmd := fmt.Sprintf("chown %s:%s %s", sm.server.AppUsername, sm.server.AppUsername, dir)
		if _, err := sm.ExecuteCommand(chownCmd); err != nil {
			return fmt.Errorf("failed to set ownership for %s: %w", dir, err)
		}

		if progressChan != nil {
			sm.SendProgressUpdate(progressChan, "create_directories", "running", fmt.Sprintf("Setting permissions for %s", dir), (i*100)/totalDirs)
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
	return sm.testUserConnectionWithProgress(nil)
}

// testUserConnectionWithProgress tests SSH connection with detailed progress reporting
func (sm *SSHManager) testUserConnectionWithProgress(progressChan chan<- SetupStep) error {
	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "test_connection", "running", fmt.Sprintf("Creating test SSH connection as %s", sm.server.AppUsername), 20)
	}

	// We need to create a new SSH connection as the pocketbase user to test
	testManager, err := NewSSHManager(sm.server, false)
	if err != nil {
		return fmt.Errorf("failed to create test connection as %s: %w", sm.server.AppUsername, err)
	}
	defer testManager.Close()

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "test_connection", "running", "Testing basic SSH connectivity", 40)
	}

	// Test basic command execution
	if err := testManager.TestConnection(); err != nil {
		return fmt.Errorf("user connection test failed: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "test_connection", "running", "Testing sudo access for deployment commands", 70)
	}

	// Test sudo access for required commands
	testSudoCmd := "sudo -n systemctl --version"
	if _, err := testManager.ExecuteCommand(testSudoCmd); err != nil {
		return fmt.Errorf("sudo access test failed: %w", err)
	}

	if progressChan != nil {
		sm.SendProgressUpdate(progressChan, "test_connection", "running", "Testing directory access permissions", 90)
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
