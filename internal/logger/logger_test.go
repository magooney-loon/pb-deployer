package logger

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger("TEST")
	if logger.prefix != "TEST" {
		t.Errorf("Expected prefix 'TEST', got '%s'", logger.prefix)
	}
}

func TestGetLoggers(t *testing.T) {
	tests := []struct {
		name     string
		logger   *Logger
		expected string
	}{
		{"Default Logger", GetLogger(), "SYSTEM"},
		{"API Logger", GetAPILogger(), "API"},
		{"Tunnel Logger", GetTunnelLogger(), "TUNNEL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.logger.prefix != tt.expected {
				t.Errorf("Expected prefix '%s', got '%s'", tt.expected, tt.logger.prefix)
			}
		})
	}
}

func captureLogOutput(fn func()) string {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	fn()
	log.SetOutput(os.Stderr) // Reset to default
	return buf.String()
}

func TestLogLevels(t *testing.T) {
	logger := NewLogger("TEST")

	tests := []struct {
		name     string
		logFunc  func()
		symbol   string
		contains []string
	}{
		{
			name:     "Info",
			logFunc:  func() { logger.Info("test info message") },
			symbol:   SymbolInfo,
			contains: []string{"test info message", "[TEST]", SymbolInfo},
		},
		{
			name:     "Success",
			logFunc:  func() { logger.Success("test success message") },
			symbol:   SymbolSuccess,
			contains: []string{"test success message", "[TEST]", SymbolSuccess},
		},
		{
			name:     "Warning",
			logFunc:  func() { logger.Warning("test warning message") },
			symbol:   SymbolWarning,
			contains: []string{"test warning message", "[TEST]", SymbolWarning},
		},
		{
			name:     "Error",
			logFunc:  func() { logger.Error("test error message") },
			symbol:   SymbolError,
			contains: []string{"test error message", "[TEST]", SymbolError},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(tt.logFunc)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

func TestLogWithArgs(t *testing.T) {
	logger := NewLogger("TEST")

	output := captureLogOutput(func() {
		logger.Info("test message with %s and %d", "string", 42)
	})

	if !strings.Contains(output, "test message with string and 42") {
		t.Errorf("Expected formatted message, got: %s", output)
	}
}

func TestDebugMode(t *testing.T) {
	logger := NewLogger("TEST")

	// Test without DEBUG env var
	os.Unsetenv("DEBUG")
	output := captureLogOutput(func() {
		logger.Debug("debug message")
	})

	if output != "" {
		t.Errorf("Expected no debug output without DEBUG env var, got: %s", output)
	}

	// Test with DEBUG env var
	os.Setenv("DEBUG", "1")
	output = captureLogOutput(func() {
		logger.Debug("debug message")
	})

	if !strings.Contains(output, "debug message") {
		t.Errorf("Expected debug output with DEBUG env var, got: %s", output)
	}

	// Clean up
	os.Unsetenv("DEBUG")
}

func TestStepLogging(t *testing.T) {
	logger := NewLogger("TEST")

	output := captureLogOutput(func() {
		logger.Step(2, 5, "processing data")
	})

	expected := []string{"Step 2/5:", "processing data", "[TEST]"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func TestStepLoggingWithArgs(t *testing.T) {
	logger := NewLogger("TEST")

	output := captureLogOutput(func() {
		logger.Step(1, 3, "processing %s with %d items", "data", 10)
	})

	if !strings.Contains(output, "processing data with 10 items") {
		t.Errorf("Expected formatted step message, got: %s", output)
	}
}

func TestRequestLogging(t *testing.T) {
	logger := NewLogger("API")

	output := captureLogOutput(func() {
		logger.Request("post", "/api/test", "192.168.1.1")
	})

	expected := []string{"POST", "/api/test", "192.168.1.1", "[API]"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func TestResponseLogging(t *testing.T) {
	logger := NewLogger("API")
	duration := 150 * time.Millisecond

	tests := []struct {
		name       string
		statusCode int
		symbol     string
	}{
		{"Success Response", 200, SymbolSuccess},
		{"Redirect Response", 301, SymbolInfo},
		{"Client Error", 404, SymbolWarning},
		{"Server Error", 500, SymbolError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(func() {
				logger.Response("GET", "/api/test", tt.statusCode, duration)
			})

			expected := []string{"GET", "/api/test", "150ms", tt.symbol}
			for _, exp := range expected {
				if !strings.Contains(output, exp) {
					t.Errorf("Expected output to contain '%s', got: %s", exp, output)
				}
			}
		})
	}
}

func TestSSHLogging(t *testing.T) {
	logger := NewLogger("TUNNEL")

	tests := []struct {
		name     string
		logFunc  func()
		contains []string
	}{
		{
			name:     "SSH Connect",
			logFunc:  func() { logger.SSHConnect("root", "example.com", 22) },
			contains: []string{"Connecting to", "root@example.com:22"},
		},
		{
			name:     "SSH Connected",
			logFunc:  func() { logger.SSHConnected("root", "example.com") },
			contains: []string{"Connected to", "root@example.com", SymbolSuccess},
		},
		{
			name:     "SSH Disconnected",
			logFunc:  func() { logger.SSHDisconnected("example.com") },
			contains: []string{"Disconnected from", "example.com"},
		},
		{
			name:     "SSH Command",
			logFunc:  func() { logger.SSHCommand("ls -la") },
			contains: []string{"Executing:", "ls -la"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(tt.logFunc)

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain '%s', got: %s", expected, output)
				}
			}
		})
	}
}

func TestSSHCommandResult(t *testing.T) {
	logger := NewLogger("TUNNEL")
	duration := 50 * time.Millisecond

	tests := []struct {
		name     string
		exitCode int
		symbol   string
	}{
		{"Success Command", 0, SymbolSuccess},
		{"Failed Command", 1, SymbolError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(func() {
				logger.SSHCommandResult("test command", tt.exitCode, duration)
			})

			expected := []string{"Command completed", "test command", "50ms", tt.symbol}
			for _, exp := range expected {
				if !strings.Contains(output, exp) {
					t.Errorf("Expected output to contain '%s', got: %s", exp, output)
				}
			}
		})
	}
}

func TestFileTransferLogging(t *testing.T) {
	logger := NewLogger("TUNNEL")

	// Test file transfer start
	output := captureLogOutput(func() {
		logger.FileTransfer("Upload", "/local/file", "/remote/file")
	})

	expected := []string{"Upload", "/local/file", "→", "/remote/file"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}

	// Test successful completion
	output = captureLogOutput(func() {
		logger.FileTransferComplete("Upload", nil)
	})

	if !strings.Contains(output, "Upload completed") || !strings.Contains(output, SymbolSuccess) {
		t.Errorf("Expected successful completion message, got: %s", output)
	}

	// Test failed completion
	output = captureLogOutput(func() {
		logger.FileTransferComplete("Upload", os.ErrNotExist)
	})

	if !strings.Contains(output, "Upload failed") || !strings.Contains(output, SymbolError) {
		t.Errorf("Expected failed completion message, got: %s", output)
	}
}

func TestSystemOperation(t *testing.T) {
	logger := NewLogger("TUNNEL")

	output := captureLogOutput(func() {
		logger.SystemOperation("Creating user: testuser")
	})

	expected := []string{"Creating user: testuser", "[TUNNEL]"}
	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func TestConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func()
		symbol  string
	}{
		{"Info", func() { Info("test") }, SymbolInfo},
		{"Success", func() { Success("test") }, SymbolSuccess},
		{"Warning", func() { Warning("test") }, SymbolWarning},
		{"Error", func() { Error("test") }, SymbolError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(tt.logFunc)

			if !strings.Contains(output, "test") || !strings.Contains(output, tt.symbol) {
				t.Errorf("Expected output to contain 'test' and '%s', got: %s", tt.symbol, output)
			}
		})
	}
}

func TestSSHConvenienceFunctions(t *testing.T) {
	output := captureLogOutput(func() {
		SSHConnect("user", "host", 22)
	})

	if !strings.Contains(output, "user@host:22") {
		t.Errorf("Expected SSH connect message, got: %s", output)
	}

	output = captureLogOutput(func() {
		SSHCommandResult("test", 0, 10*time.Millisecond)
	})

	if !strings.Contains(output, "test") || !strings.Contains(output, SymbolSuccess) {
		t.Errorf("Expected SSH command result message, got: %s", output)
	}
}

func TestTimestampFormat(t *testing.T) {
	logger := NewLogger("TEST")

	output := captureLogOutput(func() {
		logger.Info("test")
	})

	// Check that timestamp is in format [HH:MM:SS.mmm]
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Errorf("Expected timestamp brackets in output: %s", output)
	}

	// Extract timestamp part
	start := strings.Index(output, "[")
	end := strings.Index(output, "]")
	if start == -1 || end == -1 || end <= start {
		t.Errorf("Could not find valid timestamp in output: %s", output)
		return
	}

	timestamp := output[start+1 : end]
	parts := strings.Split(timestamp, ":")
	if len(parts) != 3 {
		t.Errorf("Expected timestamp format HH:MM:SS.mmm, got: %s", timestamp)
	}
}

func TestColorCodes(t *testing.T) {
	// Test that color codes are defined
	colors := map[string]string{
		"Reset":  Reset,
		"Red":    Red,
		"Green":  Green,
		"Yellow": Yellow,
		"Blue":   Blue,
		"Purple": Purple,
		"Cyan":   Cyan,
		"Gray":   Gray,
		"Bold":   Bold,
		"Dim":    Dim,
	}

	for name, code := range colors {
		if code == "" {
			t.Errorf("Color code for %s is empty", name)
		}
		if !strings.HasPrefix(code, "\033[") {
			t.Errorf("Color code for %s does not start with ANSI escape sequence: %s", name, code)
		}
	}
}

func TestSymbols(t *testing.T) {
	symbols := map[string]string{
		"Info":    SymbolInfo,
		"Success": SymbolSuccess,
		"Warning": SymbolWarning,
		"Error":   SymbolError,
		"Debug":   SymbolDebug,
	}

	for name, symbol := range symbols {
		if symbol == "" {
			t.Errorf("Symbol for %s is empty", name)
		}
	}
}
