package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"pb-deployer/internal/models"
	sshpkg "pb-deployer/internal/ssh"
)

// PostSecurityCheck represents a single diagnostic check
type PostSecurityCheck struct {
	Name        string
	Status      string // "success", "error", "warning", "info"
	Message     string
	Details     string
	Suggestions []string
}

// PostSecurityResult contains all diagnostic results
type PostSecurityResult struct {
	Checks      []PostSecurityCheck
	Summary     string
	OverallPass bool
	Suggestions []string
}

// Colors for output
const (
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[1;33m"
	ColorBlue   = "\033[0;34m"
	ColorReset  = "\033[0m"
)

// getStatusIcon returns the appropriate icon for a status
func getStatusIcon(status string) string {
	switch status {
	case "success":
		return "✅"
	case "error":
		return "❌"
	case "warning":
		return "⚠️"
	case "info":
		return "ℹ️"
	default:
		return "•"
	}
}

// formatStatusMessage formats a status message with color and icon
func formatStatusMessage(status, message string) string {
	icon := getStatusIcon(status)
	color := ColorReset

	switch status {
	case "success":
		color = ColorGreen
	case "error":
		color = ColorRed
	case "warning":
		color = ColorYellow
	case "info":
		color = ColorBlue
	}

	return fmt.Sprintf("%s%s %s%s", color, icon, message, ColorReset)
}

// testTCPConnectivity tests basic TCP connectivity to the server
func testTCPConnectivity(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "TCP Connectivity",
	}

	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		check.Status = "error"
		check.Message = "TCP connection failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Check if the server is running",
			"Verify the IP address and port",
			"Check firewall settings",
			"Ensure SSH service is running on the server",
		}
		return check
	}
	conn.Close()

	check.Status = "success"
	check.Message = "TCP connection successful"
	return check
}

// testAppUserSSH tests SSH connection as the app user
func testAppUserSSH(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "App User SSH Connection",
	}

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "App user SSH connection failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			fmt.Sprintf("SSH keys not properly configured for %s", server.AppUsername),
			fmt.Sprintf("User %s does not exist", server.AppUsername),
			"SSH service configuration issues",
			"Check authorized_keys file permissions",
		}
		return check
	}
	defer manager.Close()

	if err := manager.TestConnection(); err != nil {
		check.Status = "error"
		check.Message = "App user SSH connection test failed"
		check.Details = err.Error()
		check.Suggestions = []string{
			"Check SSH key configuration",
			"Verify user permissions",
			"Check SSH service logs",
		}
		return check
	}

	check.Status = "success"
	check.Message = "App user SSH connection successful"
	return check
}

// testRootSSH tests root SSH connection (should fail on security-locked servers)
func testRootSSH(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "Root SSH Connection",
	}

	manager, err := sshpkg.NewSSHManager(server, true)
	if err != nil {
		if server.SecurityLocked {
			check.Status = "success"
			check.Message = "Root SSH connection failed (expected after security lockdown)"
			check.Details = "This is the correct behavior for a security-locked server"
		} else {
			check.Status = "error"
			check.Message = "Root SSH connection failed"
			check.Details = err.Error()
			check.Suggestions = []string{
				"Check root SSH key configuration",
				"Verify SSH service configuration",
			}
		}
		return check
	}
	defer manager.Close()

	if err := manager.TestConnection(); err != nil {
		if server.SecurityLocked {
			check.Status = "success"
			check.Message = "Root SSH connection failed (expected after security lockdown)"
		} else {
			check.Status = "error"
			check.Message = "Root SSH connection test failed"
			check.Details = err.Error()
		}
		return check
	}

	if server.SecurityLocked {
		check.Status = "warning"
		check.Message = "Root SSH connection successful (security lockdown may not be applied)"
		check.Details = "Root SSH access should be disabled after security lockdown"
		check.Suggestions = []string{
			"Consider disabling root SSH access",
			"Review security lockdown procedures",
		}
	} else {
		check.Status = "success"
		check.Message = "Root SSH connection successful"
	}

	return check
}

// testSudoAccess tests sudo access for the app user
func testSudoAccess(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "App User Sudo Access",
	}

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot test sudo access - SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	// Test passwordless sudo access
	_, err = manager.ExecuteCommand("sudo -n systemctl --version")
	if err != nil {
		check.Status = "error"
		check.Message = fmt.Sprintf("Sudo access failed for %s", server.AppUsername)
		check.Details = err.Error()
		check.Suggestions = []string{
			fmt.Sprintf("%s not in sudoers", server.AppUsername),
			"Sudo requires password (should be passwordless)",
			"Sudoers configuration missing",
			fmt.Sprintf("Add to sudoers: echo \"%s ALL=(ALL) NOPASSWD: /bin/systemctl, /usr/bin/systemctl\" > /etc/sudoers.d/%s", server.AppUsername, server.AppUsername),
		}
		return check
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("Sudo access is working for %s", server.AppUsername)
	return check
}

// checkSSHKeys checks SSH key configuration for the app user
func checkSSHKeys(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "SSH Key Configuration",
	}

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot check SSH keys - SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	// Check if authorized_keys exists and has content
	output, err := manager.ExecuteCommand("test -f ~/.ssh/authorized_keys && test -s ~/.ssh/authorized_keys && echo 'exists' || echo 'missing'")
	if err != nil {
		check.Status = "error"
		check.Message = "Failed to check authorized_keys file"
		check.Details = err.Error()
		return check
	}

	authKeysStatus := strings.TrimSpace(output)
	if authKeysStatus != "exists" {
		check.Status = "error"
		check.Message = fmt.Sprintf("SSH authorized_keys file missing or empty for %s", server.AppUsername)
		check.Suggestions = []string{
			"Copy SSH keys from root user",
			fmt.Sprintf("Add your public key to /home/%s/.ssh/authorized_keys", server.AppUsername),
			fmt.Sprintf("Run: cp /root/.ssh/authorized_keys /home/%s/.ssh/", server.AppUsername),
			fmt.Sprintf("Run: chown -R %s:%s /home/%s/.ssh", server.AppUsername, server.AppUsername, server.AppUsername),
		}
		return check
	}

	// Check permissions
	permsOutput, err := manager.ExecuteCommand("stat -c '%a' ~/.ssh/authorized_keys")
	if err != nil {
		check.Status = "warning"
		check.Message = "Could not check authorized_keys permissions"
		check.Details = err.Error()
		return check
	}

	perms := strings.TrimSpace(permsOutput)
	if perms != "600" {
		check.Status = "warning"
		check.Message = fmt.Sprintf("SSH authorized_keys permissions are %s (should be 600)", perms)
		check.Suggestions = []string{
			"Fix permissions: chmod 600 ~/.ssh/authorized_keys",
			"Fix SSH directory: chmod 700 ~/.ssh",
		}
		return check
	}

	check.Status = "success"
	check.Message = fmt.Sprintf("SSH authorized_keys file exists and has correct permissions for %s", server.AppUsername)
	return check
}

// checkDeploymentDirectories checks if deployment directories exist and are accessible
func checkDeploymentDirectories(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "Deployment Directories",
	}

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot check directories - SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	directories := []string{"/opt/pocketbase", "/var/log/pocketbase"}
	var issues []string
	var warnings []string

	for _, dir := range directories {
		// Check if directory exists
		output, err := manager.ExecuteCommand(fmt.Sprintf("test -d %s && echo 'exists' || echo 'missing'", dir))
		if err != nil {
			issues = append(issues, fmt.Sprintf("Failed to check directory %s: %v", dir, err))
			continue
		}

		dirStatus := strings.TrimSpace(output)
		if dirStatus != "exists" {
			issues = append(issues, fmt.Sprintf("Directory %s is missing", dir))
			continue
		}

		// Check if directory is accessible
		_, err = manager.ExecuteCommand(fmt.Sprintf("ls -la %s >/dev/null 2>&1", dir))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Directory %s exists but may have permission issues", dir))
		}
	}

	if len(issues) > 0 {
		check.Status = "error"
		check.Message = "Some deployment directories are missing"
		check.Details = strings.Join(issues, "; ")
		check.Suggestions = []string{
			"Create missing directories",
			"Set proper ownership and permissions",
		}
	} else if len(warnings) > 0 {
		check.Status = "warning"
		check.Message = "Deployment directories exist but may have permission issues"
		check.Details = strings.Join(warnings, "; ")
		check.Suggestions = []string{
			fmt.Sprintf("Check directory permissions for %s", server.AppUsername),
			"Ensure proper ownership is set",
		}
	} else {
		check.Status = "success"
		check.Message = "All deployment directories exist and are accessible"
	}

	return check
}

// fixSSHPermissions attempts to fix common SSH permission issues
func fixSSHPermissions(server *models.Server) PostSecurityCheck {
	check := PostSecurityCheck{
		Name: "SSH Permission Fix",
	}

	manager, err := sshpkg.NewSSHManager(server, false)
	if err != nil {
		check.Status = "error"
		check.Message = "Cannot fix SSH permissions - SSH connection failed"
		check.Details = err.Error()
		return check
	}
	defer manager.Close()

	commands := []string{
		"chmod 700 ~/.ssh 2>/dev/null || true",
		"chmod 600 ~/.ssh/authorized_keys 2>/dev/null || true",
	}

	var errors []string
	for _, cmd := range commands {
		_, err := manager.ExecuteCommand(cmd)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to run '%s': %v", cmd, err))
		}
	}

	if len(errors) > 0 {
		check.Status = "warning"
		check.Message = "Some permission fixes failed"
		check.Details = strings.Join(errors, "; ")
	} else {
		check.Status = "success"
		check.Message = "SSH permissions fixed successfully"
	}

	return check
}

// RunPostSecurityDiagnostics runs comprehensive post-security lockdown diagnostics
func RunPostSecurityDiagnostics(server *models.Server, autoFix bool) PostSecurityResult {
	result := PostSecurityResult{
		Checks: make([]PostSecurityCheck, 0),
	}

	fmt.Printf("========================================\n")
	fmt.Printf("Post-Security SSH Troubleshooting\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Host: %s:%d\n", server.Host, server.Port)
	fmt.Printf("App User: %s\n", server.AppUsername)
	fmt.Printf("Root User: %s\n", server.RootUsername)
	fmt.Printf("Security Locked: %v\n", server.SecurityLocked)
	fmt.Printf("Auto-fix: %v\n", autoFix)
	fmt.Printf("========================================\n\n")

	// Run all diagnostic checks
	checks := []func(*models.Server) PostSecurityCheck{
		testTCPConnectivity,
		testRootSSH,
		testAppUserSSH,
		checkSSHKeys,
		testSudoAccess,
		checkDeploymentDirectories,
	}

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

	// Run auto-fixes if requested
	if autoFix {
		fmt.Printf("Running automatic fixes...\n\n")
		fixCheck := fixSSHPermissions(server)
		result.Checks = append(result.Checks, fixCheck)
		fmt.Printf("%s\n", formatStatusMessage(fixCheck.Status, fixCheck.Message))
		if fixCheck.Details != "" {
			fmt.Printf("  Details: %s\n", fixCheck.Details)
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
		result.OverallPass = true
		if hasWarnings {
			result.Summary = "All critical checks passed with some warnings"
			fmt.Printf("%s\n", formatStatusMessage("warning", result.Summary))
		} else {
			result.Summary = "All checks passed! Post-security SSH access is working correctly."
			fmt.Printf("%s\n", formatStatusMessage("success", result.Summary))
		}
		fmt.Printf("\n%s\n", formatStatusMessage("info", fmt.Sprintf("Your server is ready for deployments using the app user (%s).", server.AppUsername)))
	} else {
		result.OverallPass = false
		result.Summary = "Some issues were found that need attention"
		fmt.Printf("%s\n", formatStatusMessage("error", result.Summary))

		// Provide general suggestions
		fmt.Printf("\n%s\n", formatStatusMessage("info", "General troubleshooting suggestions:"))
		generalSuggestions := []string{
			fmt.Sprintf("Check if %s exists: ssh root@%s 'id %s'", server.AppUsername, server.Host, server.AppUsername),
			fmt.Sprintf("Copy SSH keys: ssh root@%s 'cp /root/.ssh/authorized_keys /home/%s/.ssh/'", server.Host, server.AppUsername),
			fmt.Sprintf("Fix ownership: ssh root@%s 'chown -R %s:%s /home/%s/.ssh'", server.Host, server.AppUsername, server.AppUsername, server.AppUsername),
			fmt.Sprintf("Fix permissions: ssh root@%s 'chmod 700 /home/%s/.ssh && chmod 600 /home/%s/.ssh/authorized_keys'", server.Host, server.AppUsername, server.AppUsername),
		}

		for _, suggestion := range generalSuggestions {
			fmt.Printf("  • %s\n", suggestion)
		}

		fmt.Printf("\nManual verification commands:\n")
		verificationCommands := []string{
			fmt.Sprintf("ssh %s@%s 'whoami'", server.AppUsername, server.Host),
			fmt.Sprintf("ssh %s@%s 'sudo -n systemctl --version'", server.AppUsername, server.Host),
			fmt.Sprintf("ssh %s@%s 'ls -la /opt/pocketbase'", server.AppUsername, server.Host),
		}

		for _, cmd := range verificationCommands {
			fmt.Printf("  %s\n", cmd)
		}
	}
	fmt.Printf("========================================\n")

	return result
}

// GetPostSecuritySummary returns a summary string of post-security diagnostics
func GetPostSecuritySummary(server *models.Server) (string, error) {
	var summary strings.Builder

	summary.WriteString("Post-Security SSH Diagnostics Summary\n")
	summary.WriteString("=====================================\n\n")

	// Quick checks without full diagnostics output
	checks := []struct {
		name string
		test func() (bool, string)
	}{
		{"TCP Connectivity", func() (bool, string) {
			check := testTCPConnectivity(server)
			return check.Status == "success", check.Message
		}},
		{"App User SSH", func() (bool, string) {
			check := testAppUserSSH(server)
			return check.Status == "success", check.Message
		}},
		{"Root SSH (should fail)", func() (bool, string) {
			check := testRootSSH(server)
			// For root SSH on security-locked servers, failure is success
			isGood := check.Status == "success" || (server.SecurityLocked && check.Status == "error")
			return isGood, check.Message
		}},
		{"Sudo Access", func() (bool, string) {
			check := testSudoAccess(server)
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
		summary.WriteString("✅ Post-security SSH access is configured correctly!\n")
	} else {
		summary.WriteString("❌ Some post-security issues need attention.\n")
		summary.WriteString("💡 Run with -troubleshoot for detailed diagnostics.\n")
	}

	return summary.String(), nil
}
