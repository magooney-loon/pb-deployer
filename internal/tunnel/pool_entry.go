package tunnel

import (
	"fmt"
	"sync"
	"time"
)

// poolEntryManager manages the lifecycle and metadata of pool entries
type poolEntryManager struct {
	entries map[string]*poolEntry
	mu      sync.RWMutex
	tracer  PoolTracer
}

// newPoolEntryManager creates a new pool entry manager
func newPoolEntryManager(tracer PoolTracer) *poolEntryManager {
	return &poolEntryManager{
		entries: make(map[string]*poolEntry),
		tracer:  tracer,
	}
}

// createEntry creates a new pool entry with proper initialization
func (pem *poolEntryManager) createEntry(key string, client SSHClient) *poolEntry {
	entry := &poolEntry{
		client:          client,
		connectionKey:   key,
		state:           EntryStateIdle,
		createdAt:       time.Now(),
		lastUsed:        time.Now(),
		lastHealthCheck: time.Now(),
		useCount:        0,
		healthFailures:  0,
		inUse:           false,
		metadata:        make(map[string]interface{}),
	}

	// Set initial metadata
	entry.metadata["created_at"] = entry.createdAt
	entry.metadata["version"] = "1.0"
	entry.metadata["type"] = "ssh_connection"

	return entry
}

// addEntry adds an entry to the manager
func (pem *poolEntryManager) addEntry(key string, entry *poolEntry) {
	pem.mu.Lock()
	defer pem.mu.Unlock()
	pem.entries[key] = entry
}

// getEntry retrieves an entry by key
func (pem *poolEntryManager) getEntry(key string) (*poolEntry, bool) {
	pem.mu.RLock()
	defer pem.mu.RUnlock()
	entry, exists := pem.entries[key]
	return entry, exists
}

// removeEntry removes an entry from the manager
func (pem *poolEntryManager) removeEntry(key string) (*poolEntry, bool) {
	pem.mu.Lock()
	defer pem.mu.Unlock()
	entry, exists := pem.entries[key]
	if exists {
		delete(pem.entries, key)
	}
	return entry, exists
}

// getAllEntries returns a copy of all entries
func (pem *poolEntryManager) getAllEntries() map[string]*poolEntry {
	pem.mu.RLock()
	defer pem.mu.RUnlock()

	entries := make(map[string]*poolEntry, len(pem.entries))
	for k, v := range pem.entries {
		entries[k] = v
	}
	return entries
}

// getEntriesByState returns entries in a specific state
func (pem *poolEntryManager) getEntriesByState(state EntryState) []*poolEntry {
	pem.mu.RLock()
	defer pem.mu.RUnlock()

	var entries []*poolEntry
	for _, entry := range pem.entries {
		entry.mu.RLock()
		if entry.state == state {
			entries = append(entries, entry)
		}
		entry.mu.RUnlock()
	}
	return entries
}

// getIdleEntries returns all idle entries sorted by last used time
func (pem *poolEntryManager) getIdleEntries() []*poolEntry {
	entries := pem.getEntriesByState(EntryStateIdle)

	// Sort by last used time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			entries[i].mu.RLock()
			entries[j].mu.RLock()
			if entries[i].lastUsed.After(entries[j].lastUsed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
			entries[j].mu.RUnlock()
			entries[i].mu.RUnlock()
		}
	}

	return entries
}

// markAsUsed marks an entry as being actively used
func (pe *poolEntry) markAsUsed() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.inUse = true
	pe.lastUsed = time.Now()
	pe.useCount++
	pe.state = EntryStateActive
	pe.metadata["last_used"] = pe.lastUsed
	pe.metadata["use_count"] = pe.useCount
}

// markAsIdle marks an entry as idle and available for reuse
func (pe *poolEntry) markAsIdle() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.inUse = false
	pe.state = EntryStateIdle
	pe.lastUsed = time.Now()
	pe.metadata["last_used"] = pe.lastUsed
	pe.metadata["state"] = pe.state.String()
}

// markAsUnhealthy marks an entry as unhealthy
func (pe *poolEntry) markAsUnhealthy(reason string) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.state = EntryStateUnhealthy
	pe.healthFailures++
	pe.metadata["unhealthy_reason"] = reason
	pe.metadata["health_failures"] = pe.healthFailures
	pe.metadata["unhealthy_at"] = time.Now()
}

// markAsClosed marks an entry as closed
func (pe *poolEntry) markAsClosed() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.state = EntryStateClosed
	pe.inUse = false
	pe.metadata["closed_at"] = time.Now()
	pe.metadata["state"] = pe.state.String()
}

// isAvailable checks if an entry is available for use
func (pe *poolEntry) isAvailable() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	return !pe.inUse && pe.state == EntryStateIdle && pe.client != nil
}

// isExpired checks if an entry has expired based on idle time
func (pe *poolEntry) isExpired(maxIdleTime time.Duration) bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	if pe.inUse || pe.state != EntryStateIdle {
		return false
	}

	return time.Since(pe.lastUsed) > maxIdleTime
}

// shouldEvict determines if an entry should be evicted
func (pe *poolEntry) shouldEvict(config PoolConfig) bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Evict if closed or unhealthy
	if pe.state == EntryStateClosed || pe.state == EntryStateUnhealthy {
		return true
	}

	// Evict if idle too long
	if pe.state == EntryStateIdle && !pe.inUse {
		return time.Since(pe.lastUsed) > config.MaxIdleTime
	}

	// Evict if too many health failures
	if pe.healthFailures > 3 {
		return true
	}

	return false
}

// validateHealth performs a health check on the entry
func (pe *poolEntry) validateHealth() bool {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	if pe.client == nil {
		pe.state = EntryStateClosed
		return false
	}

	// Check if client is connected
	if !pe.client.IsConnected() {
		pe.state = EntryStateUnhealthy
		pe.healthFailures++
		pe.metadata["last_health_failure"] = time.Now()
		return false
	}

	// Update health check timestamp
	pe.lastHealthCheck = time.Now()
	pe.metadata["last_health_check"] = pe.lastHealthCheck

	// Reset health failures on successful check
	if pe.state == EntryStateUnhealthy {
		pe.state = EntryStateIdle
		pe.healthFailures = 0
		pe.metadata["recovered_at"] = time.Now()
	}

	return true
}

// getMetadata returns a copy of the entry's metadata
func (pe *poolEntry) getMetadata() map[string]interface{} {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	metadata := make(map[string]interface{})
	for k, v := range pe.metadata {
		metadata[k] = v
	}

	// Add current state information
	metadata["state"] = pe.state.String()
	metadata["in_use"] = pe.inUse
	metadata["age"] = time.Since(pe.createdAt)
	metadata["idle_time"] = time.Since(pe.lastUsed)

	return metadata
}

// setMetadata sets metadata for the entry
func (pe *poolEntry) setMetadata(key string, value interface{}) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.metadata[key] = value
}

// updateResponseTime updates the average response time for this entry
func (pe *poolEntry) updateResponseTime(responseTime time.Duration) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	// Simple moving average (could be improved with more sophisticated algorithms)
	if avgTime, exists := pe.metadata["avg_response_time"]; exists {
		if existingTime, ok := avgTime.(time.Duration); ok {
			pe.metadata["avg_response_time"] = (existingTime + responseTime) / 2
		} else {
			pe.metadata["avg_response_time"] = responseTime
		}
	} else {
		pe.metadata["avg_response_time"] = responseTime
	}

	pe.metadata["last_response_time"] = responseTime
}

// getAge returns the age of the entry
func (pe *poolEntry) getAge() time.Duration {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return time.Since(pe.createdAt)
}

// getIdleTime returns how long the entry has been idle
func (pe *poolEntry) getIdleTime() time.Duration {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	if pe.inUse {
		return 0
	}

	return time.Since(pe.lastUsed)
}

// getState returns the current state of the entry
func (pe *poolEntry) getState() EntryState {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.state
}

// getUseCount returns the number of times this entry has been used
func (pe *poolEntry) getUseCount() int64 {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.useCount
}

// isInUse returns whether the entry is currently in use
func (pe *poolEntry) isInUse() bool {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.inUse
}

// close closes the entry and its associated client
func (pe *poolEntry) close() error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	var err error
	if pe.client != nil {
		err = pe.client.Close()
		pe.client = nil
	}

	pe.state = EntryStateClosed
	pe.inUse = false
	pe.metadata["closed_at"] = time.Now()

	return err
}

// String returns a string representation of the entry
func (pe *poolEntry) String() string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	return fmt.Sprintf("PoolEntry{key=%s, state=%s, inUse=%t, useCount=%d, age=%v, idleTime=%v}",
		pe.connectionKey,
		pe.state.String(),
		pe.inUse,
		pe.useCount,
		time.Since(pe.createdAt),
		time.Since(pe.lastUsed))
}

// EntrySnapshot represents a snapshot of an entry's state for debugging/monitoring
type EntrySnapshot struct {
	ConnectionKey   string                 `json:"connection_key"`
	State           string                 `json:"state"`
	InUse           bool                   `json:"in_use"`
	CreatedAt       time.Time              `json:"created_at"`
	LastUsed        time.Time              `json:"last_used"`
	LastHealthCheck time.Time              `json:"last_health_check"`
	UseCount        int64                  `json:"use_count"`
	HealthFailures  int                    `json:"health_failures"`
	Age             time.Duration          `json:"age"`
	IdleTime        time.Duration          `json:"idle_time"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// getSnapshot returns a snapshot of the entry's current state
func (pe *poolEntry) getSnapshot() EntrySnapshot {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	return EntrySnapshot{
		ConnectionKey:   pe.connectionKey,
		State:           pe.state.String(),
		InUse:           pe.inUse,
		CreatedAt:       pe.createdAt,
		LastUsed:        pe.lastUsed,
		LastHealthCheck: pe.lastHealthCheck,
		UseCount:        pe.useCount,
		HealthFailures:  pe.healthFailures,
		Age:             time.Since(pe.createdAt),
		IdleTime:        time.Since(pe.lastUsed),
		Metadata:        pe.getMetadata(),
	}
}

// PoolEntryStats represents statistics for all pool entries
type PoolEntryStats struct {
	TotalEntries     int           `json:"total_entries"`
	IdleEntries      int           `json:"idle_entries"`
	ActiveEntries    int           `json:"active_entries"`
	UnhealthyEntries int           `json:"unhealthy_entries"`
	ClosedEntries    int           `json:"closed_entries"`
	AverageAge       time.Duration `json:"average_age"`
	AverageUseCount  float64       `json:"average_use_count"`
	OldestEntry      time.Duration `json:"oldest_entry"`
	NewestEntry      time.Duration `json:"newest_entry"`
}

// getEntryStats returns statistics for all entries in the manager
func (pem *poolEntryManager) getEntryStats() PoolEntryStats {
	pem.mu.RLock()
	defer pem.mu.RUnlock()

	stats := PoolEntryStats{}

	if len(pem.entries) == 0 {
		return stats
	}

	var totalAge time.Duration
	var totalUseCount int64
	var oldestAge time.Duration
	var newestAge time.Duration = time.Since(time.Now()) // Will be negative, but we'll fix it

	for _, entry := range pem.entries {
		entry.mu.RLock()

		stats.TotalEntries++
		age := time.Since(entry.createdAt)
		totalAge += age
		totalUseCount += entry.useCount

		if age > oldestAge {
			oldestAge = age
		}
		if age < newestAge || newestAge < 0 {
			newestAge = age
		}

		switch entry.state {
		case EntryStateIdle:
			stats.IdleEntries++
		case EntryStateActive:
			stats.ActiveEntries++
		case EntryStateUnhealthy:
			stats.UnhealthyEntries++
		case EntryStateClosed:
			stats.ClosedEntries++
		}

		entry.mu.RUnlock()
	}

	if stats.TotalEntries > 0 {
		stats.AverageAge = totalAge / time.Duration(stats.TotalEntries)
		stats.AverageUseCount = float64(totalUseCount) / float64(stats.TotalEntries)
	}

	stats.OldestEntry = oldestAge
	stats.NewestEntry = newestAge

	return stats
}
