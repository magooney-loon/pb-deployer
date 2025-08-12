package tracer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// DefaultFactory is the default tracer factory implementation
type DefaultFactory struct {
	config   Config
	provider Provider
}

// NewFactory creates a new tracer factory
func NewFactory(config Config) Factory {
	return &DefaultFactory{
		config: config,
	}
}

// Create creates a new tracer with the given name
func (f *DefaultFactory) Create(name string) Tracer {
	return NewTracer(name, f.config)
}

// CreateWithConfig creates a new tracer with specific configuration
func (f *DefaultFactory) CreateWithConfig(name string, config Config) Tracer {
	return NewTracer(name, config)
}

// DefaultProvider manages tracer instances
type DefaultProvider struct {
	tracers map[string]Tracer
	hooks   []Hook
	mu      sync.RWMutex
}

// NewProvider creates a new tracer provider
func NewProvider() Provider {
	return &DefaultProvider{
		tracers: make(map[string]Tracer),
		hooks:   make([]Hook, 0),
	}
}

// GetTracer returns a tracer for the given component
func (p *DefaultProvider) GetTracer(component string) Tracer {
	p.mu.RLock()
	tracer, exists := p.tracers[component]
	p.mu.RUnlock()

	if exists {
		return tracer
	}

	// Create new tracer with default config
	config := DefaultConfig()
	tracer = NewTracer(component, config)

	p.mu.Lock()
	p.tracers[component] = tracer
	p.mu.Unlock()

	return tracer
}

// SetDefaultTracer sets the default tracer
func (p *DefaultProvider) SetDefaultTracer(tracer Tracer) {
	SetDefault(tracer)
}

// RegisterHook registers a hook with all tracers
func (p *DefaultProvider) RegisterHook(hook Hook) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.hooks = append(p.hooks, hook)

	// Register with existing tracers
	// Note: Since tracer is an interface, we can't directly access hooks
	// This would need to be handled through a method if hook registration
	// is needed for existing tracers
}

// Shutdown shuts down all tracers
func (p *DefaultProvider) Shutdown(ctx context.Context) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var errors []error
	for name, tracer := range p.tracers {
		if err := tracer.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close tracer %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}

// DefaultConfig returns the default tracer configuration
func DefaultConfig() Config {
	return Config{
		Level:            LevelInfo,
		Sampler:          NewAlwaysSampler(),
		Writer:           os.Stdout,
		EnableColors:     true,
		EnableStackTrace: false,
		BufferSize:       1000,
		FlushInterval:    30,
	}
}

// Builder provides a fluent interface for building tracer configurations
type Builder struct {
	config Config
}

// NewBuilder creates a new configuration builder
func NewBuilder() *Builder {
	return &Builder{
		config: DefaultConfig(),
	}
}

// WithLevel sets the trace level
func (b *Builder) WithLevel(level Level) *Builder {
	b.config.Level = level
	return b
}

// WithSampler sets the sampler
func (b *Builder) WithSampler(sampler Sampler) *Builder {
	b.config.Sampler = sampler
	return b
}

// WithExporter sets the exporter
func (b *Builder) WithExporter(exporter Exporter) *Builder {
	b.config.Exporter = exporter
	return b
}

// WithWriter sets the output writer
func (b *Builder) WithWriter(writer io.Writer) *Builder {
	b.config.Writer = writer
	return b
}

// WithColors enables or disables colored output
func (b *Builder) WithColors(enable bool) *Builder {
	b.config.EnableColors = enable
	return b
}

// WithStackTrace enables or disables stack traces
func (b *Builder) WithStackTrace(enable bool) *Builder {
	b.config.EnableStackTrace = enable
	return b
}

// WithServiceInfo sets service information
func (b *Builder) WithServiceInfo(name, version, environment string) *Builder {
	b.config.ServiceName = name
	b.config.ServiceVersion = version
	b.config.Environment = environment
	return b
}

// Build builds the configuration
func (b *Builder) Build() Config {
	return b.config
}

// CreateTracer creates a tracer with the built configuration
func (b *Builder) CreateTracer(name string) Tracer {
	return NewTracer(name, b.config)
}

// TunnelTracerFactory creates specialized tracers for the tunnel package
type TunnelTracerFactory struct {
	baseConfig Config
	provider   Provider
}

// NewTunnelTracerFactory creates a factory for tunnel-specific tracers
func NewTunnelTracerFactory(config Config) *TunnelTracerFactory {
	return &TunnelTracerFactory{
		baseConfig: config,
		provider:   NewProvider(),
	}
}

// CreateSSHTracer creates an SSH-specific tracer
func (f *TunnelTracerFactory) CreateSSHTracer() SSHTracer {
	base := f.provider.GetTracer("ssh")
	return NewSSHTracer(base)
}

// CreatePoolTracer creates a pool-specific tracer
func (f *TunnelTracerFactory) CreatePoolTracer() PoolTracer {
	base := f.provider.GetTracer("pool")
	return NewPoolTracer(base)
}

// CreateSecurityTracer creates a security-specific tracer
func (f *TunnelTracerFactory) CreateSecurityTracer() SecurityTracer {
	base := f.provider.GetTracer("security")
	return NewSecurityTracer(base)
}

// CreateServiceTracer creates a service-specific tracer
func (f *TunnelTracerFactory) CreateServiceTracer() ServiceTracer {
	base := f.provider.GetTracer("service")
	return NewServiceTracer(base)
}

// CreateExecutorTracer creates an executor tracer
func (f *TunnelTracerFactory) CreateExecutorTracer() Tracer {
	return f.provider.GetTracer("executor")
}

// CreateSetupManagerTracer creates a setup manager tracer
func (f *TunnelTracerFactory) CreateSetupManagerTracer() Tracer {
	return f.provider.GetTracer("setup")
}

// Shutdown shuts down all tracers
func (f *TunnelTracerFactory) Shutdown(ctx context.Context) error {
	return f.provider.Shutdown(ctx)
}

// QuickSetup provides quick setup functions for common configurations

// SetupConsoleTracing sets up console tracing with sensible defaults
func SetupConsoleTracing(level Level) *TunnelTracerFactory {
	formatter := NewConsoleFormatter()
	exporter := NewWriterExporter(os.Stdout, formatter)

	config := NewBuilder().
		WithLevel(level).
		WithExporter(exporter).
		WithColors(true).
		Build()

	return NewTunnelTracerFactory(config)
}

// SetupJSONTracing sets up JSON tracing for structured logging
func SetupJSONTracing(writer io.Writer, level Level) *TunnelTracerFactory {
	formatter := NewJSONFormatter(false)
	exporter := NewWriterExporter(writer, formatter)

	config := NewBuilder().
		WithLevel(level).
		WithExporter(exporter).
		WithColors(false).
		Build()

	return NewTunnelTracerFactory(config)
}

// SetupFileTracing sets up file-based tracing
func SetupFileTracing(filename string, level Level) (*TunnelTracerFactory, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}

	formatter := NewCompactFormatter()
	exporter := NewWriterExporter(file, formatter)

	config := NewBuilder().
		WithLevel(level).
		WithExporter(exporter).
		WithWriter(file).
		WithColors(false).
		Build()

	return NewTunnelTracerFactory(config), nil
}

// SetupDualTracing sets up both console and file tracing
func SetupDualTracing(filename string, level Level) (*TunnelTracerFactory, error) {
	// Open file for file output
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}

	// Create formatters
	consoleFormatter := NewConsoleFormatter()
	fileFormatter := NewCompactFormatter()

	// Create multi-writer
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Create multi-formatter that uses console format for stdout and compact for file
	multiFormatter := NewMultiFormatter(consoleFormatter, fileFormatter)

	// Create exporter
	exporter := NewWriterExporter(multiWriter, multiFormatter)

	config := NewBuilder().
		WithLevel(level).
		WithExporter(exporter).
		WithColors(true).
		Build()

	return NewTunnelTracerFactory(config), nil
}

// SetupDevelopmentTracing sets up tracing optimized for development
func SetupDevelopmentTracing() *TunnelTracerFactory {
	formatter := NewConsoleFormatter()
	formatter.ShowCaller = true
	exporter := NewWriterExporter(os.Stdout, formatter)

	config := NewBuilder().
		WithLevel(LevelDebug).
		WithExporter(exporter).
		WithColors(true).
		WithStackTrace(true).
		Build()

	return NewTunnelTracerFactory(config)
}

// SetupProductionTracing sets up tracing optimized for production
func SetupProductionTracing(writer io.Writer) *TunnelTracerFactory {
	formatter := NewJSONFormatter(false)
	exporter := NewWriterExporter(writer, formatter)

	// Use probability sampling in production to reduce overhead
	sampler := NewProbabilitySampler(0.1) // Sample 10% of traces

	config := NewBuilder().
		WithLevel(LevelInfo).
		WithSampler(sampler).
		WithExporter(exporter).
		WithColors(false).
		WithStackTrace(false).
		Build()

	return NewTunnelTracerFactory(config)
}

// SetupTestTracing sets up tracing for tests
func SetupTestTracing(t TestReporter) *TunnelTracerFactory {
	// TestReporter interface that test frameworks can implement
	formatter := NewCompactFormatter()
	exporter := NewWriterExporter(NewTestWriter(t), formatter)

	config := NewBuilder().
		WithLevel(LevelDebug).
		WithExporter(exporter).
		WithColors(false).
		Build()

	return NewTunnelTracerFactory(config)
}

// TestReporter is an interface for test reporting
type TestReporter interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

// TestWriter wraps a TestReporter as an io.Writer
type TestWriter struct {
	t TestReporter
}

// NewTestWriter creates a new test writer
func NewTestWriter(t TestReporter) *TestWriter {
	return &TestWriter{t: t}
}

// Write implements io.Writer
func (w *TestWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

// GlobalSetup provides global tracer setup

var (
	globalFactory *TunnelTracerFactory
	globalMu      sync.RWMutex
)

// InitGlobalTracer initializes the global tracer factory
func InitGlobalTracer(factory *TunnelTracerFactory) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalFactory = factory
}

// GetGlobalFactory returns the global tracer factory
func GetGlobalFactory() *TunnelTracerFactory {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalFactory == nil {
		// Return a no-op factory if not initialized
		return NewTunnelTracerFactory(Config{
			Sampler: NewNeverSampler(),
		})
	}

	return globalFactory
}

// MustSetupTracing sets up tracing and panics on error
func MustSetupTracing(setup func() (*TunnelTracerFactory, error)) *TunnelTracerFactory {
	factory, err := setup()
	if err != nil {
		panic(fmt.Sprintf("failed to setup tracing: %v", err))
	}
	InitGlobalTracer(factory)
	return factory
}
