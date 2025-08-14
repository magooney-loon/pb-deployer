# Setup Handler Alignment

HTTP handlers for setup management aligned with tunnel package interfaces.

## Handler Functions

### CreateUser
```http
POST /api/v1/setup/users
```
**Calls:**
- `setupMgr.CreateUser(ctx, userConfig)`
- `executor.RunCommand(ctx, verifyUserCmd)` (for verification)

### SetupSSHKeys
```http
POST /api/v1/setup/users/{username}/ssh-keys
```
**Calls:**
- `setupMgr.SetupSSHKeys(ctx, username, sshKeys)`
- `executor.RunCommand(ctx, verifySSHKeysCmd)` (for verification)

### CreateDirectories
```http
POST /api/v1/setup/directories
```
**Calls:**
- `setupMgr.CreateDirectory(ctx, dirPath, owner)` (per directory)
- `executor.RunCommand(ctx, verifyDirCmd)` (for verification)

### ConfigureSudoAccess
```http
POST /api/v1/setup/users/{username}/sudo
```
**Calls:**
- `executor.RunCommand(ctx, createSudoFileCmd)`
- `executor.RunCommand(ctx, validateSudoFileCmd)`

### InstallPackages
```http
POST /api/v1/setup/packages
```
**Calls:**
- `executor.RunCommand(ctx, updatePackageCacheCmd)` (if update_cache)
- `executor.RunCommand(ctx, installPackagesCmd)`
- `executor.RunCommand(ctx, verifyPackagesCmd)`

### SetupSystemUser
```http
POST /api/v1/setup/system-users
```
**Calls:**
- `setupMgr.CreateUser(ctx, userConfig)`
- `setupMgr.SetupSSHKeys(ctx, username, sshKeys)` (if setup_ssh)
- `setupMgr.CreateDirectory(ctx, dirPath, owner)` (per directory)
- `executor.RunCommand(ctx, createSudoFileCmd)` (if setup_sudo)

### GetSetupConfiguration
```http
GET /api/v1/setup/config
```
**Calls:**
- Database queries only (no tunnel calls for handler config)

### UpdateSetupConfiguration
```http
PUT /api/v1/setup/config
```
**Calls:**
- Database update operations only

### GetUserInformation
```http
GET /api/v1/setup/users/{username}
```
**Calls:**
- `executor.RunCommand(ctx, getUserInfoCmd)`
- `executor.RunCommand(ctx, getSSHKeysCmd)`
- `executor.RunCommand(ctx, getSudoConfigCmd)`

### ListUsers
```http
GET /api/v1/setup/users
```
**Calls:**
- `executor.RunCommand(ctx, listUsersCmd)`
- `executor.RunCommand(ctx, getUserDetailsCmd)` (per user if detailed)

### CheckPackageManager
```http
GET /api/v1/setup/package-manager
```
**Calls:**
- `executor.RunCommand(ctx, detectPackageManagerCmd)`
- `executor.RunCommand(ctx, checkLastUpdateCmd)`

## Progress Tracking Pattern

All setup operations use handler-level progress tracking:

```go
func (h *SetupHandler) setupSystemUserWithProgress(ctx context.Context, config SystemUserConfig) error {
    progressChan := make(chan SetupProgress, 10)
    go h.monitorSetupProgress(ctx, config.Username, progressChan)
    
    // Step 1: Create user
    progressChan <- SetupProgress{Step: "create_user", Status: "running"}
    err := h.setupMgr.CreateUser(ctx, config.UserConfig)
    if err != nil {
        progressChan <- SetupProgress{Step: "create_user", Status: "failed", Error: err}
        return err
    }
    progressChan <- SetupProgress{Step: "create_user", Status: "completed"}
    
    // Step 2: Setup SSH keys (if requested)
    if config.SetupSSH {
        progressChan <- SetupProgress{Step: "ssh_keys", Status: "running"}
        err = h.setupMgr.SetupSSHKeys(ctx, config.Username, config.SSHKeys)
        if err != nil {
            progressChan <- SetupProgress{Step: "ssh_keys", Status: "failed", Error: err}
            return err
        }
        progressChan <- SetupProgress{Step: "ssh_keys", Status: "completed"}
    }
    
    // Step 3: Create directories
    progressChan <- SetupProgress{Step: "directories", Status: "running"}
    for _, dir := range config.Directories {
        err = h.setupMgr.CreateDirectory(ctx, dir.Path, dir.Owner)
        if err != nil {
            progressChan <- SetupProgress{Step: "directories", Status: "failed", Error: err}
            return err
        }
    }
    progressChan <- SetupProgress{Step: "directories", Status: "completed"}
    
    // Step 4: Configure sudo (if requested)
    if config.SetupSudo {
        progressChan <- SetupProgress{Step: "sudo_config", Status: "running"}
        err = h.configureSudo(ctx, config.Username, config.SudoCommands)
        if err != nil {
            progressChan <- SetupProgress{Step: "sudo_config", Status: "failed", Error: err}
            return err
        }
        progressChan <- SetupProgress{Step: "sudo_config", Status: "completed"}
    }
    
    close(progressChan)
    return nil
}
```

## Constructor Pattern

```go
func NewSetupHandler(
    executor tunnel.Executor,
    setupMgr tunnel.SetupManager,
) *SetupHandler {
    return &SetupHandler{
        executor:  executor,
        setupMgr:  setupMgr,
    }
}
```

## Error Handling Pattern

```go
result, err := h.setupMgr.CreateUser(ctx, userConfig)
if err != nil {
    if tunnel.IsRetryable(err) {
        return h.retryUserCreation(ctx, userConfig)
    }
    if tunnel.IsAuthError(err) {
        return handleAuthError(e, err)
    }
    if tunnel.IsConnectionError(err) {
        return handleConnectionError(e, err)
    }
    return handleGenericError(e, err)
}
```

## Package Management Pattern

```go
// Detect package manager
detectCmd := tunnel.Command{
    Cmd:     "command -v apt && echo 'apt' || command -v yum && echo 'yum' || command -v dnf && echo 'dnf'",
    Timeout: 10 * time.Second,
}
result, err := h.executor.RunCommand(ctx, detectCmd)

// Install packages based on detected package manager
var installCmd tunnel.Command
switch packageManager {
case "apt":
    installCmd = tunnel.Command{
        Cmd:     fmt.Sprintf("apt-get install -y %s", strings.Join(packages, " ")),
        Sudo:    true,
        Timeout: 300 * time.Second,
    }
case "yum":
    installCmd = tunnel.Command{
        Cmd:     fmt.Sprintf("yum install -y %s", strings.Join(packages, " ")),
        Sudo:    true,
        Timeout: 300 * time.Second,
    }
}
err = h.executor.RunCommand(ctx, installCmd)
```

## Sudo Configuration Pattern

```go
// Create sudo configuration file
sudoContent := generateSudoConfig(username, commands)
createSudoCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("cat > /etc/sudoers.d/%s", username),
    Input:   sudoContent,
    Sudo:    true,
    Timeout: 10 * time.Second,
}
err := h.executor.RunCommand(ctx, createSudoCmd)

// Validate sudo configuration
validateCmd := tunnel.Command{
    Cmd:     "visudo -c",
    Sudo:    true,
    Timeout: 10 * time.Second,
}
err = h.executor.RunCommand(ctx, validateCmd)
```

## User Information Pattern

```go
// Get user information
userInfoCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("getent passwd %s", username),
    Timeout: 10 * time.Second,
}
result, err := h.executor.RunCommand(ctx, userInfoCmd)

// Check SSH keys
sshKeysCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("test -f /home/%s/.ssh/authorized_keys && wc -l /home/%s/.ssh/authorized_keys", username, username),
    Timeout: 10 * time.Second,
}
sshResult, err := h.executor.RunCommand(ctx, sshKeysCmd)

// Check sudo configuration
sudoCheckCmd := tunnel.Command{
    Cmd:     fmt.Sprintf("test -f /etc/sudoers.d/%s && echo 'configured'", username),
    Timeout: 10 * time.Second,
}
sudoResult, err := h.executor.RunCommand(ctx, sudoCheckCmd)
```

## Key Alignments

- ✅ Uses individual setup manager methods: `CreateUser()`, `SetupSSHKeys()`, `CreateDirectory()`
- ✅ No non-existent methods like `SetupServerWithProgress()` or `SetupSystemUserWithProgress()`
- ✅ Progress tracking implemented at handler level
- ✅ Package management via `Executor.RunCommand()` not PackageManager interface
- ✅ Sudo configuration via `Executor.RunCommand()` not separate manager
- ✅ User information gathering via shell commands
- ✅ Error handling uses tunnel package utilities
- ✅ File operations use `Executor.RunCommand()` not FileManager interface
- ✅ No complex workflow methods - handlers orchestrate individual manager calls