package managers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pb-deployer/internal/tunnel"
	"pb-deployer/internal/utils"
)

// setupManager implements the SetupManager interface
type setupManager struct {
	executor tunnel.Executor
	tracer   tunnel.ServiceTracer
	config   tunnel.SetupConfig
}

// NewSetupManager creates a new setup manager with default configuration
func NewSetupManager(executor tunnel.Executor, tracer tunnel.ServiceTracer) tunnel.SetupManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &setupManager{
		executor: executor,
		tracer:   tracer,
		config:   tunnel.DefaultSetupConfig(),
	}
}

// NewSetupManagerWithConfig creates a new setup manager with custom configuration
func NewSetupManagerWithConfig(executor tunnel.Executor, tracer tunnel.ServiceTracer, config tunnel.SetupConfig) tunnel.SetupManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &setupManager{
		executor: executor,
		tracer:   tracer,
		config:   config,
	}
}

// CreateUser creates a new user on the server
func (sm *setupManager) CreateUser(ctx context.Context, user tunnel.UserConfig) error {
	span := sm.tracer.TraceSetupOperation(ctx, "create_user", user.Username)
	defer span.End()

	span.SetFields(map[string]any{
		"username":    user.Username,
		"home_dir":    user.HomeDir,
		"shell":       user.Shell,
		"groups":      user.Groups,
		"create_home": user.CreateHome,
		"system_user": user.SystemUser,
	})

	// Validate user configuration
	if err := sm.validateUserConfig(user); err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("validate_user", "invalid user configuration", err)
	}

	// Set defaults
	user = sm.setUserDefaults(user)

	// Report progress
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_user",
		Status:      "running",
		Message:     fmt.Sprintf("Creating user %s", user.Username),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Check if user already exists
	exists, err := sm.userExists(ctx, user.Username)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("check_user_exists", "failed to check if user exists", err)
	}

	if exists {
		span.Event("user_already_exists", map[string]any{"username": user.Username})
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "create_user",
			Status:      "skipped",
			Message:     fmt.Sprintf("User %s already exists", user.Username),
			ProgressPct: 100,
			Timestamp:   time.Now(),
		})
		return nil
	}

	// Build create user command
	cmd, err := sm.buildCreateUserCommand(user)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("build_command", "failed to build user creation command", err)
	}

	// Execute user creation
	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("execute_command", "failed to create user", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("user creation failed with exit code %d: %s", result.ExitCode, result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("user_creation", "command failed", err)
	}

	span.Event("user_created", map[string]any{
		"username":  user.Username,
		"exit_code": result.ExitCode,
		"duration":  result.Duration,
	})

	// Setup home directory if requested
	if user.CreateHome && user.HomeDir != "" {
		if err := sm.setupHomeDirectory(ctx, user); err != nil {
			span.EndWithError(err)
			return tunnel.WrapSetupError("setup_home", "failed to setup home directory", err)
		}
	}

	// Add user to groups if specified
	if len(user.Groups) > 0 {
		if err := sm.addUserToGroups(ctx, user.Username, user.Groups); err != nil {
			span.EndWithError(err)
			return tunnel.WrapSetupError("add_groups", "failed to add user to groups", err)
		}
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_user",
		Status:      "success",
		Message:     fmt.Sprintf("User %s created successfully", user.Username),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// SetupSSHKeys configures SSH keys for a user
func (sm *setupManager) SetupSSHKeys(ctx context.Context, user string, keys []string) error {
	span := sm.tracer.TraceSetupOperation(ctx, "setup_ssh_keys", user)
	defer span.End()

	span.SetFields(map[string]any{
		"username":  user,
		"key_count": len(keys),
	})

	if len(keys) == 0 {
		span.Event("no_keys_provided")
		return nil
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_ssh_keys",
		Status:      "running",
		Message:     fmt.Sprintf("Setting up SSH keys for user %s", user),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Get user home directory
	homeDir, err := sm.getUserHomeDirectory(ctx, user)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("get_home_dir", "failed to get user home directory", err)
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	// Create .ssh directory
	createDirCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("mkdir -p %s", utils.ShellEscape(sshDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, createDirCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_ssh_dir", "failed to create .ssh directory", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to create .ssh directory: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_ssh_dir", "command failed", err)
	}

	// Set .ssh directory permissions
	chmodCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chmod 700 %s", utils.ShellEscape(sshDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, chmodCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_ssh_dir", "failed to set .ssh directory permissions", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to set .ssh directory permissions: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_ssh_dir", "command failed", err)
	}

	// Create authorized_keys file with SSH keys
	authorizedKeysPath := filepath.Join(sshDir, "authorized_keys")
	keysContent := strings.Join(keys, "\n") + "\n"

	createKeysCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", utils.ShellEscape(authorizedKeysPath), keysContent),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, createKeysCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_authorized_keys", "failed to create authorized_keys file", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to create authorized_keys file: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_authorized_keys", "command failed", err)
	}

	// Set authorized_keys permissions
	chmodKeysCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chmod 600 %s", utils.ShellEscape(authorizedKeysPath)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, chmodKeysCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_authorized_keys", "failed to set authorized_keys permissions", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to set authorized_keys permissions: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_authorized_keys", "command failed", err)
	}

	// Set ownership of .ssh directory and contents
	chownCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chown -R %s:%s %s", utils.ShellEscape(user), utils.ShellEscape(user), utils.ShellEscape(sshDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, chownCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("chown_ssh_dir", "failed to set .ssh directory ownership", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to set .ssh directory ownership: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("chown_ssh_dir", "command failed", err)
	}

	span.Event("ssh_keys_configured", map[string]any{
		"username":             user,
		"keys_count":           len(keys),
		"authorized_keys_path": authorizedKeysPath,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_ssh_keys",
		Status:      "success",
		Message:     fmt.Sprintf("SSH keys configured for user %s", user),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// CreateDirectories creates required directories
func (sm *setupManager) CreateDirectories(ctx context.Context, dirs []tunnel.DirectoryConfig) error {
	span := sm.tracer.TraceSetupOperation(ctx, "create_directories", fmt.Sprintf("%d directories", len(dirs)))
	defer span.End()

	span.SetFields(map[string]any{
		"directory_count": len(dirs),
	})

	if len(dirs) == 0 {
		span.Event("no_directories_specified")
		return nil
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_directories",
		Status:      "running",
		Message:     fmt.Sprintf("Creating %d directories", len(dirs)),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	for i, dir := range dirs {
		progress := int(float64(i+1) / float64(len(dirs)) * 90)
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "create_directories",
			Status:      "running",
			Message:     fmt.Sprintf("Creating directory %s", dir.Path),
			ProgressPct: 10 + progress,
			Timestamp:   time.Now(),
		})

		if err := sm.createDirectory(ctx, dir); err != nil {
			span.EndWithError(err)
			return tunnel.WrapSetupError("create_directory", fmt.Sprintf("failed to create directory %s", dir.Path), err)
		}
	}

	span.Event("directories_created", map[string]any{
		"count": len(dirs),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_directories",
		Status:      "success",
		Message:     fmt.Sprintf("Created %d directories successfully", len(dirs)),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// ConfigureSudo configures sudo access for a user
func (sm *setupManager) ConfigureSudo(ctx context.Context, user string, commands []string) error {
	span := sm.tracer.TraceSetupOperation(ctx, "configure_sudo", user)
	defer span.End()

	span.SetFields(map[string]any{
		"username":      user,
		"command_count": len(commands),
	})

	if len(commands) == 0 {
		span.Event("no_sudo_commands_specified")
		return nil
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "configure_sudo",
		Status:      "running",
		Message:     fmt.Sprintf("Configuring sudo access for user %s", user),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Build sudo configuration
	sudoConfig := sm.buildSudoConfig(user, commands)
	sudoFilePath := fmt.Sprintf("/etc/sudoers.d/%s", user)

	// Create sudo configuration file
	createSudoCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", utils.ShellEscape(sudoFilePath), sudoConfig),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, createSudoCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_sudo_config", "failed to create sudo configuration", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to create sudo configuration: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("create_sudo_config", "command failed", err)
	}

	// Set sudo file permissions
	chmodSudoCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chmod 440 %s", utils.ShellEscape(sudoFilePath)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, chmodSudoCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_sudo_config", "failed to set sudo file permissions", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to set sudo file permissions: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("chmod_sudo_config", "command failed", err)
	}

	// Validate sudo configuration
	validateCmd := tunnel.Command{
		Cmd:     "visudo -c",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, validateCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("validate_sudo_config", "failed to validate sudo configuration", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("sudo configuration validation failed: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("validate_sudo_config", "configuration invalid", err)
	}

	span.Event("sudo_configured", map[string]any{
		"username":      user,
		"config_file":   sudoFilePath,
		"command_count": len(commands),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "configure_sudo",
		Status:      "success",
		Message:     fmt.Sprintf("Sudo access configured for user %s", user),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// InstallPackages installs system packages
func (sm *setupManager) InstallPackages(ctx context.Context, packages []string) error {
	span := sm.tracer.TraceSetupOperation(ctx, "install_packages", fmt.Sprintf("%d packages", len(packages)))
	defer span.End()

	span.SetFields(map[string]any{
		"package_count": len(packages),
		"packages":      packages,
	})

	if len(packages) == 0 {
		span.Event("no_packages_specified")
		return nil
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "install_packages",
		Status:      "running",
		Message:     fmt.Sprintf("Installing %d packages", len(packages)),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Detect package manager
	packageManager, err := sm.detectPackageManager(ctx)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("detect_package_manager", "failed to detect package manager", err)
	}

	span.Event("package_manager_detected", map[string]any{
		"package_manager": packageManager,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "install_packages",
		Status:      "running",
		Message:     fmt.Sprintf("Updating package lists (%s)", packageManager),
		ProgressPct: 20,
		Timestamp:   time.Now(),
	})

	// Update package lists
	updateCmd, err := sm.buildPackageUpdateCommand(packageManager)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("build_update_command", "failed to build package update command", err)
	}

	result, err := sm.executor.RunCommand(ctx, updateCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("update_packages", "failed to update package lists", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("package update failed: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("update_packages", "command failed", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "install_packages",
		Status:      "running",
		Message:     "Installing packages",
		ProgressPct: 40,
		Timestamp:   time.Now(),
	})

	// Install packages
	installCmd, err := sm.buildPackageInstallCommand(packageManager, packages)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("build_install_command", "failed to build package install command", err)
	}

	result, err = sm.executor.RunCommand(ctx, installCmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapSetupError("install_packages", "failed to install packages", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("package installation failed: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapSetupError("install_packages", "command failed", err)
	}

	span.Event("packages_installed", map[string]any{
		"package_manager": packageManager,
		"package_count":   len(packages),
		"packages":        packages,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "install_packages",
		Status:      "success",
		Message:     fmt.Sprintf("Installed %d packages successfully", len(packages)),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// SetupSystemUser sets up a complete system user with all configurations
func (sm *setupManager) SetupSystemUser(ctx context.Context, config tunnel.SystemUserConfig) error {
	span := sm.tracer.TraceSetupOperation(ctx, "setup_system_user", config.Username)
	defer span.End()

	span.SetFields(map[string]any{
		"username":        config.Username,
		"setup_ssh":       config.SetupSSH,
		"setup_sudo":      config.SetupSudo,
		"directory_count": len(config.Directories),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_system_user",
		Status:      "running",
		Message:     fmt.Sprintf("Setting up system user %s", config.Username),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Create user
	userConfig := tunnel.UserConfig{
		Username:   config.Username,
		HomeDir:    config.HomeDir,
		Shell:      config.Shell,
		Groups:     config.Groups,
		CreateHome: config.CreateHome,
		SystemUser: config.SystemUser,
	}

	if err := sm.CreateUser(ctx, userConfig); err != nil {
		span.EndWithError(err)
		return err
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_system_user",
		Status:      "running",
		Message:     fmt.Sprintf("User %s created, setting up SSH", config.Username),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	// Setup SSH keys if requested
	if config.SetupSSH && len(config.SSHKeys) > 0 {
		if err := sm.SetupSSHKeys(ctx, config.Username, config.SSHKeys); err != nil {
			span.EndWithError(err)
			return err
		}
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_system_user",
		Status:      "running",
		Message:     fmt.Sprintf("SSH configured for %s, setting up sudo", config.Username),
		ProgressPct: 60,
		Timestamp:   time.Now(),
	})

	// Configure sudo if requested
	if config.SetupSudo && len(config.SudoCommands) > 0 {
		if err := sm.ConfigureSudo(ctx, config.Username, config.SudoCommands); err != nil {
			span.EndWithError(err)
			return err
		}
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_system_user",
		Status:      "running",
		Message:     fmt.Sprintf("Sudo configured for %s, creating directories", config.Username),
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	// Create directories if specified
	if len(config.Directories) > 0 {
		if err := sm.CreateDirectories(ctx, config.Directories); err != nil {
			span.EndWithError(err)
			return err
		}
	}

	span.Event("system_user_setup_complete", map[string]any{
		"username": config.Username,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "setup_system_user",
		Status:      "success",
		Message:     fmt.Sprintf("System user %s setup completed successfully", config.Username),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// Helper methods

func (sm *setupManager) validateUserConfig(user tunnel.UserConfig) error {
	if user.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if user.Shell != "" && !strings.HasPrefix(user.Shell, "/") {
		return fmt.Errorf("shell must be an absolute path")
	}

	return nil
}

func (sm *setupManager) setUserDefaults(user tunnel.UserConfig) tunnel.UserConfig {
	if user.Shell == "" {
		user.Shell = sm.config.DefaultShell
		if user.Shell == "" {
			user.Shell = "/bin/bash"
		}
	}

	if user.HomeDir == "" && !user.SystemUser {
		user.HomeDir = fmt.Sprintf("/home/%s", user.Username)
	}

	if len(user.Groups) == 0 && len(sm.config.DefaultGroups) > 0 {
		user.Groups = sm.config.DefaultGroups
	}

	return user
}

func (sm *setupManager) userExists(ctx context.Context, username string) (bool, error) {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("id %s", utils.ShellEscape(username)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}

func (sm *setupManager) buildCreateUserCommand(user tunnel.UserConfig) (tunnel.Command, error) {
	args := []string{"useradd"}

	if user.CreateHome {
		args = append(args, "-m")
	}

	if user.HomeDir != "" {
		args = append(args, "-d", utils.ShellEscape(user.HomeDir))
	}

	if user.Shell != "" {
		args = append(args, "-s", utils.ShellEscape(user.Shell))
	}

	if user.SystemUser {
		args = append(args, "-r")
	}

	args = append(args, utils.ShellEscape(user.Username))

	return tunnel.Command{
		Cmd:     strings.Join(args, " "),
		Sudo:    true,
		Timeout: 60 * time.Second,
	}, nil
}

func (sm *setupManager) setupHomeDirectory(ctx context.Context, user tunnel.UserConfig) error {
	chownCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chown %s:%s %s", utils.ShellEscape(user.Username), utils.ShellEscape(user.Username), utils.ShellEscape(user.HomeDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, chownCmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to set home directory ownership: %s", result.Output)
	}

	// Set home directory permissions
	chmodCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("chmod 755 %s", utils.ShellEscape(user.HomeDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, chmodCmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to set home directory permissions: %s", result.Output)
	}

	return nil
}

func (sm *setupManager) addUserToGroups(ctx context.Context, username string, groups []string) error {
	for _, group := range groups {
		cmd := tunnel.Command{
			Cmd:     fmt.Sprintf("usermod -aG %s %s", utils.ShellEscape(group), utils.ShellEscape(username)),
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, cmd)
		if err != nil {
			return err
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to add user to group %s: %s", group, result.Output)
		}
	}

	return nil
}

func (sm *setupManager) getUserHomeDirectory(ctx context.Context, username string) (string, error) {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("getent passwd %s | cut -d: -f6", utils.ShellEscape(username)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return "", err
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to get home directory for user %s: %s", username, result.Output)
	}

	return strings.TrimSpace(result.Output), nil
}

func (sm *setupManager) createDirectory(ctx context.Context, dir tunnel.DirectoryConfig) error {
	// Create directory
	var mkdirArgs []string
	if dir.Parents {
		mkdirArgs = append(mkdirArgs, "-p")
	}
	mkdirArgs = append(mkdirArgs, utils.ShellEscape(dir.Path))

	mkdirCmd := tunnel.Command{
		Cmd:     fmt.Sprintf("mkdir %s", strings.Join(mkdirArgs, " ")),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, mkdirCmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to create directory %s: %s", dir.Path, result.Output)
	}

	// Set permissions if specified
	if dir.Permissions != "" {
		chmodCmd := tunnel.Command{
			Cmd:     fmt.Sprintf("chmod %s %s", utils.ShellEscape(dir.Permissions), utils.ShellEscape(dir.Path)),
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, chmodCmd)
		if err != nil {
			return err
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to set permissions on directory %s: %s", dir.Path, result.Output)
		}
	}

	// Set ownership if specified
	if dir.Owner != "" && dir.Group != "" {
		chownCmd := tunnel.Command{
			Cmd:     fmt.Sprintf("chown %s:%s %s", utils.ShellEscape(dir.Owner), utils.ShellEscape(dir.Group), utils.ShellEscape(dir.Path)),
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := sm.executor.RunCommand(ctx, chownCmd)
		if err != nil {
			return err
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to set ownership on directory %s: %s", dir.Path, result.Output)
		}
	}

	return nil
}

func (sm *setupManager) buildSudoConfig(user string, commands []string) string {
	var lines []string

	// Add comment header
	lines = append(lines, fmt.Sprintf("# Sudo configuration for %s", user))
	lines = append(lines, fmt.Sprintf("# Generated by pb-deployer at %s", time.Now().Format(time.RFC3339)))
	lines = append(lines, "")

	if len(commands) == 0 {
		// Full sudo access
		lines = append(lines, fmt.Sprintf("%s ALL=(ALL:ALL) NOPASSWD:ALL", user))
	} else {
		// Specific commands
		cmdList := strings.Join(commands, ", ")
		lines = append(lines, fmt.Sprintf("%s ALL=(ALL:ALL) NOPASSWD: %s", user, cmdList))
	}

	return strings.Join(lines, "\n") + "\n"
}

func (sm *setupManager) detectPackageManager(ctx context.Context) (string, error) {
	if sm.config.PackageManager != "auto" {
		return sm.config.PackageManager, nil
	}

	// Try to detect package manager
	managers := []struct {
		name    string
		command string
	}{
		{"apt", "which apt"},
		{"yum", "which yum"},
		{"dnf", "which dnf"},
		{"pacman", "which pacman"},
		{"zypper", "which zypper"},
	}

	for _, mgr := range managers {
		cmd := tunnel.Command{
			Cmd:     mgr.command,
			Sudo:    false,
			Timeout: 10 * time.Second,
		}
		result, err := sm.executor.RunCommand(ctx, cmd)
		if err == nil && result.ExitCode == 0 {
			return mgr.name, nil
		}
	}

	return "", fmt.Errorf("could not detect package manager")
}

func (sm *setupManager) buildPackageUpdateCommand(packageManager string) (tunnel.Command, error) {
	switch packageManager {
	case "apt":
		return tunnel.Command{
			Cmd:     "apt update",
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}, nil
	case "yum":
		return tunnel.Command{
			Cmd:     "yum makecache",
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}, nil
	case "dnf":
		return tunnel.Command{
			Cmd:     "dnf makecache",
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}, nil
	case "pacman":
		return tunnel.Command{
			Cmd:     "pacman -Sy",
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}, nil
	case "zypper":
		return tunnel.Command{
			Cmd:     "zypper refresh",
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}, nil
	default:
		return tunnel.Command{}, fmt.Errorf("unsupported package manager: %s", packageManager)
	}
}

func (sm *setupManager) buildPackageInstallCommand(packageManager string, packages []string) (tunnel.Command, error) {
	if len(packages) == 0 {
		return tunnel.Command{}, fmt.Errorf("no packages specified")
	}

	packageList := strings.Join(packages, " ")

	switch packageManager {
	case "apt":
		return tunnel.Command{
			Cmd:     fmt.Sprintf("apt install -y %s", packageList),
			Sudo:    true,
			Timeout: 15 * time.Minute,
		}, nil
	case "yum":
		return tunnel.Command{
			Cmd:     fmt.Sprintf("yum install -y %s", packageList),
			Sudo:    true,
			Timeout: 15 * time.Minute,
		}, nil
	case "dnf":
		return tunnel.Command{
			Cmd:     fmt.Sprintf("dnf install -y %s", packageList),
			Sudo:    true,
			Timeout: 15 * time.Minute,
		}, nil
	case "pacman":
		return tunnel.Command{
			Cmd:     fmt.Sprintf("pacman -S --noconfirm %s", packageList),
			Sudo:    true,
			Timeout: 15 * time.Minute,
		}, nil
	case "zypper":
		return tunnel.Command{
			Cmd:     fmt.Sprintf("zypper install -y %s", packageList),
			Sudo:    true,
			Timeout: 15 * time.Minute,
		}, nil
	default:
		return tunnel.Command{}, fmt.Errorf("unsupported package manager: %s", packageManager)
	}
}

func (sm *setupManager) reportProgress(ctx context.Context, update tunnel.ProgressUpdate) {
	if reporter, ok := tunnel.GetProgressReporter(ctx); ok {
		reporter.Report(update)
	}
}

// SetConfig updates the setup manager configuration
func (sm *setupManager) SetConfig(config tunnel.SetupConfig) {
	sm.config = config
}

// GetConfig returns the current setup manager configuration
func (sm *setupManager) GetConfig() tunnel.SetupConfig {
	return sm.config
}
