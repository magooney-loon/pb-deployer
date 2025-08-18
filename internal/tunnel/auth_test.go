package tunnel

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestIsValidKnownHostsLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected bool
	}{
		{
			name:     "valid line with hostname",
			line:     "github.com ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+PXYPCPy6rbTrTtw7PHkccKrpp0yVhp5HdEIcKr6pLlVDBfOLX9QUsyCOV0wzfjIJNlGEYsdlLJizHhbn2mUjvSAHQqZETYP81eFzLQNnPHt4EVVUh7VfDESU84KezmD5QlWpXLmvU31/yMf+Se8xhHTvKSCZIFImWwoG6mbUoWf9nzpIoaSjB+weqqUUmpaaasXVal72J+UX2B+2RPW3RcT0eOzQgqlJL3RKrTJvdsjE3JEAvGq3lGHSZXy28G3skua2SmVi/w4yCE6gbODqnTWlg7+wC604ydGXA8VJiS5ap43JXiUFFAaQ==",
			expected: true,
		},
		{
			name:     "valid line with IP and port",
			line:     "192.168.1.1:22 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOM3ngl2WP9LM0/LWMfRhwKNKPvkiX+GVJC9BHPILvXq",
			expected: true,
		},
		{
			name:     "invalid line with too few fields",
			line:     "github.com ssh-rsa",
			expected: false,
		},
		{
			name:     "invalid line with bad base64",
			line:     "github.com ssh-rsa invalid-base64-key!!!",
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
			line:     "github.com ssh-rsa AAAAB3NzaC1yc2E...",
			hostname: "github.com",
			expected: true,
		},
		{
			name:     "multiple hostnames match first",
			line:     "github.com,gitlab.com ssh-rsa AAAAB3NzaC1yc2E...",
			hostname: "github.com",
			expected: true,
		},
		{
			name:     "multiple hostnames match second",
			line:     "github.com,gitlab.com ssh-rsa AAAAB3NzaC1yc2E...",
			hostname: "gitlab.com",
			expected: true,
		},
		{
			name:     "hostname with port",
			line:     "example.com:2222 ssh-rsa AAAAB3NzaC1yc2E...",
			hostname: "example.com",
			expected: true,
		},
		{
			name:     "no match",
			line:     "github.com ssh-rsa AAAAB3NzaC1yc2E...",
			hostname: "gitlab.com",
			expected: false,
		},
		{
			name:     "empty line",
			line:     "",
			hostname: "github.com",
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

func TestMatchesHost(t *testing.T) {
	// Create a mock SSH key for testing
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
			line:     "github.com ssh-rsa dGVzdC1rZXktZGF0YQ==",
			hostname: "github.com",
			key:      mockKey,
			expected: true,
		},
		{
			name:     "hostname mismatch",
			line:     "gitlab.com ssh-rsa dGVzdC1rZXktZGF0YQ==",
			hostname: "github.com",
			key:      mockKey,
			expected: false,
		},
		{
			name:     "key type mismatch",
			line:     "github.com ssh-ed25519 dGVzdC1rZXktZGF0YQ==",
			hostname: "github.com",
			key:      mockKey,
			expected: false,
		},
		{
			name:     "key data mismatch",
			line:     "github.com ssh-rsa ZGlmZmVyZW50LWtleQ==",
			hostname: "github.com",
			key:      mockKey,
			expected: false,
		},
		{
			name:     "invalid base64",
			line:     "github.com ssh-rsa invalid-base64!!!",
			hostname: "github.com",
			key:      mockKey,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesHost(tt.line, tt.hostname, tt.key)
			if result != tt.expected {
				t.Errorf("matchesHost(%q, %q, key) = %v, want %v", tt.line, tt.hostname, result, tt.expected)
			}
		})
	}
}

func TestEnsureKnownHostsFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-ssh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, ".ssh", "known_hosts")

	err = ensureKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Errorf("ensureKnownHostsFile() error = %v", err)
	}

	// Check if directory was created
	if _, err := os.Stat(filepath.Dir(knownHostsPath)); os.IsNotExist(err) {
		t.Error("SSH directory was not created")
	}

	// Check if file was created
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		t.Error("known_hosts file was not created")
	}

	// Check file permissions
	info, err := os.Stat(knownHostsPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("known_hosts file has wrong permissions: %v, want 0600", info.Mode().Perm())
	}
}

func TestAddHostKey(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-ssh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	mockKey := &mockPublicKey{
		keyType: "ssh-rsa",
		keyData: []byte("test-key-data"),
	}

	addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
	err = addHostKey(knownHostsPath, "example.com", addr, mockKey)
	if err != nil {
		t.Errorf("addHostKey() error = %v", err)
	}

	// Read file and verify content
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedLine := "example.com ssh-rsa dGVzdC1rZXktZGF0YQ=="
	if !strings.Contains(string(content), expectedLine) {
		t.Errorf("Host key not found in known_hosts file. Content: %s", string(content))
	}
}

func TestCleanKnownHostsFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-ssh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	// Create a known_hosts file with mixed valid/invalid entries
	content := `# This is a comment
github.com ssh-rsa Z2l0aHViLWtleQ==
gitlab.com ssh-rsa dGVzdC1rZXktZGF0YQ==

invalid ssh-rsa invalid-base64-data!!!
192.168.1.1 ssh-ed25519 ZWQyNTUxOS1rZXk=
`

	err = os.WriteFile(knownHostsPath, []byte(content), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cleanedPath, err := cleanKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Errorf("cleanKnownHostsFile() error = %v", err)
	}
	defer os.Remove(cleanedPath)

	// Read cleaned content
	cleanedContent, err := os.ReadFile(cleanedPath)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(cleanedContent), "\n")
	validLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			validLines++
		}
	}

	// Should have 3 valid lines (github.com, gitlab.com and 192.168.1.1)
	if validLines != 3 {
		t.Errorf("Expected 3 valid lines after cleaning, got %d", validLines)
	}

	// Should not contain the invalid line
	if strings.Contains(string(cleanedContent), "invalid ssh-rsa invalid-base64-data!!!") {
		t.Error("Invalid line was not removed")
	}
}

func TestCopyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-copy")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	testContent := "Hello, World!\nThis is a test file."
	err = os.WriteFile(srcPath, []byte(testContent), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = copyFile(srcPath, dstPath)
	if err != nil {
		t.Errorf("copyFile() error = %v", err)
	}

	// Verify content
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(copiedContent) != testContent {
		t.Errorf("File content mismatch. Got: %s, Want: %s", string(copiedContent), testContent)
	}
}

func TestAuthConfigs(t *testing.T) {
	t.Run("DefaultAuthConfig", func(t *testing.T) {
		config := DefaultAuthConfig()
		if config.SkipHostKeyVerification != false {
			t.Error("Default config should not skip host key verification")
		}
		if config.AutoAddHostKeys != false {
			t.Error("Default config should not auto-add host keys")
		}
		if config.KnownHostsFile != "" {
			t.Error("Default config should have empty known_hosts file path")
		}
	})

	t.Run("DevelopmentAuthConfig", func(t *testing.T) {
		config := DevelopmentAuthConfig()
		if config.SkipHostKeyVerification != false {
			t.Error("Development config should not skip host key verification")
		}
		if config.AutoAddHostKeys != true {
			t.Error("Development config should auto-add host keys")
		}
	})

	t.Run("InsecureAuthConfig", func(t *testing.T) {
		config := InsecureAuthConfig()
		if config.SkipHostKeyVerification != true {
			t.Error("Insecure config should skip host key verification")
		}
		if config.AutoAddHostKeys != false {
			t.Error("Insecure config should not auto-add host keys")
		}
	})
}

func TestGetHostKeyCallback(t *testing.T) {
	t.Run("skip verification", func(t *testing.T) {
		config := AuthConfig{
			SkipHostKeyVerification: true,
		}

		callback, err := GetHostKeyCallback(config)
		if err != nil {
			t.Errorf("GetHostKeyCallback() error = %v", err)
		}

		// Test that it returns the insecure callback
		mockKey := &mockPublicKey{}
		addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
		err = callback("example.com", addr, mockKey)
		if err != nil {
			t.Errorf("Insecure callback should not return error, got: %v", err)
		}
	})

	t.Run("with custom known_hosts path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "test-ssh")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		knownHostsPath := filepath.Join(tempDir, "known_hosts")
		config := AuthConfig{
			KnownHostsFile:          knownHostsPath,
			SkipHostKeyVerification: false,
		}

		callback, err := GetHostKeyCallback(config)
		if err != nil {
			t.Errorf("GetHostKeyCallback() error = %v", err)
		}

		if callback == nil {
			t.Error("Callback should not be nil")
		}
	})
}

func TestIsAgentAvailable(t *testing.T) {
	// Save original environment
	originalSock := os.Getenv("SSH_AUTH_SOCK")
	defer os.Setenv("SSH_AUTH_SOCK", originalSock)

	t.Run("no SSH_AUTH_SOCK", func(t *testing.T) {
		os.Unsetenv("SSH_AUTH_SOCK")
		if IsAgentAvailable() {
			t.Error("IsAgentAvailable() should return false when SSH_AUTH_SOCK is not set")
		}
	})

	t.Run("invalid SSH_AUTH_SOCK", func(t *testing.T) {
		os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
		if IsAgentAvailable() {
			t.Error("IsAgentAvailable() should return false for invalid socket path")
		}
	})
}

func TestPublicCleanKnownHostsFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-clean")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, "known_hosts")
	content := `github.com ssh-rsa Z2l0aHViLWtleQ==
invalid line!!!
gitlab.com ssh-rsa dGVzdC1rZXktZGF0YQ==`

	err = os.WriteFile(knownHostsPath, []byte(content), 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = CleanKnownHostsFile(knownHostsPath)
	if err != nil {
		t.Errorf("CleanKnownHostsFile() error = %v", err)
	}

	// Check that backup was created
	backupFiles, err := filepath.Glob(knownHostsPath + ".backup.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backupFiles) != 1 {
		t.Errorf("Expected 1 backup file, got %d", len(backupFiles))
	}

	// Verify cleaned content
	cleanedContent, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(cleanedContent), "invalid line!!!") {
		t.Error("Invalid line should have been removed")
	}
}

func TestGetAuthMethods(t *testing.T) {
	// Save original environment
	originalSock := os.Getenv("SSH_AUTH_SOCK")
	defer os.Setenv("SSH_AUTH_SOCK", originalSock)

	t.Run("no SSH agent available", func(t *testing.T) {
		os.Unsetenv("SSH_AUTH_SOCK")
		config := DefaultAuthConfig()

		authMethods, cleanup, err := GetAuthMethods(config)
		if err == nil {
			t.Error("Expected error when SSH agent is not available")
		}
		if authMethods != nil {
			t.Error("Expected nil auth methods when SSH agent is not available")
		}
		if cleanup != nil {
			t.Error("Expected nil cleanup function when SSH agent is not available")
		}
	})

	t.Run("invalid SSH agent socket", func(t *testing.T) {
		os.Setenv("SSH_AUTH_SOCK", "/nonexistent/socket")
		config := DefaultAuthConfig()

		authMethods, cleanup, err := GetAuthMethods(config)
		if err == nil {
			t.Error("Expected error when SSH agent socket is invalid")
		}
		if authMethods != nil {
			t.Error("Expected nil auth methods when SSH agent socket is invalid")
		}
		if cleanup != nil {
			cleanup() // Clean up if somehow returned
		}
	})
}

func TestWrapWithAutoAdd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-wrap")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	// Create a base callback that always returns an error
	baseCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return fmt.Errorf("host key not found")
	}

	wrappedCallback := wrapWithAutoAdd(baseCallback, knownHostsPath)

	mockKey := &mockPublicKey{
		keyType: "ssh-rsa",
		keyData: []byte("test-key-data"),
	}

	addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
	err = wrappedCallback("example.com", addr, mockKey)
	if err != nil {
		t.Errorf("wrapWithAutoAdd should have auto-added the key, got error: %v", err)
	}

	// Verify the key was added
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "example.com") {
		t.Error("Host key was not added to known_hosts")
	}
}

func TestCreatePermissiveHostKeyCallback(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-permissive")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	knownHostsPath := filepath.Join(tempDir, "known_hosts")

	t.Run("auto-add enabled with empty known_hosts", func(t *testing.T) {
		callback := createPermissiveHostKeyCallback(knownHostsPath, true)

		mockKey := &mockPublicKey{
			keyType: "ssh-rsa",
			keyData: []byte("test-key-data"),
		}

		addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
		err = callback("newhost.com", addr, mockKey)
		if err != nil {
			t.Errorf("Permissive callback with auto-add should not return error: %v", err)
		}

		// Verify the key was added
		content, err := os.ReadFile(knownHostsPath)
		if err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(string(content), "newhost.com") {
			t.Error("Host key was not auto-added")
		}
	})

	t.Run("auto-add disabled with unknown host", func(t *testing.T) {
		// Create a new temp dir to avoid interference
		tempDir2, err := os.MkdirTemp("", "test-permissive-2")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir2)

		knownHostsPath2 := filepath.Join(tempDir2, "known_hosts")
		callback := createPermissiveHostKeyCallback(knownHostsPath2, false)

		mockKey := &mockPublicKey{
			keyType: "ssh-rsa",
			keyData: []byte("test-key-data"),
		}

		addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
		err = callback("unknown.com", addr, mockKey)
		if err == nil {
			t.Error("Permissive callback without auto-add should return error for unknown host")
		}
	})

	t.Run("existing host key verification", func(t *testing.T) {
		// Create known_hosts with a test entry
		testKnownHostsPath := filepath.Join(tempDir, "test_known_hosts")
		testContent := "testhost.com ssh-rsa dGVzdC1rZXktZGF0YQ==\n"
		err := os.WriteFile(testKnownHostsPath, []byte(testContent), 0600)
		if err != nil {
			t.Fatal(err)
		}

		callback := createPermissiveHostKeyCallback(testKnownHostsPath, false)

		// Test with matching key
		matchingKey := &mockPublicKey{
			keyType: "ssh-rsa",
			keyData: []byte("test-key-data"),
		}

		addr, _ := net.ResolveTCPAddr("tcp", "192.168.1.1:22")
		err = callback("testhost.com", addr, matchingKey)
		if err != nil {
			t.Errorf("Should verify existing matching host key: %v", err)
		}

		// Test with non-matching key
		nonMatchingKey := &mockPublicKey{
			keyType: "ssh-rsa",
			keyData: []byte("different-key-data"),
		}

		err = callback("testhost.com", addr, nonMatchingKey)
		if err == nil {
			t.Error("Should reject non-matching host key when auto-add is disabled")
		}
	})
}

// Mock implementations for testing
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
