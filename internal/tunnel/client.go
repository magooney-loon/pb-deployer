package tunnel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client represents a single SSH connection to a server
type Client struct {
	config Config
	conn   *ssh.Client
	sftp   *sftp.Client
	tracer Tracer
}

// NewClient creates a new SSH client
func NewClient(config Config) (*Client, error) {
	// Set defaults
	if config.Port == 0 {
		config.Port = 22
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.RetryCount == 0 {
		config.RetryCount = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 5 * time.Second
	}

	// Validate config
	if config.Host == "" {
		return nil, &Error{
			Type:    ErrorConnection,
			Message: "host is required",
		}
	}
	if config.User == "" {
		return nil, &Error{
			Type:    ErrorConnection,
			Message: "user is required",
		}
	}
	// Validate SSH agent is available
	if !IsAgentAvailable() {
		return nil, &Error{
			Type:    ErrorAuth,
			Message: "SSH agent is required but not available",
		}
	}

	return &Client{
		config: config,
		tracer: &NoOpTracer{},
	}, nil
}

// SetTracer sets the tracer for logging/debugging
func (c *Client) SetTracer(tracer Tracer) {
	if tracer != nil {
		c.tracer = tracer
	} else {
		c.tracer = &NoOpTracer{}
	}
}

// Connect establishes SSH connection
func (c *Client) Connect() error {
	c.tracer.OnConnect(c.config.Host, c.config.User)

	// Build SSH client config with corruption-resistant auth
	authConfig := DevelopmentAuthConfig()
	if c.config.KnownHostsFile != "" {
		authConfig.KnownHostsFile = c.config.KnownHostsFile
	}

	var usingInsecureMode bool
	hostKeyCallback, err := GetHostKeyCallback(authConfig)
	if err != nil {
		c.tracer.OnError("get_host_key_callback", err)
		// Fallback to insecure mode for this connection attempt
		fmt.Printf("Warning: Using insecure host key verification due to error: %v\n", err)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
		usingInsecureMode = true
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.config.Timeout,
	}

	// Setup authentication
	authMethods, err := GetAuthMethods(authConfig)
	if err != nil {
		c.tracer.OnError("get_auth_methods", err)
		return &Error{
			Type:    ErrorAuth,
			Message: "failed to get authentication methods",
			Cause:   err,
		}
	}
	sshConfig.Auth = authMethods

	// Connect with retries
	var lastErr error
	for i := 0; i <= c.config.RetryCount; i++ {
		if i > 0 {
			time.Sleep(c.config.RetryDelay)
		}

		addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
		conn, err := ssh.Dial("tcp", addr, sshConfig)
		if err == nil {
			c.conn = conn
			// If we successfully connected, try to add the host key for future connections
			if usingInsecureMode {
				go c.addHostKeyAfterConnection()
			}
			return nil
		}

		// If this is a host key error and we haven't tried insecure mode, try it
		if strings.Contains(err.Error(), "key is unknown") && !usingInsecureMode {
			fmt.Printf("Host key unknown, retrying with insecure verification\n")
			sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
			usingInsecureMode = true
			// Don't increment retry counter for this attempt
			i--
			continue
		}

		lastErr = err
		c.tracer.OnError("connect", err)
	}

	c.tracer.OnDisconnect(c.config.Host)
	return &Error{
		Type:    ErrorConnection,
		Message: fmt.Sprintf("failed to connect after %d attempts", c.config.RetryCount+1),
		Cause:   lastErr,
	}
}

// Close closes the SSH connection
func (c *Client) Close() error {
	c.tracer.OnDisconnect(c.config.Host)

	if c.sftp != nil {
		c.sftp.Close()
		c.sftp = nil
	}

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}

	// Try to create a session to verify connection
	session, err := c.conn.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// Execute runs a command on the remote server
func (c *Client) Execute(cmd string, opts ...ExecOption) (*Result, error) {
	if c.conn == nil {
		return nil, &Error{
			Type:    ErrorConnection,
			Message: "not connected",
		}
	}

	// Apply options
	cfg := &execConfig{
		timeout: 60 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build command with environment and working directory
	fullCmd := c.buildCommand(cmd, cfg)
	c.tracer.OnExecute(fullCmd)

	// Create session
	session, err := c.conn.NewSession()
	if err != nil {
		c.tracer.OnError("create_session", err)
		return nil, &Error{
			Type:    ErrorExecution,
			Message: "failed to create session",
			Cause:   err,
		}
	}
	defer session.Close()

	// Set up output capture or streaming
	var stdout, stderr bytes.Buffer

	if cfg.stream != nil {
		// Stream output
		stdoutPipe, err := session.StdoutPipe()
		if err != nil {
			return nil, &Error{
				Type:    ErrorExecution,
				Message: "failed to create stdout pipe",
				Cause:   err,
			}
		}

		stderrPipe, err := session.StderrPipe()
		if err != nil {
			return nil, &Error{
				Type:    ErrorExecution,
				Message: "failed to create stderr pipe",
				Cause:   err,
			}
		}

		// Start command
		if err := session.Start(fullCmd); err != nil {
			c.tracer.OnError("start_command", err)
			return nil, &Error{
				Type:    ErrorExecution,
				Message: "failed to start command",
				Cause:   err,
			}
		}

		// Stream output
		go c.streamOutput(stdoutPipe, cfg.stream)
		go c.streamOutput(stderrPipe, cfg.stream)

		// Wait with timeout
		done := make(chan error)
		go func() {
			done <- session.Wait()
		}()

		select {
		case err := <-done:
			if err != nil {
				if exitErr, ok := err.(*ssh.ExitError); ok {
					result := &Result{
						ExitCode: exitErr.ExitStatus(),
					}
					c.tracer.OnExecuteResult(fullCmd, result, nil)
					return result, nil
				}
				c.tracer.OnExecuteResult(fullCmd, nil, err)
				return nil, &Error{
					Type:    ErrorExecution,
					Message: "command failed",
					Cause:   err,
				}
			}
		case <-time.After(cfg.timeout):
			session.Signal(ssh.SIGTERM)
			time.Sleep(2 * time.Second)
			session.Signal(ssh.SIGKILL)
			c.tracer.OnExecuteResult(fullCmd, nil, fmt.Errorf("timeout"))
			return nil, &Error{
				Type:    ErrorTimeout,
				Message: fmt.Sprintf("command timed out after %v", cfg.timeout),
			}
		}

		result := &Result{
			ExitCode: 0,
		}
		c.tracer.OnExecuteResult(fullCmd, result, nil)
		return result, nil
	} else {
		// Capture output
		session.Stdout = &stdout
		session.Stderr = &stderr

		// Run with timeout
		done := make(chan error)
		start := time.Now()

		go func() {
			done <- session.Run(fullCmd)
		}()

		select {
		case err := <-done:
			duration := time.Since(start)
			if err != nil {
				if exitErr, ok := err.(*ssh.ExitError); ok {
					result := &Result{
						Stdout:   stdout.String(),
						Stderr:   stderr.String(),
						ExitCode: exitErr.ExitStatus(),
						Duration: duration,
					}
					c.tracer.OnExecuteResult(fullCmd, result, nil)
					return result, nil
				}
				c.tracer.OnExecuteResult(fullCmd, nil, err)
				return nil, &Error{
					Type:    ErrorExecution,
					Message: "command failed",
					Cause:   err,
				}
			}

			result := &Result{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: 0,
				Duration: duration,
			}
			c.tracer.OnExecuteResult(fullCmd, result, nil)
			return result, nil

		case <-time.After(cfg.timeout):
			session.Signal(ssh.SIGTERM)
			time.Sleep(2 * time.Second)
			session.Signal(ssh.SIGKILL)
			c.tracer.OnExecuteResult(fullCmd, nil, fmt.Errorf("timeout"))
			return nil, &Error{
				Type:    ErrorTimeout,
				Message: fmt.Sprintf("command timed out after %v", cfg.timeout),
			}
		}
	}
}

// ExecuteSudo runs a command with sudo
func (c *Client) ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error) {
	// Add sudo to options
	opts = append(opts, WithSudo())

	// Apply options to get config
	cfg := &execConfig{
		timeout: 60 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Prepend sudo to command
	sudoCmd := "sudo "
	if cfg.sudoPass != "" {
		// Use sudo with password stdin
		sudoCmd = fmt.Sprintf("echo '%s' | sudo -S ", cfg.sudoPass)
	}

	return c.Execute(sudoCmd+cmd, opts...)
}

// Upload uploads a file to the remote server
func (c *Client) Upload(localPath, remotePath string, opts ...FileOption) error {
	c.tracer.OnUpload(localPath, remotePath)

	// Apply options
	cfg := &fileTransferConfig{
		mode: 0644,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Ensure SFTP client is connected
	if err := c.ensureSFTP(); err != nil {
		c.tracer.OnUploadComplete(localPath, remotePath, err)
		return err
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to open local file",
			Cause:   err,
		}
		c.tracer.OnUploadComplete(localPath, remotePath, err)
		return err
	}
	defer localFile.Close()

	// Get file info
	stat, err := localFile.Stat()
	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to stat local file",
			Cause:   err,
		}
		c.tracer.OnUploadComplete(localPath, remotePath, err)
		return err
	}

	// Create remote file
	remoteFile, err := c.sftp.Create(remotePath)
	if err != nil {
		// Try to create parent directory
		remoteDir := filepath.Dir(remotePath)
		c.sftp.MkdirAll(remoteDir)

		// Retry creating file
		remoteFile, err = c.sftp.Create(remotePath)
		if err != nil {
			err = &Error{
				Type:    ErrorFileTransfer,
				Message: "failed to create remote file",
				Cause:   err,
			}
			c.tracer.OnUploadComplete(localPath, remotePath, err)
			return err
		}
	}
	defer remoteFile.Close()

	// Copy with progress tracking
	if cfg.progress != nil {
		err = c.copyWithProgress(localFile, remoteFile, stat.Size(), cfg.progress)
	} else {
		_, err = io.Copy(remoteFile, localFile)
	}

	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to copy file",
			Cause:   err,
		}
		c.tracer.OnUploadComplete(localPath, remotePath, err)
		return err
	}

	// Set file permissions
	if cfg.preserve {
		remoteFile.Chmod(stat.Mode())
	} else {
		remoteFile.Chmod(os.FileMode(cfg.mode))
	}

	c.tracer.OnUploadComplete(localPath, remotePath, nil)
	return nil
}

// addHostKeyAfterConnection attempts to add the host key after a successful insecure connection
func (c *Client) addHostKeyAfterConnection() {
	if c.conn == nil {
		return
	}

	// Get the server's host key
	session, err := c.conn.NewSession()
	if err != nil {
		return
	}
	defer session.Close()

	// Try to get the host key and add it to known_hosts
	result, err := c.Execute("ssh-keyscan -t rsa,ecdsa,ed25519 localhost 2>/dev/null | head -1")
	if err == nil && result.ExitCode == 0 && result.Stdout != "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")

		file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return
		}
		defer file.Close()

		// Add a comment and the host key
		file.WriteString(fmt.Sprintf("# Added automatically for %s\n", c.config.Host))
		file.WriteString(strings.Replace(result.Stdout, "localhost", c.config.Host, 1))
		file.WriteString("\n")

		fmt.Printf("Added host key for %s to known_hosts\n", c.config.Host)
	}
}

// Download downloads a file from the remote server
func (c *Client) Download(remotePath, localPath string, opts ...FileOption) error {
	c.tracer.OnDownload(remotePath, localPath)

	// Apply options
	cfg := &fileTransferConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Ensure SFTP client is connected
	if err := c.ensureSFTP(); err != nil {
		c.tracer.OnDownloadComplete(remotePath, localPath, err)
		return err
	}

	// Open remote file
	remoteFile, err := c.sftp.Open(remotePath)
	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to open remote file",
			Cause:   err,
		}
		c.tracer.OnDownloadComplete(remotePath, localPath, err)
		return err
	}
	defer remoteFile.Close()

	// Get file info
	stat, err := remoteFile.Stat()
	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to stat remote file",
			Cause:   err,
		}
		c.tracer.OnDownloadComplete(remotePath, localPath, err)
		return err
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		// Try to create parent directory
		localDir := filepath.Dir(localPath)
		os.MkdirAll(localDir, 0755)

		// Retry creating file
		localFile, err = os.Create(localPath)
		if err != nil {
			err = &Error{
				Type:    ErrorFileTransfer,
				Message: "failed to create local file",
				Cause:   err,
			}
			c.tracer.OnDownloadComplete(remotePath, localPath, err)
			return err
		}
	}
	defer localFile.Close()

	// Copy with progress tracking
	if cfg.progress != nil {
		err = c.copyWithProgress(remoteFile, localFile, stat.Size(), cfg.progress)
	} else {
		_, err = io.Copy(localFile, remoteFile)
	}

	if err != nil {
		err = &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to copy file",
			Cause:   err,
		}
		c.tracer.OnDownloadComplete(remotePath, localPath, err)
		return err
	}

	// Set file permissions if preserving
	if cfg.preserve {
		localFile.Chmod(stat.Mode())
	}

	c.tracer.OnDownloadComplete(remotePath, localPath, nil)
	return nil
}

// Helper methods

func (c *Client) buildCommand(cmd string, cfg *execConfig) string {
	var parts []string

	// Add environment variables
	for k, v := range cfg.env {
		parts = append(parts, fmt.Sprintf("export %s='%s';", k, v))
	}

	// Change directory if specified
	if cfg.workDir != "" {
		parts = append(parts, fmt.Sprintf("cd '%s';", cfg.workDir))
	}

	// Add the actual command
	parts = append(parts, cmd)

	return strings.Join(parts, " ")
}

func (c *Client) ensureSFTP() error {
	if c.sftp != nil {
		return nil
	}

	if c.conn == nil {
		return &Error{
			Type:    ErrorConnection,
			Message: "not connected",
		}
	}

	sftp, err := sftp.NewClient(c.conn)
	if err != nil {
		return &Error{
			Type:    ErrorFileTransfer,
			Message: "failed to create SFTP client",
			Cause:   err,
		}
	}

	c.sftp = sftp
	return nil
}

func (c *Client) streamOutput(reader io.Reader, handler func(string)) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		handler(scanner.Text())
	}
}

func (c *Client) copyWithProgress(src io.Reader, dst io.Writer, total int64, progress func(int)) error {
	buffer := make([]byte, 32*1024) // 32KB buffer
	var written int64

	for {
		n, err := src.Read(buffer)
		if n > 0 {
			nw, err := dst.Write(buffer[:n])
			if err != nil {
				return err
			}
			if nw != n {
				return io.ErrShortWrite
			}

			written += int64(nw)
			if progress != nil && total > 0 {
				percent := int((written * 100) / total)
				progress(percent)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// Ping tests the connection with a simple command
func (c *Client) Ping() error {
	result, err := c.Execute("echo ping", WithTimeout(5*time.Second))
	if err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, "ping") {
		return &Error{
			Type:    ErrorConnection,
			Message: "ping failed",
		}
	}
	return nil
}

// HostInfo returns basic host information
func (c *Client) HostInfo() (string, error) {
	result, err := c.Execute("uname -a", WithTimeout(10*time.Second))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}
