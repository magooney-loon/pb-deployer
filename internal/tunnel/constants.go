package tunnel

import "time"

// Default configuration values
const (
	// DefaultSSHPort is the default SSH port
	DefaultSSHPort = 22

	// DefaultTimeout is the default connection timeout
	DefaultTimeout = 30 * time.Second

	// DefaultCommandTimeout is the default command execution timeout
	DefaultCommandTimeout = 5 * time.Minute

	// DefaultMaxRetries is the default number of retry attempts
	DefaultMaxRetries = 3

	// DefaultKeepAliveInterval is the default keep-alive interval
	DefaultKeepAliveInterval = 30 * time.Second

	// DefaultKeepAliveCountMax is the default max keep-alive probes
	DefaultKeepAliveCountMax = 3
)

// Pool configuration defaults
const (
	// DefaultMaxConnections is the default maximum pool connections
	DefaultMaxConnections = 10

	// DefaultMaxIdleTime is the default maximum idle time for connections
	DefaultMaxIdleTime = 15 * time.Minute

	// DefaultHealthCheckInterval is the default health check interval
	DefaultHealthCheckInterval = 30 * time.Second

	// DefaultCleanupInterval is the default cleanup interval
	DefaultCleanupInterval = 5 * time.Minute

	// DefaultPoolBufferSize is the default buffer size for pool channels
	DefaultPoolBufferSize = 100
)

// SSH configuration defaults
const (
	// DefaultUsername is the default SSH username
	DefaultUsername = "root"

	// DefaultAppUsername is the default application username
	DefaultAppUsername = "pocketbase"

	// DefaultShell is the default shell
	DefaultShell = "/bin/bash"

	// DefaultHomeDir is the default home directory pattern
	DefaultHomeDir = "/home/%s"

	// DefaultSSHDir is the default SSH directory
	DefaultSSHDir = ".ssh"

	// DefaultAuthorizedKeysFile is the default authorized_keys filename
	DefaultAuthorizedKeysFile = "authorized_keys"

	// DefaultKnownHostsFile is the default known_hosts filename
	DefaultKnownHostsFile = "known_hosts"

	// DefaultPrivateKeyFile is the default private key filename
	DefaultPrivateKeyFile = "id_rsa"

	// DefaultPublicKeyFile is the default public key filename
	DefaultPublicKeyFile = "id_rsa.pub"
)

// Directory and file permissions
const (
	// DirPermissions is the default directory permissions
	DirPermissions = 0755

	// SSHDirPermissions is the SSH directory permissions
	SSHDirPermissions = 0700

	// FilePermissions is the default file permissions
	FilePermissions = 0644

	// PrivateKeyPermissions is the private key file permissions
	PrivateKeyPermissions = 0600

	// ExecutablePermissions is the executable file permissions
	ExecutablePermissions = 0755
)

// Service management constants
const (
	// SystemctlCommand is the systemctl command path
	SystemctlCommand = "systemctl"

	// ServiceStartCommand is the service start command
	ServiceStartCommand = "start"

	// ServiceStopCommand is the service stop command
	ServiceStopCommand = "stop"

	// ServiceRestartCommand is the service restart command
	ServiceRestartCommand = "restart"

	// ServiceReloadCommand is the service reload command
	ServiceReloadCommand = "reload"

	// ServiceStatusCommand is the service status command
	ServiceStatusCommand = "status"

	// ServiceEnableCommand is the service enable command
	ServiceEnableCommand = "enable"

	// ServiceDisableCommand is the service disable command
	ServiceDisableCommand = "disable"

	// ServiceIsActiveCommand is the service is-active command
	ServiceIsActiveCommand = "is-active"

	// ServiceIsEnabledCommand is the service is-enabled command
	ServiceIsEnabledCommand = "is-enabled"
)

// Security configuration constants
const (
	// DefaultMaxAuthTries is the default maximum authentication attempts
	DefaultMaxAuthTries = 3

	// DefaultMaxSessions is the default maximum concurrent sessions
	DefaultMaxSessions = 10

	// DefaultClientAliveInterval is the default client alive interval in seconds
	DefaultClientAliveInterval = 300

	// DefaultClientAliveCountMax is the default client alive count max
	DefaultClientAliveCountMax = 2

	// DefaultLoginGraceTime is the default login grace time in seconds
	DefaultLoginGraceTime = 120

	// DefaultMaxStartups is the default maximum concurrent unauthenticated connections
	DefaultMaxStartups = "10:30:60"
)

// Firewall configuration constants
const (
	// UFWCommand is the ufw command path
	UFWCommand = "ufw"

	// IPTablesCommand is the iptables command path
	IPTablesCommand = "iptables"

	// FirewallDCommand is the firewall-cmd command path
	FirewallDCommand = "firewall-cmd"

	// DefaultSSHPort is already defined above
	// DefaultHTTPPort is the default HTTP port
	DefaultHTTPPort = 80

	// DefaultHTTPSPort is the default HTTPS port
	DefaultHTTPSPort = 443

	// DefaultPocketBasePort is the default PocketBase port
	DefaultPocketBasePort = 8090
)

// Fail2ban configuration constants
const (
	// Fail2banCommand is the fail2ban-client command path
	Fail2banCommand = "fail2ban-client"

	// DefaultBanTime is the default ban time in seconds
	DefaultBanTime = 3600

	// DefaultFindTime is the default find time in seconds
	DefaultFindTime = 600

	// DefaultMaxRetry is the default maximum retry attempts
	DefaultMaxRetry = 5

	// Fail2banConfigPath is the fail2ban configuration path
	Fail2banConfigPath = "/etc/fail2ban"

	// Fail2banJailLocal is the jail.local filename
	Fail2banJailLocal = "jail.local"
)

// Application paths
const (
	// DefaultDeploymentBase is the default deployment base directory
	DefaultDeploymentBase = "/opt/pocketbase"

	// DefaultAppsDir is the default applications directory
	DefaultAppsDir = "/opt/pocketbase/apps"

	// DefaultBackupDir is the default backup directory
	DefaultBackupDir = "/opt/pocketbase/backups"

	// DefaultLogDir is the default log directory
	DefaultLogDir = "/var/log/pocketbase"

	// DefaultTempDir is the default temporary directory
	DefaultTempDir = "/tmp/pocketbase-deploy"
)

// Command templates
const (
	// SudoCommandTemplate is the sudo command template
	SudoCommandTemplate = "sudo -n %s"

	// SudoUserCommandTemplate is the sudo with user command template
	SudoUserCommandTemplate = "sudo -n -u %s %s"

	// CreateUserCommandTemplate is the create user command template
	CreateUserCommandTemplate = "useradd -m -s %s %s"

	// AddToGroupCommandTemplate is the add to group command template
	AddToGroupCommandTemplate = "usermod -aG %s %s"

	// CreateDirectoryCommandTemplate is the create directory command template
	CreateDirectoryCommandTemplate = "mkdir -p %s"

	// ChownCommandTemplate is the chown command template
	ChownCommandTemplate = "chown %s:%s %s"

	// ChmodCommandTemplate is the chmod command template
	ChmodCommandTemplate = "chmod %s %s"
)

// SSH daemon configuration paths
const (
	// SSHDConfigPath is the SSH daemon configuration file path
	SSHDConfigPath = "/etc/ssh/sshd_config"

	// SSHDConfigBackupPath is the SSH daemon configuration backup path
	SSHDConfigBackupPath = "/etc/ssh/sshd_config.backup"

	// SSHServiceName is the SSH service name (may vary by distribution)
	SSHServiceName = "ssh"

	// SSHDServiceName is the SSHD service name (alternative)
	SSHDServiceName = "sshd"
)

// Progress reporting constants
const (
	// ProgressBufferSize is the progress channel buffer size
	ProgressBufferSize = 100

	// ProgressUpdateInterval is the progress update interval
	ProgressUpdateInterval = 500 * time.Millisecond
)

// Status constants
const (
	// StatusRunning indicates an operation is running
	StatusRunning = "running"

	// StatusSuccess indicates an operation succeeded
	StatusSuccess = "success"

	// StatusFailed indicates an operation failed
	StatusFailed = "failed"

	// StatusSkipped indicates an operation was skipped
	StatusSkipped = "skipped"

	// StatusWarning indicates an operation completed with warnings
	StatusWarning = "warning"
)

// Diagnostic constants
const (
	// DiagnosticTimeout is the timeout for diagnostic operations
	DiagnosticTimeout = 5 * time.Minute

	// NetworkTestTimeout is the timeout for network tests
	NetworkTestTimeout = 10 * time.Second

	// SSHBannerTimeout is the timeout for reading SSH banner
	SSHBannerTimeout = 5 * time.Second
)

// Rate limiting constants
const (
	// DefaultRateLimit is the default rate limit (operations per interval)
	DefaultRateLimit = 10

	// DefaultRateLimitInterval is the default rate limit interval
	DefaultRateLimitInterval = 1 * time.Second

	// DefaultBurstSize is the default burst size for rate limiting
	DefaultBurstSize = 20
)

// Retry constants
const (
	// MinRetryDelay is the minimum retry delay
	MinRetryDelay = 1 * time.Second

	// MaxRetryDelay is the maximum retry delay
	MaxRetryDelay = 30 * time.Second

	// RetryMultiplier is the retry delay multiplier
	RetryMultiplier = 2.0

	// RetryJitter is the maximum jitter to add to retry delays
	RetryJitter = 1 * time.Second
)

// Buffer sizes
const (
	// DefaultBufferSize is the default buffer size for I/O operations
	DefaultBufferSize = 32 * 1024 // 32KB

	// StreamBufferSize is the buffer size for streaming operations
	StreamBufferSize = 64 * 1024 // 64KB

	// CommandOutputBufferSize is the buffer size for command output
	CommandOutputBufferSize = 1024 * 1024 // 1MB
)

// Logging constants
const (
	// LogFieldHost is the log field name for host
	LogFieldHost = "host"

	// LogFieldPort is the log field name for port
	LogFieldPort = "port"

	// LogFieldUser is the log field name for user
	LogFieldUser = "user"

	// LogFieldCommand is the log field name for command
	LogFieldCommand = "command"

	// LogFieldDuration is the log field name for duration
	LogFieldDuration = "duration"

	// LogFieldError is the log field name for error
	LogFieldError = "error"

	// LogFieldKey is the log field name for connection key
	LogFieldKey = "key"

	// LogFieldState is the log field name for state
	LogFieldState = "state"
)

// Environment variables
const (
	// EnvSSHAuthSock is the SSH auth socket environment variable
	EnvSSHAuthSock = "SSH_AUTH_SOCK"

	// EnvHome is the home directory environment variable
	EnvHome = "HOME"

	// EnvUser is the user environment variable
	EnvUser = "USER"

	// EnvPath is the PATH environment variable
	EnvPath = "PATH"

	// EnvShell is the SHELL environment variable
	EnvShell = "SHELL"

	// EnvTerm is the TERM environment variable
	EnvTerm = "TERM"
)

// Default environment values
const (
	// DefaultPath is the default PATH value
	DefaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	// DefaultTerm is the default TERM value
	DefaultTerm = "xterm-256color"
)
