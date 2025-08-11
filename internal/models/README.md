# Models Package

Data models and database schema for PocketBase application deployment management.

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Server    │───▶│     App     │───▶│   Version   │───▶│ Deployment  │
│(server.go)  │    │ (app.go)    │    │(version.go) │    │(deployment) │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
      │                   │                   │                   │
      ▼                   ▼                   ▼                   ▼
  SSH Target        App Instance        Code Package       Deploy Operation
```

## Models

### Server (`server.go`)
Remote deployment targets with SSH configuration.

```go
type Server struct {
    ID             string    // PocketBase record ID
    Name           string    // Human-readable name
    Host           string    // SSH hostname/IP
    Port           int       // SSH port (default: 22)
    RootUsername   string    // Root user (default: "root")
    AppUsername    string    // App user (default: "pocketbase")
    UseSSHAgent    bool      // SSH agent authentication
    ManualKeyPath  string    // Custom private key path
    SetupComplete  bool      // Initial setup finished
    SecurityLocked bool      // Security hardening applied
    Created        time.Time // Auto-generated
    Updated        time.Time // Auto-updated
}
```

**Key Methods:**
```go
server := models.NewServer()
server.GetSSHAddress()              // "host" or "host:port"
server.IsReadyForDeployment()       // setup_complete && security_locked
server.IsSetupComplete()            // setup status check
server.IsSecurityLocked()           // security status check
```

### App (`app.go`)
PocketBase application instances deployed on servers.

```go
type App struct {
    ID             string    // PocketBase record ID
    Name           string    // Application name
    ServerID       string    // Foreign key to Server
    RemotePath     string    // Deployment path on server
    ServiceName    string    // Systemd service name
    Domain         string    // Production domain
    CurrentVersion string    // Active version identifier
    Status         string    // "online"/"offline"/"unknown"
    Created        time.Time // Auto-generated
    Updated        time.Time // Auto-updated
}
```

**Key Methods:**
```go
app := models.NewApp()
app.GetHealthURL()                  // "https://domain/api/health"
app.IsOnline()                      // status == "online"
```

### Version (`version.go`)
Versioned deployment packages for applications.

```go
type Version struct {
    ID            string    // PocketBase record ID
    AppID         string    // Foreign key to App
    VersionNum    string    // Version identifier (e.g., "1.0.0")
    DeploymentZip string    // Zip file with binary + pb_public
    Notes         string    // Release notes
    Created       time.Time // Auto-generated
    Updated       time.Time // Auto-updated
}
```

**Key Methods:**
```go
version := models.NewVersion()
version.HasDeploymentZip()          // deployment_zip != ""
version.HasNotes()                  // notes != ""
version.GetVersionString()          // formatted version or "unknown"
```

### Deployment (`deployment.go`)
Deployment operation tracking and lifecycle management.

```go
type Deployment struct {
    ID          string     // PocketBase record ID
    AppID       string     // Foreign key to App
    VersionID   string     // Foreign key to Version
    Status      string     // "pending"/"running"/"success"/"failed"
    Logs        string     // Deployment logs (100KB max)
    StartedAt   *time.Time // Deployment start time
    CompletedAt *time.Time // Deployment completion time
    Created     time.Time  // Auto-generated
    Updated     time.Time  // Auto-updated
}
```

**Lifecycle Methods:**
```go
deployment := models.NewDeployment()
deployment.MarkAsRunning()          // status -> "running", set StartedAt
deployment.MarkAsSuccess()          // status -> "success", set CompletedAt
deployment.MarkAsFailed()           // status -> "failed", set CompletedAt
```

**Status Checks:**
```go
deployment.IsRunning()              // status == "running"
deployment.IsComplete()             // "success" || "failed"
deployment.IsSuccessful()           // status == "success"
deployment.IsFailed()               // status == "failed"
```

**Utility Methods:**
```go
deployment.GetDuration()            // *time.Duration if completed
deployment.AppendLog("message")     // append to logs with newline
```

## Database Schema

### Collection Setup (`collections.go`)
Auto-creates PocketBase collections with proper field types and validation.

```go
// Initialize all collections
err := models.InitializeDatabase(app)

// Manual registration
err := models.RegisterCollections(app)
```

### Field Specifications

**Servers Collection:**
- `name`: TextField (required, max: 255)
- `host`: TextField (required, max: 255)
- `port`: NumberField (1-65535)
- `root_username`: TextField (max: 50)
- `app_username`: TextField (max: 50)
- `use_ssh_agent`: BoolField
- `manual_key_path`: TextField (max: 500)
- `setup_complete`: BoolField
- `security_locked`: BoolField

**Apps Collection:**
- `name`: TextField (required, max: 255)
- `server_id`: TextField (required, max: 15)
- `remote_path`: TextField (max: 500)
- `service_name`: TextField (max: 100)
- `domain`: TextField (max: 255)
- `current_version`: TextField (max: 100)
- `status`: SelectField ("online", "offline", "unknown")

**Versions Collection:**
- `app_id`: TextField (required, max: 15)
- `version_number`: TextField (max: 50)
- `deployment_zip`: FileField (150MB max, ZIP only)
- `notes`: TextField (max: 1000)

**Deployments Collection:**
- `app_id`: TextField (required, max: 15)
- `version_id`: TextField (max: 15)
- `status`: SelectField ("pending", "running", "success", "failed")
- `logs`: TextField (max: 100KB)
- `started_at`: DateField
- `completed_at`: DateField

## Data Relationships

```
Server (1) ──── (N) App (1) ──── (N) Version
                 │                      │
                 └──── (N) Deployment ──┘
```

### Foreign Key Constraints
- `App.ServerID` → `Server.ID`
- `Version.AppID` → `App.ID`
- `Deployment.AppID` → `App.ID`
- `Deployment.VersionID` → `Version.ID`

## Deployment Lifecycle

### Server Setup Flow
```go
server := models.NewServer()
// server.SetupComplete = false, SecurityLocked = false

// 1. Initial setup
ssh.RunServerSetup(server, progressChan)
// server.SetupComplete = true

// 2. Security hardening
ssh.ApplySecurityLockdown(server, progressChan)
// server.SecurityLocked = true

// 3. Ready for deployment
server.IsReadyForDeployment() // true
```

### Application Deployment Flow
```go
// 1. Create version
version := models.NewVersion()
version.AppID = app.ID
version.DeploymentZip = "uploaded_file_id"

// 2. Create deployment
deployment := models.NewDeployment()
deployment.AppID = app.ID
deployment.VersionID = version.ID

// 3. Execute deployment
deployment.MarkAsRunning()
deployment.AppendLog("Starting deployment...")

// ... deployment process ...

if success {
    deployment.MarkAsSuccess()
    app.CurrentVersion = version.VersionNum
    app.Status = "online"
} else {
    deployment.MarkAsFailed()
    app.Status = "offline"
}
```

## Status Management

### Server States
- **Fresh**: `SetupComplete=false, SecurityLocked=false`
- **Setup**: `SetupComplete=true, SecurityLocked=false`
- **Production**: `SetupComplete=true, SecurityLocked=true`

### App Health States
- **online**: App responding to health checks
- **offline**: App not responding or service down
- **unknown**: Health status not determined

### Deployment States
- **pending**: Queued for execution
- **running**: Currently deploying
- **success**: Completed successfully
- **failed**: Deployment failed

## PocketBase Integration

### Collection Rules
All collections configured with open access (`""` rules) for local tool usage:
```go
collection.ListRule = types.Pointer("")
collection.ViewRule = types.Pointer("")
collection.CreateRule = types.Pointer("")
collection.UpdateRule = types.Pointer("")
collection.DeleteRule = types.Pointer("")
```

### Auto-Date Fields
All models include automatic timestamp management:
```go
created: AutodateField{OnCreate: true}
updated: AutodateField{OnCreate: true, OnUpdate: true}
```

### File Handling
Version model supports file uploads:
```go
deployment_zip: FileField{
    MaxSelect: 1,
    MaxSize:   157286400, // 150MB
    MimeTypes: []string{"application/zip"}
}
```

## Usage Patterns

### Model Creation
```go
// Standard pattern
server := models.NewServer()
server.Name = "production-server"
server.Host = "192.168.1.100"

app := models.NewApp()
app.ServerID = server.ID
app.Name = "my-pocketbase-app"
```

### Status Checks
```go
// Server readiness
if !server.IsReadyForDeployment() {
    return errors.New("server not ready")
}

// Deployment status
if deployment.IsRunning() {
    return errors.New("deployment in progress")
}
```

### Logging Pattern
```go
deployment.AppendLog("Step 1: Uploading files...")
deployment.AppendLog(fmt.Sprintf("Progress: %d%%", progress))
if err != nil {
    deployment.AppendLog(fmt.Sprintf("Error: %v", err))
    deployment.MarkAsFailed()
}
```

## Validation & Constraints

### Server Validation
- Host: Required, non-empty
- Port: 1-65535 range
- Usernames: Required for operation type

### App Validation  
- Name: Required, max 255 chars
- ServerID: Required, valid FK
- Domain: Valid for health checks

### Version Validation
- AppID: Required, valid FK
- DeploymentZip: ZIP format, 150MB max

### Deployment Validation
- AppID/VersionID: Required, valid FKs
- Status: Enum values only
- Logs: 100KB limit

## Best Practices

1. **Use constructors**: Always use `NewX()` for default values
2. **Check readiness**: Validate server state before operations
3. **Track status**: Update deployment status throughout lifecycle
4. **Log operations**: Use `AppendLog()` for deployment tracking
5. **Handle timestamps**: Leverage auto-date fields
6. **Validate FKs**: Ensure referential integrity

## Database Operations

```go
// Via PocketBase app instance
app.FindRecordById("servers", serverID)
app.FindFirstRecordByFilter("apps", "server_id = ?", serverID)

// Direct model methods
server.TableName() // "servers"
app.TableName()    // "apps" 
version.TableName() // "versions"
deployment.TableName() // "deployments"
```

## Error Handling

Models provide validation through PocketBase's built-in mechanisms:
- Field length limits
- Required field enforcement  
- Type validation
- Enum constraint checking

Handle model errors at the service layer, not in models themselves.