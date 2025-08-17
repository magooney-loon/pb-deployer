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

// AuthConfig holds SSH authentication configuration
type AuthConfig struct {
	KnownHostsFile string
	// SkipHostKeyVerification bypasses host key verification (DANGEROUS)
	SkipHostKeyVerification bool

	AutoAddHostKeys bool
}

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

// GetHostKeyCallback returns host key verification callback
func GetHostKeyCallback(config AuthConfig) (ssh.HostKeyCallback, error) {
	// DANGEROUS: Skip host key verification if requested
	if config.SkipHostKeyVerification {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	knownHostsPath := config.KnownHostsFile
	if knownHostsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
	}

	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return nil, fmt.Errorf("failed to ensure known_hosts file: %w", err)
	}

	// Clean corrupted entries from known_hosts
	cleanedPath, err := cleanKnownHostsFile(knownHostsPath)
	if err != nil {
		// Fallback to original file if cleaning fails
		fmt.Printf("Warning: Failed to clean known_hosts file: %v\n", err)
		cleanedPath = knownHostsPath
	}

	callback, err := knownhosts.New(cleanedPath)
	if err != nil {
		// Fallback to permissive callback if knownhosts.New fails
		return createPermissiveHostKeyCallback(knownHostsPath, config.AutoAddHostKeys), nil
	}

	if config.AutoAddHostKeys {
		return wrapWithAutoAdd(callback, knownHostsPath), nil
	}

	return callback, nil
}

func cleanKnownHostsFile(originalPath string) (string, error) {

	file, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

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

		if line == "" || strings.HasPrefix(line, "#") {
			tempFile.WriteString(line + "\n")
			continue
		}

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

func isValidKnownHostsLine(line string) bool {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	keyData := parts[2]
	_, err := base64.StdEncoding.DecodeString(keyData)
	return err == nil
}

func createPermissiveHostKeyCallback(knownHostsPath string, autoAdd bool) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		fmt.Printf("Checking host key for %s (auto-add: %v)\n", hostname, autoAdd)

		// Use actual known_hosts path, not temp file
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

			if matchesHost(line, hostname, key) {
				fmt.Printf("Host key verified for %s\n", hostname)
				return nil
			}

			if containsHostname(line, hostname) {
				hostFound = true
			}
		}

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

func matchesHost(line, hostname string, key ssh.PublicKey) bool {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return false
	}

	hosts := parts[0]
	keyType := parts[1]
	keyData := parts[2]

	hostMatches := false
	for _, host := range strings.Split(hosts, ",") {
		hostPart := strings.TrimSpace(host)

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

	if keyType != key.Type() {
		return false
	}

	expectedKeyData, err := base64.StdEncoding.DecodeString(keyData)
	if err != nil {
		return false
	}

	actualKeyData := key.Marshal()
	return string(expectedKeyData) == string(actualKeyData)
}

func containsHostname(line, hostname string) bool {
	parts := strings.Fields(line)
	if len(parts) < 1 {
		return false
	}

	hosts := parts[0]
	for _, host := range strings.Split(hosts, ",") {
		hostPart := strings.TrimSpace(host)

		if strings.Contains(hostPart, ":") {
			hostPart = strings.Split(hostPart, ":")[0]
		}
		if hostPart == hostname {
			return true
		}
	}
	return false
}

func addHostKey(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey) error {

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

func wrapWithAutoAdd(callback ssh.HostKeyCallback, knownHostsPath string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no matching host key") {
				return addHostKey(knownHostsPath, hostname, remote, key)
			}
		}
		return err
	}
}

func ensureKnownHostsFile(knownHostsPath string) error {

	dir := filepath.Dir(knownHostsPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		file, err := os.Create(knownHostsPath)
		if err != nil {
			return fmt.Errorf("failed to create known_hosts file: %w", err)
		}
		file.Close()

		if err := os.Chmod(knownHostsPath, 0600); err != nil {
			return fmt.Errorf("failed to set known_hosts permissions: %w", err)
		}
	}

	return nil
}

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

func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: false,
		AutoAddHostKeys:         false,
	}
}

func DevelopmentAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: false,
		AutoAddHostKeys:         true,
	}
}

// InsecureAuthConfig skips all host verification (DANGEROUS)
func InsecureAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: true,
		AutoAddHostKeys:         false,
	}
}

func CleanKnownHostsFile(path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}

	backupPath := path + ".backup." + fmt.Sprintf("%d", time.Now().Unix())
	if err := copyFile(path, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	cleanedPath, err := cleanKnownHostsFile(path)
	if err != nil {
		return fmt.Errorf("failed to clean known_hosts: %w", err)
	}

	if err := copyFile(cleanedPath, path); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	os.Remove(cleanedPath)

	fmt.Printf("Successfully cleaned known_hosts file. Backup saved as: %s\n", backupPath)
	return nil
}

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
