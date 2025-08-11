package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// sshClient implements the SSHClient interface with proper lifecycle management
type sshClient struct {
	config      ConnectionConfig
	conn        *ssh.Client
	tracer      SSHTracer
	mu          sync.RWMutex
	connected   bool
	lastUsed    time.Time
	connectedAt time.Time
}

// NewSSHClient creates a new SSH client instance
func NewSSHClient(config ConnectionConfig, tracer SSHTracer) SSHClient {
	return &sshClient{
		config: config,
		tracer: tracer,
	}
}

// Connect establishes the SSH connection with authentication and host key verification
func (c *sshClient) Connect(ctx context.Context) error {
	span := c.tracer.TraceConnection(ctx, c.config.Host, c.config.Port, c.config.Username)
	defer span.End()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected && c.conn != nil {
		span.Event("already_connected")
		return nil
	}

	// Create SSH configuration
	sshConfig, err := c.createSSHConfig()
	if err != nil {
		span.EndWithError(err)
		return WrapConnectionError(c.config.Host, c.config.Port, c.config.Username, err)
	}

	// Create address
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	// Set up dialer with timeout
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}

	span.Event("dialing", map[string]interface{}{
		"address": addr,
		"timeout": c.config.Timeout,
	})

	// Create connection with timeout context
	dialCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	netConn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		span.EndWithError(err)
		return WrapConnectionError(c.config.Host, c.config.Port, c.config.Username, err)
	}

	// Create SSH connection
	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		netConn.Close()
		span.EndWithError(err)
		return WrapConnectionError(c.config.Host, c.config.Port, c.config.Username, err)
	}

	// Create SSH client
	c.conn = ssh.NewClient(sshConn, chans, reqs)
	c.connected = true
	c.connectedAt = time.Now()
	c.lastUsed = time.Now()

	span.Event("connection_established", map[string]interface{}{
		"host":         c.config.Host,
		"port":         c.config.Port,
		"user":         c.config.Username,
		"connected_at": c.connectedAt,
	})

	return nil
}

// Execute runs a command synchronously and returns the output
func (c *sshClient) Execute(ctx context.Context, cmd string) (string, error) {
	span := c.tracer.TraceCommand(ctx, cmd, false)
	defer span.End()

	c.mu.Lock()
	if !c.connected || c.conn == nil {
		c.mu.Unlock()
		err := ErrClientNotConnected
		span.EndWithError(err)
		return "", err
	}
	c.lastUsed = time.Now()
	conn := c.conn
	c.mu.Unlock()

	// Create session
	session, err := conn.NewSession()
	if err != nil {
		span.EndWithError(err)
		return "", WrapCommandError(cmd, 0, "", err)
	}
	defer session.Close()

	span.SetFields(map[string]interface{}{
		"command": cmd,
		"host":    c.config.Host,
		"user":    c.config.Username,
	})

	start := time.Now()

	// Run command with timeout
	outputCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		output, err := session.CombinedOutput(cmd)
		if err != nil {
			errCh <- err
		} else {
			outputCh <- output
		}
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		err := ctx.Err()
		span.EndWithError(err)
		return "", WrapCommandError(cmd, 0, "", err)

	case err := <-errCh:
		duration := time.Since(start)

		// Extract exit code if possible
		exitCode := 0
		if exitErr, ok := err.(*ssh.ExitError); ok {
			exitCode = exitErr.ExitStatus()
		}

		span.Event("command_failed", map[string]interface{}{
			"duration":  duration,
			"exit_code": exitCode,
		})
		span.EndWithError(err)

		return "", WrapCommandError(cmd, exitCode, "", err)

	case output := <-outputCh:
		duration := time.Since(start)
		outputStr := string(output)

		span.Event("command_completed", map[string]interface{}{
			"duration":    duration,
			"output_size": len(output),
		})

		return outputStr, nil
	}
}

// ExecuteStream runs a command and streams output through a channel
func (c *sshClient) ExecuteStream(ctx context.Context, cmd string) (<-chan string, error) {
	span := c.tracer.TraceCommand(ctx, cmd, true)

	c.mu.Lock()
	if !c.connected || c.conn == nil {
		c.mu.Unlock()
		err := ErrClientNotConnected
		span.EndWithError(err)
		return nil, err
	}
	c.lastUsed = time.Now()
	conn := c.conn
	c.mu.Unlock()

	// Create session
	session, err := conn.NewSession()
	if err != nil {
		span.EndWithError(err)
		return nil, WrapCommandError(cmd, 0, "", err)
	}

	// Get stdout and stderr pipes
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		span.EndWithError(err)
		return nil, WrapCommandError(cmd, 0, "", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		span.EndWithError(err)
		return nil, WrapCommandError(cmd, 0, "", err)
	}

	// Start command
	if err := session.Start(cmd); err != nil {
		session.Close()
		span.EndWithError(err)
		return nil, WrapCommandError(cmd, 0, "", err)
	}

	span.SetFields(map[string]interface{}{
		"command":   cmd,
		"streaming": true,
		"host":      c.config.Host,
		"user":      c.config.Username,
	})

	// Create output channel
	outputCh := make(chan string, StreamBufferSize/1024)

	// Start streaming goroutine
	go func() {
		defer func() {
			session.Close()
			close(outputCh)
			span.End()
		}()

		// Merge stdout and stderr
		merged := io.MultiReader(stdout, stderr)
		buffer := make([]byte, DefaultBufferSize)

		for {
			select {
			case <-ctx.Done():
				session.Signal(ssh.SIGTERM)
				return
			default:
				n, err := merged.Read(buffer)
				if n > 0 {
					output := string(buffer[:n])
					select {
					case outputCh <- output:
					case <-ctx.Done():
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						span.Event("stream_error", map[string]interface{}{
							"error": err.Error(),
						})
					}
					return
				}
			}
		}
	}()

	span.Event("stream_started", map[string]interface{}{
		"command": cmd,
	})

	return outputCh, nil
}

// IsConnected returns true if the SSH connection is active
func (c *sshClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return false
	}

	// Simple health check by creating a session
	session, err := c.conn.NewSession()
	if err != nil {
		c.connected = false
		return false
	}
	session.Close()

	return true
}

// Close closes the SSH connection and cleans up resources
func (c *sshClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.connected = false

	if c.tracer != nil {
		span := c.tracer.TraceConnection(context.Background(), c.config.Host, c.config.Port, c.config.Username)
		span.Event("connection_closed", map[string]interface{}{
			"host":      c.config.Host,
			"port":      c.config.Port,
			"user":      c.config.Username,
			"duration":  time.Since(c.connectedAt),
			"last_used": c.lastUsed,
		})
		span.End()
	}

	return err
}

// createSSHConfig creates the SSH client configuration
func (c *sshClient) createSSHConfig() (*ssh.ClientConfig, error) {
	// Create authentication methods
	authMethods, err := c.createAuthMethods()
	if err != nil {
		return nil, fmt.Errorf("failed to create auth methods: %w", err)
	}

	if len(authMethods) == 0 {
		return nil, ErrNoAuthMethod
	}

	// Create host key callback
	hostKeyCallback, err := c.createHostKeyCallback()
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            c.config.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         c.config.Timeout,
	}

	return config, nil
}

// createAuthMethods creates SSH authentication methods based on configuration
func (c *sshClient) createAuthMethods() ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	switch c.config.AuthMethod.Type {
	case "key":
		if len(c.config.AuthMethod.PrivateKey) > 0 {
			// Use provided private key
			signer, err := ssh.ParsePrivateKey(c.config.AuthMethod.PrivateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		} else if c.config.AuthMethod.KeyPath != "" {
			// Load from file
			signer, err := c.loadPrivateKeyFromFile(c.config.AuthMethod.KeyPath, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load private key from %s: %w", c.config.AuthMethod.KeyPath, err)
			}
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}

	case "agent":
		// Use SSH agent
		agentAuth, err := c.connectToAgent()
		if err != nil {
			return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
		}
		authMethods = append(authMethods, agentAuth)

	case "password":
		if c.config.AuthMethod.Password != "" {
			authMethods = append(authMethods, ssh.Password(c.config.AuthMethod.Password))
		}

	default:
		return nil, fmt.Errorf("unsupported auth method: %s", c.config.AuthMethod.Type)
	}

	return authMethods, nil
}

// loadPrivateKeyFromFile loads a private key from file
func (c *sshClient) loadPrivateKeyFromFile(keyPath, passphrase string) (ssh.Signer, error) {
	// This would typically read from file system
	// For now, we'll return an error indicating file reading is not implemented
	return nil, fmt.Errorf("file reading not implemented in this context")
}

// connectToAgent connects to SSH agent
func (c *sshClient) connectToAgent() (ssh.AuthMethod, error) {
	socket := "/tmp/ssh-agent.sock" // This should come from environment
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	return ssh.PublicKeysCallback(agentClient.Signers), nil
}

// createHostKeyCallback creates the host key verification callback
func (c *sshClient) createHostKeyCallback() (ssh.HostKeyCallback, error) {
	switch c.config.HostKeyMode {
	case HostKeyStrict:
		// In a real implementation, this would check known_hosts file
		return ssh.FixedHostKey(nil), fmt.Errorf("strict host key checking not implemented")

	case HostKeyAcceptNew:
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			// Log the new host key acceptance
			if c.tracer != nil {
				span := c.tracer.TraceConnection(context.Background(), hostname, c.config.Port, c.config.Username)
				span.Event("host_key_accepted", map[string]interface{}{
					"hostname":    hostname,
					"remote_addr": remote.String(),
					"key_type":    key.Type(),
				})
				span.End()
			}
			return nil
		}, nil

	case HostKeyInsecure:
		return ssh.InsecureIgnoreHostKey(), nil

	default:
		return nil, fmt.Errorf("unsupported host key mode: %d", c.config.HostKeyMode)
	}
}

// GetLastUsed returns the last used timestamp (for pool management)
func (c *sshClient) GetLastUsed() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUsed
}

// GetConnectionInfo returns connection information for diagnostics
func (c *sshClient) GetConnectionInfo() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info := map[string]interface{}{
		"host":      c.config.Host,
		"port":      c.config.Port,
		"user":      c.config.Username,
		"connected": c.connected,
		"last_used": c.lastUsed,
	}

	if c.connected {
		info["connected_at"] = c.connectedAt
		info["connection_duration"] = time.Since(c.connectedAt)
	}

	return info
}

// SSHTracer interface for tracing SSH operations
type SSHTracer interface {
	TraceConnection(ctx context.Context, host string, port int, user string) Span
	TraceCommand(ctx context.Context, cmd string, streaming bool) Span
}

// Span interface for tracing spans
type Span interface {
	End()
	EndWithError(error)
	Event(name string, fields ...map[string]interface{})
	SetFields(fields map[string]interface{})
}

// NoOpTracer provides a no-op implementation for when tracing is disabled
type NoOpTracer struct{}

func (t *NoOpTracer) TraceConnection(ctx context.Context, host string, port int, user string) Span {
	return &NoOpSpan{}
}

func (t *NoOpTracer) TraceCommand(ctx context.Context, cmd string, streaming bool) Span {
	return &NoOpSpan{}
}

// NoOpSpan provides a no-op implementation for spans
type NoOpSpan struct{}

func (s *NoOpSpan) End()                                                {}
func (s *NoOpSpan) EndWithError(error)                                  {}
func (s *NoOpSpan) Event(name string, fields ...map[string]interface{}) {}
func (s *NoOpSpan) SetFields(fields map[string]interface{})             {}
