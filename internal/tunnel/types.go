package tunnel

import (
	"time"
)

// SSHClient defines the interface for SSH operations
type SSHClient interface {
	Connect() error
	Close() error
	IsConnected() bool
	Execute(cmd string, opts ...ExecOption) (*Result, error)
	ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error)
	Upload(localPath, remotePath string, opts ...FileOption) error
	Download(remotePath, localPath string, opts ...FileOption) error
	Ping() error
	HostInfo() (string, error)
	SetTracer(tracer Tracer)
}

// Config holds SSH connection configuration
type Config struct {
	Host       string
	Port       int
	User       string
	Password   string        // Optional: password auth
	PrivateKey string        // Optional: key auth
	Passphrase string        // Optional: passphrase for encrypted key
	Timeout    time.Duration // Connection timeout
	RetryCount int           // Number of connection retries
	RetryDelay time.Duration // Delay between retries
}

// Result represents command execution result
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// ServiceStatus represents the status of a system service
type ServiceStatus struct {
	Name        string
	Active      bool
	Running     bool
	Enabled     bool
	Description string
	Since       time.Time
	MainPID     int
}

// FirewallRule represents a firewall rule configuration
type FirewallRule struct {
	Port        int
	Protocol    string // tcp, udp
	Source      string // IP address or CIDR
	Action      string // allow, deny
	Description string
}

// SSHConfig represents SSH hardening configuration
type SSHConfig struct {
	PasswordAuth        bool
	RootLogin           bool
	PubkeyAuth          bool
	MaxAuthTries        int
	ClientAliveInterval int
	ClientAliveCountMax int
	AllowUsers          []string
	AllowGroups         []string
	DenyUsers           []string
	DenyGroups          []string
}

// AppConfig represents application deployment configuration
type AppConfig struct {
	Name        string
	Version     string
	Source      string   // Local path or URL
	Target      string   // Remote path
	Service     string   // Service name to restart
	Backup      bool     // Backup before deploy
	PreDeploy   []string // Commands to run before
	PostDeploy  []string // Commands to run after
	HealthCheck string   // URL or command to verify
}

// ErrorType represents different types of SSH errors
type ErrorType int

const (
	ErrorUnknown ErrorType = iota
	ErrorConnection
	ErrorAuth
	ErrorExecution
	ErrorTimeout
	ErrorFileTransfer
	ErrorNotFound
	ErrorPermission
	ErrorVerification
)

// Error represents an SSH operation error
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Command represents a command to be executed
type Command struct {
	Cmd  string
	Opts []ExecOption
}

// Cmd creates a new command
func Cmd(cmd string, opts ...ExecOption) Command {
	return Command{Cmd: cmd, Opts: opts}
}

// ExecConfig holds execution configuration
type execConfig struct {
	timeout  time.Duration
	env      map[string]string
	workDir  string
	stream   func(string)
	sudo     bool
	sudoPass string
}

// ExecOption is a functional option for command execution
type ExecOption func(*execConfig)

// WithTimeout sets command timeout
func WithTimeout(d time.Duration) ExecOption {
	return func(c *execConfig) {
		c.timeout = d
	}
}

// WithEnv sets environment variable
func WithEnv(key, value string) ExecOption {
	return func(c *execConfig) {
		if c.env == nil {
			c.env = make(map[string]string)
		}
		c.env[key] = value
	}
}

// WithWorkDir sets working directory
func WithWorkDir(dir string) ExecOption {
	return func(c *execConfig) {
		c.workDir = dir
	}
}

// WithStream sets output stream handler
func WithStream(handler func(string)) ExecOption {
	return func(c *execConfig) {
		c.stream = handler
	}
}

// WithSudo executes command with sudo
func WithSudo() ExecOption {
	return func(c *execConfig) {
		c.sudo = true
	}
}

// WithSudoPassword sets sudo password (if needed)
func WithSudoPassword(pass string) ExecOption {
	return func(c *execConfig) {
		c.sudoPass = pass
	}
}

// UserConfig holds user creation configuration
type userConfig struct {
	home       string
	shell      string
	groups     []string
	sudoAccess bool
	systemUser bool
}

// UserOption is a functional option for user creation
type UserOption func(*userConfig)

// WithHome sets user home directory
func WithHome(path string) UserOption {
	return func(c *userConfig) {
		c.home = path
	}
}

// WithShell sets user shell
func WithShell(shell string) UserOption {
	return func(c *userConfig) {
		c.shell = shell
	}
}

// WithGroups adds user to groups
func WithGroups(groups ...string) UserOption {
	return func(c *userConfig) {
		c.groups = append(c.groups, groups...)
	}
}

// WithSudoAccess grants sudo access
func WithSudoAccess() UserOption {
	return func(c *userConfig) {
		c.sudoAccess = true
	}
}

// WithSystemUser creates a system user
func WithSystemUser() UserOption {
	return func(c *userConfig) {
		c.systemUser = true
	}
}

// FileTransferConfig holds file transfer configuration
type fileTransferConfig struct {
	progress func(int)
	mode     uint32
	preserve bool
}

// FileOption is a functional option for file operations
type FileOption func(*fileTransferConfig)

// WithProgress sets progress callback
func WithProgress(handler func(int)) FileOption {
	return func(c *fileTransferConfig) {
		c.progress = handler
	}
}

// WithFileMode sets file permissions
func WithFileMode(mode uint32) FileOption {
	return func(c *fileTransferConfig) {
		c.mode = mode
	}
}

// WithPreserve preserves file attributes
func WithPreserve() FileOption {
	return func(c *fileTransferConfig) {
		c.preserve = true
	}
}

// SystemInfo holds basic system information
type SystemInfo struct {
	OS           string
	Architecture string
	Hostname     string
}

// Tracer provides optional tracing/logging hooks
type Tracer interface {
	// OnConnect is called when connection is established
	OnConnect(host string, user string)
	// OnDisconnect is called when connection is closed
	OnDisconnect(host string)
	// OnExecute is called before command execution
	OnExecute(cmd string)
	// OnExecuteResult is called after command execution
	OnExecuteResult(cmd string, result *Result, err error)
	// OnUpload is called before file upload
	OnUpload(local, remote string)
	// OnUploadComplete is called after file upload
	OnUploadComplete(local, remote string, err error)
	// OnDownload is called before file download
	OnDownload(remote, local string)
	// OnDownloadComplete is called after file download
	OnDownloadComplete(remote, local string, err error)
	// OnError is called when an error occurs
	OnError(operation string, err error)
}

// NoOpTracer is a tracer that does nothing
type NoOpTracer struct{}

func (n *NoOpTracer) OnConnect(host string, user string)                    {}
func (n *NoOpTracer) OnDisconnect(host string)                              {}
func (n *NoOpTracer) OnExecute(cmd string)                                  {}
func (n *NoOpTracer) OnExecuteResult(cmd string, result *Result, err error) {}
func (n *NoOpTracer) OnUpload(local, remote string)                         {}
func (n *NoOpTracer) OnUploadComplete(local, remote string, err error)      {}
func (n *NoOpTracer) OnDownload(remote, local string)                       {}
func (n *NoOpTracer) OnDownloadComplete(remote, local string, err error)    {}
func (n *NoOpTracer) OnError(operation string, err error)                   {}

// SimpleLogger is a basic tracer that logs to stdout
type SimpleLogger struct {
	Verbose bool
}

func (s *SimpleLogger) OnConnect(host string, user string) {
	if s.Verbose {
		println("SSH: Connecting to", host, "as", user)
	}
}

func (s *SimpleLogger) OnDisconnect(host string) {
	if s.Verbose {
		println("SSH: Disconnected from", host)
	}
}

func (s *SimpleLogger) OnExecute(cmd string) {
	if s.Verbose {
		println("SSH: Executing:", cmd)
	}
}

func (s *SimpleLogger) OnExecuteResult(cmd string, result *Result, err error) {
	if s.Verbose {
		if err != nil {
			println("SSH: Command failed:", err.Error())
		} else {
			println("SSH: Command completed with exit code:", result.ExitCode)
		}
	}
}

func (s *SimpleLogger) OnUpload(local, remote string) {
	if s.Verbose {
		println("SSH: Uploading", local, "to", remote)
	}
}

func (s *SimpleLogger) OnUploadComplete(local, remote string, err error) {
	if s.Verbose {
		if err != nil {
			println("SSH: Upload failed:", err.Error())
		} else {
			println("SSH: Upload completed")
		}
	}
}

func (s *SimpleLogger) OnDownload(remote, local string) {
	if s.Verbose {
		println("SSH: Downloading", remote, "to", local)
	}
}

func (s *SimpleLogger) OnDownloadComplete(remote, local string, err error) {
	if s.Verbose {
		if err != nil {
			println("SSH: Download failed:", err.Error())
		} else {
			println("SSH: Download completed")
		}
	}
}

func (s *SimpleLogger) OnError(operation string, err error) {
	println("SSH Error in", operation+":", err.Error())
}
