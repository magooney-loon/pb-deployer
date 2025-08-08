# PocketBase Deploy Manager - Technical Breakdown

A **production-only** deployment tool for automating PocketBase application deployment with server setup, security hardening, and Cloudflare DNS integration. Each app is deployed with its own domain and served in production environments.

## 🏗️ Core Architecture

```
Svelte 5 UI ── PocketBase API ── SSH Manager ── Remote Servers
│ │ │ │
File Storage SQLite DB Security Layer
(Versions) (Metadata) (UFW/fail2ban)
```

## 📋 Essential Features Only

### Phase 1: Core Backend
- [ ] PocketBase initialization with file upload
- [ ] SSH agent integration + manual key fallback
- [ ] Basic server CRUD operations
- [ ] WebSocket support for real-time updates

### Phase 2: Server Setup Wizard
- [ ] Root SSH connection and validation
- [ ] Create `pocketbase` user with SSH keys
- [ ] Directory structure: `/opt/pocketbase/apps/`
- [ ] Real-time setup progress via WebSocket

### Phase 3: Security Lockdown
- [ ] UFW firewall (22, 80, 443)
- [ ] fail2ban SSH protection
- [ ] SSH hardening (disable root, key-only auth)

### Phase 4: Deployment Engine
- [ ] rsync file synchronization
- [ ] systemd service generation and management
- [ ] Superuser setup (first deploy only)
- [ ] First deploy: upload → service → superuser setup → start
- [ ] Update deploy: stop → upload → start
- [ ] Rollback: stop → restore → start

### Phase 5: Version Control
- [ ] File storage in PocketBase
- [ ] Version tracking and rollback
- [ ] Deployment history

### Phase 6: Svelte Frontend
- [ ] Server management interface
- [ ] Deployment wizard with file upload
- [ ] Real-time deployment progress
- [ ] Simple health status (ping only)

## 🗄️ Database Schema

```go
// Core collections
type Server struct {
ID string `json:"id"`
Name string `json:"name"`
Host string `json:"host"`
Port int `json:"port"`
RootUsername string `json:"root_username"`
AppUsername string `json:"app_username"`
UseSSHAgent bool `json:"use_ssh_agent"`
ManualKeyPath string `json:"manual_key_path"`
SetupComplete bool `json:"setup_complete"`
SecurityLocked bool `json:"security_locked"`
}

type App struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ServerID       string `json:"server_id"`
	RemotePath     string `json:"remote_path"`
	ServiceName    string `json:"service_name"`
	Domain         string `json:"domain"`         // Production domain (e.g., "myapp.example.com")
	CurrentVersion string `json:"current_version"`
	Status         string `json:"status"` // online/offline via /api/health ping
}

type Version struct {
ID string `json:"id"`
AppID string `json:"app_id"`
VersionNum string `json:"version_number"`
BinaryFile string `json:"binary_file"` // PB file field
StaticFiles string `json:"static_files"` // PB file field
Notes string `json:"notes"`
}

type Deployment struct {
ID string `json:"id"`
AppID string `json:"app_id"`
VersionID string `json:"version_id"`
Status string `json:"status"` // pending/running/success/failed
Logs string `json:"logs"`
StartedAt string `json:"started_at"`
CompletedAt string `json:"completed_at"`
}

```

## 🔧 Core Technical Components

### SSH Manager
```go
type SSHManager struct {
server *Server
conn *ssh.Client
}

// Core methods
func NewSSHManager(server *Server, asRoot bool) (*SSHManager, error)
func (sm *SSHManager) ExecuteCommand(command string) (string, error)
func (sm *SSHManager) ExecuteCommandStream(command string, output chan<- string) error
func (sm *SSHManager) Close()
```

### Server Setup Functions
```go
func (sm *SSHManager) RunServerSetup(progressChan chan<- SetupStep) error
func (sm *SSHManager) createPocketbaseUser() error
func (sm *SSHManager) setupSSHKeys() error
func (sm *SSHManager) createDirectories() error
func (sm *SSHManager) testUserConnection() error
```

### Security Lockdown
```go
func (sm *SSHManager) ApplySecurityLockdown(progressChan chan<- SetupStep) error
func (sm *SSHManager) setupFirewall() error
func (sm *SSHManager) setupFail2ban() error
func (sm *SSHManager) hardenSSH() error

```

### Deployment Functions
```go
func (sm *SSHManager) DeployApp(appName, remotePath, domain string, binaryData, staticFiles []byte) error
func (sm *SSHManager) UpdateApp(appName, remotePath string, binaryData, staticFiles []byte) error
func (sm *SSHManager) RollbackApp(appName, remotePath string, version *Version) error
func (sm *SSHManager) createSystemdService(appName, remotePath, domain string) error
func (sm *SSHManager) syncFiles(remotePath string, binaryData, staticFiles []byte) error
func (sm *SSHManager) setupSuperuser(appName, email, password string) error
```

### Service Management
```go
func (sm *SSHManager) StartService(appName string) error
func (sm *SSHManager) StopService(appName string) error
func (sm *SSHManager) RestartService(appName string) error
func (sm *SSHManager) GetServiceStatus(appName string) (string, error)
```

### Health Monitoring
```go
type HealthChecker struct{}

func (hc *HealthChecker) PingApp(domain string) (bool, error) // Checks https://{domain}/api/health
func (hc *HealthChecker) CheckAllApps(apps []App) map[string]bool
func (hc *HealthChecker) GetHealthURL(domain string) string // Returns https://{domain}/api/health
```

**Standard Health Endpoint**: All PocketBase applications expose a standardized health check at `/api/health` that returns JSON status information.

## 🚀 Deployment Workflows

### First Deploy
```go
func (ds *DeploymentService) FirstDeploy(req DeploymentRequest) error {
// 1. Upload files to PocketBase storage
// 2. SSH to server as pocketbase user
// 3. rsync files to /opt/pocketbase/apps/[app-name]/
// 4. Generate systemd service
// 5. Setup superuser (email/password from request)
// 6. Start service
// 7. Return success
}
```

### Update Deploy
```go
func (ds *DeploymentService) UpdateDeploy(appID string, req DeploymentRequest) error {
// 1. Stop service
// 2. rsync new files
// 3. Start service
// 4. Verify health
}
```

### Rollback
```go
func (ds *DeploymentService) Rollback(appID, versionID string) error {
// 1. Stop service
// 2. Download files from version storage
// 3. rsync files to server
// 4. Start service
}
```

## 🎯 Technical Priorities

### MVP Core (Phase 1-4)
1. SSH connection with agent auth
2. Server setup wizard
3. Basic deployment (upload → service → start)
4. Simple service management

### Extended MVP (Phase 5-6)
1. Version control with rollback
2. Basic UI with real-time updates
3. Health monitoring via `/api/health` endpoint

## 📁 Project Structure (Simplified)

```
pb-deploy-manager/
├── main.go # PocketBase entry point
├── go.mod
├──
├── internal/
│ ├── models/ # Data models
│ ├── ssh/ # SSH operations
│ ├── services/ # Business logic
│ └── handlers/ # API endpoints
│
├── web/ # Svelte frontend
│ ├── src/
│ │ ├── routes/ # Pages
│ │ ├── lib/components/ # UI components
│ │ ├── lib/stores/ # State management
│ │ └── lib/api/ # API client
│ └── static/
│
├── templates/ # Config templates
│ └── systemd/
│ └── pocketbase.service.tmpl
│
└── configs/ # Example systemd service
  └── pocketbase.service.example
```

## 🔄 Key Flows

### Server Setup Flow
```
Connect as root → Create pocketbase user → Setup SSH keys →
Create directories → Test connection → Apply security → Complete
```

### First Deployment Flow
```
Upload files → Store version → SSH to server →
rsync files → Create service → Setup superuser → Start → Done
```

### Update Deployment Flow
```
Stop service → rsync new files → Start service → Ping health → Success
```

## 🔧 Systemd Service Template

```ini
[Unit]
Description = pocketbase-{APP_NAME}

[Service]
Type             = simple
User             = pocketbase
Group            = pocketbase
LimitNOFILE      = 4096
Restart          = always
RestartSec       = 5s
StandardOutput   = append:/opt/pocketbase/apps/{APP_NAME}/logs/std.log
StandardError    = append:/opt/pocketbase/apps/{APP_NAME}/logs/std.log
WorkingDirectory = /opt/pocketbase/apps/{APP_NAME}
ExecStart        = /opt/pocketbase/apps/{APP_NAME}/pocketbase serve {DOMAIN}

[Install]
WantedBy = multi-user.target
```

## 🔐 Superuser Setup Process

For first deployments, the system will:
1. Deploy the PocketBase binary and files
2. Create the systemd service (but don't start it yet)
3. Run `./pocketbase superuser create EMAIL PASS` to setup the admin user
4. Start the service
5. The app will be accessible immediately with the created superuser

**Note**: The superuser email and password must be provided during the first deployment request.

## 🔍 Health Monitoring

All deployed PocketBase applications expose a standardized health endpoint:

**Endpoint**: `GET /api/health`
**URL Format**: `https://{domain}/api/health`
