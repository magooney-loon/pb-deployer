# Phase 3: Implement Connection Pool with Dependency Injection

## Overview
Phase 3 focuses on implementing a robust, thread-safe connection pool that manages multiple SSH connections efficiently. This phase builds upon the SSHClient and ConnectionFactory from Phase 2 to provide connection reuse, lifecycle management, and automatic health monitoring without any singleton patterns.

## Goals
- Implement the `Pool` interface with full dependency injection
- Create efficient connection pooling with configurable strategies
- Integrate health monitoring for automatic connection recovery
- Provide thread-safe operations for concurrent access
- Implement connection lifecycle management (creation, reuse, cleanup)
- Support multiple pooling strategies (LRU, FIFO, custom)
- Establish comprehensive observability with tracing integration

## Prerequisites
- Phase 2 must be completed (SSHClient, ConnectionFactory, AuthHandler)
- Health monitoring components are functional
- Tracer package integration is working
- All Phase 2 tests are passing

## Phase 3 Steps

### Step 3.1: Core Pool Implementation
**File**: `internal/tunnel/pool.go`
**Description**: Implement the main connection pool with dependency injection and thread-safe operations.

**Tasks**:
- [ ] Implement `connectionPool` struct with proper dependency injection
- [ ] Create pool entry management with metadata tracking
- [ ] Implement `Get()` method for connection retrieval/creation
- [ ] Implement `Release()` method for connection return
- [ ] Implement `Close()` method for cleanup
- [ ] Add thread-safe operations with proper locking
- [ ] Integrate comprehensive tracing throughout pool operations
- [ ] Implement connection key generation and management

**Key Components**:
```go
type connectionPool struct {
    factory         ConnectionFactory
    healthMonitor   *PoolHealthMonitor
    tracer          PoolTracer
    config          PoolConfig
    entries         map[string]*poolEntry
    mu              sync.RWMutex
    closed          bool
    stats           *PoolStats
    cleanupTicker   *time.Ticker
    cleanupStop     chan struct{}
}

func NewPool(factory ConnectionFactory, config PoolConfig, tracer PoolTracer) Pool
func (p *connectionPool) Get(ctx context.Context, key string) (SSHClient, error)
func (p *connectionPool) Release(key string, client SSHClient)
func (p *connectionPool) Close() error
func (p *connectionPool) HealthCheck(ctx context.Context) HealthReport
```

### Step 3.2: Pool Entry Management
**File**: `internal/tunnel/pool_entry.go`
**Description**: Implement detailed pool entry management with lifecycle tracking.

**Tasks**:
- [ ] Define `poolEntry` struct with comprehensive metadata
- [ ] Implement entry state management (idle, active, unhealthy, closed)
- [ ] Add usage tracking and statistics collection
- [ ] Implement entry expiration and cleanup logic
- [ ] Create entry validation and health checking
- [ ] Add connection replacement strategies
- [ ] Implement entry serialization for debugging

**Key Components**:
```go
type poolEntry struct {
    client          SSHClient
    connectionKey   string
    state           EntryState
    createdAt       time.Time
    lastUsed        time.Time
    lastHealthCheck time.Time
    useCount        int64
    healthFailures  int
    metadata        map[string]interface{}
    mu              sync.RWMutex
}

type EntryState int
const (
    EntryStateIdle EntryState = iota
    EntryStateActive
    EntryStateUnhealthy
    EntryStateClosed
)
```

### Step 3.3: Connection Strategies
**File**: `internal/tunnel/pool_strategies.go`
**Description**: Implement different connection pooling and eviction strategies.

**Tasks**:
- [ ] Implement LRU (Least Recently Used) eviction strategy
- [ ] Implement FIFO (First In, First Out) eviction strategy
- [ ] Create custom strategy interface for extensibility
- [ ] Implement connection warming strategies
- [ ] Add load balancing strategies for multiple connections
- [ ] Create connection affinity management
- [ ] Implement strategy configuration and switching

**Key Components**:
```go
type EvictionStrategy interface {
    SelectForEviction(entries []*poolEntry) *poolEntry
    ShouldEvict(entry *poolEntry, config PoolConfig) bool
}

type LRUStrategy struct{}
type FIFOStrategy struct{}
type CustomStrategy struct {
    selectFunc func([]*poolEntry) *poolEntry
    shouldEvictFunc func(*poolEntry, PoolConfig) bool
}

func (p *connectionPool) setEvictionStrategy(strategy EvictionStrategy)
```

### Step 3.4: Health Integration
**File**: `internal/tunnel/pool_health.go`
**Description**: Integrate health monitoring with pool operations for automatic recovery.

**Tasks**:
- [ ] Integrate PoolHealthMonitor with connection pool
- [ ] Implement automatic unhealthy connection removal
- [ ] Add health-based connection replacement
- [ ] Create health status reporting for pool entries
- [ ] Implement connection recovery workflows
- [ ] Add health monitoring configuration per connection
- [ ] Create health event publishing for monitoring

**Key Components**:
```go
type PoolHealthIntegration struct {
    pool          Pool
    healthMonitor *PoolHealthMonitor
    tracer        PoolTracer
    config        HealthIntegrationConfig
}

func (phi *PoolHealthIntegration) handleUnhealthyConnection(key string, client SSHClient)
func (phi *PoolHealthIntegration) replaceConnection(ctx context.Context, key string) error
func (phi *PoolHealthIntegration) scheduleHealthChecks(ctx context.Context)
```

### Step 3.5: Pool Statistics and Metrics
**File**: `internal/tunnel/pool_stats.go`
**Description**: Implement comprehensive statistics collection and reporting.

**Tasks**:
- [ ] Create `PoolStats` struct for metrics collection
- [ ] Implement connection usage statistics
- [ ] Add pool performance metrics (hit rate, miss rate, etc.)
- [ ] Create health statistics aggregation
- [ ] Implement metrics export for monitoring systems
- [ ] Add real-time statistics reporting
- [ ] Create historical statistics tracking

**Key Components**:
```go
type PoolStats struct {
    TotalConnections    int64
    ActiveConnections   int64
    IdleConnections     int64
    HealthyConnections  int64
    UnhealthyConnections int64
    ConnectionsCreated  int64
    ConnectionsClosed   int64
    CacheHits          int64
    CacheMisses        int64
    AverageResponseTime time.Duration
    mu                 sync.RWMutex
}

func (ps *PoolStats) RecordConnectionCreated()
func (ps *PoolStats) RecordCacheHit()
func (ps *PoolStats) GetSnapshot() PoolStatsSnapshot
```

### Step 3.6: Cleanup and Lifecycle Management
**File**: `internal/tunnel/pool_cleanup.go`
**Description**: Implement automatic cleanup and connection lifecycle management.

**Tasks**:
- [ ] Implement background cleanup goroutine
- [ ] Add idle connection timeout handling
- [ ] Create connection limit enforcement
- [ ] Implement graceful shutdown procedures
- [ ] Add connection leak detection and prevention
- [ ] Create cleanup configuration and tuning
- [ ] Implement cleanup event logging

**Key Components**:
```go
type CleanupManager struct {
    pool          Pool
    config        CleanupConfig
    tracer        PoolTracer
    ticker        *time.Ticker
    stopCh        chan struct{}
    running       bool
    mu            sync.Mutex
}

func (cm *CleanupManager) Start(ctx context.Context)
func (cm *CleanupManager) Stop()
func (cm *CleanupManager) cleanupIdleConnections()
func (cm *CleanupManager) enforceConnectionLimits()
```

### Step 3.7: Pool Configuration and Tuning
**File**: `internal/tunnel/pool_config.go`
**Description**: Extend pool configuration with advanced tuning options.

**Tasks**:
- [ ] Extend `PoolConfig` with advanced configuration options
- [ ] Implement configuration validation and normalization
- [ ] Add runtime configuration updates
- [ ] Create configuration presets (development, production, etc.)
- [ ] Implement configuration export and import
- [ ] Add configuration change impact analysis
- [ ] Create configuration best practices documentation

**Key Components**:
```go
type AdvancedPoolConfig struct {
    PoolConfig
    EvictionStrategy     string
    ConnectionWarming    bool
    WarmupConnections    int
    LoadBalancingMode    LoadBalancingMode
    AffinityStrategy     AffinityStrategy
    MetricsEnabled       bool
    DebugMode           bool
}

func ValidatePoolConfig(config AdvancedPoolConfig) error
func (config *AdvancedPoolConfig) SetDefaults()
func (config *AdvancedPoolConfig) ApplyTuning(workload WorkloadType)
```

## Success Criteria

### Functional Requirements
- [ ] Pool can successfully manage multiple SSH connections
- [ ] Connection reuse works correctly with proper lifecycle management
- [ ] Health monitoring automatically recovers unhealthy connections
- [ ] Concurrent access is thread-safe without deadlocks
- [ ] Cleanup processes work correctly without resource leaks
- [ ] All pool operations support context cancellation and timeout
- [ ] Statistics and metrics are accurately collected and reported

### Quality Requirements
- [ ] No singleton patterns or global state used anywhere
- [ ] All dependencies are injected through constructors
- [ ] Comprehensive test coverage (>95%)
- [ ] All operations are traced with structured events
- [ ] Memory usage is bounded and predictable
- [ ] Performance scales linearly with connection count
- [ ] Thread safety verified with race condition testing

### Integration Requirements
- [ ] Pool integrates seamlessly with ConnectionFactory from Phase 2
- [ ] Health monitoring works without circular dependencies
- [ ] Tracer integration provides rich observability
- [ ] Configuration is flexible and runtime-tunable
- [ ] Error handling is consistent with Phase 2 patterns

## Dependencies

### Internal Dependencies
- Phase 2 components (SSHClient, ConnectionFactory, AuthHandler)
- Health monitoring from Phase 2
- Tracer package for observability
- Error types and handling from Phase 2

### External Dependencies
- Standard library sync primitives
- Context package for cancellation
- Time package for scheduling and timeouts

## Architecture Considerations

### Connection Management
```go
// Pool manages connections through factory
factory := NewConnectionFactory(tracer)
pool := NewPool(factory, config, poolTracer)

// Connections are created on demand
client, err := pool.Get(ctx, "server-1")
defer pool.Release("server-1", client)
```

### Health Integration
```go
// Health monitoring is integrated but not circular
healthMonitor := NewPoolHealthMonitor(tracer)
pool := NewPoolWithHealth(factory, config, healthMonitor, tracer)

// Pool uses health information for decisions
report := pool.HealthCheck(ctx)
```

### Thread Safety
```go
// All pool operations are thread-safe
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        client, err := pool.Get(ctx, "server-1")
        // ... use client
        pool.Release("server-1", client)
    }()
}
wg.Wait()
```

## Performance Targets

### Connection Management
- Pool Get/Release operations: < 1ms latency
- Connection creation: < 5s for healthy connections
- Health check frequency: configurable, default 30s
- Cleanup cycle: configurable, default 5min

### Scalability
- Support up to 1000 concurrent connections per pool
- Support up to 100 pools per application instance
- Memory usage: < 1MB per 100 idle connections
- CPU usage: < 5% for pool management operations

### Reliability
- 99.9% uptime for pool operations
- Automatic recovery from connection failures
- Zero connection leaks under normal operation
- Graceful degradation under resource pressure

## Risk Mitigation

### Technical Risks
- **Connection Leaks**: Implement comprehensive tracking and automatic cleanup
- **Deadlocks**: Use consistent lock ordering and timeout-based operations
- **Memory Pressure**: Implement connection limits and aggressive cleanup
- **Race Conditions**: Extensive concurrent testing and atomic operations

### Operational Risks
- **Configuration Errors**: Provide validation and safe defaults
- **Monitoring Gaps**: Comprehensive metrics and alerting integration
- **Performance Degradation**: Load testing and performance monitoring
- **Health Check Overhead**: Configurable intervals and lightweight checks

## Migration Strategy

### From Legacy Implementation
```go
// Old singleton approach
pool := ssh.GetConnectionPool()
conn, err := pool.GetOrCreateConnection(server, asRoot)

// New dependency injection approach
factory := NewConnectionFactory(tracer)
pool := NewPool(factory, config, poolTracer)
client, err := pool.Get(ctx, connectionKey)
```

### Compatibility Layer
- Provide adapter for existing pool interfaces
- Gradual migration path with both implementations running
- Feature flag support for gradual rollout
- Rollback capability to legacy implementation

## Estimated Timeline

- **Step 3.1**: 3-4 days (Core pool implementation)
- **Step 3.2**: 2-3 days (Pool entry management)
- **Step 3.3**: 3-4 days (Connection strategies)
- **Step 3.4**: 2-3 days (Health integration)
- **Step 3.5**: 2-3 days (Statistics and metrics)
- **Step 3.6**: 2-3 days (Cleanup and lifecycle)
- **Step 3.7**: 1-2 days (Configuration and tuning)
- **Step 3.8**: 3-4 days (Testing infrastructure)
- **Step 3.9**: 1-2 days (Documentation and examples)

**Total Estimated Time**: 19-28 days

## Next Steps

Upon completion of Phase 3:
- Phase 4: Implement high-level Executor using the Pool
- Integration testing between Pool and SSHClient components
- Performance benchmarking against legacy pool implementation
- Load testing with realistic connection patterns
- Documentation updates for pool usage patterns

## Configuration Examples

### Basic Pool Configuration
```go
config := PoolConfig{
    MaxConnections:  10,
    MaxIdleTime:     15 * time.Minute,
    HealthInterval:  30 * time.Second,
    CleanupInterval: 5 * time.Minute,
    MaxRetries:      3,
}
```

### Advanced Pool Configuration
```go
advancedConfig := AdvancedPoolConfig{
    PoolConfig: config,
    EvictionStrategy:  "lru",
    ConnectionWarming: true,
    WarmupConnections: 3,
    LoadBalancingMode: LoadBalanceRoundRobin,
    MetricsEnabled:    true,
    DebugMode:         false,
}
```

### Production Tuning
```go
productionConfig := AdvancedPoolConfig{
    PoolConfig: PoolConfig{
        MaxConnections:  50,
        MaxIdleTime:     10 * time.Minute,
        HealthInterval:  15 * time.Second,
        CleanupInterval: 2 * time.Minute,
        MaxRetries:      5,
    },
    EvictionStrategy:  "lru",
    ConnectionWarming: true,
    WarmupConnections: 5,
    MetricsEnabled:    true,
}
```
