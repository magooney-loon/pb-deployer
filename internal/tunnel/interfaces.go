package tunnel

import (
	"context"
	"os"
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

// FileTransferInterface defines advanced file transfer operations
type FileTransferInterface interface {
	// UploadFile uploads a file to the remote server with advanced options
	UploadFile(ctx context.Context, localPath, remotePath string, opts *TransferOptions) error

	// DownloadFile downloads a file from the remote server with advanced options
	DownloadFile(ctx context.Context, remotePath, localPath string, opts *TransferOptions) error

	// SyncDirectory synchronizes directories between local and remote
	SyncDirectory(ctx context.Context, sourcePath, destPath string, direction TransferDirection, opts *SyncOptions) (*SyncResult, error)

	// CreateRemoteFile creates a file on the remote server with content
	CreateRemoteFile(ctx context.Context, remotePath string, content []byte, perms os.FileMode) error

	// GetRemoteFileInfo retrieves information about a remote file
	GetRemoteFileInfo(ctx context.Context, remotePath string) (os.FileInfo, error)

	// RemoveRemoteFile removes a file from the remote server
	RemoveRemoteFile(ctx context.Context, remotePath string) error

	// BatchTransfer performs multiple file transfers with optional concurrency
	BatchTransfer(ctx context.Context, operations []BatchTransferOperation, maxConcurrency int) error

	// ResumeTransfer resumes a partially transferred file
	ResumeTransfer(ctx context.Context, localPath, remotePath string, direction TransferDirection, opts *TransferOptions) error

	// CleanupTempFiles removes temporary files created during failed operations
	CleanupTempFiles(ctx context.Context, pattern string) error

	// GetDiskSpace retrieves disk space information for a remote path
	GetDiskSpace(ctx context.Context, remotePath string) (*DiskSpaceInfo, error)
}

// AdvancedHealthMonitorInterface defines advanced health monitoring operations
type AdvancedHealthMonitorInterface interface {
	// DeepHealthCheck performs comprehensive health analysis
	DeepHealthCheck(ctx context.Context) (*DetailedHealthReport, error)

	// PredictiveAnalysis performs predictive analysis
	PredictiveAnalysis(ctx context.Context) (*HealthPrediction, error)

	// AutoRecover attempts automatic recovery using specified strategy
	AutoRecover(ctx context.Context, strategy RecoveryStrategy) error

	// GetPerformanceMetrics retrieves current performance metrics
	GetPerformanceMetrics(ctx context.Context) (*PerformanceReport, error)

	// StartAdvancedMonitoring starts comprehensive monitoring
	StartAdvancedMonitoring(ctx context.Context)
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

// TransferOptions holds options for file transfer operations
type TransferOptions struct {
	Permissions         os.FileMode
	Owner               string
	Group               string
	PreservePermissions bool
	PreserveTimestamps  bool
	VerifyChecksum      bool
	ChecksumAlgorithm   ChecksumAlgorithm
	ProgressCallback    func(transferred, total int64)
	ChunkSize           int64
	AtomicOperation     bool
	CreateDirectories   bool
	OverwriteExisting   bool
}

// SyncOptions holds options for directory synchronization
type SyncOptions struct {
	DeleteExtra      bool
	PreserveLinks    bool
	FollowLinks      bool
	ExcludePatterns  []string
	IncludePatterns  []string
	DryRun           bool
	ProgressCallback func(operation, path string, transferred, total int64)
	Concurrency      int
	VerifyIntegrity  bool
}

// SyncResult represents the result of a directory sync operation
type SyncResult struct {
	FilesTransferred int
	FilesSkipped     int
	FilesDeleted     int
	BytesTransferred int64
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
	Errors           []error
}

// BatchTransferOperation represents a batch file transfer operation
type BatchTransferOperation struct {
	LocalPath  string
	RemotePath string
	Direction  TransferDirection
	Options    *TransferOptions
}

// ChecksumAlgorithm represents checksum algorithms
type ChecksumAlgorithm string

const (
	ChecksumMD5    ChecksumAlgorithm = "md5"
	ChecksumSHA256 ChecksumAlgorithm = "sha256"
)

// DiskSpaceInfo represents disk space information
type DiskSpaceInfo struct {
	Path       string
	Filesystem string
	Total      int64
	Used       int64
	Available  int64
	MountPoint string
}

// TransferProgress represents transfer progress information
type TransferProgress struct {
	FileName         string
	BytesTotal       int64
	BytesTransferred int64
	Percentage       float64
	Speed            int64 // bytes per second
	ETA              time.Duration
	StartTime        time.Time
	LastUpdate       time.Time
}

// DetailedHealthReport represents comprehensive health analysis
type DetailedHealthReport struct {
	Timestamp    time.Time
	BasicHealth  HealthResult
	Performance  *PerformanceReport
	Diagnostics  []DiagnosticCheck
	Metrics      *HealthMetricsSummary
	Predictions  *HealthPrediction
	Alerts       []HealthAlert
	OverallScore float64
}

// HealthPrediction represents predictive health analysis
type HealthPrediction struct {
	Timestamp   time.Time
	Window      time.Duration
	Confidence  float64
	TrendType   TrendType
	Predictions []HealthForecast
	Insights    []string
	Risks       []RiskFactor
}

// HealthForecast represents a future health prediction
type HealthForecast struct {
	Timestamp        time.Time
	PredictedHealthy bool
	SuccessRate      float64
	ResponseTime     time.Duration
	Confidence       float64
}

// TrendType represents health trend types
type TrendType string

const (
	TrendImproving TrendType = "improving"
	TrendStable    TrendType = "stable"
	TrendDegrading TrendType = "degrading"
)

// RiskFactor represents a potential risk
type RiskFactor struct {
	Type        RiskType
	Severity    RiskSeverity
	Description string
	Probability float64
	Impact      RiskImpact
}

// RiskType represents types of risks
type RiskType string

const (
	RiskHighFailureRate        RiskType = "high_failure_rate"
	RiskPerformanceDegradation RiskType = "performance_degradation"
	RiskResourceExhaustion     RiskType = "resource_exhaustion"
	RiskSecurityViolation      RiskType = "security_violation"
)

// RiskSeverity represents risk severity levels
type RiskSeverity string

const (
	RiskSeverityLow    RiskSeverity = "low"
	RiskSeverityMedium RiskSeverity = "medium"
	RiskSeverityHigh   RiskSeverity = "high"
)

// RiskImpact represents risk impact levels
type RiskImpact string

const (
	RiskImpactLow    RiskImpact = "low"
	RiskImpactMedium RiskImpact = "medium"
	RiskImpactHigh   RiskImpact = "high"
)

// PerformanceReport represents system performance metrics
type PerformanceReport struct {
	Timestamp   time.Time
	Tests       []PerformanceTestResult
	SystemInfo  *SystemInfo
	NetworkInfo *NetworkInfo
}

// PerformanceTestResult represents results of a performance test
type PerformanceTestResult struct {
	Name      string
	Duration  time.Duration
	Success   bool
	Output    string
	Error     string
	Timestamp time.Time
	Metrics   map[string]float64
}

// SystemInfo represents system information
type SystemInfo struct {
	Uptime        time.Duration
	KernelVersion string
	OSVersion     string
	CPUCores      string
}

// NetworkInfo represents network information
type NetworkInfo struct {
	Interfaces   []string
	NetworkStats []string
}

// DiagnosticCheck represents a diagnostic check result
type DiagnosticCheck struct {
	Name            string
	Category        string
	Status          DiagnosticStatus
	Message         string
	Timestamp       time.Time
	Details         map[string]any
	Recommendations []string
}

// DiagnosticStatus represents diagnostic check status
type DiagnosticStatus string

const (
	DiagnosticStatusPass    DiagnosticStatus = "pass"
	DiagnosticStatusWarning DiagnosticStatus = "warning"
	DiagnosticStatusFail    DiagnosticStatus = "fail"
)

// HealthMetricsSummary represents a summary of health metrics
type HealthMetricsSummary struct {
	TotalSamples    int
	SuccessRate     float64
	AverageResponse time.Duration
	OldestSample    time.Time
	LatestSample    time.Time
	DataRetention   time.Duration
}

// HealthAlert represents a health alert
type HealthAlert struct {
	Type      AlertType
	Severity  AlertSeverity
	Title     string
	Message   string
	Timestamp time.Time
	Metadata  map[string]any
}

// AlertType represents types of alerts
type AlertType string

const (
	AlertTypeHealth      AlertType = "health"
	AlertTypePerformance AlertType = "performance"
	AlertTypeSecurity    AlertType = "security"
	AlertTypeResource    AlertType = "resource"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityLow    AlertSeverity = "low"
	AlertSeverityMedium AlertSeverity = "medium"
	AlertSeverityHigh   AlertSeverity = "high"
)

// RecoveryStrategy represents recovery strategies
type RecoveryStrategy string

const (
	RecoveryStrategyReconnect RecoveryStrategy = "reconnect"
	RecoveryStrategyRestart   RecoveryStrategy = "restart"
	RecoveryStrategyReset     RecoveryStrategy = "reset"
	RecoveryStrategyEscalate  RecoveryStrategy = "escalate"
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
