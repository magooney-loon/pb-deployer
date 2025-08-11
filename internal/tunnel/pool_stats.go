package tunnel

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// PoolStatsCollector manages comprehensive statistics collection for pool operations
type PoolStatsCollector struct {
	stats           *DetailedPoolStats
	historicalStats *HistoricalStats
	mu              sync.RWMutex
	startTime       time.Time
	enabled         bool
}

// DetailedPoolStats contains comprehensive pool statistics
type DetailedPoolStats struct {
	// Connection statistics
	TotalConnections     int64 `json:"total_connections"`
	ActiveConnections    int64 `json:"active_connections"`
	IdleConnections      int64 `json:"idle_connections"`
	HealthyConnections   int64 `json:"healthy_connections"`
	UnhealthyConnections int64 `json:"unhealthy_connections"`

	// Operation statistics
	ConnectionsCreated   int64 `json:"connections_created"`
	ConnectionsClosed    int64 `json:"connections_closed"`
	ConnectionsEvicted   int64 `json:"connections_evicted"`
	ConnectionsRecovered int64 `json:"connections_recovered"`

	// Request statistics
	TotalGetRequests     int64 `json:"total_get_requests"`
	TotalReleaseRequests int64 `json:"total_release_requests"`
	CacheHits            int64 `json:"cache_hits"`
	CacheMisses          int64 `json:"cache_misses"`
	GetRequestsSucceeded int64 `json:"get_requests_succeeded"`
	GetRequestsFailed    int64 `json:"get_requests_failed"`

	// Performance statistics
	AverageGetLatency     time.Duration `json:"average_get_latency"`
	AverageReleaseLatency time.Duration `json:"average_release_latency"`
	AverageResponseTime   time.Duration `json:"average_response_time"`
	MaxResponseTime       time.Duration `json:"max_response_time"`
	MinResponseTime       time.Duration `json:"min_response_time"`

	// Health statistics
	HealthChecksPerformed   int64 `json:"health_checks_performed"`
	HealthChecksFailed      int64 `json:"health_checks_failed"`
	RecoveryAttemptsTotal   int64 `json:"recovery_attempts_total"`
	RecoveryAttemptsSuccess int64 `json:"recovery_attempts_success"`

	// Utilization statistics
	PoolUtilizationPercent float64 `json:"pool_utilization_percent"`
	CacheHitRatio          float64 `json:"cache_hit_ratio"`
	HealthRatio            float64 `json:"health_ratio"`
	SuccessRatio           float64 `json:"success_ratio"`

	// Timing statistics
	UptimeSeconds       float64   `json:"uptime_seconds"`
	LastUpdate          time.Time `json:"last_update"`
	CollectionStartTime time.Time `json:"collection_start_time"`

	mu sync.RWMutex
}

// HistoricalStats tracks statistics over time for trend analysis
type HistoricalStats struct {
	Snapshots        []PoolStatsSnapshot `json:"snapshots"`
	MaxSnapshots     int                 `json:"max_snapshots"`
	SnapshotInterval time.Duration       `json:"snapshot_interval"`
	LastSnapshot     time.Time           `json:"last_snapshot"`
	mu               sync.RWMutex
}

// PoolStatsSnapshot represents a point-in-time snapshot of pool statistics
type PoolStatsSnapshot struct {
	Timestamp           time.Time     `json:"timestamp"`
	TotalConnections    int64         `json:"total_connections"`
	ActiveConnections   int64         `json:"active_connections"`
	HealthyConnections  int64         `json:"healthy_connections"`
	CacheHitRatio       float64       `json:"cache_hit_ratio"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	PoolUtilization     float64       `json:"pool_utilization"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
}

// NewPoolStatsCollector creates a new pool statistics collector
func NewPoolStatsCollector(maxHistoricalSnapshots int, snapshotInterval time.Duration) *PoolStatsCollector {
	now := time.Now()

	return &PoolStatsCollector{
		stats: &DetailedPoolStats{
			CollectionStartTime: now,
			LastUpdate:          now,
			MinResponseTime:     time.Hour, // Initialize to high value
		},
		historicalStats: &HistoricalStats{
			Snapshots:        make([]PoolStatsSnapshot, 0, maxHistoricalSnapshots),
			MaxSnapshots:     maxHistoricalSnapshots,
			SnapshotInterval: snapshotInterval,
		},
		startTime: now,
		enabled:   true,
	}
}

// Enable enables statistics collection
func (psc *PoolStatsCollector) Enable() {
	psc.mu.Lock()
	defer psc.mu.Unlock()
	psc.enabled = true
}

// Disable disables statistics collection
func (psc *PoolStatsCollector) Disable() {
	psc.mu.Lock()
	defer psc.mu.Unlock()
	psc.enabled = false
}

// IsEnabled returns whether statistics collection is enabled
func (psc *PoolStatsCollector) IsEnabled() bool {
	psc.mu.RLock()
	defer psc.mu.RUnlock()
	return psc.enabled
}

// RecordConnectionCreated records a connection creation
func (psc *PoolStatsCollector) RecordConnectionCreated() {
	if !psc.IsEnabled() {
		return
	}

	psc.stats.mu.Lock()
	defer psc.stats.mu.Unlock()

	atomic.AddInt64(&psc.stats.ConnectionsCreated, 1)
	atomic.AddInt64(&psc.stats.TotalConnections, 1)
	psc.stats.LastUpdate = time.Now()
}

// RecordConnectionClosed records a connection closure
func (psc *PoolStatsCollector) RecordConnectionClosed() {
	if !psc.IsEnabled() {
		return
	}

	psc.stats.mu.Lock()
	defer psc.stats.mu.Unlock()

	atomic.AddInt64(&psc.stats.ConnectionsClosed, 1)
	atomic.AddInt64(&psc.stats.TotalConnections, -1)
	psc.stats.LastUpdate = time.Now()
}

// RecordConnectionEvicted records a connection eviction
func (psc *PoolStatsCollector) RecordConnectionEvicted() {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.ConnectionsEvicted, 1)
}

// RecordConnectionRecovered records a connection recovery
func (psc *PoolStatsCollector) RecordConnectionRecovered() {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.ConnectionsRecovered, 1)
}

// RecordGetRequest records a pool get request
func (psc *PoolStatsCollector) RecordGetRequest(succeeded bool, latency time.Duration) {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.TotalGetRequests, 1)

	if succeeded {
		atomic.AddInt64(&psc.stats.GetRequestsSucceeded, 1)
	} else {
		atomic.AddInt64(&psc.stats.GetRequestsFailed, 1)
	}

	psc.updateAverageLatency(&psc.stats.AverageGetLatency, latency)
}

// RecordReleaseRequest records a pool release request
func (psc *PoolStatsCollector) RecordReleaseRequest(latency time.Duration) {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.TotalReleaseRequests, 1)
	psc.updateAverageLatency(&psc.stats.AverageReleaseLatency, latency)
}

// RecordCacheHit records a cache hit
func (psc *PoolStatsCollector) RecordCacheHit() {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.CacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (psc *PoolStatsCollector) RecordCacheMiss() {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.CacheMisses, 1)
}

// RecordHealthCheck records a health check operation
func (psc *PoolStatsCollector) RecordHealthCheck(succeeded bool) {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.HealthChecksPerformed, 1)

	if !succeeded {
		atomic.AddInt64(&psc.stats.HealthChecksFailed, 1)
	}
}

// RecordRecoveryAttempt records a recovery attempt
func (psc *PoolStatsCollector) RecordRecoveryAttempt(succeeded bool) {
	if !psc.IsEnabled() {
		return
	}

	atomic.AddInt64(&psc.stats.RecoveryAttemptsTotal, 1)

	if succeeded {
		atomic.AddInt64(&psc.stats.RecoveryAttemptsSuccess, 1)
	}
}

// RecordResponseTime records a response time measurement
func (psc *PoolStatsCollector) RecordResponseTime(responseTime time.Duration) {
	if !psc.IsEnabled() {
		return
	}

	psc.stats.mu.Lock()
	defer psc.stats.mu.Unlock()

	// Update average response time
	psc.updateAverageLatency(&psc.stats.AverageResponseTime, responseTime)

	// Update min/max response times
	if responseTime > psc.stats.MaxResponseTime {
		psc.stats.MaxResponseTime = responseTime
	}

	if responseTime < psc.stats.MinResponseTime {
		psc.stats.MinResponseTime = responseTime
	}
}

// UpdateConnectionCounts updates current connection counts
func (psc *PoolStatsCollector) UpdateConnectionCounts(total, active, idle, healthy, unhealthy int64) {
	if !psc.IsEnabled() {
		return
	}

	psc.stats.mu.Lock()
	defer psc.stats.mu.Unlock()

	atomic.StoreInt64(&psc.stats.TotalConnections, total)
	atomic.StoreInt64(&psc.stats.ActiveConnections, active)
	atomic.StoreInt64(&psc.stats.IdleConnections, idle)
	atomic.StoreInt64(&psc.stats.HealthyConnections, healthy)
	atomic.StoreInt64(&psc.stats.UnhealthyConnections, unhealthy)

	psc.stats.LastUpdate = time.Now()
}

// updateAverageLatency updates an average latency using exponential moving average
func (psc *PoolStatsCollector) updateAverageLatency(current *time.Duration, newValue time.Duration) {
	psc.stats.mu.Lock()
	defer psc.stats.mu.Unlock()

	// Simple exponential moving average with α = 0.1
	if *current == 0 {
		*current = newValue
	} else {
		*current = time.Duration(float64(*current)*0.9 + float64(newValue)*0.1)
	}
}

// GetCurrentStats returns a snapshot of current statistics
func (psc *PoolStatsCollector) GetCurrentStats() DetailedPoolStats {
	if !psc.IsEnabled() {
		return DetailedPoolStats{}
	}

	psc.stats.mu.RLock()
	defer psc.stats.mu.RUnlock()

	// Calculate derived statistics
	stats := *psc.stats
	psc.calculateDerivedStats(&stats)

	return stats
}

// calculateDerivedStats calculates derived statistics like ratios and percentages
func (psc *PoolStatsCollector) calculateDerivedStats(stats *DetailedPoolStats) {
	// Calculate cache hit ratio
	totalCacheRequests := stats.CacheHits + stats.CacheMisses
	if totalCacheRequests > 0 {
		stats.CacheHitRatio = float64(stats.CacheHits) / float64(totalCacheRequests) * 100
	}

	// Calculate health ratio
	if stats.TotalConnections > 0 {
		stats.HealthRatio = float64(stats.HealthyConnections) / float64(stats.TotalConnections) * 100
	}

	// Calculate success ratio
	totalRequests := stats.GetRequestsSucceeded + stats.GetRequestsFailed
	if totalRequests > 0 {
		stats.SuccessRatio = float64(stats.GetRequestsSucceeded) / float64(totalRequests) * 100
	}

	// Calculate pool utilization
	if stats.TotalConnections > 0 {
		stats.PoolUtilizationPercent = float64(stats.ActiveConnections) / float64(stats.TotalConnections) * 100
	}

	// Calculate uptime
	stats.UptimeSeconds = time.Since(psc.startTime).Seconds()
}

// TakeSnapshot takes a snapshot of current statistics for historical tracking
func (psc *PoolStatsCollector) TakeSnapshot() {
	if !psc.IsEnabled() {
		return
	}

	now := time.Now()
	currentStats := psc.GetCurrentStats()

	// Calculate requests per second
	var requestsPerSecond float64
	if uptime := time.Since(psc.startTime).Seconds(); uptime > 0 {
		requestsPerSecond = float64(currentStats.TotalGetRequests) / uptime
	}

	snapshot := PoolStatsSnapshot{
		Timestamp:           now,
		TotalConnections:    currentStats.TotalConnections,
		ActiveConnections:   currentStats.ActiveConnections,
		HealthyConnections:  currentStats.HealthyConnections,
		CacheHitRatio:       currentStats.CacheHitRatio,
		AverageResponseTime: currentStats.AverageResponseTime,
		PoolUtilization:     currentStats.PoolUtilizationPercent,
		RequestsPerSecond:   requestsPerSecond,
	}

	psc.historicalStats.mu.Lock()
	defer psc.historicalStats.mu.Unlock()

	// Add snapshot
	psc.historicalStats.Snapshots = append(psc.historicalStats.Snapshots, snapshot)

	// Remove oldest snapshots if we exceed the limit
	if len(psc.historicalStats.Snapshots) > psc.historicalStats.MaxSnapshots {
		psc.historicalStats.Snapshots = psc.historicalStats.Snapshots[1:]
	}

	psc.historicalStats.LastSnapshot = now
}

// GetHistoricalStats returns historical statistics
func (psc *PoolStatsCollector) GetHistoricalStats() HistoricalStats {
	psc.historicalStats.mu.RLock()
	defer psc.historicalStats.mu.RUnlock()

	// Return a copy
	snapshots := make([]PoolStatsSnapshot, len(psc.historicalStats.Snapshots))
	copy(snapshots, psc.historicalStats.Snapshots)

	return HistoricalStats{
		Snapshots:        snapshots,
		MaxSnapshots:     psc.historicalStats.MaxSnapshots,
		SnapshotInterval: psc.historicalStats.SnapshotInterval,
		LastSnapshot:     psc.historicalStats.LastSnapshot,
	}
}

// GetStatsJSON returns current statistics as JSON
func (psc *PoolStatsCollector) GetStatsJSON() ([]byte, error) {
	stats := psc.GetCurrentStats()
	return json.MarshalIndent(stats, "", "  ")
}

// GetHistoricalStatsJSON returns historical statistics as JSON
func (psc *PoolStatsCollector) GetHistoricalStatsJSON() ([]byte, error) {
	stats := psc.GetHistoricalStats()
	return json.MarshalIndent(stats, "", "  ")
}

// Reset resets all statistics
func (psc *PoolStatsCollector) Reset() {
	psc.mu.Lock()
	defer psc.mu.Unlock()

	now := time.Now()

	psc.stats = &DetailedPoolStats{
		CollectionStartTime: now,
		LastUpdate:          now,
		MinResponseTime:     time.Hour, // Initialize to high value
	}

	psc.historicalStats.mu.Lock()
	psc.historicalStats.Snapshots = psc.historicalStats.Snapshots[:0]
	psc.historicalStats.LastSnapshot = time.Time{}
	psc.historicalStats.mu.Unlock()

	psc.startTime = now
}

// GetPerformanceMetrics returns key performance metrics
func (psc *PoolStatsCollector) GetPerformanceMetrics() PerformanceMetrics {
	stats := psc.GetCurrentStats()

	return PerformanceMetrics{
		CacheHitRatio:       stats.CacheHitRatio,
		AverageResponseTime: stats.AverageResponseTime,
		PoolUtilization:     stats.PoolUtilizationPercent,
		HealthRatio:         stats.HealthRatio,
		SuccessRatio:        stats.SuccessRatio,
		RequestsPerSecond:   psc.calculateRequestsPerSecond(),
		ConnectionTurnover:  psc.calculateConnectionTurnover(),
	}
}

// PerformanceMetrics contains key performance indicators
type PerformanceMetrics struct {
	CacheHitRatio       float64       `json:"cache_hit_ratio"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	PoolUtilization     float64       `json:"pool_utilization"`
	HealthRatio         float64       `json:"health_ratio"`
	SuccessRatio        float64       `json:"success_ratio"`
	RequestsPerSecond   float64       `json:"requests_per_second"`
	ConnectionTurnover  float64       `json:"connection_turnover"`
}

// calculateRequestsPerSecond calculates current requests per second
func (psc *PoolStatsCollector) calculateRequestsPerSecond() float64 {
	stats := psc.GetCurrentStats()
	uptime := time.Since(psc.startTime).Seconds()

	if uptime == 0 {
		return 0
	}

	return float64(stats.TotalGetRequests) / uptime
}

// calculateConnectionTurnover calculates connection turnover rate
func (psc *PoolStatsCollector) calculateConnectionTurnover() float64 {
	stats := psc.GetCurrentStats()

	if stats.ConnectionsCreated == 0 {
		return 0
	}

	return float64(stats.ConnectionsClosed) / float64(stats.ConnectionsCreated) * 100
}

// GetAlertMetrics returns metrics that might trigger alerts
func (psc *PoolStatsCollector) GetAlertMetrics() AlertMetrics {
	stats := psc.GetCurrentStats()

	return AlertMetrics{
		UnhealthyConnectionCount: stats.UnhealthyConnections,
		FailedRequestRatio:       100 - stats.SuccessRatio,
		AverageResponseTime:      stats.AverageResponseTime,
		CacheHitRatio:            stats.CacheHitRatio,
		PoolUtilization:          stats.PoolUtilizationPercent,
		RecoveryFailureRatio:     psc.calculateRecoveryFailureRatio(),
	}
}

// AlertMetrics contains metrics relevant for alerting
type AlertMetrics struct {
	UnhealthyConnectionCount int64         `json:"unhealthy_connection_count"`
	FailedRequestRatio       float64       `json:"failed_request_ratio"`
	AverageResponseTime      time.Duration `json:"average_response_time"`
	CacheHitRatio            float64       `json:"cache_hit_ratio"`
	PoolUtilization          float64       `json:"pool_utilization"`
	RecoveryFailureRatio     float64       `json:"recovery_failure_ratio"`
}

// calculateRecoveryFailureRatio calculates the recovery failure ratio
func (psc *PoolStatsCollector) calculateRecoveryFailureRatio() float64 {
	stats := psc.GetCurrentStats()

	if stats.RecoveryAttemptsTotal == 0 {
		return 0
	}

	failed := stats.RecoveryAttemptsTotal - stats.RecoveryAttemptsSuccess
	return float64(failed) / float64(stats.RecoveryAttemptsTotal) * 100
}

// StartPeriodicSnapshots starts taking periodic snapshots
func (psc *PoolStatsCollector) StartPeriodicSnapshots() {
	go func() {
		ticker := time.NewTicker(psc.historicalStats.SnapshotInterval)
		defer ticker.Stop()

		for range ticker.C {
			if psc.IsEnabled() {
				psc.TakeSnapshot()
			}
		}
	}()
}

// ExportStats exports statistics to a map for external monitoring systems
func (psc *PoolStatsCollector) ExportStats() map[string]interface{} {
	stats := psc.GetCurrentStats()

	return map[string]interface{}{
		"pool_total_connections":        stats.TotalConnections,
		"pool_active_connections":       stats.ActiveConnections,
		"pool_idle_connections":         stats.IdleConnections,
		"pool_healthy_connections":      stats.HealthyConnections,
		"pool_unhealthy_connections":    stats.UnhealthyConnections,
		"pool_connections_created":      stats.ConnectionsCreated,
		"pool_connections_closed":       stats.ConnectionsClosed,
		"pool_cache_hits":               stats.CacheHits,
		"pool_cache_misses":             stats.CacheMisses,
		"pool_cache_hit_ratio":          stats.CacheHitRatio,
		"pool_average_response_time_ms": float64(stats.AverageResponseTime.Nanoseconds()) / 1e6,
		"pool_utilization_percent":      stats.PoolUtilizationPercent,
		"pool_health_ratio":             stats.HealthRatio,
		"pool_success_ratio":            stats.SuccessRatio,
		"pool_uptime_seconds":           stats.UptimeSeconds,
	}
}
