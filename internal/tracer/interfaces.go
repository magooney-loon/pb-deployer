package tracer

import (
	"context"
	"io"
	"time"
)

// Tracer is the main interface for tracing operations
type Tracer interface {
	// StartSpan starts a new span for an operation
	StartSpan(ctx context.Context, operation string) Span

	// WithField adds a field to all future spans
	WithField(key string, value any) Tracer

	// WithFields adds multiple fields to all future spans
	WithFields(fields Fields) Tracer

	// SetLevel sets the minimum trace level
	SetLevel(level Level)

	// Close closes the tracer and flushes any pending data
	Close() error
}

// Span represents a traced operation with timing and context
type Span interface {
	// End marks the span as complete
	End()

	// EndWithError marks the span as complete with an error
	EndWithError(err error)

	// SetStatus sets the span status
	SetStatus(status Status)

	// SetField adds a field to this span
	SetField(key string, value any) Span

	// SetFields adds multiple fields to this span
	SetFields(fields Fields) Span

	// Event logs an event within the span
	Event(name string, fields ...Field)

	// StartChild starts a child span
	StartChild(operation string) Span

	// Context returns the context with span information
	Context() context.Context
}

// Logger provides structured logging within the tracing context
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...Field)

	// WithField returns a logger with an additional field
	WithField(key string, value any) Logger

	// WithFields returns a logger with additional fields
	WithFields(fields Fields) Logger

	// WithError returns a logger with an error field
	WithError(err error) Logger

	// WithSpan returns a logger associated with a span
	WithSpan(span Span) Logger
}

// Exporter exports trace data to external systems
type Exporter interface {
	// Export exports a completed span
	Export(ctx context.Context, span *SpanData) error

	// Flush flushes any buffered data
	Flush(ctx context.Context) error

	// Shutdown shuts down the exporter
	Shutdown(ctx context.Context) error
}

// Sampler determines whether a span should be sampled
type Sampler interface {
	// ShouldSample determines if a span should be sampled
	ShouldSample(ctx context.Context, operation string, spanID SpanID, parentSpanID SpanID) bool
}

// Level represents the logging/tracing level
type Level int

const (
	// LevelTrace includes all trace data
	LevelTrace Level = iota
	// LevelDebug includes debug and above
	LevelDebug
	// LevelInfo includes info and above
	LevelInfo
	// LevelWarn includes warnings and above
	LevelWarn
	// LevelError includes errors and above
	LevelError
	// LevelFatal includes only fatal messages
	LevelFatal
)

// Status represents the status of a span
type Status int

const (
	// StatusOK indicates success
	StatusOK Status = iota
	// StatusError indicates an error occurred
	StatusError
	// StatusCanceled indicates the operation was canceled
	StatusCanceled
	// StatusTimeout indicates the operation timed out
	StatusTimeout
	// StatusUnknown indicates unknown status
	StatusUnknown
)

// Fields is a map of field key-value pairs
type Fields map[string]any

// Field represents a single key-value pair
type Field struct {
	Key   string
	Value any
}

// SpanData contains the data for a completed span
type SpanData struct {
	SpanID       SpanID
	ParentSpanID SpanID
	TraceID      TraceID
	Operation    string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Status       Status
	Fields       Fields
	Events       []Event
	Error        error
}

// Event represents an event within a span
type Event struct {
	Name      string
	Timestamp time.Time
	Fields    Fields
}

// SpanID is a unique identifier for a span
type SpanID [8]byte

// TraceID is a unique identifier for a trace
type TraceID [16]byte

// SpanContext contains span context that can be propagated
type SpanContext struct {
	TraceID      TraceID
	SpanID       SpanID
	ParentSpanID SpanID
	Sampled      bool
}

// Factory creates tracers with specific configurations
type Factory interface {
	// Create creates a new tracer with the given name
	Create(name string) Tracer

	// CreateWithConfig creates a new tracer with configuration
	CreateWithConfig(name string, config Config) Tracer
}

// Config contains tracer configuration
type Config struct {
	// Level is the minimum trace level
	Level Level

	// Sampler determines sampling strategy
	Sampler Sampler

	// Exporter exports trace data
	Exporter Exporter

	// Writer is the output writer for logs
	Writer io.Writer

	// EnableColors enables colored output
	EnableColors bool

	// EnableStackTrace enables stack traces for errors
	EnableStackTrace bool

	// ServiceName is the name of the service
	ServiceName string

	// ServiceVersion is the version of the service
	ServiceVersion string

	// Environment is the deployment environment
	Environment string

	// BufferSize is the size of the trace buffer
	BufferSize int

	// FlushInterval is how often to flush traces
	FlushInterval time.Duration
}

// Hook allows intercepting trace events
type Hook interface {
	// OnSpanStart is called when a span starts
	OnSpanStart(span *SpanData)

	// OnSpanEnd is called when a span ends
	OnSpanEnd(span *SpanData)

	// OnEvent is called when an event is logged
	OnEvent(span *SpanData, event Event)
}

// Provider manages tracer instances
type Provider interface {
	// GetTracer returns a tracer for the given component
	GetTracer(component string) Tracer

	// SetDefaultTracer sets the default tracer
	SetDefaultTracer(tracer Tracer)

	// RegisterHook registers a hook
	RegisterHook(hook Hook)

	// Shutdown shuts down all tracers
	Shutdown(ctx context.Context) error
}

// Formatter formats trace data for output
type Formatter interface {
	// Format formats a span for output
	Format(span *SpanData) ([]byte, error)

	// FormatEvent formats an event for output
	FormatEvent(event Event) ([]byte, error)
}

// Storage provides persistent storage for traces
type Storage interface {
	// Store stores a span
	Store(ctx context.Context, span *SpanData) error

	// Query queries stored spans
	Query(ctx context.Context, query Query) ([]*SpanData, error)

	// Delete deletes spans older than the given time
	Delete(ctx context.Context, before time.Time) error
}

// Query represents a query for stored spans
type Query struct {
	TraceID   *TraceID
	SpanID    *SpanID
	Operation string
	MinTime   time.Time
	MaxTime   time.Time
	Status    *Status
	Limit     int
	Offset    int
}

// Metrics provides metrics about tracing
type Metrics interface {
	// SpanStarted increments span started counter
	SpanStarted(operation string)

	// SpanEnded increments span ended counter
	SpanEnded(operation string, status Status, duration time.Duration)

	// EventLogged increments event counter
	EventLogged(operation string, eventName string)

	// ErrorRecorded increments error counter
	ErrorRecorded(operation string, errorType string)
}

// Propagator handles context propagation across boundaries
type Propagator interface {
	// Inject injects span context into a carrier
	Inject(ctx context.Context, carrier any) error

	// Extract extracts span context from a carrier
	Extract(ctx context.Context, carrier any) (SpanContext, error)
}

// SSHTracer provides SSH-specific tracing operations
type SSHTracer interface {
	Tracer

	// TraceConnection traces an SSH connection attempt
	TraceConnection(ctx context.Context, host string, port int, user string) Span

	// TraceCommand traces an SSH command execution
	TraceCommand(ctx context.Context, command string, sudo bool) Span

	// TraceFileTransfer traces a file transfer operation
	TraceFileTransfer(ctx context.Context, source, dest string, size int64) Span

	// TraceHealthCheck traces a health check operation
	TraceHealthCheck(ctx context.Context, target string) Span
}

// PoolTracer provides connection pool tracing
type PoolTracer interface {
	Tracer

	// TraceGet traces getting a connection from pool
	TraceGet(ctx context.Context, key string) Span

	// TraceRelease traces releasing a connection to pool
	TraceRelease(ctx context.Context, key string) Span

	// TraceHealthCheck traces pool health check
	TraceHealthCheck(ctx context.Context) Span

	// TraceCleanup traces pool cleanup operation
	TraceCleanup(ctx context.Context) Span
}

// SecurityTracer provides security operation tracing
type SecurityTracer interface {
	Tracer

	// TraceSecurityCheck traces a security check
	TraceSecurityCheck(ctx context.Context, checkType string) Span

	// TraceFirewallRule traces firewall rule application
	TraceFirewallRule(ctx context.Context, rule string, action string) Span

	// TraceAuthAttempt traces an authentication attempt
	TraceAuthAttempt(ctx context.Context, method string, user string) Span

	// TraceAuditEvent traces an audit event
	TraceAuditEvent(ctx context.Context, event string, details Fields) Span
}

// ServiceTracer provides service operation tracing
type ServiceTracer interface {
	Tracer

	// TraceServiceAction traces a service action
	TraceServiceAction(ctx context.Context, service string, action string) Span

	// TraceDeployment traces a deployment operation
	TraceDeployment(ctx context.Context, app string, version string) Span

	// TraceRollback traces a rollback operation
	TraceRollback(ctx context.Context, app string, fromVersion, toVersion string) Span

	// TraceHealthEndpoint traces health endpoint check
	TraceHealthEndpoint(ctx context.Context, endpoint string) Span
}

// Writer provides structured writing capabilities
type Writer interface {
	// Write writes trace data
	Write(data *SpanData) error

	// WriteEvent writes an event
	WriteEvent(event Event) error

	// Flush flushes buffered data
	Flush() error

	// Close closes the writer
	Close() error
}

// Level methods
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Status methods
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusError:
		return "ERROR"
	case StatusCanceled:
		return "CANCELED"
	case StatusTimeout:
		return "TIMEOUT"
	case StatusUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

// Helper functions for creating fields
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}
