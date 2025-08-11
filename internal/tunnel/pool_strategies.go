package tunnel

import (
	"sort"
	"sync"
	"time"
)

// EvictionStrategy defines the interface for connection eviction strategies
type EvictionStrategy interface {
	// SelectForEviction selects an entry for eviction from the given entries
	SelectForEviction(entries []*poolEntry) *poolEntry

	// ShouldEvict determines if an entry should be evicted based on the strategy
	ShouldEvict(entry *poolEntry, config PoolConfig) bool

	// Name returns the name of the strategy
	Name() string
}

// LoadBalancingMode defines how connections are selected for use
type LoadBalancingMode int

const (
	LoadBalanceRoundRobin LoadBalancingMode = iota
	LoadBalanceLeastUsed
	LoadBalanceRandom
	LoadBalanceWeighted
)

func (lbm LoadBalancingMode) String() string {
	switch lbm {
	case LoadBalanceRoundRobin:
		return "round_robin"
	case LoadBalanceLeastUsed:
		return "least_used"
	case LoadBalanceRandom:
		return "random"
	case LoadBalanceWeighted:
		return "weighted"
	default:
		return "unknown"
	}
}

// AffinityStrategy defines connection affinity behavior
type AffinityStrategy int

const (
	AffinityNone AffinityStrategy = iota
	AffinitySticky
	AffinityConsistent
)

func (as AffinityStrategy) String() string {
	switch as {
	case AffinityNone:
		return "none"
	case AffinitySticky:
		return "sticky"
	case AffinityConsistent:
		return "consistent"
	default:
		return "unknown"
	}
}

// LRUStrategy implements Least Recently Used eviction
type LRUStrategy struct {
	name string
}

// NewLRUStrategy creates a new LRU eviction strategy
func NewLRUStrategy() EvictionStrategy {
	return &LRUStrategy{
		name: "lru",
	}
}

// SelectForEviction selects the least recently used entry for eviction
func (lru *LRUStrategy) SelectForEviction(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	var oldest *poolEntry
	var oldestTime time.Time

	for _, entry := range entries {
		entry.mu.RLock()
		// Only consider idle entries for eviction
		if entry.state == EntryStateIdle && !entry.inUse {
			if oldest == nil || entry.lastUsed.Before(oldestTime) {
				oldest = entry
				oldestTime = entry.lastUsed
			}
		}
		entry.mu.RUnlock()
	}

	return oldest
}

// ShouldEvict determines if an entry should be evicted based on LRU criteria
func (lru *LRUStrategy) ShouldEvict(entry *poolEntry, config PoolConfig) bool {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	// Always evict closed or unhealthy entries
	if entry.state == EntryStateClosed || entry.state == EntryStateUnhealthy {
		return true
	}

	// Don't evict entries that are in use
	if entry.inUse || entry.state == EntryStateActive {
		return false
	}

	// Evict if idle longer than max idle time
	return time.Since(entry.lastUsed) > config.MaxIdleTime
}

// Name returns the strategy name
func (lru *LRUStrategy) Name() string {
	return lru.name
}

// FIFOStrategy implements First In, First Out eviction
type FIFOStrategy struct {
	name string
}

// NewFIFOStrategy creates a new FIFO eviction strategy
func NewFIFOStrategy() EvictionStrategy {
	return &FIFOStrategy{
		name: "fifo",
	}
}

// SelectForEviction selects the oldest entry (by creation time) for eviction
func (fifo *FIFOStrategy) SelectForEviction(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	var oldest *poolEntry
	var oldestTime time.Time

	for _, entry := range entries {
		entry.mu.RLock()
		// Only consider idle entries for eviction
		if entry.state == EntryStateIdle && !entry.inUse {
			if oldest == nil || entry.createdAt.Before(oldestTime) {
				oldest = entry
				oldestTime = entry.createdAt
			}
		}
		entry.mu.RUnlock()
	}

	return oldest
}

// ShouldEvict determines if an entry should be evicted based on FIFO criteria
func (fifo *FIFOStrategy) ShouldEvict(entry *poolEntry, config PoolConfig) bool {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	// Always evict closed or unhealthy entries
	if entry.state == EntryStateClosed || entry.state == EntryStateUnhealthy {
		return true
	}

	// Don't evict entries that are in use
	if entry.inUse || entry.state == EntryStateActive {
		return false
	}

	// Evict if created longer ago than max idle time (simplified FIFO)
	return time.Since(entry.createdAt) > config.MaxIdleTime*2
}

// Name returns the strategy name
func (fifo *FIFOStrategy) Name() string {
	return fifo.name
}

// LFUStrategy implements Least Frequently Used eviction
type LFUStrategy struct {
	name string
}

// NewLFUStrategy creates a new LFU eviction strategy
func NewLFUStrategy() EvictionStrategy {
	return &LFUStrategy{
		name: "lfu",
	}
}

// SelectForEviction selects the least frequently used entry for eviction
func (lfu *LFUStrategy) SelectForEviction(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	var leastUsed *poolEntry
	var minUseCount int64 = -1

	for _, entry := range entries {
		entry.mu.RLock()
		// Only consider idle entries for eviction
		if entry.state == EntryStateIdle && !entry.inUse {
			if leastUsed == nil || (minUseCount == -1 || entry.useCount < minUseCount) {
				leastUsed = entry
				minUseCount = entry.useCount
			}
		}
		entry.mu.RUnlock()
	}

	return leastUsed
}

// ShouldEvict determines if an entry should be evicted based on LFU criteria
func (lfu *LFUStrategy) ShouldEvict(entry *poolEntry, config PoolConfig) bool {
	entry.mu.RLock()
	defer entry.mu.RUnlock()

	// Always evict closed or unhealthy entries
	if entry.state == EntryStateClosed || entry.state == EntryStateUnhealthy {
		return true
	}

	// Don't evict entries that are in use
	if entry.inUse || entry.state == EntryStateActive {
		return false
	}

	// Evict if idle too long or has very low usage
	idleTooLong := time.Since(entry.lastUsed) > config.MaxIdleTime
	lowUsage := entry.useCount < 5 && time.Since(entry.createdAt) > time.Hour

	return idleTooLong || lowUsage
}

// Name returns the strategy name
func (lfu *LFUStrategy) Name() string {
	return lfu.name
}

// CustomStrategy allows for custom eviction logic
type CustomStrategy struct {
	name            string
	selectFunc      func([]*poolEntry) *poolEntry
	shouldEvictFunc func(*poolEntry, PoolConfig) bool
}

// NewCustomStrategy creates a new custom eviction strategy
func NewCustomStrategy(name string,
	selectFunc func([]*poolEntry) *poolEntry,
	shouldEvictFunc func(*poolEntry, PoolConfig) bool) EvictionStrategy {

	return &CustomStrategy{
		name:            name,
		selectFunc:      selectFunc,
		shouldEvictFunc: shouldEvictFunc,
	}
}

// SelectForEviction uses the custom selection function
func (cs *CustomStrategy) SelectForEviction(entries []*poolEntry) *poolEntry {
	if cs.selectFunc == nil {
		return nil
	}
	return cs.selectFunc(entries)
}

// ShouldEvict uses the custom eviction function
func (cs *CustomStrategy) ShouldEvict(entry *poolEntry, config PoolConfig) bool {
	if cs.shouldEvictFunc == nil {
		return false
	}
	return cs.shouldEvictFunc(entry, config)
}

// Name returns the strategy name
func (cs *CustomStrategy) Name() string {
	return cs.name
}

// StrategyManager manages eviction strategies and connection selection
type StrategyManager struct {
	evictionStrategy  EvictionStrategy
	loadBalancingMode LoadBalancingMode
	affinityStrategy  AffinityStrategy
	roundRobinCounter int64
	mu                sync.RWMutex
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager() *StrategyManager {
	return &StrategyManager{
		evictionStrategy:  NewLRUStrategy(),
		loadBalancingMode: LoadBalanceRoundRobin,
		affinityStrategy:  AffinityNone,
	}
}

// SetEvictionStrategy sets the eviction strategy
func (sm *StrategyManager) SetEvictionStrategy(strategy EvictionStrategy) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.evictionStrategy = strategy
}

// SetLoadBalancingMode sets the load balancing mode
func (sm *StrategyManager) SetLoadBalancingMode(mode LoadBalancingMode) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.loadBalancingMode = mode
}

// SetAffinityStrategy sets the affinity strategy
func (sm *StrategyManager) SetAffinityStrategy(strategy AffinityStrategy) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.affinityStrategy = strategy
}

// SelectForEviction selects an entry for eviction using the current strategy
func (sm *StrategyManager) SelectForEviction(entries []*poolEntry) *poolEntry {
	sm.mu.RLock()
	strategy := sm.evictionStrategy
	sm.mu.RUnlock()

	if strategy == nil {
		return nil
	}

	return strategy.SelectForEviction(entries)
}

// ShouldEvict determines if an entry should be evicted
func (sm *StrategyManager) ShouldEvict(entry *poolEntry, config PoolConfig) bool {
	sm.mu.RLock()
	strategy := sm.evictionStrategy
	sm.mu.RUnlock()

	if strategy == nil {
		return false
	}

	return strategy.ShouldEvict(entry, config)
}

// SelectConnection selects a connection based on the load balancing strategy
func (sm *StrategyManager) SelectConnection(availableEntries []*poolEntry) *poolEntry {
	if len(availableEntries) == 0 {
		return nil
	}

	sm.mu.Lock()
	mode := sm.loadBalancingMode
	sm.mu.Unlock()

	switch mode {
	case LoadBalanceRoundRobin:
		return sm.selectRoundRobin(availableEntries)
	case LoadBalanceLeastUsed:
		return sm.selectLeastUsed(availableEntries)
	case LoadBalanceRandom:
		return sm.selectRandom(availableEntries)
	case LoadBalanceWeighted:
		return sm.selectWeighted(availableEntries)
	default:
		return availableEntries[0]
	}
}

// selectRoundRobin selects connections in round-robin fashion
func (sm *StrategyManager) selectRoundRobin(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	sm.mu.Lock()
	index := int(sm.roundRobinCounter % int64(len(entries)))
	sm.roundRobinCounter++
	sm.mu.Unlock()

	return entries[index]
}

// selectLeastUsed selects the connection with the lowest use count
func (sm *StrategyManager) selectLeastUsed(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	var selected *poolEntry
	var minUseCount int64 = -1

	for _, entry := range entries {
		entry.mu.RLock()
		if selected == nil || (minUseCount == -1 || entry.useCount < minUseCount) {
			selected = entry
			minUseCount = entry.useCount
		}
		entry.mu.RUnlock()
	}

	return selected
}

// selectRandom selects a random available connection
func (sm *StrategyManager) selectRandom(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	// Simple pseudo-random selection based on current time
	index := int(time.Now().UnixNano()) % len(entries)
	return entries[index]
}

// selectWeighted selects connections based on their performance/health weights
func (sm *StrategyManager) selectWeighted(entries []*poolEntry) *poolEntry {
	if len(entries) == 0 {
		return nil
	}

	// Sort entries by a weighted score (lower health failures = higher priority)
	weightedEntries := make([]*poolEntry, len(entries))
	copy(weightedEntries, entries)

	sort.Slice(weightedEntries, func(i, j int) bool {
		weightedEntries[i].mu.RLock()
		weightedEntries[j].mu.RLock()
		defer weightedEntries[i].mu.RUnlock()
		defer weightedEntries[j].mu.RUnlock()

		// Lower health failures and higher use count = better weight
		scoreI := weightedEntries[i].useCount - int64(weightedEntries[i].healthFailures*10)
		scoreJ := weightedEntries[j].useCount - int64(weightedEntries[j].healthFailures*10)

		return scoreI > scoreJ
	})

	return weightedEntries[0]
}

// GetStrategyInfo returns information about current strategies
func (sm *StrategyManager) GetStrategyInfo() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := map[string]interface{}{
		"load_balancing_mode": sm.loadBalancingMode.String(),
		"affinity_strategy":   sm.affinityStrategy.String(),
		"round_robin_counter": sm.roundRobinCounter,
	}

	if sm.evictionStrategy != nil {
		info["eviction_strategy"] = sm.evictionStrategy.Name()
	}

	return info
}

// CreateStrategyFromName creates an eviction strategy from a string name
func CreateStrategyFromName(name string) EvictionStrategy {
	switch name {
	case "lru":
		return NewLRUStrategy()
	case "fifo":
		return NewFIFOStrategy()
	case "lfu":
		return NewLFUStrategy()
	default:
		return NewLRUStrategy() // Default to LRU
	}
}

// WarmupStrategy defines how connections should be warmed up
type WarmupStrategy struct {
	Enabled           bool
	MinConnections    int
	MaxConnections    int
	WarmupTimeout     time.Duration
	HealthCheckAfter  time.Duration
	ConnectionConfigs []ConnectionConfig
}

// DefaultWarmupStrategy returns a default warmup strategy
func DefaultWarmupStrategy() WarmupStrategy {
	return WarmupStrategy{
		Enabled:          false,
		MinConnections:   2,
		MaxConnections:   5,
		WarmupTimeout:    30 * time.Second,
		HealthCheckAfter: 5 * time.Second,
	}
}

// ConnectionWarmer handles connection pre-warming
type ConnectionWarmer struct {
	strategy WarmupStrategy
	factory  ConnectionFactory
	tracer   PoolTracer
}

// NewConnectionWarmer creates a new connection warmer
func NewConnectionWarmer(strategy WarmupStrategy, factory ConnectionFactory, tracer PoolTracer) *ConnectionWarmer {
	return &ConnectionWarmer{
		strategy: strategy,
		factory:  factory,
		tracer:   tracer,
	}
}

// WarmupConnections pre-creates connections for faster access
func (cw *ConnectionWarmer) WarmupConnections(pool Pool) error {
	if !cw.strategy.Enabled {
		return nil
	}

	// This would implement connection pre-warming logic
	// For now, it's a placeholder for the actual implementation
	return nil
}

// StrategyConfig holds all strategy configurations
type StrategyConfig struct {
	EvictionStrategy  string
	LoadBalancingMode LoadBalancingMode
	AffinityStrategy  AffinityStrategy
	WarmupStrategy    WarmupStrategy
}

// DefaultStrategyConfig returns default strategy configuration
func DefaultStrategyConfig() StrategyConfig {
	return StrategyConfig{
		EvictionStrategy:  "lru",
		LoadBalancingMode: LoadBalanceRoundRobin,
		AffinityStrategy:  AffinityNone,
		WarmupStrategy:    DefaultWarmupStrategy(),
	}
}

// ApplyStrategyConfig applies strategy configuration to a strategy manager
func ApplyStrategyConfig(sm *StrategyManager, config StrategyConfig) {
	sm.SetEvictionStrategy(CreateStrategyFromName(config.EvictionStrategy))
	sm.SetLoadBalancingMode(config.LoadBalancingMode)
	sm.SetAffinityStrategy(config.AffinityStrategy)
}
