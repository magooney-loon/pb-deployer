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
	Details   map[string]any
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

// DeploymentConfig holds deployment-specific configuration
type DeploymentConfig struct {
	Strategy           DeploymentStrategy
	MaxRetries         int
	RetryDelay         time.Duration
	HealthCheckTimeout time.Duration
	BackupEnabled      bool
	BackupPath         string
	ValidationEnabled  bool
	ArtifactValidation bool
}

// DeploymentStrategy represents deployment strategies
type DeploymentStrategy string

const (
	DeploymentStrategyRolling   DeploymentStrategy = "rolling"
	DeploymentStrategyBlueGreen DeploymentStrategy = "blue-green"
	DeploymentStrategyCanary    DeploymentStrategy = "canary"
	DeploymentStrategyRecreate  DeploymentStrategy = "recreate"
)

// DeploymentSpec defines a deployment specification
type DeploymentSpec struct {
	Name              string
	Version           string
	Environment       string
	ArtifactPath      string
	ServiceName       string
	WorkingDirectory  string
	PreDeployHooks    []string
	PostDeployHooks   []string
	HealthCheckURL    string
	HealthCheckPath   string
	Configuration     map[string]string
	EnvironmentVars   map[string]string
	Replicas          int
	Strategy          DeploymentStrategy
	RollbackOnFailure bool
	Dependencies      []string
}

// DeploymentResult represents the result of a deployment
type DeploymentResult struct {
	Success         bool
	Message         string
	Version         string
	PreviousVersion string
	RollbackData    map[string]any
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Steps           []DeploymentStep
}

// DeploymentStep represents a single deployment step
type DeploymentStep struct {
	Name      string
	Status    StepStatus
	Message   string
	StartTime time.Time
	EndTime   time.Time
	Error     error
}

// StepStatus represents the status of a deployment step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	Name          string
	State         DeploymentState
	Version       string
	Environment   string
	Health        HealthStatus
	LastUpdated   time.Time
	Replicas      DeploymentReplicas
	Configuration map[string]string
	Events        []DeploymentEvent
}

// DeploymentState represents deployment states
type DeploymentState string

const (
	DeploymentStateDeploying   DeploymentState = "deploying"
	DeploymentStateHealthy     DeploymentState = "healthy"
	DeploymentStateUnhealthy   DeploymentState = "unhealthy"
	DeploymentStateFailed      DeploymentState = "failed"
	DeploymentStateRollingBack DeploymentState = "rolling_back"
	DeploymentStateUnknown     DeploymentState = "unknown"
)

// HealthStatus represents health check status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// DeploymentReplicas represents replica information
type DeploymentReplicas struct {
	Desired   int
	Available int
	Ready     int
	Updated   int
}

// DeploymentEvent represents a deployment event
type DeploymentEvent struct {
	Type      string
	Message   string
	Timestamp time.Time
	Reason    string
}

// DeploymentInfo provides summary information about a deployment
type DeploymentInfo struct {
	Name        string
	Version     string
	Environment string
	State       DeploymentState
	Health      HealthStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DeploymentHealth represents detailed health information
type DeploymentHealth struct {
	Overall    HealthStatus
	Components map[string]ComponentHealth
	LastCheck  time.Time
	Message    string
}

// ComponentHealth represents health of a specific component
type ComponentHealth struct {
	Name    string
	Status  HealthStatus
	Message string
	Checks  []HealthCheck
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Name     string
	Status   HealthStatus
	Message  string
	Duration time.Duration
	LastRun  time.Time
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
	Data      map[string]any
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
