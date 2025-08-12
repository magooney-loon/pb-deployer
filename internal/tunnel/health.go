package tunnel

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"
)

// HealthChecker manages health monitoring for SSH connections
type HealthChecker struct {
	client           SSHClient
	tracer           SSHTracer
	config           HealthCheckConfig
	mu               sync.RWMutex
	lastCheck        time.Time
	lastResult       HealthResult
	isRunning        bool
	stopCh           chan struct{}
	consecutiveFails int
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Healthy      bool
	ResponseTime time.Duration
	Error        error
	Timestamp    time.Time
	CheckType    string
	Details      map[string]any
}

// NewHealthChecker creates a new health checker for an SSH client
func NewHealthChecker(client SSHClient, tracer SSHTracer) *HealthChecker {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &HealthChecker{
		client:    client,
		tracer:    tracer,
		config:    *DefaultHealthCheckConfig(),
		stopCh:    make(chan struct{}),
		lastCheck: time.Now(),
	}
}

// SetConfig updates the health check configuration
func (hc *HealthChecker) SetConfig(config HealthCheckConfig) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.config = config
}

// CheckHealth performs a single health check
func (hc *HealthChecker) CheckHealth(ctx context.Context) HealthResult {
	span := hc.tracer.TraceConnection(ctx, "health_check", 0, "")
	defer span.End()

	start := time.Now()
	result := HealthResult{
		Timestamp: start,
		CheckType: "basic",
		Details:   make(map[string]any),
	}

	// Check if client is connected
	if !hc.client.IsConnected() {
		result.Healthy = false
		result.Error = ErrClientNotConnected
		result.ResponseTime = time.Since(start)

		span.Event("health_check_failed", map[string]any{
			"reason":        "not_connected",
			"response_time": result.ResponseTime,
		})

		hc.updateResult(result)
		return result
	}

	// Perform a simple command to test responsiveness
	output, err := hc.client.Execute(ctx, "echo 'health_check'")
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Healthy = false
		result.Error = err
		result.Details["error"] = err.Error()

		span.Event("health_check_failed", map[string]any{
			"reason":        "command_failed",
			"error":         err.Error(),
			"response_time": result.ResponseTime,
		})
	} else if output != "health_check\n" && output != "health_check" {
		result.Healthy = false
		result.Error = fmt.Errorf("unexpected health check response: %s", output)
		result.Details["unexpected_output"] = output

		span.Event("health_check_failed", map[string]any{
			"reason":        "unexpected_output",
			"output":        output,
			"response_time": result.ResponseTime,
		})
	} else {
		result.Healthy = true
		result.Details["output"] = output

		span.Event("health_check_passed", map[string]any{
			"response_time": result.ResponseTime,
		})
	}

	// Check response time threshold
	if result.ResponseTime > hc.config.Timeout {
		result.Healthy = false
		if result.Error == nil {
			result.Error = fmt.Errorf("health check timeout: %v > %v", result.ResponseTime, hc.config.Timeout)
		}
		result.Details["timeout_exceeded"] = true
	}

	hc.updateResult(result)
	return result
}

// IsHealthy returns the current health status
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	hc.mu.RLock()

	// If we haven't checked recently, perform a new check
	if time.Since(hc.lastCheck) > hc.config.Interval {
		hc.mu.RUnlock()
		result := hc.CheckHealth(ctx)
		return result.Healthy
	}

	healthy := hc.lastResult.Healthy
	hc.mu.RUnlock()
	return healthy
}

// GetLastResult returns the last health check result
func (hc *HealthChecker) GetLastResult() HealthResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastResult
}

// GetHealthReport returns a comprehensive health report
func (hc *HealthChecker) GetHealthReport(ctx context.Context) HealthReport {
	span := hc.tracer.TraceConnection(ctx, "health_report", 0, "")
	defer span.End()

	hc.mu.RLock()
	lastResult := hc.lastResult
	consecutiveFails := hc.consecutiveFails
	isRunning := hc.isRunning
	hc.mu.RUnlock()

	// Perform fresh health check
	currentResult := hc.CheckHealth(ctx)

	// Get connection info if available
	var connectionInfo map[string]any
	if clientImpl, ok := hc.client.(*sshClient); ok {
		connectionInfo = clientImpl.GetConnectionInfo()
	}

	report := HealthReport{
		TotalConnections:   1,
		HealthyConnections: 0,
		FailedConnections:  1,
		CheckedAt:          time.Now(),
		Connections: []ConnectionHealth{
			{
				Key:          hc.getConnectionKey(),
				Healthy:      currentResult.Healthy,
				LastUsed:     lastResult.Timestamp,
				UseCount:     int64(consecutiveFails), // Repurpose for fail count
				ResponseTime: currentResult.ResponseTime,
				Error:        hc.formatError(currentResult.Error),
			},
		},
	}

	if currentResult.Healthy {
		report.HealthyConnections = 1
		report.FailedConnections = 0
	}

	// Add connection details if available
	if connectionInfo != nil {
		report.Connections[0].LastUsed = connectionInfo["last_used"].(time.Time)
		if host, ok := connectionInfo["host"]; ok {
			report.Connections[0].Key = fmt.Sprintf("%v", host)
		}
	}

	span.Event("health_report_generated", map[string]any{
		"healthy":           currentResult.Healthy,
		"response_time":     currentResult.ResponseTime,
		"consecutive_fails": consecutiveFails,
		"monitoring_active": isRunning,
	})

	return report
}

// StartMonitoring starts continuous health monitoring
func (hc *HealthChecker) StartMonitoring(ctx context.Context) {
	hc.mu.Lock()
	if hc.isRunning {
		hc.mu.Unlock()
		return
	}
	hc.isRunning = true
	hc.mu.Unlock()

	span := hc.tracer.TraceConnection(ctx, "health_monitoring_start", 0, "")
	defer span.End()

	span.Event("monitoring_started", map[string]any{
		"interval": hc.config.Interval,
		"timeout":  hc.config.Timeout,
	})

	go hc.monitoringLoop(ctx)
}

// StopMonitoring stops continuous health monitoring
func (hc *HealthChecker) StopMonitoring() {
	hc.mu.Lock()
	if !hc.isRunning {
		hc.mu.Unlock()
		return
	}
	hc.isRunning = false
	hc.mu.Unlock()

	close(hc.stopCh)
	hc.stopCh = make(chan struct{}) // Reset for potential restart
}

// RecoverConnection attempts to recover an unhealthy connection
func (hc *HealthChecker) RecoverConnection(ctx context.Context) error {
	span := hc.tracer.TraceConnection(ctx, "connection_recovery", 0, "")
	defer span.End()

	span.Event("recovery_started")

	// Close existing connection
	if err := hc.client.Close(); err != nil {
		span.Event("close_failed", map[string]any{
			"error": err.Error(),
		})
	}

	// Attempt to reconnect
	if err := hc.client.Connect(ctx); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("recovery failed: %w", err)
	}

	// Verify recovery with health check
	result := hc.CheckHealth(ctx)
	if !result.Healthy {
		err := fmt.Errorf("recovery verification failed: %v", result.Error)
		span.EndWithError(err)
		return err
	}

	span.Event("recovery_completed", map[string]any{
		"response_time": result.ResponseTime,
	})

	return nil
}

// monitoringLoop runs the continuous health monitoring
func (hc *HealthChecker) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.mu.Lock()
			hc.isRunning = false
			hc.mu.Unlock()
			return

		case <-hc.stopCh:
			hc.mu.Lock()
			hc.isRunning = false
			hc.mu.Unlock()
			return

		case <-ticker.C:
			result := hc.CheckHealth(ctx)

			if !result.Healthy {
				hc.handleUnhealthyConnection(ctx, result)
			} else {
				hc.mu.Lock()
				hc.consecutiveFails = 0
				hc.mu.Unlock()
			}
		}
	}
}

// handleUnhealthyConnection handles an unhealthy connection
func (hc *HealthChecker) handleUnhealthyConnection(ctx context.Context, result HealthResult) {
	hc.mu.Lock()
	hc.consecutiveFails++
	consecutiveFails := hc.consecutiveFails
	enableAutoRecovery := hc.config.EnableAutoRecovery
	maxFails := hc.config.MaxConsecutiveFails
	recoveryRetries := hc.config.RecoveryRetries
	hc.mu.Unlock()

	span := hc.tracer.TraceConnection(ctx, "unhealthy_connection", 0, "")
	defer span.End()

	span.Event("unhealthy_detected", map[string]any{
		"consecutive_fails": consecutiveFails,
		"error":             hc.formatError(result.Error),
		"response_time":     result.ResponseTime,
	})

	// Attempt recovery if enabled and threshold reached
	if enableAutoRecovery && consecutiveFails >= maxFails {
		span.Event("recovery_triggered", map[string]any{
			"max_retries": recoveryRetries,
		})

		for attempt := 1; attempt <= recoveryRetries; attempt++ {
			if err := hc.RecoverConnection(ctx); err == nil {
				span.Event("recovery_successful", map[string]any{
					"attempt": attempt,
				})
				return
			} else {
				span.Event("recovery_failed", map[string]any{
					"attempt": attempt,
					"error":   err.Error(),
				})

				if attempt < recoveryRetries {
					time.Sleep(time.Duration(attempt) * time.Second)
				}
			}
		}

		span.Event("recovery_exhausted")
	}
}

// updateResult updates the stored health check result
func (hc *HealthChecker) updateResult(result HealthResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastResult = result
	hc.lastCheck = result.Timestamp
}

// getConnectionKey returns a key identifying this connection
func (hc *HealthChecker) getConnectionKey() string {
	if clientImpl, ok := hc.client.(*sshClient); ok {
		info := clientImpl.GetConnectionInfo()
		if host, ok := info["host"]; ok {
			if port, portOk := info["port"]; portOk {
				if user, userOk := info["user"]; userOk {
					return fmt.Sprintf("%v@%v:%v", user, host, port)
				}
			}
		}
	}
	return "unknown"
}

// formatError safely formats an error for reporting
func (hc *HealthChecker) formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// PoolHealthMonitor monitors health for multiple connections
type PoolHealthMonitor struct {
	checkers map[string]*HealthChecker
	tracer   SSHTracer
	mu       sync.RWMutex
	config   HealthCheckConfig
}

// NewPoolHealthMonitor creates a health monitor for connection pools
func NewPoolHealthMonitor(tracer SSHTracer) *PoolHealthMonitor {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &PoolHealthMonitor{
		checkers: make(map[string]*HealthChecker),
		tracer:   tracer,
		config:   *DefaultHealthCheckConfig(),
	}
}

// AddConnection adds a connection to monitor
func (phm *PoolHealthMonitor) AddConnection(key string, client SSHClient) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	checker := NewHealthChecker(client, phm.tracer)
	checker.SetConfig(phm.config)
	phm.checkers[key] = checker
}

// RemoveConnection removes a connection from monitoring
func (phm *PoolHealthMonitor) RemoveConnection(key string) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	if checker, exists := phm.checkers[key]; exists {
		checker.StopMonitoring()
		delete(phm.checkers, key)
	}
}

// CheckAllHealth checks health of all monitored connections
func (phm *PoolHealthMonitor) CheckAllHealth(ctx context.Context) HealthReport {
	span := phm.tracer.TraceConnection(ctx, "pool_health_check", 0, "")
	defer span.End()

	phm.mu.RLock()
	checkers := make(map[string]*HealthChecker)
	maps.Copy(checkers, phm.checkers)
	phm.mu.RUnlock()

	report := HealthReport{
		TotalConnections:   len(checkers),
		HealthyConnections: 0,
		FailedConnections:  0,
		CheckedAt:          time.Now(),
		Connections:        make([]ConnectionHealth, 0, len(checkers)),
	}

	for key, checker := range checkers {
		result := checker.CheckHealth(ctx)

		connHealth := ConnectionHealth{
			Key:          key,
			Healthy:      result.Healthy,
			LastUsed:     result.Timestamp,
			ResponseTime: result.ResponseTime,
			Error:        checker.formatError(result.Error),
		}

		if result.Healthy {
			report.HealthyConnections++
		} else {
			report.FailedConnections++
		}

		report.Connections = append(report.Connections, connHealth)
	}

	span.Event("pool_health_checked", map[string]any{
		"total_connections":   report.TotalConnections,
		"healthy_connections": report.HealthyConnections,
		"failed_connections":  report.FailedConnections,
	})

	return report
}

// StartMonitoringAll starts monitoring for all connections
func (phm *PoolHealthMonitor) StartMonitoringAll(ctx context.Context) {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	for _, checker := range phm.checkers {
		checker.StartMonitoring(ctx)
	}
}

// StopMonitoringAll stops monitoring for all connections
func (phm *PoolHealthMonitor) StopMonitoringAll() {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	for _, checker := range phm.checkers {
		checker.StopMonitoring()
	}
}

// SetConfig updates configuration for all health checkers
func (phm *PoolHealthMonitor) SetConfig(config HealthCheckConfig) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	phm.config = config
	for _, checker := range phm.checkers {
		checker.SetConfig(config)
	}
}

// GetUnhealthyConnections returns keys of unhealthy connections
func (phm *PoolHealthMonitor) GetUnhealthyConnections(ctx context.Context) []string {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	var unhealthy []string
	for key, checker := range phm.checkers {
		if !checker.IsHealthy(ctx) {
			unhealthy = append(unhealthy, key)
		}
	}

	return unhealthy
}
