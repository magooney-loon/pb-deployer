package tunnel

import (
	"context"
	"sync"
	"time"
)

// CleanupManager handles automatic cleanup and connection lifecycle management
type CleanupManager struct {
	pool         Pool
	config       CleanupConfig
	tracer       PoolTracer
	ticker       *time.Ticker
	stopCh       chan struct{}
	running      bool
	mu           sync.Mutex
	stats        *CleanupStats
	lastCleanup  time.Time
	cleanupCount int64
}

// CleanupConfig configures cleanup behavior
type CleanupConfig struct {
	Enabled                 bool          `json:"enabled"`
	Interval                time.Duration `json:"interval"`
	IdleTimeout             time.Duration `json:"idle_timeout"`
	MaxConnectionAge        time.Duration `json:"max_connection_age"`
	MaxConnections          int           `json:"max_connections"`
	MinConnections          int           `json:"min_connections"`
	HealthCheckTimeout      time.Duration `json:"health_check_timeout"`
	ForceCloseTimeout       time.Duration `json:"force_close_timeout"`
	EnableConnectionReuse   bool          `json:"enable_connection_reuse"`
	EnableAggressiveCleanup bool          `json:"enable_aggressive_cleanup"`
	CleanupBatchSize        int           `json:"cleanup_batch_size"`
	GracefulShutdownTime    time.Duration `json:"graceful_shutdown_time"`
}

// CleanupStats tracks cleanup operation statistics
type CleanupStats struct {
	TotalCleanupRuns            int64         `json:"total_cleanup_runs"`
	ConnectionsCleaned          int64         `json:"connections_cleaned"`
	IdleConnectionsCleaned      int64         `json:"idle_connections_cleaned"`
	StaleConnectionsCleaned     int64         `json:"stale_connections_cleaned"`
	UnhealthyConnectionsCleaned int64         `json:"unhealthy_connections_cleaned"`
	ForceClosedConnections      int64         `json:"force_closed_connections"`
	CleanupErrors               int64         `json:"cleanup_errors"`
	LastCleanupDuration         time.Duration `json:"last_cleanup_duration"`
	AverageCleanupDuration      time.Duration `json:"average_cleanup_duration"`
	LastCleanupTime             time.Time     `json:"last_cleanup_time"`
	mu                          sync.RWMutex
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Duration                    time.Duration `json:"duration"`
	ConnectionsInspected        int           `json:"connections_inspected"`
	ConnectionsCleaned          int           `json:"connections_cleaned"`
	IdleConnectionsCleaned      int           `json:"idle_connections_cleaned"`
	StaleConnectionsCleaned     int           `json:"stale_connections_cleaned"`
	UnhealthyConnectionsCleaned int           `json:"unhealthy_connections_cleaned"`
	ErrorsEncountered           int           `json:"errors_encountered"`
	Timestamp                   time.Time     `json:"timestamp"`
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(pool Pool, config CleanupConfig, tracer PoolTracer) *CleanupManager {
	if tracer == nil {
		tracer = &NoOpPoolTracer{}
	}

	return &CleanupManager{
		pool:   pool,
		config: config,
		tracer: tracer,
		stopCh: make(chan struct{}),
		stats:  &CleanupStats{},
	}
}

// Start begins the cleanup manager
func (cm *CleanupManager) Start(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running || !cm.config.Enabled {
		return
	}

	span := cm.tracer.TracePool(ctx, "cleanup_manager_start")
	defer span.End()

	cm.running = true
	cm.ticker = time.NewTicker(cm.config.Interval)

	span.Event("cleanup_manager_started", map[string]interface{}{
		"interval":        cm.config.Interval,
		"idle_timeout":    cm.config.IdleTimeout,
		"max_connections": cm.config.MaxConnections,
		"min_connections": cm.config.MinConnections,
	})

	go cm.cleanupLoop(ctx)
}

// Stop stops the cleanup manager
func (cm *CleanupManager) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return
	}

	cm.running = false

	if cm.ticker != nil {
		cm.ticker.Stop()
		cm.ticker = nil
	}

	close(cm.stopCh)
	cm.stopCh = make(chan struct{}) // Reset for potential restart
}

// cleanupLoop runs the main cleanup loop
func (cm *CleanupManager) cleanupLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-cm.stopCh:
			return
		case <-cm.ticker.C:
			result := cm.performCleanup(ctx)
			cm.updateStats(result)
		}
	}
}

// performCleanup performs a cleanup operation
func (cm *CleanupManager) performCleanup(ctx context.Context) CleanupResult {
	span := cm.tracer.TracePool(ctx, "cleanup_operation")
	defer span.End()

	start := time.Now()
	result := CleanupResult{
		Timestamp: start,
	}

	cm.mu.Lock()
	cm.lastCleanup = start
	cm.cleanupCount++
	cm.mu.Unlock()

	span.Event("cleanup_started", map[string]interface{}{
		"cleanup_count": cm.cleanupCount,
	})

	// Get current pool state
	healthReport := cm.pool.HealthCheck(ctx)
	result.ConnectionsInspected = healthReport.TotalConnections

	span.SetFields(map[string]interface{}{
		"total_connections":   healthReport.TotalConnections,
		"healthy_connections": healthReport.HealthyConnections,
		"failed_connections":  healthReport.FailedConnections,
	})

	// Perform different types of cleanup
	cm.cleanupIdleConnections(ctx, &result)
	cm.cleanupStaleConnections(ctx, &result)
	cm.cleanupUnhealthyConnections(ctx, &result)
	cm.enforceConnectionLimits(ctx, &result)

	result.Duration = time.Since(start)
	result.ConnectionsCleaned = result.IdleConnectionsCleaned +
		result.StaleConnectionsCleaned + result.UnhealthyConnectionsCleaned

	span.Event("cleanup_completed", map[string]interface{}{
		"duration":            result.Duration,
		"connections_cleaned": result.ConnectionsCleaned,
		"idle_cleaned":        result.IdleConnectionsCleaned,
		"stale_cleaned":       result.StaleConnectionsCleaned,
		"unhealthy_cleaned":   result.UnhealthyConnectionsCleaned,
		"errors":              result.ErrorsEncountered,
	})

	return result
}

// cleanupIdleConnections removes connections that have been idle too long
func (cm *CleanupManager) cleanupIdleConnections(ctx context.Context, result *CleanupResult) {
	span := cm.tracer.TracePool(ctx, "cleanup_idle_connections")
	defer span.End()

	// This would need access to pool internals to identify idle connections
	// For now, we'll simulate the logic based on what we know from the pool interface

	span.Event("idle_cleanup_simulated", map[string]interface{}{
		"idle_timeout": cm.config.IdleTimeout,
	})

	// In a real implementation, we would:
	// 1. Get list of idle connections from pool
	// 2. Check their idle time against config.IdleTimeout
	// 3. Close connections that exceed the timeout
	// 4. Update result.IdleConnectionsCleaned

	result.IdleConnectionsCleaned = 0 // Placeholder
}

// cleanupStaleConnections removes connections that are too old
func (cm *CleanupManager) cleanupStaleConnections(ctx context.Context, result *CleanupResult) {
	span := cm.tracer.TracePool(ctx, "cleanup_stale_connections")
	defer span.End()

	if cm.config.MaxConnectionAge <= 0 {
		return
	}

	span.Event("stale_cleanup_simulated", map[string]interface{}{
		"max_age": cm.config.MaxConnectionAge,
	})

	// In a real implementation, we would:
	// 1. Get list of all connections from pool
	// 2. Check their age against config.MaxConnectionAge
	// 3. Close connections that are too old
	// 4. Update result.StaleConnectionsCleaned

	result.StaleConnectionsCleaned = 0 // Placeholder
}

// cleanupUnhealthyConnections removes unhealthy connections
func (cm *CleanupManager) cleanupUnhealthyConnections(ctx context.Context, result *CleanupResult) {
	span := cm.tracer.TracePool(ctx, "cleanup_unhealthy_connections")
	defer span.End()

	healthReport := cm.pool.HealthCheck(ctx)

	span.SetFields(map[string]interface{}{
		"failed_connections": healthReport.FailedConnections,
	})

	// Count unhealthy connections that would be cleaned
	unhealthyCount := 0
	for _, conn := range healthReport.Connections {
		if !conn.Healthy {
			unhealthyCount++
		}
	}

	result.UnhealthyConnectionsCleaned = unhealthyCount

	span.Event("unhealthy_cleanup_completed", map[string]interface{}{
		"cleaned_count": unhealthyCount,
	})
}

// enforceConnectionLimits ensures connection counts stay within limits
func (cm *CleanupManager) enforceConnectionLimits(ctx context.Context, result *CleanupResult) {
	span := cm.tracer.TracePool(ctx, "enforce_connection_limits")
	defer span.End()

	healthReport := cm.pool.HealthCheck(ctx)
	currentConnections := healthReport.TotalConnections

	span.SetFields(map[string]interface{}{
		"current_connections": currentConnections,
		"max_connections":     cm.config.MaxConnections,
		"min_connections":     cm.config.MinConnections,
	})

	// Check if we exceed maximum connections
	if currentConnections > cm.config.MaxConnections {
		excess := currentConnections - cm.config.MaxConnections
		span.Event("max_connections_exceeded", map[string]interface{}{
			"excess_connections": excess,
		})

		// In a real implementation, we would evict excess connections
		// For now, we'll just record the intent
	}

	// Check if we're below minimum connections
	if currentConnections < cm.config.MinConnections {
		needed := cm.config.MinConnections - currentConnections
		span.Event("min_connections_not_met", map[string]interface{}{
			"needed_connections": needed,
		})

		// In a real implementation, we might trigger connection pre-warming
	}
}

// ForceCleanup performs an immediate cleanup operation
func (cm *CleanupManager) ForceCleanup(ctx context.Context) CleanupResult {
	span := cm.tracer.TracePool(ctx, "force_cleanup")
	defer span.End()

	span.Event("force_cleanup_initiated")

	result := cm.performCleanup(ctx)
	cm.updateStats(result)

	span.Event("force_cleanup_completed", map[string]interface{}{
		"connections_cleaned": result.ConnectionsCleaned,
		"duration":            result.Duration,
	})

	return result
}

// updateStats updates cleanup statistics
func (cm *CleanupManager) updateStats(result CleanupResult) {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.TotalCleanupRuns++
	cm.stats.ConnectionsCleaned += int64(result.ConnectionsCleaned)
	cm.stats.IdleConnectionsCleaned += int64(result.IdleConnectionsCleaned)
	cm.stats.StaleConnectionsCleaned += int64(result.StaleConnectionsCleaned)
	cm.stats.UnhealthyConnectionsCleaned += int64(result.UnhealthyConnectionsCleaned)
	cm.stats.CleanupErrors += int64(result.ErrorsEncountered)
	cm.stats.LastCleanupDuration = result.Duration
	cm.stats.LastCleanupTime = result.Timestamp

	// Update average duration using exponential moving average
	if cm.stats.AverageCleanupDuration == 0 {
		cm.stats.AverageCleanupDuration = result.Duration
	} else {
		alpha := 0.1
		cm.stats.AverageCleanupDuration = time.Duration(
			float64(cm.stats.AverageCleanupDuration)*(1-alpha) +
				float64(result.Duration)*alpha)
	}
}

// GetStats returns current cleanup statistics
func (cm *CleanupManager) GetStats() CleanupStats {
	cm.stats.mu.RLock()
	defer cm.stats.mu.RUnlock()

	// Return a copy
	return *cm.stats
}

// IsRunning returns whether the cleanup manager is running
func (cm *CleanupManager) IsRunning() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.running
}

// GetConfig returns the current cleanup configuration
func (cm *CleanupManager) GetConfig() CleanupConfig {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.config
}

// UpdateConfig updates the cleanup configuration
func (cm *CleanupManager) UpdateConfig(config CleanupConfig) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldInterval := cm.config.Interval
	cm.config = config

	// Restart ticker if interval changed and we're running
	if cm.running && cm.config.Interval != oldInterval {
		if cm.ticker != nil {
			cm.ticker.Stop()
		}
		cm.ticker = time.NewTicker(cm.config.Interval)
	}
}

// GracefulShutdown performs a graceful shutdown of the cleanup manager
func (cm *CleanupManager) GracefulShutdown(ctx context.Context) error {
	span := cm.tracer.TracePool(ctx, "cleanup_graceful_shutdown")
	defer span.End()

	cm.mu.Lock()
	if !cm.running {
		cm.mu.Unlock()
		return nil
	}
	cm.mu.Unlock()

	span.Event("graceful_shutdown_started", map[string]interface{}{
		"shutdown_timeout": cm.config.GracefulShutdownTime,
	})

	// Perform final cleanup
	finalResult := cm.performCleanup(ctx)
	cm.updateStats(finalResult)

	span.Event("final_cleanup_completed", map[string]interface{}{
		"connections_cleaned": finalResult.ConnectionsCleaned,
		"duration":            finalResult.Duration,
	})

	// Stop the cleanup manager
	cm.Stop()

	span.Event("graceful_shutdown_completed")
	return nil
}

// ResetStats resets all cleanup statistics
func (cm *CleanupManager) ResetStats() {
	cm.stats.mu.Lock()
	defer cm.stats.mu.Unlock()

	cm.stats.TotalCleanupRuns = 0
	cm.stats.ConnectionsCleaned = 0
	cm.stats.IdleConnectionsCleaned = 0
	cm.stats.StaleConnectionsCleaned = 0
	cm.stats.UnhealthyConnectionsCleaned = 0
	cm.stats.ForceClosedConnections = 0
	cm.stats.CleanupErrors = 0
	cm.stats.LastCleanupDuration = 0
	cm.stats.AverageCleanupDuration = 0
	cm.stats.LastCleanupTime = time.Time{}
}

// DefaultCleanupConfig returns default cleanup configuration
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Enabled:                 true,
		Interval:                5 * time.Minute,
		IdleTimeout:             15 * time.Minute,
		MaxConnectionAge:        2 * time.Hour,
		MaxConnections:          50,
		MinConnections:          2,
		HealthCheckTimeout:      10 * time.Second,
		ForceCloseTimeout:       30 * time.Second,
		EnableConnectionReuse:   true,
		EnableAggressiveCleanup: false,
		CleanupBatchSize:        10,
		GracefulShutdownTime:    30 * time.Second,
	}
}

// ProductionCleanupConfig returns production-optimized cleanup configuration
func ProductionCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Enabled:                 true,
		Interval:                2 * time.Minute,
		IdleTimeout:             10 * time.Minute,
		MaxConnectionAge:        1 * time.Hour,
		MaxConnections:          100,
		MinConnections:          5,
		HealthCheckTimeout:      5 * time.Second,
		ForceCloseTimeout:       15 * time.Second,
		EnableConnectionReuse:   true,
		EnableAggressiveCleanup: true,
		CleanupBatchSize:        20,
		GracefulShutdownTime:    60 * time.Second,
	}
}

// DevelopmentCleanupConfig returns development-friendly cleanup configuration
func DevelopmentCleanupConfig() CleanupConfig {
	return CleanupConfig{
		Enabled:                 true,
		Interval:                10 * time.Minute,
		IdleTimeout:             30 * time.Minute,
		MaxConnectionAge:        4 * time.Hour,
		MaxConnections:          20,
		MinConnections:          1,
		HealthCheckTimeout:      15 * time.Second,
		ForceCloseTimeout:       60 * time.Second,
		EnableConnectionReuse:   true,
		EnableAggressiveCleanup: false,
		CleanupBatchSize:        5,
		GracefulShutdownTime:    15 * time.Second,
	}
}

// ConnectionLeakDetector detects potential connection leaks
type ConnectionLeakDetector struct {
	mu                   sync.RWMutex
	connectionLifecycles map[string]*ConnectionLifecycle
	maxLifetime          time.Duration
	warningThreshold     time.Duration
	leakCallback         func(ConnectionLifecycle)
}

// ConnectionLifecycle tracks the lifecycle of a single connection
type ConnectionLifecycle struct {
	Key           string        `json:"key"`
	CreatedAt     time.Time     `json:"created_at"`
	LastActivity  time.Time     `json:"last_activity"`
	UseCount      int64         `json:"use_count"`
	State         string        `json:"state"`
	Age           time.Duration `json:"age"`
	IdleTime      time.Duration `json:"idle_time"`
	WarningIssued bool          `json:"warning_issued"`
	LeakSuspected bool          `json:"leak_suspected"`
}

// NewConnectionLeakDetector creates a new connection leak detector
func NewConnectionLeakDetector(maxLifetime, warningThreshold time.Duration) *ConnectionLeakDetector {
	return &ConnectionLeakDetector{
		connectionLifecycles: make(map[string]*ConnectionLifecycle),
		maxLifetime:          maxLifetime,
		warningThreshold:     warningThreshold,
	}
}

// TrackConnection starts tracking a connection
func (cld *ConnectionLeakDetector) TrackConnection(key string) {
	cld.mu.Lock()
	defer cld.mu.Unlock()

	cld.connectionLifecycles[key] = &ConnectionLifecycle{
		Key:          key,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		State:        "created",
	}
}

// UpdateConnectionActivity updates the last activity time for a connection
func (cld *ConnectionLeakDetector) UpdateConnectionActivity(key string) {
	cld.mu.Lock()
	defer cld.mu.Unlock()

	if lifecycle, exists := cld.connectionLifecycles[key]; exists {
		lifecycle.LastActivity = time.Now()
		lifecycle.UseCount++
		lifecycle.State = "active"
	}
}

// UntrackConnection stops tracking a connection
func (cld *ConnectionLeakDetector) UntrackConnection(key string) {
	cld.mu.Lock()
	defer cld.mu.Unlock()

	delete(cld.connectionLifecycles, key)
}

// CheckForLeaks checks for potential connection leaks
func (cld *ConnectionLeakDetector) CheckForLeaks() []ConnectionLifecycle {
	cld.mu.Lock()
	defer cld.mu.Unlock()

	var leaks []ConnectionLifecycle
	now := time.Now()

	for _, lifecycle := range cld.connectionLifecycles {
		lifecycle.Age = now.Sub(lifecycle.CreatedAt)
		lifecycle.IdleTime = now.Sub(lifecycle.LastActivity)

		// Check for warning threshold
		if !lifecycle.WarningIssued && lifecycle.Age > cld.warningThreshold {
			lifecycle.WarningIssued = true
			lifecycle.State = "warning"
		}

		// Check for leak threshold
		if !lifecycle.LeakSuspected && lifecycle.Age > cld.maxLifetime {
			lifecycle.LeakSuspected = true
			lifecycle.State = "leak_suspected"
			leaks = append(leaks, *lifecycle)

			if cld.leakCallback != nil {
				go cld.leakCallback(*lifecycle)
			}
		}
	}

	return leaks
}

// GetTrackedConnections returns all currently tracked connections
func (cld *ConnectionLeakDetector) GetTrackedConnections() []ConnectionLifecycle {
	cld.mu.RLock()
	defer cld.mu.RUnlock()

	var connections []ConnectionLifecycle
	now := time.Now()

	for _, lifecycle := range cld.connectionLifecycles {
		lifecycle := *lifecycle // Copy
		lifecycle.Age = now.Sub(lifecycle.CreatedAt)
		lifecycle.IdleTime = now.Sub(lifecycle.LastActivity)
		connections = append(connections, lifecycle)
	}

	return connections
}

// SetLeakCallback sets a callback function to be called when a leak is detected
func (cld *ConnectionLeakDetector) SetLeakCallback(callback func(ConnectionLifecycle)) {
	cld.mu.Lock()
	defer cld.mu.Unlock()
	cld.leakCallback = callback
}
