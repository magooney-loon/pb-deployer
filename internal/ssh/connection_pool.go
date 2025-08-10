package ssh

import (
	"fmt"
	"sync"
	"time"

	"pb-deployer/internal/models"
)

// ConnectionPool manages SSH connections with health monitoring and automatic cleanup
type ConnectionPool struct {
	connections map[string]*PooledConnection
	mutex       sync.RWMutex
	cleanup     chan string
	done        chan struct{}
	wg          sync.WaitGroup
}

// PooledConnection wraps an SSH manager with health tracking
type PooledConnection struct {
	manager   *SSHManager
	server    *models.Server
	isRoot    bool
	createdAt time.Time
	lastUsed  time.Time
	useCount  int64
	healthy   bool
	mutex     sync.RWMutex
}

// ConnectionHealthStatus represents the health status of a connection
type ConnectionHealthStatus struct {
	Healthy      bool          `json:"healthy"`
	LastUsed     time.Time     `json:"last_used"`
	Age          time.Duration `json:"age"`
	UseCount     int64         `json:"use_count"`
	LastError    string        `json:"last_error,omitempty"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
}

// GlobalConnectionPool is the singleton connection pool instance
var (
	globalPool     *ConnectionPool
	globalPoolOnce sync.Once
)

// GetConnectionPool returns the global connection pool instance
func GetConnectionPool() *ConnectionPool {
	globalPoolOnce.Do(func() {
		globalPool = NewConnectionPool()
		globalPool.Start()
	})
	return globalPool
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		cleanup:     make(chan string, 100),
		done:        make(chan struct{}),
	}
}

// Start begins the connection pool health monitoring
func (cp *ConnectionPool) Start() {
	cp.wg.Add(1)
	go cp.healthMonitor()
	cp.wg.Add(1)
	go cp.cleanupWorker()
}

// Stop gracefully shuts down the connection pool
func (cp *ConnectionPool) Stop() {
	close(cp.done)
	cp.wg.Wait()

	// Close all remaining connections
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	for key, conn := range cp.connections {
		conn.Close()
		delete(cp.connections, key)
	}
}

// GetOrCreateConnection retrieves an existing healthy connection or creates a new one
func (cp *ConnectionPool) GetOrCreateConnection(server *models.Server, asRoot bool) (*PooledConnection, error) {
	key := cp.connectionKey(server, asRoot)

	cp.mutex.RLock()
	if conn, exists := cp.connections[key]; exists {
		cp.mutex.RUnlock()

		// Check if connection is still healthy
		if conn.IsHealthy() {
			conn.UpdateLastUsed()
			return conn, nil
		}

		// Connection is unhealthy, remove it
		cp.RemoveConnection(key)
	} else {
		cp.mutex.RUnlock()
	}

	// Create new connection
	return cp.createConnection(server, asRoot)
}

// createConnection creates a new pooled connection
func (cp *ConnectionPool) createConnection(server *models.Server, asRoot bool) (*PooledConnection, error) {
	manager, err := NewSSHManager(server, asRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH manager: %w", err)
	}

	conn := &PooledConnection{
		manager:   manager,
		server:    server,
		isRoot:    asRoot,
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		useCount:  0,
		healthy:   true,
	}

	// Test the connection immediately
	if err := conn.TestHealth(); err != nil {
		manager.Close()
		return nil, fmt.Errorf("connection health test failed: %w", err)
	}

	key := cp.connectionKey(server, asRoot)

	cp.mutex.Lock()
	cp.connections[key] = conn
	cp.mutex.Unlock()

	return conn, nil
}

// RemoveConnection removes a connection from the pool
func (cp *ConnectionPool) RemoveConnection(key string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if conn, exists := cp.connections[key]; exists {
		conn.Close()
		delete(cp.connections, key)
	}
}

// GetConnectionStatus returns the status of all connections in the pool
func (cp *ConnectionPool) GetConnectionStatus() map[string]ConnectionHealthStatus {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	status := make(map[string]ConnectionHealthStatus)
	for key, conn := range cp.connections {
		status[key] = conn.GetHealthStatus()
	}

	return status
}

// CleanupStaleConnections removes connections that haven't been used recently
func (cp *ConnectionPool) CleanupStaleConnections(maxAge time.Duration) int {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	now := time.Now()
	removed := 0

	for key, conn := range cp.connections {
		if now.Sub(conn.lastUsed) > maxAge {
			conn.Close()
			delete(cp.connections, key)
			removed++
		}
	}

	return removed
}

// connectionKey generates a unique key for a server connection
func (cp *ConnectionPool) connectionKey(server *models.Server, asRoot bool) string {
	userType := "app"
	if asRoot {
		userType = "root"
	}
	return fmt.Sprintf("%s:%d:%s:%s", server.Host, server.Port, server.ID, userType)
}

// healthMonitor periodically checks the health of all connections
func (cp *ConnectionPool) healthMonitor() {
	defer cp.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.performHealthChecks()
		case <-cp.done:
			return
		}
	}
}

// cleanupWorker handles cleanup requests
func (cp *ConnectionPool) cleanupWorker() {
	defer cp.wg.Done()

	// Cleanup stale connections every 5 minutes
	staleTicker := time.NewTicker(5 * time.Minute)
	defer staleTicker.Stop()

	for {
		select {
		case key := <-cp.cleanup:
			cp.RemoveConnection(key)
		case <-staleTicker.C:
			removed := cp.CleanupStaleConnections(15 * time.Minute)
			if removed > 0 {
				fmt.Printf("Connection pool: cleaned up %d stale connections\n", removed)
			}
		case <-cp.done:
			return
		}
	}
}

// performHealthChecks checks the health of all connections
func (cp *ConnectionPool) performHealthChecks() {
	cp.mutex.RLock()
	connections := make([]*PooledConnection, 0, len(cp.connections))
	keys := make([]string, 0, len(cp.connections))

	for key, conn := range cp.connections {
		connections = append(connections, conn)
		keys = append(keys, key)
	}
	cp.mutex.RUnlock()

	// Check each connection's health
	for i, conn := range connections {
		if err := conn.TestHealth(); err != nil {
			// Mark for cleanup
			select {
			case cp.cleanup <- keys[i]:
			default:
				// Cleanup channel is full, skip
			}
		}
	}
}

// PooledConnection methods

// IsHealthy returns whether the connection is healthy
func (pc *PooledConnection) IsHealthy() bool {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	return pc.healthy && pc.manager.IsConnected()
}

// UpdateLastUsed updates the last used timestamp and increments use count
func (pc *PooledConnection) UpdateLastUsed() {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.lastUsed = time.Now()
	pc.useCount++
}

// TestHealth performs a health check on the connection
func (pc *PooledConnection) TestHealth() error {
	if pc.manager == nil {
		pc.setHealthy(false)
		return fmt.Errorf("SSH manager is nil")
	}

	start := time.Now()
	err := pc.manager.TestConnection()
	responseTime := time.Since(start)

	if err != nil {
		pc.setHealthy(false)
		return fmt.Errorf("health check failed (response time: %v): %w", responseTime, err)
	}

	pc.setHealthy(true)
	return nil
}

// setHealthy sets the healthy status thread-safely
func (pc *PooledConnection) setHealthy(healthy bool) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	pc.healthy = healthy
}

// GetHealthStatus returns the current health status
func (pc *PooledConnection) GetHealthStatus() ConnectionHealthStatus {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	status := ConnectionHealthStatus{
		Healthy:  pc.healthy,
		LastUsed: pc.lastUsed,
		Age:      time.Since(pc.createdAt),
		UseCount: pc.useCount,
	}

	// Test response time
	if pc.healthy && pc.manager != nil {
		start := time.Now()
		if err := pc.manager.TestConnection(); err != nil {
			status.LastError = err.Error()
			status.Healthy = false
		} else {
			status.ResponseTime = time.Since(start)
		}
	}

	return status
}

// ExecuteCommand executes a command using this pooled connection
func (pc *PooledConnection) ExecuteCommand(command string) (string, error) {
	if !pc.IsHealthy() {
		return "", fmt.Errorf("connection is not healthy")
	}

	pc.UpdateLastUsed()

	output, err := pc.manager.ExecuteCommand(command)
	if err != nil {
		// Check if the error indicates a connection problem
		if isConnectionError(err) {
			pc.setHealthy(false)
		}
	}

	return output, err
}

// ExecuteCommandStream executes a command with streaming output
func (pc *PooledConnection) ExecuteCommandStream(command string, output chan<- string) error {
	if !pc.IsHealthy() {
		return fmt.Errorf("connection is not healthy")
	}

	pc.UpdateLastUsed()

	err := pc.manager.ExecuteCommandStream(command, output)
	if err != nil {
		// Check if the error indicates a connection problem
		if isConnectionError(err) {
			pc.setHealthy(false)
		}
	}

	return err
}

// GetManager returns the underlying SSH manager (use with caution)
func (pc *PooledConnection) GetManager() *SSHManager {
	return pc.manager
}

// Close closes the pooled connection
func (pc *PooledConnection) Close() error {
	pc.setHealthy(false)
	if pc.manager != nil {
		return pc.manager.Close()
	}
	return nil
}

// isConnectionError checks if an error indicates a connection problem
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	connectionErrors := []string{
		"connection lost",
		"connection refused",
		"broken pipe",
		"network is unreachable",
		"no route to host",
		"connection reset by peer",
		"ssh: connection lost",
		"session not found",
		"connection appears to be dead",
	}

	for _, connErr := range connectionErrors {
		if contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// ConnectionManager provides high-level connection management
type ConnectionManager struct {
	pool *ConnectionPool
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		pool: GetConnectionPool(),
	}
}

// ExecuteCommand executes a command on a server using the connection pool
func (cm *ConnectionManager) ExecuteCommand(server *models.Server, asRoot bool, command string) (string, error) {
	conn, err := cm.pool.GetOrCreateConnection(server, asRoot)
	if err != nil {
		return "", fmt.Errorf("failed to get connection: %w", err)
	}

	return conn.ExecuteCommand(command)
}

// ExecuteCommandStream executes a command with streaming output using the connection pool
func (cm *ConnectionManager) ExecuteCommandStream(server *models.Server, asRoot bool, command string, output chan<- string) error {
	conn, err := cm.pool.GetOrCreateConnection(server, asRoot)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	return conn.ExecuteCommandStream(command, output)
}

// TestConnection tests connectivity to a server
func (cm *ConnectionManager) TestConnection(server *models.Server, asRoot bool) error {
	conn, err := cm.pool.GetOrCreateConnection(server, asRoot)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	return conn.TestHealth()
}

// GetConnectionStatus returns status of all connections
func (cm *ConnectionManager) GetConnectionStatus() map[string]ConnectionHealthStatus {
	return cm.pool.GetConnectionStatus()
}

// CleanupConnections removes stale connections
func (cm *ConnectionManager) CleanupConnections() int {
	return cm.pool.CleanupStaleConnections(15 * time.Minute)
}

// Shutdown gracefully shuts down the connection manager
func (cm *ConnectionManager) Shutdown() {
	cm.pool.Stop()
}

// Global connection manager instance
var globalConnectionManager *ConnectionManager
var globalConnectionManagerOnce sync.Once

// GetConnectionManager returns the global connection manager instance
func GetConnectionManager() *ConnectionManager {
	globalConnectionManagerOnce.Do(func() {
		globalConnectionManager = NewConnectionManager()
	})
	return globalConnectionManager
}
