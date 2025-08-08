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
DeploymentZip string `json:"deployment_zip"` // Single zip containing binary and pb_public folder
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
func (sm *SSHManager) DeployApp(appName, remotePath, domain string, deploymentZip []byte) error
func (sm *SSHManager) UpdateApp(appName, remotePath string, deploymentZip []byte) error
func (sm *SSHManager) RollbackApp(appName, remotePath string, version *Version) error
func (sm *SSHManager) createSystemdService(appName, remotePath, domain string) error
func (sm *SSHManager) rsyncDeploymentFiles(localZipPath, remotePath string) error
func (sm *SSHManager) backupCurrentVersion(remotePath string) error
func (sm *SSHManager) restoreFromBackup(remotePath string) error
func (sm *SSHManager) cleanupBackup(remotePath string) error
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
// 1. Create deployment zip (binary + pb_public folder)
// 2. Upload zip to PocketBase storage
// 3. Download zip from PocketBase storage to local temp directory
// 4. Extract zip locally to prepare files for rsync
// 5. SSH to server as pocketbase user
// 6. Use rsync to sync files to /opt/pocketbase/apps/[app-name]/
// 7. Set proper file permissions (executable for binary)
// 8. Generate systemd service
// 9. Setup superuser (email/password from request)
// 10. Start service
// 11. Return success
}
```

### Update Deploy
```go
func (ds *DeploymentService) UpdateDeploy(appID string, req DeploymentRequest) error {
// 1. Stop service
// 2. Backup current version to temp folder (for quick rollback)
// 3. Download new deployment zip from PocketBase storage
// 4. Extract zip locally to prepare files for rsync
// 5. Use rsync to sync new files to remote server
// 6. Set proper file permissions
// 7. Start service
// 8. Verify health check passes
// 9. If successful: cleanup backup, if failed: restore from backup
}
```

### Rollback
```go
func (ds *DeploymentService) Rollback(appID, versionID string) error {
// 1. Stop service
// 2. Download previous deployment zip from PocketBase version storage
// 3. Extract zip locally to prepare files for rsync
// 4. Use rsync to restore files to remote server
// 5. Set proper file permissions
// 6. Start service
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
Create deployment zip → Store version → Download zip → Extract locally →
rsync to remote server → Create service → Setup superuser → Start → Done
```

### Update Deployment Flow
```
Stop service → Backup current version → Download zip → Extract locally → rsync to remote server → Start service → Health check → Success/Rollback
```

## 📁 File Transfer Process

The deployment system uses **rsync over SSH** to efficiently transfer files from local extraction to remote servers:

### **Transfer Steps:**
1. **Local Storage**: Deployment zip is stored in PocketBase file storage
2. **Download**: Download zip file from PocketBase storage to local temp directory
3. **Local Extraction**: Unzip files locally to prepare for rsync
4. **Rsync Transfer**: Use rsync to sync files to remote server via SSH
5. **Permission Setup**: Set executable permissions on binary via SSH
6. **Cleanup**: Remove local temp files

### **Transfer Functions:**
```go
func (sm *SSHManager) downloadAndExtractZip(versionID string, tempDir string) error
func (sm *SSHManager) rsyncToRemoteServer(localPath, remotePath string) error
func (sm *SSHManager) setRemotePermissions(remotePath string) error
func (sm *SSHManager) cleanupLocalTemp(tempDir string) error
func (sm *SSHManager) createVersionBackup(remotePath string) error
func (sm *SSHManager) restoreVersionBackup(remotePath string) error
```

### **Remote File Structure:**
```
/opt/pocketbase/apps/[app-name]/
├── pocketbase                 # Executable binary (chmod +x)
├── pb_public/                 # Static files directory
│   ├── index.html
│   ├── assets/
│   └── ...
├── logs/                      # Service logs
│   └── std.log
└── .backup/                   # Backup directory (for rollback)
    ├── pocketbase.backup      # Previous version backup
    └── pb_public.backup/      # Previous static files backup
```

### **Rsync Advantages:**
- **Incremental**: Only transfers changed files (faster updates)
- **Atomic**: File permissions and timestamps preserved
- **Efficient**: Built-in compression and delta transfer
- **Reliable**: Resume interrupted transfers, verify checksums
- **Standard**: Industry-standard deployment method

## 🔄 Backup & Rollback Strategy

For safe production deployments, the system implements automatic backup and rollback:

### **Update Deployment Safety:**
1. **Pre-deployment Backup**: Current version moved to `.backup/` folder
2. **Deploy New Version**: Rsync new files to main directory
3. **Health Check**: Verify new version works correctly
4. **Success**: Remove backup files, deployment complete
5. **Failure**: Restore from backup, restart service

### **Backup Process:**
```bash
# Before deploying new version
mv pocketbase .backup/pocketbase.backup
mv pb_public .backup/pb_public.backup

# Deploy new version with rsync
rsync -avz /local/new-version/ /opt/pocketbase/apps/myapp/

# If deployment fails, restore quickly
mv .backup/pocketbase.backup pocketbase
mv .backup/pb_public.backup pb_public
```

### **Backup Functions:**
```go
func (sm *SSHManager) SafeUpdateDeploy(appName, remotePath string, newVersion []byte) error {
    // 1. Stop service
    // 2. Create backup of current version
    // 3. Deploy new version
    // 4. Start service and health check
    // 5. If success: cleanup backup, if fail: restore backup
}

func (sm *SSHManager) emergencyRestore(remotePath string) error {
    // Quick restore from .backup/ folder for failed deployments
}
```

### **Rollback Benefits:**
- **Fast Recovery**: Seconds instead of minutes to restore service
- **Zero Downtime**: Quick swap between versions
- **Safe Deployments**: Always have working version available
- **Production Ready**: Battle-tested deployment strategy

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
