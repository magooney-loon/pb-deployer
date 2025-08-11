package tunnel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// connectionPool implements the Pool interface with dependency injection
type connectionPool struct {
	factory       ConnectionFactory
	healthMonitor *PoolHealthMonitor
	tracer        PoolTracer
	config        PoolConfig
	entries       map[string]*poolEntry
	mu            sync.RWMutex
	closed        bool
	stats         *PoolStats
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
	cleanupWg     sync.WaitGroup
}

// poolEntry represents a single connection in the pool
type poolEntry struct {
	client          SSHClient
	connectionKey   string
	state           EntryState
	createdAt       time.Time
	lastUsed        time.Time
	lastHealthCheck time.Time
	useCount        int64
	healthFailures  int
	inUse           bool
	metadata        map[string]interface{}
	mu              sync.RWMutex
}

// EntryState represents the state of a pool entry
type EntryState int

const (
	EntryStateIdle EntryState = iota
	EntryStateActive
	EntryStateUnhealthy
	EntryStateClosed
)

func (es EntryState) String() string {
	switch es {
	case EntryStateIdle:
		return "idle"
	case EntryStateActive:
		return "active"
	case EntryStateUnhealthy:
		return "unhealthy"
	case EntryStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// PoolTracer interface for tracing pool operations
type PoolTracer interface {
	TracePool(ctx context.Context, operation string) Span
	TracePoolGet(ctx context.Context, key string) Span
	TracePoolRelease(ctx context.Context, key string) Span
	TracePoolHealth(ctx context.Context) Span
}

// PoolStats tracks pool statistics
type PoolStats struct {
	TotalConnections     int64
	ActiveConnections    int64
	IdleConnections      int64
	HealthyConnections   int64
	UnhealthyConnections int64
	ConnectionsCreated   int64
	ConnectionsClosed    int64
	CacheHits            int64
	CacheMisses          int64
	TotalGetRequests     int64
	TotalReleaseRequests int64
	AverageResponseTime  time.Duration
	mu                   sync.RWMutex
}

// NewPool creates a new connection pool with dependency injection
func NewPool(factory ConnectionFactory, config PoolConfig, tracer PoolTracer) Pool {
	if tracer == nil {
		tracer = &NoOpPoolTracer{}
	}

	pool := &connectionPool{
		factory:     factory,
		tracer:      tracer,
		config:      config,
		entries:     make(map[string]*poolEntry),
		stats:       &PoolStats{},
		cleanupStop: make(chan struct{}),
	}

	// Set default configuration values
	pool.setDefaults()

	// Create health monitor with a no-op SSH tracer for now
	pool.healthMonitor = NewPoolHealthMonitor(&NoOpTracer{})

	// Start cleanup routine
	pool.startCleanup()

	return pool
}

// NewPoolWithHealthMonitor creates a pool with an existing health monitor
func NewPoolWithHealthMonitor(factory ConnectionFactory, config PoolConfig,
	healthMonitor *PoolHealthMonitor, tracer PoolTracer) Pool {

	if tracer == nil {
		tracer = &NoOpPoolTracer{}
	}

	pool := &connectionPool{
		factory:       factory,
		healthMonitor: healthMonitor,
		tracer:        tracer,
		config:        config,
		entries:       make(map[string]*poolEntry),
		stats:         &PoolStats{},
		cleanupStop:   make(chan struct{}),
	}

	pool.setDefaults()
	pool.startCleanup()

	return pool
}

// Get retrieves or creates a connection for the given key
func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error) {
	span := p.tracer.TracePoolGet(ctx, key)
	defer span.End()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		err := fmt.Errorf("pool is closed")
		span.EndWithError(err)
		return nil, err
	}

	p.stats.TotalGetRequests++
	p.mu.Unlock()

	span.SetFields(map[string]interface{}{
		"pool.key": key,
	})

	// Try to get existing connection
	if client := p.getExistingConnection(key); client != nil {
		p.recordCacheHit()
		span.Event("cache_hit", map[string]interface{}{
			"key": key,
		})
		return client, nil
	}

	// Create new connection if under limit
	client, err := p.createNewConnection(ctx, key)
	if err != nil {
		span.EndWithError(err)
		return nil, err
	}

	p.recordCacheMiss()
	span.Event("connection_created", map[string]interface{}{
		"key": key,
	})

	return client, nil
}

// Release returns a connection to the pool
func (p *connectionPool) Release(key string, client SSHClient) {
	span := p.tracer.TracePoolRelease(context.Background(), key)
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.stats.TotalReleaseRequests++

	span.SetFields(map[string]interface{}{
		"pool.key": key,
	})

	entry, exists := p.entries[key]
	if !exists {
		span.Event("entry_not_found", map[string]interface{}{
			"key": key,
		})
		// Connection not in pool, close it
		client.Close()
		return
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Verify this is the same client
	if entry.client != client {
		span.Event("client_mismatch", map[string]interface{}{
			"key": key,
		})
		// Different client, close the passed one
		client.Close()
		return
	}

	// Mark as not in use and update metadata
	entry.inUse = false
	entry.lastUsed = time.Now()
	entry.state = EntryStateIdle

	span.Event("connection_released", map[string]interface{}{
		"key":       key,
		"use_count": entry.useCount,
		"last_used": entry.lastUsed,
	})
}

// Close closes the pool and all connections
func (p *connectionPool) Close() error {
	span := p.tracer.TracePool(context.Background(), "close")
	defer span.End()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}

	p.closed = true

	// Stop cleanup routine
	close(p.cleanupStop)
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	// Close all connections
	var errors []error
	for key, entry := range p.entries {
		entry.mu.Lock()
		if entry.client != nil {
			if err := entry.client.Close(); err != nil {
				errors = append(errors, fmt.Errorf("failed to close connection %s: %w", key, err))
			}
			entry.state = EntryStateClosed
		}
		entry.mu.Unlock()
	}

	// Clear entries
	p.entries = make(map[string]*poolEntry)
	p.mu.Unlock()

	// Wait for cleanup routine to finish
	p.cleanupWg.Wait()

	// Stop health monitoring
	if p.healthMonitor != nil {
		p.healthMonitor.StopMonitoringAll()
	}

	span.Event("pool_closed", map[string]interface{}{
		"total_connections": len(p.entries),
		"errors":            len(errors),
	})

	if len(errors) > 0 {
		return fmt.Errorf("errors closing connections: %v", errors)
	}

	return nil
}

// HealthCheck performs health check on all connections
func (p *connectionPool) HealthCheck(ctx context.Context) HealthReport {
	span := p.tracer.TracePoolHealth(ctx)
	defer span.End()

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return HealthReport{
			TotalConnections:   0,
			HealthyConnections: 0,
			FailedConnections:  0,
			CheckedAt:          time.Now(),
		}
	}

	entries := make([]*poolEntry, 0, len(p.entries))
	for _, entry := range p.entries {
		entries = append(entries, entry)
	}
	p.mu.RUnlock()

	report := HealthReport{
		TotalConnections: len(entries),
		CheckedAt:        time.Now(),
		Connections:      make([]ConnectionHealth, 0, len(entries)),
	}

	for _, entry := range entries {
		entry.mu.RLock()
		connHealth := ConnectionHealth{
			Key:      entry.connectionKey,
			Healthy:  entry.state != EntryStateUnhealthy && entry.state != EntryStateClosed,
			LastUsed: entry.lastUsed,
			UseCount: entry.useCount,
		}

		if entry.client != nil && connHealth.Healthy {
			// Quick health check
			connHealth.Healthy = entry.client.IsConnected()
		}

		if connHealth.Healthy {
			report.HealthyConnections++
		} else {
			report.FailedConnections++
			connHealth.Error = "connection unhealthy or closed"
		}

		report.Connections = append(report.Connections, connHealth)
		entry.mu.RUnlock()
	}

	span.Event("health_check_completed", map[string]interface{}{
		"total":   report.TotalConnections,
		"healthy": report.HealthyConnections,
		"failed":  report.FailedConnections,
	})

	return report
}

// getExistingConnection tries to get an existing healthy connection
func (p *connectionPool) getExistingConnection(key string) SSHClient {
	p.mu.RLock()
	entry, exists := p.entries[key]
	p.mu.RUnlock()

	if !exists {
		return nil
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if connection is available and healthy
	if entry.inUse || entry.state != EntryStateIdle || entry.client == nil {
		return nil
	}

	// Quick health check
	if !entry.client.IsConnected() {
		entry.state = EntryStateUnhealthy
		return nil
	}

	// Mark as in use
	entry.inUse = true
	entry.lastUsed = time.Now()
	entry.useCount++
	entry.state = EntryStateActive

	return entry.client
}

// createNewConnection creates a new connection and adds it to the pool
func (p *connectionPool) createNewConnection(ctx context.Context, key string) (SSHClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if we're at the connection limit
	if len(p.entries) >= p.config.MaxConnections {
		// Try to evict an idle connection
		if !p.evictIdleConnection() {
			return nil, ErrPoolExhausted
		}
	}

	// Parse connection configuration from key
	config, err := p.parseConnectionKey(key)
	if err != nil {
		return nil, fmt.Errorf("invalid connection key %s: %w", key, err)
	}

	// Create new connection
	client, err := p.factory.Create(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection for %s: %w", key, err)
	}

	// Connect the client
	if err := client.Connect(ctx); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect %s: %w", key, err)
	}

	// Create pool entry
	entry := &poolEntry{
		client:        client,
		connectionKey: key,
		state:         EntryStateActive,
		createdAt:     time.Now(),
		lastUsed:      time.Now(),
		useCount:      1,
		inUse:         true,
		metadata:      make(map[string]interface{}),
	}

	// Add to pool
	p.entries[key] = entry

	// Add to health monitoring
	if p.healthMonitor != nil {
		p.healthMonitor.AddConnection(key, client)
	}

	// Update statistics
	p.stats.ConnectionsCreated++

	return client, nil
}

// evictIdleConnection tries to evict an idle connection to make room
func (p *connectionPool) evictIdleConnection() bool {
	var oldestEntry *poolEntry
	var oldestKey string
	oldestTime := time.Now()

	// Find the oldest idle connection
	for key, entry := range p.entries {
		entry.mu.RLock()
		if !entry.inUse && entry.state == EntryStateIdle {
			if entry.lastUsed.Before(oldestTime) {
				oldestTime = entry.lastUsed
				oldestEntry = entry
				oldestKey = key
			}
		}
		entry.mu.RUnlock()
	}

	if oldestEntry == nil {
		return false // No idle connections to evict
	}

	// Remove from pool
	delete(p.entries, oldestKey)

	// Close the connection
	oldestEntry.mu.Lock()
	if oldestEntry.client != nil {
		oldestEntry.client.Close()
	}
	oldestEntry.state = EntryStateClosed
	oldestEntry.mu.Unlock()

	// Remove from health monitoring
	if p.healthMonitor != nil {
		p.healthMonitor.RemoveConnection(oldestKey)
	}

	p.stats.ConnectionsClosed++
	return true
}

// parseConnectionKey parses a connection key into a ConnectionConfig
func (p *connectionPool) parseConnectionKey(key string) (ConnectionConfig, error) {
	// For now, this is a simple implementation
	// In a real implementation, you might encode more information in the key
	// or maintain a separate mapping

	// Example key format: "user@host:port"
	// This is a simplified parser - in practice you'd want more robust parsing
	return ConnectionConfig{
		Host:     "localhost", // This should be parsed from key
		Port:     22,          // This should be parsed from key
		Username: "root",      // This should be parsed from key
		AuthMethod: AuthMethod{
			Type: "key",
		},
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultMaxRetries,
		HostKeyMode: HostKeyAcceptNew,
	}, nil
}

// setDefaults sets default configuration values
func (p *connectionPool) setDefaults() {
	if p.config.MaxConnections <= 0 {
		p.config.MaxConnections = DefaultMaxConnections
	}
	if p.config.MaxIdleTime <= 0 {
		p.config.MaxIdleTime = DefaultMaxIdleTime
	}
	if p.config.HealthInterval <= 0 {
		p.config.HealthInterval = DefaultHealthCheckInterval
	}
	if p.config.CleanupInterval <= 0 {
		p.config.CleanupInterval = DefaultCleanupInterval
	}
	if p.config.MaxRetries <= 0 {
		p.config.MaxRetries = DefaultMaxRetries
	}
}

// startCleanup starts the background cleanup routine
func (p *connectionPool) startCleanup() {
	p.cleanupTicker = time.NewTicker(p.config.CleanupInterval)
	p.cleanupWg.Add(1)

	go func() {
		defer p.cleanupWg.Done()
		for {
			select {
			case <-p.cleanupTicker.C:
				p.cleanup()
			case <-p.cleanupStop:
				return
			}
		}
	}()
}

// cleanup performs periodic cleanup of idle and unhealthy connections
func (p *connectionPool) cleanup() {
	span := p.tracer.TracePool(context.Background(), "cleanup")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	var toRemove []string
	now := time.Now()
	cleanedCount := 0

	for key, entry := range p.entries {
		entry.mu.RLock()
		shouldRemove := false

		// Remove if idle too long
		if !entry.inUse && entry.state == EntryStateIdle {
			if now.Sub(entry.lastUsed) > p.config.MaxIdleTime {
				shouldRemove = true
			}
		}

		// Remove if unhealthy
		if entry.state == EntryStateUnhealthy || entry.state == EntryStateClosed {
			shouldRemove = true
		}

		entry.mu.RUnlock()

		if shouldRemove {
			toRemove = append(toRemove, key)
		}
	}

	// Remove identified connections
	for _, key := range toRemove {
		entry := p.entries[key]
		delete(p.entries, key)

		entry.mu.Lock()
		if entry.client != nil {
			entry.client.Close()
		}
		entry.state = EntryStateClosed
		entry.mu.Unlock()

		if p.healthMonitor != nil {
			p.healthMonitor.RemoveConnection(key)
		}

		p.stats.ConnectionsClosed++
		cleanedCount++
	}

	span.Event("cleanup_completed", map[string]interface{}{
		"removed_connections":   cleanedCount,
		"remaining_connections": len(p.entries),
	})
}

// recordCacheHit records a cache hit in statistics
func (p *connectionPool) recordCacheHit() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	p.stats.CacheHits++
}

// recordCacheMiss records a cache miss in statistics
func (p *connectionPool) recordCacheMiss() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()
	p.stats.CacheMisses++
}

// GetStats returns a copy of current pool statistics
func (p *connectionPool) GetStats() PoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	// Return a copy without the mutex
	return PoolStats{
		TotalConnections:     p.stats.TotalConnections,
		ActiveConnections:    p.stats.ActiveConnections,
		IdleConnections:      p.stats.IdleConnections,
		HealthyConnections:   p.stats.HealthyConnections,
		UnhealthyConnections: p.stats.UnhealthyConnections,
		ConnectionsCreated:   p.stats.ConnectionsCreated,
		ConnectionsClosed:    p.stats.ConnectionsClosed,
		CacheHits:            p.stats.CacheHits,
		CacheMisses:          p.stats.CacheMisses,
		TotalGetRequests:     p.stats.TotalGetRequests,
		TotalReleaseRequests: p.stats.TotalReleaseRequests,
		AverageResponseTime:  p.stats.AverageResponseTime,
	}
}

// GetConnectionCount returns the current number of connections
func (p *connectionPool) GetConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries)
}

// NoOpPoolTracer provides a no-op implementation for when tracing is disabled
type NoOpPoolTracer struct{}

func (t *NoOpPoolTracer) TracePool(ctx context.Context, operation string) Span {
	return &NoOpSpan{}
}

func (t *NoOpPoolTracer) TracePoolGet(ctx context.Context, key string) Span {
	return &NoOpSpan{}
}

func (t *NoOpPoolTracer) TracePoolRelease(ctx context.Context, key string) Span {
	return &NoOpSpan{}
}

func (t *NoOpPoolTracer) TracePoolHealth(ctx context.Context) Span {
	return &NoOpSpan{}
}
