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
