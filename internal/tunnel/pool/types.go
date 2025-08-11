package pool

import (
	"sync"
	"time"

	"pb-deployer/internal/tunnel"
)

// ConnectionMetadata holds metadata about a connection
type ConnectionMetadata struct {
	Key          string
	CreatedAt    time.Time
	LastUsed     time.Time
	UseCount     int64
	Healthy      bool
	State        tunnel.ConnectionState
	LastError    error
	ResponseTime time.Duration
	mu           sync.RWMutex
}

// UpdateLastUsed updates the last used timestamp and increments use count
func (cm *ConnectionMetadata) UpdateLastUsed() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.LastUsed = time.Now()
	cm.UseCount++
}

// SetHealthy updates the health status
func (cm *ConnectionMetadata) SetHealthy(healthy bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.Healthy = healthy
}

// SetState updates the connection state
func (cm *ConnectionMetadata) SetState(state tunnel.ConnectionState) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.State = state
}

// GetStats returns a copy of the metadata stats
func (cm *ConnectionMetadata) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return map[string]interface{}{
		"key":           cm.Key,
		"created_at":    cm.CreatedAt,
		"last_used":     cm.LastUsed,
		"use_count":     cm.UseCount,
		"healthy":       cm.Healthy,
		"state":         cm.State.String(),
		"response_time": cm.ResponseTime,
	}
}

// Entry represents a connection pool entry
type Entry struct {
	Client   tunnel.SSHClient
	Metadata *ConnectionMetadata
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	TotalConnections    int64
	ActiveConnections   int64
	FailedConnections   int64
	TotalCommands       int64
	FailedCommands      int64
	AverageResponseTime time.Duration
	LastConnectionTime  time.Time
	LastCommandTime     time.Time
	UptimeStart         time.Time
	mu                  sync.RWMutex
}

// IncrementConnections increments connection counters
func (cs *ConnectionStats) IncrementConnections(success bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.TotalConnections++
	if success {
		cs.ActiveConnections++
	} else {
		cs.FailedConnections++
	}
	cs.LastConnectionTime = time.Now()
}

// IncrementCommands increments command counters
func (cs *ConnectionStats) IncrementCommands(success bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.TotalCommands++
	if !success {
		cs.FailedCommands++
	}
	cs.LastCommandTime = time.Now()
}

// GetSnapshot returns a snapshot of current stats
func (cs *ConnectionStats) GetSnapshot() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	uptime := time.Since(cs.UptimeStart)

	return map[string]interface{}{
		"total_connections":     cs.TotalConnections,
		"active_connections":    cs.ActiveConnections,
		"failed_connections":    cs.FailedConnections,
		"total_commands":        cs.TotalCommands,
		"failed_commands":       cs.FailedCommands,
		"average_response_time": cs.AverageResponseTime,
		"last_connection":       cs.LastConnectionTime,
		"last_command":          cs.LastCommandTime,
		"uptime":                uptime,
		"success_rate":          cs.calculateSuccessRate(),
	}
}

func (cs *ConnectionStats) calculateSuccessRate() float64 {
	if cs.TotalConnections == 0 {
		return 0
	}
	return float64(cs.TotalConnections-cs.FailedConnections) / float64(cs.TotalConnections) * 100
}

// Event represents a pool-specific event
type Event struct {
	Type      EventType
	Timestamp time.Time
	PoolKey   string
	ConnKey   string
	Message   string
	Data      map[string]interface{}
	Error     error
}

// EventType represents the type of pool event
type EventType int

const (
	EventPoolCreated EventType = iota
	EventPoolClosed
	EventPoolExhausted
	EventConnectionAcquired
	EventConnectionReleased
	EventConnectionEvicted
	EventConnectionHealthy
	EventConnectionUnhealthy
	EventCleanupStarted
	EventCleanupCompleted
)

func (et EventType) String() string {
	switch et {
	case EventPoolCreated:
		return "pool_created"
	case EventPoolClosed:
		return "pool_closed"
	case EventPoolExhausted:
		return "pool_exhausted"
	case EventConnectionAcquired:
		return "connection_acquired"
	case EventConnectionReleased:
		return "connection_released"
	case EventConnectionEvicted:
		return "connection_evicted"
	case EventConnectionHealthy:
		return "connection_healthy"
	case EventConnectionUnhealthy:
		return "connection_unhealthy"
	case EventCleanupStarted:
		return "cleanup_started"
	case EventCleanupCompleted:
		return "cleanup_completed"
	default:
		return "unknown"
	}
}

// EventHandler handles pool events
type EventHandler interface {
	HandlePoolEvent(event Event)
}

// EventBus distributes pool events to handlers
type EventBus struct {
	handlers []EventHandler
	mu       sync.RWMutex
}

// Subscribe adds an event handler
func (eb *EventBus) Subscribe(handler EventHandler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = append(eb.handlers, handler)
}

// Publish sends an event to all handlers
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	handlers := eb.handlers
	eb.mu.RUnlock()

	for _, handler := range handlers {
		go handler.HandlePoolEvent(event)
	}
}

// Metrics holds pool performance metrics
type Metrics struct {
	AcquisitionTime time.Duration
	UtilizationRate float64
	ErrorRate       float64
	ConnectionReuse float64
	HealthScore     float64
	LastUpdated     time.Time
}

// PoolState represents the current state of the pool
type PoolState struct {
	Size            int
	ActiveCount     int
	IdleCount       int
	UnhealthyCount  int
	Stats           *ConnectionStats
	Metrics         *Metrics
	LastCleanup     time.Time
	LastHealthCheck time.Time
}

// GetHealthScore calculates a health score for the pool (0-100)
func (ps *PoolState) GetHealthScore() float64 {
	if ps.Size == 0 {
		return 100.0 // Empty pool is considered healthy
	}

	healthyCount := ps.Size - ps.UnhealthyCount
	utilizationPenalty := 0.0

	// Penalize if utilization is too high (> 80%)
	utilization := float64(ps.ActiveCount) / float64(ps.Size)
	if utilization > 0.8 {
		utilizationPenalty = (utilization - 0.8) * 50 // Max 10 point penalty
	}

	baseScore := (float64(healthyCount) / float64(ps.Size)) * 100
	return baseScore - utilizationPenalty
}

// RateLimiter provides rate limiting for pool operations
type RateLimiter struct {
	rate     int // operations per interval
	interval time.Duration
	tokens   chan struct{}
	stop     chan struct{}
}

// NewRateLimiter creates a new rate limiter for pool operations
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:     rate,
		interval: interval,
		tokens:   make(chan struct{}, rate),
		stop:     make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token refill goroutine
	go rl.refill()

	return rl
}

// refill periodically adds tokens
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.interval / time.Duration(rl.rate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			select {
			case rl.tokens <- struct{}{}:
				// Token added
			default:
				// Bucket full
			}
		case <-rl.stop:
			return
		}
	}
}

// Allow checks if an operation is allowed
func (rl *RateLimiter) Allow() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Wait blocks until an operation is allowed
func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}
