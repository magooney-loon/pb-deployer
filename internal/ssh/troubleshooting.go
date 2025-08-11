package ssh

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"pb-deployer/internal/logger"
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

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": username,
		"as_root":  asRoot,
	}).Info("Starting comprehensive SSH connection troubleshooting")

	// Step 1: Basic connectivity test
	connectivityDiag := testNetworkConnectivity(server)
	diagnostics = append(diagnostics, connectivityDiag)

	// Step 1.5: If connectivity fails, check for fail2ban ban
	if connectivityDiag.Status == "error" {
		diagnostics = append(diagnostics, checkFail2banStatus(server))
	}

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

	logger.WithFields(map[string]interface{}{
		"host":             server.Host,
		"port":             server.Port,
		"username":         username,
		"diagnostic_count": len(diagnostics),
	}).Info("SSH connection troubleshooting completed")

	return diagnostics, nil
}

// testNetworkConnectivity tests basic network connectivity to the server
func testNetworkConnectivity(server *models.Server) ConnectionDiagnostic {
	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"address": address,
	}).Debug("Testing network connectivity")

	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"host":    server.Host,
			"port":    server.Port,
			"address": address,
		}).WithError(err).Warn("Network connectivity test failed")

		return ConnectionDiagnostic{
			Step:       "network_connectivity",
			Status:     "error",
			Message:    fmt.Sprintf("Cannot connect to %s", address),
			Details:    err.Error(),
			Suggestion: "Check if the server is running and the port is correct. Verify firewall settings.",
		}
	}
	defer conn.Close()

	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"address": address,
	}).Debug("Network connectivity test successful")

	return ConnectionDiagnostic{
		Step:    "network_connectivity",
		Status:  "success",
		Message: fmt.Sprintf("Successfully connected to %s", address),
	}
}

// testSSHService tests if SSH service is responding
func testSSHService(server *models.Server) ConnectionDiagnostic {
	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"address": address,
	}).Debug("Testing SSH service availability")

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
		logger.WithFields(map[string]interface{}{
			"host":            server.Host,
			"port":            server.Port,
			"received_banner": banner,
		}).Warn("Service not responding with SSH banner")

		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "warning",
			Message:    "Service is not responding with SSH banner",
			Details:    fmt.Sprintf("Received: %s", banner),
			Suggestion: "Verify that SSH daemon is running on the specified port.",
		}
	}

	logger.WithFields(map[string]interface{}{
		"host":   server.Host,
		"port":   server.Port,
		"banner": banner,
	}).Debug("SSH service responding correctly")

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

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Attempting to automatically fix common SSH issues")

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

	logger.WithFields(map[string]interface{}{
		"host":      server.Host,
		"port":      server.Port,
		"fix_count": len(results),
	}).Info("Auto-fix attempt completed")

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

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Info("Starting post-security SSH diagnostics for app user")

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

	logger.WithFields(map[string]interface{}{
		"host":             server.Host,
		"port":             server.Port,
		"username":         server.AppUsername,
		"diagnostic_count": len(diagnostics),
	}).Info("Post-security SSH diagnostics completed")

	return diagnostics, nil
}

// diagnoseAppUserConnection tests app user SSH connection with detailed diagnostics
func diagnoseAppUserConnection(server *models.Server) ConnectionDiagnostic {
	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Diagnosing app user SSH connection")

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
	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking app user sudo access")

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
	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking app user SSH keys configuration")

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
	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Verifying post-security access capabilities")

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
	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking SSH daemon configuration")

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
		summary.WriteString("\n✅ All checks passed! App user access is fully functional.\n")
	}

	return summary.String(), nil
}

// checkFail2banStatus checks if the current IP might be banned by fail2ban
func checkFail2banStatus(server *models.Server) ConnectionDiagnostic {
	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Checking fail2ban status for potential IP ban")

	// Get our current public IP
	currentIP, err := getCurrentPublicIP()
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "fail2ban_check",
			Status:     "warning",
			Message:    "Could not determine current public IP",
			Details:    err.Error(),
			Suggestion: "Unable to check if your IP is banned by fail2ban. If connection keeps failing, manually check server logs.",
		}
	}

	// Try to connect to a different server to check fail2ban status
	// Since we can't connect to the target server, we'll provide generic guidance
	return ConnectionDiagnostic{
		Step:       "fail2ban_check",
		Status:     "warning",
		Message:    fmt.Sprintf("Connection refused - possible fail2ban ban (your IP: %s)", currentIP),
		Details:    "Connection refused errors often indicate that fail2ban has banned your IP address after multiple failed connection attempts.",
		Suggestion: fmt.Sprintf("Check server logs: 'sudo fail2ban-client status sshd' and 'sudo fail2ban-client get sshd banip' on the server. To unban: 'sudo fail2ban-client set sshd unbanip %s'", currentIP),
	}
}

// getCurrentPublicIP attempts to determine the current public IP address
func getCurrentPublicIP() (string, error) {
	// Try multiple services to get public IP
	services := []string{
		"https://ipinfo.io/ip",
		"https://api.ipify.org",
		"https://checkip.amazonaws.com",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			ip := strings.TrimSpace(string(body))
			if net.ParseIP(ip) != nil {
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("could not determine public IP from any service")
}

// checkFail2banBanStatus checks if an IP is banned when we have access to the server
func checkFail2banBanStatus(manager *SSHManager, targetIP string) ConnectionDiagnostic {
	logger.WithFields(map[string]interface{}{
		"host":      manager.server.Host,
		"target_ip": targetIP,
	}).Debug("Checking fail2ban ban status for IP")

	// Check if fail2ban is running
	statusCmd := "sudo systemctl is-active fail2ban"
	output, err := manager.ExecuteCommand(statusCmd)
	if err != nil || !strings.Contains(output, "active") {
		return ConnectionDiagnostic{
			Step:       "fail2ban_status",
			Status:     "info",
			Message:    "fail2ban service is not running",
			Details:    "fail2ban is not active, so IP banning is not the issue",
			Suggestion: "Connection issues are not related to fail2ban. Check network connectivity and SSH service status.",
		}
	}

	// Check if IP is currently banned
	banCheckCmd := fmt.Sprintf("sudo fail2ban-client get sshd banip | grep -q '%s'", targetIP)
	_, err = manager.ExecuteCommand(banCheckCmd)
	if err == nil {
		// IP is banned
		return ConnectionDiagnostic{
			Step:       "fail2ban_ban_check",
			Status:     "error",
			Message:    fmt.Sprintf("IP %s is currently banned by fail2ban", targetIP),
			Details:    "Your IP address has been banned by fail2ban due to multiple failed connection attempts.",
			Suggestion: fmt.Sprintf("Unban your IP with: sudo fail2ban-client set sshd unbanip %s", targetIP),
		}
	}

	// Check recent fail2ban logs for this IP
	logCheckCmd := fmt.Sprintf("sudo journalctl -u fail2ban --since '1 hour ago' | grep '%s' | tail -5", targetIP)
	logOutput, err := manager.ExecuteCommand(logCheckCmd)
	if err == nil && strings.TrimSpace(logOutput) != "" {
		return ConnectionDiagnostic{
			Step:       "fail2ban_log_check",
			Status:     "warning",
			Message:    fmt.Sprintf("Recent fail2ban activity detected for IP %s", targetIP),
			Details:    fmt.Sprintf("Recent fail2ban logs:\n%s", logOutput),
			Suggestion: "IP may have been recently banned or unbanned. Check current ban status and consider reviewing authentication attempts.",
		}
	}

	return ConnectionDiagnostic{
		Step:    "fail2ban_ban_check",
		Status:  "success",
		Message: fmt.Sprintf("IP %s is not banned by fail2ban", targetIP),
		Details: "No active ban found for your IP address",
	}
}

// DiagnoseConnectionRefused provides specific guidance for "connection refused" errors
func DiagnoseConnectionRefused(server *models.Server) ConnectionDiagnostic {
	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Diagnosing connection refused error")

	// Get current public IP for fail2ban context
	currentIP, err := getCurrentPublicIP()
	if err != nil {
		currentIP = "unknown"
	}

	details := fmt.Sprintf(`Connection refused typically indicates one of these issues:

1. SSH service is not running on the server
2. Port %d is blocked by a firewall (UFW/iptables)
3. Your IP (%s) has been banned by fail2ban
4. The server is down or unreachable

Since this was working before and suddenly stopped, fail2ban ban is most likely.

To diagnose on the server (if you have alternative access):
• Check SSH service: sudo systemctl status ssh
• Check fail2ban status: sudo fail2ban-client status sshd
• Check if your IP is banned: sudo fail2ban-client get sshd banip | grep %s
• Check recent auth failures: sudo journalctl -u ssh --since "1 hour ago" | grep "Failed\|Invalid"

To fix fail2ban ban:
• Unban your IP: sudo fail2ban-client set sshd unbanip %s
• Restart fail2ban if needed: sudo systemctl restart fail2ban`,
		server.Port, currentIP, currentIP, currentIP)

	suggestion := fmt.Sprintf("If you have console access to the server, check: 1) SSH service status 2) fail2ban status 3) Whether IP %s is banned", currentIP)

	return ConnectionDiagnostic{
		Step:       "connection_refused_analysis",
		Status:     "error",
		Message:    fmt.Sprintf("Connection refused to %s:%d - likely fail2ban ban", server.Host, server.Port),
		Details:    details,
		Suggestion: suggestion,
	}
}

// DiagnoseConnectionRefusedImmediate provides immediate diagnostic for connection refused errors
// This is specifically for situations like the current one where connection suddenly stopped working
func DiagnoseConnectionRefusedImmediate(host string, port int) {
	fmt.Printf("🚨 CONNECTION REFUSED DIAGNOSTIC\n")
	fmt.Printf("================================\n\n")

	// Get current IP
	fmt.Printf("📍 Detecting your public IP...\n")
	currentIP, err := getCurrentPublicIP()
	if err != nil {
		fmt.Printf("⚠️  Could not determine IP: %v\n", err)
		currentIP = "unknown"
	} else {
		fmt.Printf("✓ Your current IP: %s\n\n", currentIP)
	}

	fmt.Printf("🎯 TARGET: %s:%d\n", host, port)
	fmt.Printf("🕒 ISSUE: Connection suddenly stopped working\n\n")

	fmt.Printf("🔥 MOST LIKELY CAUSE: FAIL2BAN IP BAN\n")
	fmt.Printf("=====================================\n")
	fmt.Printf("Your IP (%s) is likely banned by fail2ban because:\n", currentIP)
	fmt.Printf("• Multiple failed authentication attempts\n")
	fmt.Printf("• Dynamic IP changed and triggered security rules\n")
	fmt.Printf("• Automated security system detected suspicious activity\n\n")

	fmt.Printf("🛠️  IMMEDIATE SOLUTIONS\n")
	fmt.Printf("=======================\n\n")

	fmt.Printf("METHOD 1 - Console Access (Recommended):\n")
	fmt.Printf("1. Access server via console/VNC from hosting provider\n")
	fmt.Printf("2. Run: sudo fail2ban-client set sshd unbanip %s\n", currentIP)
	fmt.Printf("3. Verify: sudo fail2ban-client get sshd banip | grep %s\n", currentIP)
	fmt.Printf("4. Test connection again\n\n")

	fmt.Printf("METHOD 2 - Alternative IP:\n")
	fmt.Printf("1. Use mobile hotspot or VPN to get different IP\n")
	fmt.Printf("2. SSH from new IP: ssh user@%s\n", host)
	fmt.Printf("3. Unban original IP: sudo fail2ban-client set sshd unbanip %s\n", currentIP)
	fmt.Printf("4. Switch back to original connection\n\n")

	fmt.Printf("METHOD 3 - Wait it out:\n")
	fmt.Printf("1. fail2ban bans are usually temporary (default: 10 minutes)\n")
	fmt.Printf("2. Wait and try again later\n")
	fmt.Printf("3. Check ban time: sudo fail2ban-client get sshd bantime\n\n")

	fmt.Printf("🔍 VERIFICATION COMMANDS (run on server):\n")
	fmt.Printf("==========================================\n")
	fmt.Printf("• Check SSH service: sudo systemctl status ssh\n")
	fmt.Printf("• Check fail2ban: sudo systemctl status fail2ban\n")
	fmt.Printf("• List banned IPs: sudo fail2ban-client get sshd banip\n")
	fmt.Printf("• Recent failed logins: sudo journalctl -u ssh --since '1 hour ago' | grep Failed\n")
	fmt.Printf("• fail2ban logs: sudo journalctl -u fail2ban --since '1 hour ago'\n\n")

	fmt.Printf("📞 If this doesn't work, the issue might be:\n")
	fmt.Printf("• SSH service crashed: sudo systemctl restart ssh\n")
	fmt.Printf("• Firewall blocking: sudo ufw status\n")
	fmt.Printf("• Server is down: ping %s\n", host)
}

// QuickFail2banCheck performs a quick check for common fail2ban scenarios
func QuickFail2banCheck(host string, port int) error {
	currentIP, _ := getCurrentPublicIP()

	fmt.Printf("🔍 Quick fail2ban diagnostic for %s:%d\n", host, port)
	fmt.Printf("Your IP: %s\n\n", currentIP)

	// Test connectivity
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			fmt.Printf("❌ Connection refused - this is classic fail2ban behavior\n")
			fmt.Printf("💡 Solution: sudo fail2ban-client set sshd unbanip %s\n", currentIP)
			return fmt.Errorf("connection refused - likely fail2ban ban")
		} else if strings.Contains(err.Error(), "timeout") {
			fmt.Printf("⏱️  Connection timeout - could be network/firewall issue\n")
			return fmt.Errorf("connection timeout")
		} else {
			fmt.Printf("❓ Connection failed: %v\n", err)
			return err
		}
	}
	defer conn.Close()

	fmt.Printf("✅ Connection successful - SSH port is reachable\n")
	return nil
}
