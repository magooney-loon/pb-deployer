package managers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pb-deployer/internal/tunnel"
)

// serviceManager implements the ServiceManager interface
type serviceManager struct {
	executor tunnel.Executor
	tracer   tunnel.ServiceTracer
	config   tunnel.ServiceConfig
}

// NewServiceManager creates a new service manager with default configuration
func NewServiceManager(executor tunnel.Executor, tracer tunnel.ServiceTracer) tunnel.ServiceManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &serviceManager{
		executor: executor,
		tracer:   tracer,
		config:   defaultServiceConfig(),
	}
}

// NewServiceManagerWithConfig creates a new service manager with custom configuration
func NewServiceManagerWithConfig(executor tunnel.Executor, tracer tunnel.ServiceTracer, config tunnel.ServiceConfig) tunnel.ServiceManager {
	if executor == nil {
		panic("executor cannot be nil")
	}

	if tracer == nil {
		tracer = &tunnel.NoOpServiceTracer{}
	}

	return &serviceManager{
		executor: executor,
		tracer:   tracer,
		config:   config,
	}
}

// ManageService performs service management operations
func (sm *serviceManager) ManageService(ctx context.Context, action tunnel.ServiceAction, service string) error {
	span := sm.tracer.TraceServiceOperation(ctx, "manage_service", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
		"action":  sm.actionToString(action),
	})

	// Validate inputs
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, sm.actionToString(action), err)
	}

	actionStr := sm.actionToString(action)
	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "manage_service",
		Status:      "running",
		Message:     fmt.Sprintf("Performing %s action on service %s", actionStr, service),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	var err error
	switch action {
	case tunnel.ServiceStart:
		err = sm.startService(ctx, service)
	case tunnel.ServiceStop:
		err = sm.stopService(ctx, service)
	case tunnel.ServiceRestart:
		err = sm.restartService(ctx, service)
	case tunnel.ServiceReload:
		err = sm.reloadService(ctx, service)
	default:
		err = fmt.Errorf("unsupported service action: %v", action)
	}

	if err != nil {
		span.EndWithError(err)
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "manage_service",
			Status:      "failed",
			Message:     fmt.Sprintf("Failed to %s service %s: %s", actionStr, service, err.Error()),
			ProgressPct: 100,
			Timestamp:   time.Now(),
		})
		return tunnel.WrapServiceError(service, actionStr, err)
	}

	span.Event("service_action_completed", map[string]any{
		"service": service,
		"action":  actionStr,
		"success": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "manage_service",
		Status:      "success",
		Message:     fmt.Sprintf("Successfully performed %s action on service %s", actionStr, service),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// GetServiceStatus returns service status
func (sm *serviceManager) GetServiceStatus(ctx context.Context, service string) (*tunnel.ServiceStatus, error) {
	span := sm.tracer.TraceServiceOperation(ctx, "get_service_status", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
	})

	// Validate service name
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return nil, tunnel.WrapServiceError(service, "get_status", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "get_service_status",
		Status:      "running",
		Message:     fmt.Sprintf("Getting status for service %s", service),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	// Get service status
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl show %s --property=ActiveState,LoadState,SubState,UnitFileState,Description", shellEscape(service)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return nil, tunnel.WrapServiceError(service, "get_status", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to get service status: %s", result.Output)
		span.EndWithError(err)
		return nil, tunnel.WrapServiceError(service, "get_status", err)
	}

	status, err := sm.parseServiceStatus(service, result.Output)
	if err != nil {
		span.EndWithError(err)
		return nil, tunnel.WrapServiceError(service, "parse_status", err)
	}

	span.Event("service_status_retrieved", map[string]any{
		"service": service,
		"active":  status.Active,
		"enabled": status.Enabled,
		"state":   status.State,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "get_service_status",
		Status:      "success",
		Message:     fmt.Sprintf("Retrieved status for service %s: %s", service, status.State),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return status, nil
}

// GetServiceLogs retrieves service logs
func (sm *serviceManager) GetServiceLogs(ctx context.Context, service string, lines int) (string, error) {
	span := sm.tracer.TraceServiceOperation(ctx, "get_service_logs", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
		"lines":   lines,
	})

	// Validate inputs
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return "", tunnel.WrapServiceError(service, "get_logs", err)
	}

	if lines <= 0 {
		lines = sm.config.DefaultLogLines
	}
	if lines > sm.config.MaxLogLines {
		lines = sm.config.MaxLogLines
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "get_service_logs",
		Status:      "running",
		Message:     fmt.Sprintf("Retrieving %d lines of logs for service %s", lines, service),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	// Get service logs using journalctl
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("journalctl -u %s -n %d --no-pager", shellEscape(service), lines),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return "", tunnel.WrapServiceError(service, "get_logs", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to get service logs: %s", result.Output)
		span.EndWithError(err)
		return "", tunnel.WrapServiceError(service, "get_logs", err)
	}

	span.Event("service_logs_retrieved", map[string]any{
		"service":    service,
		"lines":      lines,
		"log_length": len(result.Output),
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "get_service_logs",
		Status:      "success",
		Message:     fmt.Sprintf("Retrieved %d lines of logs for service %s", lines, service),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return result.Output, nil
}

// EnableService enables a service to start on boot
func (sm *serviceManager) EnableService(ctx context.Context, service string) error {
	span := sm.tracer.TraceServiceOperation(ctx, "enable_service", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
	})

	// Validate service name
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "enable", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "enable_service",
		Status:      "running",
		Message:     fmt.Sprintf("Enabling service %s", service),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	// Enable the service
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl enable %s", shellEscape(service)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "enable", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to enable service: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "enable", err)
	}

	span.Event("service_enabled", map[string]any{
		"service": service,
		"success": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "enable_service",
		Status:      "success",
		Message:     fmt.Sprintf("Service %s enabled successfully", service),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// DisableService disables a service from starting on boot
func (sm *serviceManager) DisableService(ctx context.Context, service string) error {
	span := sm.tracer.TraceServiceOperation(ctx, "disable_service", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
	})

	// Validate service name
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "disable", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "disable_service",
		Status:      "running",
		Message:     fmt.Sprintf("Disabling service %s", service),
		ProgressPct: 30,
		Timestamp:   time.Now(),
	})

	// Disable the service
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl disable %s", shellEscape(service)),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "disable", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to disable service: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "disable", err)
	}

	span.Event("service_disabled", map[string]any{
		"service": service,
		"success": true,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "disable_service",
		Status:      "success",
		Message:     fmt.Sprintf("Service %s disabled successfully", service),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// CreateServiceFile creates a systemd service file
func (sm *serviceManager) CreateServiceFile(ctx context.Context, service tunnel.ServiceDefinition) error {
	span := sm.tracer.TraceServiceOperation(ctx, "create_service_file", service.Name)
	defer span.End()

	span.SetFields(map[string]any{
		"service_name": service.Name,
		"enabled":      service.Enabled,
		"user":         service.User,
		"working_dir":  service.WorkingDirectory,
	})

	// Validate service definition
	if err := sm.validateServiceDefinition(service); err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service.Name, "create_file", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_service_file",
		Status:      "running",
		Message:     fmt.Sprintf("Creating service file for %s", service.Name),
		ProgressPct: 20,
		Timestamp:   time.Now(),
	})

	// Generate service file content
	serviceContent := sm.generateServiceFileContent(service)
	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", service.Name)

	// Create the service file
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("cat > %s << 'EOF'\n%sEOF", shellEscape(servicePath), serviceContent),
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service.Name, "create_file", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to create service file: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapServiceError(service.Name, "create_file", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_service_file",
		Status:      "running",
		Message:     "Reloading systemd daemon",
		ProgressPct: 60,
		Timestamp:   time.Now(),
	})

	// Reload systemd daemon
	cmd = tunnel.Command{
		Cmd:     "systemctl daemon-reload",
		Sudo:    true,
		Timeout: 30 * time.Second,
	}

	result, err = sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service.Name, "daemon_reload", err)
	}

	if result.ExitCode != 0 {
		err := fmt.Errorf("failed to reload systemd daemon: %s", result.Output)
		span.EndWithError(err)
		return tunnel.WrapServiceError(service.Name, "daemon_reload", err)
	}

	// Enable service if requested
	if service.Enabled {
		sm.reportProgress(ctx, tunnel.ProgressUpdate{
			Step:        "create_service_file",
			Status:      "running",
			Message:     fmt.Sprintf("Enabling service %s", service.Name),
			ProgressPct: 80,
			Timestamp:   time.Now(),
		})

		if err := sm.EnableService(ctx, service.Name); err != nil {
			span.EndWithError(err)
			return err
		}
	}

	span.Event("service_file_created", map[string]any{
		"service_name": service.Name,
		"service_path": servicePath,
		"enabled":      service.Enabled,
	})

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "create_service_file",
		Status:      "success",
		Message:     fmt.Sprintf("Service file for %s created successfully", service.Name),
		ProgressPct: 100,
		Timestamp:   time.Now(),
	})

	return nil
}

// WaitForService waits for a service to reach the desired state
func (sm *serviceManager) WaitForService(ctx context.Context, service string, timeout time.Duration) error {
	span := sm.tracer.TraceServiceOperation(ctx, "wait_for_service", service)
	defer span.End()

	span.SetFields(map[string]any{
		"service": service,
		"timeout": timeout.Seconds(),
	})

	// Validate service name
	if err := sm.validateServiceName(service); err != nil {
		span.EndWithError(err)
		return tunnel.WrapServiceError(service, "wait", err)
	}

	sm.reportProgress(ctx, tunnel.ProgressUpdate{
		Step:        "wait_for_service",
		Status:      "running",
		Message:     fmt.Sprintf("Waiting for service %s to be ready", service),
		ProgressPct: 10,
		Timestamp:   time.Now(),
	})

	// Create a context with timeout
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(sm.config.StatusCheckInterval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-waitCtx.Done():
			err := fmt.Errorf("timeout waiting for service %s after %v", service, timeout)
			span.EndWithError(err)
			return tunnel.WrapServiceError(service, "wait_timeout", err)

		case <-ticker.C:
			status, err := sm.GetServiceStatus(ctx, service)
			if err != nil {
				span.EndWithError(err)
				return tunnel.WrapServiceError(service, "wait_status_check", err)
			}

			elapsed := time.Since(startTime)
			progress := int((elapsed.Seconds()/timeout.Seconds())*80) + 10

			if status.Active && status.State == "running" {
				span.Event("service_ready", map[string]any{
					"service": service,
					"elapsed": elapsed.Seconds(),
				})

				sm.reportProgress(ctx, tunnel.ProgressUpdate{
					Step:        "wait_for_service",
					Status:      "success",
					Message:     fmt.Sprintf("Service %s is ready", service),
					ProgressPct: 100,
					Timestamp:   time.Now(),
				})

				return nil
			}

			sm.reportProgress(ctx, tunnel.ProgressUpdate{
				Step:        "wait_for_service",
				Status:      "running",
				Message:     fmt.Sprintf("Waiting for service %s (current state: %s)", service, status.State),
				ProgressPct: progress,
				Timestamp:   time.Now(),
			})
		}
	}
}

// Helper methods

func (sm *serviceManager) startService(ctx context.Context, service string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl start %s", shellEscape(service)),
		Sudo:    true,
		Timeout: sm.config.ActionTimeout,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start service: %s", result.Output)
	}

	return nil
}

func (sm *serviceManager) stopService(ctx context.Context, service string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl stop %s", shellEscape(service)),
		Sudo:    true,
		Timeout: sm.config.ActionTimeout,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to stop service: %s", result.Output)
	}

	return nil
}

func (sm *serviceManager) restartService(ctx context.Context, service string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl restart %s", shellEscape(service)),
		Sudo:    true,
		Timeout: sm.config.ActionTimeout,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to restart service: %s", result.Output)
	}

	return nil
}

func (sm *serviceManager) reloadService(ctx context.Context, service string) error {
	cmd := tunnel.Command{
		Cmd:     fmt.Sprintf("systemctl reload %s", shellEscape(service)),
		Sudo:    true,
		Timeout: sm.config.ActionTimeout,
	}

	result, err := sm.executor.RunCommand(ctx, cmd)
	if err != nil {
		return err
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to reload service: %s", result.Output)
	}

	return nil
}

func (sm *serviceManager) validateServiceName(service string) error {
	if service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for invalid characters
	validName := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validName.MatchString(service) {
		return fmt.Errorf("invalid service name: %s", service)
	}

	return nil
}

func (sm *serviceManager) validateServiceDefinition(service tunnel.ServiceDefinition) error {
	if err := sm.validateServiceName(service.Name); err != nil {
		return err
	}

	if service.ExecStart == "" {
		return fmt.Errorf("ExecStart command cannot be empty")
	}

	return nil
}

func (sm *serviceManager) parseServiceStatus(serviceName, output string) (*tunnel.ServiceStatus, error) {
	status := &tunnel.ServiceStatus{
		Name:  serviceName,
		Since: time.Now(),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		switch key {
		case "ActiveState":
			status.State = value
			status.Active = (value == "active")
		case "UnitFileState":
			status.Enabled = (value == "enabled")
		case "Description":
			status.Description = value
		}
	}

	return status, nil
}

func (sm *serviceManager) generateServiceFileContent(service tunnel.ServiceDefinition) string {
	var lines []string

	// Unit section
	lines = append(lines, "[Unit]")
	if service.Description != "" {
		lines = append(lines, fmt.Sprintf("Description=%s", service.Description))
	} else {
		lines = append(lines, fmt.Sprintf("Description=%s service", service.Name))
	}

	if len(service.After) > 0 {
		lines = append(lines, fmt.Sprintf("After=%s", strings.Join(service.After, " ")))
	} else {
		lines = append(lines, "After=network.target")
	}

	if len(service.Requires) > 0 {
		lines = append(lines, fmt.Sprintf("Requires=%s", strings.Join(service.Requires, " ")))
	}

	lines = append(lines, "")

	// Service section
	lines = append(lines, "[Service]")
	lines = append(lines, fmt.Sprintf("Type=%s", service.Type))
	lines = append(lines, fmt.Sprintf("ExecStart=%s", service.ExecStart))

	if service.ExecStop != "" {
		lines = append(lines, fmt.Sprintf("ExecStop=%s", service.ExecStop))
	}

	if service.ExecReload != "" {
		lines = append(lines, fmt.Sprintf("ExecReload=%s", service.ExecReload))
	}

	if service.User != "" {
		lines = append(lines, fmt.Sprintf("User=%s", service.User))
	}

	if service.Group != "" {
		lines = append(lines, fmt.Sprintf("Group=%s", service.Group))
	}

	if service.WorkingDirectory != "" {
		lines = append(lines, fmt.Sprintf("WorkingDirectory=%s", service.WorkingDirectory))
	}

	if len(service.Environment) > 0 {
		for key, value := range service.Environment {
			lines = append(lines, fmt.Sprintf("Environment=%s=%s", key, value))
		}
	}

	if service.Restart != "" {
		lines = append(lines, fmt.Sprintf("Restart=%s", service.Restart))
	} else {
		lines = append(lines, "Restart=on-failure")
	}

	if service.RestartSec > 0 {
		lines = append(lines, fmt.Sprintf("RestartSec=%d", int(service.RestartSec.Seconds())))
	}

	lines = append(lines, "")

	// Install section
	if service.Enabled {
		lines = append(lines, "[Install]")
		if service.WantedBy != "" {
			lines = append(lines, fmt.Sprintf("WantedBy=%s", service.WantedBy))
		} else {
			lines = append(lines, "WantedBy=multi-user.target")
		}
	}

	return strings.Join(lines, "\n") + "\n"
}

func (sm *serviceManager) actionToString(action tunnel.ServiceAction) string {
	switch action {
	case tunnel.ServiceStart:
		return "start"
	case tunnel.ServiceStop:
		return "stop"
	case tunnel.ServiceRestart:
		return "restart"
	case tunnel.ServiceReload:
		return "reload"
	case tunnel.ServiceGetStatus:
		return "status"
	default:
		return "unknown"
	}
}

func (sm *serviceManager) reportProgress(ctx context.Context, update tunnel.ProgressUpdate) {
	if reporter, ok := tunnel.GetProgressReporter(ctx); ok {
		reporter.Report(update)
	}
}

// Configuration methods

// SetConfig updates the service manager configuration
func (sm *serviceManager) SetConfig(config tunnel.ServiceConfig) {
	sm.config = config
}

// GetConfig returns the current service manager configuration
func (sm *serviceManager) GetConfig() tunnel.ServiceConfig {
	return sm.config
}

// defaultServiceConfig returns default service configuration
func defaultServiceConfig() tunnel.ServiceConfig {
	return tunnel.ServiceConfig{
		ActionTimeout:       60 * time.Second,
		StatusCheckInterval: 2 * time.Second,
		DefaultLogLines:     50,
		MaxLogLines:         1000,
	}
}
