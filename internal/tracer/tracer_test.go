package tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewTracer(t *testing.T) {
	config := Config{
		Level:   LevelInfo,
		Sampler: NewAlwaysSampler(),
	}

	tracer := NewTracer("test", config)
	if tracer == nil {
		t.Error("NewTracer should return a non-nil tracer")
	}
}

func TestTracerStartSpan(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	if span == nil {
		t.Error("StartSpan should return a non-nil span")
	}

	span.End()
}

func TestSpanFields(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	// Test SetField
	span.SetField("key1", "value1")
	span.SetField("key2", 42)

	// Test SetFields
	fields := Fields{
		"key3": true,
		"key4": 3.14,
	}
	span.SetFields(fields)

	span.End()
}

func TestSpanEvents(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	// Test Event
	span.Event("test_event")
	span.Event("test_event_with_fields",
		String("key1", "value1"),
		Int("key2", 42),
		Bool("key3", true),
	)

	span.End()
}

func TestSpanStatus(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	// Test SetStatus
	span.SetStatus(StatusError)
	span.End()

	// Test EndWithError
	span2 := tracer.StartSpan(ctx, "test.operation2")
	testErr := errors.New("test error")
	span2.EndWithError(testErr)
}

func TestSpanChild(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	ctx := context.Background()
	parentSpan := tracer.StartSpan(ctx, "parent.operation")

	childSpan := parentSpan.StartChild("child.operation")
	if childSpan == nil {
		t.Error("StartChild should return a non-nil span")
	}

	childSpan.End()
	parentSpan.End()
}

func TestTracerWithFields(t *testing.T) {
	config := Config{
		Level:   LevelDebug,
		Sampler: NewAlwaysSampler(),
	}
	tracer := NewTracer("test", config)

	// Test WithField
	tracerWithField := tracer.WithField("global_key", "global_value")
	if tracerWithField == nil {
		t.Error("WithField should return a non-nil tracer")
	}

	// Test WithFields
	globalFields := Fields{
		"service": "test-service",
		"version": "1.0.0",
	}
	tracerWithFields := tracer.WithFields(globalFields)
	if tracerWithFields == nil {
		t.Error("WithFields should return a non-nil tracer")
	}

	ctx := context.Background()
	span := tracerWithFields.StartSpan(ctx, "test.operation")
	span.End()
}

func TestAlwaysSampler(t *testing.T) {
	sampler := NewAlwaysSampler()
	ctx := context.Background()

	result := sampler.ShouldSample(ctx, "test", SpanID{}, SpanID{})
	if !result {
		t.Error("AlwaysSampler should always return true")
	}
}

func TestNeverSampler(t *testing.T) {
	sampler := NewNeverSampler()
	ctx := context.Background()

	result := sampler.ShouldSample(ctx, "test", SpanID{}, SpanID{})
	if result {
		t.Error("NeverSampler should always return false")
	}
}

func TestProbabilitySampler(t *testing.T) {
	// Test edge cases
	alwaysSampler := NewProbabilitySampler(1.0)
	neverSampler := NewProbabilitySampler(0.0)

	ctx := context.Background()
	spanID := SpanID{1, 2, 3, 4, 5, 6, 7, 8}

	if !alwaysSampler.ShouldSample(ctx, "test", spanID, SpanID{}) {
		t.Error("ProbabilitySampler(1.0) should always return true")
	}

	if neverSampler.ShouldSample(ctx, "test", spanID, SpanID{}) {
		t.Error("ProbabilitySampler(0.0) should always return false")
	}

	// Test normal probability
	sampler := NewProbabilitySampler(0.5)
	if sampler == nil {
		t.Error("NewProbabilitySampler should return a non-nil sampler")
	}
}

func TestConsoleFormatter(t *testing.T) {
	formatter := NewConsoleFormatter()
	formatter.EnableColors = false // Disable colors for testing

	spanData := &SpanData{
		TraceID:   TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:    SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Operation: "test.operation",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Status:    StatusOK,
		Fields:    Fields{"key": "value"},
	}

	result, err := formatter.Format(spanData)
	if err != nil {
		t.Errorf("ConsoleFormatter.Format error: %v", err)
	}

	if len(result) == 0 {
		t.Error("ConsoleFormatter should return non-empty result")
	}

	output := string(result)
	if !strings.Contains(output, "test.operation") {
		t.Error("Formatted output should contain operation name")
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := NewJSONFormatter(false)

	spanData := &SpanData{
		TraceID:   TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:    SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Operation: "test.operation",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Status:    StatusOK,
		Fields:    Fields{"key": "value"},
	}

	result, err := formatter.Format(spanData)
	if err != nil {
		t.Errorf("JSONFormatter.Format error: %v", err)
	}

	// Verify it's valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(result, &jsonData); err != nil {
		t.Errorf("JSONFormatter should produce valid JSON: %v", err)
	}

	if jsonData["operation"] != "test.operation" {
		t.Error("JSON output should contain operation field")
	}
}

func TestCompactFormatter(t *testing.T) {
	formatter := NewCompactFormatter()

	spanData := &SpanData{
		TraceID:   TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:    SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Operation: "test.operation",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Status:    StatusOK,
	}

	result, err := formatter.Format(spanData)
	if err != nil {
		t.Errorf("CompactFormatter.Format error: %v", err)
	}

	output := string(result)
	if !strings.Contains(output, "test.operation") {
		t.Error("Compact output should contain operation name")
	}

	if !strings.Contains(output, "OK") {
		t.Error("Compact output should contain status")
	}
}

func TestWriterExporter(t *testing.T) {
	var buf bytes.Buffer
	formatter := NewConsoleFormatter()
	formatter.EnableColors = false

	exporter := NewWriterExporter(&buf, formatter)

	spanData := &SpanData{
		TraceID:   TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:    SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Operation: "test.operation",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(100 * time.Millisecond),
		Duration:  100 * time.Millisecond,
		Status:    StatusOK,
	}

	ctx := context.Background()
	err := exporter.Export(ctx, spanData)
	if err != nil {
		t.Errorf("WriterExporter.Export error: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("WriterExporter should write data to buffer")
	}
}

func TestDefaultFactory(t *testing.T) {
	config := DefaultConfig()
	factory := NewFactory(config)

	tracer := factory.Create("test")
	if tracer == nil {
		t.Error("Factory.Create should return a non-nil tracer")
	}
}

func TestTunnelTracerFactory(t *testing.T) {
	config := DefaultConfig()
	factory := NewTunnelTracerFactory(config)

	// Test SSH tracer
	sshTracer := factory.CreateSSHTracer()
	if sshTracer == nil {
		t.Error("CreateSSHTracer should return a non-nil tracer")
	}

	// Test Pool tracer
	poolTracer := factory.CreatePoolTracer()
	if poolTracer == nil {
		t.Error("CreatePoolTracer should return a non-nil tracer")
	}

	// Test Security tracer
	securityTracer := factory.CreateSecurityTracer()
	if securityTracer == nil {
		t.Error("CreateSecurityTracer should return a non-nil tracer")
	}

	// Test Service tracer
	serviceTracer := factory.CreateServiceTracer()
	if serviceTracer == nil {
		t.Error("CreateServiceTracer should return a non-nil tracer")
	}
}

func TestSSHTracer(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	baseTracer := NewTracer("ssh", config)
	sshTracer := NewSSHTracer(baseTracer)

	ctx := context.Background()

	// Test TraceConnection
	connSpan := sshTracer.TraceConnection(ctx, "server.com", 22, "deploy")
	if connSpan == nil {
		t.Error("TraceConnection should return a non-nil span")
	}
	connSpan.End()

	// Test TraceCommand
	cmdSpan := sshTracer.TraceCommand(ctx, "ls -la", false)
	if cmdSpan == nil {
		t.Error("TraceCommand should return a non-nil span")
	}
	cmdSpan.End()

	// Test TraceFileTransfer
	transferSpan := sshTracer.TraceFileTransfer(ctx, "/local/file", "/remote/file", 1024)
	if transferSpan == nil {
		t.Error("TraceFileTransfer should return a non-nil span")
	}
	transferSpan.End()

	// Test TraceHealthCheck
	healthSpan := sshTracer.TraceHealthCheck(ctx, "server.com")
	if healthSpan == nil {
		t.Error("TraceHealthCheck should return a non-nil span")
	}
	healthSpan.End()
}

func TestPoolTracer(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	baseTracer := NewTracer("pool", config)
	poolTracer := NewPoolTracer(baseTracer)

	ctx := context.Background()

	// Test TraceGet
	getSpan := poolTracer.TraceGet(ctx, "server-1")
	if getSpan == nil {
		t.Error("TraceGet should return a non-nil span")
	}
	getSpan.End()

	// Test TraceRelease
	releaseSpan := poolTracer.TraceRelease(ctx, "server-1")
	if releaseSpan == nil {
		t.Error("TraceRelease should return a non-nil span")
	}
	releaseSpan.End()

	// Test TraceHealthCheck
	healthSpan := poolTracer.TraceHealthCheck(ctx)
	if healthSpan == nil {
		t.Error("TraceHealthCheck should return a non-nil span")
	}
	healthSpan.End()

	// Test TraceCleanup
	cleanupSpan := poolTracer.TraceCleanup(ctx)
	if cleanupSpan == nil {
		t.Error("TraceCleanup should return a non-nil span")
	}
	cleanupSpan.End()
}

func TestSecurityTracer(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	baseTracer := NewTracer("security", config)
	securityTracer := NewSecurityTracer(baseTracer)

	ctx := context.Background()

	// Test TraceSecurityCheck
	checkSpan := securityTracer.TraceSecurityCheck(ctx, "firewall")
	if checkSpan == nil {
		t.Error("TraceSecurityCheck should return a non-nil span")
	}
	checkSpan.End()

	// Test TraceFirewallRule
	ruleSpan := securityTracer.TraceFirewallRule(ctx, "allow port 80", "apply")
	if ruleSpan == nil {
		t.Error("TraceFirewallRule should return a non-nil span")
	}
	ruleSpan.End()

	// Test TraceAuthAttempt
	authSpan := securityTracer.TraceAuthAttempt(ctx, "ssh-key", "deploy")
	if authSpan == nil {
		t.Error("TraceAuthAttempt should return a non-nil span")
	}
	authSpan.End()

	// Test TraceAuditEvent
	auditSpan := securityTracer.TraceAuditEvent(ctx, "login", Fields{"user": "admin"})
	if auditSpan == nil {
		t.Error("TraceAuditEvent should return a non-nil span")
	}
	auditSpan.End()
}

func TestServiceTracer(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	baseTracer := NewTracer("service", config)
	serviceTracer := NewServiceTracer(baseTracer)

	ctx := context.Background()

	// Test TraceServiceAction
	actionSpan := serviceTracer.TraceServiceAction(ctx, "nginx", "restart")
	if actionSpan == nil {
		t.Error("TraceServiceAction should return a non-nil span")
	}
	actionSpan.End()

	// Test TraceDeployment
	deploySpan := serviceTracer.TraceDeployment(ctx, "myapp", "1.2.3")
	if deploySpan == nil {
		t.Error("TraceDeployment should return a non-nil span")
	}
	deploySpan.End()

	// Test TraceRollback
	rollbackSpan := serviceTracer.TraceRollback(ctx, "myapp", "1.2.3", "1.2.2")
	if rollbackSpan == nil {
		t.Error("TraceRollback should return a non-nil span")
	}
	rollbackSpan.End()

	// Test TraceHealthEndpoint
	healthSpan := serviceTracer.TraceHealthEndpoint(ctx, "/health")
	if healthSpan == nil {
		t.Error("TraceHealthEndpoint should return a non-nil span")
	}
	healthSpan.End()
}

func TestNoOpTracer(t *testing.T) {
	tracer := NewNoOpTracer()

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	// All operations should be safe to call
	span.SetField("key", "value")
	span.SetFields(Fields{"key": "value"})
	span.Event("test_event")
	span.SetStatus(StatusError)

	childSpan := span.StartChild("child.operation")
	childSpan.End()

	testErr := errors.New("test error")
	span.EndWithError(testErr)

	if err := tracer.Close(); err != nil {
		t.Errorf("NoOpTracer.Close should not return error: %v", err)
	}
}

func TestContextPropagation(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	tracer := NewTracer("test", config)

	ctx := context.Background()
	parentSpan := tracer.StartSpan(ctx, "parent")

	// Get span from context
	spanFromCtx := SpanFromContext(parentSpan.Context())
	if spanFromCtx == nil {
		t.Error("SpanFromContext should return a non-nil span")
	}

	// Test with empty context
	emptySpan := SpanFromContext(context.Background())
	if emptySpan == nil {
		t.Error("SpanFromContext should return a non-nil span (no-op) for empty context")
	}

	parentSpan.End()
}

func TestUtilityFunctions(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span := tracer.StartSpan(ctx, "test.operation")

	// Test utility functions
	stats := ConnectionStats{
		TotalConnections:    10,
		ActiveConnections:   5,
		FailedConnections:   1,
		TotalCommands:       100,
		FailedCommands:      2,
		AverageResponseTime: 50 * time.Millisecond,
	}
	RecordConnectionStats(span, stats)

	RecordPoolHealth(span, 10, 8, 2)
	RecordCommandResult(span, 0, 25, 100*time.Millisecond)

	testErr := errors.New("test error")
	RecordError(span, testErr, "test context")
	RecordRetry(span, 1, 3, 1*time.Second)

	span.End()
}

func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected string
	}{
		{"String field", String("key", "value"), "key"},
		{"Int field", Int("count", 42), "count"},
		{"Int64 field", Int64("bignum", 9223372036854775807), "bignum"},
		{"Float64 field", Float64("pi", 3.14159), "pi"},
		{"Bool field", Bool("enabled", true), "enabled"},
		{"Time field", Time("timestamp", time.Now()), "timestamp"},
		{"Duration field", Duration("latency", 100*time.Millisecond), "latency"},
		{"Error field", Error(errors.New("test")), "error"},
		{"Any field", Any("data", map[string]int{"a": 1}), "data"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.field.Key != test.expected {
				t.Errorf("Field key mismatch: got %s, expected %s", test.field.Key, test.expected)
			}
		})
	}

	// Test nil error
	nilErrorField := Error(nil)
	if nilErrorField.Value != nil {
		t.Error("Error field with nil should have nil value")
	}
}

func TestLevelAndStatusStrings(t *testing.T) {
	levelTests := []struct {
		level    Level
		expected string
	}{
		{LevelTrace, "TRACE"},
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LevelFatal, "FATAL"},
		{Level(999), "UNKNOWN"},
	}

	for _, test := range levelTests {
		if test.level.String() != test.expected {
			t.Errorf("Level.String() mismatch: got %s, expected %s", test.level.String(), test.expected)
		}
	}

	statusTests := []struct {
		status   Status
		expected string
	}{
		{StatusOK, "OK"},
		{StatusError, "ERROR"},
		{StatusCanceled, "CANCELED"},
		{StatusTimeout, "TIMEOUT"},
		{StatusUnknown, "UNKNOWN"},
		{Status(999), "UNKNOWN"},
	}

	for _, test := range statusTests {
		if test.status.String() != test.expected {
			t.Errorf("Status.String() mismatch: got %s, expected %s", test.status.String(), test.expected)
		}
	}
}

func TestBuilder(t *testing.T) {
	config := NewBuilder().
		WithLevel(LevelWarn).
		WithSampler(NewNeverSampler()).
		WithColors(false).
		WithStackTrace(true).
		WithServiceInfo("test-service", "1.0.0", "test").
		Build()

	if config.Level != LevelWarn {
		t.Error("Builder should set level correctly")
	}

	if config.EnableColors {
		t.Error("Builder should disable colors")
	}

	if !config.EnableStackTrace {
		t.Error("Builder should enable stack trace")
	}

	if config.ServiceName != "test-service" {
		t.Error("Builder should set service name")
	}
}

func TestSetupFunctions(t *testing.T) {
	// Test console tracing
	factory1 := SetupConsoleTracing(LevelInfo)
	if factory1 == nil {
		t.Error("SetupConsoleTracing should return a non-nil factory")
	}

	// Test JSON tracing
	var buf bytes.Buffer
	factory2 := SetupJSONTracing(&buf, LevelInfo)
	if factory2 == nil {
		t.Error("SetupJSONTracing should return a non-nil factory")
	}

	// Test development tracing
	factory3 := SetupDevelopmentTracing()
	if factory3 == nil {
		t.Error("SetupDevelopmentTracing should return a non-nil factory")
	}

	// Test production tracing
	factory4 := SetupProductionTracing(&buf)
	if factory4 == nil {
		t.Error("SetupProductionTracing should return a non-nil factory")
	}
}

func TestTracerClose(t *testing.T) {
	config := DefaultConfig()
	tracer := NewTracer("test", config)

	// Close should be safe to call multiple times
	err1 := tracer.Close()
	err2 := tracer.Close()

	if err1 != nil {
		t.Errorf("First close should not error: %v", err1)
	}

	if err2 != nil {
		t.Errorf("Second close should not error: %v", err2)
	}

	// Operations after close should return no-op spans
	ctx := context.Background()
	span := tracer.StartSpan(ctx, "after.close")
	span.End()
}

func TestMultiFormatter(t *testing.T) {
	formatter1 := NewConsoleFormatter()
	formatter1.EnableColors = false
	formatter2 := NewCompactFormatter()

	multiFormatter := NewMultiFormatter(formatter1, formatter2)

	spanData := &SpanData{
		TraceID:   TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:    SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		Operation: "test.operation",
		StartTime: time.Now(),
		Status:    StatusOK,
	}

	result, err := multiFormatter.Format(spanData)
	if err != nil {
		t.Errorf("MultiFormatter.Format error: %v", err)
	}

	if len(result) == 0 {
		t.Error("MultiFormatter should return non-empty result")
	}
}

func TestTestWriter(t *testing.T) {
	testWriter := NewTestWriter(t)

	testData := []byte("test log message")
	n, err := testWriter.Write(testData)

	if err != nil {
		t.Errorf("TestWriter.Write error: %v", err)
	}

	if n != len(testData) {
		t.Errorf("TestWriter.Write should return correct byte count: got %d, expected %d", n, len(testData))
	}
}

func TestDefaultProvider(t *testing.T) {
	provider := NewProvider()

	// Test GetTracer
	tracer1 := provider.GetTracer("component1")
	tracer2 := provider.GetTracer("component1") // Should return same tracer
	tracer3 := provider.GetTracer("component2") // Should return different tracer

	if tracer1 == nil || tracer2 == nil || tracer3 == nil {
		t.Error("GetTracer should return non-nil tracers")
	}

	// Test SetDefaultTracer
	provider.SetDefaultTracer(tracer1)

	// Test Shutdown
	ctx := context.Background()
	err := provider.Shutdown(ctx)
	if err != nil {
		t.Errorf("Provider.Shutdown error: %v", err)
	}
}

func TestTraceWithTimeout(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	tracer := NewTracer("test", config)

	ctx := context.Background()
	span, timeoutCtx, cancel := TraceWithTimeout(ctx, tracer, "timeout.operation", 100*time.Millisecond)
	defer cancel()

	if span == nil {
		t.Error("TraceWithTimeout should return a non-nil span")
	}

	if timeoutCtx == nil {
		t.Error("TraceWithTimeout should return a non-nil context")
	}

	span.End()
}

func TestStartSSHOperation(t *testing.T) {
	config := DefaultConfig()
	config.Sampler = NewAlwaysSampler()
	baseTracer := NewTracer("ssh", config)
	sshTracer := NewSSHTracer(baseTracer)

	ctx := context.Background()
	span := StartSSHOperation(ctx, sshTracer, "connect", "server.com", "deploy")

	if span == nil {
		t.Error("StartSSHOperation should return a non-nil span")
	}

	span.End()
}

func TestGlobalTracer(t *testing.T) {
	// Test default state
	defaultTracer := GetDefault()
	if defaultTracer == nil {
		t.Error("GetDefault should return a non-nil tracer")
	}
}
