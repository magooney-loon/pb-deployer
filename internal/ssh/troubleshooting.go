package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"pb-deployer/internal/models"
)

// ConnectionDiagnostic represents the result of a connection diagnostic
type ConnectionDiagnostic struct {
	Step       string `json:"step"`
	Status     string `json:"status"` // success/warning/error
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

// TroubleshootConnection performs comprehensive SSH connection troubleshooting
func TroubleshootConnection(server *models.Server, asRoot bool) ([]ConnectionDiagnostic, error) {
	var diagnostics []ConnectionDiagnostic

	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	// Step 1: Basic connectivity test
	diagnostics = append(diagnostics, testNetworkConnectivity(server))

	// Step 2: SSH service availability
	diagnostics = append(diagnostics, testSSHService(server))

	// Step 3: Authentication methods check
	authDiag := testAuthenticationMethods(server, username)
	diagnostics = append(diagnostics, authDiag...)

	// Step 4: Host key analysis
	diagnostics = append(diagnostics, analyzeHostKey(server))

	// Step 5: SSH client configuration
	diagnostics = append(diagnostics, analyzeSSHClientConfig())

	// Step 6: Permission checks
	permDiag := checkSSHPermissions()
	diagnostics = append(diagnostics, permDiag...)

	return diagnostics, nil
}

// testNetworkConnectivity tests basic network connectivity to the server
func testNetworkConnectivity(server *models.Server) ConnectionDiagnostic {
	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "network_connectivity",
			Status:     "error",
			Message:    fmt.Sprintf("Cannot connect to %s", address),
			Details:    err.Error(),
			Suggestion: "Check if the server is running and the port is correct. Verify firewall settings.",
		}
	}
	defer conn.Close()

	return ConnectionDiagnostic{
		Step:    "network_connectivity",
		Status:  "success",
		Message: fmt.Sprintf("Successfully connected to %s", address),
	}
}

// testSSHService tests if SSH service is responding
func testSSHService(server *models.Server) ConnectionDiagnostic {
	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	// Try to get SSH version banner
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "error",
			Message:    "SSH service is not responding",
			Details:    err.Error(),
			Suggestion: "Check if SSH daemon (sshd) is running on the server.",
		}
	}
	defer conn.Close()

	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Read SSH banner
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "warning",
			Message:    "Connected but no SSH banner received",
			Details:    err.Error(),
			Suggestion: "The service might not be SSH, or there could be a network issue.",
		}
	}

	banner := strings.TrimSpace(string(buffer[:n]))
	if !strings.HasPrefix(banner, "SSH-") {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "warning",
			Message:    "Service is not responding with SSH banner",
			Details:    fmt.Sprintf("Received: %s", banner),
			Suggestion: "Verify that SSH daemon is running on the specified port.",
		}
	}

	return ConnectionDiagnostic{
		Step:    "ssh_service",
		Status:  "success",
		Message: fmt.Sprintf("SSH service is responding: %s", banner),
	}
}

// testAuthenticationMethods checks available authentication methods
func testAuthenticationMethods(server *models.Server, username string) []ConnectionDiagnostic {
	var diagnostics []ConnectionDiagnostic

	// Check SSH agent
	if server.UseSSHAgent {
		diag := checkSSHAgent()
		diagnostics = append(diagnostics, diag)
	}

	// Check manual key
	if server.ManualKeyPath != "" {
		diag := checkPrivateKey(server.ManualKeyPath)
		diagnostics = append(diagnostics, diag)
	}

	// Check default keys
	defaultKeys := []string{
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ecdsa"),
	}

	foundKeys := 0
	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			foundKeys++
			diag := checkPrivateKey(keyPath)
			diagnostics = append(diagnostics, diag)
		}
	}

	if foundKeys == 0 && server.ManualKeyPath == "" && !server.UseSSHAgent {
		diagnostics = append(diagnostics, ConnectionDiagnostic{
			Step:       "authentication_methods",
			Status:     "error",
			Message:    "No authentication methods available",
			Suggestion: "Set up SSH keys or enable SSH agent authentication.",
		})
	}

	return diagnostics
}

// checkSSHAgent checks if SSH agent is available and has keys
func checkSSHAgent() ConnectionDiagnostic {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "warning",
			Message:    "SSH_AUTH_SOCK environment variable not set",
			Suggestion: "Start SSH agent or disable SSH agent authentication.",
		}
	}

	// Try to connect to SSH agent
	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "error",
			Message:    "Cannot connect to SSH agent",
			Details:    err.Error(),
			Suggestion: "Restart SSH agent or check SSH_AUTH_SOCK path.",
		}
	}
	defer conn.Close()

	return ConnectionDiagnostic{
		Step:    "ssh_agent",
		Status:  "success",
		Message: "SSH agent is available",
	}
}

// checkPrivateKey validates a private key file
func checkPrivateKey(keyPath string) ConnectionDiagnostic {
	// Check if file exists
	info, err := os.Stat(keyPath)
	if err != nil {
		return ConnectionDiagnostic{
			Step:    "private_key",
			Status:  "warning",
			Message: fmt.Sprintf("Key file not found: %s", keyPath),
			Details: err.Error(),
		}
	}

	// Check file permissions
	mode := info.Mode()
	if mode&0077 != 0 {
		return ConnectionDiagnostic{
			Step:       "private_key",
			Status:     "warning",
			Message:    fmt.Sprintf("Key file has unsafe permissions: %s", keyPath),
			Details:    fmt.Sprintf("Current permissions: %v", mode),
			Suggestion: fmt.Sprintf("Run: chmod 600 %s", keyPath),
		}
	}

	// Try to parse the key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return ConnectionDiagnostic{
			Step:    "private_key",
			Status:  "error",
			Message: fmt.Sprintf("Cannot read key file: %s", keyPath),
			Details: err.Error(),
		}
	}

	_, err = ssh.ParsePrivateKey(keyData)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "private_key",
			Status:     "error",
			Message:    fmt.Sprintf("Invalid private key: %s", keyPath),
			Details:    err.Error(),
			Suggestion: "Check if the key file is corrupted or password-protected.",
		}
	}

	return ConnectionDiagnostic{
		Step:    "private_key",
		Status:  "success",
		Message: fmt.Sprintf("Valid private key found: %s", keyPath),
	}
}

// analyzeHostKey analyzes host key configuration and issues
func analyzeHostKey(server *models.Server) ConnectionDiagnostic {
	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")

	// Check if known_hosts exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return ConnectionDiagnostic{
			Step:       "host_key",
			Status:     "warning",
			Message:    "No known_hosts file found",
			Details:    fmt.Sprintf("Expected path: %s", knownHostsPath),
			Suggestion: "This is normal for first-time connections. The host key will be automatically accepted and stored.",
		}
	}

	// Check if host is in known_hosts
	hostname := server.Host
	if isHostInKnownHosts(knownHostsPath, hostname) {
		return ConnectionDiagnostic{
			Step:    "host_key",
			Status:  "success",
			Message: fmt.Sprintf("Host %s found in known_hosts", hostname),
		}
	}

	return ConnectionDiagnostic{
		Step:       "host_key",
		Status:     "warning",
		Message:    fmt.Sprintf("Host %s not found in known_hosts", hostname),
		Suggestion: "The host key will be automatically accepted and stored on first connection.",
	}
}

// analyzeSSHClientConfig checks SSH client configuration
func analyzeSSHClientConfig() ConnectionDiagnostic {
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")

	// Check if .ssh directory exists
	info, err := os.Stat(sshDir)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_config",
			Status:     "warning",
			Message:    ".ssh directory not found",
			Details:    fmt.Sprintf("Expected path: %s", sshDir),
			Suggestion: fmt.Sprintf("Create directory: mkdir -p %s && chmod 700 %s", sshDir, sshDir),
		}
	}

	// Check directory permissions
	mode := info.Mode()
	if mode&0077 != 0 {
		return ConnectionDiagnostic{
			Step:       "ssh_config",
			Status:     "warning",
			Message:    ".ssh directory has unsafe permissions",
			Details:    fmt.Sprintf("Current permissions: %v", mode),
			Suggestion: fmt.Sprintf("Fix permissions: chmod 700 %s", sshDir),
		}
	}

	return ConnectionDiagnostic{
		Step:    "ssh_config",
		Status:  "success",
		Message: ".ssh directory is properly configured",
	}
}

// checkSSHPermissions checks SSH-related file permissions
func checkSSHPermissions() []ConnectionDiagnostic {
	var diagnostics []ConnectionDiagnostic

	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")

	// Files to check with their expected permissions
	filesToCheck := map[string]os.FileMode{
		filepath.Join(sshDir, "id_rsa"):          0600,
		filepath.Join(sshDir, "id_ed25519"):      0600,
		filepath.Join(sshDir, "id_ecdsa"):        0600,
		filepath.Join(sshDir, "known_hosts"):     0644,
		filepath.Join(sshDir, "authorized_keys"): 0600,
	}

	for filePath, expectedMode := range filesToCheck {
		info, err := os.Stat(filePath)
		if err != nil {
			// File doesn't exist, which is OK
			continue
		}

		actualMode := info.Mode() & 0777
		if actualMode != expectedMode {
			diagnostics = append(diagnostics, ConnectionDiagnostic{
				Step:       "file_permissions",
				Status:     "warning",
				Message:    fmt.Sprintf("Incorrect permissions for %s", filepath.Base(filePath)),
				Details:    fmt.Sprintf("Current: %v, Expected: %v", actualMode, expectedMode),
				Suggestion: fmt.Sprintf("Run: chmod %o %s", expectedMode, filePath),
			})
		}
	}

	if len(diagnostics) == 0 {
		diagnostics = append(diagnostics, ConnectionDiagnostic{
			Step:    "file_permissions",
			Status:  "success",
			Message: "All SSH file permissions are correct",
		})
	}

	return diagnostics
}

// isHostInKnownHosts checks if a hostname exists in known_hosts file
func isHostInKnownHosts(knownHostsPath, hostname string) bool {
	file, err := os.Open(knownHostsPath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Simple check - just look for the hostname
	// This could be improved to parse the file more thoroughly
	content := make([]byte, 4096)
	n, err := file.Read(content)
	if err != nil && n == 0 {
		return false
	}

	return strings.Contains(string(content[:n]), hostname)
}

// FixCommonIssues attempts to automatically fix common SSH issues
func FixCommonIssues(server *models.Server) []ConnectionDiagnostic {
	var results []ConnectionDiagnostic

	// Fix .ssh directory permissions
	sshDir := filepath.Join(os.Getenv("HOME"), ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		results = append(results, ConnectionDiagnostic{
			Step:    "fix_ssh_dir",
			Status:  "error",
			Message: "Failed to create/fix .ssh directory",
			Details: err.Error(),
		})
	} else {
		results = append(results, ConnectionDiagnostic{
			Step:    "fix_ssh_dir",
			Status:  "success",
			Message: ".ssh directory created/fixed",
		})
	}

	// Pre-accept host key
	if err := AcceptHostKey(server); err != nil {
		results = append(results, ConnectionDiagnostic{
			Step:    "accept_host_key",
			Status:  "warning",
			Message: "Could not pre-accept host key",
			Details: err.Error(),
		})
	} else {
		results = append(results, ConnectionDiagnostic{
			Step:    "accept_host_key",
			Status:  "success",
			Message: "Host key pre-accepted and stored",
		})
	}

	return results
}

// GetConnectionSummary provides a summary of connection readiness
func GetConnectionSummary(server *models.Server, asRoot bool) (string, error) {
	diagnostics, err := TroubleshootConnection(server, asRoot)
	if err != nil {
		return "", err
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("SSH Connection Summary for %s:%d\n", server.Host, server.Port))
	summary.WriteString(strings.Repeat("=", 50) + "\n\n")

	errorCount := 0
	warningCount := 0
	successCount := 0

	for _, diag := range diagnostics {
		status := "✓"
		switch diag.Status {
		case "error":
			status = "✗"
			errorCount++
		case "warning":
			status = "⚠"
			warningCount++
		case "success":
			successCount++
		}

		summary.WriteString(fmt.Sprintf("%s %s: %s\n", status, diag.Step, diag.Message))
		if diag.Details != "" {
			summary.WriteString(fmt.Sprintf("   Details: %s\n", diag.Details))
		}
		if diag.Suggestion != "" {
			summary.WriteString(fmt.Sprintf("   Suggestion: %s\n", diag.Suggestion))
		}
		summary.WriteString("\n")
	}

	summary.WriteString(fmt.Sprintf("Summary: %d successful, %d warnings, %d errors\n", successCount, warningCount, errorCount))

	if errorCount > 0 {
		summary.WriteString("\n⚠️  Connection may fail due to errors above.\n")
	} else if warningCount > 0 {
		summary.WriteString("\n✓ Connection should work, but consider addressing warnings.\n")
	} else {
		summary.WriteString("\n✓ All checks passed! Connection should work smoothly.\n")
	}

	return summary.String(), nil
}

// DiagnoseAppUserPostSecurity performs specialized diagnostics for app user SSH after security lockdown
func DiagnoseAppUserPostSecurity(server *models.Server) ([]ConnectionDiagnostic, error) {
	var diagnostics []ConnectionDiagnostic

	// Test app user SSH connection specifically
	diagnostics = append(diagnostics, diagnoseAppUserConnection(server))

	// Check sudo configuration
	diagnostics = append(diagnostics, checkAppUserSudoAccess(server))

	// Check SSH key setup for app user
	diagnostics = append(diagnostics, checkAppUserSSHKeys(server))

	// Verify security lockdown didn't break app user access
	diagnostics = append(diagnostics, verifyPostSecurityAccess(server))

	// Check SSH daemon configuration for app user restrictions
	diagnostics = append(diagnostics, checkSSHDaemonConfig(server))

	return diagnostics, nil
}

// diagnoseAppUserConnection tests app user SSH connection with detailed diagnostics
func diagnoseAppUserConnection(server *models.Server) ConnectionDiagnostic {
	// Try to connect as app user
	manager, err := NewSSHManager(server, false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "app_user_connection",
			Status:     "error",
			Message:    fmt.Sprintf("Failed to connect as %s", server.AppUsername),
			Details:    err.Error(),
			Suggestion: "Check SSH key configuration and network connectivity. Try running: ssh-test -host " + server.Host + " -diagnose",
		}
	}
	defer manager.Close()

	// Test basic command execution
	if err := manager.TestConnection(); err != nil {
		return ConnectionDiagnostic{
			Step:       "app_user_connection",
			Status:     "error",
			Message:    fmt.Sprintf("App user %s can connect but command execution fails", server.AppUsername),
			Details:    err.Error(),
			Suggestion: "Check shell configuration and command permissions for the app user.",
		}
	}

	return ConnectionDiagnostic{
		Step:    "app_user_connection",
		Status:  "success",
		Message: fmt.Sprintf("App user %s connection is working", server.AppUsername),
	}
}

// checkAppUserSudoAccess verifies sudo configuration for app user
func checkAppUserSudoAccess(server *models.Server) ConnectionDiagnostic {
	manager, err := NewSSHManager(server, false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sudo_access",
			Status:     "error",
			Message:    "Cannot test sudo access - SSH connection failed",
			Details:    err.Error(),
			Suggestion: "Fix SSH connection first before testing sudo access.",
		}
	}
	defer manager.Close()

	// Test sudo access for common deployment commands
	testCommands := []string{
		"sudo -n systemctl --version",
		"sudo -n mkdir -p /tmp/sudo_test && sudo -n rmdir /tmp/sudo_test",
	}

	for _, cmd := range testCommands {
		if _, err := manager.ExecuteCommand(cmd); err != nil {
			return ConnectionDiagnostic{
				Step:       "sudo_access",
				Status:     "error",
				Message:    fmt.Sprintf("Sudo access test failed for %s", server.AppUsername),
				Details:    fmt.Sprintf("Command '%s' failed: %v", cmd, err),
				Suggestion: fmt.Sprintf("Configure passwordless sudo for %s. Check /etc/sudoers.d/%s file.", server.AppUsername, server.AppUsername),
			}
		}
	}

	return ConnectionDiagnostic{
		Step:    "sudo_access",
		Status:  "success",
		Message: fmt.Sprintf("Sudo access is properly configured for %s", server.AppUsername),
	}
}

// checkAppUserSSHKeys verifies SSH key configuration for app user
func checkAppUserSSHKeys(server *models.Server) ConnectionDiagnostic {
	manager, err := NewSSHManager(server, false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_keys",
			Status:     "error",
			Message:    "Cannot check SSH keys - connection failed",
			Details:    err.Error(),
			Suggestion: "SSH key configuration may be incorrect. Check authorized_keys file.",
		}
	}
	defer manager.Close()

	// Check if authorized_keys file exists and has content
	authKeysPath := fmt.Sprintf("/home/%s/.ssh/authorized_keys", server.AppUsername)
	checkCmd := fmt.Sprintf("test -f %s && test -s %s", authKeysPath, authKeysPath)
	if _, err := manager.ExecuteCommand(checkCmd); err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_keys",
			Status:     "error",
			Message:    fmt.Sprintf("SSH authorized_keys file missing or empty for %s", server.AppUsername),
			Details:    fmt.Sprintf("File %s does not exist or is empty", authKeysPath),
			Suggestion: fmt.Sprintf("Ensure %s exists and contains valid SSH public keys", authKeysPath),
		}
	}

	// Check file permissions
	permCmd := fmt.Sprintf("stat -c '%%a' %s", authKeysPath)
	if output, err := manager.ExecuteCommand(permCmd); err == nil {
		perms := strings.TrimSpace(output)
		if perms != "600" {
			return ConnectionDiagnostic{
				Step:       "ssh_keys",
				Status:     "warning",
				Message:    "SSH authorized_keys has incorrect permissions",
				Details:    fmt.Sprintf("Current permissions: %s, Expected: 600", perms),
				Suggestion: fmt.Sprintf("Run: chmod 600 %s", authKeysPath),
			}
		}
	}

	return ConnectionDiagnostic{
		Step:    "ssh_keys",
		Status:  "success",
		Message: fmt.Sprintf("SSH keys are properly configured for %s", server.AppUsername),
	}
}

// verifyPostSecurityAccess checks if app user can perform post-security operations
func verifyPostSecurityAccess(server *models.Server) ConnectionDiagnostic {
	manager, err := NewSSHManager(server, false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "post_security_access",
			Status:     "error",
			Message:    "Cannot verify post-security access - SSH connection failed",
			Details:    err.Error(),
			Suggestion: "Fix SSH connection issues first.",
		}
	}
	defer manager.Close()

	// Test deployment-related operations
	testOperations := []struct {
		name string
		cmd  string
	}{
		{"directory_access", "ls -la /opt/pocketbase"},
		{"service_management", "sudo -n systemctl status ssh"},
		{"file_operations", "sudo -n touch /tmp/deploy_test && sudo -n rm /tmp/deploy_test"},
	}

	for _, op := range testOperations {
		if _, err := manager.ExecuteCommand(op.cmd); err != nil {
			return ConnectionDiagnostic{
				Step:       "post_security_access",
				Status:     "error",
				Message:    fmt.Sprintf("Post-security access test failed: %s", op.name),
				Details:    fmt.Sprintf("Command '%s' failed: %v", op.cmd, err),
				Suggestion: "App user lacks necessary permissions for deployment operations. Check sudo configuration and file permissions.",
			}
		}
	}

	return ConnectionDiagnostic{
		Step:    "post_security_access",
		Status:  "success",
		Message: "App user has necessary access for post-security operations",
	}
}

// checkSSHDaemonConfig checks SSH daemon configuration that might affect app user
func checkSSHDaemonConfig(server *models.Server) ConnectionDiagnostic {
	manager, err := NewSSHManager(server, false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sshd_config",
			Status:     "error",
			Message:    "Cannot check SSH daemon config - SSH connection failed",
			Details:    err.Error(),
			Suggestion: "Fix SSH connection issues first.",
		}
	}
	defer manager.Close()

	// Check key SSH settings that might affect app user access
	criticalSettings := []struct {
		setting string
		check   string
	}{
		{"PubkeyAuthentication", "grep -q '^PubkeyAuthentication yes' /etc/ssh/sshd_config"},
		{"PasswordAuthentication", "grep -q '^PasswordAuthentication no' /etc/ssh/sshd_config"},
		{"PermitRootLogin", "grep -q '^PermitRootLogin no' /etc/ssh/sshd_config"},
	}

	var warnings []string
	for _, setting := range criticalSettings {
		if _, err := manager.ExecuteCommand(fmt.Sprintf("sudo %s", setting.check)); err != nil {
			warnings = append(warnings, fmt.Sprintf("%s setting may be incorrect", setting.setting))
		}
	}

	if len(warnings) > 0 {
		return ConnectionDiagnostic{
			Step:       "sshd_config",
			Status:     "warning",
			Message:    "SSH daemon configuration has potential issues",
			Details:    strings.Join(warnings, "; "),
			Suggestion: "Review /etc/ssh/sshd_config for security hardening settings that might affect access.",
		}
	}

	return ConnectionDiagnostic{
		Step:    "sshd_config",
		Status:  "success",
		Message: "SSH daemon configuration appears correct",
	}
}

// GetPostSecurityTroubleshootingSummary provides a summary for post-security issues
func GetPostSecurityTroubleshootingSummary(server *models.Server) (string, error) {
	diagnostics, err := DiagnoseAppUserPostSecurity(server)
	if err != nil {
		return "", err
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Post-Security SSH Diagnostics for %s:%d\n", server.Host, server.Port))
	summary.WriteString(strings.Repeat("=", 60) + "\n\n")

	errorCount := 0
	warningCount := 0
	successCount := 0

	for _, diag := range diagnostics {
		status := "✓"
		switch diag.Status {
		case "error":
			status = "✗"
			errorCount++
		case "warning":
			status = "⚠"
			warningCount++
		case "success":
			successCount++
		}

		summary.WriteString(fmt.Sprintf("%s %s: %s\n", status, diag.Step, diag.Message))
		if diag.Details != "" {
			summary.WriteString(fmt.Sprintf("   Details: %s\n", diag.Details))
		}
		if diag.Suggestion != "" {
			summary.WriteString(fmt.Sprintf("   Suggestion: %s\n", diag.Suggestion))
		}
		summary.WriteString("\n")
	}

	summary.WriteString(fmt.Sprintf("Summary: %d successful, %d warnings, %d errors\n", successCount, warningCount, errorCount))

	if errorCount > 0 {
		summary.WriteString("\n⚠️  Critical issues found that prevent proper app user access.\n")
		summary.WriteString("🔧 Run the following to attempt automatic fixes:\n")
		summary.WriteString("   pb-deployer fix-ssh-access --server-id <id>\n")
	} else if warningCount > 0 {
		summary.WriteString("\n✓ App user access is working, but consider addressing warnings.\n")
	} else {
		summary.WriteString("\n✅ All post-security checks passed! App user access is fully functional.\n")
	}

	return summary.String(), nil
}
