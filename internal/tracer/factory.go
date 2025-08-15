package tracer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

type DefaultFactory struct {
	config   Config
	provider Provider
}

func NewFactory(config Config) Factory {
	return &DefaultFactory{
		config: config,
	}
}

func (f *DefaultFactory) Create(name string) Tracer {
	return NewTracer(name, f.config)
}

func (f *DefaultFactory) CreateWithConfig(name string, config Config) Tracer {
	return NewTracer(name, config)
}

type DefaultProvider struct {
	tracers map[string]Tracer
	hooks   []Hook
	mu      sync.RWMutex
}

func NewProvider() Provider {
	return &DefaultProvider{
		tracers: make(map[string]Tracer),
		hooks:   make([]Hook, 0),
	}
}

func (p *DefaultProvider) GetTracer(component string) Tracer {
	p.mu.RLock()
	tracer, exists := p.tracers[component]
	p.mu.RUnlock()

	if exists {
		return tracer
	}

	config := DefaultConfig()
	tracer = NewTracer(component, config)

	p.mu.Lock()
	p.tracers[component] = tracer
	p.mu.Unlock()

	return tracer
}

func (p *DefaultProvider) SetDefaultTracer(tracer Tracer) {
	SetDefault(tracer)
}

func (p *DefaultProvider) RegisterHook(hook Hook) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.hooks = append(p.hooks, hook)

	// Register with existing tracers
	// Note: Since tracer is an interface, we can't directly access hooks
	// This would need to be handled through a method if hook registration
	// is needed for existing tracers
}

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

type Builder struct {
	config Config
}

func NewBuilder() *Builder {
	return &Builder{
		config: DefaultConfig(),
	}
}

func (b *Builder) WithLevel(level Level) *Builder {
	b.config.Level = level
	return b
}

func (b *Builder) WithSampler(sampler Sampler) *Builder {
	b.config.Sampler = sampler
	return b
}

func (b *Builder) WithExporter(exporter Exporter) *Builder {
	b.config.Exporter = exporter
	return b
}

func (b *Builder) WithWriter(writer io.Writer) *Builder {
	b.config.Writer = writer
	return b
}

func (b *Builder) WithColors(enable bool) *Builder {
	b.config.EnableColors = enable
	return b
}

func (b *Builder) WithStackTrace(enable bool) *Builder {
	b.config.EnableStackTrace = enable
	return b
}

func (b *Builder) WithServiceInfo(name, version, environment string) *Builder {
	b.config.ServiceName = name
	b.config.ServiceVersion = version
	b.config.Environment = environment
	return b
}

func (b *Builder) Build() Config {
	return b.config
}

func (b *Builder) CreateTracer(name string) Tracer {
	return NewTracer(name, b.config)
}

type TunnelTracerFactory struct {
	baseConfig Config
	provider   Provider
}

func NewTunnelTracerFactory(config Config) *TunnelTracerFactory {
	return &TunnelTracerFactory{
		baseConfig: config,
		provider:   NewProvider(),
	}
}

func (f *TunnelTracerFactory) CreateSSHTracer() SSHTracer {
	base := f.provider.GetTracer("ssh")
	return NewSSHTracer(base)
}

func (f *TunnelTracerFactory) CreatePoolTracer() PoolTracer {
	base := f.provider.GetTracer("pool")
	return NewPoolTracer(base)
}

func (f *TunnelTracerFactory) CreateSecurityTracer() SecurityTracer {
	base := f.provider.GetTracer("security")
	return NewSecurityTracer(base)
}

func (f *TunnelTracerFactory) CreateServiceTracer() ServiceTracer {
	base := f.provider.GetTracer("service")
	return NewServiceTracer(base)
}

func (f *TunnelTracerFactory) CreateExecutorTracer() Tracer {
	return f.provider.GetTracer("executor")
}

func (f *TunnelTracerFactory) CreateSetupManagerTracer() Tracer {
	return f.provider.GetTracer("setup")
}

func (f *TunnelTracerFactory) Shutdown(ctx context.Context) error {
	return f.provider.Shutdown(ctx)
}

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

func SetupDualTracing(filename string, level Level) (*TunnelTracerFactory, error) {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}

	consoleFormatter := NewConsoleFormatter()
	fileFormatter := NewCompactFormatter()

	multiWriter := io.MultiWriter(os.Stdout, file)

	multiFormatter := NewMultiFormatter(consoleFormatter, fileFormatter)

	exporter := NewWriterExporter(multiWriter, multiFormatter)

	config := NewBuilder().
		WithLevel(level).
		WithExporter(exporter).
		WithColors(true).
		Build()

	return NewTunnelTracerFactory(config), nil
}

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

type TestReporter interface {
	Log(args ...any)
	Logf(format string, args ...any)
}

type TestWriter struct {
	t TestReporter
}

func NewTestWriter(t TestReporter) *TestWriter {
	return &TestWriter{t: t}
}

func (w *TestWriter) Write(p []byte) (n int, err error) {
	w.t.Log(string(p))
	return len(p), nil
}

var (
	globalFactory *TunnelTracerFactory
	globalMu      sync.RWMutex
)

func InitGlobalTracer(factory *TunnelTracerFactory) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalFactory = factory
}

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

func MustSetupTracing(setup func() (*TunnelTracerFactory, error)) *TunnelTracerFactory {
	factory, err := setup()
	if err != nil {
		panic(fmt.Sprintf("failed to setup tracing: %v", err))
	}
	InitGlobalTracer(factory)
	return factory
}
