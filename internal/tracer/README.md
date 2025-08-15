# Tracer Package

Modern tracing and observability library with dependency injection for SSH operations.

## Features

- **Dependency Injection**: Clean architecture, no singletons
- **Context Propagation**: Traces flow through operations
- **Multiple Formatters**: Console, JSON, compact output
- **Sampling Control**: Probability and custom sampling strategies
- **Performance Optimized**: Zero-allocation paths, async export
- **Domain-Specific**: SSH, Pool, Security, Service specialized tracers
- **Rich Metadata**: Structured fields and events
- **Error Correlation**: Automatic error recording and correlation

## Core Interfaces

```go
// Main tracer interface
type Tracer interface {
    StartSpan(ctx context.Context, operation string) Span
    WithFields(fields Fields) Tracer
    WithLevel(level Level) Tracer
}

// Span operations
type Span interface {
    SetField(key string, value any)
    SetFields(fields Fields)
    Event(name string, fields ...Field)
    StartChild(operation string) Span
    End()
    EndWithError(err error)
}

// Specialized tracers
type SSHTracer interface {
    TraceConnection(ctx context.Context, host string, port int, username string) Span
    TraceCommand(ctx context.Context, cmd string, sudo bool) Span
    TraceFileTransfer(ctx context.Context, source, dest string, size int64) Span
}

type PoolTracer interface {
    TraceGet(ctx context.Context, key string) Span
    TraceRelease(ctx context.Context, key string) Span
    TraceHealthCheck(ctx context.Context) Span
    TraceCleanup(ctx context.Context) Span
}

type SecurityTracer interface {
    TraceFirewallRule(ctx context.Context, rule, action string) Span
    TraceAuthAttempt(ctx context.Context, method, user string) Span
    TraceAuditEvent(ctx context.Context, event string, fields Fields) Span
}

type ServiceTracer interface {
    TraceDeployment(ctx context.Context, app, version string) Span
    TraceServiceAction(ctx context.Context, service, action string) Span
}
```

## Quick Start

```go
// Setup
factory := tracer.SetupProductionTracing(os.Stdout)
defer factory.Shutdown(context.Background())

// Get specialized tracers
sshTracer := factory.CreateSSHTracer()
poolTracer := factory.CreatePoolTracer()
securityTracer := factory.CreateSecurityTracer()
serviceTracer := factory.CreateServiceTracer()

// Basic tracing
span := sshTracer.TraceConnection(ctx, "server.com", 22, "deploy")
span.SetField("ssh.key_type", "ed25519")
span.Event("auth_success")
defer span.End()

// Command execution
cmdSpan := sshTracer.TraceCommand(ctx, "systemctl restart nginx", true)
tracer.RecordCommandResult(cmdSpan, exitCode, lineCount, duration)
defer cmdSpan.End()

// Pool operations
getSpan := poolTracer.TraceGet(ctx, "server-1")
tracer.RecordPoolHealth(getSpan, 10, 8, 2)
defer getSpan.End()
```

## Factory Setup

```go
// Development (console output)
factory := tracer.SetupDevelopmentTracing()

// Production (JSON output)
factory := tracer.SetupProductionTracing(os.Stdout)

// File output
factory, err := tracer.SetupFileTracing("/var/log/tunnel.trace", tracer.LevelInfo)

// Custom configuration
config := tracer.NewBuilder().
    WithLevel(tracer.LevelDebug).
    WithSampler(tracer.NewProbabilitySampler(0.1)).
    WithColors(true).
    Build()
factory := tracer.NewTunnelTracerFactory(config)
```

## Utility Functions

```go
// Record metrics
tracer.RecordConnectionStats(span, stats)
tracer.RecordSSHMetrics(span, metrics)
tracer.RecordRetry(span, attempt, maxAttempts, delay)

// Field helpers
tracer.String("key", "value")
tracer.Int("count", 42)
tracer.Bool("success", true)
tracer.Duration("latency", 100*time.Millisecond)

// Error handling
if err != nil {
    tracer.RecordError(span, err, "operation failed")
    span.EndWithError(err)
    return err
}
```

## Testing

```go
func TestWithTracing(t *testing.T) {
    factory := tracer.SetupTestTracing(t)
    defer factory.Shutdown(context.Background())

    sshTracer := factory.CreateSSHTracer()
    span := sshTracer.TraceConnection(ctx, "localhost", 22, "test")
    defer span.End()

    // Test operations...
}

// Mock tracer
type mockTracer struct{}
func (m *mockTracer) StartSpan(ctx context.Context, op string) tracer.Span {
    return tracer.NewNoOpSpan()
}
```
