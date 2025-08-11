package pool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"pb-deployer/internal/tracer"
	"pb-deployer/internal/tunnel"
)

// connectionPool implements the Pool interface with dependency injection
type connectionPool struct {
	factory       tunnel.ConnectionFactory
	tracer        tracer.PoolTracer
	config        tunnel.PoolConfig
	connections   map[string]*poolEntry
	mu            sync.RWMutex
	closed        bool
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// poolEntry represents a connection in the pool
type poolEntry struct {
	client    tunnel.SSHClient
	createdAt time.Time
	lastUsed  time.Time
	useCount  int64
	healthy   bool
	mu        sync.RWMutex
}

// NewPool creates a new connection pool with dependency injection
func NewPool(factory tunnel.ConnectionFactory, config tunnel.PoolConfig, poolTracer tracer.PoolTracer) tunnel.Pool {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid pool config: %v", err))
	}

	pool := &connectionPool{
		factory:     factory,
		tracer:      poolTracer,
		config:      config,
		connections: make(map[string]*poolEntry),
		stopCleanup: make(chan struct{}),
	}

	pool.startCleanup()
	return pool
}

// Get retrieves or creates a connection for the given key
func (p *connectionPool) Get(ctx context.Context, key string) (tunnel.SSHClient, error) {
	span := p.tracer.TraceGet(ctx, key)
	defer span.End()

	span.SetField("pool.key", key)

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		err := fmt.Errorf("pool is closed")
		span.EndWithError(err)
		return nil, err
	}

	// Check for existing healthy connection
	if entry, exists := p.connections[key]; exists {
		entry.mu.RLock()
		isHealthy := entry.healthy
		lastUsed := entry.lastUsed
		entry.mu.RUnlock()

		if isHealthy && time.Since(lastUsed) < p.config.MaxIdleTime {
			// Check if connection is still alive
			if entry.client.IsConnected() {
				entry.mu.Lock()
				entry.lastUsed = time.Now()
				entry.useCount++
				entry.mu.Unlock()

				span.Event("connection_reused", tracer.Fields{
					"pool.total_connections": len(p.connections),
					"entry.use_count":        entry.useCount,
				})
				return entry.client, nil
			} else {
				// Connection is dead, mark as unhealthy
				entry.mu.Lock()
				entry.healthy = false
				entry.mu.Unlock()
			}
		}

		// Remove stale/unhealthy connection
		entry.client.Close()
		delete(p.connections, key)
		span.Event("connection_removed", tracer.String("reason", "stale_or_unhealthy"))
	}

	// Check connection limit and evict if necessary
	if len(p.connections) >= p.config.MaxConnections {
		if err := p.evictOldest(); err != nil {
			span.EndWithError(err)
			return nil, fmt.Errorf("failed to evict connection: %w", err)
		}
		span.Event("connection_evicted", tracer.String("reason", "max_connections_reached"))
	}

	// Parse connection key to get config
	config, err := parseConnectionKey(key)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("invalid connection key: %w", err)
	}

	// Create new connection
	client, err := p.factory.Create(config)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Connect the client
	if err := client.Connect(ctx); err != nil {
		client.Close()
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Store in pool
	now := time.Now()
	p.connections[key] = &poolEntry{
		client:    client,
		createdAt: now,
		lastUsed:  now,
		useCount:  1,
		healthy:   true,
	}

	span.Event("connection_created", tracer.Fields{
		"pool.total_connections": len(p.connections),
		"connection.host":        config.Host,
		"connection.port":        config.Port,
		"connection.user":        config.Username,
	})

	return client, nil
}

// Release returns a connection to the pool
func (p *connectionPool) Release(key string, client tunnel.SSHClient) {
	span := p.tracer.TraceRelease(context.Background(), key)
	defer span.End()

	p.mu.RLock()
	entry, exists := p.connections[key]
	p.mu.RUnlock()

	if exists && entry.client == client {
		entry.mu.Lock()
		entry.lastUsed = time.Now()
		entry.mu.Unlock()

		span.Event("connection_released", tracer.Fields{
			"pool.key":        key,
			"entry.use_count": entry.useCount,
		})
	} else {
		span.Event("connection_not_found", tracer.String("reason", "client_mismatch_or_missing"))
	}
}

// Close closes all connections in the pool
func (p *connectionPool) Close() error {
	span := p.tracer.TracePool(context.Background(), "close")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	// Stop cleanup goroutine
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}
	close(p.stopCleanup)

	// Close all connections
	var errors []error
	for key, entry := range p.connections {
		if err := entry.client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close connection %s: %w", key, err))
		}
	}

	// Clear connections map
	p.connections = make(map[string]*poolEntry)

	if len(errors) > 0 {
		err := fmt.Errorf("errors closing connections: %v", errors)
		span.EndWithError(err)
		return err
	}

	span.Event("pool_closed", tracer.Int("connections_closed", len(p.connections)))
	return nil
}

// HealthCheck performs health check on all connections
func (p *connectionPool) HealthCheck(ctx context.Context) tunnel.HealthReport {
	span := p.tracer.TraceHealthCheck(ctx)
	defer span.End()

	p.mu.RLock()
	defer p.mu.RUnlock()

	report := tunnel.HealthReport{
		TotalConnections: len(p.connections),
		CheckedAt:        time.Now(),
		Connections:      make([]tunnel.ConnectionHealth, 0, len(p.connections)),
	}

	for key, entry := range p.connections {
		entry.mu.RLock()

		// Quick health check
		isConnected := entry.client.IsConnected()
		isIdle := time.Since(entry.lastUsed) > p.config.MaxIdleTime

		connectionHealth := tunnel.ConnectionHealth{
			Key:          key,
			Healthy:      isConnected && entry.healthy,
			LastUsed:     entry.lastUsed,
			UseCount:     entry.useCount,
			ResponseTime: 0, // Could add ping test here if needed
			Error:        "",
		}

		if !isConnected {
			connectionHealth.Healthy = false
			connectionHealth.Error = "connection lost"
			entry.healthy = false
		}

		if connectionHealth.Healthy {
			report.HealthyConnections++
			if isIdle {
				// Don't count idle connections as failed, just track them separately
			}
		} else {
			report.FailedConnections++
		}

		report.Connections = append(report.Connections, connectionHealth)
		entry.mu.RUnlock()
	}

	span.SetFields(tracer.Fields{
		"pool.total":   report.TotalConnections,
		"pool.healthy": report.HealthyConnections,
		"pool.failed":  report.FailedConnections,
	})

	span.Event("health_check_completed")
	return report
}

// startCleanup starts the background cleanup goroutine
func (p *connectionPool) startCleanup() {
	p.cleanupTicker = time.NewTicker(p.config.CleanupInterval)
	go func() {
		for {
			select {
			case <-p.cleanupTicker.C:
				p.cleanup()
			case <-p.stopCleanup:
				return
			}
		}
	}()
}

// cleanup removes stale and unhealthy connections
func (p *connectionPool) cleanup() {
	span := p.tracer.TracePool(context.Background(), "cleanup")
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	var removed int
	now := time.Now()
	keysToRemove := make([]string, 0)

	for key, entry := range p.connections {
		entry.mu.RLock()
		shouldRemove := false

		// Remove if idle too long
		if now.Sub(entry.lastUsed) > p.config.MaxIdleTime {
			shouldRemove = true
		}

		// Remove if unhealthy or disconnected
		if !entry.healthy || !entry.client.IsConnected() {
			shouldRemove = true
		}

		entry.mu.RUnlock()

		if shouldRemove {
			keysToRemove = append(keysToRemove, key)
		}
	}

	// Remove connections outside the iteration
	for _, key := range keysToRemove {
		if entry, exists := p.connections[key]; exists {
			entry.client.Close()
			delete(p.connections, key)
			removed++
		}
	}

	span.Event("cleanup_completed", tracer.Fields{
		"connections_removed":   removed,
		"connections_remaining": len(p.connections),
	})
}

// evictOldest removes the oldest unused connection to make room for a new one
func (p *connectionPool) evictOldest() error {
	if len(p.connections) == 0 {
		return nil
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	// Find the connection that was used longest ago
	for key, entry := range p.connections {
		entry.mu.RLock()
		lastUsed := entry.lastUsed
		entry.mu.RUnlock()

		if first || lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = lastUsed
			first = false
		}
	}

	// Remove the oldest connection
	if oldestKey != "" {
		if entry, exists := p.connections[oldestKey]; exists {
			entry.client.Close()
			delete(p.connections, oldestKey)
		}
	}

	return nil
}

// parseConnectionKey parses a connection key into ConnectionConfig
// Expected format: "username@host:port" or "username@host" (default port 22)
func parseConnectionKey(key string) (tunnel.ConnectionConfig, error) {
	// This is a simplified parser - in a real implementation you'd want more robust parsing
	// For now, we'll create a basic config that works with the factory
	config := tunnel.ConnectionConfig{
		Host:     "localhost", // Default - should be parsed from key
		Port:     22,          // Default SSH port
		Username: "root",      // Default - should be parsed from key
		Timeout:  30 * time.Second,
	}

	// TODO: Implement proper parsing of key format "user@host:port"
	// For now, return a basic config
	return config, nil
}
