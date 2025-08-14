# Handlers Package

REST API handlers for PocketBase application deployment management with real-time WebSocket support.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Servers     │───▶│      Apps       │───▶│    Versions     │───▶│   Deployments   │
│   /api/servers  │    │   /api/apps     │    │  /api/versions  │    │ /api/deployments│
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │                       │
         ▼                       ▼                       ▼                       ▼
    SSH Operations        Service Management       File Management        Process Tracking

                                    Manager Services
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Setup Manager  │    │ Service Manager │    │Security Manager │    │Deployment Mgr   │
│   /api/v1/setup │    │ /api/v1/services│    │/api/v1/security │    │/api/v1/deployments│
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │                       │
         ▼                       ▼                       ▼                       ▼
   User & Directory        Systemd Services        SSH & Firewall         App Deployment
   Package Management      Log Management          Fail2ban & Audit       Health & Rollback
```

## API Endpoints

### Servers (`/api/servers`)

**Connection & Status**
```http
POST   /servers/{id}/test       # Comprehensive connection test
GET    /servers/{id}/status     # Server status check
GET    /servers/{id}/health     # Connection pool health
```

**Server Lifecycle**
```http
POST   /servers/{id}/setup      # Initial server setup
POST   /servers/{id}/security   # Security hardening
```

**Real-time Progress**
```http
GET    /servers/{id}/setup-ws   # Setup progress WebSocket
GET    /servers/{id}/security-ws # Security progress WebSocket
```

**Manager Integration**
```http
POST   /servers/{id}/managers/setup    # Integrated setup via Setup Manager
POST   /servers/{id}/managers/security # Integrated security via Security Manager
```

### Apps (`/api/apps`)

**CRUD Operations**
```http
GET    /apps                    # List apps (filter: ?server_id=)
POST   /apps                    # Create app
GET    /apps/{id}               # Get app details
PUT    /apps/{id}               # Update app
DELETE /apps/{id}               # Delete app
```

**Status & Health**
```http
GET    /apps/{id}/status        # App status
POST   /apps/{id}/health-check  # Trigger health check
GET    /apps/{id}/logs          # Service logs (?lines=100)
```

**Service Management**
```http
POST   /apps/{id}/start         # Start app service
POST   /apps/{id}/stop          # Stop app service
POST   /apps/{id}/restart       # Restart app service
```

**Deployment Operations**
```http
POST   /apps/{id}/deploy        # Deploy version
POST   /apps/{id}/rollback      # Rollback to version
GET    /apps/{id}/deploy-ws     # Deployment progress WebSocket
```

### Versions (`/api/versions`)

**CRUD Operations**
```http
GET    /versions                # List versions (filter: ?app_id=)
POST   /versions                # Create version
GET    /versions/{id}           # Get version details
PUT    /versions/{id}           # Update version
DELETE /versions/{id}           # Delete version
```

**File Management**
```http
POST   /versions/{id}/upload    # Upload deployment ZIP
GET    /versions/{id}/download  # Download deployment ZIP
POST   /versions/{id}/validate  # Validate deployment package
```

**App-Specific Versions**
```http
GET    /apps/{app_id}/versions  # List app versions
POST   /apps/{app_id}/versions  # Create app version
```

**Metadata**
```http
GET    /versions/{id}/metadata  # Get version metadata
PUT    /versions/{id}/metadata  # Update version metadata
```

### Deployments (`/api/deployments`)

**Listing & Details**
```http
GET    /deployments             # List deployments (filter: ?app_id=, ?status=)
GET    /deployments/{id}        # Get deployment details
GET    /deployments/{id}/status # Get deployment status
GET    /deployments/{id}/logs   # Get deployment logs
```

**Process Control**
```http
POST   /deployments/{id}/cancel # Cancel running deployment
POST   /deployments/{id}/retry  # Retry failed deployment
```

**App-Specific Deployments**
```http
GET    /apps/{app_id}/deployments        # List app deployments
GET    /apps/{app_id}/deployments/latest # Get latest deployment
```

**Analytics & Management**
```http
GET    /deployments/stats       # Deployment statistics (?days=30)
POST   /deployments/cleanup     # Cleanup old deployments (?keep_days=90, ?dry_run=true)
```

**Real-time Progress**
```http
GET    /deployments/{id}/ws     # Deployment progress WebSocket
```

### Manager Services (`/api/v1/*`)

**Setup Manager**
```http
POST   /setup/users             # Create system users
POST   /setup/users/{username}/ssh-keys # Configure SSH keys
POST   /setup/directories       # Create directories
POST   /setup/system-users      # Complete user setup
GET    /setup/users             # List users
```

**Service Manager**
```http
POST   /services/{name}/action  # Start/stop/restart services
GET    /services/{name}         # Service status
GET    /services/{name}/logs    # Service logs
POST   /services                # Create service files
GET    /services                # List services
```

**Security Manager**
```http
POST   /security/lockdown       # Full security lockdown
POST   /security/firewall       # Configure firewall
POST   /security/fail2ban       # Setup fail2ban
GET    /security/audit          # Security audit
GET    /security/status         # Security status
```

**Deployment Manager**
```http
POST   /deployments             # Deploy application
POST   /deployments/{name}/rollback # Rollback deployment
GET    /deployments/{name}      # Deployment status
POST   /deployments/validate    # Validate deployment
GET    /deployments/{name}/health # Health check
```

## Request/Response Formats

### Server Connection Test
```http
POST /api/servers/{id}/test
```

**Response:**
```json
{
  "success": true,
  "tcp_connection": {
    "success": true,
    "latency": "15.23ms"
  },
  "root_ssh_connection": {
    "success": false,
    "username": "root",
    "error": "Root SSH access disabled after security lockdown"
  },
  "app_ssh_connection": {
    "success": true,
    "username": "pocketbase",
    "auth_method": "ssh_agent"
  },
  "overall_status": "healthy_secured"
}
```

### App Creation
```http
POST /api/apps
Content-Type: application/json

{
  "name": "my-app",
  "server_id": "abc123",
  "domain": "myapp.example.com"
}
```

**Response:**
```json
{
  "id": "def456",
  "name": "my-app",
  "server_id": "abc123",
  "domain": "myapp.example.com",
  "remote_path": "/opt/pocketbase/apps/my-app",
  "service_name": "pocketbase-my-app",
  "status": "offline"
}
```

### Version Upload
```http
POST /api/versions/{id}/upload
Content-Type: multipart/form-data

pocketbase_binary: <binary_file>
pb_public_files: <file1>, <file2>, ...
```

**Response:**
```json
{
  "message": "Version files uploaded successfully",
  "version_id": "ghi789",
  "binary_size": 12345678,
  "public_files_count": 15,
  "deployment_file": "deployment_1.0.0_1703123456.zip",
  "deployment_size": 15234567
}
```

### Deployment Request
```http
POST /api/apps/{id}/deploy
Content-Type: application/json

{
  "version_id": "ghi789",
  "superuser_email": "admin@example.com",    // First deploy only
  "superuser_password": "secure_password",   // First deploy only
  "notes": "Production deployment"
}
```

**Response:**
```json
{
  "message": "Deployment started",
  "deployment_id": "jkl012",
  "app_id": "def456",
  "version_id": "ghi789",
  "is_first_deploy": true
}
```

## Real-time Updates

### WebSocket Subscriptions
All long-running operations support real-time progress via PocketBase realtime system:

- `server_setup_{server_id}` - Server setup progress
- `server_security_{server_id}` - Security lockdown progress
- `app_deployment_{app_id}` - App deployment progress
- `deployment_progress_{deployment_id}` - Specific deployment progress

### Progress Message Format
```json
{
  "step": "create_user",
  "status": "running",
  "message": "Creating pocketbase user",
  "details": "Setting up user with sudo access",
  "timestamp": "2023-12-07T10:30:00Z",
  "progress_pct": 25
}
```

## Business Logic

### Server Lifecycle
1. **Fresh Server**: `setup_complete=false, security_locked=false`
2. **Setup**: POST `/servers/{id}/setup` → Setup Manager creates users, directories, installs packages
3. **Security**: POST `/servers/{id}/security` → Security Manager lockdown, firewall, fail2ban
4. **Ready**: `setup_complete=true, security_locked=true` → Can deploy applications

### Manager Integration Workflow
1. **Server Setup**: Handlers → Setup Manager → User/directory creation, package installation
2. **Security Lockdown**: Handlers → Security Manager → SSH hardening, firewall, intrusion prevention
3. **App Deployment**: Handlers → Deployment Manager → Service management, health checks
4. **Service Management**: Handlers → Service Manager → Systemd operations, log retrieval

### App Deployment Flow
1. **Create Version**: Upload binary + public files as ZIP
2. **Trigger Deploy**: POST `/apps/{id}/deploy` with version_id
3. **Monitor Progress**: WebSocket subscription for real-time updates
4. **Status Updates**: App status updated based on deployment result

### Service Management
Security-aware service operations:
- **Pre-security**: Direct systemctl via root SSH
- **Post-security**: systemctl via app user + sudo

### File Handling
- **Upload Limit**: 150MB total (100MB binary + 50MB public)
- **Format**: ZIP containing `pocketbase` binary + `pb_public/` folder
- **Storage**: PocketBase filesystem with automatic cleanup

## Error Handling

### HTTP Status Patterns
- `400 Bad Request`: Invalid parameters, missing required fields
- `404 Not Found`: Resource doesn't exist
- `409 Conflict`: Duplicate names, version conflicts
- `500 Internal Server Error`: SSH failures, database errors

### Error Response Format
```json
{
  "error": "Human-readable error message",
  "details": "Technical details for debugging",
  "suggestion": "Actionable resolution steps"
}
```

### Async Operation Handling
Long-running operations (setup, security, deployment) follow pattern:
1. **Immediate Response**: `202 Accepted` with operation ID
2. **Background Execution**: Process runs asynchronously
3. **Progress Updates**: Real-time WebSocket notifications
4. **Final Status**: Database record updated with result

## Security Considerations

### SSH Connection Management
- **Connection Pooling**: Automatic connection reuse and health monitoring
- **Security-Aware**: Automatically switches to app user after lockdown
- **Timeout Handling**: Context-based timeouts prevent hanging operations

### Authentication Flow
- **Pre-Security**: Root SSH access for setup operations
- **Post-Security**: App user SSH with sudo for privileged operations
- **Validation**: Connection pre-validation before disabling root access

### File Security
- **Upload Validation**: File type and size restrictions
- **Path Sanitization**: Prevents directory traversal
- **Cleanup**: Automatic removal of temporary files

## Performance Optimizations

### Connection Reuse
- **Connection Pool**: 50-80% faster subsequent operations
- **Health Monitoring**: Proactive failure detection and recovery
- **Resource Management**: Automatic cleanup of stale connections

### Async Processing
- **Background Operations**: Long-running tasks don't block API
- **Progress Streaming**: Real-time updates without polling
- **Graceful Degradation**: Continues operation if WebSocket fails

### Efficient Queries
- **Filtered Listing**: Database-level filtering for large datasets
- **Pagination Support**: Configurable limits for large result sets
- **Selective Loading**: Only load required data for list operations

## Integration Points

### SSH Service Integration
```go
sshService := ssh.GetSSHService()
err := sshService.RunServerSetup(server, progressChan)
```

### Manager Service Integration
```go
// Setup Manager Integration
setupMgr := managers.GetSetupManager(sshService)
err := setupMgr.CreateSystemUser(server, userConfig)
err := setupMgr.SetupDirectories(server, dirConfigs)
err := setupMgr.InstallPackages(server, packages)

// Service Manager Integration  
serviceMgr := managers.GetServiceManager(sshService)
err := serviceMgr.ManageService(server, serviceName, "restart")
status := serviceMgr.GetServiceStatus(server, serviceName)
logs := serviceMgr.GetServiceLogs(server, serviceName, 100)

// Security Manager Integration
securityMgr := managers.GetSecurityManager(sshService)
err := securityMgr.ApplySecurityLockdown(server, securityConfig)
audit := securityMgr.PerformSecurityAudit(server)

// Deployment Manager Integration
deployMgr := managers.GetDeploymentManager(sshService, serviceMgr)
err := deployMgr.Deploy(server, deploymentSpec)
health := deployMgr.CheckHealth(server, appName)
```

### PocketBase Integration
```go
// Database operations
record, err := app.FindRecordById("apps", appID)
err := app.Save(record)

// File operations
filesystem, err := app.NewFilesystem()
err := filesystem.Serve(response, request, key, filename)

// Real-time notifications
app.SubscriptionsBroker().ChunkedClients(300)
```

### Model Conversion
```go
// PocketBase record → Model struct
server := &models.Server{
    ID:             record.Id,
    Host:           record.GetString("host"),
    SecurityLocked: record.GetBool("security_locked"),
}
```

## Usage Patterns

### Handler Registration
```go
func RegisterHandlers(app core.App) {
    apiGroup := e.Router.Group("/api")
    v1Group := e.Router.Group("/api/v1")
    
    // Core handlers
    server.RegisterServerHandlers(app, apiGroup)
    apps.RegisterAppsHandlers(app, apiGroup)
    version.RegisterVersionHandlers(app, apiGroup)
    deployment.RegisterDeploymentHandlers(app, apiGroup)
    
    // Manager handlers
    setup.RegisterSetupManagerHandlers(app, v1Group)
    service.RegisterServiceManagerHandlers(app, v1Group)
    security.RegisterSecurityManagerHandlers(app, v1Group)
    deploymentv1.RegisterDeploymentManagerHandlers(app, v1Group)
}
```

### Progress Monitoring
```go
progressChan := make(chan ProgressUpdate, 10)
go func() {
    for update := range progressChan {
        notifyProgress(app, update.Operation, update.ServerID, update)
    }
}()
```

### Error Propagation
```go
if err != nil {
    app.Logger().Error("Operation failed", "context", value, "error", err)
    return e.JSON(http.StatusInternalServerError, map[string]string{
        "error": "User-friendly message",
    })
}
```

## API Contracts

### Query Parameters
- `server_id`: Filter resources by server
- `app_id`: Filter resources by app
- `status`: Filter by status enum
- `limit`: Result pagination (max: 100)
- `lines`: Log line count (default: 100)
- `days`: Statistics time range (default: 30)

### Path Parameters
- `{id}`: Resource identifier (15-char PocketBase ID)
- `{app_id}`: App identifier for nested resources
- `{server_id}`: Server identifier for nested resources

### Content Types
- `application/json`: Standard API requests/responses
- `multipart/form-data`: File uploads
- `application/octet-stream`: File downloads

## Testing Considerations

### Connection Testing
- **Comprehensive**: TCP + SSH (root + app user)
- **Context Timeouts**: Prevents hanging tests
- **Security-Aware**: Expects root SSH failure on locked servers
- **Detailed Results**: Separate results for each connection type

### Background Operations
- **Progress Tracking**: All async operations provide progress updates
- **Error Handling**: Failures logged and propagated via WebSocket
- **Resource Cleanup**: Automatic cleanup on operation completion

### File Operations
- **Size Limits**: 150MB total upload limit
- **Format Validation**: ZIP structure and content validation
- **Storage Integration**: PocketBase filesystem for file management

## Deployment Patterns

### First Deployment
Traditional approach requires superuser credentials:
```json
{
  "version_id": "abc123",
  "superuser_email": "admin@example.com",
  "superuser_password": "secure_password"
}
```

Manager-based deployment with automated setup:
```json
{
  "name": "my-app",
  "version": "1.0.0",
  "environment": "production",
  "strategy": "rolling",
  "artifact_path": "/path/to/deployment.zip",
  "service_name": "pocketbase-my-app",
  "health_check_url": "http://localhost:8080/api/health"
}
```

### Subsequent Deployments
Only requires version reference:
```json
{
  "version_id": "def456",
  "notes": "Bug fixes and improvements"
}
```

### Rollback Operations
Uses same deployment mechanism with previous version:
```json
{
  "version_id": "abc123",
  "notes": "Rolling back due to issues"
}
```

## Monitoring & Analytics

### Health Endpoints
- **Connection Health**: Real-time SSH connection pool status
- **App Health**: HTTP health check results
- **Service Status**: Systemd service state

### Statistics
- **Deployment Metrics**: Success rates, average duration
- **App Distribution**: Deployment counts by app
- **Status Breakdown**: Pending/running/success/failed counts

### Cleanup Operations
- **Old Deployments**: Configurable retention (default: 90 days)
- **Dry Run Support**: Preview cleanup without execution
- **Bulk Operations**: Efficient cleanup of multiple records

## Error Recovery

### Connection Failures
- **Automatic Retry**: Exponential backoff for transient failures
- **Health Recovery**: Connection pool automatically recovers failed connections
- **Graceful Degradation**: Operations continue with available connections

### Deployment Failures
- **Retry Mechanism**: Failed deployments can be retried
- **Status Tracking**: Detailed failure reason in logs
- **Rollback Support**: Automatic rollback on critical failures

### File Operation Failures
- **Upload Validation**: Pre-upload size and format checks
- **Atomic Operations**: Upload completion before database updates
- **Cleanup on Failure**: Automatic cleanup of partial uploads

## Security Model

### Pre-Security State
- **Root Access**: Full SSH access for setup operations
- **Service Management**: Direct systemctl via root
- **File Operations**: Root-level file system access

### Post-Security State
- **App User Only**: Root SSH disabled, app user + sudo
- **Privilege Escalation**: Automatic sudo for privileged operations
- **Audit Trail**: All privileged operations logged

### Connection Management
- **SSH Key Management**: Automatic key setup and validation via Setup Manager
- **Host Key Handling**: Automatic acceptance with security logging
- **Connection Validation**: Pre-security lockdown validation via Security Manager
- **Local Tool Access**: Direct manager service access (no authentication required)
