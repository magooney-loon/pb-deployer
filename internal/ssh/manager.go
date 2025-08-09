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

// NewSSHManager creates a new SSH manager instance and establishes connection
func NewSSHManager(server *models.Server, asRoot bool) (*SSHManager, error) {
	if server == nil {
		return nil, fmt.Errorf("server cannot be nil")
	}

	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	// Create SSH client configuration
	config, err := createSSHConfig(server, username)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH config: %w", err)
	}

	// Establish connection
	address := fmt.Sprintf("%s:%d", server.Host, server.Port)
	conn, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s as %s: %w", address, username, err)
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key verification
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

// ExecuteCommand runs a command on the remote server and returns the output
func (sm *SSHManager) ExecuteCommand(command string) (string, error) {
	if sm.conn == nil {
		return "", fmt.Errorf("SSH connection is not established")
	}

	session, err := sm.conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w (output: %s)", err, string(output))
	}

	return string(output), nil
}

// ExecuteCommandStream runs a command and streams output in real-time
func (sm *SSHManager) ExecuteCommandStream(command string, output chan<- string) error {
	if sm.conn == nil {
		return fmt.Errorf("SSH connection is not established")
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
