package ssh

import (
	"fmt"
	"sync"
	"time"

	"pb-deployer/internal/logger"
	"pb-deployer/internal/models"
)

// HealthMonitor monitors SSH connection health and provides recovery mechanisms
type HealthMonitor struct {
	connections map[string]*MonitoredConnection
	mutex       sync.RWMutex
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	metrics     *HealthMetrics
}

// MonitoredConnection wraps an SSH connection with health tracking
type MonitoredConnection struct {
	manager          *SSHManager
	server           *models.Server
	isRoot           bool
	lastHealthCheck  time.Time
	consecutiveFails int
	totalConnections int64
	totalErrors      int64
	avgResponseTime  time.Duration
	status           ConnectionStatus
	mutex            sync.RWMutex
}

// ConnectionStatus represents the current status of a connection
type ConnectionStatus int

const (
	StatusHealthy ConnectionStatus = iota
	StatusDegraded
	StatusUnhealthy
	StatusRecovering
	StatusFailed
)

// HealthMetrics tracks overall health statistics
type HealthMetrics struct {
	TotalConnections     int64         `json:"total_connections"`
	HealthyConnections   int64         `json:"healthy_connections"`
	UnhealthyConnections int64         `json:"unhealthy_connections"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	ErrorRate            float64       `json:"error_rate"`
	LastUpdate           time.Time     `json:"last_update"`
	mutex                sync.RWMutex
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	ConnectionKey    string           `json:"connection_key"`
	Status           ConnectionStatus `json:"status"`
	ResponseTime     time.Duration    `json:"response_time"`
	Error            error            `json:"error,omitempty"`
	Timestamp        time.Time        `json:"timestamp"`
	ConsecutiveFails int              `json:"consecutive_fails"`
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor() *HealthMonitor {
	return &HealthMonitor{
		connections: make(map[string]*MonitoredConnection),
		stopChan:    make(chan struct{}),
		metrics:     &HealthMetrics{},
	}
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() {
	hm.mutex.Lock()
	if hm.running {
		hm.mutex.Unlock()
		logger.Debug("Health monitor is already running")
		return
	}
	hm.running = true
	hm.mutex.Unlock()

	logger.Info("Starting SSH connection health monitor")
	hm.wg.Add(1)
	go hm.healthCheckLoop()
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.mutex.Lock()
	if !hm.running {
		hm.mutex.Unlock()
		logger.Debug("Health monitor is not running")
		return
	}
	hm.running = false
	hm.mutex.Unlock()

	logger.Info("Stopping SSH connection health monitor")
	close(hm.stopChan)
	hm.wg.Wait()

	// Clean up all connections
	hm.mutex.Lock()
	connectionCount := len(hm.connections)
	for key, conn := range hm.connections {
		conn.Close()
		delete(hm.connections, key)
	}
	hm.mutex.Unlock()

	logger.WithField("closed_connections", connectionCount).Info("Health monitor stopped and connections cleaned up")
}

// RegisterConnection registers a connection for monitoring
func (hm *HealthMonitor) RegisterConnection(server *models.Server, asRoot bool, manager *SSHManager) string {
	key := hm.connectionKey(server, asRoot)

	conn := &MonitoredConnection{
		manager:         manager,
		server:          server,
		isRoot:          asRoot,
		lastHealthCheck: time.Now(),
		status:          StatusHealthy,
	}

	hm.mutex.Lock()
	hm.connections[key] = conn
	hm.mutex.Unlock()

	logger.WithFields(map[string]interface{}{
		"host":           server.Host,
		"port":           server.Port,
		"as_root":        asRoot,
		"connection_key": key,
	}).Debug("Registered SSH connection for health monitoring")

	return key
}

// UnregisterConnection removes a connection from monitoring
func (hm *HealthMonitor) UnregisterConnection(key string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()

	if conn, exists := hm.connections[key]; exists {
		logger.WithFields(map[string]interface{}{
			"host":           conn.server.Host,
			"port":           conn.server.Port,
			"connection_key": key,
		}).Debug("Unregistering SSH connection from health monitoring")
		conn.Close()
		delete(hm.connections, key)
	}
}

// GetConnection retrieves a monitored connection
func (hm *HealthMonitor) GetConnection(key string) (*MonitoredConnection, bool) {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	conn, exists := hm.connections[key]
	return conn, exists
}

// CheckConnectionHealth performs a health check on a specific connection
func (hm *HealthMonitor) CheckConnectionHealth(key string) (*HealthCheckResult, error) {
	conn, exists := hm.GetConnection(key)
	if !exists {
		return nil, fmt.Errorf("connection %s not found", key)
	}

	return hm.performHealthCheck(key, conn)
}

// GetHealthMetrics returns current health metrics
func (hm *HealthMonitor) GetHealthMetrics() *HealthMetrics {
	hm.metrics.mutex.RLock()
	defer hm.metrics.mutex.RUnlock()

	// Create a copy to avoid race conditions
	return &HealthMetrics{
		TotalConnections:     hm.metrics.TotalConnections,
		HealthyConnections:   hm.metrics.HealthyConnections,
		UnhealthyConnections: hm.metrics.UnhealthyConnections,
		AverageResponseTime:  hm.metrics.AverageResponseTime,
		ErrorRate:            hm.metrics.ErrorRate,
		LastUpdate:           hm.metrics.LastUpdate,
	}
}

// GetConnectionStatus returns the status of all monitored connections
func (hm *HealthMonitor) GetConnectionStatus() map[string]ConnectionStatus {
	hm.mutex.RLock()
	defer hm.mutex.RUnlock()

	status := make(map[string]ConnectionStatus)
	for key, conn := range hm.connections {
		conn.mutex.RLock()
		status[key] = conn.status
		conn.mutex.RUnlock()
	}

	return status
}

// RecoverConnection attempts to recover an unhealthy connection
func (hm *HealthMonitor) RecoverConnection(key string) error {
	conn, exists := hm.GetConnection(key)
	if !exists {
		logger.WithField("connection_key", key).Warn("Attempted to recover connection that was not found")
		return fmt.Errorf("connection %s not found", key)
	}

	logger.WithFields(map[string]interface{}{
		"host":           conn.server.Host,
		"port":           conn.server.Port,
		"connection_key": key,
	}).Info("Attempting to recover unhealthy SSH connection")

	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	// Mark as recovering
	conn.status = StatusRecovering

	// Close existing manager
	if conn.manager != nil {
		conn.manager.Close()
	}

	// Create new manager
	newManager, err := NewSSHManager(conn.server, conn.isRoot)
	if err != nil {
		conn.status = StatusFailed
		logger.WithFields(map[string]interface{}{
			"host":           conn.server.Host,
			"port":           conn.server.Port,
			"connection_key": key,
		}).WithError(err).Error("Failed to create new SSH manager during recovery")
		return fmt.Errorf("failed to create new SSH manager: %w", err)
	}

	// Test the new connection
	if err := newManager.TestConnection(); err != nil {
		newManager.Close()
		conn.status = StatusFailed
		logger.WithFields(map[string]interface{}{
			"host":           conn.server.Host,
			"port":           conn.server.Port,
			"connection_key": key,
		}).WithError(err).Error("New connection test failed during recovery")
		return fmt.Errorf("new connection test failed: %w", err)
	}

	// Update connection
	conn.manager = newManager
	conn.consecutiveFails = 0
	conn.status = StatusHealthy
	conn.lastHealthCheck = time.Now()

	logger.WithFields(map[string]interface{}{
		"host":           conn.server.Host,
		"port":           conn.server.Port,
		"connection_key": key,
	}).Info("Successfully recovered SSH connection")

	return nil
}

// healthCheckLoop runs continuous health checks
func (hm *HealthMonitor) healthCheckLoop() {
	defer hm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.performAllHealthChecks()
		case <-hm.stopChan:
			return
		}
	}
}

// performAllHealthChecks checks health of all connections
func (hm *HealthMonitor) performAllHealthChecks() {
	hm.mutex.RLock()
	connections := make(map[string]*MonitoredConnection)
	for key, conn := range hm.connections {
		connections[key] = conn
	}
	hm.mutex.RUnlock()

	var totalHealthy, totalUnhealthy int64
	var totalResponseTime time.Duration
	var responseCount int64

	for key, conn := range connections {
		result, err := hm.performHealthCheck(key, conn)
		if err != nil {
			continue
		}

		if result.Status == StatusHealthy {
			totalHealthy++
		} else {
			totalUnhealthy++
		}

		if result.ResponseTime > 0 {
			totalResponseTime += result.ResponseTime
			responseCount++
		}
	}

	// Update metrics
	hm.updateMetrics(totalHealthy, totalUnhealthy, totalResponseTime, responseCount)
}

// performHealthCheck performs a health check on a single connection
func (hm *HealthMonitor) performHealthCheck(key string, conn *MonitoredConnection) (*HealthCheckResult, error) {
	conn.mutex.Lock()
	defer conn.mutex.Unlock()

	result := &HealthCheckResult{
		ConnectionKey: key,
		Timestamp:     time.Now(),
	}

	if conn.manager == nil {
		result.Status = StatusFailed
		result.Error = fmt.Errorf("SSH manager is nil")
		conn.status = StatusFailed
		conn.consecutiveFails++
		logger.WithFields(map[string]interface{}{
			"host":              conn.server.Host,
			"port":              conn.server.Port,
			"connection_key":    key,
			"consecutive_fails": conn.consecutiveFails,
		}).Error("Health check failed - SSH manager is nil")
		return result, result.Error
	}

	// Perform health check
	start := time.Now()
	err := conn.manager.TestConnection()
	responseTime := time.Since(start)
	result.ResponseTime = responseTime

	if err != nil {
		conn.consecutiveFails++
		result.Error = err
		result.ConsecutiveFails = conn.consecutiveFails

		// Determine status based on consecutive failures
		if conn.consecutiveFails >= 5 {
			conn.status = StatusFailed
			result.Status = StatusFailed
		} else if conn.consecutiveFails >= 3 {
			conn.status = StatusUnhealthy
			result.Status = StatusUnhealthy
		} else {
			conn.status = StatusDegraded
			result.Status = StatusDegraded
		}

		conn.totalErrors++

		logger.WithFields(map[string]interface{}{
			"host":              conn.server.Host,
			"port":              conn.server.Port,
			"connection_key":    key,
			"consecutive_fails": conn.consecutiveFails,
			"status":            result.Status.String(),
			"response_time":     responseTime.String(),
		}).WithError(err).Debug("SSH connection health check failed")
	} else {
		// Reset failure count on success
		wasUnhealthy := conn.consecutiveFails > 0
		conn.consecutiveFails = 0
		conn.status = StatusHealthy
		result.Status = StatusHealthy

		// Update average response time
		conn.totalConnections++
		if conn.avgResponseTime == 0 {
			conn.avgResponseTime = responseTime
		} else {
			// Simple moving average
			conn.avgResponseTime = (conn.avgResponseTime + responseTime) / 2
		}

		if wasUnhealthy {
			logger.WithFields(map[string]interface{}{
				"host":           conn.server.Host,
				"port":           conn.server.Port,
				"connection_key": key,
				"response_time":  responseTime.String(),
			}).Info("SSH connection health check recovered - connection is now healthy")
		}
	}

	conn.lastHealthCheck = time.Now()
	result.ConsecutiveFails = conn.consecutiveFails

	return result, nil
}

// updateMetrics updates the overall health metrics
func (hm *HealthMonitor) updateMetrics(healthy, unhealthy int64, totalResponseTime time.Duration, responseCount int64) {
	hm.metrics.mutex.Lock()
	defer hm.metrics.mutex.Unlock()

	hm.metrics.HealthyConnections = healthy
	hm.metrics.UnhealthyConnections = unhealthy
	hm.metrics.TotalConnections = healthy + unhealthy

	if responseCount > 0 {
		hm.metrics.AverageResponseTime = totalResponseTime / time.Duration(responseCount)
	}

	if hm.metrics.TotalConnections > 0 {
		hm.metrics.ErrorRate = float64(unhealthy) / float64(hm.metrics.TotalConnections)
	}

	hm.metrics.LastUpdate = time.Now()
}

// connectionKey generates a unique key for a connection
func (hm *HealthMonitor) connectionKey(server *models.Server, asRoot bool) string {
	userType := "app"
	if asRoot {
		userType = "root"
	}
	return fmt.Sprintf("%s:%d:%s:%s", server.Host, server.Port, server.ID, userType)
}

// MonitoredConnection methods

// IsHealthy returns whether the connection is healthy
func (mc *MonitoredConnection) IsHealthy() bool {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.status == StatusHealthy
}

// GetStatus returns the current connection status
func (mc *MonitoredConnection) GetStatus() ConnectionStatus {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.status
}

// GetManager returns the underlying SSH manager
func (mc *MonitoredConnection) GetManager() *SSHManager {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	return mc.manager
}

// GetStats returns connection statistics
func (mc *MonitoredConnection) GetStats() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	return map[string]interface{}{
		"status":            mc.status,
		"consecutive_fails": mc.consecutiveFails,
		"total_connections": mc.totalConnections,
		"total_errors":      mc.totalErrors,
		"avg_response_time": mc.avgResponseTime,
		"last_health_check": mc.lastHealthCheck,
	}
}

// Close closes the monitored connection
func (mc *MonitoredConnection) Close() error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	if mc.manager != nil {
		err := mc.manager.Close()
		mc.manager = nil
		mc.status = StatusFailed
		return err
	}
	return nil
}

// String methods for ConnectionStatus
func (cs ConnectionStatus) String() string {
	switch cs {
	case StatusHealthy:
		return "healthy"
	case StatusDegraded:
		return "degraded"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusRecovering:
		return "recovering"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Global health monitor instance
var globalHealthMonitor *HealthMonitor
var globalHealthMonitorOnce sync.Once

// GetHealthMonitor returns the global health monitor instance
func GetHealthMonitor() *HealthMonitor {
	globalHealthMonitorOnce.Do(func() {
		logger.Debug("Initializing global SSH health monitor")
		globalHealthMonitor = NewHealthMonitor()
		globalHealthMonitor.Start()
	})
	return globalHealthMonitor
}

// HealthCheckConfig represents configuration for health checks
type HealthCheckConfig struct {
	Interval            time.Duration `json:"interval"`
	Timeout             time.Duration `json:"timeout"`
	MaxConsecutiveFails int           `json:"max_consecutive_fails"`
	RecoveryRetries     int           `json:"recovery_retries"`
	EnableAutoRecovery  bool          `json:"enable_auto_recovery"`
}

// DefaultHealthCheckConfig returns default health check configuration
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:            30 * time.Second,
		Timeout:             10 * time.Second,
		MaxConsecutiveFails: 3,
		RecoveryRetries:     2,
		EnableAutoRecovery:  true,
	}
}

// PerformQuickHealthCheck performs a quick health check without full monitoring overhead
func PerformQuickHealthCheck(server *models.Server, asRoot bool) (*HealthCheckResult, error) {
	logger.WithFields(map[string]interface{}{
		"host":    server.Host,
		"port":    server.Port,
		"as_root": asRoot,
	}).Debug("Performing quick SSH health check")

	start := time.Now()

	// Try to create a temporary connection
	manager, err := NewSSHManager(server, asRoot)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"host":    server.Host,
			"port":    server.Port,
			"as_root": asRoot,
		}).WithError(err).Error("Quick health check failed - could not create SSH manager")
		return &HealthCheckResult{
			Status:    StatusFailed,
			Error:     fmt.Errorf("failed to create SSH manager: %w", err),
			Timestamp: start,
		}, err
	}
	defer manager.Close()

	// Test the connection
	testErr := manager.TestConnection()
	responseTime := time.Since(start)

	result := &HealthCheckResult{
		ResponseTime: responseTime,
		Timestamp:    start,
	}

	if testErr != nil {
		result.Status = StatusUnhealthy
		result.Error = testErr
		logger.WithFields(map[string]interface{}{
			"host":          server.Host,
			"port":          server.Port,
			"as_root":       asRoot,
			"response_time": responseTime.String(),
		}).WithError(testErr).Debug("Quick health check failed")
	} else {
		result.Status = StatusHealthy
		logger.WithFields(map[string]interface{}{
			"host":          server.Host,
			"port":          server.Port,
			"as_root":       asRoot,
			"response_time": responseTime.String(),
		}).Debug("Quick health check successful")
	}

	return result, testErr
}
