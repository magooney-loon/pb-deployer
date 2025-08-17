package tunnel

import (
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// AuthConfig holds SSH authentication configuration
type AuthConfig struct {
	// UseAgent attempts to use SSH agent if available
	UseAgent bool
	// KeyPath is the path to a private key file (optional)
	KeyPath string
	// KeyPassphrase is the passphrase for encrypted keys (optional)
	KeyPassphrase string
	// PreferAgent prioritizes agent over key file when both available
	PreferAgent bool
	// HostKeyVerification configures host key verification
	HostKeyVerification HostKeyConfig
}

// HostKeyConfig configures host key verification behavior
type HostKeyConfig struct {
	// Mode determines how host keys are verified
	Mode HostKeyMode
	// KnownHostsFile path to known_hosts file (default: ~/.ssh/known_hosts)
	KnownHostsFile string
	// StrictHostKeyChecking enforces strict host key checking
	StrictHostKeyChecking bool
	// AcceptNewKeys automatically accepts new host keys and adds them to known_hosts
	AcceptNewKeys bool
	// HostKeyCallback custom host key callback function
	HostKeyCallback ssh.HostKeyCallback
}

// HostKeyMode defines different host key verification modes
type HostKeyMode int

const (
	// HostKeyModeStrict uses known_hosts file with strict checking
	HostKeyModeStrict HostKeyMode = iota
	// HostKeyModeAcceptNew accepts new keys and adds them to known_hosts
	HostKeyModeAcceptNew
	// HostKeyModeInsecure disables host key verification (NOT RECOMMENDED)
	HostKeyModeInsecure
	// HostKeyModeCustom uses a custom callback function
	HostKeyModeCustom
)

// GetAuthMethods returns SSH authentication methods based on configuration
func GetAuthMethods(config AuthConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	var agentMethods []ssh.AuthMethod
	var keyMethods []ssh.AuthMethod
	var err error

	// Try SSH agent first if enabled
	if config.UseAgent {
		agentMethods, err = getAgentAuthMethods()
		if err == nil && len(agentMethods) > 0 {
			if config.PreferAgent {
				methods = append(methods, agentMethods...)
			}
		}
	}

	// Try key file if specified
	if config.KeyPath != "" {
		keyMethods, err = getKeyFileAuthMethods(config.KeyPath, config.KeyPassphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to load key file: %w", err)
		}
		methods = append(methods, keyMethods...)
	}

	// Add agent methods if not already added and available
	if config.UseAgent && !config.PreferAgent && len(agentMethods) > 0 {
		methods = append(methods, agentMethods...)
	}

	// Auto-detect keys if no specific key provided
	if config.KeyPath == "" && (!config.UseAgent || len(agentMethods) == 0) {
		autoMethods, err := getAutoDetectedKeyMethods(config.KeyPassphrase)
		if err == nil {
			methods = append(methods, autoMethods...)
		}
	}

	if len(methods) == 0 {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "no valid SSH authentication methods available",
		}
	}

	return methods, nil
}

// GetHostKeyCallback returns the appropriate host key callback based on configuration
func GetHostKeyCallback(config HostKeyConfig) (ssh.HostKeyCallback, error) {
	// Use custom callback if provided
	if config.HostKeyCallback != nil && config.Mode == HostKeyModeCustom {
		return config.HostKeyCallback, nil
	}

	switch config.Mode {
	case HostKeyModeInsecure:
		return ssh.InsecureIgnoreHostKey(), nil

	case HostKeyModeStrict:
		return createKnownHostsCallback(config, false)

	case HostKeyModeAcceptNew:
		return createKnownHostsCallback(config, true)

	default:
		return createKnownHostsCallback(config, false)
	}
}

// createKnownHostsCallback creates a host key callback using known_hosts file
func createKnownHostsCallback(config HostKeyConfig, acceptNew bool) (ssh.HostKeyCallback, error) {
	// Determine known_hosts file path
	knownHostsPath := config.KnownHostsFile
	if knownHostsPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsPath = filepath.Join(home, ".ssh", "known_hosts")
	}

	// Expand tilde if present
	if strings.HasPrefix(knownHostsPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsPath = filepath.Join(home, knownHostsPath[2:])
	}

	// Ensure the known_hosts file and directory exist
	if err := ensureKnownHostsFile(knownHostsPath); err != nil {
		return nil, fmt.Errorf("failed to ensure known_hosts file: %w", err)
	}

	// Create the host key callback
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create known_hosts callback: %w", err)
	}

	// If acceptNew is enabled, wrap the callback to handle new keys
	if acceptNew {
		return createAcceptNewKeysCallback(callback, knownHostsPath), nil
	}

	return callback, nil
}

// createAcceptNewKeysCallback wraps a known_hosts callback to accept new keys
func createAcceptNewKeysCallback(baseCallback ssh.HostKeyCallback, knownHostsPath string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := baseCallback(hostname, remote, key)
		if err != nil {
			// Check if this is a "key not found" error
			if knownHostsErr, ok := err.(*knownhosts.KeyError); ok && len(knownHostsErr.Want) == 0 {
				// Key not found, add it to known_hosts
				if addErr := addHostKey(knownHostsPath, hostname, key); addErr != nil {
					return fmt.Errorf("failed to add host key: %w", addErr)
				}
				return nil // Accept the key
			}
			// Other errors (like key mismatch) should still fail
			return err
		}
		return nil
	}
}

// addHostKey adds a host key to the known_hosts file
func addHostKey(knownHostsPath, hostname string, key ssh.PublicKey) error {
	// Format the host key entry
	keyLine := fmt.Sprintf("%s %s %s\n", hostname, key.Type(), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))

	// Append to known_hosts file
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(keyLine); err != nil {
		return fmt.Errorf("failed to write to known_hosts file: %w", err)
	}

	return nil
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

// getAgentAuthMethods returns SSH agent authentication methods
func getAgentAuthMethods() ([]ssh.AuthMethod, error) {
	// Try to connect to SSH agent
	sock, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(sock)

	// Get available keys from agent
	keys, err := agentClient.List()
	if err != nil {
		sock.Close()
		return nil, fmt.Errorf("failed to list SSH agent keys: %w", err)
	}

	if len(keys) == 0 {
		sock.Close()
		return nil, fmt.Errorf("no keys available in SSH agent")
	}

	// Create auth method using agent
	return []ssh.AuthMethod{ssh.PublicKeysCallback(agentClient.Signers)}, nil
}

// getKeyFileAuthMethods returns authentication methods from a key file
func getKeyFileAuthMethods(keyPath, passphrase string) ([]ssh.AuthMethod, error) {
	// Expand tilde in path
	if strings.HasPrefix(keyPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}

	// Read key file
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", keyPath, err)
	}

	// Parse the key
	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}

// getAutoDetectedKeyMethods attempts to auto-detect SSH keys in standard locations
func getAutoDetectedKeyMethods(passphrase string) ([]ssh.AuthMethod, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(home, ".ssh")

	// Standard key file names to check
	keyFiles := []string{
		"id_rsa",
		"id_ecdsa",
		"id_ed25519",
		"id_dsa",
	}

	var signers []ssh.Signer

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)

		// Check if key file exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			continue
		}

		// Try to load the key
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			continue // Skip this key and try the next one
		}

		var signer ssh.Signer
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(keyData)
		}

		if err != nil {
			// Try without passphrase if it failed with passphrase
			if passphrase != "" {
				signer, err = ssh.ParsePrivateKey(keyData)
			}
			if err != nil {
				continue // Skip this key
			}
		}

		signers = append(signers, signer)
	}

	if len(signers) == 0 {
		return nil, fmt.Errorf("no valid SSH keys found in %s", sshDir)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(signers...)}, nil
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

// ValidateKeyFile checks if a key file is valid and readable
func ValidateKeyFile(keyPath string) error {
	// Expand tilde in path
	if strings.HasPrefix(keyPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}

	// Check if file exists
	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("key file does not exist: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("key path is not a regular file")
	}

	// Check file permissions (should not be world readable)
	if info.Mode().Perm()&0044 != 0 {
		return fmt.Errorf("key file has unsafe permissions (readable by group/others)")
	}

	// Try to read and parse the key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Check if it looks like a private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return fmt.Errorf("key file does not contain valid PEM data")
	}

	// Check for common private key types
	validTypes := []string{
		"RSA PRIVATE KEY",
		"EC PRIVATE KEY",
		"DSA PRIVATE KEY",
		"OPENSSH PRIVATE KEY",
		"PRIVATE KEY", // PKCS#8
	}

	isValidType := false
	for _, validType := range validTypes {
		if block.Type == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return fmt.Errorf("key file does not contain a recognized private key type")
	}

	return nil
}

// GetKeyFingerprint returns the fingerprint of a public key file
func GetKeyFingerprint(keyPath string) (string, error) {
	// Try to read corresponding public key file first
	pubKeyPath := keyPath + ".pub"
	if pubData, err := os.ReadFile(pubKeyPath); err == nil {
		return getFingerprintFromPublicKey(pubData)
	}

	// If no .pub file, extract from private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.FingerprintSHA256(signer.PublicKey()), nil
}

// getFingerprintFromPublicKey extracts fingerprint from public key data
func getFingerprintFromPublicKey(pubData []byte) (string, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	return ssh.FingerprintSHA256(pubKey), nil
}

// DefaultAuthConfig returns a sensible default authentication configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		UseAgent:            IsAgentAvailable(),
		PreferAgent:         true,
		HostKeyVerification: DefaultHostKeyConfig(),
	}
}

// DefaultHostKeyConfig returns a sensible default host key configuration
func DefaultHostKeyConfig() HostKeyConfig {
	return HostKeyConfig{
		Mode:                  HostKeyModeAcceptNew,
		StrictHostKeyChecking: false,
		AcceptNewKeys:         true,
	}
}

// StrictHostKeyConfig returns a strict host key configuration for production
func StrictHostKeyConfig() HostKeyConfig {
	return HostKeyConfig{
		Mode:                  HostKeyModeStrict,
		StrictHostKeyChecking: true,
		AcceptNewKeys:         false,
	}
}

// InsecureHostKeyConfig returns an insecure host key configuration (NOT RECOMMENDED)
func InsecureHostKeyConfig() HostKeyConfig {
	return HostKeyConfig{
		Mode: HostKeyModeInsecure,
	}
}

// AuthConfigFromKeyPath creates auth config for a specific key path
func AuthConfigFromKeyPath(keyPath string, passphrase ...string) AuthConfig {
	config := AuthConfig{
		UseAgent:            false,
		KeyPath:             keyPath,
		PreferAgent:         false,
		HostKeyVerification: DefaultHostKeyConfig(),
	}

	if len(passphrase) > 0 {
		config.KeyPassphrase = passphrase[0]
	}

	return config
}

// AuthConfigWithAgent creates auth config that prefers SSH agent
func AuthConfigWithAgent() AuthConfig {
	return AuthConfig{
		UseAgent:            true,
		PreferAgent:         true,
		HostKeyVerification: DefaultHostKeyConfig(),
	}
}

// AuthConfigWithFallback creates auth config that tries agent first, then key file
func AuthConfigWithFallback(keyPath string, passphrase ...string) AuthConfig {
	config := AuthConfig{
		UseAgent:            IsAgentAvailable(),
		KeyPath:             keyPath,
		PreferAgent:         true,
		HostKeyVerification: DefaultHostKeyConfig(),
	}

	if len(passphrase) > 0 {
		config.KeyPassphrase = passphrase[0]
	}

	return config
}

// AuthConfigFromEnv creates auth config from environment variables
// Looks for SSH_KEY_PATH, SSH_KEY_PASSPHRASE, SSH_AUTH_SOCK, and SSH_KNOWN_HOSTS
func AuthConfigFromEnv() AuthConfig {
	config := AuthConfig{
		UseAgent:            IsAgentAvailable(),
		PreferAgent:         true,
		HostKeyVerification: DefaultHostKeyConfig(),
	}

	if keyPath := os.Getenv("SSH_KEY_PATH"); keyPath != "" {
		config.KeyPath = keyPath
		if passphrase := os.Getenv("SSH_KEY_PASSPHRASE"); passphrase != "" {
			config.KeyPassphrase = passphrase
		}
	}

	// Configure host key verification from environment
	if knownHosts := os.Getenv("SSH_KNOWN_HOSTS"); knownHosts != "" {
		config.HostKeyVerification.KnownHostsFile = knownHosts
	}
	if strictHost := os.Getenv("SSH_STRICT_HOST_KEY_CHECKING"); strictHost == "yes" || strictHost == "true" {
		config.HostKeyVerification = StrictHostKeyConfig()
	}

	return config
}

// AuthConfigAutoDetect attempts to automatically detect the best authentication method
func AuthConfigAutoDetect() AuthConfig {
	// Check if agent is available first
	if IsAgentAvailable() {
		return AuthConfigWithAgent()
	}

	// Try to find a default key
	home, err := os.UserHomeDir()
	if err != nil {
		return DefaultAuthConfig()
	}

	sshDir := filepath.Join(home, ".ssh")
	keyFiles := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		if err := ValidateKeyFile(keyPath); err == nil {
			return AuthConfigFromKeyPath(keyPath)
		}
	}

	return DefaultAuthConfig()
}

// AuthConfigSecure creates a secure auth config for production use
func AuthConfigSecure(keyPath string, passphrase ...string) AuthConfig {
	config := AuthConfigFromKeyPath(keyPath, passphrase...)
	config.HostKeyVerification = StrictHostKeyConfig()
	return config
}

// ValidateHostKey validates a host key against known_hosts
func ValidateHostKey(hostname string, key ssh.PublicKey, knownHostsPath ...string) error {
	var knownHosts string
	if len(knownHostsPath) > 0 {
		knownHosts = knownHostsPath[0]
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHosts = filepath.Join(home, ".ssh", "known_hosts")
	}

	callback, err := knownhosts.New(knownHosts)
	if err != nil {
		return fmt.Errorf("failed to create known_hosts callback: %w", err)
	}

	// Create a dummy remote address for validation
	addr, _ := net.ResolveTCPAddr("tcp", hostname+":22")
	return callback(hostname, addr, key)
}

// MustGetAuthMethods is like GetAuthMethods but panics on error
func MustGetAuthMethods(config AuthConfig) []ssh.AuthMethod {
	methods, err := GetAuthMethods(config)
	if err != nil {
		panic(fmt.Sprintf("failed to get auth methods: %v", err))
	}
	return methods
}
