package tunnel

import (
	"context"
	"time"
)

// SSHClient represents the core SSH interface with single responsibility
type SSHClient interface {
	// Connect establishes the SSH connection
	Connect(ctx context.Context) error

	// Execute runs a command and returns the output
	Execute(ctx context.Context, cmd string) (string, error)

	// ExecuteStream runs a command and streams output
	ExecuteStream(ctx context.Context, cmd string) (<-chan string, error)

	// Close closes the SSH connection
	Close() error

	// IsConnected returns true if the connection is active
	IsConnected() bool
}

// ConnectionFactory creates SSH clients
type ConnectionFactory interface {
	// Create creates a new SSH client with the given configuration
	Create(config ConnectionConfig) (SSHClient, error)
}

// Pool manages multiple SSH connections
type Pool interface {
	// Get retrieves or creates a connection for the given key
	Get(ctx context.Context, key string) (SSHClient, error)

	// Release returns a connection to the pool
	Release(key string, client SSHClient)

	// Close closes all connections in the pool
	Close() error

	// HealthCheck performs health check on all connections
	HealthCheck(ctx context.Context) HealthReport
}

// Executor handles high-level SSH operations
type Executor interface {
	// RunCommand executes a command with options
	RunCommand(ctx context.Context, cmd Command) (*Result, error)

	// RunScript executes a script on the remote server
	RunScript(ctx context.Context, script Script) (*Result, error)

	// TransferFile transfers a file to/from the remote server
	TransferFile(ctx context.Context, transfer Transfer) error
}

// ConnectionConfig contains SSH connection configuration
type ConnectionConfig struct {
	Host        string
	Port        int
	Username    string
	AuthMethod  AuthMethod
	Timeout     time.Duration
	HostKeyMode HostKeyMode
	MaxRetries  int
}

// AuthMethod represents SSH authentication method
type AuthMethod struct {
	Type       string // "key", "agent", "password"
	PrivateKey []byte
	Password   string
	KeyPath    string
}

// HostKeyMode defines how host keys are handled
type HostKeyMode int

const (
	// HostKeyStrict requires known host key
	HostKeyStrict HostKeyMode = iota
	// HostKeyAcceptNew accepts new host keys but rejects changed ones
	HostKeyAcceptNew
	// HostKeyInsecure accepts any host key (not recommended)
	HostKeyInsecure
)

// Command represents a command to execute
type Command struct {
	Cmd         string
	Sudo        bool
	Timeout     time.Duration
	Environment map[string]string
	WorkingDir  string
	User        string
}

// Script represents a script to execute
type Script struct {
	Content     string
	Interpreter string // e.g., "/bin/bash", "/usr/bin/python3"
	Args        []string
	Timeout     time.Duration
	Environment map[string]string
}

// Transfer represents a file transfer operation
type Transfer struct {
	Source      string
	Destination string
	Direction   TransferDirection
	Permissions string
	Owner       string
	Group       string
}

// TransferDirection indicates file transfer direction
type TransferDirection int

const (
	// TransferUpload uploads file to remote server
	TransferUpload TransferDirection = iota
	// TransferDownload downloads file from remote server
	TransferDownload
)

// Result represents command execution result
type Result struct {
	Output   string
	Error    error
	ExitCode int
	Duration time.Duration
	Started  time.Time
	Finished time.Time
}

// HealthReport contains health check results
type HealthReport struct {
	TotalConnections   int
	HealthyConnections int
	FailedConnections  int
	Connections        []ConnectionHealth
	CheckedAt          time.Time
}

// ConnectionHealth represents health status of a single connection
type ConnectionHealth struct {
	Key          string
	Healthy      bool
	LastUsed     time.Time
	UseCount     int64
	ResponseTime time.Duration
	Error        string
}

// PoolConfig contains pool configuration
type PoolConfig struct {
	MaxConnections  int
	MaxIdleTime     time.Duration
	HealthInterval  time.Duration
	CleanupInterval time.Duration
	MaxRetries      int
}

// SetupManager handles server setup operations
type SetupManager interface {
	// CreateUser creates a new user on the server
	CreateUser(ctx context.Context, user UserConfig) error

	// SetupSSHKeys configures SSH keys for a user
	SetupSSHKeys(ctx context.Context, user string, keys []string) error

	// CreateDirectories creates required directories
	CreateDirectories(ctx context.Context, dirs []DirectoryConfig) error

	// ConfigureSudo configures sudo access for a user
	ConfigureSudo(ctx context.Context, user string, commands []string) error

	// InstallPackages installs system packages
	InstallPackages(ctx context.Context, packages []string) error

	// SetupSystemUser sets up a complete system user with all configurations
	SetupSystemUser(ctx context.Context, config SystemUserConfig) error
}

// SecurityManager handles security operations
type SecurityManager interface {
	// ApplyLockdown applies security lockdown configuration
	ApplyLockdown(ctx context.Context, config SecurityConfig) error

	// SetupFirewall configures firewall rules
	SetupFirewall(ctx context.Context, rules []FirewallRule) error

	// SetupFail2ban configures fail2ban
	SetupFail2ban(ctx context.Context, config Fail2banConfig) error

	// HardenSSH applies SSH hardening configuration
	HardenSSH(ctx context.Context, settings SSHHardeningConfig) error

	// ConfigureAutoUpdates configures automatic security updates
	ConfigureAutoUpdates(ctx context.Context) error

	// AuditSecurity performs comprehensive security audit and compliance checking
	AuditSecurity(ctx context.Context) (*SecurityReport, error)
}

// ServiceManager handles systemd service operations
type ServiceManager interface {
	// ManageService performs service management operations
	ManageService(ctx context.Context, action ServiceAction, service string) error

	// GetServiceStatus returns service status
	GetServiceStatus(ctx context.Context, service string) (*ServiceStatus, error)

	// GetServiceLogs retrieves service logs
	GetServiceLogs(ctx context.Context, service string, lines int) (string, error)

	// EnableService enables a service to start on boot
	EnableService(ctx context.Context, service string) error

	// DisableService disables a service from starting on boot
	DisableService(ctx context.Context, service string) error

	// CreateServiceFile creates a systemd service file
	CreateServiceFile(ctx context.Context, service ServiceDefinition) error

	// WaitForService waits for a service to reach the desired state
	WaitForService(ctx context.Context, service string, timeout time.Duration) error
}

// DeploymentManager handles application deployment operations
type DeploymentManager interface {
	// Deploy performs application deployment with the given specification
	Deploy(ctx context.Context, deployment DeploymentSpec) (*DeploymentResult, error)

	// Rollback rolls back a deployment to a previous version
	Rollback(ctx context.Context, deployment string, version string) error

	// ValidateDeployment validates a deployment specification
	ValidateDeployment(ctx context.Context, deployment DeploymentSpec) error

	// GetDeploymentStatus returns the current status of a deployment
	GetDeploymentStatus(ctx context.Context, deployment string) (*DeploymentStatus, error)

	// ListDeployments returns a list of all deployments
	ListDeployments(ctx context.Context) ([]DeploymentInfo, error)

	// HealthCheck performs health check on a deployment
	HealthCheck(ctx context.Context, deployment string) (*DeploymentHealth, error)
}

// ServiceAction represents systemd service actions
type ServiceAction int

const (
	ServiceStart ServiceAction = iota
	ServiceStop
	ServiceRestart
	ServiceReload
	ServiceGetStatus
)

// ServiceStatus represents systemd service status
type ServiceStatus struct {
	Name        string
	Active      bool
	Enabled     bool
	State       string
	Description string
	Since       time.Time
}

// ServiceDefinition represents a systemd service definition
type ServiceDefinition struct {
	Name             string
	Description      string
	Type             string            // Service type (simple, forking, oneshot, etc.)
	ExecStart        string            // Command to start the service
	ExecStop         string            // Command to stop the service
	ExecReload       string            // Command to reload the service
	User             string            // User to run the service as
	Group            string            // Group to run the service as
	WorkingDirectory string            // Working directory for the service
	Environment      map[string]string // Environment variables
	Restart          string            // Restart policy (no, on-failure, always, etc.)
	RestartSec       time.Duration     // Time to wait before restarting
	After            []string          // Services to start after
	Requires         []string          // Required services
	WantedBy         string            // Target to enable service for
	Enabled          bool              // Whether to enable the service
}

// ServiceConfig holds configuration for service management
type ServiceConfig struct {
	ActionTimeout       time.Duration // Timeout for service actions
	StatusCheckInterval time.Duration // Interval for status checks when waiting
	DefaultLogLines     int           // Default number of log lines to retrieve
	MaxLogLines         int           // Maximum number of log lines allowed
}

// UserConfig contains user configuration
type UserConfig struct {
	Username   string
	HomeDir    string
	Shell      string
	Groups     []string
	CreateHome bool
	SystemUser bool
}

// DirectoryConfig contains directory configuration
type DirectoryConfig struct {
	Path        string
	Permissions string
	Owner       string
	Group       string
	Parents     bool // Create parent directories if needed
}

// SystemUserConfig holds configuration for complete system user setup
type SystemUserConfig struct {
	Username     string
	HomeDir      string
	Shell        string
	Groups       []string
	CreateHome   bool
	SystemUser   bool
	SetupSSH     bool
	SSHKeys      []string
	SetupSudo    bool
	SudoCommands []string
	Directories  []DirectoryConfig
}

// SetupConfig holds configuration for setup operations
type SetupConfig struct {
	DefaultShell     string
	DefaultGroups    []string
	PackageManager   string
	EnableBackup     bool
	BackupPath       string
	CreateHomeDir    bool
	SetPermissions   bool
	ValidateCommands bool
	MaxRetries       int
	RetryDelay       time.Duration
}

// DefaultSetupConfig returns default setup configuration
func DefaultSetupConfig() SetupConfig {
	return SetupConfig{
		DefaultShell:     "/bin/bash",
		DefaultGroups:    []string{"users"},
		PackageManager:   "auto", // Auto-detect
		EnableBackup:     true,
		BackupPath:       "/tmp/pb-deployer-backup",
		CreateHomeDir:    true,
		SetPermissions:   true,
		ValidateCommands: true,
		MaxRetries:       3,
		RetryDelay:       2 * time.Second,
	}
}

// SecurityConfig contains security lockdown configuration
type SecurityConfig struct {
	DisableRootLogin    bool
	DisablePasswordAuth bool
	FirewallRules       []FirewallRule
	Fail2banConfig      Fail2banConfig
	SSHHardeningConfig  SSHHardeningConfig
	AllowedPorts        []int
	AllowedUsers        []string
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	Port        int
	Protocol    string // "tcp", "udp"
	Action      string // "allow", "deny"
	Source      string // IP or CIDR
	Description string
}

// Fail2banConfig contains fail2ban configuration
type Fail2banConfig struct {
	Enabled    bool
	MaxRetries int
	BanTime    time.Duration
	FindTime   time.Duration
	Services   []string
}

// SSHHardeningConfig contains SSH hardening settings
type SSHHardeningConfig struct {
	PasswordAuthentication          bool
	PubkeyAuthentication            bool
	PermitRootLogin                 bool
	X11Forwarding                   bool
	AllowAgentForwarding            bool
	AllowTcpForwarding              bool
	ClientAliveInterval             int
	ClientAliveCountMax             int
	MaxAuthTries                    int
	MaxSessions                     int
	Protocol                        int
	IgnoreRhosts                    bool
	HostbasedAuthentication         bool
	PermitEmptyPasswords            bool
	ChallengeResponseAuthentication bool
	KerberosAuthentication          bool
	GSSAPIAuthentication            bool
}

// SecurityReport contains the results of a security audit
type SecurityReport struct {
	Timestamp       time.Time
	Overall         string // "excellent", "good", "fair", "poor", "critical", "unknown"
	Checks          []SecurityCheck
	Recommendations []string
}

// SecurityCheck represents a single security check result
type SecurityCheck struct {
	Name     string
	Category string // "ssh", "firewall", "intrusion_prevention", "system", etc.
	Status   string // "pass", "warning", "fail"
	Score    int    // 0-100
	Issues   []string
	Details  map[string]any
}

// SetupStep represents a step in setup process
type SetupStep struct {
	Name        string
	Function    func(context.Context) error
	Description string
	Required    bool
	Retry       bool
	MaxRetries  int
}

// ProgressUpdate represents progress update during operations
type ProgressUpdate struct {
	Step        string
	Status      string // "running", "success", "failed", "skipped"
	Message     string
	Details     string
	ProgressPct int
	Timestamp   time.Time
}

// Troubleshooter handles diagnostic operations
type Troubleshooter interface {
	// Diagnose performs comprehensive diagnostics
	Diagnose(ctx context.Context, config ConnectionConfig) []DiagnosticResult

	// TestNetwork tests network connectivity
	TestNetwork(ctx context.Context, host string, port int) DiagnosticResult

	// TestSSHService tests SSH service availability
	TestSSHService(ctx context.Context, host string, port int) DiagnosticResult

	// TestAuthentication tests authentication methods
	TestAuthentication(ctx context.Context, config ConnectionConfig) DiagnosticResult
}

// DiagnosticResult represents a diagnostic check result
type DiagnosticResult struct {
	Step       string
	Status     string // "success", "warning", "error"
	Message    string
	Details    string
	Suggestion string
	Duration   time.Duration
	Timestamp  time.Time
	Metadata   map[string]string
}

// ServiceTracer interface for tracing service operations
type ServiceTracer interface {
	TraceSetupOperation(ctx context.Context, operation, target string) ServiceSpan
	TraceServiceOperation(ctx context.Context, operation, service string) ServiceSpan
	TraceSecurityOperation(ctx context.Context, operation, component string) ServiceSpan
	TraceDeployment(ctx context.Context, name, version string) ServiceSpan
}

// ServiceSpan interface for service operation spans
type ServiceSpan interface {
	End()
	EndWithError(error)
	Event(name string, fields ...map[string]any)
	SetFields(fields map[string]any)
}

// SSHError represents SSH-specific errors
type SSHError struct {
	Op        string
	Server    string
	User      string
	Err       error
	Retryable bool
}

func (e *SSHError) Error() string {
	if e.Server != "" && e.User != "" {
		return "ssh: " + e.Op + " " + e.User + "@" + e.Server + ": " + e.Err.Error()
	}
	return "ssh: " + e.Op + ": " + e.Err.Error()
}

func (e *SSHError) Unwrap() error {
	return e.Err
}

// ConnectionState represents the state of a connection
type ConnectionState int

const (
	StateConnected ConnectionState = iota
	StateConnecting
	StateDisconnected
	StateError
	StateReconnecting
)

func (s ConnectionState) String() string {
	switch s {
	case StateConnected:
		return "connected"
	case StateConnecting:
		return "connecting"
	case StateDisconnected:
		return "disconnected"
	case StateError:
		return "error"
	case StateReconnecting:
		return "reconnecting"
	default:
		return "unknown"
	}
}

// NoOpServiceTracer provides a no-op implementation for when tracing is disabled
type NoOpServiceTracer struct{}

func (t *NoOpServiceTracer) TraceSetupOperation(ctx context.Context, operation, target string) ServiceSpan {
	return &NoOpServiceSpan{}
}

func (t *NoOpServiceTracer) TraceServiceOperation(ctx context.Context, operation, service string) ServiceSpan {
	return &NoOpServiceSpan{}
}

func (t *NoOpServiceTracer) TraceSecurityOperation(ctx context.Context, operation, component string) ServiceSpan {
	return &NoOpServiceSpan{}
}

func (t *NoOpServiceTracer) TraceDeployment(ctx context.Context, name, version string) ServiceSpan {
	return &NoOpServiceSpan{}
}

// NoOpServiceSpan provides a no-op implementation for spans
type NoOpServiceSpan struct{}

func (s *NoOpServiceSpan) End()                                        {}
func (s *NoOpServiceSpan) EndWithError(error)                          {}
func (s *NoOpServiceSpan) Event(name string, fields ...map[string]any) {}
func (s *NoOpServiceSpan) SetFields(fields map[string]any)             {}
