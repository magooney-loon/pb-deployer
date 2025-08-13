# Models Package

PocketBase data models for deployment management with SSH targets.

## Core Models

```go
// SSH deployment target
type Server struct {
    ID             string
    Name           string
    Host           string
    Port           int
    RootUsername   string
    AppUsername    string
    UseSSHAgent    bool
    ManualKeyPath  string
    SetupComplete  bool
    SecurityLocked bool
    Created        time.Time
    Updated        time.Time
}

// PocketBase application instance
type App struct {
    ID             string
    Name           string
    ServerID       string
    RemotePath     string
    ServiceName    string
    Domain         string
    CurrentVersion string
    Status         string // "online"/"offline"/"unknown"
    Created        time.Time
    Updated        time.Time
}

// Versioned deployment package
type Version struct {
    ID            string
    AppID         string
    VersionNum    string
    DeploymentZip string
    Notes         string
    Created       time.Time
    Updated       time.Time
}

// Deployment operation tracking
type Deployment struct {
    ID          string
    AppID       string
    VersionID   string
    Status      string // "pending"/"running"/"success"/"failed"
    Logs        string
    StartedAt   *time.Time
    CompletedAt *time.Time
    Created     time.Time
    Updated     time.Time
}
```

## Key Methods

```go
// Server
server := models.NewServer()
server.GetSSHAddress()              // "host:port"
server.IsReadyForDeployment()       // setup && security complete
server.IsSetupComplete()            // setup status
server.IsSecurityLocked()           // security status

// App
app := models.NewApp()
app.GetHealthURL()                  // "https://domain/api/health"
app.IsOnline()                      // status == "online"

// Version
version := models.NewVersion()
version.HasDeploymentZip()          // deployment_zip exists
version.GetVersionString()          // formatted version

// Deployment
deployment := models.NewDeployment()
deployment.MarkAsRunning()          // status -> "running"
deployment.MarkAsSuccess()          // status -> "success"
deployment.MarkAsFailed()           // status -> "failed"
deployment.IsComplete()             // success || failed
deployment.GetDuration()            // completion time
deployment.AppendLog("message")     // add to logs
```

## Usage

```go
// Database setup
err := models.InitializeDatabase(app)

// Server lifecycle
server := models.NewServer()
server.Name = "prod-server"
server.Host = "192.168.1.100"

// Check readiness
if !server.IsReadyForDeployment() {
    return errors.New("server not ready")
}

// Deploy application
deployment := models.NewDeployment()
deployment.AppID = app.ID
deployment.VersionID = version.ID
deployment.MarkAsRunning()

// Track progress
deployment.AppendLog("Uploading files...")
deployment.AppendLog("Starting service...")

// Complete deployment
if success {
    deployment.MarkAsSuccess()
    app.CurrentVersion = version.VersionNum
    app.Status = "online"
} else {
    deployment.MarkAsFailed()
    app.Status = "offline"
}
```

## Relationships

```
Server (1) ──── (N) App (1) ──── (N) Version
                 │                      │
                 └──── (N) Deployment ──┘
```

## Features

- **PocketBase Integration**: Auto-creates collections with proper schema
- **Lifecycle Management**: Built-in status tracking and transitions
- **Foreign Key Constraints**: Referential integrity between models
- **Auto Timestamps**: Created/Updated fields managed automatically
- **File Uploads**: Version model supports ZIP deployment packages
- **Validation**: Field length limits and type constraints
- **Progress Logging**: Deployment log tracking with size limits