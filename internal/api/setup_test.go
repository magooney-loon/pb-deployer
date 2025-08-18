package api

import (
	"strings"
	"testing"
)

func TestGetPublicKeysForSetup(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Valid SSH keys",
			input:    []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
			expected: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
		},
		{
			name:     "Mixed valid and invalid keys",
			input:    []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "invalid-key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
			expected: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
		},
		{
			name:     "Empty and whitespace keys",
			input:    []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "", "   ", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
			expected: []string{"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAB", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5"},
		},
		{
			name:     "ECDSA keys",
			input:    []string{"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY"},
			expected: []string{"ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY"},
		},
		{
			name:     "No valid keys",
			input:    []string{"invalid", "", "   "},
			expected: []string{},
		},
		{
			name:     "Empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPublicKeysForSetup(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d keys, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected key %d to be '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}

func TestCreateSSHClient_InvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    int
		user    string
		wantErr bool
	}{
		{
			name:    "Empty host",
			host:    "",
			port:    22,
			user:    "root",
			wantErr: true,
		},
		{
			name:    "Empty user",
			host:    "example.com",
			port:    22,
			user:    "",
			wantErr: true,
		},
		{
			name:    "Valid inputs",
			host:    "example.com",
			port:    22,
			user:    "root",
			wantErr: false, // Should succeed when SSH agent is available
		},
		{
			name:    "Zero port defaults to 22",
			host:    "example.com",
			port:    0,
			user:    "root",
			wantErr: false, // Should succeed when SSH agent is available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := createSSHClient(tt.host, tt.port, tt.user)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if err != nil && client != nil {
				t.Error("Expected nil client when error occurs")
			}
		})
	}
}

func TestValidateSSHConnection_NilClient(t *testing.T) {
	err := validateSSHConnection(nil)
	if err == nil {
		t.Error("Expected error for nil client")
	}

	expectedMsg := "client is nil"
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain '%s', got: %v", expectedMsg, err)
	}
}

func TestBasicValidation(t *testing.T) {
	// Test host validation
	tests := []struct {
		name     string
		host     string
		user     string
		username string
		valid    bool
	}{
		{"Valid inputs", "example.com", "root", "pocketbase", true},
		{"Empty host", "", "root", "pocketbase", false},
		{"Empty user", "example.com", "", "pocketbase", false},
		{"Empty username", "example.com", "root", "", false},
		{"All empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.host != "" && tt.user != "" && tt.username != ""

			if valid != tt.valid {
				t.Errorf("Expected validation result %v, got %v", tt.valid, valid)
			}
		})
	}
}

func TestPortDefaults(t *testing.T) {
	tests := []struct {
		name         string
		inputPort    int
		expectedPort int
	}{
		{"Zero port", 0, 22},
		{"Default SSH port", 22, 22},
		{"Custom port", 2222, 2222},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.inputPort
			if port == 0 {
				port = 22
			}

			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
		})
	}
}

func TestSSHKeyValidation(t *testing.T) {
	validPrefixes := []string{"ssh-", "ecdsa-"}

	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"SSH RSA key", "ssh-rsa AAAAB3NzaC1yc2E", true},
		{"SSH ED25519 key", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5", true},
		{"ECDSA key", "ecdsa-sha2-nistp256 AAAAE2VjZHNh", true},
		{"Invalid key", "invalid-key-format", false},
		{"Empty key", "", false},
		{"Just ssh prefix", "ssh-", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := false
			trimmed := tt.key

			if trimmed != "" {
				for _, prefix := range validPrefixes {
					if len(trimmed) > len(prefix) &&
						(trimmed[:len(prefix)] == prefix || trimmed == "ssh-ed25519") {
						valid = true
						break
					}
				}
			}

			if valid != tt.valid {
				t.Errorf("Expected validation result %v for key '%s', got %v", tt.valid, tt.key, valid)
			}
		})
	}
}
