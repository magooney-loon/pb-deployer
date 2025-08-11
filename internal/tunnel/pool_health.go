package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PoolHealthIntegration manages health monitoring integration with the connection pool
type PoolHealthIntegration struct {
	pool          Pool
	healthMonitor *PoolHealthMonitor
	tracer        PoolTracer
	config        HealthIntegrationConfig
	mu            sync.RWMutex
	running       bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// HealthIntegrationConfig configures health integration behavior
type HealthIntegrationConfig struct {
	AutoRecovery           bool
	RecoveryRetries        int
	RecoveryDelay          time.Duration
	HealthCheckInterval    time.Duration
	UnhealthyThreshold     int
	RecoveryTimeout        time.Duration
	EnablePreemptiveChecks bool
	AlertThresholds        AlertThresholds
}

// NewPoolHealthIntegration creates a new health integration manager
func NewPoolHealthIntegration(pool Pool, healthMonitor *PoolHealthMonitor,
	tracer PoolTracer, config HealthIntegrationConfig) *PoolHealthIntegration {

	if tracer == nil {
		tracer = &NoOpPoolTracer{}
	}

	return &PoolHealthIntegration{
		pool:          pool,
		healthMonitor: healthMonitor,
		tracer:        tracer,
		config:        config,
		stopCh:        make(chan struct{}),
	}
}

// Start begins health monitoring and integration
func (phi *PoolHealthIntegration) Start(ctx context.Context) {
	phi.mu.Lock()
	if phi.running {
		phi.mu.Unlock()
		return
	}
	phi.running = true
	phi.mu.Unlock()

	span := phi.tracer.TracePool(ctx, "health_integration_start")
	defer span.End()

	// Start health monitoring for all connections
	phi.healthMonitor.StartMonitoringAll(ctx)

	// Start integration goroutine
	phi.wg.Add(1)
	go phi.integrationLoop(ctx)

	span.Event("health_integration_started", map[string]interface{}{
		"auto_recovery":         phi.config.AutoRecovery,
		"health_check_interval": phi.config.HealthCheckInterval,
		"recovery_retries":      phi.config.RecoveryRetries,
	})
}

// Stop stops health monitoring and integration
func (phi *PoolHealthIntegration) Stop() {
	phi.mu.Lock()
	if !phi.running {
		phi.mu.Unlock()
		return
	}
	phi.running = false
	phi.mu.Unlock()

	// Stop monitoring
	phi.healthMonitor.StopMonitoringAll()

	// Stop integration loop
	close(phi.stopCh)
	phi.wg.Wait()

	// Reset stop channel for potential restart
	phi.stopCh = make(chan struct{})
}

// integrationLoop runs the main health integration loop
func (phi *PoolHealthIntegration) integrationLoop(ctx context.Context) {
	defer phi.wg.Done()

	ticker := time.NewTicker(phi.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-phi.stopCh:
			return
		case <-ticker.C:
			phi.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs a comprehensive health check and takes action
func (phi *PoolHealthIntegration) performHealthCheck(ctx context.Context) {
	span := phi.tracer.TracePool(ctx, "health_check")
	defer span.End()

	// Get health report from pool
	report := phi.pool.HealthCheck(ctx)

	span.SetFields(map[string]interface{}{
		"total_connections":   report.TotalConnections,
		"healthy_connections": report.HealthyConnections,
		"failed_connections":  report.FailedConnections,
	})

	// Process unhealthy connections
	for _, connHealth := range report.Connections {
		if !connHealth.Healthy {
			phi.handleUnhealthyConnection(ctx, connHealth.Key)
		}
	}

	// Check alert thresholds
	phi.checkAlertThresholds(ctx, report)

	span.Event("health_check_completed", map[string]interface{}{
		"unhealthy_count": report.FailedConnections,
		"health_ratio":    float64(report.HealthyConnections) / float64(report.TotalConnections),
	})
}

// handleUnhealthyConnection handles an unhealthy connection
func (phi *PoolHealthIntegration) handleUnhealthyConnection(ctx context.Context, key string) {
	span := phi.tracer.TracePool(ctx, "handle_unhealthy_connection")
	defer span.End()

	span.SetFields(map[string]interface{}{
		"connection_key": key,
		"auto_recovery":  phi.config.AutoRecovery,
	})

	if !phi.config.AutoRecovery {
		span.Event("auto_recovery_disabled")
		return
	}

	// Attempt to replace the connection
	if err := phi.replaceConnection(ctx, key); err != nil {
		span.EndWithError(err)
		span.Event("connection_replacement_failed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		span.Event("connection_replaced", map[string]interface{}{
			"connection_key": key,
		})
	}
}

// replaceConnection attempts to replace an unhealthy connection
func (phi *PoolHealthIntegration) replaceConnection(ctx context.Context, key string) error {
	span := phi.tracer.TracePool(ctx, "replace_connection")
	defer span.End()

	span.SetFields(map[string]interface{}{
		"connection_key": key,
		"max_retries":    phi.config.RecoveryRetries,
	})

	// Remove the unhealthy connection from monitoring
	phi.healthMonitor.RemoveConnection(key)

	// Try to create a new connection
	for attempt := 1; attempt <= phi.config.RecoveryRetries; attempt++ {
		span.Event("replacement_attempt", map[string]interface{}{
			"attempt": attempt,
		})

		// Create recovery context with timeout
		recoveryCtx, cancel := context.WithTimeout(ctx, phi.config.RecoveryTimeout)

		// Try to get a new connection (this will create one if needed)
		client, err := phi.pool.Get(recoveryCtx, key)
		cancel()

		if err == nil {
			// Success - release the connection back to the pool
			phi.pool.Release(key, client)

			// Add back to health monitoring
			phi.healthMonitor.AddConnection(key, client)

			span.Event("replacement_successful", map[string]interface{}{
				"attempt": attempt,
			})
			return nil
		}

		span.Event("replacement_attempt_failed", map[string]interface{}{
			"attempt": attempt,
			"error":   err.Error(),
		})

		// Wait before retrying (except on last attempt)
		if attempt < phi.config.RecoveryRetries {
			time.Sleep(phi.config.RecoveryDelay * time.Duration(attempt))
		}
	}

	err := fmt.Errorf("failed to replace connection %s after %d attempts", key, phi.config.RecoveryRetries)
	span.EndWithError(err)
	return err
}

// checkAlertThresholds checks if any alert thresholds are breached
func (phi *PoolHealthIntegration) checkAlertThresholds(ctx context.Context, report HealthReport) {
	span := phi.tracer.TracePool(ctx, "check_alert_thresholds")
	defer span.End()

	thresholds := phi.config.AlertThresholds
	alerts := []string{}

	// Check connection failure threshold
	if report.FailedConnections >= thresholds.MaxConnectionFailures {
		alerts = append(alerts, fmt.Sprintf("connection_failures_exceeded: %d >= %d",
			report.FailedConnections, thresholds.MaxConnectionFailures))
	}

	// Check minimum healthy connections
	if report.HealthyConnections < thresholds.MinHealthyConnections {
		alerts = append(alerts, fmt.Sprintf("min_healthy_connections_breached: %d < %d",
			report.HealthyConnections, thresholds.MinHealthyConnections))
	}

	// Check average response time
	avgResponseTime := phi.calculateAverageResponseTime(report)
	if avgResponseTime > thresholds.MaxResponseTime {
		alerts = append(alerts, fmt.Sprintf("response_time_exceeded: %v > %v",
			avgResponseTime, thresholds.MaxResponseTime))
	}

	if len(alerts) > 0 {
		span.Event("alert_thresholds_breached", map[string]interface{}{
			"alerts":      alerts,
			"alert_count": len(alerts),
		})

		// Here you would integrate with your alerting system
		phi.sendAlerts(ctx, alerts, report)
	}
}

// calculateAverageResponseTime calculates the average response time from the health report
func (phi *PoolHealthIntegration) calculateAverageResponseTime(report HealthReport) time.Duration {
	if len(report.Connections) == 0 {
		return 0
	}

	var total time.Duration
	count := 0

	for _, conn := range report.Connections {
		if conn.ResponseTime > 0 {
			total += conn.ResponseTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// sendAlerts sends alerts to the configured alerting system
func (phi *PoolHealthIntegration) sendAlerts(ctx context.Context, alerts []string, report HealthReport) {
	span := phi.tracer.TracePool(ctx, "send_alerts")
	defer span.End()

	span.SetFields(map[string]interface{}{
		"alert_count":         len(alerts),
		"total_connections":   report.TotalConnections,
		"healthy_connections": report.HealthyConnections,
		"failed_connections":  report.FailedConnections,
	})

	// This is where you would integrate with your alerting system
	// For now, we just log the alerts through tracing
	for i, alert := range alerts {
		span.Event("alert_triggered", map[string]interface{}{
			"alert_index": i,
			"alert":       alert,
		})
	}
}

// GetHealthStatus returns the current health integration status
func (phi *PoolHealthIntegration) GetHealthStatus(ctx context.Context) HealthIntegrationStatus {
	phi.mu.RLock()
	running := phi.running
	phi.mu.RUnlock()

	poolReport := phi.pool.HealthCheck(ctx)
	avgResponseTime := phi.calculateAverageResponseTime(poolReport)

	return HealthIntegrationStatus{
		Running:              running,
		AutoRecoveryEnabled:  phi.config.AutoRecovery,
		TotalConnections:     poolReport.TotalConnections,
		HealthyConnections:   poolReport.HealthyConnections,
		UnhealthyConnections: poolReport.FailedConnections,
		AverageResponseTime:  avgResponseTime,
		LastCheck:            poolReport.CheckedAt,
		Config:               phi.config,
	}
}

// HealthIntegrationStatus represents the current status of health integration
type HealthIntegrationStatus struct {
	Running              bool                    `json:"running"`
	AutoRecoveryEnabled  bool                    `json:"auto_recovery_enabled"`
	TotalConnections     int                     `json:"total_connections"`
	HealthyConnections   int                     `json:"healthy_connections"`
	UnhealthyConnections int                     `json:"unhealthy_connections"`
	AverageResponseTime  time.Duration           `json:"average_response_time"`
	LastCheck            time.Time               `json:"last_check"`
	Config               HealthIntegrationConfig `json:"config"`
}

// UpdateConfig updates the health integration configuration
func (phi *PoolHealthIntegration) UpdateConfig(config HealthIntegrationConfig) {
	phi.mu.Lock()
	defer phi.mu.Unlock()
	phi.config = config
}

// ForceHealthCheck forces an immediate health check
func (phi *PoolHealthIntegration) ForceHealthCheck(ctx context.Context) HealthReport {
	span := phi.tracer.TracePool(ctx, "force_health_check")
	defer span.End()

	report := phi.pool.HealthCheck(ctx)

	// Process any unhealthy connections immediately
	for _, connHealth := range report.Connections {
		if !connHealth.Healthy {
			phi.handleUnhealthyConnection(ctx, connHealth.Key)
		}
	}

	span.Event("forced_health_check_completed", map[string]interface{}{
		"total_connections":   report.TotalConnections,
		"healthy_connections": report.HealthyConnections,
		"failed_connections":  report.FailedConnections,
	})

	return report
}

// scheduleHealthChecks schedules periodic health checks (alternative to the main loop)
func (phi *PoolHealthIntegration) scheduleHealthChecks(ctx context.Context) {
	if !phi.config.EnablePreemptiveChecks {
		return
	}

	go func() {
		ticker := time.NewTicker(phi.config.HealthCheckInterval / 2)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-phi.stopCh:
				return
			case <-ticker.C:
				// Perform lightweight preemptive checks
				phi.performPreemptiveCheck(ctx)
			}
		}
	}()
}

// performPreemptiveCheck performs lightweight preemptive health checks
func (phi *PoolHealthIntegration) performPreemptiveCheck(ctx context.Context) {
	span := phi.tracer.TracePool(ctx, "preemptive_health_check")
	defer span.End()

	// Get list of unhealthy connections without full health check
	unhealthyKeys := phi.healthMonitor.GetUnhealthyConnections(ctx)

	if len(unhealthyKeys) > 0 {
		span.Event("unhealthy_connections_detected", map[string]interface{}{
			"count": len(unhealthyKeys),
			"keys":  unhealthyKeys,
		})

		// Handle each unhealthy connection
		for _, key := range unhealthyKeys {
			phi.handleUnhealthyConnection(ctx, key)
		}
	}
}

// DefaultHealthIntegrationConfig returns default health integration configuration
func DefaultHealthIntegrationConfig() HealthIntegrationConfig {
	return HealthIntegrationConfig{
		AutoRecovery:           true,
		RecoveryRetries:        3,
		RecoveryDelay:          2 * time.Second,
		HealthCheckInterval:    30 * time.Second,
		UnhealthyThreshold:     3,
		RecoveryTimeout:        30 * time.Second,
		EnablePreemptiveChecks: true,
		AlertThresholds: AlertThresholds{
			MaxConnectionFailures: 5,
			MinHealthyConnections: 1,
			MaxResponseTime:       10 * time.Second,
		},
	}
}

// HealthIntegrationMetrics tracks metrics for health integration
type HealthIntegrationMetrics struct {
	TotalHealthChecks    int64     `json:"total_health_checks"`
	UnhealthyDetected    int64     `json:"unhealthy_detected"`
	RecoveryAttempts     int64     `json:"recovery_attempts"`
	SuccessfulRecoveries int64     `json:"successful_recoveries"`
	FailedRecoveries     int64     `json:"failed_recoveries"`
	AlertsTriggered      int64     `json:"alerts_triggered"`
	LastHealthCheck      time.Time `json:"last_health_check"`
	LastRecoveryAttempt  time.Time `json:"last_recovery_attempt"`
	mu                   sync.RWMutex
}

// GetMetrics returns health integration metrics
func (phi *PoolHealthIntegration) GetMetrics() HealthIntegrationMetrics {
	// This would be implemented with actual metrics tracking
	// For now, return empty metrics
	return HealthIntegrationMetrics{}
}

// ResetMetrics resets all health integration metrics
func (phi *PoolHealthIntegration) ResetMetrics() {
	// This would reset actual metrics
	// For now, it's a no-op
}
