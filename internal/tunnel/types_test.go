package tunnel

import (
	"testing"
	"time"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		expected ErrorType
	}{
		{"Unknown", ErrorUnknown, ErrorType(0)},
		{"Connection", ErrorConnection, ErrorType(1)},
		{"Auth", ErrorAuth, ErrorType(2)},
		{"Execution", ErrorExecution, ErrorType(3)},
		{"Timeout", ErrorTimeout, ErrorType(4)},
		{"FileTransfer", ErrorFileTransfer, ErrorType(5)},
		{"NotFound", ErrorNotFound, ErrorType(6)},
		{"Permission", ErrorPermission, ErrorType(7)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.errType != tt.expected {
				t.Errorf("ErrorType %s = %d, expected %d", tt.name, tt.errType, tt.expected)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Host:       "example.com",
		Port:       2222,
		User:       "testuser",
		Password:   "testpass",
		PrivateKey: "key-content",
		Passphrase: "key-passphrase",
		Timeout:    60 * time.Second,
		RetryCount: 5,
		RetryDelay: 10 * time.Second,
	}

	if config.Host != "example.com" {
		t.Errorf("expected Host 'example.com', got %q", config.Host)
	}
	if config.Port != 2222 {
		t.Errorf("expected Port 2222, got %d", config.Port)
	}
	if config.User != "testuser" {
		t.Errorf("expected User 'testuser', got %q", config.User)
	}
	if config.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s, got %v", config.Timeout)
	}
}

func TestResult(t *testing.T) {
	result := Result{
		Stdout:   "output",
		Stderr:   "error",
		ExitCode: 1,
		Duration: 5 * time.Second,
	}

	if result.Stdout != "output" {
		t.Errorf("expected Stdout 'output', got %q", result.Stdout)
	}
	if result.Stderr != "error" {
		t.Errorf("expected Stderr 'error', got %q", result.Stderr)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected ExitCode 1, got %d", result.ExitCode)
	}
	if result.Duration != 5*time.Second {
		t.Errorf("expected Duration 5s, got %v", result.Duration)
	}
}

func TestServiceStatus(t *testing.T) {
	now := time.Now()
	status := ServiceStatus{
		Name:        "nginx",
		Active:      true,
		Running:     true,
		Enabled:     true,
		Description: "Web server",
		Since:       now,
		MainPID:     1234,
	}

	if status.Name != "nginx" {
		t.Errorf("expected Name 'nginx', got %q", status.Name)
	}
	if !status.Active {
		t.Error("expected Active to be true")
	}
	if !status.Running {
		t.Error("expected Running to be true")
	}
	if !status.Enabled {
		t.Error("expected Enabled to be true")
	}
	if status.MainPID != 1234 {
		t.Errorf("expected MainPID 1234, got %d", status.MainPID)
	}
	if status.Since != now {
		t.Errorf("expected Since %v, got %v", now, status.Since)
	}
}

func TestFirewallRule(t *testing.T) {
	rule := FirewallRule{
		Port:        443,
		Protocol:    "tcp",
		Source:      "192.168.1.0/24",
		Action:      "allow",
		Description: "HTTPS traffic",
	}

	if rule.Port != 443 {
		t.Errorf("expected Port 443, got %d", rule.Port)
	}
	if rule.Protocol != "tcp" {
		t.Errorf("expected Protocol 'tcp', got %q", rule.Protocol)
	}
	if rule.Source != "192.168.1.0/24" {
		t.Errorf("expected Source '192.168.1.0/24', got %q", rule.Source)
	}
	if rule.Action != "allow" {
		t.Errorf("expected Action 'allow', got %q", rule.Action)
	}
	if rule.Description != "HTTPS traffic" {
		t.Errorf("expected Description 'HTTPS traffic', got %q", rule.Description)
	}
}

func TestSSHConfig(t *testing.T) {
	config := SSHConfig{
		PasswordAuth:        false,
		RootLogin:           false,
		PubkeyAuth:          true,
		MaxAuthTries:        3,
		ClientAliveInterval: 300,
		ClientAliveCountMax: 2,
		AllowUsers:          []string{"user1", "user2"},
		AllowGroups:         []string{"sudo", "admin"},
		DenyUsers:           []string{"baduser"},
		DenyGroups:          []string{"restricted"},
	}

	if config.PasswordAuth {
		t.Error("expected PasswordAuth to be false")
	}
	if config.RootLogin {
		t.Error("expected RootLogin to be false")
	}
	if !config.PubkeyAuth {
		t.Error("expected PubkeyAuth to be true")
	}
	if config.MaxAuthTries != 3 {
		t.Errorf("expected MaxAuthTries 3, got %d", config.MaxAuthTries)
	}
	if config.ClientAliveInterval != 300 {
		t.Errorf("expected ClientAliveInterval 300, got %d", config.ClientAliveInterval)
	}
	if config.ClientAliveCountMax != 2 {
		t.Errorf("expected ClientAliveCountMax 2, got %d", config.ClientAliveCountMax)
	}
	if len(config.AllowUsers) != 2 {
		t.Errorf("expected 2 AllowUsers, got %d", len(config.AllowUsers))
	}
	if len(config.AllowGroups) != 2 {
		t.Errorf("expected 2 AllowGroups, got %d", len(config.AllowGroups))
	}
}

func TestAppConfig(t *testing.T) {
	app := AppConfig{
		Name:        "myapp",
		Version:     "v1.2.3",
		Source:      "/local/app.tar.gz",
		Target:      "/opt/myapp",
		Service:     "myapp-service",
		Backup:      true,
		PreDeploy:   []string{"stop-service.sh"},
		PostDeploy:  []string{"start-service.sh", "notify.sh"},
		HealthCheck: "http://localhost:8080/health",
	}

	if app.Name != "myapp" {
		t.Errorf("expected Name 'myapp', got %q", app.Name)
	}
	if app.Version != "v1.2.3" {
		t.Errorf("expected Version 'v1.2.3', got %q", app.Version)
	}
	if app.Source != "/local/app.tar.gz" {
		t.Errorf("expected Source '/local/app.tar.gz', got %q", app.Source)
	}
	if app.Target != "/opt/myapp" {
		t.Errorf("expected Target '/opt/myapp', got %q", app.Target)
	}
	if app.Service != "myapp-service" {
		t.Errorf("expected Service 'myapp-service', got %q", app.Service)
	}
	if !app.Backup {
		t.Error("expected Backup to be true")
	}
	if len(app.PreDeploy) != 1 {
		t.Errorf("expected 1 PreDeploy command, got %d", len(app.PreDeploy))
	}
	if len(app.PostDeploy) != 2 {
		t.Errorf("expected 2 PostDeploy commands, got %d", len(app.PostDeploy))
	}
	if app.HealthCheck != "http://localhost:8080/health" {
		t.Errorf("expected HealthCheck 'http://localhost:8080/health', got %q", app.HealthCheck)
	}
}

func TestBatchResult(t *testing.T) {
	result := BatchResult{
		Results: []Result{
			{Stdout: "output1", ExitCode: 0},
			{Stdout: "output2", ExitCode: 0},
			{Stdout: "output3", ExitCode: 1},
		},
		Errors: []error{
			nil,
			nil,
			&Error{Type: ErrorExecution, Message: "command failed"},
		},
	}

	if len(result.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(result.Results))
	}
	if len(result.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d", len(result.Errors))
	}
	if result.Results[2].ExitCode != 1 {
		t.Errorf("expected third result ExitCode 1, got %d", result.Results[2].ExitCode)
	}
	if result.Errors[2] == nil {
		t.Error("expected third error to be non-nil")
	}
}

func TestPackage(t *testing.T) {
	pkg := Package{
		Name:    "nginx",
		Version: "1.18.0",
		Status:  "installed",
	}

	if pkg.Name != "nginx" {
		t.Errorf("expected Name 'nginx', got %q", pkg.Name)
	}
	if pkg.Version != "1.18.0" {
		t.Errorf("expected Version '1.18.0', got %q", pkg.Version)
	}
	if pkg.Status != "installed" {
		t.Errorf("expected Status 'installed', got %q", pkg.Status)
	}
}

func TestSystemInfoTypes(t *testing.T) {
	info := SystemInfo{
		OS:           "Ubuntu 20.04",
		OSVersion:    "20.04",
		Kernel:       "5.4.0-42-generic",
		Architecture: "x86_64",
		Hostname:     "server1",
		CPUCount:     8,
		MemoryMB:     16384,
		DiskGB:       500,
		Uptime:       24 * time.Hour,
	}

	if info.OS != "Ubuntu 20.04" {
		t.Errorf("expected OS 'Ubuntu 20.04', got %q", info.OS)
	}
	if info.Kernel != "5.4.0-42-generic" {
		t.Errorf("expected Kernel '5.4.0-42-generic', got %q", info.Kernel)
	}
	if info.Architecture != "x86_64" {
		t.Errorf("expected Architecture 'x86_64', got %q", info.Architecture)
	}
	if info.Hostname != "server1" {
		t.Errorf("expected Hostname 'server1', got %q", info.Hostname)
	}
	if info.CPUCount != 8 {
		t.Errorf("expected CPUCount 8, got %d", info.CPUCount)
	}
	if info.MemoryMB != 16384 {
		t.Errorf("expected MemoryMB 16384, got %d", info.MemoryMB)
	}
	if info.DiskGB != 500 {
		t.Errorf("expected DiskGB 500, got %d", info.DiskGB)
	}
	if info.Uptime != 24*time.Hour {
		t.Errorf("expected Uptime 24h, got %v", info.Uptime)
	}
}

func TestDirectory(t *testing.T) {
	dir := Directory{
		Path:        "/opt/myapp",
		Permissions: "755",
		Owner:       "appuser",
		Group:       "appgroup",
	}

	if dir.Path != "/opt/myapp" {
		t.Errorf("expected Path '/opt/myapp', got %q", dir.Path)
	}
	if dir.Permissions != "755" {
		t.Errorf("expected Permissions '755', got %q", dir.Permissions)
	}
	if dir.Owner != "appuser" {
		t.Errorf("expected Owner 'appuser', got %q", dir.Owner)
	}
	if dir.Group != "appgroup" {
		t.Errorf("expected Group 'appgroup', got %q", dir.Group)
	}
}

func TestTemplateWebServer(t *testing.T) {
	template := TemplateWebServer{
		Domain:   "example.com",
		SSL:      true,
		PHP:      true,
		Database: "mysql",
		Firewall: true,
	}

	if template.Domain != "example.com" {
		t.Errorf("expected Domain 'example.com', got %q", template.Domain)
	}
	if !template.SSL {
		t.Error("expected SSL to be true")
	}
	if !template.PHP {
		t.Error("expected PHP to be true")
	}
	if template.Database != "mysql" {
		t.Errorf("expected Database 'mysql', got %q", template.Database)
	}
	if !template.Firewall {
		t.Error("expected Firewall to be true")
	}
}

func TestTemplateDocker(t *testing.T) {
	template := TemplateDocker{
		ComposeVersion: true,
		Swarm:          true,
		Registry:       "registry.example.com",
	}

	if !template.ComposeVersion {
		t.Error("expected ComposeVersion to be true")
	}
	if !template.Swarm {
		t.Error("expected Swarm to be true")
	}
	if template.Registry != "registry.example.com" {
		t.Errorf("expected Registry 'registry.example.com', got %q", template.Registry)
	}
}

func TestOptionPatterns(t *testing.T) {
	t.Run("multiple exec options", func(t *testing.T) {
		cfg := &execConfig{}
		opts := []ExecOption{
			WithTimeout(2 * time.Minute),
			WithEnv("KEY1", "value1"),
			WithEnv("KEY2", "value2"),
			WithWorkDir("/app"),
			WithSudo(),
			WithSudoPassword("pass123"),
		}

		for _, opt := range opts {
			opt(cfg)
		}

		if cfg.timeout != 2*time.Minute {
			t.Errorf("expected timeout 2m, got %v", cfg.timeout)
		}
		if len(cfg.env) != 2 {
			t.Errorf("expected 2 env vars, got %d", len(cfg.env))
		}
		if cfg.workDir != "/app" {
			t.Errorf("expected workDir '/app', got %q", cfg.workDir)
		}
		if !cfg.sudo {
			t.Error("expected sudo to be true")
		}
		if cfg.sudoPass != "pass123" {
			t.Errorf("expected sudoPass 'pass123', got %q", cfg.sudoPass)
		}
	})

	t.Run("multiple user options", func(t *testing.T) {
		cfg := &userConfig{}
		opts := []UserOption{
			WithHome("/home/user"),
			WithShell("/bin/zsh"),
			WithGroups("group1", "group2"),
			WithGroups("group3"), // Additional groups
			WithSudoAccess(),
			WithSystemUser(),
		}

		for _, opt := range opts {
			opt(cfg)
		}

		if cfg.home != "/home/user" {
			t.Errorf("expected home '/home/user', got %q", cfg.home)
		}
		if cfg.shell != "/bin/zsh" {
			t.Errorf("expected shell '/bin/zsh', got %q", cfg.shell)
		}
		if len(cfg.groups) != 3 {
			t.Errorf("expected 3 groups, got %d", len(cfg.groups))
		}
		if !cfg.sudoAccess {
			t.Error("expected sudoAccess to be true")
		}
		if !cfg.systemUser {
			t.Error("expected systemUser to be true")
		}
	})

	t.Run("multiple file options", func(t *testing.T) {
		cfg := &fileTransferConfig{}
		progressValues := []int{}

		opts := []FileOption{
			WithProgress(func(pct int) {
				progressValues = append(progressValues, pct)
			}),
			WithFileMode(0644),
			WithPreserve(),
		}

		for _, opt := range opts {
			opt(cfg)
		}

		if cfg.mode != 0644 {
			t.Errorf("expected mode 0644, got %o", cfg.mode)
		}
		if !cfg.preserve {
			t.Error("expected preserve to be true")
		}

		// Test progress callback
		if cfg.progress != nil {
			cfg.progress(50)
			cfg.progress(100)
			if len(progressValues) != 2 {
				t.Errorf("expected 2 progress values, got %d", len(progressValues))
			}
			if progressValues[0] != 50 {
				t.Errorf("expected first progress 50, got %d", progressValues[0])
			}
			if progressValues[1] != 100 {
				t.Errorf("expected second progress 100, got %d", progressValues[1])
			}
		} else {
			t.Error("expected progress handler to be set")
		}
	})
}

func TestCommandCreation(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		opts     []ExecOption
		expected string
	}{
		{
			name:     "simple command",
			cmd:      "ls -la",
			opts:     nil,
			expected: "ls -la",
		},
		{
			name:     "command with timeout",
			cmd:      "sleep 10",
			opts:     []ExecOption{WithTimeout(5 * time.Second)},
			expected: "sleep 10",
		},
		{
			name:     "command with environment",
			cmd:      "echo $TEST",
			opts:     []ExecOption{WithEnv("TEST", "value")},
			expected: "echo $TEST",
		},
		{
			name:     "command with workdir",
			cmd:      "pwd",
			opts:     []ExecOption{WithWorkDir("/tmp")},
			expected: "pwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := Cmd(tt.cmd, tt.opts...)
			if cmd.Cmd != tt.expected {
				t.Errorf("expected command %q, got %q", tt.expected, cmd.Cmd)
			}
			if len(cmd.Opts) != len(tt.opts) {
				t.Errorf("expected %d options, got %d", len(tt.opts), len(cmd.Opts))
			}
		})
	}
}
