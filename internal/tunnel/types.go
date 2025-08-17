package tunnel

import (
	"fmt"
	"time"
)

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

type Config struct {
	Host           string
	Port           int
	User           string
	KnownHostsFile string
	Timeout        time.Duration
	RetryCount     int
	RetryDelay     time.Duration
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type ServiceStatus struct {
	Name        string
	Active      bool
	Running     bool
	Enabled     bool
	Description string
	Since       time.Time
	MainPID     int
}

type FirewallRule struct {
	Port        int
	Protocol    string
	Source      string
	Action      string
	Description string
}

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

type AppConfig struct {
	Name        string
	Version     string
	Source      string
	Target      string
	Service     string
	Backup      bool
	PreDeploy   []string
	PostDeploy  []string
	HealthCheck string
}

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

type Error struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

type Command struct {
	Cmd  string
	Opts []ExecOption
}

func Cmd(cmd string, opts ...ExecOption) Command {
	return Command{Cmd: cmd, Opts: opts}
}

type execConfig struct {
	timeout  time.Duration
	env      map[string]string
	workDir  string
	stream   func(string)
	sudo     bool
	sudoPass string
}

type ExecOption func(*execConfig)

func WithTimeout(d time.Duration) ExecOption {
	return func(c *execConfig) {
		c.timeout = d
	}
}

func WithEnv(key, value string) ExecOption {
	return func(c *execConfig) {
		if c.env == nil {
			c.env = make(map[string]string)
		}
		c.env[key] = value
	}
}

func WithWorkDir(dir string) ExecOption {
	return func(c *execConfig) {
		c.workDir = dir
	}
}

func WithStream(handler func(string)) ExecOption {
	return func(c *execConfig) {
		c.stream = handler
	}
}

func WithSudo() ExecOption {
	return func(c *execConfig) {
		c.sudo = true
	}
}

func WithSudoPassword(pass string) ExecOption {
	return func(c *execConfig) {
		c.sudoPass = pass
	}
}

type userConfig struct {
	home       string
	shell      string
	groups     []string
	sudoAccess bool
	systemUser bool
}

type UserOption func(*userConfig)

func WithHome(path string) UserOption {
	return func(c *userConfig) {
		c.home = path
	}
}

func WithShell(shell string) UserOption {
	return func(c *userConfig) {
		c.shell = shell
	}
}

func WithGroups(groups ...string) UserOption {
	return func(c *userConfig) {
		c.groups = append(c.groups, groups...)
	}
}

func WithSudoAccess() UserOption {
	return func(c *userConfig) {
		c.sudoAccess = true
	}
}

func WithSystemUser() UserOption {
	return func(c *userConfig) {
		c.systemUser = true
	}
}

type fileTransferConfig struct {
	progress func(int)
	mode     uint32
	preserve bool
}

type FileOption func(*fileTransferConfig)

func WithProgress(handler func(int)) FileOption {
	return func(c *fileTransferConfig) {
		c.progress = handler
	}
}

func WithFileMode(mode uint32) FileOption {
	return func(c *fileTransferConfig) {
		c.mode = mode
	}
}

func WithPreserve() FileOption {
	return func(c *fileTransferConfig) {
		c.preserve = true
	}
}

type SystemInfo struct {
	OS           string
	Architecture string
	Hostname     string
}

type Tracer interface {
	OnConnect(host string, user string)
	OnDisconnect(host string)
	OnExecute(cmd string)
	OnExecuteResult(cmd string, result *Result, err error)
	OnUpload(local, remote string)
	OnUploadComplete(local, remote string, err error)
	OnDownload(remote, local string)
	OnDownloadComplete(remote, local string, err error)
	OnError(operation string, err error)
}

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

type SimpleLogger struct {
	Verbose bool
}

func (s *SimpleLogger) OnConnect(host string, user string) {
	if s.Verbose {
		fmt.Printf("SSH: Connecting to %s as %s\n", host, user)
	}
}

func (s *SimpleLogger) OnDisconnect(host string) {
	if s.Verbose {
		fmt.Printf("SSH: Disconnected from %s\n", host)
	}
}

func (s *SimpleLogger) OnExecute(cmd string) {
	if s.Verbose {
		fmt.Printf("SSH: Executing: %s\n", cmd)
	}
}

func (s *SimpleLogger) OnExecuteResult(cmd string, result *Result, err error) {
	if s.Verbose {
		if err != nil {
			fmt.Printf("SSH: Command failed: %s\n", err.Error())
		} else {
			fmt.Printf("SSH: Command completed with exit code: %d\n", result.ExitCode)
		}
	}
}

func (s *SimpleLogger) OnUpload(local, remote string) {
	if s.Verbose {
		fmt.Printf("SSH: Uploading %s to %s\n", local, remote)
	}
}

func (s *SimpleLogger) OnUploadComplete(local, remote string, err error) {
	if s.Verbose {
		if err != nil {
			fmt.Printf("SSH: Upload failed: %s\n", err.Error())
		} else {
			fmt.Println("SSH: Upload completed")
		}
	}
}

func (s *SimpleLogger) OnDownload(remote, local string) {
	if s.Verbose {
		fmt.Printf("SSH: Downloading %s to %s\n", remote, local)
	}
}

func (s *SimpleLogger) OnDownloadComplete(remote, local string, err error) {
	if s.Verbose {
		if err != nil {
			fmt.Printf("SSH: Download failed: %s\n", err.Error())
		} else {
			fmt.Println("SSH: Download completed")
		}
	}
}

func (s *SimpleLogger) OnError(operation string, err error) {
	fmt.Printf("SSH Error in %s: %s\n", operation, err.Error())
}
