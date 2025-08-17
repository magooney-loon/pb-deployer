package tunnel

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// AuthConfig holds SSH authentication configuration (SSH agent only)
type AuthConfig struct {
	// KnownHostsFile path to known_hosts file (default: ~/.ssh/known_hosts)
	KnownHostsFile string
	// SkipHostKeyVerification bypasses host key verification (DANGEROUS - use only for development)
	SkipHostKeyVerification bool
	// AutoAddHostKeys automatically adds unknown host keys to known_hosts
	AutoAddHostKeys bool
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

// GetHostKeyCallback returns host key verification callback with corruption handling
func GetHostKeyCallback(config AuthConfig) (ssh.HostKeyCallback, error) {
	// Skip host key verification if requested (DANGEROUS)
	if config.SkipHostKeyVerification {
		return ssh.InsecureIgnoreHostKey(), nil
	}

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

	// Clean the known_hosts file first
	cleanedPath, err := cleanKnownHostsFile(knownHostsPath)
	if err != nil {
		// If cleaning fails, try with original file
		fmt.Printf("Warning: Failed to clean known_hosts file: %v\n", err)
		cleanedPath = knownHostsPath
	}

	// Create host key callback with the cleaned file
	callback, err := knownhosts.New(cleanedPath)
	if err != nil {
		// If still failing, create a more permissive callback
		return createPermissiveHostKeyCallback(knownHostsPath, config.AutoAddHostKeys), nil
	}

	// If auto-add is enabled, wrap the callback
	if config.AutoAddHostKeys {
		return wrapWithAutoAdd(callback, knownHostsPath), nil
	}

	return callback, nil
}

// cleanKnownHostsFile creates a cleaned version of the known_hosts file
func cleanKnownHostsFile(originalPath string) (string, error) {
	// Read the original file
	file, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	// Create a temporary cleaned file
	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "known_hosts_clean_*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	validLines := 0
	skippedLines := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			tempFile.WriteString(line + "\n")
			continue
		}

		// Validate the line format
		if isValidKnownHostsLine(line) {
			tempFile.WriteString(line + "\n")
			validLines++
		} else {
			fmt.Printf("Skipping corrupted known_hosts line %d: %s\n", lineNum, line)
			skippedLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading known_hosts file: %w", err)
	}

	if skippedLines > 0 {
		fmt.Printf("Cleaned known_hosts: %d valid lines, %d corrupted lines skipped\n", validLines, skippedLines)
	}

	return tempFile.Name(), nil
}

// isValidKnownHostsLine checks if a known_hosts line is valid
func isValidKnownHostsLine(line string) bool {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	// Check if the key data is valid base64
	keyData := parts[2]
	_, err := base64.StdEncoding.DecodeString(keyData)
	return err == nil
}

// createPermissiveHostKeyCallback creates a callback that handles errors gracefully
func createPermissiveHostKeyCallback(knownHostsPath string, autoAdd bool) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fmt.Printf("Checking host key for %s (auto-add: %v)\n", hostname, autoAdd)

		// Get the actual known_hosts file path (not temp file)
		actualKnownHostsPath := knownHostsPath
		if knownHostsPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				if autoAdd {
					fmt.Printf("Failed to get home dir, attempting to add host key anyway\n")
					return addHostKey("/tmp/known_hosts", hostname, remote, key)
				}
				return fmt.Errorf("unable to get home directory: %w", err)
			}
			actualKnownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
		}

		// Try to read and check the known_hosts file manually
		file, err := os.Open(actualKnownHostsPath)
		if err != nil {
			fmt.Printf("Cannot read known_hosts file: %v\n", err)
			if autoAdd {
				fmt.Printf("Adding host key for %s to %s\n", hostname, actualKnownHostsPath)
				return addHostKey(actualKnownHostsPath, hostname, remote, key)
			}
			return fmt.Errorf("unable to read known_hosts file: %w", err)
		}
		defer file.Close()

		hostFound := false
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Check if this line matches our host
			if matchesHost(line, hostname, key) {
				fmt.Printf("Host key verified for %s\n", hostname)
				return nil // Host key matches
			}

			// Check if this line contains our hostname (but maybe with wrong key)
			if containsHostname(line, hostname) {
				hostFound = true
			}
		}

		// Host not found or key mismatch
		if autoAdd {
			if hostFound {
				fmt.Printf("Host %s found but key mismatch, adding new key\n", hostname)
			} else {
				fmt.Printf("Host %s not found, adding to known_hosts\n", hostname)
			}
			return addHostKey(actualKnownHostsPath, hostname, remote, key)
		}

		return fmt.Errorf("host key verification failed: %s not found in known_hosts", hostname)
	}
}

// matchesHost checks if a known_hosts line matches the given hostname and key
func matchesHost(line, hostname string, key ssh.PublicKey) bool {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	hosts := parts[0]
	keyType := parts[1]
	keyData := parts[2]

	// Check if hostname matches
	hostMatches := false
	for _, host := range strings.Split(hosts, ",") {
		hostPart := strings.TrimSpace(host)
		// Handle hostname:port format
		if strings.Contains(hostPart, ":") {
			hostPart = strings.Split(hostPart, ":")[0]
		}
		if hostPart == hostname {
			hostMatches = true
			break
		}
	}

	if !hostMatches {
		return false
	}

	// Check if key matches
	if keyType != key.Type() {
		return false
	}

	expectedKeyData, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return false // Corrupted key data
	}

	actualKeyData := key.Marshal()
	return string(expectedKeyData) == string(actualKeyData)
}

// containsHostname checks if a known_hosts line contains the given hostname
func containsHostname(line, hostname string) bool {
	parts := strings.Fields(line)
	if len(parts) < 1 {
		return false
	}

	hosts := parts[0]
	for _, host := range strings.Split(hosts, ",") {
		hostPart := strings.TrimSpace(host)
		// Handle hostname:port format
		if strings.Contains(hostPart, ":") {
			hostPart = strings.Split(hostPart, ":")[0]
		}
		if hostPart == hostname {
			return true
		}
	}
	return false
}

// addHostKey adds a new host key to the known_hosts file
func addHostKey(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey) error {
	// Ensure the file and directory exist
	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return fmt.Errorf("failed to ensure known_hosts file: %w", err)
	}

	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file for writing: %w", err)
	}
	defer file.Close()

	keyData := base64.StdEncoding.EncodeToString(key.Marshal())
	line := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), keyData)

	_, err = file.WriteString(line)
	if err != nil {
		return fmt.Errorf("failed to write host key: %w", err)
	}

	fmt.Printf("Successfully added host key for %s (%s) to %s\n", hostname, key.Type(), knownHostsPath)
	return nil
}

// wrapWithAutoAdd wraps a host key callback to automatically add unknown hosts
func wrapWithAutoAdd(callback ssh.HostKeyCallback, knownHostsPath string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err != nil {
			// If it's a "not in known_hosts" error, try to add it
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no matching host key") {
				return addHostKey(knownHostsPath, hostname, remote, key)
			}
		}
		return err
	}
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
		KnownHostsFile:          "", // Will use ~/.ssh/known_hosts
		SkipHostKeyVerification: false,
		AutoAddHostKeys:         false,
	}
}

// DevelopmentAuthConfig returns a permissive auth config for development
func DevelopmentAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: false, // Still verify, but be more permissive
		AutoAddHostKeys:         true,  // Automatically add new host keys
	}
}

// InsecureAuthConfig returns auth config that skips all host verification (DANGEROUS)
func InsecureAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: true, // Skip all verification
		AutoAddHostKeys:         false,
	}
}

// CleanKnownHostsFile manually cleans a corrupted known_hosts file
func CleanKnownHostsFile(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}

	// Create backup
	backupPath := path + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
	if err := copyFile(path, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Clean the file
	cleanedPath, err := cleanKnownHostsFile(path)
	if err != nil {
		return fmt.Errorf("failed to clean known_hosts: %w", err)
	}

	// Replace original with cleaned version
	if err := copyFile(cleanedPath, path); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	// Remove temp file
	os.Remove(cleanedPath)

	fmt.Printf("Successfully cleaned known_hosts file. Backup saved as: %s\n", backupPath)
	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	buf := make([]byte, 4096)
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, err := destFile.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err != nil {
			break
		}
	}

	return nil
}
