# Phase 3: Simple Connection Pool Implementation

## Overview
Phase 3 implements a straightforward, thread-safe connection pool that manages SSH connections efficiently. This phase focuses on essential pooling functionality without over-engineering.

## Goals
- Simple connection pooling with dependency injection
- Thread-safe operations for concurrent access
- Basic health monitoring and cleanup
- Connection reuse and lifecycle management
- Integration with tracer package for observability

## Prerequisites
- Phase 2 must be completed (SSHClient, ConnectionFactory, AuthHandler)
- Tracer package integration is working
- All Phase 2 tests are passing

## Phase 3 Implementation

### Step 3.1: Core Pool Implementation
**File**: `internal/tunnel/pool.go`
**Description**: Main connection pool with all essential functionality in one place.

**Tasks**:
- [ ] Implement `Pool` interface with dependency injection
- [ ] Create thread-safe connection management
- [ ] Implement `Get()`, `Release()`, and `Close()` methods
- [ ] Add basic health checking
- [ ] Integrate tracing throughout operations
- [ ] Implement automatic cleanup of idle connections

**Key Components**:
```go
type Pool interface {
    Get(ctx context.Context, key string) (SSHClient, error)
    Release(key string, client SSHClient)
    Close() error
    HealthCheck(ctx context.Context) HealthReport
}

type connectionPool struct {
    factory       ConnectionFactory
    tracer        PoolTracer
    config        PoolConfig
    connections   map[string]*poolEntry
    mu            sync.RWMutex
    closed        bool
    cleanupTicker *time.Ticker
    stopCleanup   chan struct{}
}

type poolEntry struct {
    client    SSHClient
    createdAt time.Time
    lastUsed  time.Time
    useCount  int64
    healthy   bool
}

type PoolConfig struct {
    MaxConnections  int
    MaxIdleTime     time.Duration
    HealthInterval  time.Duration
    CleanupInterval time.Duration
}

type HealthReport struct {
    TotalConnections   int
    HealthyConnections int
    FailedConnections  int
    IdleConnections    int
}
```

### Step 3.2: Pool Configuration
**File**: `internal/tunnel/pool_config.go`
**Description**: Simple configuration with sensible defaults.

**Tasks**:
- [ ] Define `PoolConfig` struct with essential options
- [ ] Implement configuration validation
- [ ] Provide sensible defaults for all options
- [ ] Add configuration helper functions

**Key Components**:
```go
type PoolConfig struct {
    MaxConnections  int           // Maximum connections per pool
    MaxIdleTime     time.Duration // How long to keep idle connections
    HealthInterval  time.Duration // How often to check connection health
    CleanupInterval time.Duration // How often to clean up stale connections
}

func DefaultPoolConfig() PoolConfig {
    return PoolConfig{
        MaxConnections:  10,
        MaxIdleTime:     15 * time.Minute,
        HealthInterval:  30 * time.Second,
        CleanupInterval: 5 * time.Minute,
    }
}

func (c PoolConfig) Validate() error
```

### Step 3.3: Tests
**File**: `internal/tunnel/pool_test.go`
**Description**: Comprehensive tests for pool functionality.

**Tasks**:
- [ ] Test connection creation and reuse
- [ ] Test concurrent access safety
- [ ] Test cleanup and health monitoring
- [ ] Test error handling and recovery
- [ ] Test pool closure and resource cleanup

## Core Implementation Details

### Connection Management
```go
func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error) {
    span := p.tracer.TracePoolGet(ctx, key)
    defer span.End()

    p.mu.Lock()
    defer p.mu.Unlock()

    if p.closed {
        return nil, ErrPoolClosed
    }

    // Check for existing healthy connection
    if entry, exists := p.connections[key]; exists {
        if entry.healthy && time.Since(entry.lastUsed) < p.config.MaxIdleTime {
            entry.lastUsed = time.Now()
            entry.useCount++
            span.Event("connection_reused")
            return entry.client, nil
        }
        // Remove stale/unhealthy connection
        entry.client.Close()
        delete(p.connections, key)
    }

    // Check connection limit
    if len(p.connections) >= p.config.MaxConnections {
        p.evictOldest()
    }

    // Create new connection
    client, err := p.factory.Create(ctx, parseConnectionKey(key))
    if err != nil {
        span.EndWithError(err)
        return nil, err
    }

    // Store in pool
    p.connections[key] = &poolEntry{
        client:    client,
        createdAt: time.Now(),
        lastUsed:  time.Now(),
        useCount:  1,
        healthy:   true,
    }

    span.Event("connection_created")
    return client, nil
}

func (p *connectionPool) Release(key string, client SSHClient) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if entry, exists := p.connections[key]; exists && entry.client == client {
        entry.lastUsed = time.Now()
    }
}
```

### Health Monitoring
```go
func (p *connectionPool) HealthCheck(ctx context.Context) HealthReport {
    span := p.tracer.TraceHealthCheck(ctx)
    defer span.End()

    p.mu.RLock()
    defer p.mu.RUnlock()

    report := HealthReport{
        TotalConnections: len(p.connections),
    }

    for key, entry := range p.connections {
        // Quick health check
        if entry.client.IsConnected() {
            report.HealthyConnections++
            if time.Since(entry.lastUsed) > p.config.MaxIdleTime {
                report.IdleConnections++
            }
        } else {
            report.FailedConnections++
            entry.healthy = false
        }
    }

    span.SetFields(tracer.Fields{
        "pool.total":   report.TotalConnections,
        "pool.healthy": report.HealthyConnections,
        "pool.failed":  report.FailedConnections,
        "pool.idle":    report.IdleConnections,
    })

    return report
}
```

### Cleanup Process
```go
func (p *connectionPool) startCleanup() {
    p.cleanupTicker = time.NewTicker(p.config.CleanupInterval)
    go func() {
        for {
            select {
            case <-p.cleanupTicker.C:
                p.cleanup()
            case <-p.stopCleanup:
                return
            }
        }
    }()
}

func (p *connectionPool) cleanup() {
    span := p.tracer.TraceCleanup(context.Background())
    defer span.End()

    p.mu.Lock()
    defer p.mu.Unlock()

    var removed int
    now := time.Now()

    for key, entry := range p.connections {
        shouldRemove := false

        // Remove if idle too long
        if now.Sub(entry.lastUsed) > p.config.MaxIdleTime {
            shouldRemove = true
        }

        // Remove if unhealthy
        if !entry.healthy || !entry.client.IsConnected() {
            shouldRemove = true
        }

        if shouldRemove {
            entry.client.Close()
            delete(p.connections, key)
            removed++
        }
    }

    span.Event("cleanup_completed", tracer.Int("removed", removed))
}
```

## Usage Example

```go
// Create pool
tracerFactory := tracer.SetupProductionTracing(os.Stdout)
poolTracer := tracerFactory.CreatePoolTracer()
factory := tunnel.NewConnectionFactory(sshTracer)

config := tunnel.DefaultPoolConfig()
config.MaxConnections = 20

pool := tunnel.NewPool(factory, config, poolTracer)
defer pool.Close()

// Use pool
client, err := pool.Get(ctx, "user@server:22")
if err != nil {
    return err
}
defer pool.Release("user@server:22", client)

// Execute command
output, err := client.Execute(ctx, "uptime")
```

## Success Criteria

### Functional Requirements
- [ ] Pool manages connections efficiently with reuse
- [ ] Thread-safe concurrent access
- [ ] Automatic cleanup of stale connections
- [ ] Health monitoring works correctly
- [ ] Proper resource cleanup on close
- [ ] Context cancellation support

### Quality Requirements
- [ ] No singleton patterns used
- [ ] All dependencies injected
- [ ] Comprehensive test coverage (>90%)
- [ ] Operations traced with structured events
- [ ] Memory usage is bounded
- [ ] Simple, maintainable code

## File Structure

```
internal/tunnel/
├── pool.go           # Main pool implementation (~300 lines)
├── pool_config.go    # Configuration (~50 lines)
└── pool_test.go      # Comprehensive tests (~200 lines)
```

## Estimated Timeline

- **Step 3.1**: 2-3 days (Core pool implementation)
- **Step 3.2**: 1 day (Configuration)
- **Step 3.3**: 2 days (Testing)

**Total Estimated Time**: 5-6 days

## Key Design Decisions

### Simplicity First
- Single file for main implementation
- Basic LRU eviction (oldest unused connection)
- Simple health checking (IsConnected())
- Minimal configuration options

### Thread Safety
- Single RWMutex for all operations
- Lock-free read operations where possible
- Atomic operations for counters

### Resource Management
- Automatic cleanup with ticker
- Bounded connection count
- Proper connection closure

### Observability
- Tracing integration throughout
- Health reporting
- Event-based monitoring

This simplified approach gives us all the essential pooling functionality without the complexity overhead. It's maintainable, testable, and sufficient for our needs.