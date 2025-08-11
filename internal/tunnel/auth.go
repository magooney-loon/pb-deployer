package tunnel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AuthHandler provides authentication method creation and management
type AuthHandler struct {
	tracer SSHTracer
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(tracer SSHTracer) *AuthHandler {
	return &AuthHandler{
		tracer: tracer,
	}
}

// CreateAuthMethod creates an SSH authentication method based on the provided configuration
func (ah *AuthHandler) CreateAuthMethod(auth AuthMethod) (ssh.AuthMethod, error) {
	switch auth.Type {
	case "key":
		return ah.createKeyAuth(auth)
	case "agent":
		return ah.createAgentAuth()
	case "password":
		return ah.createPasswordAuth(auth)
	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", auth.Type)
	}
}

// CreateAuthMethods creates multiple authentication methods with fallback
func (ah *AuthHandler) CreateAuthMethods(auth AuthMethod) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	switch auth.Type {
	case "key":
		// Try specific key first
		if keyAuth, err := ah.createKeyAuth(auth); err == nil {
			methods = append(methods, keyAuth)
		}

		// Try SSH agent as fallback
		if agentAuth, err := ah.createAgentAuth(); err == nil {
			methods = append(methods, agentAuth)
		}

	case "agent":
		// Try SSH agent first
		if agentAuth, err := ah.createAgentAuth(); err == nil {
			methods = append(methods, agentAuth)
		}

		// Try default SSH keys as fallback
		if keyAuth, err := ah.createDefaultKeyAuth(); err == nil {
			methods = append(methods, keyAuth)
		}

	case "password":
		if passwordAuth, err := ah.createPasswordAuth(auth); err == nil {
			methods = append(methods, passwordAuth)
		}

	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", auth.Type)
	}

	if len(methods) == 0 {
		return nil, ErrNoAuthMethod
	}

	return methods, nil
}

// createKeyAuth creates private key authentication
func (ah *AuthHandler) createKeyAuth(auth AuthMethod) (ssh.AuthMethod, error) {
	var signer ssh.Signer
	var err error

	if len(auth.PrivateKey) > 0 {
		// Use provided private key data
		signer, err = ah.parsePrivateKey(auth.PrivateKey, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	} else if auth.KeyPath != "" {
		// Load private key from file
		signer, err = ah.loadPrivateKeyFromFile(auth.KeyPath, "")
		if err != nil {
			return nil, fmt.Errorf("failed to load private key from %s: %w", auth.KeyPath, err)
		}
	} else {
		// Try default SSH key locations
		signer, err = ah.loadDefaultPrivateKey()
		if err != nil {
			return nil, fmt.Errorf("failed to load default private key: %w", err)
		}
	}

	return ssh.PublicKeys(signer), nil
}

// createAgentAuth creates SSH agent authentication
func (ah *AuthHandler) createAgentAuth() (ssh.AuthMethod, error) {
	socket := os.Getenv(EnvSSHAuthSock)
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent at %s: %w", socket, err)
	}

	agentClient := agent.NewClient(conn)

	// Test if agent has any keys
	keys, err := agentClient.List()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to list SSH agent keys: %w", err)
	}

	if len(keys) == 0 {
		conn.Close()
		return nil, fmt.Errorf("no keys available in SSH agent")
	}

	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

// createPasswordAuth creates password authentication
func (ah *AuthHandler) createPasswordAuth(auth AuthMethod) (ssh.AuthMethod, error) {
	if auth.Password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	return ssh.Password(auth.Password), nil
}

// createDefaultKeyAuth creates authentication using default SSH key locations
func (ah *AuthHandler) createDefaultKeyAuth() (ssh.AuthMethod, error) {
	homeDir := os.Getenv(EnvHome)
	if homeDir == "" {
		return nil, fmt.Errorf("HOME environment variable not set")
	}

	// Common SSH key locations
	keyPaths := []string{
		filepath.Join(homeDir, DefaultSSHDir, "id_rsa"),
		filepath.Join(homeDir, DefaultSSHDir, "id_ecdsa"),
		filepath.Join(homeDir, DefaultSSHDir, "id_ed25519"),
	}

	for _, keyPath := range keyPaths {
		if signer, err := ah.loadPrivateKeyFromFile(keyPath, ""); err == nil {
			return ssh.PublicKeys(signer), nil
		}
	}

	return nil, fmt.Errorf("no default SSH keys found")
}

// parsePrivateKey parses a private key with optional passphrase
func (ah *AuthHandler) parsePrivateKey(keyData []byte, passphrase string) (ssh.Signer, error) {
	var signer ssh.Signer
	var err error

	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyData, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(keyData)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return signer, nil
}

// loadPrivateKeyFromFile loads a private key from file
func (ah *AuthHandler) loadPrivateKeyFromFile(keyPath, passphrase string) (ssh.Signer, error) {
	keyData, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", keyPath, err)
	}

	return ah.parsePrivateKey(keyData, passphrase)
}

// loadDefaultPrivateKey attempts to load a private key from default locations
func (ah *AuthHandler) loadDefaultPrivateKey() (ssh.Signer, error) {
	homeDir := os.Getenv(EnvHome)
	if homeDir == "" {
		return nil, fmt.Errorf("HOME environment variable not set")
	}

	keyPaths := []string{
		filepath.Join(homeDir, DefaultSSHDir, DefaultPrivateKeyFile),
		filepath.Join(homeDir, DefaultSSHDir, "id_ecdsa"),
		filepath.Join(homeDir, DefaultSSHDir, "id_ed25519"),
	}

	var lastErr error
	for _, keyPath := range keyPaths {
		if signer, err := ah.loadPrivateKeyFromFile(keyPath, ""); err == nil {
			return signer, nil
		} else {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to load any default private key: %w", lastErr)
	}

	return nil, fmt.Errorf("no default private keys found")
}

// GenerateKeyPair generates a new SSH key pair
func (ah *AuthHandler) GenerateKeyPair(keyType string, bitSize int) ([]byte, []byte, error) {
	switch strings.ToLower(keyType) {
	case "rsa":
		return ah.generateRSAKeyPair(bitSize)
	default:
		return nil, nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// generateRSAKeyPair generates an RSA key pair
func (ah *AuthHandler) generateRSAKeyPair(bitSize int) ([]byte, []byte, error) {
	if bitSize < 2048 {
		bitSize = 2048 // Minimum secure size
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate SSH public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return privateKeyPEM, publicKeyBytes, nil
}

// ValidateKeyPair validates that a private and public key pair match
func (ah *AuthHandler) ValidateKeyPair(privateKeyData, publicKeyData []byte) error {
	// Parse private key
	privateKey, err := ssh.ParsePrivateKey(privateKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Parse public key
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Compare public key from private key with provided public key
	privatePublicKey := privateKey.PublicKey()
	if privatePublicKey.Type() != publicKey.Type() ||
		string(privatePublicKey.Marshal()) != string(publicKey.Marshal()) {
		return fmt.Errorf("private and public keys do not match")
	}

	return nil
}

// GetKeyFingerprint returns the fingerprint of a public key
func (ah *AuthHandler) GetKeyFingerprint(publicKeyData []byte) (string, error) {
	publicKey, _, _, _, err := ssh.ParseAuthorizedKey(publicKeyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	return ssh.FingerprintSHA256(publicKey), nil
}

// SaveKeyPair saves a key pair to files with appropriate permissions
func (ah *AuthHandler) SaveKeyPair(privateKeyData, publicKeyData []byte, basePath string) error {
	privateKeyPath := basePath
	publicKeyPath := basePath + ".pub"

	// Create directory if it doesn't exist
	dir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(dir, SSHDirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Save private key with restricted permissions
	if err := ioutil.WriteFile(privateKeyPath, privateKeyData, PrivateKeyPermissions); err != nil {
		return fmt.Errorf("failed to write private key to %s: %w", privateKeyPath, err)
	}

	// Save public key with normal permissions
	if err := ioutil.WriteFile(publicKeyPath, publicKeyData, FilePermissions); err != nil {
		return fmt.Errorf("failed to write public key to %s: %w", publicKeyPath, err)
	}

	return nil
}

// ListAgentKeys lists all keys available in the SSH agent
func (ah *AuthHandler) ListAgentKeys() ([]*agent.Key, error) {
	socket := os.Getenv(EnvSSHAuthSock)
	if socket == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH agent keys: %w", err)
	}

	return keys, nil
}

// AddKeyToAgent adds a private key to the SSH agent
func (ah *AuthHandler) AddKeyToAgent(privateKeyData []byte, comment string) error {
	socket := os.Getenv(EnvSSHAuthSock)
	if socket == "" {
		return fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer conn.Close()

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(privateKeyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	agentClient := agent.NewClient(conn)

	// Create agent key
	agentKey := agent.AddedKey{
		PrivateKey: signer,
		Comment:    comment,
	}

	if err := agentClient.Add(agentKey); err != nil {
		return fmt.Errorf("failed to add key to SSH agent: %w", err)
	}

	return nil
}

// RemoveKeyFromAgent removes a key from the SSH agent
func (ah *AuthHandler) RemoveKeyFromAgent(publicKey ssh.PublicKey) error {
	socket := os.Getenv(EnvSSHAuthSock)
	if socket == "" {
		return fmt.Errorf("SSH_AUTH_SOCK environment variable not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	if err := agentClient.Remove(publicKey); err != nil {
		return fmt.Errorf("failed to remove key from SSH agent: %w", err)
	}

	return nil
}

// TestAuthentication tests if authentication works with the given method
func (ah *AuthHandler) TestAuthentication(config ConnectionConfig) error {
	authMethods, err := ah.CreateAuthMethods(config.AuthMethod)
	if err != nil {
		return fmt.Errorf("failed to create auth methods: %w", err)
	}

	// Create a minimal SSH config for testing
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For testing only
		Timeout:         DefaultTimeout,
	}

	// Attempt connection
	address := fmt.Sprintf("%s:%d", config.Host, config.Port)
	conn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return fmt.Errorf("authentication test failed: %w", err)
	}
	defer conn.Close()

	return nil
}

// DetectAuthMethod attempts to automatically detect the best authentication method
func (ah *AuthHandler) DetectAuthMethod(username string) (AuthMethod, error) {
	// Try SSH agent first
	if _, err := ah.createAgentAuth(); err == nil {
		return AuthMethod{Type: "agent"}, nil
	}

	// Try default SSH keys
	if _, err := ah.createDefaultKeyAuth(); err == nil {
		return AuthMethod{Type: "key"}, nil
	}

	// Default to key auth (user will need to provide key)
	return AuthMethod{Type: "key"}, fmt.Errorf("no automatic authentication method detected")
}

// GetPublicKeyFromPrivate extracts the public key from a private key
func (ah *AuthHandler) GetPublicKeyFromPrivate(privateKeyData []byte) ([]byte, error) {
	signer, err := ssh.ParsePrivateKey(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	publicKey := signer.PublicKey()
	return ssh.MarshalAuthorizedKey(publicKey), nil
}
