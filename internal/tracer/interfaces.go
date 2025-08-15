package tracer

import (
	"context"
	"io"
	"time"
)

// Tracer is the main interface for tracing operations
type Tracer interface {
	StartSpan(ctx context.Context, operation string) Span

	WithField(key string, value any) Tracer

	WithFields(fields Fields) Tracer

	SetLevel(level Level)

	Close() error
}

// Span represents a traced operation with timing and context
type Span interface {
	End()

	EndWithError(err error)

	SetStatus(status Status)

	SetField(key string, value any) Span

	SetFields(fields Fields) Span

	Event(name string, fields ...Field)

	StartChild(operation string) Span

	Context() context.Context
}

type Logger interface {
	Debug(msg string, fields ...Field)

	Info(msg string, fields ...Field)

	Warn(msg string, fields ...Field)

	Error(msg string, fields ...Field)

	Fatal(msg string, fields ...Field)

	WithField(key string, value any) Logger

	WithFields(fields Fields) Logger

	WithError(err error) Logger

	WithSpan(span Span) Logger
}

type Exporter interface {
	Export(ctx context.Context, span *SpanData) error

	Flush(ctx context.Context) error

	Shutdown(ctx context.Context) error
}

type Sampler interface {
	ShouldSample(ctx context.Context, operation string, spanID SpanID, parentSpanID SpanID) bool
}

type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// Status represents the status of a span
type Status int

const (
	StatusOK Status = iota
	StatusError
	StatusCanceled
	StatusTimeout
	StatusUnknown
)

type Fields map[string]any

type Field struct {
	Key   string
	Value any
}

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

type Event struct {
	Name      string
	Timestamp time.Time
	Fields    Fields
}

type SpanID [8]byte

type TraceID [16]byte

type SpanContext struct {
	TraceID      TraceID
	SpanID       SpanID
	ParentSpanID SpanID
	Sampled      bool
}

type Factory interface {
	Create(name string) Tracer

	CreateWithConfig(name string, config Config) Tracer
}

type Config struct {
	Level Level

	Sampler Sampler

	Exporter Exporter

	Writer io.Writer

	EnableColors bool

	EnableStackTrace bool

	ServiceName string

	ServiceVersion string

	Environment string

	BufferSize int

	FlushInterval time.Duration
}

// Hook allows intercepting trace events
type Hook interface {
	OnSpanStart(span *SpanData)

	OnSpanEnd(span *SpanData)

	OnEvent(span *SpanData, event Event)
}

type Provider interface {
	GetTracer(component string) Tracer

	SetDefaultTracer(tracer Tracer)

	RegisterHook(hook Hook)

	Shutdown(ctx context.Context) error
}

type Formatter interface {
	Format(span *SpanData) ([]byte, error)

	FormatEvent(event Event) ([]byte, error)
}

type Storage interface {
	Store(ctx context.Context, span *SpanData) error

	Query(ctx context.Context, query Query) ([]*SpanData, error)

	Delete(ctx context.Context, before time.Time) error
}

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

type Metrics interface {
	SpanStarted(operation string)

	SpanEnded(operation string, status Status, duration time.Duration)

	EventLogged(operation string, eventName string)

	ErrorRecorded(operation string, errorType string)
}

// Propagator handles context propagation across boundaries
type Propagator interface {
	// Inject injects span context into a carrier
	Inject(ctx context.Context, carrier any) error

	// Extract extracts span context from a carrier
	Extract(ctx context.Context, carrier any) (SpanContext, error)
}

type SSHTracer interface {
	Tracer

	TraceConnection(ctx context.Context, host string, port int, user string) Span

	TraceCommand(ctx context.Context, command string, sudo bool) Span

	TraceFileTransfer(ctx context.Context, source, dest string, size int64) Span

	TraceHealthCheck(ctx context.Context, target string) Span
}

type PoolTracer interface {
	Tracer

	TraceGet(ctx context.Context, key string) Span

	TraceRelease(ctx context.Context, key string) Span

	TraceHealthCheck(ctx context.Context) Span

	TraceCleanup(ctx context.Context) Span
}

type SecurityTracer interface {
	Tracer

	TraceSecurityCheck(ctx context.Context, checkType string) Span

	TraceFirewallRule(ctx context.Context, rule string, action string) Span

	TraceAuthAttempt(ctx context.Context, method string, user string) Span

	TraceAuditEvent(ctx context.Context, event string, details Fields) Span
}

type ServiceTracer interface {
	Tracer

	TraceServiceAction(ctx context.Context, service string, action string) Span

	TraceDeployment(ctx context.Context, app string, version string) Span

	TraceRollback(ctx context.Context, app string, fromVersion, toVersion string) Span

	TraceHealthEndpoint(ctx context.Context, endpoint string) Span
}

type Writer interface {
	Write(data *SpanData) error

	WriteEvent(event Event) error

	Flush() error

	Close() error
}

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
