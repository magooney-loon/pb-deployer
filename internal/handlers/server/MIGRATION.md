# Server Handlers Migration Guide

## Overview

Migrate server handlers from direct SSH operations to the modern tunnel/tracer/models architecture with enhanced security, observability, and reliability.

## Current State Analysis

### Current Issues
- Direct SSH connection creation and management
- Manual connection testing without pooling
- Basic troubleshooting with limited diagnostics
- No structured tracing or observability
- Manual database record operations
- Mixed security/setup concerns in handlers
- Basic error handling without retry logic
- No connection health monitoring

### Files to Migrate
- `handlers.go` - Handler registration (needs dependency injection)
- `connection.go` - Connection testing and health checks
- `security.go` - Security lockdown operations
- `setup.go` - Server setup operations  
- `troubleshoot.go` - SSH troubleshooting and diagnostics
- `notifications.go` - Real-time progress updates

## Migration Strategy

### Phase 1: Dependency Injection Architecture
Replace direct SSH operations with injected tunnel services and managers.

### Phase 2: Observability Integration
Add comprehensive tracing for all server operations.

### Phase 3: Enhanced Operations
Use specialized managers for setup, security, and troubleshooting.

## File-by-File Migration

### `handlers.go` - Handler Registration

**Current:**
```go
func RegisterServerHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
    // Direct handler functions
}
```

**Target:**
```go
type ServerHandlers struct {
    executor       tunnel.Executor
    setupMgr       tunnel.SetupManager
    securityMgr    tunnel.SecurityManager
    serviceMgr     tunnel.ServiceManager
    pool           tunnel.Pool
    sshTracer      tracer.SSHTracer
    poolTracer     tracer.PoolTracer
    securityTracer tracer.SecurityTracer
}

func NewServerHandlers(
    executor tunnel.Executor,
    setupMgr tunnel.SetupManager,
    securityMgr tunnel.SecurityManager,
    serviceMgr tunnel.ServiceManager,
    pool tunnel.Pool,
    tracerFactory tracer.TracerFactory,
) *ServerHandlers {
    return &ServerHandlers{
        executor:       executor,
        setupMgr:       setupMgr,
        securityMgr:    securityMgr,
        serviceMgr:     serviceMgr,
        pool:           pool,
        sshTracer:      tracerFactory.CreateSSHTracer(),
        poolTracer:     tracerFactory.CreatePoolTracer(),
        securityTracer: tracerFactory.CreateSecurityTracer(),
    }
}

func (h *ServerHandlers) RegisterRoutes(group *router.RouterGroup[*core.RequestEvent]) {
    // Handler methods with dependency injection
}
```

### `connection.go` - Connection Testing & Health

#### Current Issues
- Manual SSH connection creation
- Basic TCP testing
- No connection pooling usage
- Limited error categorization
- No health metrics collection

#### Migration Changes

**Connection Testing:**
```go
// BEFORE
func testServerConnection(app core.App, e *core.RequestEvent) error {
    // Manual SSH manager creation
    sshManager, err := ssh.NewSSHManager(server, true)
    defer sshManager.Close()
    
    // Basic connection test
    tcpResult := testTCPConnection(host, port)
    rootSSHResult := testSSHConnection(server, true, app)
    appSSHResult := testSSHConnection(server, false, app)
}

// AFTER
func (h *ServerHandlers) testServerConnection(app core.App, e *core.RequestEvent) error {
    span := h.sshTracer.TraceConnection(e.Request.Context(), server.Host, server.Port, "connection_test")
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Use connection pool for testing
    connectionKey := h.pool.GetConnectionKey(serverModel, false)
    
    // Test through pool with health monitoring
    client, err := h.pool.Get(e.Request.Context(), connectionKey)
    if err != nil {
        span.EndWithError(err)
        return handleConnectionError(e, err, "Failed to get connection from pool")
    }
    defer h.pool.Release(connectionKey, client)
    
    // Enhanced connection test with metrics
    result := h.performEnhancedConnectionTest(e.Request.Context(), serverModel, client)
    
    span.SetFields(tracer.Fields{
        "server.id":            serverModel.ID,
        "server.host":          serverModel.Host,
        "server.security_locked": serverModel.SecurityLocked,
        "test.success":         result.Success,
        "test.duration":        result.Duration,
    })
    
    return e.JSON(http.StatusOK, result)
}
```

**Health Monitoring:**
```go
// BEFORE
func getConnectionHealth(app core.App, e *core.RequestEvent) error {
    // Basic SSH service status
    sshService := ssh.GetSSHService()
    connectionStatus := sshService.GetConnectionStatus()
}

// AFTER
func (h *ServerHandlers) getConnectionHealth(app core.App, e *core.RequestEvent) error {
    span := h.poolTracer.TraceHealthCheck(e.Request.Context())
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Get comprehensive pool health
    healthReport := h.pool.HealthCheck(e.Request.Context())
    
    // Get server-specific metrics
    serverHealth := h.getServerHealthMetrics(serverModel)
    
    response := ConnectionHealthResponse{
        ServerID:      serverModel.ID,
        OverallHealth: healthReport.Overall,
        Connections:   healthReport.Connections,
        ServerMetrics: serverHealth,
        Timestamp:     time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":           serverModel.ID,
        "health.overall":      healthReport.Overall,
        "health.active_conns": len(healthReport.Connections),
    })
    
    return e.JSON(http.StatusOK, response)
}
```

### `setup.go` - Server Setup Operations

#### Current Issues
- Direct SSH service usage
- Basic progress notifications
- No transaction-like setup operations
- Limited error recovery

#### Migration Changes

**Setup Process:**
```go
// BEFORE
func runServerSetup(app core.App, e *core.RequestEvent) error {
    // Get SSH service directly
    sshService := ssh.GetSSHService()
    
    // Background goroutine with manual progress
    go func() {
        progressChan := make(chan ssh.SetupStep, 10)
        err := sshService.RunServerSetup(server, progressChan)
    }()
}

// AFTER
func (h *ServerHandlers) runServerSetup(app core.App, e *core.RequestEvent) error {
    span := h.sshTracer.TraceConnection(e.Request.Context(), server.Host, server.Port, "server_setup")
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    if serverModel.IsSetupComplete() {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server setup is already complete",
        })
    }
    
    // Use setup manager with comprehensive configuration
    setupConfig := tunnel.SetupConfig{
        ServerConfig:      serverModel.ToTunnelConfig(),
        CreateUser:        true,
        Username:          serverModel.AppUsername,
        Groups:            []string{"sudo", "systemd-journal"},
        InstallPackages:   true,
        Packages:          []string{"curl", "wget", "unzip", "git", "systemd"},
        SetupDirectories:  true,
        ConfigureFirewall: false, // Separate security step
    }
    
    // Start setup with progress tracking
    go h.performServerSetup(e.Request.Context(), serverModel, setupConfig, span)
    
    span.SetFields(tracer.Fields{
        "server.id":   serverModel.ID,
        "setup.user":  setupConfig.Username,
        "setup.packages": len(setupConfig.Packages),
    })
    
    return e.JSON(http.StatusAccepted, map[string]any{
        "message":   "Server setup started",
        "server_id": serverID,
        "config":    setupConfig,
    })
}

func (h *ServerHandlers) performServerSetup(ctx context.Context, server *models.Server, config tunnel.SetupConfig, parentSpan tracer.Span) {
    span := parentSpan.StartChild("setup_execution")
    defer span.End()
    
    // Create progress channel for real-time updates
    progressChan := make(chan tunnel.SetupProgress, 20)
    
    // Monitor progress and send notifications
    go h.monitorSetupProgress(ctx, server.ID, progressChan)
    
    // Use setup manager for server setup
    if err := h.setupMgr.CreateUser(ctx, config.UserConfig); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to create user: %w", err)
    }
    
    if err := h.setupMgr.SetupSSHKeys(ctx, config.Username, config.SSHKeys); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to setup SSH keys: %w", err)
    }
    
    if err := h.setupMgr.CreateDirectory(ctx, config.AppDirectory, config.Username); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to create directory: %w", err)
    }
    
    if err != nil {
        server.SetupComplete = false
        span.EndWithError(err)
        h.notifySetupFailure(server.ID, err)
    } else {
        server.SetupComplete = true
        span.Event("setup_completed")
        h.notifySetupSuccess(server.ID, result)
    }
    
    // Update server model
    models.SaveServer(app, server)
    
    tracer.RecordSetupResult(span, result)
}
```

### `security.go` - Security Lockdown

#### Current Issues
- Basic security operations
- No comprehensive security assessment
- Limited lockdown configuration
- Manual SSH hardening

#### Migration Changes

**Security Lockdown:**
```go
// BEFORE
func applySecurityLockdown(app core.App, e *core.RequestEvent) error {
    sshService := ssh.GetSSHService()
    go func() {
        err := sshService.ApplySecurityLockdown(server, progressChan)
    }()
}

// AFTER
func (h *ServerHandlers) applySecurityLockdown(app core.App, e *core.RequestEvent) error {
    span := h.securityTracer.TraceAuditEvent(e.Request.Context(), "security_lockdown", tracer.Fields{
        "server.id": serverID,
    })
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    if !serverModel.IsSetupComplete() {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server setup must be completed before security lockdown",
        })
    }
    
    if serverModel.IsSecurityLocked() {
        return e.JSON(http.StatusBadRequest, map[string]string{
            "error": "Server security is already locked down",
        })
    }
    
    // Comprehensive security configuration
    securityConfig := tunnel.SecurityConfig{
        DisableRootLogin:    true,
        DisablePasswordAuth: true,
        AllowedPorts:        []int{22, 80, 443},
        Fail2banConfig: tunnel.Fail2banConfig{
            Enabled:    true,
            MaxRetries: 5,
            BanTime:    3600,
            Services:   []string{"ssh", "nginx"},
        },
        FirewallRules: []tunnel.FirewallRule{
            {Port: 22, Protocol: "tcp", Action: "allow", Source: "any"},
            {Port: 80, Protocol: "tcp", Action: "allow", Source: "any"},
            {Port: 443, Protocol: "tcp", Action: "allow", Source: "any"},
        },
    }
    
    // Start security lockdown with progress tracking
    go h.performSecurityLockdown(e.Request.Context(), serverModel, securityConfig, span)
    
    span.SetFields(tracer.Fields{
        "server.id":                 serverModel.ID,
        "security.disable_root":     securityConfig.DisableRootLogin,
        "security.disable_password": securityConfig.DisablePasswordAuth,
        "security.firewall_rules":   len(securityConfig.FirewallRules),
    })
    
    return e.JSON(http.StatusAccepted, map[string]any{
        "message":   "Security lockdown started",
        "server_id": serverID,
        "config":    securityConfig,
    })
}

func (h *ServerHandlers) performSecurityLockdown(ctx context.Context, server *models.Server, config tunnel.SecurityConfig, parentSpan tracer.Span) {
    span := parentSpan.StartChild("security_execution")
    defer span.End()
    
    // Create progress channel
    progressChan := make(chan tunnel.SecurityProgress, 15)
    
    // Monitor progress
    go h.monitorSecurityProgress(ctx, server.ID, progressChan)
    
    // Apply security lockdown using security manager
    if err := h.securityMgr.ApplyLockdown(ctx, config.SecurityConfig); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to apply security lockdown: %w", err)
    }
    
    if err := h.securityMgr.ConfigureFirewall(ctx, config.FirewallRules); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to configure firewall: %w", err)
    }
    
    if err := h.securityMgr.SetupFail2ban(ctx, config.Fail2banConfig); err != nil {
        span.EndWithError(err)
        return fmt.Errorf("failed to setup fail2ban: %w", err)
    }
    
    if err != nil {
        server.SecurityLocked = false
        span.EndWithError(err)
        h.notifySecurityFailure(server.ID, err)
    } else {
        server.SecurityLocked = true
        span.Event("security_lockdown_completed")
        h.notifySecuritySuccess(server.ID, result)
    }
    
    // Update server model
    models.SaveServer(app, server)
    
    tracer.RecordSecurityResult(span, result)
}
```

### `security.go` - Security Audit Integration

#### Current Issues
- No automated security auditing
- Limited security compliance checking
- Manual security assessment
- No continuous security monitoring

#### Migration Changes

**Enhanced Security Audit:**
```go
// BEFORE
func auditServerSecurity(app core.App, e *core.RequestEvent) error {
    auditResult, err := security.AuditSecurity(server)
    response := processSecurityAudit(server, auditResult, &auditTime)
}

// AFTER
func (h *ServerHandlers) auditServerSecurity(app core.App, e *core.RequestEvent) error {
    span := h.securityTracer.TraceAuditEvent(e.Request.Context(), "security_audit", tracer.Fields{
        "server.id": serverID,
    })
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive security audit using security manager
    auditConfig := tunnel.SecurityAuditConfig{
        CheckSSHConfig:     true,
        CheckFirewall:      true,
        CheckFail2ban:      true,
        CheckUserPerms:     true,
        CheckSystemPerms:   true,
        CheckNetworkPerms:  true,
    }
    
    auditResult, err := h.securityMgr.AuditSecurity(e.Request.Context())
    if err != nil {
        span.EndWithError(err)
        return handleSecurityError(e, err, "Security audit failed")
    }
    
    // Generate security score and recommendations
    securityScore := h.calculateSecurityScore(auditResult)
    recommendations := h.generateSecurityRecommendations(auditResult, serverModel)
    
    response := SecurityAuditResponse{
        ServerID:        serverModel.ID,
        SecurityScore:   securityScore,
        AuditResult:     auditResult,
        Recommendations: recommendations,
        ComplianceStatus: h.assessCompliance(auditResult),
        RiskLevel:       h.calculateRiskLevel(auditResult),
        NextAuditDue:    time.Now().UTC().Add(24 * time.Hour),
        Timestamp:       time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":           serverModel.ID,
        "security.score":      securityScore.Overall,
        "security.ssh_score":  securityScore.SSH,
        "security.fw_score":   securityScore.Firewall,
        "security.risk_level": response.RiskLevel,
        "audit.checks_run":    len(auditResult.Checks),
    })
    
    return e.JSON(http.StatusOK, response)
}

func (h *ServerHandlers) performSecurityAudit(ctx context.Context, server *models.Server) (*tunnel.SecurityAuditResult, error) {
    span := h.securityTracer.TraceAuditEvent(ctx, "security_audit", tracer.Fields{
        "server.id":   server.ID,
        "server.host": server.Host,
    })
    defer span.End()
    
    // Use security manager for comprehensive audit
    auditConfig := tunnel.SecurityAuditConfig{
        CheckSSHConfig:     true,
        CheckFirewall:      true,
        CheckFail2ban:      true,
        CheckUserPerms:     true,
        CheckSystemPerms:   true,
        CheckNetworkPerms:  true,
        CheckCompliance:    true,
        GenerateReport:     true,
    }
    
    auditResult, err := h.securityMgr.AuditSecurity(ctx)
    if err != nil {
        span.EndWithError(err)
        return nil, fmt.Errorf("failed to perform security audit: %w", err)
    }
    
    span.SetFields(tracer.Fields{
        "audit.checks_run":    len(auditResult.Checks),
        "audit.issues_found":  auditResult.IssuesFound,
        "audit.score":         auditResult.OverallScore,
        "audit.risk_level":    auditResult.RiskLevel,
    })
    
    return auditResult, nil
}
```

**Security Monitoring Implementation:**
```go
// BEFORE
func monitorSecurityStatus(app core.App, e *core.RequestEvent) error {
    auditResult := security.ContinuousAudit(server)
    response := processSecurityStatus(server, auditResult, &monitorTime)
}

// AFTER
func (h *ServerHandlers) monitorSecurityStatus(app core.App, e *core.RequestEvent) error {
    span := h.securityTracer.TraceAuditEvent(e.Request.Context(), "security_monitoring", tracer.Fields{
        "server.id": serverID,
    })
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Perform continuous security monitoring
    auditResult, err := h.performSecurityAudit(e.Request.Context(), serverModel)
    if err != nil {
        span.EndWithError(err)
        return handleSecurityError(e, err, "Security audit failed")
    }
    
    // Check for security issues that need immediate attention
    criticalIssues := h.identifyCriticalSecurityIssues(auditResult)
    
    if len(criticalIssues) == 0 {
        return e.JSON(http.StatusOK, map[string]any{
            "message":    "No critical security issues found",
            "audit":      auditResult,
            "next_audit": time.Now().UTC().Add(1 * time.Hour),
        })
    }
    
    // Generate security alerts for critical issues
    alerts := h.generateSecurityAlerts(serverModel, criticalIssues)
    
    // Schedule follow-up audit
    h.scheduleSecurityFollowUp(e.Request.Context(), serverModel.ID, criticalIssues)
    
    monitoringResult := SecurityMonitoringResult{
        ServerID:        serverModel.ID,
        AuditResult:     auditResult,
        CriticalIssues:  criticalIssues,
        SecurityAlerts:  alerts,
        NextAudit:       time.Now().UTC().Add(30 * time.Minute),
        MonitoringLevel: h.determineMonitoringLevel(auditResult),
        Timestamp:       time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":            serverModel.ID,
        "security.score":       auditResult.OverallScore,
        "security.critical":    len(criticalIssues),
        "security.alerts":      len(alerts),
        "monitoring.level":     monitoringResult.MonitoringLevel,
    })
    
    return e.JSON(http.StatusOK, monitoringResult)
}
```

### `security.go` - Security Operations

#### Migration Changes

**Security Assessment:**
```go
// NEW: Enhanced security assessment
func (h *ServerHandlers) assessServerSecurity(app core.App, e *core.RequestEvent) error {
    span := h.securityTracer.TraceAuditEvent(e.Request.Context(), "security_assessment", tracer.Fields{
        "server.id": serverID,
    })
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive security audit
    auditConfig := tunnel.SecurityAuditConfig{
        CheckSSHConfig:     true,
        CheckFirewall:     true,
        CheckFail2ban:     true,
        CheckUserPerms:    true,
        CheckSystemPerms:  true,
        CheckNetworkPerms: true,
    }
    
    auditResult, err := h.securityMgr.AuditSecurity(e.Request.Context())
    if err != nil {
        span.EndWithError(err)
        return handleSecurityError(e, err, "Security audit failed")
    }
    
    // Generate security score and recommendations
    securityScore := h.calculateSecurityScore(auditResult)
    recommendations := h.generateSecurityRecommendations(auditResult, serverModel)
    
    response := SecurityAssessmentResponse{
        ServerID:        serverModel.ID,
        SecurityScore:   securityScore,
        AuditResult:     auditResult,
        Recommendations: recommendations,
        Timestamp:       time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":           serverModel.ID,
        "security.score":      securityScore.Overall,
        "security.issues":     len(auditResult.Issues),
        "security.warnings":   len(auditResult.Warnings),
    })
    
    return e.JSON(http.StatusOK, response)
}
```

### `connection.go` - Enhanced Connection Management

**Pool-Based Connection Testing:**
```go
// NEW: Pool-aware connection testing
func (h *ServerHandlers) testConnectionWithPool(app core.App, e *core.RequestEvent) error {
    span := h.poolTracer.TraceGet(e.Request.Context(), serverID)
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Test both connection types based on security status
    tests := []ConnectionTest{}
    
    if !serverModel.IsSecurityLocked() {
        // Test root connection for non-secured servers
        tests = append(tests, ConnectionTest{
            Type:     "root",
            Username: serverModel.RootUsername,
            AsRoot:   true,
        })
    }
    
    // Always test app user connection
    tests = append(tests, ConnectionTest{
        Type:     "app",
        Username: serverModel.AppUsername,
        AsRoot:   false,
    })
    
    // Execute tests with pool
    results := make([]ConnectionTestResult, len(tests))
    for i, test := range tests {
        results[i] = h.executeConnectionTest(e.Request.Context(), serverModel, test)
    }
    
    // Aggregate results
    overall := h.aggregateConnectionResults(results, serverModel)
    
    span.SetFields(tracer.Fields{
        "server.id":            serverModel.ID,
        "server.security_locked": serverModel.IsSecurityLocked(),
        "tests.count":          len(tests),
        "tests.success":        overall.Success,
    })
    
    tracer.RecordConnectionStats(span, overall.Stats)
    
    return e.JSON(http.StatusOK, overall)
}
```

## New Dependencies and Injection

### Main Constructor
```go
func NewServerHandlers(app core.App) (*ServerHandlers, error) {
    // Setup tracing
    tracerFactory := tracer.SetupProductionTracing(os.Stdout)
    sshTracer := tracerFactory.CreateSSHTracer()
    poolTracer := tracerFactory.CreatePoolTracer()
    securityTracer := tracerFactory.CreateSecurityTracer()
    
    // Setup tunnel components
    factory := tunnel.NewConnectionFactory(sshTracer)
    poolConfig := tunnel.PoolConfig{
        MaxConnections:     50,
        IdleTimeout:       30 * time.Minute,
        HealthCheckInterval: 5 * time.Minute,
        CleanupInterval:   10 * time.Minute,
    }
    pool := tunnel.NewPool(factory, poolConfig, poolTracer)
    executor := tunnel.NewExecutor(pool, sshTracer)
    
    // Setup specialized managers
    setupMgr := tunnel.NewSetupManager(executor, sshTracer)
    securityMgr := tunnel.NewSecurityManager(executor, securityTracer)
    serviceMgr := tunnel.NewServiceManager(executor, sshTracer)
    
    return &ServerHandlers{
        executor:       executor,
        setupMgr:       setupMgr,
        securityMgr:    securityMgr,
        serviceMgr:     serviceMgr,
        pool:           pool,
        sshTracer:      sshTracer,
        poolTracer:     poolTracer,
        securityTracer: securityTracer,
    }, nil
}
```

### Error Handling Strategy

**Structured Error Responses:**
```go
func handleServerError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsConnectionError(err) {
        return e.JSON(http.StatusBadGateway, ErrorResponse{
            Error:      message,
            Details:    "Server connection failed",
            Suggestion: "Check server status and network connectivity",
            Code:       "CONNECTION_FAILED",
        })
    }
    
    if tunnel.IsAuthError(err) {
        return e.JSON(http.StatusUnauthorized, ErrorResponse{
            Error:      message,
            Details:    "SSH authentication failed",
            Suggestion: "Verify SSH keys and user permissions",
            Code:       "AUTH_FAILED",
        })
    }
    
    if tunnel.IsTimeoutError(err) {
        return e.JSON(http.StatusRequestTimeout, ErrorResponse{
            Error:      message,
            Details:    "Operation timed out",
            Suggestion: "Server may be under high load, try again",
            Code:       "TIMEOUT",
        })
    }
    
    return e.JSON(http.StatusInternalServerError, ErrorResponse{
        Error:   message,
        Details: err.Error(),
        Code:    "INTERNAL_ERROR",
    })
}

func handleSetupError(e *core.RequestEvent, err error, message string) error {
    if tunnel.IsRetryable(err) {
        return e.JSON(http.StatusServiceUnavailable, ErrorResponse{
            Error:      message,
            Details:    "Temporary setup failure",
            Suggestion: "Setup will be retried automatically",
            Code:       "SETUP_RETRYABLE",
            Retryable:  true,
        })
    }
    
    return handleServerError(e, err, message)
}
```

## Enhanced Features to Implement

### Connection Pool Monitoring
```go
func (h *ServerHandlers) getPoolMetrics(app core.App, e *core.RequestEvent) error {
    span := h.poolTracer.TraceHealthCheck(e.Request.Context())
    defer span.End()
    
    // Get comprehensive pool health
    healthReport := h.pool.HealthCheck(e.Request.Context())
    
    // Get detailed metrics
    metrics := PoolMetricsResponse{
        TotalConnections:     healthReport.Total,
        HealthyConnections:   healthReport.Healthy,
        UnhealthyConnections: healthReport.Unhealthy,
        IdleConnections:      healthReport.Idle,
        ActiveConnections:    healthReport.Active,
        AverageResponseTime:  healthReport.AvgResponseTime,
        ErrorRate:           healthReport.ErrorRate,
        LastCleanup:         healthReport.LastCleanup,
        Connections:         []ConnectionInfo{},
    }
    
    // Add per-connection details
    for key, conn := range healthReport.Connections {
        metrics.Connections = append(metrics.Connections, ConnectionInfo{
            Key:          key,
            Healthy:      conn.Healthy,
            LastUsed:     conn.LastUsed,
            UseCount:     conn.UseCount,
            ResponseTime: conn.ResponseTime,
            Age:          conn.Age,
        })
    }
    
    span.SetFields(tracer.Fields{
        "pool.total":      metrics.TotalConnections,
        "pool.healthy":    metrics.HealthyConnections,
        "pool.error_rate": metrics.ErrorRate,
    })
    
    return e.JSON(http.StatusOK, metrics)
}
```

### Security Audit Integration
```go
func (h *ServerHandlers) runSecurityAudit(app core.App, e *core.RequestEvent) error {
    span := h.securityTracer.TraceAuditEvent(e.Request.Context(), "security_audit", tracer.Fields{
        "server.id": serverID,
    })
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive security audit
    auditConfig := tunnel.SecurityAuditConfig{
        CheckSSHHardening:     true,
        CheckFirewallRules:    true,
        CheckFail2banStatus:   true,
        CheckUserPermissions:  true,
        CheckSystemServices:   true,
        CheckLogRotation:      true,
        CheckUpdates:          true,
        GenerateReport:        true,
    }
    
    auditResult, err := h.securityMgr.AuditSecurity(e.Request.Context())
    if err != nil {
        span.EndWithError(err)
        return handleSecurityError(e, err, "Security audit failed")
    }
    
    // Generate security score
    securityScore := h.calculateSecurityScore(auditResult)
    
    // Generate recommendations
    recommendations := h.generateSecurityRecommendations(auditResult, serverModel)
    
    response := SecurityAuditResponse{
        ServerID:        serverModel.ID,
        SecurityScore:   securityScore,
        AuditResult:     auditResult,
        Recommendations: recommendations,
        Timestamp:       time.Now().UTC(),
        NextAuditDue:    time.Now().AddDate(0, 1, 0), // Monthly audits
    }
    
    span.SetFields(tracer.Fields{
        "server.id":           serverModel.ID,
        "security.score":      securityScore.Overall,
        "security.issues":     len(auditResult.Issues),
        "security.warnings":   len(auditResult.Warnings),
        "audit.duration":      auditResult.Duration,
    })
    
    return e.JSON(http.StatusOK, response)
}
```

### `notifications.go` - Real-time Progress Updates

#### Current Issues
- Basic WebSocket notifications
- Manual progress channel management
- Limited error propagation

#### Migration Changes

**Enhanced Progress Monitoring:**
```go
// BEFORE
func notifySetupProgress(app core.App, serverID string, step ssh.SetupStep) error {
    subscription := fmt.Sprintf("server_setup_%s", serverID)
    return notifyClients(app, subscription, step)
}

// AFTER
func (h *ServerHandlers) monitorSetupProgress(ctx context.Context, serverID string, progressChan <-chan tunnel.SetupProgress) {
    span := h.sshTracer.TraceConnection(ctx, "", 0, "setup_progress_monitor")
    defer span.End()
    
    for progress := range progressChan {
        // Enhanced progress notification with metadata
        notification := ProgressNotification{
            Type:        "server_setup",
            ServerID:    serverID,
            Step:        progress.Step,
            Status:      progress.Status,
            Message:     progress.Message,
            Details:     progress.Details,
            Progress:    progress.ProgressPct,
            Timestamp:   time.Now().UTC(),
            Duration:    progress.Duration,
            Metadata:    progress.Metadata,
        }
        
        // Send to multiple channels
        h.notifyProgress(app, fmt.Sprintf("server_setup_%s", serverID), notification)
        h.notifyProgress(app, "global_setup_progress", notification)
        
        // Record in tracing
        span.Event("setup_progress", tracer.Fields{
            "step":        progress.Step,
            "status":      progress.Status,
            "progress":    progress.ProgressPct,
        })
        
        // Log significant events
        if progress.Status == "error" || progress.ProgressPct == 100 {
            app.Logger().Info("Setup progress milestone",
                "server_id", serverID,
                "step", progress.Step,
                "status", progress.Status,
                "progress", progress.ProgressPct)
        }
    }
    
    span.Event("setup_progress_monitoring_complete")
}
```

## Advanced Features to Implement

### Health Monitoring Dashboard
```go
func (h *ServerHandlers) getServerHealthDashboard(app core.App, e *core.RequestEvent) error {
    span := h.poolTracer.TraceHealthCheck(e.Request.Context())
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive health dashboard
    dashboard := ServerHealthDashboard{
        ServerID:      serverModel.ID,
        ServerName:    serverModel.Name,
        LastUpdated:   time.Now().UTC(),
        
        // Connection health
        ConnectionHealth: h.getConnectionHealth(serverModel),
        
        // Pool metrics
        PoolMetrics: h.getPoolMetrics(serverModel),
        
        // System metrics (if available)
        SystemMetrics: h.getSystemMetrics(e.Request.Context(), serverModel),
        
        // Security status
        SecurityStatus: h.getSecurityStatus(serverModel),
        
        // Service status
        ServiceStatus: h.getServiceStatus(e.Request.Context(), serverModel),
        
        // Recent activity
        RecentActivity: h.getRecentActivity(serverModel),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":              serverModel.ID,
        "health.connection":      dashboard.ConnectionHealth.Status,
        "health.pool_active":     dashboard.PoolMetrics.ActiveConnections,
        "health.security_score":  dashboard.SecurityStatus.Score,
    })
    
    return e.JSON(http.StatusOK, dashboard)
}
```

### Automated Recovery System
```go
func (h *ServerHandlers) autoRecoverConnection(app core.App, e *core.RequestEvent) error {
    span := h.poolTracer.TraceCleanup(e.Request.Context())
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive recovery process
    recoveryPlan := RecoveryPlan{
        ServerID:   serverModel.ID,
        StartTime:  time.Now(),
        Steps:      []RecoveryStep{},
    }
    
    // Step 1: Connection pool cleanup
    step1 := h.performPoolCleanup(e.Request.Context(), serverModel)
    recoveryPlan.Steps = append(recoveryPlan.Steps, step1)
    
    // Step 2: Connection re-establishment
    step2 := h.performConnectionReestablish(e.Request.Context(), serverModel)
    recoveryPlan.Steps = append(recoveryPlan.Steps, step2)
    
    // Step 3: Health verification
    step3 := h.performHealthVerification(e.Request.Context(), serverModel)
    recoveryPlan.Steps = append(recoveryPlan.Steps, step3)
    
    // Calculate overall success
    recoveryPlan.Success = h.calculateRecoverySuccess(recoveryPlan.Steps)
    recoveryPlan.EndTime = time.Now()
    recoveryPlan.Duration = recoveryPlan.EndTime.Sub(recoveryPlan.StartTime)
    
    span.SetFields(tracer.Fields{
        "server.id":        serverModel.ID,
        "recovery.success": recoveryPlan.Success,
        "recovery.steps":   len(recoveryPlan.Steps),
        "recovery.duration": recoveryPlan.Duration,
    })
    
    return e.JSON(http.StatusOK, recoveryPlan)
}
```

### Configuration Validation
```go
func (h *ServerHandlers) validateServerConfig(app core.App, e *core.RequestEvent) error {
    span := h.sshTracer.TraceConnection(e.Request.Context(), "", 0, "config_validation")
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return handleServerError(e, err, "Server not found")
    }
    
    // Comprehensive configuration validation
    validator := tunnel.NewConfigurationValidator()
    
    validationResult, err := validator.ValidateServerConfiguration(tunnel.ServerConfig{
        Host:            serverModel.Host,
        Port:            serverModel.Port,
        RootUsername:    serverModel.RootUsername,
        AppUsername:     serverModel.AppUsername,
        AuthMethod:      serverModel.GetAuthMethod(),
        SecurityLocked:  serverModel.SecurityLocked,
        SetupComplete:   serverModel.SetupComplete,
    })
    
    if err != nil {
        span.EndWithError(err)
        return handleValidationError(e, err, "Configuration validation failed")
    }
    
    response := ConfigValidationResponse{
        ServerID:    serverModel.ID,
        Valid:       validationResult.Valid,
        Issues:      validationResult.Issues,
        Warnings:    validationResult.Warnings,
        Suggestions: validationResult.Suggestions,
        Score:       validationResult.Score,
        Timestamp:   time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":          serverModel.ID,
        "validation.valid":   validationResult.Valid,
        "validation.issues":  len(validationResult.Issues),
        "validation.score":   validationResult.Score,
    })
    
    return e.JSON(http.StatusOK, response)
}
```

## Step-by-Step Migration Process

### Step 1: Update Handler Structure
```go
// 1. Create new handler struct with dependencies
type ServerHandlers struct {
    executor       tunnel.Executor
    setupMgr       tunnel.SetupManager
    securityMgr    tunnel.SecurityManager
    serviceMgr     tunnel.ServiceManager
    pool           tunnel.Pool
    sshTracer      tracer.SSHTracer
    poolTracer     tracer.PoolTracer
    securityTracer tracer.SecurityTracer
}

// 2. Convert functions to methods
func (h *ServerHandlers) testServerConnection(app core.App, e *core.RequestEvent) error {
    // Implementation with dependencies
}
```

### Step 2: Replace SSH Operations
```go
// BEFORE: Direct SSH operations
sshManager, err := ssh.NewSSHManager(server, useRoot)
result, err := sshManager.ExecuteCommand(cmd)

// AFTER: Use tunnel executor
cmd := tunnel.Command{
    Cmd:     command,
    Sudo:    needsPrivileges,
    Timeout: 30 * time.Second,
}
result, err := h.executor.RunCommand(e.Request.Context(), cmd)
```

### Step 3: Add Comprehensive Tracing
```go
// Add to every operation
span := h.sshTracer.TraceConnection(ctx, host, port, operation)
defer span.End()

// Record operation metadata
span.SetFields(tracer.Fields{
    "server.id":   serverID,
    "operation":   operationType,
    "user":        username,
})

// Record events
span.Event("operation_started")
span.Event("operation_completed")

// Handle errors
if err != nil {
    tracer.RecordError(span, err, "operation failed")
    span.EndWithError(err)
}
```

### Step 4: Enhance Error Handling
```go
// Structured error handling with retry logic
func (h *ServerHandlers) handleOperationError(e *core.RequestEvent, err error, operation string) error {
    if tunnel.IsRetryable(err) {
        return e.JSON(http.StatusServiceUnavailable, RetryableErrorResponse{
            Error:       fmt.Sprintf("%s failed", operation),
            Details:     err.Error(),
            Retryable:   true,
            RetryAfter:  30, // seconds
            Suggestion:  "Operation will be retried automatically",
        })
    }
    
    return handleServerError(e, err, fmt.Sprintf("%s failed", operation))
}
```

### Step 5: Update Database Operations
```go
// BEFORE: Direct record manipulation
record, err := app.FindRecordById("servers", serverID)
record.Set("setup_complete", true)
app.Save(record)

// AFTER: Use models package
serverModel, err := models.GetServer(app, serverID)
serverModel.SetupComplete = true
models.SaveServer(app, serverModel)
```

## New Response Structures

### Enhanced Connection Test Response
```go
type EnhancedConnectionTestResponse struct {
    ServerID          string                    `json:"server_id"`
    ServerName        string                    `json:"server_name"`
    Host              string                    `json:"host"`
    Port              int                       `json:"port"`
    SecurityLocked    bool                      `json:"security_locked"`
    TestResults       []ConnectionTestResult    `json:"test_results"`
    OverallStatus     string                    `json:"overall_status"`
    OverallSuccess    bool                      `json:"overall_success"`
    ConnectionHealth  ConnectionHealthSummary   `json:"connection_health"`
    PoolMetrics       PoolMetricsSummary        `json:"pool_metrics"`
    Recommendations   []string                  `json:"recommendations"`
    NextSteps         []string                  `json:"next_steps"`
    Timestamp         time.Time                 `json:"timestamp"`
    TestDuration      time.Duration             `json:"test_duration"`
}
```

### Security Assessment Response
```go
type SecurityAssessmentResponse struct {
    ServerID        string                 `json:"server_id"`
    SecurityScore   SecurityScore          `json:"security_score"`
    AuditResult     tunnel.SecurityAudit   `json:"audit_result"`
    Recommendations []SecurityRecommendation `json:"recommendations"`
    ComplianceStatus map[string]bool       `json:"compliance_status"`
    RiskLevel       string                `json:"risk_level"`
    NextAuditDue    time.Time             `json:"next_audit_due"`
    Timestamp       time.Time             `json:"timestamp"`
}
```

## Integration Points

### With App Handlers
```go
// Server handlers provide health status for app operations
func (h *ServerHandlers) getServerReadinessForApp(ctx context.Context, serverID string) (*ServerReadiness, error) {
    span := h.sshTracer.TraceConnection(ctx, "", 0, "readiness_check")
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    readiness := &ServerReadiness{
        ServerID:           serverModel.ID,
        SetupComplete:      serverModel.IsSetupComplete(),
        SecurityLocked:     serverModel.IsSecurityLocked(),
        ReadyForDeployment: serverModel.IsReadyForDeployment(),
        ConnectionHealthy:  h.isConnectionHealthy(ctx, serverModel),
        LastChecked:        time.Now().UTC(),
    }
    
    span.SetFields(tracer.Fields{
        "server.id":        serverModel.ID,
        "server.ready":     readiness.ReadyForDeployment,
        "server.healthy":   readiness.ConnectionHealthy,
    })
    
    return readiness, nil
}
```

### With Deployment Handlers
```go
// Server handlers validate deployment prerequisites
func (h *ServerHandlers) validateDeploymentPrerequisites(ctx context.Context, serverID string) (*DeploymentPrereqs, error) {
    span := h.sshTracer.TraceConnection(ctx, "", 0, "deployment_prereqs")
    defer span.End()
    
    serverModel, err := models.GetServer(app, serverID)
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }
    
    // Check all prerequisites
    prereqs := &DeploymentPrereqs{
        ServerReady:        serverModel.IsReadyForDeployment(),
        ConnectionHealthy:  h.isConnectionHealthy(ctx, serverModel),
        StorageAvailable:   h.checkStorageSpace(ctx, serverModel),
        ServicesRunning:    h.checkRequiredServices(ctx, serverModel),
        PermissionsValid:   h.checkDeploymentPermissions(ctx, serverModel),
        Issues:            []string{},
        Warnings:          []string{},
    }
    
    // Validate each requirement
    if !prereqs.ServerReady {
        prereqs.Issues = append(prereqs.Issues, "Server setup or security lockdown not complete")
    }
    
    if !prereqs.ConnectionHealthy {
        prereqs.Issues = append(prereqs.Issues, "SSH connection not healthy")
    }
    
    if !prereqs.StorageAvailable {
        prereqs.Warnings = append(prereqs.Warnings, "Low disk space detected")
    }
    
    prereqs.Valid = len(prereqs.Issues) == 0
    
    span.SetFields(tracer.Fields{
        "server.id":     serverModel.ID,
        "prereqs.valid": prereqs.Valid,
        "prereqs.issues": len(prereqs.Issues),
    })
    
    return prereqs, nil
}
```

## Performance Improvements

### Connection Pool Optimization
```go
// Connection pool configuration for server operations
func (h *ServerHandlers) optimizePoolForServer(serverModel *models.Server) tunnel.PoolConfig {
    baseConfig := tunnel.PoolConfig{
        MaxConnections:      50,
        IdleTimeout:        30 * time.Minute,
        HealthCheckInterval: 5 * time.Minute,
        CleanupInterval:    10 * time.Minute,
    }
    
    // Adjust based on server characteristics
    if serverModel.IsSecurityLocked() {
        // Security-locked servers may have stricter limits
        baseConfig.MaxConnections = 25
        baseConfig.HealthCheckInterval = 2 * time.Minute
    }
    
    if serverModel.Port != 22 {
        // Non-standard ports may need different timeouts
        baseConfig.IdleTimeout = 15 * time.Minute
    }
    
    return baseConfig
}
```

### Caching Strategy
```go
// Cache frequently accessed server data
type ServerCache struct {
    HealthStatus    map[string]*CachedHealth
    ConfigStatus    map[string]*CachedConfig
    SecurityStatus  map[string]*CachedSecurity
    cacheTTL       time.Duration
    mu             sync.RWMutex
}

func (h *ServerHandlers) getCachedServerHealth(serverID string) (*CachedHealth, bool) {
    h.cache.mu.RLock()
    defer h.cache.mu.RUnlock()
    
    if cached, exists := h.cache.HealthStatus[serverID]; exists {
        if time.Since(cached.Timestamp) < h.cache.cacheTTL {
            return cached, true
        }
    }
    
    return nil, false
}
```



## Breaking Changes Summary

### Function Signatures
- All handlers become methods on `ServerHandlers` struct
- Context propagation through all operations
- Structured error responses with actionable suggestions
- Enhanced response types with metadata

### Dependencies
- Replace direct SSH imports with tunnel/tracer packages
- Add dependency injection constructor
- Remove singleton SSH service usage

### Error Handling
- Structured error types instead of simple strings
- Retry logic for transient failures
- Circuit breaker patterns for unstable connections
- Categorized error responses (connection, auth, timeout, etc.)

### Response Formats
- Enhanced response structures with metadata
- Operation timing and tracing information
- Actionable error messages and suggestions
- Real-time progress tracking data

## Validation Checklist

### ✅ Pre-Migration Validation
- [ ] Map all current SSH operations to tunnel managers
- [ ] Identify tracing integration points
- [ ] Plan error handling strategy
- [ ] Design response structure enhancements
- [ ] Assess connection pooling requirements

### ✅ Migration Execution
- [ ] Replace direct SSH with tunnel.Executor
- [ ] Add comprehensive tracing to all operations
- [ ] Use models package for all database operations
- [ ] Implement structured error handling with retry logic
- [ ] Add connection pool integration
- [ ] Enhance real-time progress notifications
- [ ] Add health monitoring and metrics collection

### ✅ Post-Migration Validation
- [ ] All handlers use dependency injection
- [ ] No direct SSH connections in handlers
- [ ] Connection pooling active and optimized
- [ ] Comprehensive tracing coverage
- [ ] Structured error responses implemented
- [ ] Health monitoring enhanced
- [ ] Progress tracking reliable
- [ ] Auto-recovery mechanisms functional
- [ ] Test coverage maintained or improved
- [ ] Performance improvements measurable

## Security Considerations

### Post-Migration Security Model
```go
// Security-aware connection management
func (h *ServerHandlers) getSecurityAwareConnection(ctx context.Context, server *models.Server, operation string) (tunnel.SSHClient, error) {
    span := h.sshTracer.TraceConnection(ctx, server.Host, server.Port, operation)
    defer span.End()
    
    // Choose connection type based on security status and operation
    var connectionKey string
    if server.IsSecurityLocked() {
        // Always use app user for security-locked servers
        connectionKey = h.pool.GetConnectionKey(server, false)
        span.SetField("connection.type", "app_user")
    } else {
        // Choose based on operation requirements
        needsRoot := h.operationRequiresRoot(operation)
        connectionKey = h.pool.GetConnectionKey(server, needsRoot)
        span.SetField("connection.type", map[bool]string{true: "root", false: "app_user"}[needsRoot])
    }
    
    client, err := h.pool.Get(ctx, connectionKey)
    if err != nil {
        span.EndWithError(err)
        return nil, fmt.Errorf("failed to get secure connection: %w", err)
    }
    
    span.SetField("connection.established", true)
    return client, nil
}
```

### Audit Trail Integration
```go
// Enhanced audit logging
func (h *ServerHandlers) auditServerOperation(ctx context.Context, serverID, operation, user string, success bool, err error) {
    span := h.securityTracer.TraceAuditEvent(ctx, "server_operation", tracer.Fields{
        "server.id":        serverID,
        "operation":        operation,
        "user":            user,
        "success":         success,
        "client.ip":       getClientIP(ctx),
    })
    defer span.End()
    
    auditEvent := tunnel.AuditEvent{
        Timestamp:  time.Now().UTC(),
        ServerID:   serverID,
        Operation:  operation,
        User:       user,
        Success:    success,
        ClientIP:   getClientIP(ctx),
        Error:      "",
    }
    
    if err != nil {
        auditEvent.Error = err.Error()
        span.EndWithError(err)
    }
    
    // Record audit event in tracer
    span.Event("security_audit_performed")
    
    span.Event("audit_logged")
}
```

## Rollback Plan

### Incremental Migration Strategy
1. **Keep Old Handlers**: Maintain existing handlers alongside new ones
2. **Feature Flags**: Use configuration to switch between implementations
3. **Gradual Migration**: Migrate endpoint by endpoint
4. **A/B Testing**: Test new implementation with subset of servers
5. **Monitoring**: Enhanced monitoring during migration period

### Rollback Triggers
- Performance degradation > 20%
- Error rate increase > 5%
- Connection pool exhaustion
- Tracing overhead issues
- Memory leaks in long-running operations

### Emergency Rollback Process
```go
// Emergency fallback to old SSH implementation
if config.UseOldSSHImplementation {
    return h.legacyTestServerConnection(app, e)
}
```

## Timeline and Milestones

### Week 1: Foundation
- **Day 1-2**: Update handler structure and dependency injection
- **Day 3-4**: Migrate connection testing and health checks
- **Day 5**: Add comprehensive tracing integration

### Week 2: Core Operations
- **Day 1-2**: Migrate setup operations to use SetupManager
- **Day 3-4**: Migrate security operations to use SecurityManager
- **Day 5**: Enhanced troubleshooting with tunnel diagnostics

### Week 3: Enhancement and Testing
- **Day 1-2**: Add auto-recovery and health monitoring
- **Day 3-4**: Comprehensive testing and validation
- **Day 5**: Performance optimization and tuning

### Week 4: Finalization
- **Day 1-2**: Documentation and code cleanup
- **Day 3-4**: Load testing and performance validation
- **Day 5**: Production deployment preparation

## Success Metrics

### Performance
- Connection reuse rate > 80%
- Average response time improvement > 30%
- Memory usage stable under load
- Zero connection leaks

### Reliability
- Error rate < 1%
- Automatic recovery success rate > 95%
- Health check accuracy > 99%
- Zero hanging operations

### Observability
- 100% operation tracing coverage
- Comprehensive error categorization
- Real-time progress tracking
- Detailed performance metrics

### Security
- All operations audited
- Security assessment automated
- Compliance monitoring active
- Zero privilege escalation issues