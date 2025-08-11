package tunnel

import (
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerConfig represents server connection information
type ServerConfig struct {
	ID             string
	Name           string
	Host           string
	Port           int
	RootUsername   string
	AppUsername    string
	UseSSHAgent    bool
	ManualKeyPath  string
	SecurityLocked bool
}

// ConnectionMetadata holds metadata about a connection
type ConnectionMetadata struct {
	Key          string
	CreatedAt    time.Time
	LastUsed     time.Time
	UseCount     int64
	Healthy      bool
	State        ConnectionState
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
func (cm *ConnectionMetadata) SetState(state ConnectionState) {
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

// PoolEntry represents a connection pool entry
type PoolEntry struct {
	Client   SSHClient
	Metadata *ConnectionMetadata
}

// SessionConfig holds configuration for SSH sessions
type SessionConfig struct {
	Timeout     time.Duration
	Environment map[string]string
	PTY         bool
	PTYConfig   *PTYConfig
}

// PTYConfig holds PTY configuration
type PTYConfig struct {
	Term   string
	Height int
	Width  int
	Modes  ssh.TerminalModes
}

// DefaultPTYConfig returns default PTY configuration
func DefaultPTYConfig() *PTYConfig {
	return &PTYConfig{
		Term:   "xterm-256color",
		Height: 24,
		Width:  80,
		Modes: ssh.TerminalModes{
			ssh.ECHO:          0,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		},
	}
}

// OperationResult represents the result of an operation
type OperationResult struct {
	Success   bool
	Message   string
	Details   map[string]interface{}
	Duration  time.Duration
	Timestamp time.Time
}

// ExecutionOptions holds options for command execution
type ExecutionOptions struct {
	Timeout       time.Duration
	Sudo          bool
	SudoUser      string
	Environment   map[string]string
	WorkingDir    string
	CombineOutput bool
	Stream        bool
}

// DefaultExecutionOptions returns default execution options
func DefaultExecutionOptions() *ExecutionOptions {
	return &ExecutionOptions{
		Timeout:       30 * time.Second,
		Sudo:          false,
		CombineOutput: true,
		Stream:        false,
	}
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Interval            time.Duration
	Timeout             time.Duration
	MaxConsecutiveFails int
	RecoveryRetries     int
	EnableAutoRecovery  bool
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

// DeploymentConfig holds deployment-specific configuration
type DeploymentConfig struct {
	AppName         string
	DeploymentPath  string
	ServiceName     string
	BackupEnabled   bool
	BackupPath      string
	PreDeployHooks  []string
	PostDeployHooks []string
	HealthCheckURL  string
	RollbackOnFail  bool
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	Enabled         bool
	MetricsEndpoint string
	HealthEndpoint  string
	LogLevel        string
	LogPath         string
	AlertThresholds AlertThresholds
}

// AlertThresholds defines thresholds for alerts
type AlertThresholds struct {
	MaxConnectionFailures int
	MaxCommandFailures    int
	MaxResponseTime       time.Duration
	MinHealthyConnections int
}

// SecurityContext holds security-related context
type SecurityContext struct {
	RequiresSudo      bool
	SudoPassword      string // Should be handled securely
	AllowedCommands   []string
	ForbiddenCommands []string
	MaxCommandLength  int
	EnableAudit       bool
}

// ValidateCommand checks if a command is allowed
func (sc *SecurityContext) ValidateCommand(cmd string) error {
	if sc.MaxCommandLength > 0 && len(cmd) > sc.MaxCommandLength {
		return ErrCommandFailed
	}

	// Check forbidden commands
	for _, forbidden := range sc.ForbiddenCommands {
		if containsCommand(cmd, forbidden) {
			return ErrPermissionDenied
		}
	}

	// If allowed commands are specified, check if command is in the list
	if len(sc.AllowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range sc.AllowedCommands {
			if containsCommand(cmd, allowedCmd) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrPermissionDenied
		}
	}

	return nil
}

// containsCommand is a helper to check if a command contains a pattern
func containsCommand(cmd, pattern string) bool {
	// Simple contains check - could be enhanced with regex
	return len(cmd) >= len(pattern) &&
		(cmd == pattern ||
			(len(cmd) > len(pattern) && cmd[:len(pattern)+1] == pattern+" "))
}

// Event represents a system event
type Event struct {
	Type      EventType
	Timestamp time.Time
	Source    string
	Target    string
	Message   string
	Data      map[string]interface{}
	Error     error
}

// EventType represents the type of event
type EventType int

const (
	EventConnectionCreated EventType = iota
	EventConnectionClosed
	EventConnectionFailed
	EventCommandExecuted
	EventCommandFailed
	EventHealthCheckPassed
	EventHealthCheckFailed
	EventPoolCreated
	EventPoolClosed
	EventPoolExhausted
	EventSecurityViolation
	EventConfigurationChanged
)

func (et EventType) String() string {
	switch et {
	case EventConnectionCreated:
		return "connection_created"
	case EventConnectionClosed:
		return "connection_closed"
	case EventConnectionFailed:
		return "connection_failed"
	case EventCommandExecuted:
		return "command_executed"
	case EventCommandFailed:
		return "command_failed"
	case EventHealthCheckPassed:
		return "health_check_passed"
	case EventHealthCheckFailed:
		return "health_check_failed"
	case EventPoolCreated:
		return "pool_created"
	case EventPoolClosed:
		return "pool_closed"
	case EventPoolExhausted:
		return "pool_exhausted"
	case EventSecurityViolation:
		return "security_violation"
	case EventConfigurationChanged:
		return "configuration_changed"
	default:
		return "unknown"
	}
}

// EventHandler handles events
type EventHandler interface {
	HandleEvent(event Event)
}

// EventBus distributes events to handlers
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
		go handler.HandleEvent(event)
	}
}

// RateLimiter provides rate limiting for operations
type RateLimiter struct {
	rate     int // operations per interval
	interval time.Duration
	tokens   chan struct{}
	stop     chan struct{}
}

// NewRateLimiter creates a new rate limiter
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

// ProgressReporter handles progress reporting
type ProgressReporter struct {
	updates chan ProgressUpdate
	done    chan struct{}
}

// NewProgressReporter creates a new progress reporter
func NewProgressReporter(bufferSize int) *ProgressReporter {
	return &ProgressReporter{
		updates: make(chan ProgressUpdate, bufferSize),
		done:    make(chan struct{}),
	}
}

// Report sends a progress update
func (pr *ProgressReporter) Report(update ProgressUpdate) {
	select {
	case pr.updates <- update:
	case <-pr.done:
	}
}

// Updates returns the updates channel
func (pr *ProgressReporter) Updates() <-chan ProgressUpdate {
	return pr.updates
}

// Close closes the progress reporter
func (pr *ProgressReporter) Close() {
	close(pr.done)
	close(pr.updates)
}
