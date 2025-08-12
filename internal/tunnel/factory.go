package tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"
)

// connectionFactory implements the ConnectionFactory interface
type connectionFactory struct {
	tracer      SSHTracer
	authHandler *AuthHandler
}

// NewConnectionFactory creates a new connection factory with dependency injection
func NewConnectionFactory(tracer SSHTracer) ConnectionFactory {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &connectionFactory{
		tracer:      tracer,
		authHandler: NewAuthHandler(tracer),
	}
}

// Create creates a new SSH client with the given configuration
func (f *connectionFactory) Create(config ConnectionConfig) (SSHClient, error) {
	span := f.tracer.TraceConnection(context.Background(), config.Host, config.Port, config.Username)
	defer span.End()

	span.Event("factory_create_start", map[string]any{
		"host":      config.Host,
		"port":      config.Port,
		"user":      config.Username,
		"auth_type": config.AuthMethod.Type,
	})

	// Validate configuration
	if err := f.validateConfig(config); err != nil {
		span.EndWithError(err)
		return nil, WrapConnectionError(config.Host, config.Port, config.Username, err)
	}

	span.Event("config_validated")

	// Set defaults for missing values
	config = f.setDefaults(config)

	span.Event("defaults_applied", map[string]any{
		"timeout":       config.Timeout.String(),
		"max_retries":   config.MaxRetries,
		"host_key_mode": int(config.HostKeyMode),
	})

	// Test connectivity before creating client
	if err := f.testConnectivity(config); err != nil {
		span.EndWithError(err)
		return nil, WrapConnectionError(config.Host, config.Port, config.Username, err)
	}

	span.Event("connectivity_tested")

	// Create SSH client
	client := NewSSHClient(config, f.tracer)

	span.Event("client_created")

	return client, nil
}

// validateConfig validates the connection configuration
func (f *connectionFactory) validateConfig(config ConnectionConfig) error {
	if config.Host == "" {
		return ErrInvalidConfig
	}

	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidConfig, config.Port)
	}

	if config.Username == "" {
		return fmt.Errorf("%w: username cannot be empty", ErrInvalidConfig)
	}

	// Validate authentication method
	if err := f.validateAuthMethod(config.AuthMethod); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Validate timeout
	if config.Timeout < 0 {
		return fmt.Errorf("%w: timeout cannot be negative", ErrInvalidConfig)
	}

	// Validate retry count
	if config.MaxRetries < 0 {
		return fmt.Errorf("%w: max retries cannot be negative", ErrInvalidConfig)
	}

	return nil
}

// validateAuthMethod validates the authentication method configuration
func (f *connectionFactory) validateAuthMethod(auth AuthMethod) error {
	switch auth.Type {
	case "key":
		if len(auth.PrivateKey) == 0 && auth.KeyPath == "" {
			return fmt.Errorf("key authentication requires either PrivateKey data or KeyPath")
		}
		return nil

	case "agent":
		// Agent auth doesn't require additional validation
		return nil

	case "password":
		if auth.Password == "" {
			return fmt.Errorf("password authentication requires a password")
		}
		return nil

	case "":
		return fmt.Errorf("authentication type cannot be empty")

	default:
		return fmt.Errorf("unsupported authentication type: %s", auth.Type)
	}
}

// setDefaults sets default values for missing configuration
func (f *connectionFactory) setDefaults(config ConnectionConfig) ConnectionConfig {
	if config.Port == 0 {
		config.Port = DefaultSSHPort
	}

	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	// Default to accepting new host keys if not specified
	// This is more permissive but practical for automated deployments
	if config.HostKeyMode < HostKeyStrict || config.HostKeyMode > HostKeyInsecure {
		config.HostKeyMode = HostKeyAcceptNew
	}

	return config
}

// testConnectivity performs a basic network connectivity test
func (f *connectionFactory) testConnectivity(config ConnectionConfig) error {
	address := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// Create a shorter timeout for connectivity test
	testTimeout := config.Timeout
	if testTimeout > 10*time.Second {
		testTimeout = 10 * time.Second
	}

	conn, err := net.DialTimeout("tcp", address, testTimeout)
	if err != nil {
		return fmt.Errorf("connectivity test failed: %w", err)
	}
	conn.Close()

	return nil
}

// CreateWithRetry creates a connection with automatic retry logic
func (f *connectionFactory) CreateWithRetry(config ConnectionConfig, strategy RetryStrategy) (SSHClient, error) {
	span := f.tracer.TraceConnection(context.Background(), config.Host, config.Port, config.Username)
	defer span.End()

	var lastErr error

	for attempt := 1; attempt <= strategy.MaxAttempts; attempt++ {
		span.Event("retry_attempt", map[string]any{
			"attempt":      attempt,
			"max_attempts": strategy.MaxAttempts,
		})

		client, err := f.Create(config)
		if err == nil {
			span.Event("retry_success", map[string]any{
				"attempt": attempt,
			})
			return client, nil
		}

		lastErr = err

		// Check if we should retry
		if !strategy.ShouldRetry(attempt, err) {
			break
		}

		// Don't sleep on the last attempt
		if attempt < strategy.MaxAttempts {
			delay := strategy.CalculateBackoff(attempt)
			span.Event("retry_delay", map[string]any{
				"delay": delay.String(),
				"error": err.Error(),
			})
			time.Sleep(delay)
		}
	}

	span.EndWithError(lastErr)
	return nil, lastErr
}

// CreateBatch creates multiple SSH clients concurrently
func (f *connectionFactory) CreateBatch(configs []ConnectionConfig) ([]SSHClient, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configurations provided")
	}

	type result struct {
		client SSHClient
		err    error
		index  int
	}

	resultCh := make(chan result, len(configs))

	// Create clients concurrently
	for i, config := range configs {
		go func(idx int, cfg ConnectionConfig) {
			client, err := f.Create(cfg)
			resultCh <- result{client: client, err: err, index: idx}
		}(i, config)
	}

	// Collect results
	clients := make([]SSHClient, len(configs))
	var errors []error

	for i := 0; i < len(configs); i++ {
		res := <-resultCh
		if res.err != nil {
			errors = append(errors, fmt.Errorf("config %d (%s): %w",
				res.index, configs[res.index].Host, res.err))
		} else {
			clients[res.index] = res.client
		}
	}

	if len(errors) > 0 {
		// Close any successful clients
		for _, client := range clients {
			if client != nil {
				client.Close()
			}
		}
		return nil, fmt.Errorf("batch creation failed: %v", errors)
	}

	return clients, nil
}

// TestConfiguration tests a configuration without creating a persistent client
func (f *connectionFactory) TestConfiguration(config ConnectionConfig) error {
	span := f.tracer.TraceConnection(context.Background(), config.Host, config.Port, config.Username)
	defer span.End()

	span.Event("test_config_start")

	// Validate configuration
	if err := f.validateConfig(config); err != nil {
		span.EndWithError(err)
		return err
	}

	// Set defaults
	config = f.setDefaults(config)

	// Test connectivity
	if err := f.testConnectivity(config); err != nil {
		span.EndWithError(err)
		return err
	}

	// Test authentication
	if err := f.authHandler.TestAuthentication(config); err != nil {
		span.EndWithError(err)
		return err
	}

	span.Event("test_config_success")
	return nil
}

// GetSupportedAuthMethods returns the authentication methods supported by this factory
func (f *connectionFactory) GetSupportedAuthMethods() []string {
	return []string{"key", "agent", "password"}
}

// CreateConfigFromServer creates a ConnectionConfig from a ServerConfig
func (f *connectionFactory) CreateConfigFromServer(server ServerConfig, authMethod AuthMethod) ConnectionConfig {
	config := ConnectionConfig{
		Host:        server.Host,
		Port:        server.Port,
		Username:    server.RootUsername,
		AuthMethod:  authMethod,
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultMaxRetries,
		HostKeyMode: HostKeyAcceptNew,
	}

	// Use app username if specified and not using root
	if server.AppUsername != "" && authMethod.Type != "password" {
		config.Username = server.AppUsername
	}

	// Set port default if not specified
	if config.Port == 0 {
		config.Port = DefaultSSHPort
	}

	return config
}

// DetectAndCreateConfig attempts to detect the best configuration for a server
func (f *connectionFactory) DetectAndCreateConfig(host string, port int, username string) (ConnectionConfig, error) {
	span := f.tracer.TraceConnection(context.Background(), host, port, username)
	defer span.End()

	config := ConnectionConfig{
		Host:        host,
		Port:        port,
		Username:    username,
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultMaxRetries,
		HostKeyMode: HostKeyAcceptNew,
	}

	// Set defaults
	config = f.setDefaults(config)

	// Test basic connectivity first
	if err := f.testConnectivity(config); err != nil {
		span.EndWithError(err)
		return config, fmt.Errorf("connectivity failed: %w", err)
	}

	// Try to detect authentication method
	authMethod, err := f.authHandler.DetectAuthMethod(username)
	if err != nil {
		span.Event("auth_detection_failed", map[string]any{
			"error": err.Error(),
		})
		// Default to key auth
		authMethod = AuthMethod{Type: "key"}
	}

	config.AuthMethod = authMethod

	span.Event("config_detected", map[string]any{
		"auth_type": authMethod.Type,
	})

	return config, nil
}

// ValidateAndNormalizeConfig validates and normalizes a configuration
func (f *connectionFactory) ValidateAndNormalizeConfig(config *ConnectionConfig) error {
	// Validate first
	if err := f.validateConfig(*config); err != nil {
		return err
	}

	// Apply defaults
	*config = f.setDefaults(*config)

	return nil
}

// GetConnectionString returns a string representation of the connection
func (f *connectionFactory) GetConnectionString(config ConnectionConfig) string {
	return fmt.Sprintf("%s@%s:%d", config.Username, config.Host, config.Port)
}

// FactoryInfo returns information about this factory
type FactoryInfo struct {
	SupportedAuthMethods []string
	DefaultPort          int
	DefaultTimeout       time.Duration
	DefaultMaxRetries    int
	Version              string
}

// GetInfo returns information about this connection factory
func (f *connectionFactory) GetInfo() FactoryInfo {
	return FactoryInfo{
		SupportedAuthMethods: f.GetSupportedAuthMethods(),
		DefaultPort:          DefaultSSHPort,
		DefaultTimeout:       DefaultTimeout,
		DefaultMaxRetries:    DefaultMaxRetries,
		Version:              "1.0.0",
	}
}
