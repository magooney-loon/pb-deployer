package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/models"
)

// ConnectionDiagnostic represents a single diagnostic check result
type ConnectionDiagnostic struct {
	Step       string            `json:"step"`
	Status     string            `json:"status"` // "success", "warning", "error"
	Message    string            `json:"message"`
	Details    string            `json:"details"`
	Suggestion string            `json:"suggestion"`
	Duration   time.Duration     `json:"duration"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DiagnosticContext provides context for diagnostic operations
type DiagnosticContext struct {
	server            *models.Server
	startTime         time.Time
	clientIP          string
	connectionManager *ConnectionManager
	ctx               context.Context
	cancel            context.CancelFunc
}

// NewDiagnosticContext creates a new diagnostic context
func NewDiagnosticContext(server *models.Server, clientIP string) *DiagnosticContext {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	return &DiagnosticContext{
		server:            server,
		startTime:         time.Now(),
		clientIP:          clientIP,
		connectionManager: GetConnectionManager(),
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Close cleans up the diagnostic context
func (dc *DiagnosticContext) Close() error {
	if dc.cancel != nil {
		dc.cancel()
	}
	return nil
}

// GetCachedConnection attempts to get a cached connection, creating one if needed
func (dc *DiagnosticContext) GetCachedConnection(asRoot bool) (*PooledConnection, error) {
	return dc.connectionManager.pool.GetOrCreateConnection(dc.server, asRoot)
}

// TroubleshootConnection performs comprehensive SSH connection diagnostics
func TroubleshootConnection(server *models.Server, clientIP string) ([]ConnectionDiagnostic, error) {
	ctx := NewDiagnosticContext(server, clientIP)
	defer ctx.Close()

	var diagnostics []ConnectionDiagnostic

	logger.WithFields(map[string]interface{}{
		"host":      server.Host,
		"port":      server.Port,
		"client_ip": clientIP,
	}).Info("Starting comprehensive SSH connection diagnostics")

	// Test network connectivity first
	diagnostics = append(diagnostics, testNetworkConnectivityWithContext(ctx))

	// Test SSH service availability
	diagnostics = append(diagnostics, testSSHServiceEnhanced(ctx))

	// Test SSH protocol negotiation
	diagnostics = append(diagnostics, testSSHProtocolNegotiation(ctx))

	// Test actual SSH connection with authentication
	connectionDiag := testActualSSHConnectionPooled(ctx)
	diagnostics = append(diagnostics, connectionDiag)

	// If basic connection fails, skip advanced tests
	if connectionDiag.Status == "error" {
		logger.WithFields(map[string]interface{}{
			"host": server.Host,
			"port": server.Port,
		}).Warn("Basic SSH connection failed, skipping advanced diagnostics")
		return diagnostics, nil
	}

	// Test authentication methods
	diagnostics = append(diagnostics, testAuthenticationMethodsEnhanced(ctx))

	// Check SSH agent
	diagnostics = append(diagnostics, checkSSHAgentEnhanced(ctx))

	// Check private key
	diagnostics = append(diagnostics, checkPrivateKey(ctx))

	// Analyze host key
	diagnostics = append(diagnostics, analyzeHostKey(ctx))

	// Check SSH client config
	diagnostics = append(diagnostics, analyzeSSHClientConfig(ctx))

	// Check SSH permissions
	diagnostics = append(diagnostics, checkSSHPermissions(ctx))

	// Check known hosts
	diagnostics = append(diagnostics, isHostInKnownHostsEnhanced(ctx))

	logger.WithFields(map[string]interface{}{
		"host":             server.Host,
		"port":             server.Port,
		"client_ip":        clientIP,
		"diagnostic_count": len(diagnostics),
		"duration":         time.Since(ctx.startTime).String(),
	}).Info("SSH connection diagnostics completed")

	return diagnostics, nil
}

// testNetworkConnectivity is a simple wrapper for backward compatibility
func testNetworkConnectivity(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return testNetworkConnectivityWithContext(ctx)
}

// testNetworkConnectivityWithContext tests basic network connectivity to the server
func testNetworkConnectivityWithContext(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Testing network connectivity")

	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)

	duration := time.Since(start)

	if err != nil {
		return ConnectionDiagnostic{
			Step:       "network_connectivity",
			Status:     "error",
			Message:    "Cannot reach server",
			Details:    fmt.Sprintf("Failed to connect to %s: %v", address, err),
			Suggestion: "Check if the server is running and accessible. Verify firewall settings.",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	conn.Close()

	return ConnectionDiagnostic{
		Step:      "network_connectivity",
		Status:    "success",
		Message:   fmt.Sprintf("Network connectivity to %s established", address),
		Duration:  duration,
		Timestamp: start,
	}
}

// testNetworkConnectivityEnhanced performs enhanced network connectivity tests
func testNetworkConnectivityEnhanced(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Testing enhanced network connectivity")

	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))

	// Test with different timeouts
	timeouts := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second}
	var lastErr error

	for _, timeout := range timeouts {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err == nil {
			conn.Close()
			return ConnectionDiagnostic{
				Step:      "network_connectivity_enhanced",
				Status:    "success",
				Message:   fmt.Sprintf("Network connectivity established (timeout: %v)", timeout),
				Duration:  time.Since(start),
				Timestamp: start,
				Metadata: map[string]string{
					"successful_timeout": timeout.String(),
				},
			}
		}
		lastErr = err
	}

	return ConnectionDiagnostic{
		Step:       "network_connectivity_enhanced",
		Status:     "error",
		Message:    "Network connectivity failed with all timeouts",
		Details:    fmt.Sprintf("Failed to connect to %s: %v", address, lastErr),
		Suggestion: "Check network configuration, firewall rules, and server availability.",
		Duration:   time.Since(start),
		Timestamp:  start,
	}
}

// testSSHService tests SSH service availability
func testSSHService(server *models.Server) ConnectionDiagnostic {
	start := time.Now()

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Testing SSH service availability")

	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := net.DialTimeout("tcp", address, 30*time.Second)

	duration := time.Since(start)

	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "error",
			Message:    "SSH service not responding",
			Details:    fmt.Sprintf("Failed to connect to SSH service at %s: %v", address, err),
			Suggestion: "Ensure SSH daemon is running: sudo systemctl status ssh",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	// Read SSH banner
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	conn.Close()

	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "warning",
			Message:    "SSH service responding but no banner received",
			Details:    fmt.Sprintf("Connection established but failed to read SSH banner: %v", err),
			Suggestion: "SSH service might be misconfigured. Check SSH daemon logs.",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	banner := strings.TrimSpace(string(buffer[:n]))
	if !strings.HasPrefix(banner, "SSH-") {
		return ConnectionDiagnostic{
			Step:       "ssh_service",
			Status:     "warning",
			Message:    "Unexpected response from SSH service",
			Details:    fmt.Sprintf("Received: %s", banner),
			Suggestion: "Service might not be SSH or is misconfigured.",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_service",
		Status:    "success",
		Message:   fmt.Sprintf("SSH service available: %s", banner),
		Duration:  duration,
		Timestamp: start,
		Metadata: map[string]string{
			"ssh_banner": banner,
		},
	}
}

// testSSHProtocolNegotiation tests SSH protocol negotiation
func testSSHProtocolNegotiation(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Testing SSH protocol negotiation")

	config := &ssh.ClientConfig{
		User:            "test", // dummy user for protocol test
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := ssh.Dial("tcp", address, config)

	duration := time.Since(start)

	if err != nil {
		// Check if it's an authentication error (which means protocol worked)
		if strings.Contains(err.Error(), "unable to authenticate") ||
			strings.Contains(err.Error(), "authentication failed") ||
			strings.Contains(err.Error(), "no supported methods remain") {
			return ConnectionDiagnostic{
				Step:      "ssh_protocol",
				Status:    "success",
				Message:   "SSH protocol negotiation successful",
				Details:   "Authentication failed as expected with test credentials",
				Duration:  duration,
				Timestamp: start,
			}
		}

		return ConnectionDiagnostic{
			Step:       "ssh_protocol",
			Status:     "error",
			Message:    "SSH protocol negotiation failed",
			Details:    fmt.Sprintf("Failed to establish SSH protocol connection: %v", err),
			Suggestion: "Check SSH daemon configuration and supported protocols.",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	conn.Close()
	return ConnectionDiagnostic{
		Step:      "ssh_protocol",
		Status:    "success",
		Message:   "SSH protocol negotiation successful",
		Duration:  duration,
		Timestamp: start,
	}
}

// testSSHServiceEnhanced performs enhanced SSH service testing
func testSSHServiceEnhanced(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Testing SSH service availability (enhanced)")

	address := net.JoinHostPort(server.Host, fmt.Sprintf("%d", server.Port))
	conn, err := net.DialTimeout("tcp", address, 30*time.Second)

	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service_enhanced",
			Status:     "error",
			Message:    "SSH service not responding",
			Details:    fmt.Sprintf("Failed to connect to SSH service at %s: %v", address, err),
			Suggestion: "Ensure SSH daemon is running: sudo systemctl status ssh or sudo systemctl status sshd",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Set a reasonable deadline for reading the banner
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	conn.Close()

	duration := time.Since(start)

	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_service_enhanced",
			Status:     "warning",
			Message:    "SSH service responding but no banner received",
			Details:    fmt.Sprintf("Connection established but failed to read SSH banner: %v", err),
			Suggestion: "SSH service might be misconfigured or very slow. Check SSH daemon logs: sudo journalctl -u ssh -n 20",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	banner := strings.TrimSpace(string(buffer[:n]))
	if !strings.HasPrefix(banner, "SSH-") {
		return ConnectionDiagnostic{
			Step:       "ssh_service_enhanced",
			Status:     "warning",
			Message:    "Unexpected response from SSH service",
			Details:    fmt.Sprintf("Expected SSH banner, received: %s", banner),
			Suggestion: "Service on port might not be SSH or is misconfigured. Verify SSH daemon configuration.",
			Duration:   duration,
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_service_enhanced",
		Status:    "success",
		Message:   fmt.Sprintf("SSH service available: %s", banner),
		Duration:  duration,
		Timestamp: start,
		Metadata: map[string]string{
			"ssh_banner":    banner,
			"response_time": duration.String(),
		},
	}
}

// testActualSSHConnectionPooled tests actual SSH connection using connection pool
func testActualSSHConnectionPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Testing actual SSH connection with credentials using connection pool")

	// Try to get connection from pool (this will create one if needed)
	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		// Analyze the specific authentication failure
		authError := analyzeAuthenticationError(err)
		return ConnectionDiagnostic{
			Step:       "ssh_connection",
			Status:     "error",
			Message:    fmt.Sprintf("Failed to establish SSH connection as %s", server.AppUsername),
			Details:    authError.Details,
			Suggestion: authError.Suggestion,
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Test the connection
	if err := conn.TestHealth(); err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_connection",
			Status:     "error",
			Message:    "SSH connection established but health check failed",
			Details:    err.Error(),
			Suggestion: "Connection might be unstable. Check server resources and SSH daemon status.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_connection",
		Status:    "success",
		Message:   fmt.Sprintf("SSH connection established successfully as %s", server.AppUsername),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// testActualSSHConnection tests actual SSH connection (legacy function for backward compatibility)
func testActualSSHConnection(server *models.Server, username string) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return testActualSSHConnectionPooled(ctx)
}

// analyzeAuthenticationError provides detailed analysis of authentication errors
func analyzeAuthenticationError(err error) ConnectionDiagnostic {
	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "connection refused"):
		return ConnectionDiagnostic{
			Details:    "Connection refused - SSH daemon not running or port blocked",
			Suggestion: "Check if SSH daemon is running: sudo systemctl status ssh. Verify firewall rules and port accessibility.",
		}
	case strings.Contains(errStr, "permission denied"):
		return ConnectionDiagnostic{
			Details:    "Authentication failed - invalid credentials or key",
			Suggestion: "Verify SSH key is properly configured. Check ~/.ssh/authorized_keys on the server.",
		}
	case strings.Contains(errStr, "host key verification failed"):
		return ConnectionDiagnostic{
			Details:    "Host key verification failed",
			Suggestion: "Update known_hosts file or verify server identity. Use ssh-keyscan to update host keys.",
		}
	case strings.Contains(errStr, "no route to host"):
		return ConnectionDiagnostic{
			Details:    "Network routing issue - host unreachable",
			Suggestion: "Check network connectivity and routing. Verify the server IP address is correct.",
		}
	case strings.Contains(errStr, "timeout"):
		return ConnectionDiagnostic{
			Details:    "Connection timeout",
			Suggestion: "Server might be slow or overloaded. Check server resources and network latency.",
		}
	default:
		return ConnectionDiagnostic{
			Details:    fmt.Sprintf("Authentication error: %v", err),
			Suggestion: "Check SSH configuration, credentials, and server accessibility.",
		}
	}
}

// testAuthenticationMethods tests available authentication methods
func testAuthenticationMethods(server *models.Server) []ConnectionDiagnostic {
	var diagnostics []ConnectionDiagnostic

	// Test SSH agent
	diagnostics = append(diagnostics, checkSSHAgent(server))

	// Test private key
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	diagnostics = append(diagnostics, checkPrivateKey(ctx))

	return diagnostics
}

// checkSSHAgent checks SSH agent availability and loaded keys
func checkSSHAgent(server *models.Server) ConnectionDiagnostic {
	start := time.Now()

	logger.Debug("Checking SSH agent availability")

	agentSock := os.Getenv("SSH_AUTH_SOCK")
	if agentSock == "" {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "warning",
			Message:    "SSH agent not available",
			Details:    "SSH_AUTH_SOCK environment variable not set",
			Suggestion: "Start SSH agent: eval $(ssh-agent) && ssh-add",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	conn, err := net.Dial("unix", agentSock)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "warning",
			Message:    "Cannot connect to SSH agent",
			Details:    fmt.Sprintf("Failed to connect to agent socket %s: %v", agentSock, err),
			Suggestion: "Restart SSH agent: eval $(ssh-agent) && ssh-add",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "warning",
			Message:    "SSH agent available but cannot list keys",
			Details:    fmt.Sprintf("Error listing keys: %v", err),
			Suggestion: "Add keys to agent: ssh-add ~/.ssh/id_rsa",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	if len(keys) == 0 {
		return ConnectionDiagnostic{
			Step:       "ssh_agent",
			Status:     "warning",
			Message:    "SSH agent available but no keys loaded",
			Suggestion: "Load keys into agent: ssh-add ~/.ssh/id_rsa",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_agent",
		Status:    "success",
		Message:   fmt.Sprintf("SSH agent available with %d keys loaded", len(keys)),
		Duration:  time.Since(start),
		Timestamp: start,
		Metadata: map[string]string{
			"key_count": fmt.Sprintf("%d", len(keys)),
		},
	}
}

// checkPrivateKey checks for private key files
func checkPrivateKey(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()

	logger.Debug("Checking for SSH private key files")

	currentUser, err := user.Current()
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "private_key",
			Status:     "warning",
			Message:    "Cannot determine current user",
			Details:    fmt.Sprintf("Error getting current user: %v", err),
			Suggestion: "Verify user permissions and environment",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	keyFiles := []string{"id_rsa", "id_ecdsa", "id_ed25519", "id_dsa"}

	var foundKeys []string
	var keyDetails []string

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		if info, err := os.Stat(keyPath); err == nil {
			foundKeys = append(foundKeys, keyFile)

			// Check permissions
			mode := info.Mode()
			if mode&0077 != 0 {
				keyDetails = append(keyDetails, fmt.Sprintf("%s (WARNING: permissions %o, should be 600)", keyFile, mode))
			} else {
				keyDetails = append(keyDetails, fmt.Sprintf("%s (permissions OK)", keyFile))
			}

			// Check if public key exists
			pubKeyPath := keyPath + ".pub"
			if _, err := os.Stat(pubKeyPath); err != nil {
				keyDetails = append(keyDetails, fmt.Sprintf("  -> Missing public key: %s", pubKeyPath))
			}
		}
	}

	if len(foundKeys) == 0 {
		return ConnectionDiagnostic{
			Step:       "private_key",
			Status:     "warning",
			Message:    "No SSH private keys found",
			Details:    fmt.Sprintf("Searched in %s for: %v", sshDir, keyFiles),
			Suggestion: "Generate SSH key pair: ssh-keygen -t ed25519 -C \"your_email@example.com\"",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Check for any permission issues
	hasPermissionIssues := false
	for _, detail := range keyDetails {
		if strings.Contains(detail, "WARNING") {
			hasPermissionIssues = true
			break
		}
	}

	status := "success"
	suggestion := ""
	if hasPermissionIssues {
		status = "warning"
		suggestion = "Fix key permissions: chmod 600 ~/.ssh/id_* && chmod 644 ~/.ssh/*.pub"
	}

	return ConnectionDiagnostic{
		Step:       "private_key",
		Status:     status,
		Message:    fmt.Sprintf("Found %d SSH private keys", len(foundKeys)),
		Details:    strings.Join(keyDetails, "\n"),
		Suggestion: suggestion,
		Duration:   time.Since(start),
		Timestamp:  start,
		Metadata: map[string]string{
			"found_keys": strings.Join(foundKeys, ","),
		},
	}
}

// analyzeHostKey analyzes host key issues
func analyzeHostKey(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Analyzing host key configuration")

	currentUser, err := user.Current()
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "host_key",
			Status:    "warning",
			Message:   "Cannot check known_hosts",
			Details:   fmt.Sprintf("Error getting current user: %v", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	knownHostsPath := filepath.Join(currentUser.HomeDir, ".ssh", "known_hosts")

	// Check if known_hosts exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return ConnectionDiagnostic{
			Step:       "host_key",
			Status:     "warning",
			Message:    "known_hosts file does not exist",
			Details:    fmt.Sprintf("File not found: %s", knownHostsPath),
			Suggestion: fmt.Sprintf("Add host key: ssh-keyscan -H %s >> ~/.ssh/known_hosts", server.Host),
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Read known_hosts and check for this host
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "host_key",
			Status:    "warning",
			Message:   "Cannot read known_hosts file",
			Details:   fmt.Sprintf("Error reading %s: %v", knownHostsPath, err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	lines := strings.Split(string(content), "\n")
	hostFound := false
	hashedHost := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		hostPart := parts[0]

		// Check for direct host match
		if strings.Contains(hostPart, server.Host) {
			hostFound = true
			break
		}

		// Check for hashed hosts (starts with |1|)
		if strings.HasPrefix(hostPart, "|1|") {
			hashedHost = true
		}
	}

	if hostFound {
		return ConnectionDiagnostic{
			Step:      "host_key",
			Status:    "success",
			Message:   "Host key found in known_hosts",
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	message := "Host key not found in known_hosts"
	suggestion := fmt.Sprintf("Add host key: ssh-keyscan -H %s >> ~/.ssh/known_hosts", server.Host)

	if hashedHost {
		message += " (some entries are hashed)"
		suggestion += " or verify with: ssh-keygen -F " + server.Host
	}

	return ConnectionDiagnostic{
		Step:       "host_key",
		Status:     "warning",
		Message:    message,
		Details:    fmt.Sprintf("Host %s not found in %s", server.Host, knownHostsPath),
		Suggestion: suggestion,
		Duration:   time.Since(start),
		Timestamp:  start,
	}
}

// analyzeSSHClientConfig analyzes SSH client configuration
func analyzeSSHClientConfig(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.Debug("Analyzing SSH client configuration")

	currentUser, err := user.Current()
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "ssh_config",
			Status:    "warning",
			Message:   "Cannot check SSH config",
			Details:   fmt.Sprintf("Error getting current user: %v", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	configPath := filepath.Join(currentUser.HomeDir, ".ssh", "config")

	// Check if SSH config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return ConnectionDiagnostic{
			Step:       "ssh_config",
			Status:     "info",
			Message:    "No SSH client config found",
			Details:    fmt.Sprintf("File not found: %s", configPath),
			Suggestion: "Consider creating ~/.ssh/config for host-specific settings",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Read and analyze config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "ssh_config",
			Status:    "warning",
			Message:   "Cannot read SSH config",
			Details:   fmt.Sprintf("Error reading %s: %v", configPath, err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	// Analyze config for relevant settings
	hasHostConfig := strings.Contains(string(content), server.Host)

	return ConnectionDiagnostic{
		Step:      "ssh_config",
		Status:    "success",
		Message:   "SSH client config found",
		Details:   fmt.Sprintf("Config file exists (%d bytes)", len(content)),
		Duration:  time.Since(start),
		Timestamp: start,
		Metadata: map[string]string{
			"has_host_config": fmt.Sprintf("%t", hasHostConfig),
		},
	}
}

// checkSSHPermissions checks SSH-related file and directory permissions
func checkSSHPermissions(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()

	logger.Debug("Checking SSH file and directory permissions")

	currentUser, err := user.Current()
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "ssh_permissions",
			Status:    "warning",
			Message:   "Cannot check SSH permissions",
			Details:   fmt.Sprintf("Error getting current user: %v", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")

	// Check .ssh directory
	if info, err := os.Stat(sshDir); err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_permissions",
			Status:     "warning",
			Message:    ".ssh directory does not exist",
			Details:    fmt.Sprintf("Directory not found: %s", sshDir),
			Suggestion: fmt.Sprintf("Create SSH directory: mkdir -p %s && chmod 700 %s", sshDir, sshDir),
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	} else {
		mode := info.Mode()
		if mode&0077 != 0 {
			return ConnectionDiagnostic{
				Step:       "ssh_permissions",
				Status:     "warning",
				Message:    ".ssh directory has incorrect permissions",
				Details:    fmt.Sprintf("Current permissions: %o, should be 700", mode&0777),
				Suggestion: fmt.Sprintf("Fix permissions: chmod 700 %s", sshDir),
				Duration:   time.Since(start),
				Timestamp:  start,
			}
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_permissions",
		Status:    "success",
		Message:   "SSH permissions are correct",
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// isHostInKnownHostsEnhanced checks if host is in known_hosts with enhanced analysis
func isHostInKnownHostsEnhanced(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Checking known_hosts file (enhanced)")

	currentUser, err := user.Current()
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "known_hosts",
			Status:    "warning",
			Message:   "Cannot check known_hosts",
			Details:   fmt.Sprintf("Error getting current user: %v", err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	knownHostsPath := filepath.Join(currentUser.HomeDir, ".ssh", "known_hosts")

	// Check if file exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return ConnectionDiagnostic{
			Step:       "known_hosts",
			Status:     "warning",
			Message:    "known_hosts file does not exist",
			Details:    fmt.Sprintf("File not found: %s", knownHostsPath),
			Suggestion: fmt.Sprintf("Add host key: ssh-keyscan -H %s >> ~/.ssh/known_hosts", server.Host),
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Read and check file
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		return ConnectionDiagnostic{
			Step:      "known_hosts",
			Status:    "warning",
			Message:   "Cannot read known_hosts file",
			Details:   fmt.Sprintf("Error reading %s: %v", knownHostsPath, err),
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	lines := strings.Split(string(content), "\n")
	hostFound := false
	hashedEntries := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		hostPart := parts[0]

		// Check for hashed entries
		if strings.HasPrefix(hostPart, "|1|") {
			hashedEntries++
			continue
		}

		// Check for direct host match
		if strings.Contains(hostPart, server.Host) {
			hostFound = true
			break
		}
	}

	if hostFound {
		return ConnectionDiagnostic{
			Step:      "known_hosts",
			Status:    "success",
			Message:   "Host key found in known_hosts",
			Duration:  time.Since(start),
			Timestamp: start,
		}
	}

	message := "Host key not found in known_hosts"
	details := fmt.Sprintf("Host %s not found in %s", server.Host, knownHostsPath)

	if hashedEntries > 0 {
		details += fmt.Sprintf(" (%d hashed entries present)", hashedEntries)
	}

	return ConnectionDiagnostic{
		Step:       "known_hosts",
		Status:     "warning",
		Message:    message,
		Details:    details,
		Suggestion: fmt.Sprintf("Add host key: ssh-keyscan -H %s >> ~/.ssh/known_hosts", server.Host),
		Duration:   time.Since(start),
		Timestamp:  start,
	}
}

// testAuthenticationMethodsEnhanced performs enhanced authentication testing using connection pool
func testAuthenticationMethodsEnhanced(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"username": server.AppUsername,
	}).Debug("Testing authentication methods (enhanced)")

	// Try to get connection from pool - this tests actual authentication
	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		authError := analyzeAuthenticationError(err)
		return ConnectionDiagnostic{
			Step:       "authentication_methods",
			Status:     "error",
			Message:    "Authentication failed",
			Details:    authError.Details,
			Suggestion: authError.Suggestion,
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Test basic command execution to ensure auth is fully working
	testOutput, err := conn.ExecuteCommand("echo 'auth_test_successful'")
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "authentication_methods",
			Status:     "warning",
			Message:    "Authentication succeeded but command execution failed",
			Details:    fmt.Sprintf("Test command failed: %v", err),
			Suggestion: "Check user shell configuration and PATH settings.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	if !strings.Contains(testOutput, "auth_test_successful") {
		return ConnectionDiagnostic{
			Step:       "authentication_methods",
			Status:     "warning",
			Message:    "Authentication works but unexpected command output",
			Details:    fmt.Sprintf("Expected 'auth_test_successful', got: %s", strings.TrimSpace(testOutput)),
			Suggestion: "Check shell configuration and environment variables.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "authentication_methods",
		Status:    "success",
		Message:   fmt.Sprintf("Authentication successful as %s", server.AppUsername),
		Details:   "SSH connection established and command execution working",
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// checkSSHAgentEnhanced performs enhanced SSH agent checking
func checkSSHAgentEnhanced(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()

	logger.Debug("Checking SSH agent (enhanced)")

	agentSock := os.Getenv("SSH_AUTH_SOCK")
	if agentSock == "" {
		return ConnectionDiagnostic{
			Step:       "ssh_agent_enhanced",
			Status:     "warning",
			Message:    "SSH agent not available",
			Details:    "SSH_AUTH_SOCK environment variable not set",
			Suggestion: "Start SSH agent: eval $(ssh-agent) && ssh-add",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	conn, err := net.Dial("unix", agentSock)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_agent_enhanced",
			Status:     "warning",
			Message:    "Cannot connect to SSH agent",
			Details:    fmt.Sprintf("Failed to connect to agent socket %s: %v", agentSock, err),
			Suggestion: "Restart SSH agent: eval $(ssh-agent) && ssh-add",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_agent_enhanced",
			Status:     "warning",
			Message:    "SSH agent accessible but cannot list keys",
			Details:    fmt.Sprintf("Error listing keys: %v", err),
			Suggestion: "Add keys to agent: ssh-add ~/.ssh/id_rsa",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	if len(keys) == 0 {
		return ConnectionDiagnostic{
			Step:       "ssh_agent_enhanced",
			Status:     "warning",
			Message:    "SSH agent available but no keys loaded",
			Suggestion: "Load keys into agent: ssh-add ~/.ssh/id_rsa",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Analyze key types
	var keyTypes []string
	for _, key := range keys {
		keyTypes = append(keyTypes, key.Type())
	}

	return ConnectionDiagnostic{
		Step:      "ssh_agent_enhanced",
		Status:    "success",
		Message:   fmt.Sprintf("SSH agent available with %d keys loaded", len(keys)),
		Details:   fmt.Sprintf("Key types: %s", strings.Join(keyTypes, ", ")),
		Duration:  time.Since(start),
		Timestamp: start,
		Metadata: map[string]string{
			"key_count": fmt.Sprintf("%d", len(keys)),
			"key_types": strings.Join(keyTypes, ","),
		},
	}
}

// DiagnoseAppUserPostSecurity performs post-security diagnostics using connection pool
func DiagnoseAppUserPostSecurity(server *models.Server) ([]ConnectionDiagnostic, error) {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()

	var diagnostics []ConnectionDiagnostic

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Info("Starting post-security SSH diagnostics for app user")

	// Test basic connection first using connection pool
	basicConnDiag := diagnoseAppUserConnectionPooled(ctx)
	diagnostics = append(diagnostics, basicConnDiag)

	// If basic connection fails, skip other tests that require SSH access
	if basicConnDiag.Status == "error" {
		logger.WithFields(map[string]interface{}{
			"host": server.Host,
			"port": server.Port,
		}).Warn("Basic app user connection failed, skipping advanced post-security diagnostics")

		return diagnostics, nil
	}

	// Continue with other diagnostics only if basic connection works
	diagnostics = append(diagnostics, checkAppUserSudoAccessPooled(ctx))
	diagnostics = append(diagnostics, checkAppUserSSHKeysPooled(ctx))
	diagnostics = append(diagnostics, verifyPostSecurityAccessPooled(ctx))
	diagnostics = append(diagnostics, checkSSHDaemonConfigPooled(ctx))

	logger.WithFields(map[string]interface{}{
		"host":             server.Host,
		"port":             server.Port,
		"username":         server.AppUsername,
		"diagnostic_count": len(diagnostics),
	}).Info("Post-security SSH diagnostics completed")

	return diagnostics, nil
}

// diagnoseAppUserConnectionPooled tests app user connection using connection pool
func diagnoseAppUserConnectionPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Diagnosing app user SSH connection using connection pool")

	// Try to get connection from pool
	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		authError := analyzeAuthenticationError(err)
		return ConnectionDiagnostic{
			Step:       "app_user_connection",
			Status:     "error",
			Message:    fmt.Sprintf("Failed to connect as %s", server.AppUsername),
			Details:    authError.Details,
			Suggestion: authError.Suggestion,
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Test basic command execution
	if err := conn.TestHealth(); err != nil {
		return ConnectionDiagnostic{
			Step:       "app_user_connection",
			Status:     "error",
			Message:    fmt.Sprintf("App user %s can connect but health check fails", server.AppUsername),
			Details:    fmt.Sprintf("Health check error: %v", err),
			Suggestion: "Check shell configuration and command permissions for the app user.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "app_user_connection",
		Status:    "success",
		Message:   fmt.Sprintf("App user %s connection is working", server.AppUsername),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// checkAppUserSudoAccessPooled verifies sudo configuration using connection pool
func checkAppUserSudoAccessPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking app user sudo access using connection pool")

	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sudo_access",
			Status:     "error",
			Message:    "Cannot test sudo access - SSH connection failed",
			Details:    fmt.Sprintf("Connection error: %v", err),
			Suggestion: "Fix SSH connection first before testing sudo access.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Test sudo access
	testCmd := "sudo -n whoami"
	output, err := conn.ExecuteCommand(testCmd)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sudo_access",
			Status:     "error",
			Message:    fmt.Sprintf("Sudo access test failed for %s", server.AppUsername),
			Details:    fmt.Sprintf("Command '%s' failed: %v", testCmd, err),
			Suggestion: fmt.Sprintf("Configure passwordless sudo for %s. Add to /etc/sudoers.d/%s: '%s ALL=(ALL) NOPASSWD:ALL'", server.AppUsername, server.AppUsername, server.AppUsername),
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	if !strings.Contains(output, "root") {
		return ConnectionDiagnostic{
			Step:       "sudo_access",
			Status:     "warning",
			Message:    "Sudo command executed but unexpected output",
			Details:    fmt.Sprintf("Expected 'root', got: %s", strings.TrimSpace(output)),
			Suggestion: "Check sudo configuration and shell environment.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "sudo_access",
		Status:    "success",
		Message:   fmt.Sprintf("Sudo access is properly configured for %s", server.AppUsername),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// checkAppUserSSHKeysPooled verifies SSH key configuration using connection pool
func checkAppUserSSHKeysPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking app user SSH keys using connection pool")

	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_keys",
			Status:     "error",
			Message:    "Cannot check SSH keys - connection failed",
			Details:    fmt.Sprintf("Connection error: %v", err),
			Suggestion: "SSH key configuration may be incorrect. Check authorized_keys file.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Check authorized_keys file
	authKeysPath := ".ssh/authorized_keys"
	checkCmd := fmt.Sprintf("test -f %s && wc -l %s", authKeysPath, authKeysPath)
	output, err := conn.ExecuteCommand(checkCmd)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "ssh_keys",
			Status:     "error",
			Message:    fmt.Sprintf("SSH authorized_keys file missing for %s", server.AppUsername),
			Details:    "File ~/.ssh/authorized_keys does not exist or is not accessible",
			Suggestion: "Ensure authorized_keys file exists and contains valid SSH public keys",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	keyCount := strings.TrimSpace(output)
	if keyCount == "0" {
		return ConnectionDiagnostic{
			Step:       "ssh_keys",
			Status:     "warning",
			Message:    "SSH authorized_keys file is empty",
			Details:    "No public keys found in authorized_keys file",
			Suggestion: "Add your public key to ~/.ssh/authorized_keys on the server",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "ssh_keys",
		Status:    "success",
		Message:   fmt.Sprintf("SSH keys are properly configured for %s (%s keys)", server.AppUsername, keyCount),
		Duration:  time.Since(start),
		Timestamp: start,
		Metadata: map[string]string{
			"key_count": keyCount,
		},
	}
}

// verifyPostSecurityAccessPooled checks post-security operations using connection pool
func verifyPostSecurityAccessPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Verifying post-security access using connection pool")

	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "post_security_access",
			Status:     "error",
			Message:    "Cannot verify post-security access - SSH connection failed",
			Details:    fmt.Sprintf("Connection error: %v", err),
			Suggestion: "Fix SSH connection issues first.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Test deployment-related operations
	testCmd := "ls -la /opt/pocketbase 2>/dev/null || echo 'directory_not_found'"
	output, err := conn.ExecuteCommand(testCmd)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "post_security_access",
			Status:     "warning",
			Message:    "Post-security access test had issues",
			Details:    fmt.Sprintf("Directory access test failed: %v", err),
			Suggestion: "Check if deployment directories exist and are accessible.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	if strings.Contains(output, "directory_not_found") {
		return ConnectionDiagnostic{
			Step:       "post_security_access",
			Status:     "info",
			Message:    "Deployment directory not yet created",
			Details:    "/opt/pocketbase directory does not exist",
			Suggestion: "This is normal for new servers. Directory will be created during deployment.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "post_security_access",
		Status:    "success",
		Message:   "App user has necessary access for post-security operations",
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// checkSSHDaemonConfigPooled checks SSH daemon configuration using connection pool
func checkSSHDaemonConfigPooled(ctx *DiagnosticContext) ConnectionDiagnostic {
	start := time.Now()
	server := ctx.server

	logger.WithFields(map[string]interface{}{
		"host":     server.Host,
		"port":     server.Port,
		"username": server.AppUsername,
	}).Debug("Checking SSH daemon configuration using connection pool")

	conn, err := ctx.GetCachedConnection(false)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sshd_config",
			Status:     "error",
			Message:    "Cannot check SSH daemon config - SSH connection failed",
			Details:    fmt.Sprintf("Connection error: %v", err),
			Suggestion: "Fix SSH connection issues first.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Check if we can view SSH config (requires sudo)
	configCmd := "sudo -n cat /etc/ssh/sshd_config | grep -E '^(PubkeyAuthentication|PasswordAuthentication|PermitRootLogin)' | head -3"
	output, err := conn.ExecuteCommand(configCmd)
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "sshd_config",
			Status:     "warning",
			Message:    "Cannot read SSH daemon configuration",
			Details:    fmt.Sprintf("Command failed: %v", err),
			Suggestion: "May need sudo access to view SSH daemon configuration.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Analyze SSH config settings
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var configIssues []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "PasswordAuthentication yes") {
			configIssues = append(configIssues, "Password authentication is enabled (security risk)")
		}
		if strings.Contains(line, "PermitRootLogin yes") {
			configIssues = append(configIssues, "Root login is permitted (security risk)")
		}
	}

	if len(configIssues) > 0 {
		return ConnectionDiagnostic{
			Step:       "sshd_config",
			Status:     "warning",
			Message:    "SSH daemon configuration has security concerns",
			Details:    strings.Join(configIssues, "; "),
			Suggestion: "Review /etc/ssh/sshd_config for security hardening settings.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "sshd_config",
		Status:    "success",
		Message:   "SSH daemon configuration appears secure",
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// Legacy functions for backward compatibility

// FixCommonIssues attempts to fix common SSH configuration issues
func FixCommonIssues(server *models.Server) []ConnectionDiagnostic {
	var results []ConnectionDiagnostic

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Attempting to fix common SSH issues")

	currentUser, err := user.Current()
	if err != nil {
		results = append(results, ConnectionDiagnostic{
			Step:    "fix_user_detection",
			Status:  "error",
			Message: "Cannot determine current user",
			Details: fmt.Sprintf("Error: %v", err),
		})
		return results
	}

	// Fix .ssh directory
	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		results = append(results, ConnectionDiagnostic{
			Step:    "fix_ssh_dir",
			Status:  "error",
			Message: "Failed to create/fix .ssh directory",
			Details: fmt.Sprintf("Error: %v", err),
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
			Details: fmt.Sprintf("Error: %v", err),
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

// GetConnectionSummary provides a summary of connection diagnostics
func GetConnectionSummary(server *models.Server, asRoot bool) (string, error) {
	diagnostics, err := TroubleshootConnection(server, "")
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

// DiagnoseConnectionRefused provides guidance for connection refused errors
func DiagnoseConnectionRefused(server *models.Server) ConnectionDiagnostic {
	start := time.Now()

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Info("Diagnosing connection refused error")

	currentIP, _ := getCurrentPublicIP()

	details := fmt.Sprintf(`Connection refused typically indicates:

1. SSH service is not running on the server
2. Port %d is blocked by a firewall
3. Your IP (%s) has been banned by fail2ban
4. The server is down or unreachable

To diagnose on the server (if you have console access):
• Check SSH service: sudo systemctl status ssh
• Check fail2ban status: sudo fail2ban-client status sshd
• Check if your IP is banned: sudo fail2ban-client get sshd banip | grep %s

To fix fail2ban ban:
• Unban your IP: sudo fail2ban-client set sshd unbanip %s`,
		server.Port, currentIP, currentIP, currentIP)

	suggestion := fmt.Sprintf("If you have console access to the server, check: 1) SSH service status 2) fail2ban status 3) Whether IP %s is banned", currentIP)

	return ConnectionDiagnostic{
		Step:       "connection_refused_analysis",
		Status:     "error",
		Message:    fmt.Sprintf("Connection refused to %s:%d - likely fail2ban ban", server.Host, server.Port),
		Details:    details,
		Suggestion: suggestion,
		Duration:   time.Since(start),
		Timestamp:  start,
	}
}

// DiagnoseConnectionRefusedImmediate provides immediate diagnostic for connection refused errors
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

	fmt.Printf("METHOD 2 - Wait it out:\n")
	fmt.Printf("1. fail2ban bans are usually temporary (default: 10 minutes)\n")
	fmt.Printf("2. Wait and try again later\n\n")
}

// QuickFail2banCheck performs a quick connectivity test with fail2ban analysis
func QuickFail2banCheck(host string, port int) error {
	currentIP, ipErr := getCurrentPublicIP()

	fmt.Printf("🔍 Quick fail2ban diagnostic for %s:%d\n", host, port)
	if ipErr != nil {
		fmt.Printf("⚠️  Could not determine your IP: %v\n", ipErr)
		currentIP = "unknown"
	} else {
		fmt.Printf("📍 Your IP: %s\n\n", currentIP)
	}

	// Test connectivity
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", address, 8*time.Second)

	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			fmt.Printf("❌ Connection refused - likely fail2ban ban\n")
			if currentIP != "unknown" {
				fmt.Printf("💡 Solution: sudo fail2ban-client set sshd unbanip %s\n", currentIP)
			}
			return fmt.Errorf("connection refused - likely fail2ban ban")
		}
		fmt.Printf("❌ Connection failed: %v\n", err)
		return err
	}

	conn.Close()
	fmt.Printf("✅ Connection successful\n")
	return nil
}

// getCurrentPublicIP attempts to determine the current public IP address
func getCurrentPublicIP() (string, error) {
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

		if resp.StatusCode != 200 {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		if net.ParseIP(ip) != nil {
			return ip, nil
		}
	}

	return "", fmt.Errorf("could not determine public IP")
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
	} else if warningCount > 0 {
		summary.WriteString("\n✓ App user access is working, but consider addressing warnings.\n")
	} else {
		summary.WriteString("\n✅ All checks passed! App user access is fully functional.\n")
	}

	return summary.String(), nil
}

// checkFail2banStatus checks if the current IP might be banned by fail2ban
func checkFail2banStatus(server *models.Server) ConnectionDiagnostic {
	start := time.Now()

	logger.WithFields(map[string]interface{}{
		"host": server.Host,
		"port": server.Port,
	}).Debug("Checking fail2ban status for potential IP ban")

	currentIP, err := getCurrentPublicIP()
	if err != nil {
		return ConnectionDiagnostic{
			Step:       "fail2ban_check",
			Status:     "warning",
			Message:    "Could not determine current public IP",
			Details:    fmt.Sprintf("Error: %v", err),
			Suggestion: "Unable to check if your IP is banned by fail2ban. Try from different IP if issues persist.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:       "fail2ban_check",
		Status:     "warning",
		Message:    fmt.Sprintf("Connection refused - possible fail2ban ban (your IP: %s)", currentIP),
		Details:    "Connection refused errors often indicate that fail2ban has banned your IP address.",
		Suggestion: fmt.Sprintf("Check server logs and unban if needed: sudo fail2ban-client set sshd unbanip %s", currentIP),
		Duration:   time.Since(start),
		Timestamp:  start,
	}
}

// checkFail2banBanStatus checks if an IP is banned (requires server access)
func checkFail2banBanStatus(conn *PooledConnection, targetIP string) ConnectionDiagnostic {
	start := time.Now()

	logger.WithFields(map[string]interface{}{
		"target_ip": targetIP,
	}).Debug("Checking fail2ban ban status for IP")

	// Check if fail2ban is running
	statusCmd := "sudo systemctl is-active fail2ban"
	output, err := conn.ExecuteCommand(statusCmd)
	if err != nil || !strings.Contains(output, "active") {
		return ConnectionDiagnostic{
			Step:       "fail2ban_status",
			Status:     "info",
			Message:    "fail2ban service is not running",
			Details:    "fail2ban is not active, so IP banning is not the issue",
			Suggestion: "Connection issues are not related to fail2ban. Check SSH service status.",
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	// Check if IP is currently banned
	banCheckCmd := fmt.Sprintf("sudo fail2ban-client get sshd banip | grep -q '%s'", targetIP)
	_, err = conn.ExecuteCommand(banCheckCmd)
	if err == nil {
		return ConnectionDiagnostic{
			Step:       "fail2ban_ban_check",
			Status:     "error",
			Message:    fmt.Sprintf("IP %s is currently banned by fail2ban", targetIP),
			Details:    "Your IP address has been banned by fail2ban due to multiple failed attempts.",
			Suggestion: fmt.Sprintf("Unban your IP: sudo fail2ban-client set sshd unbanip %s", targetIP),
			Duration:   time.Since(start),
			Timestamp:  start,
		}
	}

	return ConnectionDiagnostic{
		Step:      "fail2ban_ban_check",
		Status:    "success",
		Message:   fmt.Sprintf("IP %s is not banned by fail2ban", targetIP),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// DiagnoseWithContext performs troubleshooting with timeout context
func DiagnoseWithContext(server *models.Server, asRoot bool, timeout time.Duration) ([]ConnectionDiagnostic, error) {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()

	// Override context timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx.ctx, timeout)
	defer cancel()
	ctx.ctx = ctxWithTimeout

	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"timeout": timeout.String(),
	}).Info("Starting context-aware SSH diagnostics")

	// Run diagnostics with timeout
	done := make(chan []ConnectionDiagnostic, 1)
	errCh := make(chan error, 1)

	go func() {
		diags, err := TroubleshootConnection(server, "")
		if err != nil {
			errCh <- err
			return
		}
		done <- diags
	}()

	select {
	case <-ctxWithTimeout.Done():
		return []ConnectionDiagnostic{{
			Step:       "diagnostic_timeout",
			Status:     "error",
			Message:    "Diagnostic operation timed out",
			Details:    fmt.Sprintf("Operation exceeded %v timeout", timeout),
			Suggestion: "Server may be overloaded. Try again with longer timeout.",
			Duration:   timeout,
			Timestamp:  time.Now(),
		}}, fmt.Errorf("diagnostic timeout after %v", timeout)
	case err := <-errCh:
		return nil, err
	case diags := <-done:
		return diags, nil
	}
}

// AnalyzeDiagnosticPatterns analyzes diagnostic results to detect patterns
func AnalyzeDiagnosticPatterns(diagnostics []ConnectionDiagnostic) map[string]interface{} {
	analysis := map[string]interface{}{
		"pattern_detected": "unknown",
		"confidence":       0.0,
		"auto_fixable":     false,
		"priority":         "medium",
		"category":         "general",
	}

	errorCount := 0
	warningCount := 0
	hasNetworkIssue := false
	hasAuthIssue := false

	for _, diag := range diagnostics {
		switch diag.Status {
		case "error":
			errorCount++
			if diag.Step == "network_connectivity" {
				hasNetworkIssue = true
			}
			if diag.Step == "ssh_connection" {
				hasAuthIssue = true
			}
		case "warning":
			warningCount++
		}
	}

	if hasNetworkIssue {
		analysis["pattern_detected"] = "network_connectivity"
		analysis["confidence"] = 0.9
		analysis["priority"] = "critical"
		analysis["category"] = "infrastructure"
	} else if hasAuthIssue {
		analysis["pattern_detected"] = "authentication_failure"
		analysis["confidence"] = 0.8
		analysis["priority"] = "high"
		analysis["category"] = "authentication"
	} else if errorCount == 0 && warningCount == 0 {
		analysis["pattern_detected"] = "healthy"
		analysis["confidence"] = 1.0
		analysis["priority"] = "info"
		analysis["category"] = "success"
	}

	analysis["error_count"] = errorCount
	analysis["warning_count"] = warningCount

	return analysis
}

// GenerateActionableSuggestions creates actionable suggestions from diagnostics
func GenerateActionableSuggestions(diagnostics []ConnectionDiagnostic, server *models.Server) []map[string]interface{} {
	var suggestions []map[string]interface{}
	seen := make(map[string]bool)

	for _, diag := range diagnostics {
		if diag.Suggestion == "" {
			continue
		}

		key := fmt.Sprintf("%s:%s", diag.Step, diag.Suggestion)
		if seen[key] {
			continue
		}
		seen[key] = true

		priority := "medium"
		category := "general"
		automated := false

		if strings.Contains(diag.Suggestion, "fail2ban") {
			priority = "critical"
			category = "security"
		} else if strings.Contains(diag.Suggestion, "chmod") {
			priority = "medium"
			category = "permissions"
			automated = true
		} else if strings.Contains(diag.Suggestion, "ssh-keygen") {
			priority = "high"
			category = "authentication"
		}

		suggestion := map[string]interface{}{
			"category":    category,
			"action":      diag.Suggestion,
			"description": diag.Message,
			"automated":   automated,
			"priority":    priority,
		}

		suggestions = append(suggestions, suggestion)
	}

	// Sort by priority
	sort.Slice(suggestions, func(i, j int) bool {
		priorities := map[string]int{"critical": 1, "high": 2, "medium": 3, "low": 4}
		pi := priorities[suggestions[i]["priority"].(string)]
		pj := priorities[suggestions[j]["priority"].(string)]
		return pi < pj
	})

	return suggestions
}

// DetectCriticalIssues identifies critical issues that prevent SSH access
func DetectCriticalIssues(diagnostics []ConnectionDiagnostic) []string {
	var critical []string

	for _, diag := range diagnostics {
		if diag.Status == "error" {
			switch diag.Step {
			case "network_connectivity":
				if strings.Contains(diag.Details, "connection refused") {
					critical = append(critical, "fail2ban_ip_ban")
				} else {
					critical = append(critical, "network_error")
				}
			case "ssh_service":
				critical = append(critical, "ssh_service_down")
			case "ssh_connection":
				critical = append(critical, "authentication_failure")
			}
		}
	}

	return critical
}

// GenerateRecoveryPlan creates a recovery plan based on diagnostics
func GenerateRecoveryPlan(diagnostics []ConnectionDiagnostic, server *models.Server) map[string]interface{} {
	criticalIssues := DetectCriticalIssues(diagnostics)

	plan := map[string]interface{}{
		"has_critical_issues": len(criticalIssues) > 0,
		"critical_issues":     criticalIssues,
		"estimated_time":      "5-30 minutes",
		"success_probability": 0.5,
		"steps":               []map[string]interface{}{},
	}

	var steps []map[string]interface{}

	if containsString(criticalIssues, "fail2ban_ip_ban") {
		plan["success_probability"] = 0.9
		plan["estimated_time"] = "2-5 minutes"

		steps = append(steps, map[string]interface{}{
			"step":        1,
			"title":       "Access Server Console",
			"description": "Use hosting provider's console/VNC to access server",
			"required":    true,
		})

		steps = append(steps, map[string]interface{}{
			"step":        2,
			"title":       "Unban IP Address",
			"description": "Remove IP ban from fail2ban",
			"command":     "sudo fail2ban-client set sshd unbanip <your_ip>",
			"required":    true,
		})
	}

	if containsString(criticalIssues, "authentication_failure") {
		steps = append(steps, map[string]interface{}{
			"step":        1,
			"title":       "Verify SSH Key",
			"description": "Check SSH key configuration",
			"command":     "ssh-add -l",
			"required":    true,
		})
	}

	plan["steps"] = steps
	return plan
}

// EstimateDiagnosticDuration estimates diagnostic duration
func EstimateDiagnosticDuration(server *models.Server, comprehensive bool) time.Duration {
	baseTime := 15 * time.Second

	if comprehensive {
		baseTime += 30 * time.Second
	}

	if server.SecurityLocked {
		baseTime += 20 * time.Second
	}

	return baseTime
}

// Utility functions

// contains checks if a string slice contains a specific string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Legacy diagnostic functions for backward compatibility

// diagnoseAppUserConnection wraps the pooled version for backward compatibility
func diagnoseAppUserConnection(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return diagnoseAppUserConnectionPooled(ctx)
}

// checkAppUserSudoAccess wraps the pooled version for backward compatibility
func checkAppUserSudoAccess(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return checkAppUserSudoAccessPooled(ctx)
}

// checkAppUserSSHKeys wraps the pooled version for backward compatibility
func checkAppUserSSHKeys(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return checkAppUserSSHKeysPooled(ctx)
}

// verifyPostSecurityAccess wraps the pooled version for backward compatibility
func verifyPostSecurityAccess(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return verifyPostSecurityAccessPooled(ctx)
}

// checkSSHDaemonConfig wraps the pooled version for backward compatibility
func checkSSHDaemonConfig(server *models.Server) ConnectionDiagnostic {
	ctx := NewDiagnosticContext(server, "")
	defer ctx.Close()
	return checkSSHDaemonConfigPooled(ctx)
}

// isHostInKnownHosts is a simple wrapper for backward compatibility
func isHostInKnownHosts(knownHostsPath, hostname string) bool {
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return false
	}

	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		return false
	}

	return strings.Contains(string(content), hostname)
}
