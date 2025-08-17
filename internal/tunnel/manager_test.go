package tunnel

import (
	"strings"
	"testing"
)

func TestNewManager(t *testing.T) {
	t.Run("valid client", func(t *testing.T) {
		client := NewMockClient()
		manager := NewManager(client)

		if manager == nil {
			t.Error("expected manager but got nil")
		}
		if manager.client != client {
			t.Error("manager should use provided client")
		}
	})

	t.Run("nil client panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("NewManager should panic with nil client")
			}
		}()
		NewManager(nil)
	})
}

func TestUserManagement(t *testing.T) {
	t.Run("CreateUser", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock user doesn't exist
		client.OnExecute("id testuser").ReturnFailure("user not found", 1)
		// Mock user creation
		client.OnExecute("sudo useradd -d '/home/testuser' -m -s '/bin/bash' 'testuser'").ReturnSuccess("")
		// Mock group addition
		client.OnExecute("sudo usermod -aG 'sudo,docker' 'testuser'").ReturnSuccess("")
		// Mock sudo setup
		client.OnExecute("sudo echo 'testuser ALL=(ALL:ALL) NOPASSWD:ALL' > /etc/sudoers.d/testuser").ReturnSuccess("")
		client.OnExecute("sudo chmod 0440 /etc/sudoers.d/testuser").ReturnSuccess("")

		err := manager.CreateUser("testuser",
			WithGroups("sudo", "docker"),
			WithSudoAccess(),
		)

		if err != nil {
			t.Errorf("CreateUser() failed: %v", err)
		}

		// Verify commands were executed
		history := client.GetExecutionHistory()
		if len(history) < 4 {
			t.Error("expected at least 4 commands to be executed")
		}
	})

	t.Run("CreateUser existing user", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock user exists
		client.OnExecute("id testuser").ReturnSuccess("uid=1000(testuser)")

		err := manager.CreateUser("testuser")
		if err != nil {
			t.Errorf("CreateUser() should not fail for existing user: %v", err)
		}

		// Should only check if user exists
		history := client.GetExecutionHistory()
		if len(history) != 1 {
			t.Errorf("expected 1 command, got %d", len(history))
		}
	})

	t.Run("DeleteUser", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		client.OnExecute("sudo userdel -r 'testuser'").ReturnSuccess("")

		err := manager.DeleteUser("testuser")
		if err != nil {
			t.Errorf("DeleteUser() failed: %v", err)
		}
	})

	t.Run("SetupSSHKeys", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock getting home directory
		client.OnExecute("getent passwd testuser | cut -d: -f6").ReturnSuccess("/home/testuser")
		// Mock creating .ssh directory and setting up keys
		client.OnExecute("sudo mkdir -p '/home/testuser/.ssh' && chmod 700 '/home/testuser/.ssh' && chown 'testuser:testuser' '/home/testuser/.ssh'").ReturnSuccess("")
		client.OnExecute("sudo echo 'ssh-rsa AAAAB3... key1\nssh-rsa AAAAB3... key2' > '/home/testuser/.ssh/authorized_keys' && chmod 600 '/home/testuser/.ssh/authorized_keys' && chown 'testuser:testuser' '/home/testuser/.ssh/authorized_keys'").ReturnSuccess("")

		keys := []string{"ssh-rsa AAAAB3... key1", "ssh-rsa AAAAB3... key2"}
		err := manager.SetupSSHKeys("testuser", keys)

		if err != nil {
			t.Errorf("SetupSSHKeys() failed: %v", err)
		}
	})
}

func TestServiceManagement(t *testing.T) {
	services := []string{"nginx", "mysql", "docker"}

	t.Run("ServiceStart", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		for _, service := range services {
			client.OnExecute("sudo systemctl start " + service).ReturnSuccess("")
			err := manager.ServiceStart(service)
			if err != nil {
				t.Errorf("ServiceStart(%s) failed: %v", service, err)
			}
		}
	})

	t.Run("ServiceStop", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		for _, service := range services {
			client.OnExecute("sudo systemctl stop " + service).ReturnSuccess("")
			err := manager.ServiceStop(service)
			if err != nil {
				t.Errorf("ServiceStop(%s) failed: %v", service, err)
			}
		}
	})

	t.Run("ServiceRestart", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		for _, service := range services {
			client.OnExecute("sudo systemctl restart " + service).ReturnSuccess("")
			err := manager.ServiceRestart(service)
			if err != nil {
				t.Errorf("ServiceRestart(%s) failed: %v", service, err)
			}
		}
	})

	t.Run("ServiceStatus", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		output := `ActiveState=active
UnitFileState=enabled
Description=A high performance web server
MainPID=1234`

		client.OnExecute("systemctl show nginx --no-page").ReturnSuccess(output)

		status, err := manager.ServiceStatus("nginx")
		if err != nil {
			t.Errorf("ServiceStatus() failed: %v", err)
		}

		if !status.Active {
			t.Error("service should be active")
		}
		if !status.Enabled {
			t.Error("service should be enabled")
		}
		if status.MainPID != 1234 {
			t.Errorf("expected MainPID 1234, got %d", status.MainPID)
		}
	})

	t.Run("ServiceLogs", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		expectedLogs := "Jan 01 12:00:00 server nginx[1234]: Started"
		client.OnExecute("sudo journalctl -u nginx -n 50 --no-pager").ReturnSuccess(expectedLogs)

		logs, err := manager.ServiceLogs("nginx", 50)
		if err != nil {
			t.Errorf("ServiceLogs() failed: %v", err)
		}

		if logs != expectedLogs {
			t.Errorf("expected logs %q, got %q", expectedLogs, logs)
		}
	})

	t.Run("ServiceEnable", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		client.OnExecute("sudo systemctl enable nginx").ReturnSuccess("")

		err := manager.ServiceEnable("nginx")
		if err != nil {
			t.Errorf("ServiceEnable() failed: %v", err)
		}
	})

	t.Run("ServiceDisable", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		client.OnExecute("sudo systemctl disable nginx").ReturnSuccess("")

		err := manager.ServiceDisable("nginx")
		if err != nil {
			t.Errorf("ServiceDisable() failed: %v", err)
		}
	})
}

func TestSecurityManagement(t *testing.T) {
	t.Run("SetupFirewall with UFW", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock UFW is available
		client.OnExecute("which ufw").ReturnSuccess("/usr/sbin/ufw")
		// Mock UFW commands
		client.OnExecute("sudo ufw --force reset").ReturnSuccess("")
		client.OnExecute("sudo ufw default deny incoming").ReturnSuccess("")
		client.OnExecute("sudo ufw default allow outgoing").ReturnSuccess("")
		client.OnExecute("sudo ufw allow 22/tcp").ReturnSuccess("")
		client.OnExecute("sudo ufw allow 80/tcp").ReturnSuccess("")
		client.OnExecute("sudo ufw allow 443/tcp").ReturnSuccess("")
		client.OnExecute("sudo ufw --force enable").ReturnSuccess("")

		rules := []FirewallRule{
			{Port: 22, Protocol: "tcp", Action: "allow"},
			{Port: 80, Protocol: "tcp", Action: "allow"},
			{Port: 443, Protocol: "tcp", Action: "allow"},
		}

		err := manager.SetupFirewall(rules)
		if err != nil {
			t.Errorf("SetupFirewall() failed: %v", err)
		}

		history := client.GetExecutionHistory()
		// Should check for ufw and execute setup commands
		if len(history) < 7 {
			t.Errorf("expected at least 7 commands, got %d", len(history))
		}
	})

	t.Run("HardenSSH", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock SSH config commands
		client.OnExecute("sudo cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak").ReturnSuccess("")
		client.OnExecute("sudo echo").ReturnSuccess("") // Config write
		client.OnExecute("sudo sshd -t").ReturnSuccess("")
		client.OnExecute("sudo systemctl restart sshd").ReturnSuccess("")

		config := SSHConfig{
			PasswordAuth:        false,
			RootLogin:           false,
			PubkeyAuth:          true,
			MaxAuthTries:        3,
			ClientAliveInterval: 300,
			ClientAliveCountMax: 2,
		}

		err := manager.HardenSSH(config)
		if err != nil {
			t.Errorf("HardenSSH() failed: %v", err)
		}
	})

	t.Run("SetupFail2ban", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock package installation
		client.OnExecute("which apt").ReturnSuccess("/usr/bin/apt")
		client.OnExecute("sudo apt update && apt install -y fail2ban").ReturnSuccess("")
		// Mock fail2ban configuration
		client.OnExecute("sudo echo").ReturnSuccess("") // Config write
		client.OnExecute("sudo systemctl enable fail2ban").ReturnSuccess("")
		client.OnExecute("sudo systemctl restart fail2ban").ReturnSuccess("")

		err := manager.SetupFail2ban()
		if err != nil {
			t.Errorf("SetupFail2ban() failed: %v", err)
		}
	})
}

func TestDeployment(t *testing.T) {
	t.Run("Deploy from URL", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock deployment commands
		client.OnExecute("sudo cp -r '/opt/myapp' '/opt/myapp.bak").ReturnSuccess("") // Backup
		client.OnExecute("sudo systemctl stop myapp").ReturnSuccess("")               // Pre-deploy
		client.OnExecute("sudo wget -O /tmp/deploy.tar.gz 'https://example.com/app.tar.gz'").ReturnSuccess("")
		client.OnExecute("sudo mkdir -p '/opt/myapp' && tar -xzf /tmp/deploy.tar.gz -C '/opt/myapp'").ReturnSuccess("")
		client.OnExecute("sudo systemctl start myapp").ReturnSuccess("") // Post-deploy
		client.OnExecute("sudo systemctl restart myapp").ReturnSuccess("")
		client.OnExecute("curl -f -s 'http://localhost:8080/health'").ReturnSuccess("OK")

		app := AppConfig{
			Name:        "myapp",
			Version:     "v1.0.0",
			Source:      "https://example.com/app.tar.gz",
			Target:      "/opt/myapp",
			Service:     "myapp",
			Backup:      true,
			PreDeploy:   []string{"systemctl stop myapp"},
			PostDeploy:  []string{"systemctl start myapp"},
			HealthCheck: "http://localhost:8080/health",
		}

		err := manager.Deploy(app)
		if err != nil {
			t.Errorf("Deploy() failed: %v", err)
		}
	})

	t.Run("Rollback", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock rollback commands
		client.OnExecute("test -d '/opt/myapp.v0.9.0'").ReturnSuccess("")
		client.OnExecute("sudo systemctl stop myapp").ReturnSuccess("")
		client.OnExecute("sudo rm -rf '/opt/myapp' && cp -r '/opt/myapp.v0.9.0' '/opt/myapp'").ReturnSuccess("")
		client.OnExecute("sudo systemctl start myapp").ReturnSuccess("")

		err := manager.Rollback("myapp", "v0.9.0")
		if err != nil {
			t.Errorf("Rollback() failed: %v", err)
		}
	})
}

func TestPackageManagement(t *testing.T) {
	t.Run("InstallPackages with apt", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock apt is available
		client.OnExecute("which apt").ReturnSuccess("/usr/bin/apt")
		client.OnExecute("sudo apt update && apt install -y nginx git docker").ReturnSuccess("")

		err := manager.InstallPackages("nginx", "git", "docker")
		if err != nil {
			t.Errorf("InstallPackages() failed: %v", err)
		}
	})

	t.Run("InstallPackages with yum", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock apt not available, yum is
		client.OnExecute("which apt").ReturnFailure("", 1)
		client.OnExecute("which yum").ReturnSuccess("/usr/bin/yum")
		client.OnExecute("sudo yum install -y nginx git docker").ReturnSuccess("")

		err := manager.InstallPackages("nginx", "git", "docker")
		if err != nil {
			t.Errorf("InstallPackages() failed: %v", err)
		}
	})

	t.Run("UpdateSystem with apt", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		client.OnExecute("which apt").ReturnSuccess("/usr/bin/apt")
		client.OnExecute("sudo apt update && apt upgrade -y && apt autoremove -y").ReturnSuccess("")

		err := manager.UpdateSystem()
		if err != nil {
			t.Errorf("UpdateSystem() failed: %v", err)
		}
	})
}

func TestAdvancedOperations(t *testing.T) {
	t.Run("Batch operations", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Setup mock responses
		client.OnExecute("ls -la").ReturnSuccess("file list")
		client.OnExecute("pwd").ReturnSuccess("/home/user")
		client.OnExecute("whoami").ReturnSuccess("user")

		commands := []Command{
			Cmd("ls -la"),
			Cmd("pwd"),
			Cmd("whoami"),
		}

		result, err := manager.Batch(commands...)
		if err != nil {
			t.Errorf("Batch() failed: %v", err)
		}

		if len(result.Results) != 3 {
			t.Errorf("expected 3 results, got %d", len(result.Results))
		}

		if result.Results[0].Stdout != "file list" {
			t.Errorf("expected first result 'file list', got %q", result.Results[0].Stdout)
		}
	})

	t.Run("Transaction with rollback", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Setup for failure scenario
		client.OnExecute("test command 1").ReturnSuccess("")
		client.OnExecute("test command 2").ReturnFailure("command failed", 1)
		// Rollback commands
		client.OnExecute("sudo rm -f '/tmp/testfile'").ReturnSuccess("")

		rollbackExecuted := false
		err := manager.Transaction(func(tx *Transaction) error {
			// First command succeeds
			if err := tx.Execute("test command 1"); err != nil {
				return err
			}

			// Create a file (with rollback)
			if err := tx.CreateFile("/tmp/testfile", "content"); err != nil {
				return err
			}

			// This command fails
			if err := tx.Execute("test command 2"); err != nil {
				rollbackExecuted = true
				return err
			}

			return nil
		})

		if err == nil {
			t.Error("Transaction should have failed")
		}

		if !rollbackExecuted {
			t.Error("Rollback should have been triggered")
		}
	})
}

func TestSystemInfoManager(t *testing.T) {
	client := NewMockClient()
	client.Connect()
	manager := NewManager(client)

	// Mock system info commands
	client.OnExecute("lsb_release -a 2>/dev/null || cat /etc/os-release").ReturnSuccess(`PRETTY_NAME="Ubuntu 20.04 LTS"`)
	client.OnExecute("uname -r").ReturnSuccess("5.4.0-42-generic")
	client.OnExecute("uname -m").ReturnSuccess("x86_64")
	client.OnExecute("hostname").ReturnSuccess("test-server")
	client.OnExecute("nproc").ReturnSuccess("4")
	client.OnExecute("free -m | grep '^Mem:' | awk '{print $2}'").ReturnSuccess("8192")
	client.OnExecute("df -BG / | tail -1 | awk '{print $2}'").ReturnSuccess("100G")
	client.OnExecute("cat /proc/uptime | awk '{print $1}'").ReturnSuccess("86400.5")

	info, err := manager.SystemInfo()
	if err != nil {
		t.Errorf("SystemInfo() failed: %v", err)
	}

	if !strings.Contains(info.OS, "Ubuntu") {
		t.Errorf("expected OS to contain 'Ubuntu', got %q", info.OS)
	}
	if info.Kernel != "5.4.0-42-generic" {
		t.Errorf("expected kernel '5.4.0-42-generic', got %q", info.Kernel)
	}
	if info.Architecture != "x86_64" {
		t.Errorf("expected architecture 'x86_64', got %q", info.Architecture)
	}
	if info.Hostname != "test-server" {
		t.Errorf("expected hostname 'test-server', got %q", info.Hostname)
	}
	if info.CPUCount != 4 {
		t.Errorf("expected 4 CPUs, got %d", info.CPUCount)
	}
	if info.MemoryMB != 8192 {
		t.Errorf("expected 8192 MB memory, got %d", info.MemoryMB)
	}
	if info.DiskGB != 100 {
		t.Errorf("expected 100 GB disk, got %d", info.DiskGB)
	}
}

func TestCreateDirectory(t *testing.T) {
	client := NewMockClient()
	client.Connect()
	manager := NewManager(client)

	// Mock directory creation commands
	client.OnExecute("sudo mkdir -p '/opt/myapp'").ReturnSuccess("")
	client.OnExecute("sudo chmod 755 '/opt/myapp'").ReturnSuccess("")
	client.OnExecute("sudo chown www-data:www-data '/opt/myapp'").ReturnSuccess("")

	dir := Directory{
		Path:        "/opt/myapp",
		Permissions: "755",
		Owner:       "www-data",
		Group:       "www-data",
	}

	err := manager.CreateDirectory(dir)
	if err != nil {
		t.Errorf("CreateDirectory() failed: %v", err)
	}

	history := client.GetExecutionHistory()
	if len(history) != 3 {
		t.Errorf("expected 3 commands, got %d", len(history))
	}
}

func TestCheckPort(t *testing.T) {
	client := NewMockClient()
	client.Connect()
	manager := NewManager(client)

	t.Run("port open", func(t *testing.T) {
		client.OnExecute("ss -tuln | grep ':80 '").ReturnSuccess("LISTEN 0 128 *:80 *:*")

		open, err := manager.CheckPort(80)
		if err != nil {
			t.Errorf("CheckPort() failed: %v", err)
		}
		if !open {
			t.Error("port should be open")
		}
	})

	t.Run("port closed", func(t *testing.T) {
		client.OnExecute("ss -tuln | grep ':8080 '").ReturnFailure("", 1)

		open, err := manager.CheckPort(8080)
		if err != nil {
			t.Errorf("CheckPort() failed: %v", err)
		}
		if open {
			t.Error("port should be closed")
		}
	})
}

func TestGetInstalledPackages(t *testing.T) {
	t.Run("with dpkg", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		dpkgOutput := `ii  nginx     1.18.0-0ubuntu1.2  amd64  high performance web server
ii  git       1:2.25.1-1ubuntu3  amd64  fast, scalable distributed revision control`

		client.OnExecute("dpkg -l | grep '^ii'").ReturnSuccess(dpkgOutput)

		packages, err := manager.GetInstalledPackages()
		if err != nil {
			t.Errorf("GetInstalledPackages() failed: %v", err)
		}

		if len(packages) != 2 {
			t.Errorf("expected 2 packages, got %d", len(packages))
		}

		if packages[0].Name != "nginx" {
			t.Errorf("expected first package 'nginx', got %q", packages[0].Name)
		}
	})

	t.Run("with rpm", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// dpkg fails, try rpm
		client.OnExecute("dpkg -l | grep '^ii'").ReturnFailure("", 1)
		client.OnExecute("rpm -qa --qf '%{NAME}|%{VERSION}-%{RELEASE}\n'").ReturnSuccess("nginx|1.18.0-1.el8\ngit|2.27.0-1.el8\n")

		packages, err := manager.GetInstalledPackages()
		if err != nil {
			t.Errorf("GetInstalledPackages() failed: %v", err)
		}

		if len(packages) != 2 {
			t.Errorf("expected 2 packages, got %d", len(packages))
		}
	})
}

func TestRunScript(t *testing.T) {
	client := NewMockClient()
	client.Connect()
	manager := NewManager(client)

	// Mock script upload and execution
	client.OnUpload("/local/script.sh", "/tmp/script_").Succeed()
	client.OnExecute("sudo chmod +x").ReturnSuccess("")
	client.OnExecute("sudo /tmp/script").ReturnSuccess("script output")
	client.OnExecute("sudo rm -f").ReturnSuccess("")

	result, err := manager.RunScript("/local/script.sh", "arg1", "arg2")
	if err != nil {
		t.Errorf("RunScript() failed: %v", err)
	}

	if result.Stdout != "script output" {
		t.Errorf("expected output 'script output', got %q", result.Stdout)
	}
}

func TestTemplates(t *testing.T) {
	t.Run("WebServer template", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock package installation
		client.OnExecute("which apt").ReturnSuccess("/usr/bin/apt")
		client.OnExecute("sudo apt update && apt install -y nginx certbot python3-certbot-nginx").ReturnSuccess("")
		// Mock nginx configuration
		client.OnExecute("sudo echo").ReturnSuccess("") // Config write
		client.OnExecute("sudo ln -sf").ReturnSuccess("")
		client.OnExecute("sudo mkdir -p").ReturnSuccess("")
		client.OnExecute("sudo chmod").ReturnSuccess("")
		client.OnExecute("sudo chown").ReturnSuccess("")
		client.OnExecute("sudo systemctl restart nginx").ReturnSuccess("")
		client.OnExecute("sudo systemctl enable nginx").ReturnSuccess("")
		// Mock SSL setup
		client.OnExecute("sudo certbot").ReturnSuccess("")
		// Mock firewall
		client.OnExecute("which ufw").ReturnSuccess("/usr/sbin/ufw")
		client.OnExecute("sudo ufw").ReturnSuccess("")

		template := TemplateWebServer{
			Domain:   "example.com",
			SSL:      true,
			PHP:      false,
			Database: "none",
			Firewall: true,
		}

		err := manager.ApplyTemplate(template)
		if err != nil {
			t.Errorf("ApplyTemplate() failed: %v", err)
		}
	})

	t.Run("Docker template", func(t *testing.T) {
		client := NewMockClient()
		client.Connect()
		manager := NewManager(client)

		// Mock Docker installation
		installScript := `curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
rm get-docker.sh`
		client.OnExecute(installScript).ReturnSuccess("")
		// Mock Docker Compose installation
		client.OnExecute("sudo curl -L").ReturnSuccess("")
		// Mock Docker service
		client.OnExecute("sudo systemctl enable docker").ReturnSuccess("")
		client.OnExecute("sudo systemctl restart docker").ReturnSuccess("")

		template := TemplateDocker{
			ComposeVersion: true,
			Swarm:          false,
			Registry:       "",
		}

		err := manager.ApplyTemplate(template)
		if err != nil {
			t.Errorf("ApplyTemplate() failed: %v", err)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("boolToYesNo", func(t *testing.T) {
		if boolToYesNo(true) != "yes" {
			t.Error("boolToYesNo(true) should return 'yes'")
		}
		if boolToYesNo(false) != "no" {
			t.Error("boolToYesNo(false) should return 'no'")
		}
	})
}
