package managers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pb-deployer/internal/tunnel"
)

// deploymentManager implements the DeploymentManager interface
type deploymentManager struct {
	executor         tunnel.Executor
	serviceManager   tunnel.ServiceManager
	tracer           tunnel.ServiceTracer
	config           tunnel.DeploymentConfig
	deploymentStates map[string]*tunnel.DeploymentStatus
}

// NewDeploymentManager creates a new deployment manager with default configuration
func NewDeploymentManager(executor tunnel.Executor, serviceManager tunnel.ServiceManager, tracer tunnel.ServiceTracer) tunnel.DeploymentManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if serviceManager == nil {
		panic("serviceManager cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &deploymentManager{
		executor:         executor,
		serviceManager:   serviceManager,
		tracer:           tracer,
		config:           defaultDeploymentConfig(),
		deploymentStates: make(map[string]*tunnel.DeploymentStatus),
	}
}

// NewDeploymentManagerWithConfig creates a new deployment manager with custom configuration
func NewDeploymentManagerWithConfig(executor tunnel.Executor, serviceManager tunnel.ServiceManager, tracer tunnel.ServiceTracer, config tunnel.DeploymentConfig) tunnel.DeploymentManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if serviceManager == nil {
		panic("serviceManager cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &deploymentManager{
		executor:         executor,
		serviceManager:   serviceManager,
		tracer:           tracer,
		config:           config,
		deploymentStates: make(map[string]*tunnel.DeploymentStatus),
	}
}

// Deploy performs application deployment with the given specification
func (dm *deploymentManager) Deploy(ctx context.Context, deployment tunnel.DeploymentSpec) (*tunnel.DeploymentResult, error) {
	span := dm.tracer.TraceDeployment(ctx, deployment.Name, deployment.Version)
	defer span.End()

	span.SetFields(map[string]any{
		"deployment_name": deployment.Name,
		"version":         deployment.Version,
		"environment":     deployment.Environment,
		"strategy":        deployment.Strategy,
		"service_name":    deployment.ServiceName,
	})

	result := &tunnel.DeploymentResult{
		Version:   deployment.Version,
		StartTime: time.Now(),
		Steps:     make([]tunnel.DeploymentStep, 0),
	}

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "deploy",
		Status:      "running",
		Message:     fmt.Sprintf("Starting deployment of %s version %s", deployment.Name, deployment.Version),
		ProgressPct: 5,
		Timestamp:   time.Now(),
	})

	// Step 1: Validate deployment specification
	step := dm.createStep("validate_deployment", "Validating deployment specification")
	result.Steps = append(result.Steps, step)

	if err := dm.ValidateDeployment(ctx, deployment); err != nil {
		step.Status = tunnel.StepStatusFailed
		step.Error = err
		step.EndTime = time.Now()
		result.Success = false
		result.Message = fmt.Sprintf("Deployment validation failed: %s", err.Error())
		span.EndWithError(err)
		return result, tunnel.WrapDeploymentError(deployment.Name, "validate", err)
	}

	step.Status = tunnel.StepStatusCompleted
	step.EndTime = time.Now()

	// Step 2: Get current version for rollback
	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "deploy",
		Status:      "running",
		Message:     "Getting current deployment version",
		ProgressPct: 15,
		Timestamp:   time.Now(),
	})

	currentVersion, err := dm.getCurrentVersion(ctx, deployment.Name)
	if err == nil {
		result.PreviousVersion = currentVersion
	}

	// Step 3: Create backup if enabled
	if dm.config.BackupEnabled {
		step = dm.createStep("create_backup", "Creating backup")
		result.Steps = append(result.Steps, step)

		dm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "deploy",
			Status:      "running",
			Message:     "Creating deployment backup",
			ProgressPct: 25,
			Timestamp:   time.Now(),
		})

		if err := dm.createBackup(ctx, deployment); err != nil {
			step.Status = tunnel.StepStatusFailed
			step.Error = err
			step.EndTime = time.Now()
			if !deployment.RollbackOnFailure {
				result.Success = false
				result.Message = fmt.Sprintf("Backup creation failed: %s", err.Error())
				span.EndWithError(err)
				return result, tunnel.WrapDeploymentError(deployment.Name, "backup", err)
			}
		} else {
			step.Status = tunnel.StepStatusCompleted
			step.EndTime = time.Now()
		}
	}

	// Step 4: Execute pre-deploy hooks
	if len(deployment.PreDeployHooks) > 0 {
		step = dm.createStep("pre_deploy_hooks", "Executing pre-deploy hooks")
		result.Steps = append(result.Steps, step)

		dm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "deploy",
			Status:      "running",
			Message:     "Executing pre-deploy hooks",
			ProgressPct: 35,
			Timestamp:   time.Now(),
		})

		if err := dm.executeHooks(ctx, deployment.PreDeployHooks); err != nil {
			step.Status = tunnel.StepStatusFailed
			step.Error = err
			step.EndTime = time.Now()
			if deployment.RollbackOnFailure {
				dm.performRollback(ctx, deployment, result)
			}
			result.Success = false
			result.Message = fmt.Sprintf("Pre-deploy hooks failed: %s", err.Error())
			span.EndWithError(err)
			return result, tunnel.WrapDeploymentError(deployment.Name, "pre_deploy_hooks", err)
		}

		step.Status = tunnel.StepStatusCompleted
		step.EndTime = time.Now()
	}

	// Step 5: Deploy application based on strategy
	step = dm.createStep("deploy_application", "Deploying application")
	result.Steps = append(result.Steps, step)

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "deploy",
		Status:      "running",
		Message:     fmt.Sprintf("Deploying application using %s strategy", deployment.Strategy),
		ProgressPct: 50,
		Timestamp:   time.Now(),
	})

	if err := dm.deployApplication(ctx, deployment); err != nil {
		step.Status = tunnel.StepStatusFailed
		step.Error = err
		step.EndTime = time.Now()
		if deployment.RollbackOnFailure {
			dm.performRollback(ctx, deployment, result)
		}
		result.Success = false
		result.Message = fmt.Sprintf("Application deployment failed: %s", err.Error())
		span.EndWithError(err)
		return result, tunnel.WrapDeploymentError(deployment.Name, "deploy_application", err)
	}

	step.Status = tunnel.StepStatusCompleted
	step.EndTime = time.Now()

	// Step 6: Update service configuration
	if deployment.ServiceName != "" {
		step = dm.createStep("update_service", "Updating service configuration")
		result.Steps = append(result.Steps, step)

		dm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "deploy",
			Status:      "running",
			Message:     "Updating service configuration",
			ProgressPct: 65,
			Timestamp:   time.Now(),
		})

		if err := dm.updateService(ctx, deployment); err != nil {
			step.Status = tunnel.StepStatusFailed
			step.Error = err
			step.EndTime = time.Now()
			if deployment.RollbackOnFailure {
				dm.performRollback(ctx, deployment, result)
			}
			result.Success = false
			result.Message = fmt.Sprintf("Service update failed: %s", err.Error())
			span.EndWithError(err)
			return result, tunnel.WrapDeploymentError(deployment.Name, "update_service", err)
		}

		step.Status = tunnel.StepStatusCompleted
		step.EndTime = time.Now()
	}

	// Step 7: Health check
	step = dm.createStep("health_check", "Performing health check")
	result.Steps = append(result.Steps, step)

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "deploy",
		Status:      "running",
		Message:     "Performing deployment health check",
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	if err := dm.performHealthCheck(ctx, deployment); err != nil {
		step.Status = tunnel.StepStatusFailed
		step.Error = err
		step.EndTime = time.Now()
		if deployment.RollbackOnFailure {
			dm.performRollback(ctx, deployment, result)
		}
		result.Success = false
		result.Message = fmt.Sprintf("Health check failed: %s", err.Error())
		span.EndWithError(err)
		return result, tunnel.WrapDeploymentError(deployment.Name, "health_check", err)
	}

	step.Status = tunnel.StepStatusCompleted
	step.EndTime = time.Now()

	// Step 8: Execute post-deploy hooks
	if len(deployment.PostDeployHooks) > 0 {
		step = dm.createStep("post_deploy_hooks", "Executing post-deploy hooks")
		result.Steps = append(result.Steps, step)

		dm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "deploy",
			Status:      "running",
			Message:     "Executing post-deploy hooks",
			ProgressPct: 90,
			Timestamp:   time.Now(),
		})

		if err := dm.executeHooks(ctx, deployment.PostDeployHooks); err != nil {
			step.Status = tunnel.StepStatusFailed
			step.Error = err
			step.EndTime = time.Now()
			// Post-deploy hook failures are typically non-fatal
			step.Message = fmt.Sprintf("Warning: Post-deploy hooks failed: %s", err.Error())
		} else {
			step.Status = tunnel.StepStatusCompleted
			step.EndTime = time.Now()
		}
	}

	// Update deployment state
	dm.updateDeploymentState(deployment.Name, tunnel.DeploymentStateHealthy, deployment.Version, deployment.Environment)

	result.Success = true
	result.Message = "Deployment completed successfully"
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	span.Event("deployment_completed", map[string]any{
		"deployment_name": deployment.Name,
		"version":         deployment.Version,
		"duration":        result.Duration.Seconds(),
		"success":         true,
	})

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "deploy",
		Status:      "success",
		Message:     fmt.Sprintf("Deployment of %s version %s completed successfully", deployment.Name, deployment.Version),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return result, nil
}

// Rollback rolls back a deployment to a previous version
func (dm *deploymentManager) Rollback(ctx context.Context, deployment string, version string) error {
	span := dm.tracer.TraceDeployment(ctx, deployment, version)
	defer span.End()

	span.SetFields(map[string]any{
		"deployment_name": deployment,
		"target_version":  version,
		"operation":       "rollback",
	})

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "rollback",
		Status:      "running",
		Message:     fmt.Sprintf("Rolling back %s to version %s", deployment, version),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Update deployment state to rolling back
	dm.updateDeploymentState(deployment, tunnel.DeploymentStateRollingBack, version, "")

	// Check if backup exists for the target version
	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "rollback",
		Status:      "running",
		Message:     "Checking backup availability",
		ProgressPct: 20,
		Timestamp:   time.Now(),
	})

	backupPath := filepath.Join(dm.config.BackupPath, deployment, version)
	if err := dm.validateBackupExists(ctx, backupPath); err != nil {
		span.EndWithError(err)
		return tunnel.WrapDeploymentError(deployment, "rollback_validation", err)
	}

	// Restore from backup
	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "rollback",
		Status:      "running",
		Message:     "Restoring from backup",
		ProgressPct: 50,
		Timestamp:   time.Now(),
	})

	if err := dm.restoreFromBackup(ctx, deployment, version); err != nil {
		dm.updateDeploymentState(deployment, tunnel.DeploymentStateFailed, version, "")
		span.EndWithError(err)
		return tunnel.WrapDeploymentError(deployment, "restore_backup", err)
	}

	// Restart services if needed
	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "rollback",
		Status:      "running",
		Message:     "Restarting services",
		ProgressPct: 80,
		Timestamp:   time.Now(),
	})

	if err := dm.restartServices(ctx, deployment); err != nil {
		span.EndWithError(err)
		return tunnel.WrapDeploymentError(deployment, "restart_services", err)
	}

	// Update deployment state to healthy
	dm.updateDeploymentState(deployment, tunnel.DeploymentStateHealthy, version, "")

	span.Event("rollback_completed", map[string]any{
		"deployment_name": deployment,
		"version":         version,
		"success":         true,
	})

	dm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "rollback",
		Status:      "success",
		Message:     fmt.Sprintf("Rollback of %s to version %s completed successfully", deployment, version),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// ValidateDeployment validates a deployment specification
func (dm *deploymentManager) ValidateDeployment(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	span := dm.tracer.TraceDeployment(ctx, deployment.Name, deployment.Version)
	defer span.End()

	span.SetFields(map[string]any{
		"deployment_name": deployment.Name,
		"operation":       "validate",
	})

	// Basic validation
	if deployment.Name == "" {
		err := fmt.Errorf("deployment name cannot be empty")
		span.EndWithError(err)
		return err
	}

	if deployment.Version == "" {
		err := fmt.Errorf("deployment version cannot be empty")
		span.EndWithError(err)
		return err
	}

	if deployment.ArtifactPath == "" {
		err := fmt.Errorf("artifact path cannot be empty")
		span.EndWithError(err)
		return err
	}

	// Validate artifact exists
	if dm.config.ArtifactValidation {
		if err := dm.validateArtifact(ctx, deployment.ArtifactPath); err != nil {
			span.EndWithError(err)
			return tunnel.WrapDeploymentError(deployment.Name, "artifact_validation", err)
		}
	}

	// Validate strategy
	if err := dm.validateStrategy(deployment.Strategy); err != nil {
		span.EndWithError(err)
		return tunnel.WrapDeploymentError(deployment.Name, "strategy_validation", err)
	}

	// Validate dependencies
	if err := dm.validateDependencies(ctx, deployment.Dependencies); err != nil {
		span.EndWithError(err)
		return tunnel.WrapDeploymentError(deployment.Name, "dependency_validation", err)
	}

	span.Event("validation_completed", map[string]any{
		"deployment_name": deployment.Name,
		"valid":           true,
	})

	return nil
}

// GetDeploymentStatus returns the current status of a deployment
func (dm *deploymentManager) GetDeploymentStatus(ctx context.Context, deployment string) (*tunnel.DeploymentStatus, error) {
	span := dm.tracer.TraceDeployment(ctx, deployment, "")
	defer span.End()

	span.SetFields(map[string]any{
		"deployment_name": deployment,
		"operation":       "get_status",
	})

	// Check if we have cached status
	if status, exists := dm.deploymentStates[deployment]; exists {
		// Update health status
		health, err := dm.checkDeploymentHealth(ctx, deployment)
		if err == nil {
			status.Health = health.Overall
		}
		status.LastUpdated = time.Now()
		return status, nil
	}

	// Create default status if not found
	status := &tunnel.DeploymentStatus{
		Name:        deployment,
		State:       tunnel.DeploymentStateUnknown,
		Health:      tunnel.HealthStatusUnknown,
		LastUpdated: time.Now(),
		Replicas: tunnel.DeploymentReplicas{
			Desired:   1,
			Available: 0,
			Ready:     0,
			Updated:   0,
		},
		Configuration: make(map[string]string),
		Events:        make([]tunnel.DeploymentEvent, 0),
	}

	dm.deploymentStates[deployment] = status

	span.Event("status_retrieved", map[string]any{
		"deployment_name": deployment,
		"state":           status.State,
		"health":          status.Health,
	})

	return status, nil
}

// ListDeployments returns a list of all deployments
func (dm *deploymentManager) ListDeployments(ctx context.Context) ([]tunnel.DeploymentInfo, error) {
	span := dm.tracer.TraceDeployment(ctx, "all", "")
	defer span.End()

	span.SetFields(map[string]any{
		"operation": "list_deployments",
	})

	deployments := make([]tunnel.DeploymentInfo, 0, len(dm.deploymentStates))

	for name, status := range dm.deploymentStates {
		info := tunnel.DeploymentInfo{
			Name:        name,
			Version:     status.Version,
			Environment: status.Environment,
			State:       status.State,
			Health:      status.Health,
			UpdatedAt:   status.LastUpdated,
		}
		deployments = append(deployments, info)
	}

	span.Event("deployments_listed", map[string]any{
		"count": len(deployments),
	})

	return deployments, nil
}

// HealthCheck performs health check on a deployment
func (dm *deploymentManager) HealthCheck(ctx context.Context, deployment string) (*tunnel.DeploymentHealth, error) {
	span := dm.tracer.TraceDeployment(ctx, deployment, "")
	defer span.End()

	span.SetFields(map[string]any{
		"deployment_name": deployment,
		"operation":       "health_check",
	})

	return dm.checkDeploymentHealth(ctx, deployment)
}

// Helper methods

func (dm *deploymentManager) createStep(name, message string) tunnel.DeploymentStep {
	return tunnel.DeploymentStep{
		Name:      name,
		Status:    tunnel.StepStatusRunning,
		Message:   message,
		StartTime: time.Now(),
	}
}

func (dm *deploymentManager) getCurrentVersion(ctx context.Context, deployment string) (string, error) {
	// Try to get version from deployment state
	if status, exists := dm.deploymentStates[deployment]; exists {
		return status.Version, nil
	}

	// Try to get version from version file
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat /opt/%s/VERSION 2>/dev/null || echo 'unknown'", shellEscape(deployment)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Output), nil
}

func (dm *deploymentManager) createBackup(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	backupDir := filepath.Join(dm.config.BackupPath, deployment.Name, deployment.Version)

	// Create backup directory
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("mkdir -p %s", shellEscape(backupDir)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to create backup directory: %s", result.Output)
	}

	// Backup current deployment if it exists
	if deployment.WorkingDirectory != "" {
		cmd = tunnel.Command{
			Cmd:     fmt.Sprintf("cp -r %s/* %s/ 2>/dev/null || true", shellEscape(deployment.WorkingDirectory), shellEscape(backupDir)),
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}

		result, err = dm.executor.RunCommand(ctx, cmd)
		if err != nil {
			return err
		}
	}

	// Save current version info
	versionFile := filepath.Join(backupDir, "VERSION")
	currentVersion, _ := dm.getCurrentVersion(ctx, deployment.Name)

	cmd = tunnel.Command{
		Cmd:     fmt.Sprintf("echo %s > %s", shellEscape(currentVersion), shellEscape(versionFile)),
		Sudo:    true,
		Timeout: 10 * time.Second,
	}

	dm.executor.RunCommand(ctx, cmd) // Ignore errors for version file

	return nil
}

func (dm *deploymentManager) executeHooks(ctx context.Context, hooks []string) error {
	for i, hook := range hooks {
		cmd := tunnel.Command{
			Cmd:     hook,
			Sudo:    true,
			Timeout: 5 * time.Minute,
		}

		result, err := dm.executor.RunCommand(ctx, cmd)
		if err != nil {
			return fmt.Errorf("hook %d failed: %w", i+1, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("hook %d failed with exit code %d: %s", i+1, result.ExitCode, result.Output)
		}
	}

	return nil
}

func (dm *deploymentManager) deployApplication(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	switch deployment.Strategy {
	case tunnel.DeploymentStrategyRolling:
		return dm.deployRolling(ctx, deployment)
	case tunnel.DeploymentStrategyBlueGreen:
		return dm.deployBlueGreen(ctx, deployment)
	case tunnel.DeploymentStrategyCanary:
		return dm.deployCanary(ctx, deployment)
	case tunnel.DeploymentStrategyRecreate:
		return dm.deployRecreate(ctx, deployment)
	default:
		return dm.deployRecreate(ctx, deployment) // Default to recreate
	}
}

func (dm *deploymentManager) deployRecreate(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// Stop service if it exists
	if deployment.ServiceName != "" {
		dm.serviceManager.ManageService(ctx, tunnel.ServiceStop, deployment.ServiceName)
	}

	// Create working directory if it doesn't exist
	if deployment.WorkingDirectory != "" {
		cmd := tunnel.Command{
			Cmd:     fmt.Sprintf("mkdir -p %s", shellEscape(deployment.WorkingDirectory)),
			Sudo:    true,
			Timeout: 30 * time.Second,
		}

		result, err := dm.executor.RunCommand(ctx, cmd)
		if err != nil {
			return err
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to create working directory: %s", result.Output)
		}
	}

	// Copy artifact to deployment location
	if err := dm.deployArtifact(ctx, deployment); err != nil {
		return err
	}

	// Start service if it exists
	if deployment.ServiceName != "" {
		return dm.serviceManager.ManageService(ctx, tunnel.ServiceStart, deployment.ServiceName)
	}

	return nil
}

func (dm *deploymentManager) deployRolling(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// For rolling deployment, we update gradually
	// This is a simplified implementation
	return dm.deployRecreate(ctx, deployment)
}

func (dm *deploymentManager) deployBlueGreen(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// For blue-green deployment, we would deploy to a parallel environment
	// This is a simplified implementation
	return dm.deployRecreate(ctx, deployment)
}

func (dm *deploymentManager) deployCanary(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// For canary deployment, we would deploy to a subset of instances
	// This is a simplified implementation
	return dm.deployRecreate(ctx, deployment)
}

func (dm *deploymentManager) deployArtifact(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// Extract or copy artifact based on type
	artifactExt := filepath.Ext(deployment.ArtifactPath)

	switch artifactExt {
	case ".tar.gz", ".tgz":
		return dm.extractTarGz(ctx, deployment.ArtifactPath, deployment.WorkingDirectory)
	case ".zip":
		return dm.extractZip(ctx, deployment.ArtifactPath, deployment.WorkingDirectory)
	default:
		return dm.copyFile(ctx, deployment.ArtifactPath, deployment.WorkingDirectory)
	}
}

func (dm *deploymentManager) extractTarGz(ctx context.Context, artifactPath, destPath string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("tar -xzf %s -C %s", shellEscape(artifactPath), shellEscape(destPath)),
		Sudo:    true,
		Timeout: 10 * time.Minute,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to extract tar.gz: %s", result.Output)
	}

	return nil
}

func (dm *deploymentManager) extractZip(ctx context.Context, artifactPath, destPath string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("unzip -o %s -d %s", shellEscape(artifactPath), shellEscape(destPath)),
		Sudo:    true,
		Timeout: 10 * time.Minute,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to extract zip: %s", result.Output)
	}

	return nil
}

func (dm *deploymentManager) copyFile(ctx context.Context, artifactPath, destPath string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cp %s %s/", shellEscape(artifactPath), shellEscape(destPath)),
		Sudo:    true,
		Timeout: 5 * time.Minute,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to copy file: %s", result.Output)
	}

	return nil
}

func (dm *deploymentManager) updateService(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// Restart the service to pick up new deployment
	return dm.serviceManager.ManageService(ctx, tunnel.ServiceRestart, deployment.ServiceName)
}

func (dm *deploymentManager) performHealthCheck(ctx context.Context, deployment tunnel.DeploymentSpec) error {
	// Wait for the deployment to be ready
	maxWait := dm.config.HealthCheckTimeout
	if maxWait == 0 {
		maxWait = 2 * time.Minute
	}

	timeout := time.After(maxWait)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("health check timeout after %v", maxWait)
		case <-ticker.C:
			if deployment.HealthCheckURL != "" {
				if err := dm.checkHTTPHealth(ctx, deployment.HealthCheckURL); err == nil {
					return nil
				}
			} else if deployment.ServiceName != "" {
				status, err := dm.serviceManager.GetServiceStatus(ctx, deployment.ServiceName)
				if err == nil && status.Active && status.State == "running" {
					return nil
				}
			} else {
				// Simple file existence check
				if deployment.WorkingDirectory != "" {
					cmd := tunnel.Command{
						Cmd:     fmt.Sprintf("test -d %s", shellEscape(deployment.WorkingDirectory)),
						Sudo:    false,
						Timeout: 10 * time.Second,
					}

					result, err := dm.executor.RunCommand(ctx, cmd)
					if err == nil && result.ExitCode == 0 {
						return nil
					}
				}
			}
		}
	}
}

func (dm *deploymentManager) checkHTTPHealth(ctx context.Context, url string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("curl -f -s %s > /dev/null", shellEscape(url)),
		Sudo:    false,
		Timeout: 30 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("health check failed: HTTP %s returned non-200", url)
	}

	return nil
}

func (dm *deploymentManager) performRollback(ctx context.Context, deployment tunnel.DeploymentSpec, result *tunnel.DeploymentResult) {
	if result.PreviousVersion != "" {
		step := dm.createStep("automatic_rollback", "Performing automatic rollback")
		result.Steps = append(result.Steps, step)

		if err := dm.Rollback(ctx, deployment.Name, result.PreviousVersion); err != nil {
			step.Status = tunnel.StepStatusFailed
			step.Error = err
			step.Message = fmt.Sprintf("Automatic rollback failed: %s", err.Error())
		} else {
			step.Status = tunnel.StepStatusCompleted
			step.Message = "Automatic rollback completed successfully"
		}
		step.EndTime = time.Now()
	}
}

func (dm *deploymentManager) updateDeploymentState(name string, state tunnel.DeploymentState, version, environment string) {
	if status, exists := dm.deploymentStates[name]; exists {
		status.State = state
		status.Version = version
		if environment != "" {
			status.Environment = environment
		}
		status.LastUpdated = time.Now()
	} else {
		dm.deploymentStates[name] = &tunnel.DeploymentStatus{
			Name:        name,
			State:       state,
			Version:     version,
			Environment: environment,
			LastUpdated: time.Now(),
			Replicas: tunnel.DeploymentReplicas{
				Desired:   1,
				Available: 1,
				Ready:     1,
				Updated:   1,
			},
			Configuration: make(map[string]string),
			Events:        make([]tunnel.DeploymentEvent, 0),
		}
	}
}

func (dm *deploymentManager) validateArtifact(ctx context.Context, artifactPath string) error {
	// Check if artifact file exists
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("test -f %s", shellEscape(artifactPath)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("artifact file does not exist: %s", artifactPath)
	}

	// Check if artifact is readable
	cmd = tunnel.Command{
		Cmd:     fmt.Sprintf("test -r %s", shellEscape(artifactPath)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err = dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("artifact file is not readable: %s", artifactPath)
	}

	return nil
}

func (dm *deploymentManager) validateStrategy(strategy tunnel.DeploymentStrategy) error {
	switch strategy {
	case tunnel.DeploymentStrategyRolling, tunnel.DeploymentStrategyBlueGreen,
		tunnel.DeploymentStrategyCanary, tunnel.DeploymentStrategyRecreate:
		return nil
	case "":
		return nil // Empty strategy defaults to recreate
	default:
		return fmt.Errorf("unsupported deployment strategy: %s", strategy)
	}
}

func (dm *deploymentManager) validateDependencies(ctx context.Context, dependencies []string) error {
	for _, dep := range dependencies {
		status, err := dm.GetDeploymentStatus(ctx, dep)
		if err != nil {
			return fmt.Errorf("dependency %s not found: %w", dep, err)
		}

		if status.State != tunnel.DeploymentStateHealthy {
			return fmt.Errorf("dependency %s is not healthy (state: %s)", dep, status.State)
		}
	}

	return nil
}

func (dm *deploymentManager) validateBackupExists(ctx context.Context, backupPath string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("test -d %s", shellEscape(backupPath)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("backup directory does not exist: %s", backupPath)
	}

	return nil
}

func (dm *deploymentManager) restoreFromBackup(ctx context.Context, deployment, version string) error {
	backupPath := filepath.Join(dm.config.BackupPath, deployment, version)
	deploymentPath := fmt.Sprintf("/opt/%s", deployment)

	// Create deployment directory
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("mkdir -p %s", shellEscape(deploymentPath)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to create deployment directory: %s", result.Output)
	}

	// Restore files from backup
	cmd = tunnel.Command{
		Cmd:     fmt.Sprintf("cp -r %s/* %s/", shellEscape(backupPath), shellEscape(deploymentPath)),
		Sudo:    true,
		Timeout: 5 * time.Minute,
	}

	result, err = dm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to restore from backup: %s", result.Output)
	}

	return nil
}

func (dm *deploymentManager) restartServices(ctx context.Context, deployment string) error {
	// Try to restart service based on deployment name
	serviceName := deployment
	if err := dm.serviceManager.ManageService(ctx, tunnel.ServiceRestart, serviceName); err != nil {
		// If service doesn't exist, it's not an error for rollback
		return nil
	}

	return nil
}

func (dm *deploymentManager) checkDeploymentHealth(ctx context.Context, deployment string) (*tunnel.DeploymentHealth, error) {
	health := &tunnel.DeploymentHealth{
		Overall:    tunnel.HealthStatusHealthy,
		Components: make(map[string]tunnel.ComponentHealth),
		LastCheck:  time.Now(),
		Message:    "All components healthy",
	}

	// Check service health if deployment has a service
	serviceName := deployment
	if status, err := dm.serviceManager.GetServiceStatus(ctx, serviceName); err == nil {
		componentHealth := tunnel.ComponentHealth{
			Name:    "service",
			Status:  tunnel.HealthStatusHealthy,
			Message: "Service is active and running",
			Checks: []tunnel.HealthCheck{
				{
					Name:     "service_status",
					Status:   tunnel.HealthStatusHealthy,
					Message:  fmt.Sprintf("Service %s is %s", serviceName, status.State),
					Duration: 0,
					LastRun:  time.Now(),
				},
			},
		}

		if !status.Active || status.State != "running" {
			componentHealth.Status = tunnel.HealthStatusUnhealthy
			componentHealth.Message = fmt.Sprintf("Service is %s", status.State)
			componentHealth.Checks[0].Status = tunnel.HealthStatusUnhealthy
			health.Overall = tunnel.HealthStatusUnhealthy
			health.Message = "Service is not healthy"
		}

		health.Components["service"] = componentHealth
	}

	// Check deployment directory
	deploymentPath := fmt.Sprintf("/opt/%s", deployment)
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("test -d %s", shellEscape(deploymentPath)),
		Sudo:    false,
		Timeout: 10 * time.Second,
	}

	result, err := dm.executor.RunCommand(ctx, cmd)

	componentHealth := tunnel.ComponentHealth{
		Name:    "deployment_files",
		Status:  tunnel.HealthStatusHealthy,
		Message: "Deployment files are present",
		Checks: []tunnel.HealthCheck{
			{
				Name:     "directory_exists",
				Status:   tunnel.HealthStatusHealthy,
				Message:  "Deployment directory exists",
				Duration: 0,
				LastRun:  time.Now(),
			},
		},
	}

	if err != nil || result.ExitCode != 0 {
		componentHealth.Status = tunnel.HealthStatusUnhealthy
		componentHealth.Message = "Deployment files are missing"
		componentHealth.Checks[0].Status = tunnel.HealthStatusUnhealthy
		componentHealth.Checks[0].Message = "Deployment directory does not exist"
		health.Overall = tunnel.HealthStatusUnhealthy
		if health.Message == "All components healthy" {
			health.Message = "Deployment files are missing"
		}
	}

	health.Components["deployment_files"] = componentHealth

	return health, nil
}

func (dm *deploymentManager) reportProgress(ctx context.Context, update tunnel.ProgressUpdate) {
	if reporter, ok := tunnel.GetProgressReporter(ctx); ok {
		reporter.Report(update)
	}
}

// Configuration methods

// SetConfig updates the deployment manager configuration
func (dm *deploymentManager) SetConfig(config tunnel.DeploymentConfig) {
	dm.config = config
}

// GetConfig returns the current deployment manager configuration
func (dm *deploymentManager) GetConfig() tunnel.DeploymentConfig {
	return dm.config
}

// defaultDeploymentConfig returns default deployment configuration
func defaultDeploymentConfig() tunnel.DeploymentConfig {
	return tunnel.DeploymentConfig{
		Strategy:           tunnel.DeploymentStrategyRecreate,
		MaxRetries:         3,
		RetryDelay:         30 * time.Second,
		HealthCheckTimeout: 2 * time.Minute,
		BackupEnabled:      true,
		BackupPath:         "/opt/pb-deployer/backups",
		ValidationEnabled:  true,
		ArtifactValidation: true,
	}
}
