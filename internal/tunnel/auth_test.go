package tunnel

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestIsValidKnownHostsLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "valid line with RSA key",
			line:     "example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			expected: true,
		},
		{
			name:     "valid line with ECDSA key",
			line:     "192.168.1.1 ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBKnJ+GJIwX3dGzGCOwKGfzJrwh3vhUhT7f7a8GzfxkJBKz8=",
			expected: true,
		},
		{
			name:     "valid line with port",
			line:     "[example.com]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			expected: true,
		},
		{
			name:     "invalid line - missing key data",
			line:     "example.com ssh-rsa",
			expected: false,
		},
		{
			name:     "invalid line - invalid base64",
			line:     "example.com ssh-rsa invalid-base64!",
			expected: false,
		},
		{
			name:     "invalid line - too few parts",
			line:     "example.com",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			expected: false,
		},
		{
			name:     "comment line",
			line:     "# This is a comment",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidKnownHostsLine(tt.line)
			if result != tt.expected {
				t.Errorf("isValidKnownHostsLine(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func TestContainsHostname(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		hostname string
		expected bool
	}{
		{
			name:     "single hostname match",
			line:     "example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			hostname: "example.com",
			expected: true,
		},
		{
			name:     "multiple hostnames match first",
			line:     "example.com,test.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			hostname: "example.com",
			expected: true,
		},
		{
			name:     "multiple hostnames match second",
			line:     "example.com,test.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			hostname: "test.com",
			expected: true,
		},
		{
			name:     "hostname with port",
			line:     "[example.com]:2222 ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			hostname: "example.com",
			expected: true,
		},
		{
			name:     "no match",
			line:     "example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA",
			hostname: "other.com",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			hostname: "example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsHostname(tt.line, tt.hostname)
			if result != tt.expected {
				t.Errorf("containsHostname(%q, %q) = %v, want %v", tt.line, tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestMatchesHostKey(t *testing.T) {
	mockKey := &mockPublicKey{
		keyType: "ssh-rsa",
		keyData: []byte("test-key-data"),
	}

	tests := []struct {
		name     string
		line     string
		hostname string
		key      ssh.PublicKey
		expected bool
	}{
		{
			name:     "matching host and key",
			line:     "example.com ssh-rsa dGVzdC1rZXktZGF0YQ==",
			hostname: "example.com",
			key:      mockKey,
			expected: true,
		},
		{
			name:     "matching host but different key type",
			line:     "example.com ecdsa-sha2-nistp256 dGVzdC1rZXktZGF0YQ==",
			hostname: "example.com",
			key:      mockKey,
			expected: false,
		},
		{
			name:     "different hostname",
			line:     "other.com ssh-rsa dGVzdC1rZXktZGF0YQ==",
			hostname: "example.com",
			key:      mockKey,
			expected: false,
		},
		{
			name:     "invalid key data",
			line:     "example.com ssh-rsa invalid-base64!",
			hostname: "example.com",
			key:      mockKey,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesHostKey(tt.line, tt.hostname, tt.key)
			if result != tt.expected {
				t.Errorf("matchesHostKey(%q, %q, key) = %v, want %v", tt.line, tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestEnsureKnownHostsFile(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsPath := filepath.Join(tempDir, ".ssh", "known_hosts")

	err := ensureKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Fatalf("ensureKnownHostsFile failed: %v", err)
	}

	// Check that directory was created
	sshDir := filepath.Dir(knownHostsPath)
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		t.Error("SSH directory was not created")
	}

	// Check that file was created
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		t.Error("known_hosts file was not created")
	}

	// Check file permissions
	stat, err := os.Stat(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to stat known_hosts file: %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Errorf("Wrong permissions on known_hosts file: got %o, want 0600", stat.Mode().Perm())
	}
}

func TestAddHostKey(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	mockKey := &mockPublicKey{
		keyType: "ssh-rsa",
		keyData: []byte("test-key-data"),
	}

	err := addHostKey(knownHostsPath, "example.com", nil, mockKey, false)
	if err != nil {
		t.Fatalf("addHostKey failed: %v", err)
	}

	// Check that file was created and contains the key
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known_hosts file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "example.com") {
		t.Error("known_hosts file should contain hostname")
	}
	if !strings.Contains(contentStr, "ssh-rsa") {
		t.Error("known_hosts file should contain key type")
	}
}

func TestCleanKnownHostsFile(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := filepath.Join(tempDir, "known_hosts")

	// Create a known_hosts file with mixed valid and invalid entries
	content := `# Comment line
example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA
invalid-line-missing-parts
test.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBKnJ+GJIwX3dGzGCOwKGfzJrwh3vhUhT7f7a8GzfxkJBKz8=
another-invalid-line invalid-base64!
valid.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDf8GKXlsJ8QjKwKw8w9FG7d3vQ5KJdGgRz8w8kF3qG5
`

	err := os.WriteFile(originalPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test known_hosts file: %v", err)
	}

	cleanedPath, cleaned, err := cleanKnownHostsFile(originalPath, false)
	if err != nil {
		t.Fatalf("cleanKnownHostsFile failed: %v", err)
	}
	defer os.Remove(cleanedPath)

	if !cleaned {
		t.Error("Expected file to be marked as cleaned")
	}

	// Read cleaned file
	cleanedContent, err := os.ReadFile(cleanedPath)
	if err != nil {
		t.Fatalf("Failed to read cleaned file: %v", err)
	}

	lines := strings.Split(string(cleanedContent), "\n")
	validLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "invalid") {
			t.Errorf("Cleaned file should not contain invalid lines: %s", line)
		}
		validLines++
	}

	// Should have 3 valid entries (example.com, test.com, valid.com)
	if validLines != 3 {
		t.Errorf("Expected 3 valid lines, got %d", validLines)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	content := "test file content\nwith multiple lines\n"
	err := os.WriteFile(srcPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Check that destination file exists and has correct content
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != content {
		t.Errorf("Destination file content doesn't match source")
	}
}

func TestAuthConfigs(t *testing.T) {
	tests := []struct {
		name   string
		config AuthConfig
	}{
		{
			name:   "DefaultAuthConfig",
			config: DefaultAuthConfig(),
		},
		{
			name:   "DevelopmentAuthConfig",
			config: DevelopmentAuthConfig(),
		},
		{
			name:   "InsecureAuthConfig",
			config: InsecureAuthConfig(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation that config is properly initialized
			if tt.config.MaxAuthAttempts == 0 {
				t.Error("MaxAuthAttempts should be set")
			}
			if tt.config.AuthTimeout == 0 {
				t.Error("AuthTimeout should be set")
			}
		})
	}

	// Test specific values for development config
	devConfig := DevelopmentAuthConfig()
	if !devConfig.AutoAddHostKeys {
		t.Error("DevelopmentAuthConfig should have AutoAddHostKeys enabled")
	}
	if !devConfig.DebugAuth {
		t.Error("DevelopmentAuthConfig should have DebugAuth enabled")
	}

	// Test specific values for insecure config
	insecureConfig := InsecureAuthConfig()
	if !insecureConfig.SkipHostKeyVerification {
		t.Error("InsecureAuthConfig should have SkipHostKeyVerification enabled")
	}
}

func TestGetHostKeyCallback(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	// Create a minimal known_hosts file
	content := "example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA\n"
	err := os.WriteFile(knownHostsPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create known_hosts file: %v", err)
	}

	tests := []struct {
		name   string
		config AuthConfig
	}{
		{
			name: "with known_hosts file",
			config: AuthConfig{
				KnownHostsFile:  knownHostsPath,
				AutoAddHostKeys: false,
				DebugAuth:       false,
			},
		},
		{
			name: "with auto-add enabled",
			config: AuthConfig{
				KnownHostsFile:  knownHostsPath,
				AutoAddHostKeys: true,
				DebugAuth:       false,
			},
		},
		{
			name: "insecure mode",
			config: AuthConfig{
				SkipHostKeyVerification: true,
				DebugAuth:               false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callback, err := GetHostKeyCallback(tt.config)
			if err != nil {
				t.Fatalf("GetHostKeyCallback failed: %v", err)
			}
			if callback == nil {
				t.Error("GetHostKeyCallback returned nil callback")
			}
		})
	}
}

func TestIsAgentAvailable(t *testing.T) {
	// Save original environment
	originalSock := os.Getenv("SSH_AUTH_SOCK")
	defer func() {
		if originalSock != "" {
			os.Setenv("SSH_AUTH_SOCK", originalSock)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	// Test with no SSH_AUTH_SOCK
	os.Unsetenv("SSH_AUTH_SOCK")
	if IsAgentAvailable() {
		t.Error("IsAgentAvailable should return false when SSH_AUTH_SOCK is not set")
	}

	// Test with invalid SSH_AUTH_SOCK
	os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
	if IsAgentAvailable() {
		t.Error("IsAgentAvailable should return false for invalid socket")
	}
}

func TestPublicCleanKnownHostsFile(t *testing.T) {
	tempDir := t.TempDir()
	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	// Create a known_hosts file with some invalid entries
	content := `example.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7vbqajDhA
invalid-line
test.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBKnJ+GJIwX3dGzGCOwKGfzJrwh3vhUhT7f7a8GzfxkJBKz8=
`

	err := os.WriteFile(knownHostsPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = CleanKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Fatalf("CleanKnownHostsFile failed: %v", err)
	}

	// Check that backup was created
	backupFiles, err := filepath.Glob(knownHostsPath + ".backup.*")
	if err != nil {
		t.Fatalf("Failed to check for backup files: %v", err)
	}
	if len(backupFiles) == 0 {
		t.Error("Backup file should have been created")
	}

	// Check that original file was cleaned
	cleanedContent, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read cleaned file: %v", err)
	}
	if strings.Contains(string(cleanedContent), "invalid-line") {
		t.Error("Cleaned file should not contain invalid entries")
	}
}

func TestGetAuthMethods(t *testing.T) {
	// Skip test if SSH agent is not available
	if !IsAgentAvailable() {
		t.Skip("SSH agent not available, skipping test")
	}

	config := AuthConfig{
		DebugAuth:       false,
		MaxAuthAttempts: 3,
		AuthTimeout:     30 * time.Second,
	}

	result, err := GetAuthMethods(config)
	if err != nil {
		t.Fatalf("GetAuthMethods failed: %v", err)
	}

	if result == nil {
		t.Fatal("GetAuthMethods returned nil result")
	}

	if len(result.Methods) == 0 {
		t.Error("GetAuthMethods should return at least one auth method")
	}

	if result.Cleanup == nil {
		t.Error("GetAuthMethods should return cleanup function")
	}

	if !result.Info.AgentAvailable {
		t.Error("AuthInfo should indicate agent is available")
	}

	// Test cleanup
	result.Cleanup()
}

func TestPrioritizeSigners(t *testing.T) {
	// Create mock signers
	rsaKey := &mockPublicKey{keyType: "ssh-rsa", keyData: []byte("rsa-data")}
	ecdsaKey := &mockPublicKey{keyType: "ecdsa-sha2-nistp256", keyData: []byte("ecdsa-data")}
	ed25519Key := &mockPublicKey{keyType: "ssh-ed25519", keyData: []byte("ed25519-data")}

	signers := []ssh.Signer{
		&mockSigner{key: rsaKey},
		&mockSigner{key: ecdsaKey},
		&mockSigner{key: ed25519Key},
	}

	// Test with preferred types
	preferred := []string{"ssh-ed25519", "ecdsa-sha2-nistp256"}
	prioritized := prioritizeSigners(signers, preferred)

	if len(prioritized) != 3 {
		t.Errorf("Expected 3 signers, got %d", len(prioritized))
	}

	// Check that ed25519 and ecdsa are prioritized (order within preferred doesn't matter)
	preferredCount := 0
	for i := 0; i < 2; i++ {
		keyType := prioritized[i].PublicKey().Type()
		if keyType == "ssh-ed25519" || keyType == "ecdsa-sha2-nistp256" {
			preferredCount++
		}
	}
	if preferredCount != 2 {
		t.Errorf("First two signers should be preferred types, got %d preferred", preferredCount)
	}

	// Third should be rsa (non-preferred)
	if prioritized[2].PublicKey().Type() != "ssh-rsa" {
		t.Errorf("Third signer should be rsa, got %s", prioritized[2].PublicKey().Type())
	}
}

func TestDiagnoseAuth(t *testing.T) {
	// Skip test if SSH agent is not available
	if !IsAgentAvailable() {
		t.Skip("SSH agent not available, skipping test")
	}

	info, err := DiagnoseAuth("example.com", "testuser")
	if err != nil {
		t.Fatalf("DiagnoseAuth failed: %v", err)
	}

	if !info.AgentAvailable {
		t.Error("DiagnoseAuth should detect available agent")
	}

	if info.KeysInAgent == 0 {
		t.Error("DiagnoseAuth should detect keys in agent")
	}

	if len(info.KeyTypes) == 0 {
		t.Error("DiagnoseAuth should list key types")
	}
}

// Mock types for testing
type mockPublicKey struct {
	keyType string
	keyData []byte
}

func (m *mockPublicKey) Type() string {
	return m.keyType
}

func (m *mockPublicKey) Marshal() []byte {
	return m.keyData
}

func (m *mockPublicKey) Verify(data []byte, sig *ssh.Signature) error {
	return nil
}

type mockSigner struct {
	key ssh.PublicKey
}

func (m *mockSigner) PublicKey() ssh.PublicKey {
	return m.key
}

func (m *mockSigner) Sign(rand io.Reader, data []byte) (*ssh.Signature, error) {
	return &ssh.Signature{
		Format: m.key.Type(),
		Blob:   []byte("mock-signature"),
	}, nil
}
