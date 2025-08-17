package tunnel

import (
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorType ErrorType
	}{
		{
			name: "valid config with password",
			config: Config{
				Host:     "example.com",
				Port:     22,
				User:     "testuser",
				Password: "testpass",
			},
			wantError: false,
		},
		{
			name: "valid config with private key",
			config: Config{
				Host:       "example.com",
				Port:       22,
				User:       "testuser",
				PrivateKey: "test-key-content",
			},
			wantError: false,
		},
		{
			name: "missing host",
			config: Config{
				User:     "testuser",
				Password: "testpass",
			},
			wantError: true,
			errorType: ErrorConnection,
		},
		{
			name: "missing user",
			config: Config{
				Host:     "example.com",
				Password: "testpass",
			},
			wantError: true,
			errorType: ErrorConnection,
		},
		{
			name: "missing auth",
			config: Config{
				Host: "example.com",
				User: "testuser",
			},
			wantError: true,
			errorType: ErrorAuth,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				} else if sshErr, ok := err.(*Error); ok {
					if sshErr.Type != tt.errorType {
						t.Errorf("expected error type %v, got %v", tt.errorType, sshErr.Type)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Error("expected client but got nil")
				}
				// Check defaults are set
				if tt.config.Port == 0 && client.config.Port != 22 {
					t.Error("default port should be 22")
				}
				if tt.config.Timeout == 0 && client.config.Timeout != 30*time.Second {
					t.Error("default timeout should be 30 seconds")
				}
			}
		})
	}
}

func TestMockClient(t *testing.T) {
	t.Run("basic operations", func(t *testing.T) {
		client := NewMockClient()

		// Test connection
		if err := client.Connect(); err != nil {
			t.Errorf("Connect() failed: %v", err)
		}

		if !client.IsConnected() {
			t.Error("IsConnected() should return true after Connect()")
		}

		// Test command execution
		client.OnExecute("echo test").ReturnSuccess("test")
		result, err := client.Execute("echo test")
		if err != nil {
			t.Errorf("Execute() failed: %v", err)
		}
		if result.Stdout != "test" {
			t.Errorf("expected stdout 'test', got %q", result.Stdout)
		}

		// Test sudo command
		client.OnExecute("sudo apt update").ReturnSuccess("packages updated")
		result, err = client.ExecuteSudo("apt update")
		if err != nil {
			t.Errorf("ExecuteSudo() failed: %v", err)
		}
		if result.Stdout != "packages updated" {
			t.Errorf("expected stdout 'packages updated', got %q", result.Stdout)
		}

		// Test close
		if err := client.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}

		if client.IsConnected() {
			t.Error("IsConnected() should return false after Close()")
		}
	})

	t.Run("command failures", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()

		// Test command error
		expectedErr := &Error{Type: ErrorExecution, Message: "command failed"}
		client.OnExecute("failing-command").ReturnError(expectedErr)

		_, err := client.Execute("failing-command")
		if err == nil {
			t.Error("expected error but got none")
		}
		if err.Error() != expectedErr.Error() {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}

		// Test command with non-zero exit code
		client.OnExecute("exit 1").ReturnFailure("error output", 1)
		result, err := client.Execute("exit 1")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", result.ExitCode)
		}
		if result.Stderr != "error output" {
			t.Errorf("expected stderr 'error output', got %q", result.Stderr)
		}
	})

	t.Run("file operations", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()

		// Test upload
		if err := client.Upload("/local/file.txt", "/remote/file.txt"); err != nil {
			t.Errorf("Upload() failed: %v", err)
		}

		uploads := client.GetUploads()
		if remotePath, exists := uploads["/local/file.txt"]; !exists || remotePath != "/remote/file.txt" {
			t.Error("Upload was not recorded correctly")
		}

		// Test download
		if err := client.Download("/remote/file.txt", "/local/file.txt"); err != nil {
			t.Errorf("Download() failed: %v", err)
		}

		downloads := client.GetDownloads()
		if localPath, exists := downloads["/remote/file.txt"]; !exists || localPath != "/local/file.txt" {
			t.Error("Download was not recorded correctly")
		}

		// Test upload failure
		uploadErr := &Error{Type: ErrorFileTransfer, Message: "upload failed"}
		client.OnUpload("/fail.txt", "/remote/fail.txt").Fail(uploadErr)

		if err := client.Upload("/fail.txt", "/remote/fail.txt"); err == nil {
			t.Error("expected upload to fail")
		}
	})

	t.Run("execution history", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()

		commands := []string{"ls -la", "pwd", "whoami"}
		for _, cmd := range commands {
			client.Execute(cmd)
		}

		history := client.GetExecutionHistory()
		if len(history) != len(commands) {
			t.Errorf("expected %d commands in history, got %d", len(commands), len(history))
		}

		for i, cmd := range commands {
			if history[i] != cmd {
				t.Errorf("history[%d] = %q, expected %q", i, history[i], cmd)
			}
		}
	})

	t.Run("pattern matching", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()

		// Setup pattern response
		client.OnExecute("systemctl").ReturnSuccess("systemctl output")

		// Test commands containing the pattern
		testCmds := []string{
			"systemctl status nginx",
			"systemctl restart apache",
			"sudo systemctl stop mysql",
		}

		for _, cmd := range testCmds {
			result, err := client.Execute(cmd)
			if err != nil {
				t.Errorf("Execute(%q) failed: %v", cmd, err)
			}
			if !strings.Contains(result.Stdout, "systemctl") {
				t.Errorf("expected output to contain 'systemctl', got %q", result.Stdout)
			}
		}
	})

	t.Run("reset functionality", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()

		// Add some data
		client.OnExecute("test").ReturnSuccess("test output")
		client.Execute("command1")
		client.Execute("command2")
		client.Upload("/local", "/remote")

		// Reset
		client.Reset()

		// Verify everything is cleared
		if client.IsConnected() {
			t.Error("client should not be connected after reset")
		}

		history := client.GetExecutionHistory()
		if len(history) != 0 {
			t.Error("execution history should be empty after reset")
		}

		uploads := client.GetUploads()
		if len(uploads) != 0 {
			t.Error("uploads should be empty after reset")
		}
	})
}

func TestError(t *testing.T) {
	t.Run("error with cause", func(t *testing.T) {
		cause := &Error{Type: ErrorConnection, Message: "connection refused"}
		err := &Error{
			Type:    ErrorExecution,
			Message: "failed to execute command",
			Cause:   cause,
		}

		expectedMsg := "failed to execute command: connection refused"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
		}

		if err.Unwrap() != cause {
			t.Error("Unwrap() should return the cause")
		}
	})

	t.Run("error without cause", func(t *testing.T) {
		err := &Error{
			Type:    ErrorAuth,
			Message: "authentication failed",
		}

		if err.Error() != "authentication failed" {
			t.Errorf("expected error message 'authentication failed', got %q", err.Error())
		}

		if err.Unwrap() != nil {
			t.Error("Unwrap() should return nil when there's no cause")
		}
	})
}

func TestExecOptions(t *testing.T) {
	t.Run("WithTimeout", func(t *testing.T) {
		cfg := &execConfig{}
		opt := WithTimeout(5 * time.Minute)
		opt(cfg)

		if cfg.timeout != 5*time.Minute {
			t.Errorf("expected timeout 5m, got %v", cfg.timeout)
		}
	})

	t.Run("WithEnv", func(t *testing.T) {
		cfg := &execConfig{}
		opt1 := WithEnv("KEY1", "value1")
		opt2 := WithEnv("KEY2", "value2")

		opt1(cfg)
		opt2(cfg)

		if cfg.env["KEY1"] != "value1" {
			t.Error("KEY1 not set correctly")
		}
		if cfg.env["KEY2"] != "value2" {
			t.Error("KEY2 not set correctly")
		}
	})

	t.Run("WithWorkDir", func(t *testing.T) {
		cfg := &execConfig{}
		opt := WithWorkDir("/opt/app")
		opt(cfg)

		if cfg.workDir != "/opt/app" {
			t.Errorf("expected workDir '/opt/app', got %q", cfg.workDir)
		}
	})

	t.Run("WithSudo", func(t *testing.T) {
		cfg := &execConfig{}
		opt := WithSudo()
		opt(cfg)

		if !cfg.sudo {
			t.Error("sudo should be true")
		}
	})

	t.Run("WithStream", func(t *testing.T) {
		cfg := &execConfig{}
		called := false
		handler := func(line string) {
			called = true
		}

		opt := WithStream(handler)
		opt(cfg)

		if cfg.stream == nil {
			t.Error("stream handler should be set")
		}

		cfg.stream("test")
		if !called {
			t.Error("stream handler was not called")
		}
	})
}

func TestUserOptions(t *testing.T) {
	t.Run("WithHome", func(t *testing.T) {
		cfg := &userConfig{}
		opt := WithHome("/home/testuser")
		opt(cfg)

		if cfg.home != "/home/testuser" {
			t.Errorf("expected home '/home/testuser', got %q", cfg.home)
		}
	})

	t.Run("WithShell", func(t *testing.T) {
		cfg := &userConfig{}
		opt := WithShell("/bin/zsh")
		opt(cfg)

		if cfg.shell != "/bin/zsh" {
			t.Errorf("expected shell '/bin/zsh', got %q", cfg.shell)
		}
	})

	t.Run("WithGroups", func(t *testing.T) {
		cfg := &userConfig{}
		opt := WithGroups("sudo", "docker", "www-data")
		opt(cfg)

		if len(cfg.groups) != 3 {
			t.Errorf("expected 3 groups, got %d", len(cfg.groups))
		}

		expectedGroups := []string{"sudo", "docker", "www-data"}
		for i, group := range expectedGroups {
			if cfg.groups[i] != group {
				t.Errorf("expected group[%d] = %q, got %q", i, group, cfg.groups[i])
			}
		}
	})

	t.Run("WithSudoAccess", func(t *testing.T) {
		cfg := &userConfig{}
		opt := WithSudoAccess()
		opt(cfg)

		if !cfg.sudoAccess {
			t.Error("sudoAccess should be true")
		}
	})

	t.Run("WithSystemUser", func(t *testing.T) {
		cfg := &userConfig{}
		opt := WithSystemUser()
		opt(cfg)

		if !cfg.systemUser {
			t.Error("systemUser should be true")
		}
	})
}

func TestFileOptions(t *testing.T) {
	t.Run("WithProgress", func(t *testing.T) {
		cfg := &fileTransferConfig{}
		progressCalled := false
		handler := func(percent int) {
			progressCalled = true
		}

		opt := WithProgress(handler)
		opt(cfg)

		if cfg.progress == nil {
			t.Error("progress handler should be set")
		}

		cfg.progress(50)
		if !progressCalled {
			t.Error("progress handler was not called")
		}
	})

	t.Run("WithFileMode", func(t *testing.T) {
		cfg := &fileTransferConfig{}
		opt := WithFileMode(0755)
		opt(cfg)

		if cfg.mode != 0755 {
			t.Errorf("expected mode 0755, got %o", cfg.mode)
		}
	})

	t.Run("WithPreserve", func(t *testing.T) {
		cfg := &fileTransferConfig{}
		opt := WithPreserve()
		opt(cfg)

		if !cfg.preserve {
			t.Error("preserve should be true")
		}
	})
}

func TestCommand(t *testing.T) {
	cmd := Cmd("ls -la", WithTimeout(30*time.Second), WithWorkDir("/tmp"))

	if cmd.Cmd != "ls -la" {
		t.Errorf("expected command 'ls -la', got %q", cmd.Cmd)
	}

	if len(cmd.Opts) != 2 {
		t.Errorf("expected 2 options, got %d", len(cmd.Opts))
	}
}

func TestTracers(t *testing.T) {
	t.Run("NoOpTracer", func(t *testing.T) {
		tracer := &NoOpTracer{}

		// These should all be no-ops and not panic
		tracer.OnConnect("host", "user")
		tracer.OnDisconnect("host")
		tracer.OnExecute("command")
		tracer.OnExecuteResult("command", nil, nil)
		tracer.OnUpload("local", "remote")
		tracer.OnUploadComplete("local", "remote", nil)
		tracer.OnDownload("remote", "local")
		tracer.OnDownloadComplete("remote", "local", nil)
		tracer.OnError("operation", &Error{})
	})

	t.Run("SimpleLogger", func(t *testing.T) {
		logger := &SimpleLogger{Verbose: true}

		// These should log but not panic
		logger.OnConnect("host", "user")
		logger.OnDisconnect("host")
		logger.OnExecute("command")
		logger.OnExecuteResult("command", &Result{ExitCode: 0}, nil)
		logger.OnUpload("local", "remote")
		logger.OnUploadComplete("local", "remote", nil)
		logger.OnDownload("remote", "local")
		logger.OnDownloadComplete("remote", "local", nil)
		logger.OnError("operation", &Error{Message: "test error"})
	})

	t.Run("TestTracer", func(t *testing.T) {
		tracer := NewTestTracer()

		// Generate some events
		tracer.OnConnect("test-host", "test-user")
		tracer.OnExecute("test command")
		tracer.OnExecuteResult("test command", &Result{ExitCode: 0}, nil)
		tracer.OnDisconnect("test-host")

		// Check events were recorded
		events := tracer.GetEvents()
		if len(events) != 4 {
			t.Errorf("expected 4 events, got %d", len(events))
		}

		// Check specific event types
		connectEvents := tracer.GetEventsByType("connect")
		if len(connectEvents) != 1 {
			t.Errorf("expected 1 connect event, got %d", len(connectEvents))
		}

		if connectEvents[0].Data["host"] != "test-host" {
			t.Error("connect event should have correct host")
		}

		// Clear and verify
		tracer.Clear()
		events = tracer.GetEvents()
		if len(events) != 0 {
			t.Error("events should be empty after Clear()")
		}
	})
}
