package pool

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	stats         *ConnectionStats
	eventBus      *EventBus
	mu            sync.RWMutex
	closed        bool
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// poolEntry represents a connection in the pool
type poolEntry struct {
	client   tunnel.SSHClient
	metadata *ConnectionMetadata
	mu       sync.RWMutex
}

// NewPool creates a new connection pool with dependency injection
func NewPool(factory tunnel.ConnectionFactory, config tunnel.PoolConfig, poolTracer tracer.PoolTracer) tunnel.Pool {
	if err := validatePoolConfig(config); err != nil {
		panic(fmt.Sprintf("invalid pool config: %v", err))
	}

	pool := &connectionPool{
		factory:     factory,
		tracer:      poolTracer,
		config:      config,
		connections: make(map[string]*poolEntry),
		stats:       &ConnectionStats{UptimeStart: time.Now()},
		eventBus:    &EventBus{},
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
		isHealthy := entry.metadata.Healthy
		lastUsed := entry.metadata.LastUsed
		entry.mu.RUnlock()

		if isHealthy && time.Since(lastUsed) < p.config.MaxIdleTime {
			// Check if connection is still alive (with proper locking)
			entry.mu.Lock()
			isConnected := entry.client.IsConnected()
			if isConnected {
				entry.metadata.UpdateLastUsed()
				useCount := entry.metadata.UseCount
				entry.mu.Unlock()

				span.Event("connection_reused",
					tracer.Int("pool.total_connections", len(p.connections)),
					tracer.Int64("entry.use_count", useCount),
				)

				// Publish event
				p.eventBus.Publish(Event{
					Type:      EventConnectionAcquired,
					Timestamp: time.Now(),
					ConnKey:   key,
					Message:   "Connection reused from pool",
				})

				return entry.client, nil
			} else {
				// Connection is dead, mark as unhealthy
				entry.metadata.SetHealthy(false)
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
	metadata := &ConnectionMetadata{
		Key:       key,
		CreatedAt: now,
		LastUsed:  now,
		UseCount:  1,
		Healthy:   true,
		State:     tunnel.StateConnected,
	}

	p.connections[key] = &poolEntry{
		client:   client,
		metadata: metadata,
	}

	// Update stats
	p.stats.IncrementConnections(true)

	// Publish event
	p.eventBus.Publish(Event{
		Type:      EventConnectionAcquired,
		Timestamp: time.Now(),
		ConnKey:   key,
		Message:   "New connection created and added to pool",
	})

	span.Event("connection_created",
		tracer.Int("pool.total_connections", len(p.connections)),
		tracer.String("connection.host", config.Host),
		tracer.Int("connection.port", config.Port),
		tracer.String("connection.user", config.Username),
	)

	return client, nil
}

// Release returns a connection to the pool
func (p *connectionPool) Release(key string, client tunnel.SSHClient) {
	span := p.tracer.TraceRelease(context.Background(), key)
	defer span.End()

	if client == nil {
		span.Event("connection_not_released", tracer.String("reason", "client_is_nil"))
		return
	}

	p.mu.RLock()
	entry, exists := p.connections[key]
	p.mu.RUnlock()

	if !exists {
		span.Event("connection_not_found", tracer.String("reason", "key_not_in_pool"))
		return
	}

	if entry.client != client {
		span.Event("connection_not_released", tracer.String("reason", "client_mismatch"))
		return
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Check if connection is still healthy
	isConnected := client.IsConnected()
	if !isConnected {
		entry.metadata.SetHealthy(false)
		span.Event("connection_marked_unhealthy", tracer.String("reason", "connection_lost"))
	}

	entry.metadata.LastUsed = time.Now()
	useCount := entry.metadata.UseCount

	span.Event("connection_released",
		tracer.String("pool.key", key),
		tracer.Int64("entry.use_count", useCount),
		tracer.Bool("entry.healthy", entry.metadata.Healthy),
	)

	// Publish event
	p.eventBus.Publish(Event{
		Type:      EventConnectionReleased,
		Timestamp: time.Now(),
		ConnKey:   key,
		Message:   "Connection released back to pool",
	})

	// If connection is unhealthy, remove it from pool
	if !entry.metadata.Healthy {
		entry.mu.Unlock()
		p.mu.Lock()
		if poolEntry, stillExists := p.connections[key]; stillExists && poolEntry == entry {
			delete(p.connections, key)
			client.Close()
			span.Event("unhealthy_connection_removed", tracer.String("pool.key", key))

			// Publish event
			p.eventBus.Publish(Event{
				Type:      EventConnectionEvicted,
				Timestamp: time.Now(),
				ConnKey:   key,
				Message:   "Unhealthy connection removed from pool",
			})
		}
		p.mu.Unlock()
		entry.mu.Lock() // Re-acquire lock for defer
	}
}

// Close closes all connections in the pool
func (p *connectionPool) Close() error {
	span := p.tracer.StartSpan(context.Background(), "pool_close")
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
	connectionCount := len(p.connections)
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

	span.Event("pool_closed", tracer.Int("connections_closed", connectionCount))
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
		isIdle := time.Since(entry.metadata.LastUsed) > p.config.MaxIdleTime
		lastUsed := entry.metadata.LastUsed
		useCount := entry.metadata.UseCount
		healthy := entry.metadata.Healthy

		entry.mu.RUnlock()

		// Update health status if needed
		if !isConnected && healthy {
			entry.mu.Lock()
			entry.metadata.SetHealthy(false)
			healthy = false
			entry.mu.Unlock()
		}

		connectionHealth := tunnel.ConnectionHealth{
			Key:          key,
			Healthy:      isConnected && healthy,
			LastUsed:     lastUsed,
			UseCount:     useCount,
			ResponseTime: entry.metadata.ResponseTime,
			Error:        "",
		}

		if !isConnected {
			connectionHealth.Healthy = false
			connectionHealth.Error = "connection lost"
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
	}

	span.SetField("pool.total", report.TotalConnections)
	span.SetField("pool.healthy", report.HealthyConnections)
	span.SetField("pool.failed", report.FailedConnections)

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
	span := p.tracer.StartSpan(context.Background(), "pool_cleanup")
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
		if now.Sub(entry.metadata.LastUsed) > p.config.MaxIdleTime {
			shouldRemove = true
		}

		// Remove if unhealthy or disconnected
		if !entry.metadata.Healthy || !entry.client.IsConnected() {
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

	span.Event("cleanup_completed",
		tracer.Int("connections_removed", removed),
		tracer.Int("connections_remaining", len(p.connections)),
	)
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
		lastUsed := entry.metadata.LastUsed
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
	if key == "" {
		return tunnel.ConnectionConfig{}, fmt.Errorf("connection key cannot be empty")
	}

	config := tunnel.ConnectionConfig{
		Port:    22, // Default SSH port
		Timeout: 30 * time.Second,
	}

	// Split on @ to get username and host:port
	atIndex := strings.LastIndex(key, "@")
	if atIndex == -1 {
		return tunnel.ConnectionConfig{}, fmt.Errorf("invalid key format: missing '@' separator")
	}

	config.Username = key[:atIndex]
	hostPort := key[atIndex+1:]

	if config.Username == "" {
		return tunnel.ConnectionConfig{}, fmt.Errorf("invalid key format: empty username")
	}

	if hostPort == "" {
		return tunnel.ConnectionConfig{}, fmt.Errorf("invalid key format: empty host")
	}

	// Check if port is specified
	colonIndex := strings.LastIndex(hostPort, ":")
	if colonIndex == -1 {
		// No port specified, use default
		config.Host = hostPort
	} else {
		config.Host = hostPort[:colonIndex]
		portStr := hostPort[colonIndex+1:]

		if config.Host == "" {
			return tunnel.ConnectionConfig{}, fmt.Errorf("invalid key format: empty host")
		}

		if portStr == "" {
			return tunnel.ConnectionConfig{}, fmt.Errorf("invalid key format: empty port")
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return tunnel.ConnectionConfig{}, fmt.Errorf("invalid port number: %w", err)
		}

		if port <= 0 || port > 65535 {
			return tunnel.ConnectionConfig{}, fmt.Errorf("port number out of range: %d", port)
		}

		config.Port = port
	}

	return config, nil
}

// CreateConnectionKey creates a properly formatted connection key from config
func CreateConnectionKey(config tunnel.ConnectionConfig) string {
	if config.Port == 22 {
		// Omit default port for cleaner keys
		return fmt.Sprintf("%s@%s", config.Username, config.Host)
	}
	return fmt.Sprintf("%s@%s:%d", config.Username, config.Host, config.Port)
}

// ValidateConnectionKey validates that a connection key has the correct format
func ValidateConnectionKey(key string) error {
	_, err := parseConnectionKey(key)
	return err
}
