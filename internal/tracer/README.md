# Tracer Package

A modern, dependency-injection based tracing and observability library designed specifically for the tunnel package architecture. Built with clean interfaces, no singletons, and proper separation of concerns.

## Overview

The `tracer` package provides comprehensive tracing, logging, and observability features for the tunnel SSH architecture:

- **Dependency Injection**: No singletons, all tracers are injected
- **Context Propagation**: Traces flow through context across operations
- **Structured Logging**: Rich field support with type safety
- **Multiple Formatters**: Console, JSON, compact formats
- **Sampling**: Control trace volume with various sampling strategies
- **Specialized Tracers**: SSH, Pool, Security, Service-specific tracing

## Architecture

### Core Components

1. **Tracer**: Main interface for starting spans and managing trace context
2. **Span**: Represents a single operation with timing and metadata
3. **Logger**: Structured logging within trace context
4. **Exporter**: Sends trace data to external systems
5. **Formatter**: Formats trace data for output
6. **Sampler**: Determines which traces to record

### Design Principles

- **Interface-First**: All components defined as interfaces
- **Immutable**: Tracers return new instances when modified
- **Context-Aware**: Uses Go context for propagation
- **Zero-Allocation**: Optimized for performance
- **Testable**: Easy to mock and test

## Usage Examples

### Basic Setup

```go
// Development setup with console output
factory := tracer.SetupDevelopmentTracing()
defer factory.Shutdown(context.Background())

// Production setup with JSON output
factory := tracer.SetupProductionTracing(os.Stdout)
defer factory.Shutdown(context.Background())

// File-based tracing
factory, err := tracer.SetupFileTracing("/var/log/tunnel.trace", tracer.LevelInfo)
if err != nil {
    log.Fatal(err)
}
defer factory.Shutdown(context.Background())
```

### SSH Operations

```go
// Get SSH tracer
sshTracer := factory.CreateSSHTracer()

// Trace a connection
span := sshTracer.TraceConnection(ctx, "server.example.com", 22, "deploy")
defer span.End()

// Add fields
span.SetField("ssh.key_type", "ed25519")
span.SetField("ssh.timeout", "30s")

// Log events
span.Event("auth_attempted", 
    tracer.String("method", "publickey"),
    tracer.Bool("success", true),
)

// Handle errors
if err != nil {
    span.EndWithError(err)
    return err
}
```

### Command Execution

```go
// Trace command execution
cmdSpan := sshTracer.TraceCommand(ctx, "systemctl restart nginx", true)
defer cmdSpan.End()

// Record metrics
tracer.RecordCommandResult(cmdSpan, exitCode, lineCount, duration)

// Add command-specific fields
cmdSpan.SetFields(tracer.Fields{
    "command.user": "root",
    "command.env": "production",
    "command.host": "web-01",
})
```

### Connection Pool

```go
// Get pool tracer
poolTracer := factory.CreatePoolTracer()

// Trace getting connection
getSpan := poolTracer.TraceGet(ctx, "server-1")
defer getSpan.End()

// Record pool health
tracer.RecordPoolHealth(getSpan, totalConns, healthyConns, unhealthyConns)

// Trace cleanup
cleanupSpan := poolTracer.TraceCleanup(ctx)
defer cleanupSpan.End()
```

### Security Operations

```go
// Get security tracer
secTracer := factory.CreateSecurityTracer()

// Trace firewall rule
fwSpan := secTracer.TraceFirewallRule(ctx, "allow 443/tcp", "apply")
defer fwSpan.End()

// Trace authentication
authSpan := secTracer.TraceAuthAttempt(ctx, "publickey", "deploy")
defer authSpan.End()

// Audit event
auditSpan := secTracer.TraceAuditEvent(ctx, "ssh_login", tracer.Fields{
    "user": "deploy",
    "from": "10.0.0.5",
    "success": true,
})
defer auditSpan.End()
```

### Service Management

```go
// Get service tracer
svcTracer := factory.CreateServiceTracer()

// Trace deployment
deploySpan := svcTracer.TraceDeployment(ctx, "myapp", "v2.1.0")
defer deploySpan.End()

// Add deployment stages
deploySpan.Event("build_completed", tracer.Duration("duration", buildTime))
deploySpan.Event("tests_passed", tracer.Int("count", testCount))
deploySpan.Event("deployment_started")

// Trace service action
actionSpan := svcTracer.TraceServiceAction(ctx, "nginx", "restart")
defer actionSpan.End()
```

### Child Spans

```go
// Start parent operation
parentSpan := tracer.StartSpan(ctx, "deploy_application")
defer parentSpan.End()

// Create child spans
setupSpan := parentSpan.StartChild("setup_environment")
setupSpan.SetField("step", "1/3")
// ... do setup work
setupSpan.End()

deploySpan := parentSpan.StartChild("deploy_code")
deploySpan.SetField("step", "2/3")
// ... deploy code
deploySpan.End()

verifySpan := parentSpan.StartChild("verify_deployment")
verifySpan.SetField("step", "3/3")
// ... verify deployment
verifySpan.End()
```

### Custom Configuration

```go
// Build custom configuration
config := tracer.NewBuilder().
    WithLevel(tracer.LevelDebug).
    WithSampler(tracer.NewProbabilitySampler(0.1)).
    WithColors(true).
    WithStackTrace(true).
    WithServiceInfo("tunnel", "1.0.0", "production").
    Build()

// Create factory with custom config
factory := tracer.NewTunnelTracerFactory(config)
```

### Formatters

```go
// Console formatter (colored, human-readable)
consoleFormatter := tracer.NewConsoleFormatter()
consoleFormatter.EnableColors = true
consoleFormatter.ShowTimestamp = true

// JSON formatter (structured, machine-readable)
jsonFormatter := tracer.NewJSONFormatter(true) // pretty print

// Compact formatter (single-line, space-efficient)
compactFormatter := tracer.NewCompactFormatter()

// Custom exporter with formatter
exporter := tracer.NewWriterExporter(os.Stdout, consoleFormatter)
```

### Sampling Strategies

```go
// Always sample (development)
sampler := tracer.NewAlwaysSampler()

// Never sample (disabled)
sampler := tracer.NewNeverSampler()

// Probability sampling (production)
sampler := tracer.NewProbabilitySampler(0.1) // 10% of traces

// Custom sampler
type CustomSampler struct{}
func (s *CustomSampler) ShouldSample(ctx context.Context, operation string, spanID tracer.SpanID, parentSpanID tracer.SpanID) bool {
    // Custom logic, e.g., sample all errors
    return strings.Contains(operation, "error")
}
```

### Testing

```go
// Setup test tracing
func TestSSHOperation(t *testing.T) {
    factory := tracer.SetupTestTracing(t)
    defer factory.Shutdown(context.Background())
    
    sshTracer := factory.CreateSSHTracer()
    
    // Your test code with tracing
    span := sshTracer.TraceConnection(context.Background(), "localhost", 22, "test")
    // ... test operations
    span.End()
}

// Mock tracer for unit tests
type mockTracer struct{}
func (m *mockTracer) StartSpan(ctx context.Context, operation string) tracer.Span {
    return tracer.NewNoOpSpan()
}
```

### Error Handling

```go
// Trace with error handling
span := tracer.StartSpan(ctx, "risky_operation")
defer func() {
    if r := recover(); r != nil {
        span.SetStatus(tracer.StatusError)
        span.SetField("panic", fmt.Sprintf("%v", r))
        span.End()
        panic(r) // re-panic
    }
}()

result, err := doRiskyOperation()
if err != nil {
    tracer.RecordError(span, err, "operation failed")
    span.EndWithError(err)
    return err
}

span.SetField("result", result)
span.End()
```

### Metrics Recording

```go
// Record connection stats
tracer.RecordConnectionStats(span, stats)

// Record SSH metrics
tracer.RecordSSHMetrics(span, map[string]interface{}{
    "connections_total": 100,
    "connections_active": 45,
    "commands_executed": 1523,
    "avg_latency_ms": 23.5,
})

// Record retry attempts
tracer.RecordRetry(span, attempt, maxAttempts, backoffDelay)
```

## Integration with Tunnel Package

The tracer is designed to integrate seamlessly with the tunnel architecture:

```go
// In tunnel's client.go
type sshClient struct {
    config  ConnectionConfig
    conn    *ssh.Client
    tracer  tracer.SSHTracer  // Injected tracer
}

func (c *sshClient) Connect(ctx context.Context) error {
    span := c.tracer.TraceConnection(ctx, c.config.Host, c.config.Port, c.config.Username)
    defer span.End()
    
    // Connection logic...
    
    if err != nil {
        span.EndWithError(err)
        return err
    }
    
    span.Event("connection_established")
    return nil
}

// In tunnel's pool.go
type connectionPool struct {
    factory ConnectionFactory
    tracer  tracer.PoolTracer  // Injected tracer
}

func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error) {
    span := p.tracer.TraceGet(ctx, key)
    defer span.End()
    
    // Pool logic...
    
    span.SetField("pool.size", len(p.connections))
    span.SetField("pool.key", key)
    
    return client, nil
}
```

## Output Examples

### Console Output
```
10:23:45.123 ✓ ssh.connect (125ms) host=server.example.com user=deploy
  10:23:45.124 → connection_initiated host=server.example.com port=22 user=deploy
  10:23:45.248 → auth_attempt method=publickey user=deploy
  10:23:45.249 → connection_established
10:23:45.250 ✓ ssh.execute.sudo (45ms) command="systemctl restart nginx" sudo=true
  10:23:45.251 → command_started command="systemctl restart nginx" sudo=true
  10:23:45.295 → command_completed exit_code=0 output_lines=3 duration=44ms
```

### JSON Output
```json
{
  "trace_id": "a1b2c3d4e5f6",
  "span_id": "1234567890ab",
  "operation": "ssh.connect",
  "start_time": "2024-01-15T10:23:45.123Z",
  "end_time": "2024-01-15T10:23:45.248Z",
  "duration_ms": 125,
  "status": "OK",
  "fields": {
    "ssh.host": "server.example.com",
    "ssh.port": 22,
    "ssh.user": "deploy"
  },
  "events": [
    {
      "name": "connection_initiated",
      "timestamp": "2024-01-15T10:23:45.124Z",
      "fields": {
        "host": "server.example.com",
        "port": 22,
        "user": "deploy"
      }
    }
  ]
}
```

### Compact Output
```
10:23:45.123 | OK | ssh.connect | 125ms | host=server.example.com | user=deploy
10:23:45.250 | OK | ssh.execute.sudo | 45ms | host=server.example.com | user=deploy
```

## Performance Considerations

- **Sampling**: Use probability sampling in production to reduce overhead
- **Async Export**: Exporters run asynchronously to minimize latency
- **Buffer Pooling**: Reuses buffers to reduce allocations
- **Level Filtering**: Skip trace creation for disabled levels
- **NoOp Mode**: Zero overhead when tracing is disabled

## Best Practices

1. **Always defer span.End()**: Ensures spans are closed even on panic
2. **Use context propagation**: Pass context through your call chain
3. **Add meaningful fields**: Include relevant metadata for debugging
4. **Use appropriate levels**: Debug for development, Info for production
5. **Sample in production**: Reduce overhead with probability sampling
6. **Structure your spans**: Use parent-child relationships for complex operations
7. **Record errors properly**: Use EndWithError for failed operations
8. **Use specialized tracers**: SSH, Pool, Security, Service tracers for domain operations

## File Structure

```
internal/tracer/
├── interfaces.go    # Core interfaces and contracts
├── tracer.go       # Main tracer implementation
├── ssh.go          # SSH-specific tracing
├── formatter.go    # Output formatters
├── factory.go      # Factory and configuration
└── README.md       # This file
```

## Migration from Logger

The tracer package is designed to replace the logger for the tunnel package while maintaining compatibility:

```go
// Old logger approach
logger.SSHConnect(host, port, username, "connected")

// New tracer approach
span := sshTracer.TraceConnection(ctx, host, port, username)
span.Event("connected")
span.End()
```

The tracer provides richer context, better performance, and more flexibility while maintaining clean separation from the legacy logger package.