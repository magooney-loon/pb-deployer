package logger

import (
	"fmt"
	"strings"
	"time"

	"pb-deployer/internal/models"
)

// SSHLogger provides SSH-specific logging utilities with structured context
type SSHLogger struct {
	*Logger
	server   *models.Server
	username string
	isRoot   bool
}

// SSHOperationLogger tracks SSH operations with timing and context
type SSHOperationLogger struct {
	logger    *SSHLogger
	operation string
	startTime time.Time
	fields    map[string]any
}

// SSHProgressLogger handles progress tracking for long-running SSH operations
type SSHProgressLogger struct {
	logger      *SSHLogger
	operation   string
	totalSteps  int
	currentStep int
	startTime   time.Time
}

// NewSSHLogger creates a logger specialized for SSH operations
func NewSSHLogger(server *models.Server, asRoot bool) *SSHLogger {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	baseLogger := New()
	baseLogger.SetPrefix("SSH")

	return &SSHLogger{
		Logger:   baseLogger,
		server:   server,
		username: username,
		isRoot:   asRoot,
	}
}

// WithServer creates an SSH logger for a specific server
func WithServer(server *models.Server, asRoot bool) *SSHLogger {
	return NewSSHLogger(server, asRoot)
}

// Connection logging methods

// ConnectStart logs the beginning of an SSH connection attempt
func (s *SSHLogger) ConnectStart() {
	s.WithFields(map[string]any{
		"host":     s.server.Host,
		"port":     s.server.Port,
		"username": s.username,
		"is_root":  s.isRoot,
		"server":   s.server.Name,
	}).Info("Attempting SSH connection")
}

// ConnectSuccess logs successful SSH connection
func (s *SSHLogger) ConnectSuccess(responseTime time.Duration, authMethod string) {
	s.WithFields(map[string]any{
		"host":          s.server.Host,
		"port":          s.server.Port,
		"username":      s.username,
		"is_root":       s.isRoot,
		"server":        s.server.Name,
		"response_time": responseTime.String(),
		"auth_method":   authMethod,
	}).Info("SSH connection established successfully")
}

// ConnectFailed logs failed SSH connection with detailed error context
func (s *SSHLogger) ConnectFailed(err error, attemptNum int, maxAttempts int) {
	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"port":         s.server.Port,
		"username":     s.username,
		"is_root":      s.isRoot,
		"server":       s.server.Name,
		"attempt":      attemptNum,
		"max_attempts": maxAttempts,
	}).WithError(err).Error("SSH connection failed")
}

// ConnectRetry logs connection retry attempts
func (s *SSHLogger) ConnectRetry(attemptNum int, maxAttempts int, delay time.Duration) {
	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"attempt":      attemptNum,
		"max_attempts": maxAttempts,
		"retry_delay":  delay.String(),
	}).Warn("Retrying SSH connection after failure")
}

// Command execution logging

// CommandStart logs the start of command execution
func (s *SSHLogger) CommandStart(command string) {
	displayCmd := command
	if len(displayCmd) > 100 {
		displayCmd = displayCmd[:97] + "..."
	}

	s.WithFields(map[string]any{
		"host":     s.server.Host,
		"username": s.username,
		"command":  displayCmd,
		"is_root":  s.isRoot,
	}).Debug("Executing SSH command")
}

// CommandSuccess logs successful command execution
func (s *SSHLogger) CommandSuccess(command string, duration time.Duration, outputLines int) {
	displayCmd := command
	if len(displayCmd) > 100 {
		displayCmd = displayCmd[:97] + "..."
	}

	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"command":      displayCmd,
		"duration":     duration.String(),
		"output_lines": outputLines,
	}).Debug("SSH command executed successfully")
}

// CommandFailed logs failed command execution
func (s *SSHLogger) CommandFailed(command string, err error, output string) {
	displayCmd := command
	if len(displayCmd) > 100 {
		displayCmd = displayCmd[:97] + "..."
	}

	displayOutput := output
	if len(displayOutput) > 200 {
		displayOutput = displayOutput[:197] + "..."
	}

	s.WithFields(map[string]any{
		"host":     s.server.Host,
		"username": s.username,
		"command":  displayCmd,
		"output":   displayOutput,
	}).WithError(err).Error("SSH command execution failed")
}

// Service management logging

// ServiceAction logs systemd service operations
func (s *SSHLogger) ServiceAction(action string, serviceName string, success bool, duration time.Duration) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	entry := s.WithFields(map[string]any{
		"host":            s.server.Host,
		"username":        s.username,
		"service":         serviceName,
		"action":          action,
		"duration":        duration.String(),
		"success":         success,
		"uses_sudo":       !s.isRoot,
		"security_locked": s.server.SecurityLocked,
	})

	message := fmt.Sprintf("Service %s %s", action, serviceName)
	if success {
		message += " completed successfully"
	} else {
		message += " failed"
	}

	entry.log(level, "%s", message)
}

// Health monitoring logging

// HealthCheck logs connection health check results
func (s *SSHLogger) HealthCheck(healthy bool, responseTime time.Duration, consecutiveFailures int) {
	status := "healthy"
	level := DebugLevel
	if !healthy {
		status = "unhealthy"
		level = WarnLevel
		if consecutiveFailures >= 3 {
			level = ErrorLevel
		}
	}

	s.WithFields(map[string]any{
		"host":                 s.server.Host,
		"username":             s.username,
		"status":               status,
		"response_time":        responseTime.String(),
		"consecutive_failures": consecutiveFailures,
	}).log(level, "SSH connection health check completed")
}

// PoolConnection logs connection pool operations
func (s *SSHLogger) PoolConnection(action string, poolSize int, healthy int, unhealthy int) {
	s.WithFields(map[string]any{
		"host":                  s.server.Host,
		"username":              s.username,
		"action":                action,
		"pool_size":             poolSize,
		"healthy_connections":   healthy,
		"unhealthy_connections": unhealthy,
	}).Debug("Connection pool operation")
}

// Security operations logging

// SecurityStep logs individual steps in security lockdown process
func (s *SSHLogger) SecurityStep(step string, status string, message string, progressPct int) {
	level := InfoLevel
	if status == "failed" {
		level = ErrorLevel
	} else if status == "warning" {
		level = WarnLevel
	}

	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"step":         step,
		"status":       status,
		"progress_pct": progressPct,
		"operation":    "security_lockdown",
	}).log(level, "%s", message)
}

// SetupStep logs individual steps in server setup process
func (s *SSHLogger) SetupStep(step string, status string, message string, progressPct int) {
	level := InfoLevel
	if status == "failed" {
		level = ErrorLevel
	} else if status == "warning" {
		level = WarnLevel
	}

	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"step":         step,
		"status":       status,
		"progress_pct": progressPct,
		"operation":    "server_setup",
	}).log(level, "%s", message)
}

// HostKeyAccepted logs when a host key is accepted and stored
func (s *SSHLogger) HostKeyAccepted(hostname string, keyType string, fingerprint string) {
	s.WithFields(map[string]any{
		"hostname":    hostname,
		"key_type":    keyType,
		"fingerprint": fingerprint,
		"action":      "host_key_accepted",
	}).Warn("Host key automatically accepted and stored")
}

// AuthMethodAttempt logs authentication method attempts
func (s *SSHLogger) AuthMethodAttempt(method string, success bool, details string) {
	level := DebugLevel
	if !success {
		level = WarnLevel
	}

	s.WithFields(map[string]any{
		"host":        s.server.Host,
		"username":    s.username,
		"auth_method": method,
		"success":     success,
		"details":     details,
	}).log(level, "SSH authentication method attempt")
}

// Operation tracking methods

// StartOperation begins tracking an SSH operation
func (s *SSHLogger) StartOperation(operation string) *SSHOperationLogger {
	s.WithFields(map[string]any{
		"host":      s.server.Host,
		"username":  s.username,
		"operation": operation,
	}).Info("Starting SSH operation")

	return &SSHOperationLogger{
		logger:    s,
		operation: operation,
		startTime: time.Now(),
		fields:    make(map[string]any),
	}
}

// SSHOperationLogger methods

// AddField adds a field to the operation context
func (o *SSHOperationLogger) AddField(key string, value any) *SSHOperationLogger {
	o.fields[key] = value
	return o
}

// AddFields adds multiple fields to the operation context
func (o *SSHOperationLogger) AddFields(fields map[string]any) *SSHOperationLogger {
	for k, v := range fields {
		o.fields[k] = v
	}
	return o
}

// Progress logs progress during the operation
func (o *SSHOperationLogger) Progress(message string, progressPct int) {
	allFields := map[string]any{
		"host":         o.logger.server.Host,
		"username":     o.logger.username,
		"operation":    o.operation,
		"progress_pct": progressPct,
		"elapsed":      time.Since(o.startTime).String(),
	}

	// Add operation-specific fields
	for k, v := range o.fields {
		allFields[k] = v
	}

	o.logger.WithFields(allFields).Info("%s", message)
}

// Success completes the operation successfully
func (o *SSHOperationLogger) Success(message string) {
	duration := time.Since(o.startTime)

	allFields := map[string]any{
		"host":      o.logger.server.Host,
		"username":  o.logger.username,
		"operation": o.operation,
		"duration":  duration.String(),
		"success":   true,
	}

	// Add operation-specific fields
	for k, v := range o.fields {
		allFields[k] = v
	}

	o.logger.WithFields(allFields).Info("%s", message)
}

// Failure completes the operation with failure
func (o *SSHOperationLogger) Failure(err error, message string) {
	duration := time.Since(o.startTime)

	allFields := map[string]any{
		"host":      o.logger.server.Host,
		"username":  o.logger.username,
		"operation": o.operation,
		"duration":  duration.String(),
		"success":   false,
	}

	// Add operation-specific fields
	for k, v := range o.fields {
		allFields[k] = v
	}

	o.logger.WithFields(allFields).WithError(err).Error("%s", message)
}

// Progress tracking methods

// StartProgress begins tracking progress for a multi-step operation
func (s *SSHLogger) StartProgress(operation string, totalSteps int) *SSHProgressLogger {
	s.WithFields(map[string]any{
		"host":        s.server.Host,
		"username":    s.username,
		"operation":   operation,
		"total_steps": totalSteps,
	}).Info("Starting multi-step SSH operation")

	return &SSHProgressLogger{
		logger:      s,
		operation:   operation,
		totalSteps:  totalSteps,
		currentStep: 0,
		startTime:   time.Now(),
	}
}

// SSHProgressLogger methods

// NextStep advances to the next step and logs progress
func (p *SSHProgressLogger) NextStep(stepName string, message string) {
	p.currentStep++
	progressPct := (p.currentStep * 100) / p.totalSteps

	p.logger.WithFields(map[string]any{
		"host":         p.logger.server.Host,
		"username":     p.logger.username,
		"operation":    p.operation,
		"step":         stepName,
		"step_number":  p.currentStep,
		"total_steps":  p.totalSteps,
		"progress_pct": progressPct,
		"elapsed":      time.Since(p.startTime).String(),
	}).Info("%s", message)
}

// StepProgress updates progress within the current step
func (p *SSHProgressLogger) StepProgress(message string, subProgressPct int) {
	// Calculate overall progress including sub-progress
	baseProgress := ((p.currentStep - 1) * 100) / p.totalSteps
	stepProgress := subProgressPct / p.totalSteps
	totalProgress := baseProgress + stepProgress

	p.logger.WithFields(map[string]any{
		"host":           p.logger.server.Host,
		"username":       p.logger.username,
		"operation":      p.operation,
		"step_number":    p.currentStep,
		"total_steps":    p.totalSteps,
		"step_progress":  subProgressPct,
		"total_progress": totalProgress,
		"elapsed":        time.Since(p.startTime).String(),
	}).Debug("%s", message)
}

// StepFailed logs a step failure
func (p *SSHProgressLogger) StepFailed(stepName string, err error, message string) {
	p.logger.WithFields(map[string]any{
		"host":        p.logger.server.Host,
		"username":    p.logger.username,
		"operation":   p.operation,
		"step":        stepName,
		"step_number": p.currentStep,
		"total_steps": p.totalSteps,
		"elapsed":     time.Since(p.startTime).String(),
	}).WithError(err).Error("%s", message)
}

// Complete logs the completion of the multi-step operation
func (p *SSHProgressLogger) Complete(success bool, message string) {
	duration := time.Since(p.startTime)

	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	p.logger.WithFields(map[string]any{
		"host":            p.logger.server.Host,
		"username":        p.logger.username,
		"operation":       p.operation,
		"total_steps":     p.totalSteps,
		"completed_steps": p.currentStep,
		"duration":        duration.String(),
		"success":         success,
	}).log(level, "%s", message)
}

// Specialized logging methods for common SSH operations

// FileTransfer logs file transfer operations
func (s *SSHLogger) FileTransfer(operation string, localPath string, remotePath string, size int64, success bool, duration time.Duration) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":        s.server.Host,
		"username":    s.username,
		"operation":   operation,
		"local_path":  localPath,
		"remote_path": remotePath,
		"size_bytes":  size,
		"duration":    duration.String(),
		"success":     success,
	}).log(level, "File transfer operation")
}

// ServiceOperation logs systemd service operations
func (s *SSHLogger) ServiceOperation(action string, serviceName string, result string, duration time.Duration) {
	var level LogLevel
	switch result {
	case "success":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	default:
		level = WarnLevel
	}

	s.WithFields(map[string]any{
		"host":            s.server.Host,
		"username":        s.username,
		"service":         serviceName,
		"action":          action,
		"result":          result,
		"duration":        duration.String(),
		"uses_sudo":       !s.isRoot,
		"security_locked": s.server.SecurityLocked,
	}).log(level, "Service %s %s: %s", action, serviceName, result)
}

// UserOperation logs user management operations
func (s *SSHLogger) UserOperation(action string, targetUser string, success bool, details string) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":        s.server.Host,
		"username":    s.username,
		"action":      action,
		"target_user": targetUser,
		"success":     success,
		"details":     details,
	}).log(level, "User management operation: %s for %s", action, targetUser)
}

// DirectoryOperation logs directory creation/management operations
func (s *SSHLogger) DirectoryOperation(action string, path string, permissions string, owner string, success bool) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":        s.server.Host,
		"username":    s.username,
		"action":      action,
		"path":        path,
		"permissions": permissions,
		"owner":       owner,
		"success":     success,
	}).log(level, "Directory operation: %s %s", action, path)
}

// SecurityOperation logs security-related operations
func (s *SSHLogger) SecurityOperation(operation string, component string, success bool, details string) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":      s.server.Host,
		"username":  s.username,
		"operation": operation,
		"component": component,
		"success":   success,
		"details":   details,
	}).log(level, "Security operation: %s %s", operation, component)
}

// Troubleshooting and diagnostics

// TroubleshootStart logs the beginning of troubleshooting
func (s *SSHLogger) TroubleshootStart(diagnosticType string) {
	s.WithFields(map[string]any{
		"host":            s.server.Host,
		"username":        s.username,
		"diagnostic_type": diagnosticType,
	}).Info("Starting SSH troubleshooting")
}

// DiagnosticStep logs individual diagnostic steps
func (s *SSHLogger) DiagnosticStep(step string, status string, message string, suggestion string) {
	var level LogLevel
	switch status {
	case "success":
		level = DebugLevel
	case "warning":
		level = WarnLevel
	case "error":
		level = ErrorLevel
	default:
		level = InfoLevel
	}

	fields := map[string]any{
		"host":      s.server.Host,
		"username":  s.username,
		"diag_step": step,
		"status":    status,
	}

	if suggestion != "" {
		fields["suggestion"] = suggestion
	}

	s.WithFields(fields).log(level, "Diagnostic: %s", message)
}

// AutoFix logs automatic fix attempts
func (s *SSHLogger) AutoFix(issue string, fix string, success bool, details string) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":     s.server.Host,
		"username": s.username,
		"issue":    issue,
		"fix":      fix,
		"success":  success,
		"details":  details,
	}).log(level, "Auto-fix attempt: %s", issue)
}

// Connection pool and health logging

// PoolHealthMetrics logs connection pool health metrics
func (s *SSHLogger) PoolHealthMetrics(totalConns int64, healthyConns int64, avgResponseTime time.Duration, errorRate float64) {
	s.WithFields(map[string]any{
		"total_connections":     totalConns,
		"healthy_connections":   healthyConns,
		"unhealthy_connections": totalConns - healthyConns,
		"avg_response_time":     avgResponseTime.String(),
		"error_rate_percent":    fmt.Sprintf("%.2f", errorRate*100),
	}).Debug("Connection pool health metrics")
}

// ConnectionRecovery logs connection recovery attempts
func (s *SSHLogger) ConnectionRecovery(attempt int, maxAttempts int, success bool, err error) {
	level := InfoLevel
	if !success {
		level = WarnLevel
		if attempt >= maxAttempts {
			level = ErrorLevel
		}
	}

	fields := map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"attempt":      attempt,
		"max_attempts": maxAttempts,
		"success":      success,
	}

	entry := s.WithFields(fields)
	if err != nil {
		entry = entry.WithError(err)
	}

	entry.log(level, "Connection recovery attempt")
}

// Deployment-specific logging

// DeploymentStart logs the start of a deployment operation
func (s *SSHLogger) DeploymentStart(appName string, version string, isFirstDeploy bool) {
	s.WithFields(map[string]any{
		"host":            s.server.Host,
		"username":        s.username,
		"app":             appName,
		"version":         version,
		"is_first_deploy": isFirstDeploy,
		"operation":       "deployment",
	}).Info("Starting application deployment")
}

// DeploymentStep logs deployment steps
func (s *SSHLogger) DeploymentStep(appName string, step string, message string, progressPct int) {
	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"app":          appName,
		"step":         step,
		"progress_pct": progressPct,
		"operation":    "deployment",
	}).Info("Deployment step: %s", message)
}

// DeploymentComplete logs deployment completion
func (s *SSHLogger) DeploymentComplete(appName string, version string, success bool, duration time.Duration, finalStatus string) {
	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"app":          appName,
		"version":      version,
		"duration":     duration.String(),
		"success":      success,
		"final_status": finalStatus,
		"operation":    "deployment",
	}).log(level, "Deployment completed")
}

// Utility functions for SSH context

// WithSSHContext adds common SSH context fields to a log entry
func WithSSHContext(server *models.Server, asRoot bool) *LogEntry {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	return WithFields(map[string]any{
		"ssh_host":     server.Host,
		"ssh_port":     server.Port,
		"ssh_username": username,
		"ssh_is_root":  asRoot,
		"server_name":  server.Name,
		"server_id":    server.ID,
	})
}

// LogSSHSetupStep logs setup steps in the format expected by the SSH package
func LogSSHSetupStep(server *models.Server, step string, status string, message string, progressPct int, details string) {
	fields := map[string]any{
		"server":       server.Name,
		"host":         server.Host,
		"step":         step,
		"status":       status,
		"progress_pct": progressPct,
		"operation":    "server_setup",
	}

	if details != "" {
		fields["details"] = details
	}

	var level LogLevel
	switch status {
	case "running":
		level = InfoLevel
	case "success":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	default:
		level = DebugLevel
	}

	WithFields(fields).log(level, "Setup: %s", message)
}

// LogSSHSecurityStep logs security steps in the format expected by the SSH package
func LogSSHSecurityStep(server *models.Server, step string, status string, message string, progressPct int, details string) {
	fields := map[string]any{
		"server":       server.Name,
		"host":         server.Host,
		"step":         step,
		"status":       status,
		"progress_pct": progressPct,
		"operation":    "security_lockdown",
	}

	if details != "" {
		fields["details"] = details
	}

	var level LogLevel
	switch status {
	case "running":
		level = InfoLevel
	case "success":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	default:
		level = DebugLevel
	}

	WithFields(fields).log(level, "Security: %s", message)
}

// FormatSSHOutput formats SSH command output for display
func FormatSSHOutput(output string, maxLines int) string {
	if output == "" {
		return "(no output)"
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= maxLines {
		return output
	}

	// Show first few and last few lines with truncation indicator
	showFirst := maxLines / 2
	showLast := maxLines - showFirst - 1

	result := strings.Join(lines[:showFirst], "\n")
	result += fmt.Sprintf("\n... (%d lines omitted) ...\n", len(lines)-maxLines+1)
	result += strings.Join(lines[len(lines)-showLast:], "\n")

	return result
}

// FormatConnectionError formats SSH connection errors with helpful context
func FormatConnectionError(err error, server *models.Server, asRoot bool) string {
	if err == nil {
		return ""
	}

	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	address := fmt.Sprintf("%s:%d", server.Host, server.Port)
	errStr := err.Error()

	// Provide context-specific error messages
	var suggestion string
	switch {
	case strings.Contains(errStr, "connection refused"):
		suggestion = "Check if SSH service is running and the port is correct"
	case strings.Contains(errStr, "no route to host"):
		suggestion = "Check network connectivity and firewall settings"
	case strings.Contains(errStr, "permission denied"):
		suggestion = "Check SSH key configuration and user permissions"
	case strings.Contains(errStr, "host key verification failed"):
		suggestion = "Run ssh-keyscan to accept the host key or clear old keys"
	case strings.Contains(errStr, "timeout"):
		suggestion = "Check network latency and server responsiveness"
	case strings.Contains(errStr, "no supported methods remain"):
		suggestion = "Check SSH key setup and authentication configuration"
	default:
		suggestion = "Check SSH configuration and network connectivity"
	}

	return fmt.Sprintf("SSH connection to %s@%s failed: %s\nSuggestion: %s",
		username, address, errStr, suggestion)
}

// Performance and metrics logging

// ConnectionMetrics logs detailed connection performance metrics
func (s *SSHLogger) ConnectionMetrics(connectTime time.Duration, firstCommandTime time.Duration, throughput float64) {
	s.WithFields(map[string]any{
		"host":            s.server.Host,
		"username":        s.username,
		"connect_time":    connectTime.String(),
		"first_cmd_time":  firstCommandTime.String(),
		"throughput_mbps": fmt.Sprintf("%.2f", throughput),
		"total_time":      (connectTime + firstCommandTime).String(),
	}).Debug("SSH connection performance metrics")
}

// CommandMetrics logs command execution metrics
func (s *SSHLogger) CommandMetrics(commandType string, count int, avgDuration time.Duration, successRate float64) {
	s.WithFields(map[string]any{
		"host":         s.server.Host,
		"username":     s.username,
		"command_type": commandType,
		"count":        count,
		"avg_duration": avgDuration.String(),
		"success_rate": fmt.Sprintf("%.2f%%", successRate*100),
	}).Debug("SSH command execution metrics")
}

// SecurityAudit logs security audit results
func (s *SSHLogger) SecurityAudit(auditType string, findings map[string]any, passed bool) {
	level := InfoLevel
	if !passed {
		level = WarnLevel
	}

	s.WithFields(map[string]any{
		"host":       s.server.Host,
		"audit_type": auditType,
		"passed":     passed,
		"findings":   findings,
	}).log(level, "Security audit: %s", auditType)
}

// Package-level SSH convenience functions

// LogSSHConnect logs SSH connection using the default logger
func LogSSHConnect(server *models.Server, asRoot bool, status string, duration time.Duration) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":     server.Host,
		"port":     server.Port,
		"username": username,
		"is_root":  asRoot,
		"server":   server.Name,
		"status":   status,
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	var level LogLevel
	switch status {
	case "connecting":
		level = InfoLevel
	case "connected":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	case "timeout":
		level = WarnLevel
	default:
		level = DebugLevel
	}

	WithFields(fields).log(level, "SSH connection %s", status)
}

// LogSSHCommand logs SSH command execution using the default logger
func LogSSHCommand(server *models.Server, asRoot bool, command string, success bool, duration time.Duration, output string) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	displayCmd := command
	if len(displayCmd) > 100 {
		displayCmd = displayCmd[:97] + "..."
	}

	fields := map[string]any{
		"host":     server.Host,
		"username": username,
		"command":  displayCmd,
		"success":  success,
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	if output != "" && len(output) < 200 {
		fields["output"] = output
	}

	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	WithFields(fields).log(level, "SSH command execution")
}

// LogSSHOperation logs SSH operation start/completion using the default logger
func LogSSHOperation(server *models.Server, asRoot bool, operation string, status string, duration time.Duration, err error) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":      server.Host,
		"username":  username,
		"operation": operation,
		"status":    status,
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	entry := WithFields(fields)
	if err != nil {
		entry = entry.WithError(err)
	}

	var level LogLevel
	switch status {
	case "started", "starting":
		level = InfoLevel
	case "completed", "success":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	default:
		level = DebugLevel
	}

	entry.log(level, "SSH operation %s: %s", operation, status)
}

// LogSSHError logs SSH errors with context using the default logger
func LogSSHError(server *models.Server, asRoot bool, operation string, err error) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	WithFields(map[string]any{
		"host":      server.Host,
		"username":  username,
		"operation": operation,
	}).WithError(err).Error("SSH operation failed")
}

// LogSSHFileTransfer logs file transfer operations using the default logger
func LogSSHFileTransfer(server *models.Server, asRoot bool, operation string, localPath string, remotePath string, success bool, size int64, duration time.Duration) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":        server.Host,
		"username":    username,
		"operation":   operation,
		"local_path":  localPath,
		"remote_path": remotePath,
		"success":     success,
	}

	if size > 0 {
		fields["size_bytes"] = size
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	WithFields(fields).log(level, "SSH file transfer: %s", operation)
}

// LogSSHService logs service operations using the default logger
func LogSSHService(server *models.Server, asRoot bool, action string, serviceName string, success bool, duration time.Duration) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":            server.Host,
		"username":        username,
		"service":         serviceName,
		"action":          action,
		"success":         success,
		"uses_sudo":       !asRoot,
		"security_locked": server.SecurityLocked,
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	WithFields(fields).log(level, "Service %s %s", action, serviceName)
}

// LogSSHHealth logs SSH health check results using the default logger
func LogSSHHealth(server *models.Server, asRoot bool, healthy bool, responseTime time.Duration, consecutiveFailures int) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	status := "healthy"
	level := DebugLevel
	if !healthy {
		status = "unhealthy"
		level = WarnLevel
		if consecutiveFailures >= 3 {
			level = ErrorLevel
		}
	}

	WithFields(map[string]any{
		"host":                 server.Host,
		"username":             username,
		"status":               status,
		"response_time":        responseTime.String(),
		"consecutive_failures": consecutiveFailures,
	}).log(level, "SSH health check: %s", status)
}

// LogSSHDeployment logs deployment operations using the default logger
func LogSSHDeployment(server *models.Server, appName string, version string, step string, status string, progressPct int, duration time.Duration) {
	fields := map[string]any{
		"host":         server.Host,
		"app":          appName,
		"version":      version,
		"step":         step,
		"status":       status,
		"progress_pct": progressPct,
		"operation":    "deployment",
	}

	if duration > 0 {
		fields["duration"] = duration.String()
	}

	var level LogLevel
	switch status {
	case "running", "started":
		level = InfoLevel
	case "success", "completed":
		level = InfoLevel
	case "failed":
		level = ErrorLevel
	default:
		level = DebugLevel
	}

	WithFields(fields).log(level, "Deployment %s: %s", step, status)
}

// LogSSHSecurity logs security operations using the default logger
func LogSSHSecurity(server *models.Server, asRoot bool, operation string, component string, success bool, details string) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":      server.Host,
		"username":  username,
		"operation": operation,
		"component": component,
		"success":   success,
	}

	if details != "" {
		fields["details"] = details
	}

	level := InfoLevel
	if !success {
		level = ErrorLevel
	}

	WithFields(fields).log(level, "Security operation: %s %s", operation, component)
}

// LogSSHTroubleshoot logs troubleshooting operations using the default logger
func LogSSHTroubleshoot(server *models.Server, asRoot bool, step string, status string, message string, suggestion string) {
	username := server.AppUsername
	if asRoot {
		username = server.RootUsername
	}

	fields := map[string]any{
		"host":      server.Host,
		"username":  username,
		"diag_step": step,
		"status":    status,
	}

	if suggestion != "" {
		fields["suggestion"] = suggestion
	}

	var level LogLevel
	switch status {
	case "success":
		level = DebugLevel
	case "warning":
		level = WarnLevel
	case "error":
		level = ErrorLevel
	default:
		level = InfoLevel
	}

	WithFields(fields).log(level, "SSH diagnostic: %s", message)
}
