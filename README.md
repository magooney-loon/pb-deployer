# PocketBase Deploy Manager - Technical Breakdown

A personal deployment tool for automating PocketBase application deployment with server setup, security hardening, and Cloudflare DNS integration.

## 🏗️ Core Architecture

```
Svelte 5 UI ── PocketBase API ── SSH Manager ── Remote Servers
│ │ │ │
File Storage SQLite DB Cloudflare API Security Layer
(Versions) (Metadata) (DNS Only) (UFW/fail2ban)
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
- [ ] Cloudflare fail2ban integration

### Phase 4: Deployment Engine
- [ ] rsync file synchronization
- [ ] systemd service generation and management
- [ ] First deploy: upload → service → start
- [ ] Update deploy: stop → upload → start
- [ ] Rollback: stop → restore → start

### Phase 5: Cloudflare DNS
- [ ] Automatic A record creation
- [ ] DNS API integration
- [ ] Subdomain management (app.domain.com)

### Phase 6: Version Control
- [ ] File storage in PocketBase
- [ ] Version tracking and rollback
- [ ] Deployment history

### Phase 7: Svelte Frontend
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
ID string `json:"id"`
Name string `json:"name"`
ServerID string `json:"server_id"`
RemotePath string `json:"remote_path"`
ServiceName string `json:"service_name"`
Subdomain string `json:"subdomain"`
HealthURL string `json:"health_url"`
CurrentVersion string `json:"current_version"`
Status string `json:"status"` // online/offline via ping
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

type CloudflareZone struct {
ID string `json:"id"`
ZoneName string `json:"zone_name"`
ZoneID string `json:"zone_id"`
APIToken string `json:"api_token"`
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
func (sm *SSHManager) setupCloudflareFailban() error
```

### Deployment Functions
```go
func (sm *SSHManager) DeployApp(appName, remotePath string, binaryData, staticFiles []byte) error
func (sm *SSHManager) UpdateApp(appName, remotePath string, binaryData, staticFiles []byte) error
func (sm *SSHManager) RollbackApp(appName, remotePath string, version *Version) error
func (sm *SSHManager) createSystemdService(appName, remotePath string) error
func (sm *SSHManager) syncFiles(remotePath string, binaryData, staticFiles []byte) error
```

### Service Management
```go
func (sm *SSHManager) StartService(appName string) error
func (sm *SSHManager) StopService(appName string) error
func (sm *SSHManager) RestartService(appName string) error
func (sm *SSHManager) GetServiceStatus(appName string) (string, error)
```

### Cloudflare Integration
```go
type CloudflareManager struct {
APIToken string
ZoneID string
}

func (cm *CloudflareManager) CreateDNSRecord(subdomain, serverIP string) error
func (cm *CloudflareManager) DeleteDNSRecord(subdomain string) error
func (cm *CloudflareManager) UpdateDNSRecord(subdomain, newIP string) error
```

### Health Monitoring
```go
type HealthChecker struct{}

func (hc *HealthChecker) PingApp(healthURL string) (bool, error)
func (hc *HealthChecker) CheckAllApps(apps []App) map[string]bool
```

## 🚀 Deployment Workflows

### First Deploy
```go
func (ds *DeploymentService) FirstDeploy(req DeploymentRequest) error {
// 1. Upload files to PocketBase storage
// 2. SSH to server as pocketbase user
// 3. rsync files to /opt/pocketbase/apps/[app-name]/
// 4. Generate systemd service
// 5. Start service
// 6. Create DNS record
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

### Extended MVP (Phase 5-7)
1. DNS automation
2. Version control with rollback
3. Basic UI with real-time updates
4. Health ping monitoring

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
│ ├── handlers/ # API endpoints
│ └── cloudflare/ # DNS management
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
└── data/ # PocketBase data (gitignored)
```

## 🔄 Key Flows

### Server Setup Flow
```
Connect as root → Create pocketbase user → Setup SSH keys →
Create directories → Test connection → Apply security → Complete
```

### Deployment Flow
```
Upload files → Store version → SSH to server →
rsync files → Create/update service → Start → Create DNS → Done
```

### Update Flow
```
Stop service → rsync new files → Start service → Ping health → Success
```
