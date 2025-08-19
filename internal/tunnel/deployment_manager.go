package tunnel

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pb-deployer/internal/logger"

	"github.com/pocketbase/pocketbase/core"
)

type DeploymentManager struct {
	manager *Manager
	logger  *logger.Logger
	app     core.App
	cleanup []func()
	mu      sync.Mutex
	closed  bool
}

type DeploymentRequest struct {
	AppName          string
	AppID            string
	VersionID        string
	DeploymentID     string
	Domain           string
	ServiceName      string
	RemotePath       string
	ZipDownloadURL   string
	IsInitialDeploy  bool
	SuperuserEmail   string
	SuperuserPass    string
	AppUsername      string
	ProgressCallback func(int, int, string)
	LogCallback      func(string)
}

type DeploymentContext struct {
	Request           *DeploymentRequest
	StagingPath       string
	BackupPath        string
	ServicePath       string
	BinaryPath        string
	WorkingDir        string
	SystemdService    string
	RollbackNeeded    bool
	ServiceWasRunning bool
}

func NewDeploymentManager(manager *Manager, app core.App) *DeploymentManager {
	return &DeploymentManager{
		manager: manager,
		logger:  logger.GetTunnelLogger(),
		app:     app,
	}
}

func (d *DeploymentManager) Deploy(ctx context.Context, req *DeploymentRequest) error {
	d.logger.SystemOperation(fmt.Sprintf("Starting deployment: %s (version: %s)", req.AppName, req.VersionID))

	deployCtx := &DeploymentContext{
		Request:        req,
		StagingPath:    fmt.Sprintf("/tmp/pb-deploy-%s-%d", req.AppName, time.Now().Unix()),
		BackupPath:     fmt.Sprintf("/opt/pocketbase/backups/%s-%d", req.AppName, time.Now().Unix()),
		ServicePath:    fmt.Sprintf("/etc/systemd/system/%s.service", req.ServiceName),
		BinaryPath:     fmt.Sprintf("/opt/pocketbase/apps/%s/%s", req.AppName, req.AppName),
		WorkingDir:     fmt.Sprintf("/opt/pocketbase/apps/%s", req.AppName),
		SystemdService: req.ServiceName,
	}

	defer func() {
		if deployCtx.RollbackNeeded {
			d.logger.Warning("Deployment failed, performing rollback")
			d.rollback(deployCtx)
		}
		// Cleanup staging
		d.manager.client.ExecuteSudo(fmt.Sprintf("rm -rf %s", deployCtx.StagingPath))
	}()

	// Mark deployment as running
	d.updateDeploymentStatus(deployCtx.Request.DeploymentID, "running", "")

	steps := []struct {
		step    int
		total   int
		message string
		fn      func(context.Context, *DeploymentContext) error
	}{
		{1, 11, "Downloading and staging deployment package", d.downloadAndStageVersion},
		{2, 11, "Checking service status", d.checkServiceStatus},
		{3, 11, "Stopping existing service", d.stopService},
		{4, 11, "Creating backup of current deployment", d.backupCurrentDeployment},
		{5, 11, "Preparing deployment directory", d.prepareDeploymentDir},
		{6, 11, "Installing new version", d.swapDeployment},
		{7, 11, "Creating/updating systemd service", d.createSystemdService},
		{8, 11, "Creating superuser (if initial deployment)", d.createSuperuser},
		{9, 11, "Starting service", d.startService},
		{10, 11, "Verifying deployment health", d.verifyDeployment},
		{11, 11, "Finalizing deployment", d.finalizeDeployment},
	}

	for _, step := range steps {
		if req.ProgressCallback != nil {
			req.ProgressCallback(step.step, step.total, step.message)
		}

		d.logProgress(req, step.message)

		if err := step.fn(ctx, deployCtx); err != nil {
			deployCtx.RollbackNeeded = true
			errMsg := fmt.Sprintf("deployment failed at step %d (%s): %v", step.step, step.message, err)
			d.updateDeploymentStatus(deployCtx.Request.DeploymentID, "failed", errMsg)
			return fmt.Errorf("%s", errMsg)
		}
	}

	d.logger.Success("Deployment completed successfully: %s", req.AppName)
	d.updateDeploymentStatus(deployCtx.Request.DeploymentID, "success", "")
	return nil
}

func (d *DeploymentManager) downloadAndStageVersion(ctx context.Context, deployCtx *DeploymentContext) error {
	req := deployCtx.Request

	// Create staging directory
	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("mkdir -p %s", deployCtx.StagingPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}

	// Download the ZIP file locally first
	localZipPath := fmt.Sprintf("/tmp/pb-deploy-%s-%d.zip", req.AppName, time.Now().Unix())
	defer os.Remove(localZipPath)

	d.logProgress(req, "Downloading deployment package...")
	resp, err := http.Get(req.ZipDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download deployment package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download deployment package: HTTP %d", resp.StatusCode)
	}

	localFile, err := os.Create(localZipPath)
	if err != nil {
		return fmt.Errorf("failed to create local zip file: %w", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save deployment package: %w", err)
	}

	// Upload to staging directory
	d.logProgress(req, "Uploading deployment package to server...")
	remoteZipPath := fmt.Sprintf("%s/deployment.zip", deployCtx.StagingPath)
	err = d.manager.client.Upload(localZipPath, remoteZipPath)
	if err != nil {
		return fmt.Errorf("failed to upload deployment package: %w", err)
	}

	// Extract the ZIP file
	d.logProgress(req, "Extracting deployment package...")
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("cd %s && unzip -o deployment.zip", deployCtx.StagingPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to extract deployment package: %s", result.Stderr)
	}

	// Find executable binary (could be named anything)
	d.logProgress(req, "Locating executable binary...")
	result, err = d.manager.client.Execute(fmt.Sprintf("find %s -type f -executable ! -name '*.zip' ! -name '*.txt' ! -name '*.md' ! -name '*.json'", deployCtx.StagingPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to find executable binary in package")
	}

	// Get all executables and find the largest one (likely the main binary)
	executables := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	var pocketbasePath string
	var maxSize int64

	for _, executable := range executables {
		if executable == "" {
			continue
		}
		// Get file size
		sizeResult, err := d.manager.client.Execute(fmt.Sprintf("stat -f%%z %s 2>/dev/null || stat -c%%s %s 2>/dev/null", executable, executable))
		if err == nil && sizeResult.ExitCode == 0 {
			var size int64
			fmt.Sscanf(strings.TrimSpace(sizeResult.Stdout), "%d", &size)
			if size > maxSize {
				maxSize = size
				pocketbasePath = executable
			}
		}
	}

	if pocketbasePath == "" {
		return fmt.Errorf("no suitable executable binary found in deployment package")
	}

	// Rename binary to app name
	newBinaryPath := fmt.Sprintf("%s/%s", deployCtx.StagingPath, req.AppName)
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("mv %s %s && chmod +x %s", pocketbasePath, newBinaryPath, newBinaryPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to rename binary: %s", result.Stderr)
	}

	return nil
}

func (d *DeploymentManager) checkServiceStatus(ctx context.Context, deployCtx *DeploymentContext) error {
	result, err := d.manager.client.Execute(fmt.Sprintf("systemctl is-active %s", deployCtx.SystemdService))
	if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "active" {
		deployCtx.ServiceWasRunning = true
		d.logProgress(deployCtx.Request, fmt.Sprintf("Service %s is currently running", deployCtx.SystemdService))
	} else {
		deployCtx.ServiceWasRunning = false
		d.logProgress(deployCtx.Request, fmt.Sprintf("Service %s is not running", deployCtx.SystemdService))
	}
	return nil
}

func (d *DeploymentManager) stopService(ctx context.Context, deployCtx *DeploymentContext) error {
	if !deployCtx.ServiceWasRunning {
		d.logProgress(deployCtx.Request, "Service not running, skipping stop")
		return nil
	}

	d.logProgress(deployCtx.Request, fmt.Sprintf("Stopping service: %s", deployCtx.SystemdService))
	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("systemctl stop %s", deployCtx.SystemdService))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to stop service: %s", result.Stderr)
	}

	// Wait for service to stop
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		result, err = d.manager.client.Execute(fmt.Sprintf("systemctl is-active %s", deployCtx.SystemdService))
		if err != nil || result.ExitCode != 0 || strings.TrimSpace(result.Stdout) != "active" {
			break
		}
	}

	return nil
}

func (d *DeploymentManager) backupCurrentDeployment(ctx context.Context, deployCtx *DeploymentContext) error {
	// Check if deployment directory exists
	result, err := d.manager.client.Execute(fmt.Sprintf("test -d %s", deployCtx.WorkingDir))
	if err != nil || result.ExitCode != 0 {
		d.logProgress(deployCtx.Request, "No existing deployment to backup")
		return nil
	}

	d.logProgress(deployCtx.Request, "Creating backup of current deployment...")
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("mkdir -p %s && cp -r %s/* %s/",
		filepath.Dir(deployCtx.BackupPath), deployCtx.WorkingDir, deployCtx.BackupPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create backup: %s", result.Stderr)
	}

	d.logProgress(deployCtx.Request, fmt.Sprintf("Backup created at: %s", deployCtx.BackupPath))
	return nil
}

func (d *DeploymentManager) prepareDeploymentDir(ctx context.Context, deployCtx *DeploymentContext) error {
	d.logProgress(deployCtx.Request, "Preparing deployment directory...")

	// Create deployment directory if it doesn't exist
	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("mkdir -p %s", deployCtx.WorkingDir))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create deployment directory: %s", result.Stderr)
	}

	// Set appropriate ownership and permissions
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("chown -R %s:%s %s && chmod 755 %s",
		deployCtx.Request.AppUsername, deployCtx.Request.AppUsername, deployCtx.WorkingDir, deployCtx.WorkingDir))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to set directory permissions: %s", result.Stderr)
	}

	return nil
}

func (d *DeploymentManager) swapDeployment(ctx context.Context, deployCtx *DeploymentContext) error {
	req := deployCtx.Request

	d.logProgress(req, "Installing new version...")

	// Remove old binary if exists
	d.manager.client.ExecuteSudo(fmt.Sprintf("rm -f %s", deployCtx.BinaryPath))

	// Copy new binary from staging
	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("cp %s/%s %s && chmod +x %s",
		deployCtx.StagingPath, req.AppName, deployCtx.BinaryPath, deployCtx.BinaryPath))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to install new binary: %s", result.Stderr)
	}

	// Copy any additional files (excluding the binary and zip)
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("cd %s && find . -type f ! -name '%s' ! -name 'deployment.zip' -exec cp {} %s/ \\;",
		deployCtx.StagingPath, req.AppName, deployCtx.WorkingDir))
	if err != nil {
		d.logger.Warning("Failed to copy additional files: %v", err)
	}

	return nil
}

func (d *DeploymentManager) createSystemdService(ctx context.Context, deployCtx *DeploymentContext) error {
	req := deployCtx.Request

	d.logProgress(req, "Creating/updating systemd service...")

	serviceContent := fmt.Sprintf(`[Unit]
Description=%s PocketBase Server
After=network.target

[Service]
Type=simple
User=%s
Group=%s
LimitNOFILE=4096
Restart=always
RestartSec=5s
StandardOutput=append:/opt/pocketbase/logs/%s.log
StandardError=append:/opt/pocketbase/logs/%s.log
WorkingDirectory=%s
ExecStart=%s serve %s

[Install]
WantedBy=multi-user.target
`, req.AppName, req.AppUsername, req.AppUsername, req.AppName, req.AppName, deployCtx.WorkingDir, deployCtx.BinaryPath, req.Domain)

	// Write service file
	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", deployCtx.ServicePath, serviceContent))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create systemd service: %s", result.Stderr)
	}

	// Reload systemd and enable service
	result, err = d.manager.client.ExecuteSudo("systemctl daemon-reload")
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to reload systemd: %s", result.Stderr)
	}

	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("systemctl enable %s", deployCtx.SystemdService))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to enable service: %s", result.Stderr)
	}

	return nil
}

func (d *DeploymentManager) startService(ctx context.Context, deployCtx *DeploymentContext) error {
	d.logProgress(deployCtx.Request, fmt.Sprintf("Starting service: %s", deployCtx.SystemdService))

	result, err := d.manager.client.ExecuteSudo(fmt.Sprintf("systemctl start %s", deployCtx.SystemdService))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to start service: %s", result.Stderr)
	}

	// Wait for service to start
	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		result, err = d.manager.client.Execute(fmt.Sprintf("systemctl is-active %s", deployCtx.SystemdService))
		if err == nil && result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "active" {
			d.logProgress(deployCtx.Request, "Service started successfully")
			return nil
		}
	}

	return fmt.Errorf("service failed to start within timeout period")
}

func (d *DeploymentManager) createSuperuser(ctx context.Context, deployCtx *DeploymentContext) error {
	req := deployCtx.Request

	if !req.IsInitialDeploy || req.SuperuserEmail == "" || req.SuperuserPass == "" {
		d.logProgress(req, "Skipping superuser creation (not initial deployment or credentials not provided)")
		return nil
	}

	d.logProgress(req, "Creating initial superuser...")

	// Wait a bit for PocketBase to fully initialize
	time.Sleep(5 * time.Second)

	cmd := fmt.Sprintf("cd %s && ./%s superuser create %s %s",
		deployCtx.WorkingDir, req.AppName, req.SuperuserEmail, req.SuperuserPass)

	result, err := d.manager.client.ExecuteSudo(cmd, WithTimeout(30*time.Second))
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create superuser: %s", result.Stderr)
	}

	d.logProgress(req, "Initial superuser created successfully")
	return nil
}

func (d *DeploymentManager) verifyDeployment(ctx context.Context, deployCtx *DeploymentContext) error {
	req := deployCtx.Request

	d.logProgress(req, "Verifying deployment health...")

	healthURL := fmt.Sprintf("https://%s/api/health", req.Domain)

	// Try HTTP first in case HTTPS isn't set up yet
	httpURL := fmt.Sprintf("http://%s/api/health", req.Domain)

	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)

		// Try HTTPS first
		result, err := d.manager.client.Execute(fmt.Sprintf("curl -s -f -m 10 %s", healthURL), WithTimeout(15*time.Second))
		if err == nil && result.ExitCode == 0 {
			d.logProgress(req, "Health check passed (HTTPS)")
			return nil
		}

		// Try HTTP
		result, err = d.manager.client.Execute(fmt.Sprintf("curl -s -f -m 10 %s", httpURL), WithTimeout(15*time.Second))
		if err == nil && result.ExitCode == 0 {
			d.logProgress(req, "Health check passed (HTTP)")
			return nil
		}

		d.logProgress(req, fmt.Sprintf("Health check attempt %d/15 failed, retrying...", i+1))
	}

	return fmt.Errorf("deployment health verification failed after 15 attempts")
}

func (d *DeploymentManager) finalizeDeployment(ctx context.Context, deployCtx *DeploymentContext) error {
	d.logProgress(deployCtx.Request, "Finalizing deployment...")

	// Clean up old backups (keep last 5)
	backupDir := filepath.Dir(deployCtx.BackupPath)
	_, err := d.manager.client.ExecuteSudo(fmt.Sprintf("cd %s && ls -1t | tail -n +6 | xargs -r rm -rf", backupDir))
	if err != nil {
		d.logger.Warning("Failed to clean up old backups: %v", err)
	}

	// Update app status to online and set current version
	d.updateAppStatus(deployCtx.Request.AppID, "online", deployCtx.Request.VersionID)

	d.logProgress(deployCtx.Request, "Deployment finalized successfully")
	return nil
}

func (d *DeploymentManager) rollback(deployCtx *DeploymentContext) error {
	d.logger.SystemOperation(fmt.Sprintf("Rolling back deployment: %s", deployCtx.Request.AppName))

	// Stop the service
	d.manager.client.ExecuteSudo(fmt.Sprintf("systemctl stop %s", deployCtx.SystemdService))

	// Check if backup exists
	result, err := d.manager.client.Execute(fmt.Sprintf("test -d %s", deployCtx.BackupPath))
	if err != nil || result.ExitCode != 0 {
		d.logger.Error("No backup found for rollback")
		return fmt.Errorf("rollback failed: no backup found")
	}

	// Restore from backup
	result, err = d.manager.client.ExecuteSudo(fmt.Sprintf("rm -rf %s/* && cp -r %s/* %s/",
		deployCtx.WorkingDir, deployCtx.BackupPath, deployCtx.WorkingDir))
	if err != nil || result.ExitCode != 0 {
		d.logger.Error("Failed to restore from backup: %s", result.Stderr)
		return fmt.Errorf("rollback failed: %s", result.Stderr)
	}

	// Restart service if it was running
	if deployCtx.ServiceWasRunning {
		d.manager.client.ExecuteSudo(fmt.Sprintf("systemctl start %s", deployCtx.SystemdService))
	}

	// Update app status to offline due to rollback
	d.updateAppStatus(deployCtx.Request.AppID, "offline", "")

	d.logger.Success("Rollback completed")
	return nil
}

func (d *DeploymentManager) logProgress(req *DeploymentRequest, message string) {
	d.logger.SystemOperation(fmt.Sprintf("[%s] %s", req.AppName, message))
	if req.LogCallback != nil {
		req.LogCallback(message)
	}
	// Also append to deployment logs in database
	d.appendDeploymentLog(req.DeploymentID, message)
}

// Close performs cleanup and closes the deployment manager
func (d *DeploymentManager) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	d.logger.SystemOperation("Shutting down deployment manager")

	// Run all cleanup functions in reverse order
	for i := len(d.cleanup) - 1; i >= 0; i-- {
		if d.cleanup[i] != nil {
			d.cleanup[i]()
		}
	}
	d.cleanup = nil

	return nil
}

// AddCleanup adds a cleanup function to be called when the deployment manager is closed
func (d *DeploymentManager) AddCleanup(cleanup func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.closed {
		d.cleanup = append(d.cleanup, cleanup)
	}
}

// IsClosed returns true if the deployment manager has been closed
func (d *DeploymentManager) IsClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func (d *DeploymentManager) updateDeploymentStatus(deploymentID, status, errorMsg string) {
	if d.app == nil {
		return
	}

	record, err := d.app.FindRecordById("deployments", deploymentID)
	if err != nil {
		d.logger.Warning("Failed to find deployment record: %v", err)
		return
	}

	record.Set("status", status)
	if status == "running" {
		record.Set("started_at", time.Now())
	} else if status == "success" || status == "failed" {
		record.Set("completed_at", time.Now())
		if errorMsg != "" {
			d.appendDeploymentLog(deploymentID, errorMsg)
		}
	}

	if err := d.app.Save(record); err != nil {
		d.logger.Warning("Failed to update deployment status: %v", err)
	}
}

func (d *DeploymentManager) appendDeploymentLog(deploymentID, message string) {
	if d.app == nil {
		return
	}

	record, err := d.app.FindRecordById("deployments", deploymentID)
	if err != nil {
		d.logger.Warning("Failed to find deployment record: %v", err)
		return
	}

	currentLogs := record.GetString("logs")
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	newLog := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// Limit log size to prevent bloat (keep last 50KB)
	allLogs := currentLogs + newLog
	if len(allLogs) > 50000 {
		lines := strings.Split(allLogs, "\n")
		// Keep roughly half the lines
		start := len(lines) / 2
		allLogs = strings.Join(lines[start:], "\n")
	}

	record.Set("logs", allLogs)

	if err := d.app.Save(record); err != nil {
		d.logger.Warning("Failed to append deployment log: %v", err)
	}
}

func (d *DeploymentManager) updateAppStatus(appID, status, currentVersion string) {
	if d.app == nil {
		return
	}

	record, err := d.app.FindRecordById("apps", appID)
	if err != nil {
		d.logger.Warning("Failed to find app record: %v", err)
		return
	}

	record.Set("status", status)
	if currentVersion != "" {
		record.Set("current_version", currentVersion)
	}

	if err := d.app.Save(record); err != nil {
		d.logger.Warning("Failed to update app status: %v", err)
	}
}
