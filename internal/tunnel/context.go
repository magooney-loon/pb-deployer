package tunnel

import (
	"context"
	"time"
)

// Context keys for storing values in context
type contextKey string

const (
	// ContextKeyConnectionKey stores the connection key for pool operations
	ContextKeyConnectionKey contextKey = "connection_key"

	// ContextKeyServerConfig stores the server configuration
	ContextKeyServerConfig contextKey = "server_config"

	// ContextKeyExecutionID stores a unique execution ID for tracing
	ContextKeyExecutionID contextKey = "execution_id"

	// ContextKeyOperationTimeout stores operation-specific timeout
	ContextKeyOperationTimeout contextKey = "operation_timeout"

	// ContextKeySecurityContext stores security context
	ContextKeySecurityContext contextKey = "security_context"

	// ContextKeyProgressReporter stores progress reporter
	ContextKeyProgressReporter contextKey = "progress_reporter"

	// ContextKeyEventBus stores event bus for publishing events
	ContextKeyEventBus contextKey = "event_bus"

	// ContextKeyAuditEnabled stores whether audit logging is enabled
	ContextKeyAuditEnabled contextKey = "audit_enabled"

	// ContextKeyRetryAttempt stores current retry attempt number
	ContextKeyRetryAttempt contextKey = "retry_attempt"

	// ContextKeyUserAgent stores user agent information
	ContextKeyUserAgent contextKey = "user_agent"
)

// WithConnectionKey adds a connection key to the context
func WithConnectionKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ContextKeyConnectionKey, key)
}

// GetConnectionKey retrieves the connection key from context
func GetConnectionKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(ContextKeyConnectionKey).(string)
	return key, ok && key != ""
}

// WithServerConfig adds server configuration to the context
func WithServerConfig(ctx context.Context, config ServerConfig) context.Context {
	return context.WithValue(ctx, ContextKeyServerConfig, config)
}

// GetServerConfig retrieves server configuration from context
func GetServerConfig(ctx context.Context) (ServerConfig, bool) {
	config, ok := ctx.Value(ContextKeyServerConfig).(ServerConfig)
	return config, ok
}

// WithExecutionID adds an execution ID to the context
func WithExecutionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ContextKeyExecutionID, id)
}

// GetExecutionID retrieves the execution ID from context
func GetExecutionID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ContextKeyExecutionID).(string)
	return id, ok && id != ""
}

// WithOperationTimeout adds an operation timeout to the context
func WithOperationTimeout(ctx context.Context, timeout time.Duration) context.Context {
	return context.WithValue(ctx, ContextKeyOperationTimeout, timeout)
}

// GetOperationTimeout retrieves the operation timeout from context
func GetOperationTimeout(ctx context.Context) (time.Duration, bool) {
	timeout, ok := ctx.Value(ContextKeyOperationTimeout).(time.Duration)
	return timeout, ok && timeout > 0
}

// WithSecurityContext adds security context
func WithSecurityContext(ctx context.Context, secCtx SecurityContext) context.Context {
	return context.WithValue(ctx, ContextKeySecurityContext, secCtx)
}

// GetSecurityContext retrieves security context from context
func GetSecurityContext(ctx context.Context) (SecurityContext, bool) {
	secCtx, ok := ctx.Value(ContextKeySecurityContext).(SecurityContext)
	return secCtx, ok
}

// WithProgressReporter adds a progress reporter to the context
func WithProgressReporter(ctx context.Context, reporter *ProgressReporter) context.Context {
	return context.WithValue(ctx, ContextKeyProgressReporter, reporter)
}

// GetProgressReporter retrieves the progress reporter from context
func GetProgressReporter(ctx context.Context) (*ProgressReporter, bool) {
	reporter, ok := ctx.Value(ContextKeyProgressReporter).(*ProgressReporter)
	return reporter, ok && reporter != nil
}

// WithEventBus adds an event bus to the context
func WithEventBus(ctx context.Context, bus *EventBus) context.Context {
	return context.WithValue(ctx, ContextKeyEventBus, bus)
}

// GetEventBus retrieves the event bus from context
func GetEventBus(ctx context.Context) (*EventBus, bool) {
	bus, ok := ctx.Value(ContextKeyEventBus).(*EventBus)
	return bus, ok && bus != nil
}

// WithAuditEnabled adds audit logging flag to the context
func WithAuditEnabled(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, ContextKeyAuditEnabled, enabled)
}

// GetAuditEnabled retrieves audit logging flag from context
func GetAuditEnabled(ctx context.Context) (bool, bool) {
	enabled, ok := ctx.Value(ContextKeyAuditEnabled).(bool)
	return enabled, ok
}

// WithRetryAttempt adds retry attempt number to the context
func WithRetryAttempt(ctx context.Context, attempt int) context.Context {
	return context.WithValue(ctx, ContextKeyRetryAttempt, attempt)
}

// GetRetryAttempt retrieves retry attempt number from context
func GetRetryAttempt(ctx context.Context) (int, bool) {
	attempt, ok := ctx.Value(ContextKeyRetryAttempt).(int)
	return attempt, ok && attempt > 0
}

// WithUserAgent adds user agent information to the context
func WithUserAgent(ctx context.Context, userAgent string) context.Context {
	return context.WithValue(ctx, ContextKeyUserAgent, userAgent)
}

// GetUserAgent retrieves user agent information from context
func GetUserAgent(ctx context.Context) (string, bool) {
	userAgent, ok := ctx.Value(ContextKeyUserAgent).(string)
	return userAgent, ok && userAgent != ""
}

// ContextInfo holds comprehensive context information
type ContextInfo struct {
	ConnectionKey       string
	ServerConfig        *ServerConfig
	ExecutionID         string
	OperationTimeout    time.Duration
	SecurityContext     *SecurityContext
	AuditEnabled        bool
	RetryAttempt        int
	UserAgent           string
	HasProgressReporter bool
	HasEventBus         bool
}

// GetContextInfo extracts all tunnel-related information from context
func GetContextInfo(ctx context.Context) ContextInfo {
	info := ContextInfo{}

	if key, ok := GetConnectionKey(ctx); ok {
		info.ConnectionKey = key
	}

	if config, ok := GetServerConfig(ctx); ok {
		info.ServerConfig = &config
	}

	if id, ok := GetExecutionID(ctx); ok {
		info.ExecutionID = id
	}

	if timeout, ok := GetOperationTimeout(ctx); ok {
		info.OperationTimeout = timeout
	}

	if secCtx, ok := GetSecurityContext(ctx); ok {
		info.SecurityContext = &secCtx
	}

	if enabled, ok := GetAuditEnabled(ctx); ok {
		info.AuditEnabled = enabled
	}

	if attempt, ok := GetRetryAttempt(ctx); ok {
		info.RetryAttempt = attempt
	}

	if userAgent, ok := GetUserAgent(ctx); ok {
		info.UserAgent = userAgent
	}

	_, info.HasProgressReporter = GetProgressReporter(ctx)
	_, info.HasEventBus = GetEventBus(ctx)

	return info
}

// CreateConnectionContext creates a context with connection information
func CreateConnectionContext(parent context.Context, config ServerConfig, connectionKey string) context.Context {
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	ctx = WithServerConfig(ctx, config)
	ctx = WithConnectionKey(ctx, connectionKey)

	return ctx
}

// CreateExecutionContext creates a context for command execution
func CreateExecutionContext(parent context.Context, executionID string, timeout time.Duration) context.Context {
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	ctx = WithExecutionID(ctx, executionID)
	if timeout > 0 {
		ctx = WithOperationTimeout(ctx, timeout)
	}

	return ctx
}

// CreateSecureExecutionContext creates a context with security context
func CreateSecureExecutionContext(parent context.Context, securityContext SecurityContext, auditEnabled bool) context.Context {
	ctx := parent
	if ctx == nil {
		ctx = context.Background()
	}

	ctx = WithSecurityContext(ctx, securityContext)
	ctx = WithAuditEnabled(ctx, auditEnabled)

	return ctx
}

// WithTimeout creates a context with the specified timeout, preferring operation timeout from context
func WithTimeout(parent context.Context, defaultTimeout time.Duration) (context.Context, context.CancelFunc) {
	timeout := defaultTimeout

	// Use operation timeout from context if available
	if opTimeout, ok := GetOperationTimeout(parent); ok {
		timeout = opTimeout
	}

	return context.WithTimeout(parent, timeout)
}

// WithDeadline creates a context with the specified deadline, adjusting for operation timeout
func WithDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	// Check if operation timeout would create an earlier deadline
	if opTimeout, ok := GetOperationTimeout(parent); ok {
		opDeadline := time.Now().Add(opTimeout)
		if opDeadline.Before(deadline) {
			deadline = opDeadline
		}
	}

	return context.WithDeadline(parent, deadline)
}

// ReportProgress reports progress if a reporter is available in context
func ReportProgress(ctx context.Context, update ProgressUpdate) {
	if reporter, ok := GetProgressReporter(ctx); ok {
		reporter.Report(update)
	}
}

// PublishEvent publishes an event if an event bus is available in context
func PublishEvent(ctx context.Context, event Event) {
	if bus, ok := GetEventBus(ctx); ok {
		bus.Publish(event)
	}
}

// ShouldAudit returns whether operations should be audited based on context
func ShouldAudit(ctx context.Context) bool {
	enabled, ok := GetAuditEnabled(ctx)
	return ok && enabled
}

// IsRetryAttempt checks if this is a retry attempt
func IsRetryAttempt(ctx context.Context) bool {
	attempt, ok := GetRetryAttempt(ctx)
	return ok && attempt > 1
}

// GetCurrentRetryAttempt returns the current retry attempt number (1-based)
func GetCurrentRetryAttempt(ctx context.Context) int {
	attempt, ok := GetRetryAttempt(ctx)
	if !ok {
		return 1 // First attempt
	}
	return attempt
}

// ValidateContext validates that required context values are present
func ValidateContext(ctx context.Context) error {
	if _, ok := GetConnectionKey(ctx); !ok {
		return ErrInvalidConfig // No connection key
	}

	return nil
}

// CloneContextWithTimeout creates a new context with the same values but different timeout
func CloneContextWithTimeout(source context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	// Create new context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// Copy values from source
	if key, ok := GetConnectionKey(source); ok {
		ctx = WithConnectionKey(ctx, key)
	}

	if config, ok := GetServerConfig(source); ok {
		ctx = WithServerConfig(ctx, config)
	}

	if id, ok := GetExecutionID(source); ok {
		ctx = WithExecutionID(ctx, id)
	}

	if secCtx, ok := GetSecurityContext(source); ok {
		ctx = WithSecurityContext(ctx, secCtx)
	}

	if enabled, ok := GetAuditEnabled(source); ok {
		ctx = WithAuditEnabled(ctx, enabled)
	}

	if attempt, ok := GetRetryAttempt(source); ok {
		ctx = WithRetryAttempt(ctx, attempt)
	}

	if userAgent, ok := GetUserAgent(source); ok {
		ctx = WithUserAgent(ctx, userAgent)
	}

	if reporter, ok := GetProgressReporter(source); ok {
		ctx = WithProgressReporter(ctx, reporter)
	}

	if bus, ok := GetEventBus(source); ok {
		ctx = WithEventBus(ctx, bus)
	}

	return ctx, cancel
}

// ExtractConnectionConfig creates a ConnectionConfig from context information
func ExtractConnectionConfig(ctx context.Context) (ConnectionConfig, error) {
	serverConfig, ok := GetServerConfig(ctx)
	if !ok {
		return ConnectionConfig{}, ErrInvalidConfig
	}

	config := ConnectionConfig{
		Host:        serverConfig.Host,
		Port:        serverConfig.Port,
		Username:    serverConfig.RootUsername,
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultMaxRetries,
		HostKeyMode: HostKeyAcceptNew,
	}

	// Use app username if available and appropriate
	if serverConfig.AppUsername != "" {
		config.Username = serverConfig.AppUsername
	}

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = DefaultSSHPort
	}

	// Apply operation timeout if available
	if timeout, ok := GetOperationTimeout(ctx); ok {
		config.Timeout = timeout
	}

	// Determine auth method based on server config
	if serverConfig.UseSSHAgent {
		config.AuthMethod = AuthMethod{Type: "agent"}
	} else if serverConfig.ManualKeyPath != "" {
		config.AuthMethod = AuthMethod{
			Type:    "key",
			KeyPath: serverConfig.ManualKeyPath,
		}
	} else {
		config.AuthMethod = AuthMethod{Type: "key"} // Default to key auth
	}

	return config, nil
}

// MustGetConnectionKey panics if connection key is not found in context
func MustGetConnectionKey(ctx context.Context) string {
	key, ok := GetConnectionKey(ctx)
	if !ok {
		panic("connection key not found in context")
	}
	return key
}

// MustGetServerConfig panics if server config is not found in context
func MustGetServerConfig(ctx context.Context) ServerConfig {
	config, ok := GetServerConfig(ctx)
	if !ok {
		panic("server config not found in context")
	}
	return config
}
