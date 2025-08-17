package tunnel

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// AuthConfig holds SSH authentication configuration (SSH agent only)
type AuthConfig struct {
	// KnownHostsFile path to known_hosts file (default: ~/.ssh/known_hosts)
	KnownHostsFile string
}

// GetAuthMethods returns SSH agent authentication methods
func GetAuthMethods(config AuthConfig) ([]ssh.AuthMethod, error) {
	if !IsAgentAvailable() {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "SSH agent is required but not available",
		}
	}

	// Connect to SSH agent
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "failed to connect to SSH agent",
			Cause:   err,
		}
	}

	agentClient := agent.NewClient(sock)

	// Get available keys from agent
	keys, err := agentClient.List()
	if err != nil {
		sock.Close()
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "failed to list SSH agent keys",
			Cause:   err,
		}
	}

	if len(keys) == 0 {
		sock.Close()
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "no keys available in SSH agent",
		}
	}

	return []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)}, nil
}

// GetHostKeyCallback returns strict host key verification callback
func GetHostKeyCallback(config AuthConfig) (ssh.HostKeyCallback, error) {
	// Determine known_hosts file path
	knownHostsPath := config.KnownHostsFile
	if knownHostsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
	}

	// Ensure the known_hosts file exists
	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return nil, fmt.Errorf("failed to ensure known_hosts file: %w", err)
	}

	// Create strict host key callback
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create known_hosts callback: %w", err)
	}

	return callback, nil
}

// ensureKnownHostsFile ensures the known_hosts file and its parent directory exist
func ensureKnownHostsFile(knownHostsPath string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Create the known_hosts file if it doesn't exist
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		file, err := os.Create(knownHostsPath)
		if err != nil {
			return fmt.Errorf("failed to create known_hosts file: %w", err)
		}
		file.Close()

		// Set proper permissions
		if err := os.Chmod(knownHostsPath, 0600); err != nil {
			return fmt.Errorf("failed to set known_hosts permissions: %w", err)
		}
	}

	return nil
}

// IsAgentAvailable checks if SSH agent is available
func IsAgentAvailable() bool {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return false
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// DefaultAuthConfig returns the default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile: "", // Will use ~/.ssh/known_hosts
	}
}
