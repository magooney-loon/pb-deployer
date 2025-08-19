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

type AuthConfig struct {
	KnownHostsFile          string
	SkipHostKeyVerification bool
	AutoAddHostKeys         bool
	DebugAuth               bool
	PreferredKeyTypes       []string
	MaxAuthAttempts         int
	AuthTimeout             time.Duration
}

type AuthResult struct {
	Methods []ssh.AuthMethod
	Cleanup func()
	Info    AuthInfo
}

type AuthInfo struct {
	AgentAvailable   bool
	KeysInAgent      int
	KeyTypes         []string
	HostInKnownHosts bool
	AuthMethod       string
}

func GetAuthMethods(config AuthConfig) (*AuthResult, error) {
	result := &AuthResult{
		Info: AuthInfo{},
	}

	if config.DebugAuth {
		fmt.Printf("[AUTH] Starting authentication process\n")
	}

	// Check SSH agent availability
	if !IsAgentAvailable() {
		if config.DebugAuth {
			fmt.Printf("[AUTH] SSH agent not available\n")
		}
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "SSH agent is required but not available. Please run: eval $(ssh-agent) && ssh-add",
		}
	}

	result.Info.AgentAvailable = true
	if config.DebugAuth {
		fmt.Printf("[AUTH] SSH agent is available\n")
	}

	// Connect to SSH agent with timeout
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "SSH_AUTH_SOCK environment variable not set",
		}
	}

	conn, err := net.DialTimeout("unix", sock, 5*time.Second)
	if err != nil {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "failed to connect to SSH agent",
			Cause:   err,
		}
	}

	agentClient := agent.NewClient(conn)

	// List available keys
	keys, err := agentClient.List()
	if err != nil {
		conn.Close()
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "failed to list SSH agent keys",
			Cause:   err,
		}
	}

	if len(keys) == 0 {
		conn.Close()
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "no keys available in SSH agent. Please add keys with: ssh-add ~/.ssh/id_rsa",
		}
	}

	result.Info.KeysInAgent = len(keys)
	for _, key := range keys {
		result.Info.KeyTypes = append(result.Info.KeyTypes, key.Type())
	}

	if config.DebugAuth {
		fmt.Printf("[AUTH] Found %d keys in agent: %v\n", len(keys), result.Info.KeyTypes)
	}

	// Create cleanup function
	result.Cleanup = func() {
		if conn != nil {
			conn.Close()
		}
	}

	// Create authentication methods with prioritization
	var authMethods []ssh.AuthMethod

	// Primary method: SSH agent keys
	signers, err := agentClient.Signers()
	if err != nil {
		result.Cleanup()
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "failed to get signers from SSH agent",
			Cause:   err,
		}
	}

	// Filter and prioritize signers based on preferred key types
	prioritizedSigners := prioritizeSigners(signers, config.PreferredKeyTypes)

	authMethods = append(authMethods, ssh.PublicKeys(prioritizedSigners...))
	result.Info.AuthMethod = "ssh-agent"

	if config.DebugAuth {
		fmt.Printf("[AUTH] Configured %d authentication methods\n", len(authMethods))
	}

	result.Methods = authMethods
	return result, nil
}

func GetHostKeyCallback(config AuthConfig) (ssh.HostKeyCallback, error) {
	if config.DebugAuth {
		fmt.Printf("[AUTH] Setting up host key verification\n")
	}

	// DANGEROUS: Skip host key verification if requested
	if config.SkipHostKeyVerification {
		if config.DebugAuth {
			fmt.Printf("[AUTH] WARNING: Skipping host key verification (INSECURE)\n")
		}
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

	if config.DebugAuth {
		fmt.Printf("[AUTH] Using known_hosts file: %s\n", knownHostsPath)
	}

	// Ensure known_hosts file exists
	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return nil, fmt.Errorf("failed to ensure known_hosts file: %w", err)
	}

	// Clean and validate known_hosts file
	cleanedPath, cleaned, err := cleanKnownHostsFile(knownHostsPath, config.DebugAuth)
	if err != nil {
		if config.DebugAuth {
			fmt.Printf("[AUTH] Warning: Failed to clean known_hosts file: %v\n", err)
		}
		cleanedPath = knownHostsPath
	} else if cleaned && config.DebugAuth {
		fmt.Printf("[AUTH] Cleaned known_hosts file\n")
	}

	// Ensure cleanup of temp file if different from original
	defer func() {
		if cleanedPath != knownHostsPath {
			os.Remove(cleanedPath)
		}
	}()

	// Try to create knownhosts callback
	callback, err := knownhosts.New(cleanedPath)
	if err != nil {
		if config.DebugAuth {
			fmt.Printf("[AUTH] knownhosts.New failed: %v, using permissive callback\n", err)
		}
		return createEnhancedHostKeyCallback(knownHostsPath, config), nil
	}

	if config.AutoAddHostKeys {
		if config.DebugAuth {
			fmt.Printf("[AUTH] Auto-add host keys enabled\n")
		}
		return wrapWithAutoAdd(callback, knownHostsPath, config.DebugAuth), nil
	}

	return callback, nil
}

func cleanKnownHostsFile(originalPath string, debug bool) (string, bool, error) {
	file, err := os.Open(originalPath)
	if err != nil {
		return "", false, fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "known_hosts_clean_*.tmp")
	if err != nil {
		return "", false, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFileName := tempFile.Name()

	defer func() {
		tempFile.Close()
		if err != nil {
			os.Remove(tempFileName)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	validLines := 0
	skippedLines := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Keep empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			tempFile.WriteString(line + "\n")
			continue
		}

		if isValidKnownHostsLine(line) {
			tempFile.WriteString(line + "\n")
			validLines++
		} else {
			if debug {
				fmt.Printf("[AUTH] Skipping corrupted known_hosts line %d: %s\n", lineNum, line)
			}
			skippedLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("error reading known_hosts file: %w", err)
	}

	cleaned := skippedLines > 0
	if cleaned && debug {
		fmt.Printf("[AUTH] Cleaned known_hosts: %d valid lines, %d corrupted lines skipped\n", validLines, skippedLines)
	}

	return tempFileName, cleaned, nil
}

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

func createEnhancedHostKeyCallback(knownHostsPath string, config AuthConfig) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if config.DebugAuth {
			fmt.Printf("[AUTH] Verifying host key for %s (type: %s)\n", hostname, key.Type())
		}

		// Determine actual known_hosts path
		actualKnownHostsPath := knownHostsPath
		if knownHostsPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				if config.AutoAddHostKeys {
					if config.DebugAuth {
						fmt.Printf("[AUTH] Failed to get home dir, using temp file\n")
					}
					return addHostKey("/tmp/known_hosts", hostname, remote, key, config.DebugAuth)
				}
				return fmt.Errorf("unable to get home directory: %w", err)
			}
			actualKnownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
		}

		// Check if host exists in known_hosts
		hostFound, keyMatches, err := checkHostInKnownHosts(actualKnownHostsPath, hostname, key, config.DebugAuth)
		if err != nil && !config.AutoAddHostKeys {
			return fmt.Errorf("unable to read known_hosts file: %w", err)
		}

		if keyMatches {
			if config.DebugAuth {
				fmt.Printf("[AUTH] Host key verified for %s\n", hostname)
			}
			return nil
		}

		if config.AutoAddHostKeys {
			if hostFound {
				if config.DebugAuth {
					fmt.Printf("[AUTH] Host %s found but key mismatch, adding new key\n", hostname)
				}
			} else {
				if config.DebugAuth {
					fmt.Printf("[AUTH] Host %s not found, adding to known_hosts\n", hostname)
				}
			}
			return addHostKey(actualKnownHostsPath, hostname, remote, key, config.DebugAuth)
		}

		if hostFound {
			return fmt.Errorf("host key verification failed: key mismatch for %s", hostname)
		}
		return fmt.Errorf("host key verification failed: %s not found in known_hosts", hostname)
	}
}

func checkHostInKnownHosts(knownHostsPath, hostname string, key ssh.PublicKey, debug bool) (hostFound, keyMatches bool, err error) {
	file, err := os.Open(knownHostsPath)
	if err != nil {
		return false, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if containsHostname(line, hostname) {
			hostFound = true
			if matchesHostKey(line, hostname, key) {
				keyMatches = true
				return hostFound, keyMatches, nil
			}
		}
	}

	if debug && hostFound && !keyMatches {
		fmt.Printf("[AUTH] Host %s found in known_hosts but key doesn't match\n", hostname)
	}

	return hostFound, keyMatches, scanner.Err()
}

func matchesHostKey(line, hostname string, key ssh.PublicKey) bool {
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

		// Handle bracket notation for ports: [hostname]:port
		if strings.HasPrefix(hostPart, "[") && strings.Contains(hostPart, "]:") {
			// Extract hostname from [hostname]:port
			if idx := strings.Index(hostPart, "]:"); idx > 1 {
				hostPart = hostPart[1:idx]
			}
		} else if strings.Contains(hostPart, ":") {
			// Handle standard notation: hostname:port
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

	// Check key type
	if keyType != key.Type() {
		return false
	}

	// Check key data
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

		// Handle bracket notation for ports: [hostname]:port
		if strings.HasPrefix(hostPart, "[") && strings.Contains(hostPart, "]:") {
			// Extract hostname from [hostname]:port
			if idx := strings.Index(hostPart, "]:"); idx > 1 {
				hostPart = hostPart[1:idx]
			}
		} else if strings.Contains(hostPart, ":") {
			// Handle standard notation: hostname:port
			hostPart = strings.Split(hostPart, ":")[0]
		}

		if hostPart == hostname {
			return true
		}
	}
	return false
}

func addHostKey(knownHostsPath, hostname string, remote net.Addr, key ssh.PublicKey, debug bool) error {
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

	if debug {
		fmt.Printf("[AUTH] Successfully added host key for %s (%s) to %s\n", hostname, key.Type(), knownHostsPath)
	}
	return nil
}

func wrapWithAutoAdd(callback ssh.HostKeyCallback, knownHostsPath string, debug bool) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		if err != nil {
			if strings.Contains(err.Error(), "not found") ||
				strings.Contains(err.Error(), "no matching host key") ||
				strings.Contains(err.Error(), "key is unknown") {
				if debug {
					fmt.Printf("[AUTH] Auto-adding unknown host key for %s\n", hostname)
				}
				return addHostKey(knownHostsPath, hostname, remote, key, debug)
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

func prioritizeSigners(signers []ssh.Signer, preferredTypes []string) []ssh.Signer {
	if len(preferredTypes) == 0 {
		return signers
	}

	var prioritized []ssh.Signer
	var others []ssh.Signer

	for _, signer := range signers {
		preferred := false
		for _, prefType := range preferredTypes {
			if signer.PublicKey().Type() == prefType {
				prioritized = append(prioritized, signer)
				preferred = true
				break
			}
		}
		if !preferred {
			others = append(others, signer)
		}
	}

	return append(prioritized, others...)
}

func IsAgentAvailable() bool {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return false
	}

	conn, err := net.DialTimeout("unix", sock, 2*time.Second)
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
		DebugAuth:               false,
		MaxAuthAttempts:         3,
		AuthTimeout:             30 * time.Second,
	}
}

func DevelopmentAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: false,
		AutoAddHostKeys:         true,
		DebugAuth:               true,
		MaxAuthAttempts:         3,
		AuthTimeout:             30 * time.Second,
	}
}

func InsecureAuthConfig() AuthConfig {
	return AuthConfig{
		KnownHostsFile:          "",
		SkipHostKeyVerification: true,
		AutoAddHostKeys:         false,
		DebugAuth:               true,
		MaxAuthAttempts:         3,
		AuthTimeout:             30 * time.Second,
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

	cleanedPath, cleaned, err := cleanKnownHostsFile(path, true)
	if err != nil {
		return fmt.Errorf("failed to clean known_hosts: %w", err)
	}

	if !cleaned {
		os.Remove(backupPath) // Remove backup if no changes were made
		fmt.Printf("Known_hosts file was already clean\n")
		return nil
	}

	if err := copyFile(cleanedPath, path); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	if err := os.Remove(cleanedPath); err != nil {
		fmt.Printf("Warning: Failed to remove temporary file %s: %v\n", cleanedPath, err)
	}

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

// DiagnoseAuth performs comprehensive authentication diagnostics
func DiagnoseAuth(host, user string) (*AuthInfo, error) {
	info := &AuthInfo{}

	// Check SSH agent
	info.AgentAvailable = IsAgentAvailable()
	if !info.AgentAvailable {
		return info, fmt.Errorf("SSH agent not available")
	}

	// Get keys from agent
	sock := os.Getenv("SSH_AUTH_SOCK")
	conn, err := net.DialTimeout("unix", sock, 5*time.Second)
	if err != nil {
		return info, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return info, fmt.Errorf("failed to list SSH agent keys: %w", err)
	}

	info.KeysInAgent = len(keys)
	for _, key := range keys {
		info.KeyTypes = append(info.KeyTypes, key.Type())
	}

	// Check known_hosts
	home, err := os.UserHomeDir()
	if err == nil {
		knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
		if _, err := os.Stat(knownHostsPath); err == nil {
			hostFound, _, _ := checkHostInKnownHosts(knownHostsPath, host, nil, false)
			info.HostInKnownHosts = hostFound
		}
	}

	return info, nil
}
