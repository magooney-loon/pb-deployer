package pool

import (
	"fmt"
	"time"

	"pb-deployer/internal/tunnel"
)

// DefaultPoolConfig returns a pool configuration with sensible defaults
func DefaultPoolConfig() tunnel.PoolConfig {
	return tunnel.PoolConfig{
		MaxConnections:  10,
		MaxIdleTime:     15 * time.Minute,
		HealthInterval:  30 * time.Second,
		CleanupInterval: 5 * time.Minute,
		MaxRetries:      3,
	}
}

// DevelopmentPoolConfig returns a pool configuration optimized for development
func DevelopmentPoolConfig() tunnel.PoolConfig {
	return tunnel.PoolConfig{
		MaxConnections:  5,
		MaxIdleTime:     5 * time.Minute,
		HealthInterval:  10 * time.Second,
		CleanupInterval: 2 * time.Minute,
		MaxRetries:      2,
	}
}

// ProductionPoolConfig returns a pool configuration optimized for production
func ProductionPoolConfig() tunnel.PoolConfig {
	return tunnel.PoolConfig{
		MaxConnections:  50,
		MaxIdleTime:     10 * time.Minute,
		HealthInterval:  15 * time.Second,
		CleanupInterval: 3 * time.Minute,
		MaxRetries:      5,
	}
}

// WithMaxConnections returns a copy of the config with the specified max connections
func WithMaxConnections(c tunnel.PoolConfig, max int) tunnel.PoolConfig {
	c.MaxConnections = max
	return c
}

// WithMaxIdleTime returns a copy of the config with the specified max idle time
func WithMaxIdleTime(c tunnel.PoolConfig, duration time.Duration) tunnel.PoolConfig {
	c.MaxIdleTime = duration
	return c
}

// WithHealthInterval returns a copy of the config with the specified health interval
func WithHealthInterval(c tunnel.PoolConfig, duration time.Duration) tunnel.PoolConfig {
	c.HealthInterval = duration
	return c
}

// WithCleanupInterval returns a copy of the config with the specified cleanup interval
func WithCleanupInterval(c tunnel.PoolConfig, duration time.Duration) tunnel.PoolConfig {
	c.CleanupInterval = duration
	return c
}

// WithMaxRetries returns a copy of the config with the specified max retries
func WithMaxRetries(c tunnel.PoolConfig, retries int) tunnel.PoolConfig {
	c.MaxRetries = retries
	return c
}

// validatePoolConfig validates the pool configuration
func validatePoolConfig(c tunnel.PoolConfig) error {
	if c.MaxConnections <= 0 {
		return fmt.Errorf("MaxConnections must be greater than 0, got %d", c.MaxConnections)
	}

	if c.MaxConnections > 1000 {
		return fmt.Errorf("MaxConnections too high (max 1000), got %d", c.MaxConnections)
	}

	if c.MaxIdleTime <= 0 {
		return fmt.Errorf("MaxIdleTime must be greater than 0, got %v", c.MaxIdleTime)
	}

	if c.MaxIdleTime > 24*time.Hour {
		return fmt.Errorf("MaxIdleTime too high (max 24h), got %v", c.MaxIdleTime)
	}

	if c.HealthInterval <= 0 {
		return fmt.Errorf("HealthInterval must be greater than 0, got %v", c.HealthInterval)
	}

	if c.HealthInterval < time.Second {
		return fmt.Errorf("HealthInterval too low (min 1s), got %v", c.HealthInterval)
	}

	if c.CleanupInterval <= 0 {
		return fmt.Errorf("CleanupInterval must be greater than 0, got %v", c.CleanupInterval)
	}

	if c.CleanupInterval < 10*time.Second {
		return fmt.Errorf("CleanupInterval too low (min 10s), got %v", c.CleanupInterval)
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries must be non-negative, got %d", c.MaxRetries)
	}

	if c.MaxRetries > 10 {
		return fmt.Errorf("MaxRetries too high (max 10), got %d", c.MaxRetries)
	}

	// Logical validations
	if c.HealthInterval >= c.MaxIdleTime {
		return fmt.Errorf("HealthInterval (%v) must be less than MaxIdleTime (%v)", c.HealthInterval, c.MaxIdleTime)
	}

	if c.CleanupInterval < c.HealthInterval {
		return fmt.Errorf("CleanupInterval (%v) should be at least as long as HealthInterval (%v)", c.CleanupInterval, c.HealthInterval)
	}

	return nil
}
