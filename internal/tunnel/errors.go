package tunnel

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrConnectionLost indicates the SSH connection was lost
	ErrConnectionLost = errors.New("ssh connection lost")

	// ErrAuthenticationFailed indicates authentication failed
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrConnectionRefused indicates the connection was refused
	ErrConnectionRefused = errors.New("connection refused")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrHostKeyMismatch indicates host key verification failed
	ErrHostKeyMismatch = errors.New("host key mismatch")

	// ErrNoAuthMethod indicates no authentication method available
	ErrNoAuthMethod = errors.New("no authentication method available")

	// ErrPoolExhausted indicates the connection pool is exhausted
	ErrPoolExhausted = errors.New("connection pool exhausted")

	// ErrClientNotConnected indicates the client is not connected
	ErrClientNotConnected = errors.New("client not connected")

	// ErrInvalidConfig indicates invalid configuration
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrCommandFailed indicates command execution failed
	ErrCommandFailed = errors.New("command execution failed")

	// ErrPermissionDenied indicates permission was denied
	ErrPermissionDenied = errors.New("permission denied")

	// ErrServiceNotFound indicates a systemd service was not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrOperationCanceled indicates operation was canceled
	ErrOperationCanceled = errors.New("operation canceled")
)

// ConnectionError represents connection-specific errors
type ConnectionError struct {
	Host      string
	Port      int
	User      string
	Err       error
	Timestamp time.Time
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("connection to %s@%s:%d failed: %v", e.User, e.Host, e.Port, e.Err)
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// CommandError represents command execution errors
type CommandError struct {
	Command  string
	ExitCode int
	Output   string
	Err      error
}

func (e *CommandError) Error() string {
	if e.ExitCode != 0 {
		return fmt.Sprintf("command '%s' failed with exit code %d: %v", e.Command, e.ExitCode, e.Err)
	}
	return fmt.Sprintf("command '%s' failed: %v", e.Command, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// PoolError represents connection pool errors
type PoolError struct {
	Op  string
	Key string
	Err error
}

func (e *PoolError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("pool operation '%s' failed for key '%s': %v", e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("pool operation '%s' failed: %v", e.Op, e.Err)
}

func (e *PoolError) Unwrap() error {
	return e.Err
}

// SetupError represents setup operation errors
type SetupError struct {
	Step    string
	Details string
	Err     error
}

func (e *SetupError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("setup step '%s' failed: %s: %v", e.Step, e.Details, e.Err)
	}
	return fmt.Sprintf("setup step '%s' failed: %v", e.Step, e.Err)
}

func (e *SetupError) Unwrap() error {
	return e.Err
}

// SecurityError represents security operation errors
type SecurityError struct {
	Operation string
	Component string // "firewall", "fail2ban", "ssh", etc.
	Err       error
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security operation '%s' failed for %s: %v", e.Operation, e.Component, e.Err)
}

func (e *SecurityError) Unwrap() error {
	return e.Err
}

// ServiceError represents service management errors
type ServiceError struct {
	Service string
	Action  string
	Err     error
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("service '%s' %s failed: %v", e.Service, e.Action, e.Err)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// DeploymentError represents deployment operation errors
type DeploymentError struct {
	Deployment string
	Operation  string
	Err        error
}

func (e *DeploymentError) Error() string {
	return fmt.Sprintf("deployment '%s' %s failed: %v", e.Deployment, e.Operation, e.Err)
}

func (e *DeploymentError) Unwrap() error {
	return e.Err
}

// RetryableError wraps an error to indicate it can be retried
type RetryableError struct {
	Err       error
	Retries   int
	MaxDelay  time.Duration
	LastRetry time.Time
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (attempt %d): %v", e.Retries, e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable returns true if the error is retryable
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's explicitly marked as retryable
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}

	// Check for specific error types that are retryable
	errStr := err.Error()
	retryablePatterns := []string{
		"connection reset by peer",
		"broken pipe",
		"connection refused",
		"i/o timeout",
		"network is unreachable",
		"no route to host",
		"connection timed out",
		"temporary failure",
		"resource temporarily unavailable",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	// Check for specific error values
	switch {
	case errors.Is(err, ErrConnectionLost):
		return true
	case errors.Is(err, ErrConnectionRefused):
		return true
	case errors.Is(err, ErrTimeout):
		return true
	case errors.Is(err, ErrOperationCanceled):
		return false
	case errors.Is(err, ErrAuthenticationFailed):
		return false
	case errors.Is(err, ErrPermissionDenied):
		return false
	}

	return false
}

// IsConnectionError checks if an error is connection-related
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var connErr *ConnectionError
	if errors.As(err, &connErr) {
		return true
	}

	connectionPatterns := []string{
		"connection",
		"network",
		"socket",
		"dial",
		"tcp",
		"ssh",
	}

	errStr := strings.ToLower(err.Error())
	for _, pattern := range connectionPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsAuthError checks if an error is authentication-related
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrAuthenticationFailed) {
		return true
	}

	authPatterns := []string{
		"authentication",
		"permission denied",
		"publickey",
		"password",
		"unable to authenticate",
		"no supported methods remain",
	}

	errStr := strings.ToLower(err.Error())
	for _, pattern := range authPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// WrapConnectionError wraps an error as a connection error
func WrapConnectionError(host string, port int, user string, err error) error {
	return &ConnectionError{
		Host:      host,
		Port:      port,
		User:      user,
		Err:       err,
		Timestamp: time.Now(),
	}
}

// WrapCommandError wraps an error as a command error
func WrapCommandError(command string, exitCode int, output string, err error) error {
	return &CommandError{
		Command:  command,
		ExitCode: exitCode,
		Output:   output,
		Err:      err,
	}
}

// WrapPoolError wraps an error as a pool error
func WrapPoolError(op, key string, err error) error {
	return &PoolError{
		Op:  op,
		Key: key,
		Err: err,
	}
}

// WrapSetupError wraps an error as a setup error
func WrapSetupError(step, details string, err error) error {
	return &SetupError{
		Step:    step,
		Details: details,
		Err:     err,
	}
}

// WrapSecurityError wraps an error as a security error
func WrapSecurityError(operation, component string, err error) error {
	return &SecurityError{
		Operation: operation,
		Component: component,
		Err:       err,
	}
}

// WrapServiceError wraps an error as a service error
func WrapServiceError(service, action string, err error) error {
	return &ServiceError{
		Service: service,
		Action:  action,
		Err:     err,
	}
}

// WrapDeploymentError wraps an error as a deployment error
func WrapDeploymentError(deployment, operation string, err error) error {
	return &DeploymentError{
		Deployment: deployment,
		Operation:  operation,
		Err:        err,
	}
}

// MakeRetryable wraps an error to make it retryable
func MakeRetryable(err error, maxDelay time.Duration) error {
	return &RetryableError{
		Err:       err,
		Retries:   0,
		MaxDelay:  maxDelay,
		LastRetry: time.Now(),
	}
}

// RetryStrategy defines retry behavior
type RetryStrategy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryStrategy returns default retry configuration
func DefaultRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// CalculateBackoff calculates the next backoff delay
func (rs RetryStrategy) CalculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return rs.InitialDelay
	}

	delay := float64(rs.InitialDelay)
	for i := 0; i < attempt-1; i++ {
		delay *= rs.Multiplier
	}

	if delay > float64(rs.MaxDelay) {
		return rs.MaxDelay
	}

	return time.Duration(delay)
}

// ShouldRetry determines if an operation should be retried
func (rs RetryStrategy) ShouldRetry(attempt int, err error) bool {
	if attempt >= rs.MaxAttempts {
		return false
	}

	return IsRetryable(err)
}

// ErrorClassifier classifies errors for better handling
type ErrorClassifier struct {
	err error
}

// NewErrorClassifier creates a new error classifier
func NewErrorClassifier(err error) *ErrorClassifier {
	return &ErrorClassifier{err: err}
}

// Category returns the error category
func (ec *ErrorClassifier) Category() string {
	if ec.err == nil {
		return "none"
	}

	switch {
	case IsAuthError(ec.err):
		return "authentication"
	case IsConnectionError(ec.err):
		return "connection"
	case errors.Is(ec.err, ErrPermissionDenied):
		return "permission"
	case errors.Is(ec.err, ErrTimeout):
		return "timeout"
	case errors.Is(ec.err, ErrInvalidConfig):
		return "configuration"
	default:
		return "unknown"
	}
}

// Severity returns the error severity
func (ec *ErrorClassifier) Severity() string {
	if ec.err == nil {
		return "none"
	}

	switch {
	case errors.Is(ec.err, ErrAuthenticationFailed):
		return "critical"
	case errors.Is(ec.err, ErrPermissionDenied):
		return "critical"
	case errors.Is(ec.err, ErrInvalidConfig):
		return "critical"
	case errors.Is(ec.err, ErrConnectionLost):
		return "high"
	case errors.Is(ec.err, ErrTimeout):
		return "medium"
	case IsRetryable(ec.err):
		return "low"
	default:
		return "medium"
	}
}

// UserMessage returns a user-friendly error message
func (ec *ErrorClassifier) UserMessage() string {
	if ec.err == nil {
		return ""
	}

	switch ec.Category() {
	case "authentication":
		return "Authentication failed. Please check your credentials and SSH keys."
	case "connection":
		return "Could not connect to the server. Please check network connectivity and firewall settings."
	case "permission":
		return "Permission denied. Please check user permissions and sudo configuration."
	case "timeout":
		return "Operation timed out. The server may be slow or unresponsive."
	case "configuration":
		return "Configuration error. Please check your SSH settings."
	default:
		return "An error occurred during SSH operation."
	}
}

// FormatConnectionError formats a connection error for logging
func FormatConnectionError(err error, server, user string) string {
	if err == nil {
		return ""
	}

	classifier := NewErrorClassifier(err)

	return fmt.Sprintf(
		"SSH Error [%s/%s] %s@%s: %v - %s",
		classifier.Category(),
		classifier.Severity(),
		user,
		server,
		err,
		classifier.UserMessage(),
	)
}
