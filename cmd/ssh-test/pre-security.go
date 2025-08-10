package main

import (
	"fmt"
	"strings"

	"pb-deployer/internal/models"
	sshpkg "pb-deployer/internal/ssh"
)

// PreSecurityCheck represents a single pre-security diagnostic check
type PreSecurityCheck struct {
	Name        string
	Status      string // "success", "error", "warning", "info"
	Message     string
	Details     string
	Suggestions []string
}

// PreSecurityResult contains all pre-security diagnostic results
type PreSecurityResult struct {
	Checks           []PreSecurityCheck
	Summary          string
	ReadyForLockdown bool
	Suggestions      []string
}

// testRootSSHAccess tests that root SSH access is working (required before lockdown)
func testRootSSHAccess(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "Root SSH Access",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		check.Status = "error"
		check.Message = "Root SSH connection failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Ensure root SSH access is enabled",
			"Check SSH key configuration for root",
			"Verify SSH service is running",
			"Root access is required to perform security lockdown",
		}
		return check
	}
	defer manager.Close()

	if err := manager.TestConnection(); err != nil {
		check.Status = "error"
		check.Message = "Root SSH connection test failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Check SSH configuration",
			"Verify network connectivity",
			"Check firewall settings",
		}
		return check
	}

	check.Status = "success"
	check.Message = "Root SSH access is working"
	check.Details = "Root access is available for security lockdown procedures"
	return check
}

// checkAppUserExists verifies that the app user exists and is properly configured
func checkAppUserExists(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "App User Existence",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot check app user - root SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	// Check if user exists
	output, err := manager.ExecuteCommand(fmt.Sprintf("id %s", server.AppUsername))
	if err != nil {
		check.Status = "error"
		check.Message = fmt.Sprintf("App user %s does not exist", server.AppUsername)
		check.Details = err.Error()
		check.Suggestions = []string{
			fmt.Sprintf("Create user: useradd -m -s /bin/bash %s", server.AppUsername),
			fmt.Sprintf("Set user password: passwd %s", server.AppUsername),
			"Create home directory if missing",
		}
		return check
	}

	// Check if user has a home directory
	homeOutput, err := manager.ExecuteCommand(fmt.Sprintf("test -d /home/%s && echo 'exists' || echo 'missing'", server.AppUsername))
	if err != nil {
		check.Status = "warning"
		check.Message = fmt.Sprintf("Cannot verify home directory for %s", server.AppUsername)
		check.Details = err.Error()
	} else {
		homeStatus := strings.TrimSpace(homeOutput)
		if homeStatus != "exists" {
			check.Status = "warning"
			check.Message = fmt.Sprintf("Home directory missing for %s", server.AppUsername)
			check.Suggestions = []string{
				fmt.Sprintf("Create home directory: mkdir -p /home/%s", server.AppUsername),
				fmt.Sprintf("Set ownership: chown %s:%s /home/%s", server.AppUsername, server.AppUsername, server.AppUsername),
				fmt.Sprintf("Set permissions: chmod 755 /home/%s", server.AppUsername),
			}
			return check
		}
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("App user %s exists and has home directory", server.AppUsername)
	check.Details = strings.TrimSpace(output)
	return check
}

// setupAppUserSSHKeys ensures SSH keys are properly configured for app user
func setupAppUserSSHKeys(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "App User SSH Keys Setup",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot setup SSH keys - root SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	var commands []string
	var results []string

	// Ensure .ssh directory exists
	commands = append(commands, fmt.Sprintf("mkdir -p /home/%s/.ssh", server.AppUsername))

	// Copy authorized_keys from root if it doesn't exist
	commands = append(commands, fmt.Sprintf("if [ ! -f /home/%s/.ssh/authorized_keys ]; then cp /root/.ssh/authorized_keys /home/%s/.ssh/; fi", server.AppUsername, server.AppUsername))

	// Set proper ownership
	commands = append(commands, fmt.Sprintf("chown -R %s:%s /home/%s/.ssh", server.AppUsername, server.AppUsername, server.AppUsername))

	// Set proper permissions
	commands = append(commands, fmt.Sprintf("chmod 700 /home/%s/.ssh", server.AppUsername))
	commands = append(commands, fmt.Sprintf("chmod 600 /home/%s/.ssh/authorized_keys", server.AppUsername))

	for _, cmd := range commands {
		output, err := manager.ExecuteCommand(cmd)
		if err != nil {
			check.Status = "error"
			check.Message = "Failed to setup SSH keys for app user"
			check.Details = fmt.Sprintf("Command failed: %s - %v", cmd, err)
			check.Suggestions = []string{
				"Manually copy SSH keys from root to app user",
				"Check file permissions and ownership",
				"Ensure authorized_keys file exists",
			}
			return check
		}
		if output != "" {
			results = append(results, strings.TrimSpace(output))
		}
	}

	// Verify the setup worked
	testOutput, err := manager.ExecuteCommand(fmt.Sprintf("test -f /home/%s/.ssh/authorized_keys && echo 'success' || echo 'failed'", server.AppUsername))
	if err != nil || strings.TrimSpace(testOutput) != "success" {
		check.Status = "error"
		check.Message = "SSH keys setup verification failed"
		check.Details = "authorized_keys file not found after setup"
		return check
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("SSH keys configured successfully for %s", server.AppUsername)
	if len(results) > 0 {
		check.Details = strings.Join(results, "; ")
	}
	return check
}

// setupAppUserSudo configures sudo access for the app user
func setupAppUserSudo(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "App User Sudo Setup",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot setup sudo - root SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	// Create sudoers file for app user
	sudoersContent := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: /bin/systemctl, /usr/bin/systemctl", server.AppUsername)
	cmd := fmt.Sprintf("echo '%s' > /etc/sudoers.d/%s", sudoersContent, server.AppUsername)

	_, err = manager.ExecuteCommand(cmd)
	if err != nil {
		check.Status = "error"
		check.Message = "Failed to create sudoers configuration"
		check.Details = err.Error()
		check.Suggestions = []string{
			fmt.Sprintf("Manually create: /etc/sudoers.d/%s", server.AppUsername),
			fmt.Sprintf("Add line: %s", sudoersContent),
			"Ensure sudoers.d directory is included in main sudoers file",
		}
		return check
	}

	// Set proper permissions on sudoers file
	_, err = manager.ExecuteCommand(fmt.Sprintf("chmod 440 /etc/sudoers.d/%s", server.AppUsername))
	if err != nil {
		check.Status = "warning"
		check.Message = "Sudoers file created but permission setting failed"
		check.Details = err.Error()
	}

	// Test sudo access by switching to app user and testing
	testCmd := fmt.Sprintf("su - %s -c 'sudo -n systemctl --version'", server.AppUsername)
	_, err = manager.ExecuteCommand(testCmd)
	if err != nil {
		check.Status = "warning"
		check.Message = "Sudo configuration created but test failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Verify sudoers syntax is correct",
			"Check that sudo is installed",
			"Test manually after setup",
		}
		return check
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("Sudo access configured successfully for %s", server.AppUsername)
	check.Details = "Passwordless sudo for systemctl commands"
	return check
}

// testAppUserSSHFromExternal tests SSH connection as app user from external perspective
func testAppUserSSHFromExternal(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "App User SSH External Test",
	}

	// Try to connect as app user (this should work after setup)
	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "App user SSH connection failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"SSH keys may not be properly configured",
			"Check authorized_keys file",
			"Verify SSH service configuration",
		}
		return check
	}
	defer manager.Close()

	if err := manager.TestConnection(); err != nil {
		check.Status = "error"
		check.Message = "App user SSH connection test failed"
		check.Details = err.Error()
		return check
	}

	// Test sudo access from app user perspective
	_, err = manager.ExecuteCommand("sudo -n systemctl --version")
	if err != nil {
		check.Status = "warning"
		check.Message = "App user SSH works but sudo access failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Check sudoers configuration",
			"Verify sudo permissions",
		}
		return check
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("App user %s SSH and sudo access working correctly", server.AppUsername)
	return check
}

// prepareDeploymentDirectories ensures deployment directories exist with proper permissions
func prepareDeploymentDirectories(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "Deployment Directories Preparation",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot prepare directories - root SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	directories := []string{"/opt/pocketbase", "/var/log/pocketbase"}
	var results []string

	for _, dir := range directories {
		// Create directory if it doesn't exist
		cmd := fmt.Sprintf("mkdir -p %s", dir)
		_, err := manager.ExecuteCommand(cmd)
		if err != nil {
			check.Status = "error"
			check.Message = fmt.Sprintf("Failed to create directory %s", dir)
			check.Details = err.Error()
			return check
		}

		// Set ownership to app user
		cmd = fmt.Sprintf("chown -R %s:%s %s", server.AppUsername, server.AppUsername, dir)
		_, err = manager.ExecuteCommand(cmd)
		if err != nil {
			check.Status = "warning"
			check.Message = fmt.Sprintf("Directory %s created but ownership setting failed", dir)
			check.Details = err.Error()
		} else {
			results = append(results, fmt.Sprintf("Created and configured %s", dir))
		}
	}

	if len(results) == len(directories) {
		check.Status = "success"
		check.Message = "All deployment directories prepared successfully"
		check.Details = strings.Join(results, "; ")
	} else {
		check.Status = "warning"
		check.Message = "Some deployment directories may have issues"
	}

	return check
}

// checkSecurityLockdownReadiness performs a final readiness check
func checkSecurityLockdownReadiness(server *models.Server) PreSecurityCheck {
	check := PreSecurityCheck{
		Name: "Security Lockdown Readiness",
	}

	// This is a meta-check that summarizes readiness based on previous checks
	// In a real implementation, you might want to run quick verification tests

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "Not ready for security lockdown - app user SSH failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Fix app user SSH access first",
			"Ensure all previous checks pass",
		}
		return check
	}
	defer manager.Close()

	// Quick verification tests
	tests := []struct {
		name string
		cmd  string
	}{
		{"Basic connectivity", "echo 'test'"},
		{"Sudo access", "sudo -n systemctl --version"},
		{"Directory access", "ls /opt/pocketbase"},
	}

	for _, test := range tests {
		_, err := manager.ExecuteCommand(test.cmd)
		if err != nil {
			check.Status = "error"
			check.Message = fmt.Sprintf("Not ready for lockdown - %s test failed", test.name)
			check.Details = err.Error()
			check.Suggestions = []string{
				"Fix the failing test before proceeding with security lockdown",
				"Review previous check results",
			}
			return check
		}
	}

	check.Status = "success"
	check.Message = "Server is ready for security lockdown"
	check.Details = "All prerequisite checks passed successfully"
	check.Suggestions = []string{
		"You can now proceed with security lockdown",
		"After lockdown, root SSH access will be disabled",
		fmt.Sprintf("Ensure you can connect as %s before lockdown", server.AppUsername),
	}
	return check
}

// RunPreSecurityDiagnostics runs comprehensive pre-security lockdown diagnostics and setup
func RunPreSecurityDiagnostics(server *models.Server, autoSetup bool) PreSecurityResult {
	result := PreSecurityResult{
		Checks: make([]PreSecurityCheck, 0),
	}

	fmt.Printf("========================================\n")
	fmt.Printf("Pre-Security Lockdown Preparation\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Host: %s:%d\n", server.Host, server.Port)
	fmt.Printf("App User: %s\n", server.AppUsername)
	fmt.Printf("Root User: %s\n", server.RootUsername)
	fmt.Printf("Auto-setup: %v\n", autoSetup)
	fmt.Printf("========================================\n\n")

	// Run diagnostic and setup checks
	checks := []func(*models.Server) PreSecurityCheck{
		testRootSSHAccess,
		checkAppUserExists,
	}

	// If auto-setup is enabled, include setup functions
	if autoSetup {
		checks = append(checks,
			setupAppUserSSHKeys,
			setupAppUserSudo,
			prepareDeploymentDirectories,
		)
	}

	// Always run these tests
	checks = append(checks,
		testAppUserSSHFromExternal,
		checkSecurityLockdownReadiness,
	)

	for _, checkFunc := range checks {
		check := checkFunc(server)
		result.Checks = append(result.Checks, check)

		// Print check result
		fmt.Printf("%s\n", formatStatusMessage(check.Status, check.Message))
		if check.Details != "" {
			fmt.Printf("  Details: %s\n", check.Details)
		}
		if len(check.Suggestions) > 0 {
			fmt.Printf("  Suggestions:\n")
			for _, suggestion := range check.Suggestions {
				fmt.Printf("    • %s\n", suggestion)
			}
		}
		fmt.Println()
	}

	// Determine overall result
	hasErrors := false
	hasWarnings := false
	for _, check := range result.Checks {
		if check.Status == "error" {
			hasErrors = true
		}
		if check.Status == "warning" {
			hasWarnings = true
		}
	}

	// Generate summary
	fmt.Printf("========================================\n")
	if !hasErrors {
		result.ReadyForLockdown = true
		if hasWarnings {
			result.Summary = "Ready for security lockdown with some warnings"
			fmt.Printf("%s\n", formatStatusMessage("warning", result.Summary))
		} else {
			result.Summary = "Server is fully prepared for security lockdown!"
			fmt.Printf("%s\n", formatStatusMessage("success", result.Summary))
		}

		fmt.Printf("\n%s\n", formatStatusMessage("info", "Next steps:"))
		fmt.Printf("  • Review all configurations above\n")
		fmt.Printf("  • Test app user access one more time\n")
		fmt.Printf("  • Proceed with security lockdown when ready\n")
		fmt.Printf("  • After lockdown, root SSH access will be disabled\n")

	} else {
		result.ReadyForLockdown = false
		result.Summary = "Server is NOT ready for security lockdown"
		fmt.Printf("%s\n", formatStatusMessage("error", result.Summary))

		fmt.Printf("\n%s\n", formatStatusMessage("info", "Required actions:"))
		fmt.Printf("  • Fix all error conditions above\n")
		fmt.Printf("  • Re-run diagnostics to verify fixes\n")
		fmt.Printf("  • Do not proceed with security lockdown until all checks pass\n")

		if !autoSetup {
			fmt.Printf("  • Consider using -setup flag for automatic configuration\n")
		}
	}
	fmt.Printf("========================================\n")

	return result
}

// GetPreSecuritySummary returns a summary string of pre-security diagnostics
func GetPreSecuritySummary(server *models.Server) (string, error) {
	var summary strings.Builder

	summary.WriteString("Pre-Security Lockdown Readiness Summary\n")
	summary.WriteString("======================================\n\n")

	// Quick checks without full diagnostics output
	checks := []struct {
		name string
		test func() (bool, string)
	}{
		{"Root SSH Access", func() (bool, string) {
			check := testRootSSHAccess(server)
			return check.Status == "success", check.Message
		}},
		{"App User Exists", func() (bool, string) {
			check := checkAppUserExists(server)
			return check.Status == "success", check.Message
		}},
		{"App User SSH", func() (bool, string) {
			check := testAppUserSSHFromExternal(server)
			return check.Status == "success", check.Message
		}},
		{"Lockdown Readiness", func() (bool, string) {
			check := checkSecurityLockdownReadiness(server)
			return check.Status == "success", check.Message
		}},
	}

	allPassed := true
	for _, check := range checks {
		passed, message := check.test()
		icon := "✅"
		if !passed {
			icon = "❌"
			allPassed = false
		}
		summary.WriteString(fmt.Sprintf("%s %s: %s\n", icon, check.name, message))
	}

	summary.WriteString("\n")
	if allPassed {
		summary.WriteString("✅ Server is ready for security lockdown!\n")
		summary.WriteString("🔒 You can safely proceed with disabling root SSH access.\n")
	} else {
		summary.WriteString("❌ Server is NOT ready for security lockdown.\n")
		summary.WriteString("💡 Run with -setup to automatically configure prerequisites.\n")
	}

	return summary.String(), nil
}
