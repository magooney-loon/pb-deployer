package tracer

import (
	"context"
	"fmt"
	"time"
)

type SSHTracerImpl struct {
	Tracer
	component string
}

func NewSSHTracer(base Tracer) SSHTracer {
	return &SSHTracerImpl{
		Tracer:    base.WithField("component", "ssh"),
		component: "ssh",
	}
}

func (t *SSHTracerImpl) TraceConnection(ctx context.Context, host string, port int, user string) Span {
	span := t.StartSpan(ctx, "ssh.connect")
	span.SetFields(Fields{
		"ssh.host":    host,
		"ssh.port":    port,
		"ssh.user":    user,
		"ssh.address": fmt.Sprintf("%s:%d", host, port),
		"operation":   "connection",
	})

	span.Event("connection_initiated",
		String("host", host),
		Int("port", port),
		String("user", user),
	)

	return span
}

func (t *SSHTracerImpl) TraceCommand(ctx context.Context, command string, sudo bool) Span {
	operation := "ssh.execute"
	if sudo {
		operation = "ssh.execute.sudo"
	}

	span := t.StartSpan(ctx, operation)

	// Truncate long commands for display
	displayCmd := command
	if len(displayCmd) > 100 {
		displayCmd = displayCmd[:97] + "..."
	}

	span.SetFields(Fields{
		"ssh.command":      displayCmd,
		"ssh.command_full": command,
		"ssh.sudo":         sudo,
		"operation":        "command",
	})

	span.Event("command_started",
		String("command", displayCmd),
		Bool("sudo", sudo),
	)

	return span
}

func (t *SSHTracerImpl) TraceFileTransfer(ctx context.Context, source, dest string, size int64) Span {
	span := t.StartSpan(ctx, "ssh.transfer")
	span.SetFields(Fields{
		"ssh.source":      source,
		"ssh.destination": dest,
		"ssh.size_bytes":  size,
		"operation":       "file_transfer",
	})

	span.Event("transfer_initiated",
		String("source", source),
		String("destination", dest),
		Int64("size", size),
	)

	return span
}

func (t *SSHTracerImpl) TraceHealthCheck(ctx context.Context, target string) Span {
	span := t.StartSpan(ctx, "ssh.health_check")
	span.SetFields(Fields{
		"ssh.target": target,
		"operation":  "health_check",
	})

	span.Event("health_check_started",
		String("target", target),
	)

	return span
}

type PoolTracerImpl struct {
	Tracer
}

func NewPoolTracer(base Tracer) PoolTracer {
	return &PoolTracerImpl{
		Tracer: base.WithField("component", "pool"),
	}
}

func (t *PoolTracerImpl) TraceGet(ctx context.Context, key string) Span {
	span := t.StartSpan(ctx, "pool.get")
	span.SetFields(Fields{
		"pool.key":       key,
		"pool.operation": "get",
	})

	span.Event("connection_requested",
		String("key", key),
	)

	return span
}

func (t *PoolTracerImpl) TraceRelease(ctx context.Context, key string) Span {
	span := t.StartSpan(ctx, "pool.release")
	span.SetFields(Fields{
		"pool.key":       key,
		"pool.operation": "release",
	})

	span.Event("connection_released",
		String("key", key),
	)

	return span
}

func (t *PoolTracerImpl) TraceHealthCheck(ctx context.Context) Span {
	span := t.StartSpan(ctx, "pool.health_check")
	span.SetField("pool.operation", "health_check")

	span.Event("health_check_initiated")

	return span
}

func (t *PoolTracerImpl) TraceCleanup(ctx context.Context) Span {
	span := t.StartSpan(ctx, "pool.cleanup")
	span.SetField("pool.operation", "cleanup")

	span.Event("cleanup_started")

	return span
}

type SecurityTracerImpl struct {
	Tracer
}

func NewSecurityTracer(base Tracer) SecurityTracer {
	return &SecurityTracerImpl{
		Tracer: base.WithField("component", "security"),
	}
}

func (t *SecurityTracerImpl) TraceSecurityCheck(ctx context.Context, checkType string) Span {
	span := t.StartSpan(ctx, "security.check")
	span.SetFields(Fields{
		"security.check_type": checkType,
		"security.operation":  "check",
	})

	span.Event("security_check_started",
		String("type", checkType),
	)

	return span
}

func (t *SecurityTracerImpl) TraceFirewallRule(ctx context.Context, rule string, action string) Span {
	span := t.StartSpan(ctx, "security.firewall")
	span.SetFields(Fields{
		"security.rule":      rule,
		"security.action":    action,
		"security.operation": "firewall",
	})

	span.Event("firewall_rule_applied",
		String("rule", rule),
		String("action", action),
	)

	return span
}

func (t *SecurityTracerImpl) TraceAuthAttempt(ctx context.Context, method string, user string) Span {
	span := t.StartSpan(ctx, "security.auth")
	span.SetFields(Fields{
		"security.auth_method": method,
		"security.user":        user,
		"security.operation":   "authentication",
	})

	span.Event("auth_attempt",
		String("method", method),
		String("user", user),
	)

	return span
}

func (t *SecurityTracerImpl) TraceAuditEvent(ctx context.Context, event string, details Fields) Span {
	span := t.StartSpan(ctx, "security.audit")
	span.SetFields(Fields{
		"security.event":     event,
		"security.operation": "audit",
	})

	// Add all details to the span
	for k, v := range details {
		span.SetField(fmt.Sprintf("audit.%s", k), v)
	}

	fields := make([]Field, 0, len(details)+1)
	fields = append(fields, String("event", event))
	for k, v := range details {
		fields = append(fields, Any(k, v))
	}

	span.Event("audit_event", fields...)

	return span
}

type ServiceTracerImpl struct {
	Tracer
}

func NewServiceTracer(base Tracer) ServiceTracer {
	return &ServiceTracerImpl{
		Tracer: base.WithField("component", "service"),
	}
}

func (t *ServiceTracerImpl) TraceServiceAction(ctx context.Context, service string, action string) Span {
	span := t.StartSpan(ctx, fmt.Sprintf("service.%s", action))
	span.SetFields(Fields{
		"service.name":      service,
		"service.action":    action,
		"service.operation": action,
	})

	span.Event("service_action_started",
		String("service", service),
		String("action", action),
	)

	return span
}

func (t *ServiceTracerImpl) TraceDeployment(ctx context.Context, app string, version string) Span {
	span := t.StartSpan(ctx, "service.deploy")
	span.SetFields(Fields{
		"service.app":       app,
		"service.version":   version,
		"service.operation": "deployment",
	})

	span.Event("deployment_started",
		String("app", app),
		String("version", version),
	)

	return span
}

func (t *ServiceTracerImpl) TraceRollback(ctx context.Context, app string, fromVersion, toVersion string) Span {
	span := t.StartSpan(ctx, "service.rollback")
	span.SetFields(Fields{
		"service.app":          app,
		"service.from_version": fromVersion,
		"service.to_version":   toVersion,
		"service.operation":    "rollback",
	})

	span.Event("rollback_initiated",
		String("app", app),
		String("from_version", fromVersion),
		String("to_version", toVersion),
	)

	return span
}

func (t *ServiceTracerImpl) TraceHealthEndpoint(ctx context.Context, endpoint string) Span {
	span := t.StartSpan(ctx, "service.health")
	span.SetFields(Fields{
		"service.endpoint":  endpoint,
		"service.operation": "health_check",
	})

	span.Event("health_check_started",
		String("endpoint", endpoint),
	)

	return span
}

func RecordSSHMetrics(span Span, metrics map[string]any) {
	for key, value := range metrics {
		span.SetField(fmt.Sprintf("metrics.%s", key), value)
	}

	fields := make([]Field, 0, len(metrics))
	for k, v := range metrics {
		fields = append(fields, Any(k, v))
	}

	span.Event("metrics_recorded", fields...)
}

type ConnectionStats struct {
	TotalConnections    int
	ActiveConnections   int
	FailedConnections   int
	TotalCommands       int
	FailedCommands      int
	AverageResponseTime time.Duration
}

func RecordConnectionStats(span Span, stats ConnectionStats) {
	span.SetFields(Fields{
		"stats.total_connections":  stats.TotalConnections,
		"stats.active_connections": stats.ActiveConnections,
		"stats.failed_connections": stats.FailedConnections,
		"stats.total_commands":     stats.TotalCommands,
		"stats.failed_commands":    stats.FailedCommands,
		"stats.avg_response_time":  stats.AverageResponseTime.String(),
	})
}

func RecordPoolHealth(span Span, totalConns, healthyConns, unhealthyConns int) {
	span.SetFields(Fields{
		"pool.total_connections":     totalConns,
		"pool.healthy_connections":   healthyConns,
		"pool.unhealthy_connections": unhealthyConns,
		"pool.health_percentage":     float64(healthyConns) / float64(totalConns) * 100,
	})

	span.Event("pool_health_recorded",
		Int("total", totalConns),
		Int("healthy", healthyConns),
		Int("unhealthy", unhealthyConns),
	)
}

func RecordCommandResult(span Span, exitCode int, outputLines int, duration time.Duration) {
	span.SetFields(Fields{
		"command.exit_code":    exitCode,
		"command.output_lines": outputLines,
		"command.duration":     duration.String(),
		"command.success":      exitCode == 0,
	})

	if exitCode == 0 {
		span.SetStatus(StatusOK)
	} else {
		span.SetStatus(StatusError)
	}

	span.Event("command_completed",
		Int("exit_code", exitCode),
		Int("output_lines", outputLines),
		Duration("duration", duration),
	)
}

func RecordError(span Span, err error, context string) {
	span.SetFields(Fields{
		"error.message": err.Error(),
		"error.context": context,
		"error.type":    fmt.Sprintf("%T", err),
	})

	span.SetStatus(StatusError)
	span.Event("error_occurred",
		Error(err),
		String("context", context),
	)
}

func RecordRetry(span Span, attempt int, maxAttempts int, delay time.Duration) {
	span.SetFields(Fields{
		"retry.attempt":      attempt,
		"retry.max_attempts": maxAttempts,
		"retry.delay":        delay.String(),
	})

	span.Event("retry_attempt",
		Int("attempt", attempt),
		Int("max_attempts", maxAttempts),
		Duration("delay", delay),
	)
}

func TraceWithTimeout(ctx context.Context, tracer Tracer, operation string, timeout time.Duration) (Span, context.Context, context.CancelFunc) {
	span := tracer.StartSpan(ctx, operation)
	span.SetField("timeout", timeout.String())

	ctx, cancel := context.WithTimeout(span.Context(), timeout)

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			span.SetStatus(StatusTimeout)
			span.Event("operation_timeout",
				Duration("timeout", timeout),
			)
		}
	}()

	return span, ctx, cancel
}

func StartSSHOperation(ctx context.Context, tracer SSHTracer, operation string, server, user string) Span {
	span := tracer.StartSpan(ctx, fmt.Sprintf("ssh.%s", operation))
	span.SetFields(Fields{
		"server.host":   server,
		"server.user":   user,
		"ssh.operation": operation,
	})

	return span
}
