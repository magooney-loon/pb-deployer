package ssh

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"pb-deployer/internal/models"
)

// SSHManager handles SSH connections and command execution for server management
type SSHManager struct {
	server   *models.Server
	conn     *ssh.Client
	username string
	isRoot   bool
}

// SetupStep represents a step in the server setup process
type SetupStep struct {
	Step        string `json:"step"`
	Status      string `json:"status"` // running/success/failed
	Message     string `json:"message"`
	Details     string `json:"details,omitempty"`
	Timestamp   string `json:"timestamp"`
	ProgressPct int    `json:"progress_pct"`
}

// SendProgressUpdate is a helper to send progress updates with logging
func (sm *SSHManager) SendProgressUpdate(progressChan chan<- SetupStep, step, status, message string, progressPct int, details ...string) {
	if progressChan == nil {
		return
	}

	detailsStr := ""
	if len(details) > 0 {
		detailsStr = details[0]
	}

	progressChan <- SetupStep{
		Step:        step,
		Status:      status,
		Message:     message,
		Details:     detailsStr,
		Timestamp:   time.Now().Format(time.RFC3339),
		ProgressPct: progressPct,
	}
}

// AcceptHostKey pre-accepts a host key for a server to avoid connection failures
func AcceptHostKey(server *models.Server) error {
	if server == nil {
		return fmt.Errorf("server cannot be nil")
	}

	// Create a temporary connection just to get and store the host key
	config := &ssh.ClientConfig{
		User:            "dummy", // We just want the host key
		HostKeyCallback: createHostKeyAcceptorCallback(server),
		Timeout:         10 * time.Second,
		Auth:            []ssh.AuthMethod{}, // No auth needed for host key
	}

	address := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		// This is expected since we have no auth methods
		// But the host key should have been captured
		if !strings.Contains(err.Error(), "no supported methods remain") &&
			!strings.Contains(err.Error(), "unable to authenticate") {
			return fmt.Errorf("failed to capture host key: %w", err)
		}
	}
	if conn != nil {
		conn.Close()
	}

	return nil
}

// NewSSHManager creates a new SSH manager instance and establishes connection
func NewSSHManager(server *models.Server, asRoot bool) (*SSHManager, error) {
	if server == nil {
		return nil, fmt.Errorf("server cannot be nil")
	}

	// Validate server configuration
	if err := validateServerConfig(server); err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	// Validate username
	if username == "" {
		if asRoot {
			return nil, fmt.Errorf("root username cannot be empty")
		}
		return nil, fmt.Errorf("app username cannot be empty")
	}

	// Create SSH client configuration
	config, err := createSSHConfig(server, username)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Establish connection with retry logic
	address := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := establishConnectionWithRetry(address, config, 3)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s as %s after retries: %w", address, username, err)
	}

	return &SSHManager{
		server:   server,
		conn:     conn,
		username: username,
		isRoot:   asRoot,
	}, nil
}

// createSSHConfig builds the SSH client configuration based on server settings
func createSSHConfig(server *models.Server, username string) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: createHostKeyCallback(server),
		Timeout:         30 * time.Second,
	}

	var authMethods []ssh.AuthMethod

	// Try SSH agent first if enabled
	if server.UseSSHAgent {
		if agentAuth := getSSHAgentAuth(); agentAuth != nil {
			authMethods = append(authMethods, agentAuth)
		}
	}

	// Add manual key authentication if specified
	if server.ManualKeyPath != "" {
		keyAuth, err := getPrivateKeyAuth(server.ManualKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key from %s: %w", server.ManualKeyPath, err)
		}
		authMethods = append(authMethods, keyAuth)
	}

	// Add default key locations as fallback
	defaultKeys := []string{
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ecdsa"),
	}

	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			if keyAuth, err := getPrivateKeyAuth(keyPath); err == nil {
				authMethods = append(authMethods, keyAuth)
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods available (SSH agent: %v, manual key: %s)",
			server.UseSSHAgent, server.ManualKeyPath)
	}

	// Validate that we have at least one viable authentication method
	if err := validateAuthMethods(authMethods); err != nil {
		return nil, fmt.Errorf("authentication validation failed: %w", err)
	}

	config.Auth = authMethods
	return config, nil
}

// getSSHAgentAuth returns SSH agent authentication method if available
func getSSHAgentAuth() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
}

// getPrivateKeyAuth returns private key authentication method
func getPrivateKeyAuth(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// createHostKeyCallback creates a secure host key verification callback
func createHostKeyCallback(server *models.Server) ssh.HostKeyCallback {
	// For deployment scenarios, we need to be more permissive with unknown hosts
	// while still maintaining security logging and host key storage
	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")

	// Check if known_hosts file exists and try to use it, but allow unknown hosts
	if _, err := os.Stat(knownHostsPath); err == nil {
		if strictCallback, err := knownhosts.New(knownHostsPath); err == nil {
			// Wrap the strict callback to handle unknown hosts gracefully
			return createPermissiveHostKeyCallback(strictCallback, server)
		}
	}

	// If known_hosts is not available, create a permissive callback with warnings
	return createHostKeyCallbackWithWarning(server)
}

// createHostKeyAcceptorCallback creates a callback that always accepts and stores host keys
func createHostKeyAcceptorCallback(server *models.Server) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		keyType := key.Type()
		fingerprint := ssh.FingerprintSHA256(key)

		fmt.Printf("Accepting and storing host key for %s (%s)\n", hostname, remote.String())
		fmt.Printf("Host key type: %s\n", keyType)
		fmt.Printf("Host key fingerprint: %s\n", fingerprint)

		// Store the key for future reference
		if err := storeHostKey(hostname, key); err != nil {
			fmt.Printf("Warning: Could not store host key: %v\n", err)
		}

		// Always accept the host key
		return nil
	}
}

// createPermissiveHostKeyCallback wraps a strict callback to allow unknown hosts
func createPermissiveHostKeyCallback(strictCallback ssh.HostKeyCallback, server *models.Server) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// First try the strict callback (checks known_hosts)
		err := strictCallback(hostname, remote, key)
		if err == nil {
			// Host key is known and valid
			fmt.Printf("Host key for %s is known and verified\n", hostname)
			return nil
		}

		// If the error is about unknown host, we'll accept it but log a warning
		if strings.Contains(err.Error(), "key is unknown") {
			keyType := key.Type()
			fingerprint := ssh.FingerprintSHA256(key)

			fmt.Printf("WARNING: Accepting unknown host key for %s (%s)\n", hostname, remote.String())
			fmt.Printf("Host key type: %s\n", keyType)
			fmt.Printf("Host key fingerprint: %s\n", fingerprint)
			fmt.Printf("This host key will be added to known_hosts for future connections\n")

			// Store the key for future reference
			if err := storeHostKey(hostname, key); err != nil {
				fmt.Printf("Warning: Could not store host key: %v\n", err)
			} else {
				fmt.Printf("Host key successfully stored in known_hosts\n")
			}

			// Accept the unknown host key
			return nil
		}

		// For other errors (like key mismatch), still reject but provide helpful info
		fmt.Printf("ERROR: Host key verification failed for %s: %v\n", hostname, err)
		return err
	}
}

// createHostKeyCallbackWithWarning creates a host key callback that accepts keys but logs warnings
func createHostKeyCallbackWithWarning(server *models.Server) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Log the host key for manual verification
		keyType := key.Type()
		fingerprint := ssh.FingerprintSHA256(key)

		// In a production environment, you should log this to a secure location
		// and implement proper host key verification
		fmt.Printf("WARNING: Accepting host key for %s (%s)\n", hostname, remote.String())
		fmt.Printf("Host key type: %s\n", keyType)
		fmt.Printf("Host key fingerprint: %s\n", fingerprint)
		fmt.Printf("Please verify this fingerprint matches the server's host key\n")

		// Optionally, store the key for future reference
		if err := storeHostKey(hostname, key); err != nil {
			fmt.Printf("Warning: Could not store host key: %v\n", err)
		}

		// Accept the key (still not fully secure, but better than InsecureIgnoreHostKey)
		return nil
	}
}

// storeHostKey stores a host key to the known_hosts file for future reference
func storeHostKey(hostname string, key ssh.PublicKey) error {
	if hostname == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if key == nil {
		return fmt.Errorf("key cannot be nil")
	}

	// Get HOME directory
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return fmt.Errorf("HOME environment variable not set")
	}

	// Create .ssh directory with proper permissions
	knownHostsDir := filepath.Join(homeDir, ".ssh")
	if err := os.MkdirAll(knownHostsDir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	knownHostsPath := filepath.Join(knownHostsDir, "known_hosts")

	// Check if the host key already exists in known_hosts
	if exists, err := hostKeyExists(knownHostsPath, hostname, key); err != nil {
		fmt.Printf("Warning: Could not check existing host keys: %v\n", err)
	} else if exists {
		fmt.Printf("Host key for %s already exists in known_hosts\n", hostname)
		return nil
	}

	// Format the host key entry
	keyData := ssh.MarshalAuthorizedKey(key)
	if len(keyData) == 0 {
		return fmt.Errorf("failed to marshal host key")
	}

	entry := fmt.Sprintf("%s %s", hostname, strings.TrimSpace(string(keyData)))

	// Append to known_hosts file with proper error handling
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	if _, err = file.WriteString(entry + "\n"); err != nil {
		return fmt.Errorf("failed to write host key to known_hosts: %w", err)
	}

	fmt.Printf("Successfully added host key for %s to known_hosts\n", hostname)
	return nil
}

// hostKeyExists checks if a host key already exists in the known_hosts file
func hostKeyExists(knownHostsPath, hostname string, key ssh.PublicKey) (bool, error) {
	// Check if known_hosts file exists
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		return false, nil
	}

	// Read the known_hosts file
	file, err := os.Open(knownHostsPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	keyData := ssh.MarshalAuthorizedKey(key)
	expectedEntry := strings.TrimSpace(string(keyData))

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line: hostname keytype keydata [comment]
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[0] == hostname {
			// Check if the key data matches
			if len(parts) >= 3 {
				existingKey := strings.Join(parts[1:], " ")
				if strings.TrimSpace(existingKey) == expectedEntry {
					return true, nil
				}
			}
		}
	}

	return false, scanner.Err()
}

// ExecuteCommand runs a command on the remote server and returns the output
func (sm *SSHManager) ExecuteCommand(command string) (string, error) {
	if err := sm.validateConnection(); err != nil {
		return "", err
	}

	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	session, err := sm.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Set session timeout
	session.Setenv("TMOUT", "300") // 5 minute timeout

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// ExecuteCommandStream runs a command and streams output in real-time
func (sm *SSHManager) ExecuteCommandStream(command string, output chan<- string) error {
	if err := sm.validateConnection(); err != nil {
		return err
	}

	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	if output == nil {
		return fmt.Errorf("output channel cannot be nil")
	}

	session, err := sm.conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Create pipes for stdout and stderr
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output from both stdout and stderr
	done := make(chan error, 2)

	// Stream stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case output <- "[OUT] " + scanner.Text():
			default:
				// Channel is full or closed, skip
			}
		}
		done <- scanner.Err()
	}()

	// Stream stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case output <- "[ERR] " + scanner.Text():
			default:
				// Channel is full or closed, skip
			}
		}
		done <- scanner.Err()
	}()

	// Wait for command to complete
	cmdErr := session.Wait()

	// Wait for both output streams to finish
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil && err != io.EOF {
			// Log stream error but don't fail the command
			select {
			case output <- fmt.Sprintf("[ERR] Stream error: %v", err):
			default:
			}
		}
	}

	return cmdErr
}

// RunCommand is an alias for ExecuteCommand for simpler usage
func (sm *SSHManager) RunCommand(command string) error {
	_, err := sm.ExecuteCommand(command)
	return err
}

// TestConnection verifies the SSH connection is working
func (sm *SSHManager) TestConnection() error {
	output, err := sm.ExecuteCommand("echo 'SSH connection test successful'")
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	if !strings.Contains(output, "SSH connection test successful") {
		return fmt.Errorf("connection test returned unexpected output: %s", output)
	}

	return nil
}

// GetConnectionInfo returns information about the current SSH connection
func (sm *SSHManager) GetConnectionInfo() map[string]interface{} {
	if sm.conn == nil {
		return map[string]interface{}{
			"connected": false,
		}
	}

	return map[string]interface{}{
		"connected":   true,
		"server_host": sm.server.Host,
		"server_port": sm.server.Port,
		"username":    sm.username,
		"is_root":     sm.isRoot,
		"server_name": sm.server.Name,
		"remote_addr": sm.conn.RemoteAddr().String(),
		"local_addr":  sm.conn.LocalAddr().String(),
	}
}

// Close closes the SSH connection
func (sm *SSHManager) Close() error {
	if sm.conn != nil {
		err := sm.conn.Close()
		sm.conn = nil
		return err
	}
	return nil
}

// IsConnected returns true if the SSH connection is active
func (sm *SSHManager) IsConnected() bool {
	return sm.conn != nil
}

// GetUsername returns the username used for this SSH connection
func (sm *SSHManager) GetUsername() string {
	return sm.username
}

// IsRoot returns true if this connection is using root privileges
func (sm *SSHManager) IsRoot() bool {
	return sm.isRoot
}

// GetServer returns the server associated with this SSH manager
func (sm *SSHManager) GetServer() *models.Server {
	return sm.server
}

// AcceptHostKeyForServer pre-accepts the host key for this SSH manager's server
func (sm *SSHManager) AcceptHostKeyForServer() error {
	return AcceptHostKey(sm.server)
}

// SwitchToAppUser switches the SSH manager from root to app user mode
// This is essential after security lockdown when root login is disabled
func (sm *SSHManager) SwitchToAppUser() error {
	if !sm.isRoot {
		return fmt.Errorf("SSH manager is already using app user")
	}

	if sm.server.AppUsername == "" {
		return fmt.Errorf("app username is not configured")
	}

	// Close existing root connection
	if sm.conn != nil {
		sm.conn.Close()
		sm.conn = nil
	}

	// Create new SSH client configuration for app user
	config, err := createSSHConfig(sm.server, sm.server.AppUsername)
	if err != nil {
		return fmt.Errorf("failed to create SSH config for app user: %w", err)
	}

	// Establish new connection as app user with retry logic
	address := fmt.Sprintf("%s:%d", sm.server.Host, sm.server.Port)
	conn, err := establishConnectionWithRetry(address, config, 3)
	if err != nil {
		return fmt.Errorf("failed to connect as app user %s after retries: %w", sm.server.AppUsername, err)
	}

	// Update SSH manager state
	sm.conn = conn
	sm.username = sm.server.AppUsername
	sm.isRoot = false

	// Test the new connection
	if err := sm.TestConnection(); err != nil {
		sm.conn.Close()
		sm.conn = nil
		return fmt.Errorf("app user connection test failed: %w", err)
	}

	return nil
}

// CreateAppUserManager creates a new SSH manager instance using app user credentials
// This is useful when you need a separate app user connection alongside an existing root connection
func (sm *SSHManager) CreateAppUserManager() (*SSHManager, error) {
	if sm.server.AppUsername == "" {
		return nil, fmt.Errorf("app username is not configured")
	}

	// Create new SSH manager as app user
	appManager, err := NewSSHManager(sm.server, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create app user SSH manager: %w", err)
	}

	return appManager, nil
}

// validateServerConfig validates the server configuration
func validateServerConfig(server *models.Server) error {
	if server.Host == "" {
		return fmt.Errorf("server host cannot be empty")
	}

	if server.Port <= 0 || server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", server.Port)
	}

	if server.AppUsername == "" {
		return fmt.Errorf("app username cannot be empty")
	}

	if server.RootUsername == "" {
		return fmt.Errorf("root username cannot be empty")
	}

	return nil
}

// validateAuthMethods validates that authentication methods are viable
func validateAuthMethods(authMethods []ssh.AuthMethod) error {
	if len(authMethods) == 0 {
		return fmt.Errorf("no authentication methods provided")
	}

	// Additional validation could be added here to test auth methods
	// For now, we just ensure we have at least one method
	return nil
}

// validateConnection validates that the SSH connection is still active
func (sm *SSHManager) validateConnection() error {
	if sm.conn == nil {
		return fmt.Errorf("SSH connection is not established")
	}

	// Test if connection is still alive with a simple keepalive
	_, _, err := sm.conn.SendRequest("keepalive@openssh.com", true, nil)
	if err != nil {
		return fmt.Errorf("SSH connection appears to be dead: %w", err)
	}

	return nil
}

// establishConnectionWithRetry attempts to establish SSH connection with retry logic
func establishConnectionWithRetry(address string, config *ssh.ClientConfig, maxRetries int) (*ssh.Client, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		conn, err := ssh.Dial("tcp", address, config)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		if attempt < maxRetries {
			// Wait before retrying (exponential backoff)
			waitTime := time.Duration(attempt*2) * time.Second
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed to establish connection after %d attempts: %w", maxRetries, lastErr)
}
