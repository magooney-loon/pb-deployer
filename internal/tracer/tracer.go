package tracer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Default tracer instance (NOT a singleton - for convenience only)
var defaultTracer Tracer

// SetDefault sets the default tracer (for testing/convenience)
func SetDefault(t Tracer) {
	defaultTracer = t
}

// GetDefault gets the default tracer
func GetDefault() Tracer {
	if defaultTracer == nil {
		return NewNoOpTracer()
	}
	return defaultTracer
}

// Implementation of the core Tracer interface
type tracer struct {
	name       string
	level      Level
	sampler    Sampler
	exporter   Exporter
	logger     Logger
	fields     Fields
	hooks      []Hook
	writer     io.Writer
	bufferSize int
	mu         sync.RWMutex
	wg         sync.WaitGroup
	closed     int32
}

// NewTracer creates a new tracer instance
func NewTracer(name string, config Config) Tracer {
	t := &tracer{
		name:       name,
		level:      config.Level,
		sampler:    config.Sampler,
		exporter:   config.Exporter,
		fields:     make(Fields),
		hooks:      make([]Hook, 0),
		writer:     config.Writer,
		bufferSize: config.BufferSize,
	}

	if t.sampler == nil {
		t.sampler = NewAlwaysSampler()
	}

	if t.writer == nil {
		t.writer = os.Stdout
	}

	if t.bufferSize <= 0 {
		t.bufferSize = 1000
	}

	// Create logger
	t.logger = newTracerLogger(t)

	return t
}

// StartSpan starts a new span for an operation
func (t *tracer) StartSpan(ctx context.Context, operation string) Span {
	if atomic.LoadInt32(&t.closed) == 1 {
		return newNoOpSpan()
	}

	// Extract parent context if exists
	parentCtx := extractSpanContext(ctx)

	// Generate new span and trace IDs
	spanID := generateSpanID()
	traceID := parentCtx.TraceID
	if traceID == (TraceID{}) {
		traceID = generateTraceID()
	}

	// Check if we should sample
	sampled := t.sampler.ShouldSample(ctx, operation, spanID, parentCtx.SpanID)

	span := &span{
		tracer:       t,
		spanID:       spanID,
		traceID:      traceID,
		parentSpanID: parentCtx.SpanID,
		operation:    operation,
		startTime:    time.Now(),
		fields:       make(Fields),
		events:       make([]Event, 0),
		sampled:      sampled,
		status:       StatusOK,
	}

	// Copy tracer fields
	t.mu.RLock()
	maps.Copy(span.fields, t.fields)
	t.mu.RUnlock()

	// Create context with span
	span.ctx = context.WithValue(ctx, spanContextKey{}, span)

	// Call hooks
	t.callOnSpanStart(span.toSpanData())

	t.wg.Add(1)

	return span
}

// WithField adds a field to all future spans
func (t *tracer) WithField(key string, value any) Tracer {
	t.mu.Lock()
	defer t.mu.Unlock()

	newTracer := &tracer{
		name:       t.name,
		level:      t.level,
		sampler:    t.sampler,
		exporter:   t.exporter,
		logger:     t.logger,
		fields:     make(Fields),
		hooks:      t.hooks,
		writer:     t.writer,
		bufferSize: t.bufferSize,
	}

	// Copy existing fields
	maps.Copy(newTracer.fields, t.fields)
	newTracer.fields[key] = value

	return newTracer
}

// WithFields adds multiple fields to all future spans
func (t *tracer) WithFields(fields Fields) Tracer {
	t.mu.Lock()
	defer t.mu.Unlock()

	newTracer := &tracer{
		name:       t.name,
		level:      t.level,
		sampler:    t.sampler,
		exporter:   t.exporter,
		logger:     t.logger,
		fields:     make(Fields),
		hooks:      t.hooks,
		writer:     t.writer,
		bufferSize: t.bufferSize,
	}

	// Copy existing fields
	maps.Copy(newTracer.fields, t.fields)
	maps.Copy(newTracer.fields, fields)

	return newTracer
}

// SetLevel sets the minimum trace level
func (t *tracer) SetLevel(level Level) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.level = level
}

// Close closes the tracer and flushes any pending data
func (t *tracer) Close() error {
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		return nil
	}

	// Wait for all spans to complete
	t.wg.Wait()

	// Flush exporter if present
	if t.exporter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.exporter.Flush(ctx); err != nil {
			return fmt.Errorf("failed to flush exporter: %w", err)
		}

		if err := t.exporter.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown exporter: %w", err)
		}
	}

	return nil
}

// callOnSpanStart calls all hooks for span start
func (t *tracer) callOnSpanStart(spanData *SpanData) {
	t.mu.RLock()
	hooks := t.hooks
	t.mu.RUnlock()

	for _, hook := range hooks {
		hook.OnSpanStart(spanData)
	}
}

// callOnSpanEnd calls all hooks for span end
func (t *tracer) callOnSpanEnd(spanData *SpanData) {
	t.mu.RLock()
	hooks := t.hooks
	t.mu.RUnlock()

	for _, hook := range hooks {
		hook.OnSpanEnd(spanData)
	}
}

// Span implementation
type span struct {
	tracer       *tracer
	spanID       SpanID
	traceID      TraceID
	parentSpanID SpanID
	operation    string
	startTime    time.Time
	endTime      time.Time
	duration     time.Duration
	status       Status
	fields       Fields
	events       []Event
	error        error
	sampled      bool
	ctx          context.Context
	mu           sync.Mutex
	ended        int32
}

// End marks the span as complete
func (s *span) End() {
	s.endWithStatus(StatusOK, nil)
}

// EndWithError marks the span as complete with an error
func (s *span) EndWithError(err error) {
	s.endWithStatus(StatusError, err)
}

// endWithStatus is the internal method to end a span
func (s *span) endWithStatus(status Status, err error) {
	if !atomic.CompareAndSwapInt32(&s.ended, 0, 1) {
		return
	}

	s.mu.Lock()
	s.endTime = time.Now()
	s.duration = s.endTime.Sub(s.startTime)
	s.status = status
	s.error = err
	s.mu.Unlock()

	// Export if sampled
	if s.sampled && s.tracer.exporter != nil {
		spanData := s.toSpanData()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = s.tracer.exporter.Export(ctx, spanData)
	}

	// Call hooks
	s.tracer.callOnSpanEnd(s.toSpanData())

	// Log if appropriate level
	s.logSpanEnd()

	s.tracer.wg.Done()
}

// SetStatus sets the span status
func (s *span) SetStatus(status Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status = status
}

// SetField adds a field to this span
func (s *span) SetField(key string, value any) Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fields[key] = value
	return s
}

// SetFields adds multiple fields to this span
func (s *span) SetFields(fields Fields) Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range fields {
		s.fields[k] = v
	}
	return s
}

// Event logs an event within the span
func (s *span) Event(name string, fields ...Field) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event := Event{
		Name:      name,
		Timestamp: time.Now(),
		Fields:    make(Fields),
	}

	for _, field := range fields {
		event.Fields[field.Key] = field.Value
	}

	s.events = append(s.events, event)
}

// StartChild starts a child span
func (s *span) StartChild(operation string) Span {
	return s.tracer.StartSpan(s.ctx, operation)
}

// Context returns the context with span information
func (s *span) Context() context.Context {
	return s.ctx
}

// toSpanData converts span to SpanData
func (s *span) toSpanData() *SpanData {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Copy fields to avoid race conditions
	fields := make(Fields)
	maps.Copy(fields, s.fields)

	// Copy events
	events := make([]Event, len(s.events))
	copy(events, s.events)

	return &SpanData{
		SpanID:       s.spanID,
		ParentSpanID: s.parentSpanID,
		TraceID:      s.traceID,
		Operation:    s.operation,
		StartTime:    s.startTime,
		EndTime:      s.endTime,
		Duration:     s.duration,
		Status:       s.status,
		Fields:       fields,
		Events:       events,
		Error:        s.error,
	}
}

// logSpanEnd logs span completion if appropriate
func (s *span) logSpanEnd() {
	level := LevelDebug
	if s.status == StatusError {
		level = LevelError
	}

	if level < s.tracer.level {
		return
	}

	fields := make([]Field, 0)
	fields = append(fields,
		String("operation", s.operation),
		Duration("duration", s.duration),
		String("status", s.status.String()),
		String("trace_id", hex.EncodeToString(s.traceID[:])),
		String("span_id", hex.EncodeToString(s.spanID[:])),
	)

	if s.parentSpanID != (SpanID{}) {
		fields = append(fields, String("parent_span_id", hex.EncodeToString(s.parentSpanID[:])))
	}

	// Add span fields
	for k, v := range s.fields {
		fields = append(fields, Any(k, v))
	}

	if s.error != nil {
		fields = append(fields, Error(s.error))
	}

	msg := fmt.Sprintf("Span completed: %s", s.operation)
	s.tracer.logger.Info(msg, fields...)
}

// Context key for span storage
type spanContextKey struct{}

// extractSpanContext extracts span context from context
func extractSpanContext(ctx context.Context) SpanContext {
	if span, ok := ctx.Value(spanContextKey{}).(*span); ok {
		return SpanContext{
			TraceID:      span.traceID,
			SpanID:       span.spanID,
			ParentSpanID: span.parentSpanID,
			Sampled:      span.sampled,
		}
	}
	return SpanContext{}
}

// SpanFromContext returns the span from context
func SpanFromContext(ctx context.Context) Span {
	if span, ok := ctx.Value(spanContextKey{}).(*span); ok {
		return span
	}
	return newNoOpSpan()
}

// generateSpanID generates a new span ID
func generateSpanID() SpanID {
	var id SpanID
	rand.Read(id[:])
	return id
}

// generateTraceID generates a new trace ID
func generateTraceID() TraceID {
	var id TraceID
	rand.Read(id[:])
	return id
}

// NoOp implementations for when tracing is disabled

type noOpTracer struct{}

func NewNoOpTracer() Tracer {
	return &noOpTracer{}
}

func (t *noOpTracer) StartSpan(ctx context.Context, operation string) Span {
	return newNoOpSpan()
}

func (t *noOpTracer) WithField(key string, value any) Tracer {
	return t
}

func (t *noOpTracer) WithFields(fields Fields) Tracer {
	return t
}

func (t *noOpTracer) SetLevel(level Level) {}

func (t *noOpTracer) Close() error {
	return nil
}

type noOpSpan struct{}

func newNoOpSpan() Span {
	return &noOpSpan{}
}

func (s *noOpSpan) End()                                {}
func (s *noOpSpan) EndWithError(err error)              {}
func (s *noOpSpan) SetStatus(status Status)             {}
func (s *noOpSpan) SetField(key string, value any) Span { return s }
func (s *noOpSpan) SetFields(fields Fields) Span        { return s }
func (s *noOpSpan) Event(name string, fields ...Field)  {}
func (s *noOpSpan) StartChild(operation string) Span    { return s }
func (s *noOpSpan) Context() context.Context            { return context.Background() }

// Sampler implementations

type alwaysSampler struct{}

func NewAlwaysSampler() Sampler {
	return &alwaysSampler{}
}

func (s *alwaysSampler) ShouldSample(ctx context.Context, operation string, spanID SpanID, parentSpanID SpanID) bool {
	return true
}

type neverSampler struct{}

func NewNeverSampler() Sampler {
	return &neverSampler{}
}

func (s *neverSampler) ShouldSample(ctx context.Context, operation string, spanID SpanID, parentSpanID SpanID) bool {
	return false
}

type probabilitySampler struct {
	probability float64
}

func NewProbabilitySampler(probability float64) Sampler {
	if probability >= 1.0 {
		return NewAlwaysSampler()
	}
	if probability <= 0.0 {
		return NewNeverSampler()
	}
	return &probabilitySampler{probability: probability}
}

func (s *probabilitySampler) ShouldSample(ctx context.Context, operation string, spanID SpanID, parentSpanID SpanID) bool {
	// Use first 8 bytes of span ID as random value
	var val uint64
	for i := range 8 {
		val = (val << 8) | uint64(spanID[i])
	}
	// Simple probability check
	return float64(val)/float64(^uint64(0)) < s.probability
}

// tracerLogger implements Logger for the tracer
type tracerLogger struct {
	tracer *tracer
	fields Fields
}

func newTracerLogger(t *tracer) Logger {
	return &tracerLogger{
		tracer: t,
		fields: make(Fields),
	}
}

func (l *tracerLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, fields...)
}

func (l *tracerLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, fields...)
}

func (l *tracerLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, fields...)
}

func (l *tracerLogger) Error(msg string, fields ...Field) {
	l.log(LevelError, msg, fields...)
}

func (l *tracerLogger) Fatal(msg string, fields ...Field) {
	l.log(LevelFatal, msg, fields...)
	os.Exit(1)
}

func (l *tracerLogger) WithField(key string, value any) Logger {
	newLogger := &tracerLogger{
		tracer: l.tracer,
		fields: make(Fields),
	}
	maps.Copy(newLogger.fields, l.fields)
	newLogger.fields[key] = value
	return newLogger
}

func (l *tracerLogger) WithFields(fields Fields) Logger {
	newLogger := &tracerLogger{
		tracer: l.tracer,
		fields: make(Fields),
	}
	maps.Copy(newLogger.fields, l.fields)
	maps.Copy(newLogger.fields, fields)
	return newLogger
}

func (l *tracerLogger) WithError(err error) Logger {
	return l.WithField("error", err.Error())
}

func (l *tracerLogger) WithSpan(span Span) Logger {
	// Since span is an interface, we can't access internal fields directly
	// We'll add span context through the span's context method
	ctx := span.Context()
	spanCtx := extractSpanContext(ctx)
	if spanCtx.TraceID != (TraceID{}) {
		return l.WithFields(Fields{
			"trace_id": hex.EncodeToString(spanCtx.TraceID[:]),
			"span_id":  hex.EncodeToString(spanCtx.SpanID[:]),
		})
	}
	return l
}

func (l *tracerLogger) log(level Level, msg string, fields ...Field) {
	if level < l.tracer.level {
		return
	}

	// Build log entry
	entry := map[string]any{
		"timestamp": time.Now(),
		"level":     level.String(),
		"message":   msg,
		"component": l.tracer.name,
	}

	// Add logger fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Add passed fields
	for _, field := range fields {
		entry[field.Key] = field.Value
	}

	// Add caller information
	if pc, file, line, ok := runtime.Caller(2); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			parts := strings.Split(file, "/")
			if len(parts) > 0 {
				file = parts[len(parts)-1]
			}
			entry["caller"] = fmt.Sprintf("%s:%d", file, line)
		}
	}

	// Write to output (simplified for now)
	output := fmt.Sprintf("[%s] %s %s",
		entry["timestamp"].(time.Time).Format(time.RFC3339),
		level.String(),
		msg,
	)

	// Add fields
	for k, v := range entry {
		if k != "timestamp" && k != "level" && k != "message" {
			output += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	fmt.Fprintln(l.tracer.writer, output)
}
