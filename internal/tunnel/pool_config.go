package tunnel

import (
	"encoding/json"
	"fmt"
	"time"
)

// AdvancedPoolConfig extends the base PoolConfig with advanced configuration options
type AdvancedPoolConfig struct {
	PoolConfig

	// Strategy configuration
	EvictionStrategy  string            `json:"eviction_strategy"`
	LoadBalancingMode LoadBalancingMode `json:"load_balancing_mode"`
	AffinityStrategy  AffinityStrategy  `json:"affinity_strategy"`

	// Connection management
	ConnectionWarming       bool   `json:"connection_warming"`
	WarmupConnections       int    `json:"warmup_connections"`
	PreferLocalConnections  bool   `json:"prefer_local_connections"`
	ConnectionTimeoutPolicy string `json:"connection_timeout_policy"`

	// Performance tuning
	MetricsEnabled bool `json:"metrics_enabled"`
	TracingEnabled bool `json:"tracing_enabled"`
	DebugMode      bool `json:"debug_mode"`
	VerboseLogging bool `json:"verbose_logging"`

	// Advanced features
	CircuitBreakerEnabled bool `json:"circuit_breaker_enabled"`
	RateLimitingEnabled   bool `json:"rate_limiting_enabled"`
	CompressionEnabled    bool `json:"compression_enabled"`
	KeepAliveEnabled      bool `json:"keep_alive_enabled"`

	// Monitoring and alerting
	HealthMonitoringConfig HealthMonitoringConfig `json:"health_monitoring_config"`
	AlertingConfig         AlertingConfig         `json:"alerting_config"`
	MetricsConfig          MetricsConfig          `json:"metrics_config"`

	// Environment-specific settings
	Environment string            `json:"environment"`
	Profile     string            `json:"profile"`
	Tags        map[string]string `json:"tags"`
}

// HealthMonitoringConfig configures health monitoring behavior
type HealthMonitoringConfig struct {
	Enabled                bool          `json:"enabled"`
	CheckInterval          time.Duration `json:"check_interval"`
	HealthCheckTimeout     time.Duration `json:"health_check_timeout"`
	UnhealthyThreshold     int           `json:"unhealthy_threshold"`
	RecoveryThreshold      int           `json:"recovery_threshold"`
	EnablePreemptiveChecks bool          `json:"enable_preemptive_checks"`
	HealthCheckCommand     string        `json:"health_check_command"`
}

// AlertingConfig configures alerting thresholds and behavior
type AlertingConfig struct {
	Enabled                    bool          `json:"enabled"`
	AlertOnConnectionFailures  bool          `json:"alert_on_connection_failures"`
	AlertOnHighLatency         bool          `json:"alert_on_high_latency"`
	AlertOnLowCacheHitRatio    bool          `json:"alert_on_low_cache_hit_ratio"`
	ConnectionFailureThreshold int           `json:"connection_failure_threshold"`
	HighLatencyThreshold       time.Duration `json:"high_latency_threshold"`
	LowCacheHitRatioThreshold  float64       `json:"low_cache_hit_ratio_threshold"`
	AlertCooldown              time.Duration `json:"alert_cooldown"`
	AlertWebhookURL            string        `json:"alert_webhook_url"`
}

// MetricsConfig configures metrics collection and export
type MetricsConfig struct {
	Enabled            bool          `json:"enabled"`
	CollectionInterval time.Duration `json:"collection_interval"`
	ExportInterval     time.Duration `json:"export_interval"`
	RetentionPeriod    time.Duration `json:"retention_period"`
	MetricsEndpoint    string        `json:"metrics_endpoint"`
	CustomMetrics      []string      `json:"custom_metrics"`
}

// WorkloadType defines different workload characteristics for tuning
type WorkloadType int

const (
	WorkloadTypeDevelopment WorkloadType = iota
	WorkloadTypeProduction
	WorkloadTypeHighThroughput
	WorkloadTypeLowLatency
	WorkloadTypeBatch
	WorkloadTypeInteractive
	WorkloadTypeStandby
)

func (wt WorkloadType) String() string {
	switch wt {
	case WorkloadTypeDevelopment:
		return "development"
	case WorkloadTypeProduction:
		return "production"
	case WorkloadTypeHighThroughput:
		return "high_throughput"
	case WorkloadTypeLowLatency:
		return "low_latency"
	case WorkloadTypeBatch:
		return "batch"
	case WorkloadTypeInteractive:
		return "interactive"
	case WorkloadTypeStandby:
		return "standby"
	default:
		return "unknown"
	}
}

// ConfigurationProfile defines pre-configured settings for different use cases
type ConfigurationProfile struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Config      AdvancedPoolConfig `json:"config"`
	Workload    WorkloadType       `json:"workload"`
}

// ValidatePoolConfig validates an advanced pool configuration
func ValidatePoolConfig(config AdvancedPoolConfig) error {
	// Validate base pool config
	if config.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be greater than 0")
	}

	if config.MaxIdleTime <= 0 {
		return fmt.Errorf("max_idle_time must be greater than 0")
	}

	if config.HealthInterval <= 0 {
		return fmt.Errorf("health_interval must be greater than 0")
	}

	if config.CleanupInterval <= 0 {
		return fmt.Errorf("cleanup_interval must be greater than 0")
	}

	// Validate warmup configuration
	if config.ConnectionWarming && config.WarmupConnections <= 0 {
		return fmt.Errorf("warmup_connections must be greater than 0 when connection_warming is enabled")
	}

	if config.WarmupConnections > config.MaxConnections {
		return fmt.Errorf("warmup_connections cannot exceed max_connections")
	}

	// Validate eviction strategy
	validStrategies := []string{"lru", "fifo", "lfu", "custom"}
	validStrategy := false
	for _, strategy := range validStrategies {
		if config.EvictionStrategy == strategy {
			validStrategy = true
			break
		}
	}
	if !validStrategy {
		return fmt.Errorf("invalid eviction_strategy: %s, must be one of %v", config.EvictionStrategy, validStrategies)
	}

	// Validate health monitoring config
	if config.HealthMonitoringConfig.Enabled {
		if config.HealthMonitoringConfig.CheckInterval <= 0 {
			return fmt.Errorf("health check_interval must be greater than 0")
		}

		if config.HealthMonitoringConfig.HealthCheckTimeout <= 0 {
			return fmt.Errorf("health check_timeout must be greater than 0")
		}

		if config.HealthMonitoringConfig.UnhealthyThreshold <= 0 {
			return fmt.Errorf("unhealthy_threshold must be greater than 0")
		}
	}

	// Validate alerting config
	if config.AlertingConfig.Enabled {
		if config.AlertingConfig.ConnectionFailureThreshold <= 0 {
			return fmt.Errorf("connection_failure_threshold must be greater than 0")
		}

		if config.AlertingConfig.LowCacheHitRatioThreshold < 0 || config.AlertingConfig.LowCacheHitRatioThreshold > 100 {
			return fmt.Errorf("low_cache_hit_ratio_threshold must be between 0 and 100")
		}
	}

	return nil
}

// SetDefaults sets default values for missing configuration options
func (config *AdvancedPoolConfig) SetDefaults() {
	// Set base pool config defaults
	if config.MaxConnections <= 0 {
		config.MaxConnections = DefaultMaxConnections
	}
	if config.MaxIdleTime <= 0 {
		config.MaxIdleTime = DefaultMaxIdleTime
	}
	if config.HealthInterval <= 0 {
		config.HealthInterval = DefaultHealthCheckInterval
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = DefaultCleanupInterval
	}
	if config.MaxRetries <= 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	// Set advanced config defaults
	if config.EvictionStrategy == "" {
		config.EvictionStrategy = "lru"
	}
	if config.LoadBalancingMode == 0 {
		config.LoadBalancingMode = LoadBalanceRoundRobin
	}
	if config.AffinityStrategy == 0 {
		config.AffinityStrategy = AffinityNone
	}

	// Set health monitoring defaults
	if config.HealthMonitoringConfig.CheckInterval == 0 {
		config.HealthMonitoringConfig.CheckInterval = 30 * time.Second
	}
	if config.HealthMonitoringConfig.HealthCheckTimeout == 0 {
		config.HealthMonitoringConfig.HealthCheckTimeout = 10 * time.Second
	}
	if config.HealthMonitoringConfig.UnhealthyThreshold == 0 {
		config.HealthMonitoringConfig.UnhealthyThreshold = 3
	}
	if config.HealthMonitoringConfig.RecoveryThreshold == 0 {
		config.HealthMonitoringConfig.RecoveryThreshold = 2
	}

	// Set alerting defaults
	if config.AlertingConfig.ConnectionFailureThreshold == 0 {
		config.AlertingConfig.ConnectionFailureThreshold = 5
	}
	if config.AlertingConfig.HighLatencyThreshold == 0 {
		config.AlertingConfig.HighLatencyThreshold = 10 * time.Second
	}
	if config.AlertingConfig.LowCacheHitRatioThreshold == 0 {
		config.AlertingConfig.LowCacheHitRatioThreshold = 80.0
	}
	if config.AlertingConfig.AlertCooldown == 0 {
		config.AlertingConfig.AlertCooldown = 5 * time.Minute
	}

	// Set metrics defaults
	if config.MetricsConfig.CollectionInterval == 0 {
		config.MetricsConfig.CollectionInterval = 15 * time.Second
	}
	if config.MetricsConfig.ExportInterval == 0 {
		config.MetricsConfig.ExportInterval = 60 * time.Second
	}
	if config.MetricsConfig.RetentionPeriod == 0 {
		config.MetricsConfig.RetentionPeriod = 24 * time.Hour
	}

	// Set environment defaults
	if config.Environment == "" {
		config.Environment = "development"
	}
	if config.Profile == "" {
		config.Profile = "default"
	}
	if config.Tags == nil {
		config.Tags = make(map[string]string)
	}
}

// ApplyTuning applies workload-specific tuning to the configuration
func (config *AdvancedPoolConfig) ApplyTuning(workload WorkloadType) {
	switch workload {
	case WorkloadTypeDevelopment:
		config.applyDevelopmentTuning()
	case WorkloadTypeProduction:
		config.applyProductionTuning()
	case WorkloadTypeHighThroughput:
		config.applyHighThroughputTuning()
	case WorkloadTypeLowLatency:
		config.applyLowLatencyTuning()
	case WorkloadTypeBatch:
		config.applyBatchTuning()
	case WorkloadTypeInteractive:
		config.applyInteractiveTuning()
	case WorkloadTypeStandby:
		config.applyStandbyTuning()
	}

	config.Tags["workload"] = workload.String()
}

// applyDevelopmentTuning applies development-friendly settings
func (config *AdvancedPoolConfig) applyDevelopmentTuning() {
	config.MaxConnections = 10
	config.MaxIdleTime = 30 * time.Minute
	config.HealthInterval = 60 * time.Second
	config.CleanupInterval = 10 * time.Minute
	config.DebugMode = true
	config.VerboseLogging = true
	config.MetricsEnabled = true
	config.TracingEnabled = true
	config.Environment = "development"
}

// applyProductionTuning applies production-optimized settings
func (config *AdvancedPoolConfig) applyProductionTuning() {
	config.MaxConnections = 50
	config.MaxIdleTime = 15 * time.Minute
	config.HealthInterval = 30 * time.Second
	config.CleanupInterval = 5 * time.Minute
	config.ConnectionWarming = true
	config.WarmupConnections = 5
	config.DebugMode = false
	config.VerboseLogging = false
	config.MetricsEnabled = true
	config.TracingEnabled = true
	config.CircuitBreakerEnabled = true
	config.HealthMonitoringConfig.Enabled = true
	config.AlertingConfig.Enabled = true
	config.Environment = "production"
}

// applyHighThroughputTuning applies high-throughput optimized settings
func (config *AdvancedPoolConfig) applyHighThroughputTuning() {
	config.MaxConnections = 100
	config.MaxIdleTime = 10 * time.Minute
	config.HealthInterval = 15 * time.Second
	config.CleanupInterval = 2 * time.Minute
	config.ConnectionWarming = true
	config.WarmupConnections = 20
	config.LoadBalancingMode = LoadBalanceLeastUsed
	config.EvictionStrategy = "lru"
	config.RateLimitingEnabled = false
	config.CompressionEnabled = false
	config.KeepAliveEnabled = true
	config.Environment = "production"
}

// applyLowLatencyTuning applies low-latency optimized settings
func (config *AdvancedPoolConfig) applyLowLatencyTuning() {
	config.MaxConnections = 30
	config.MaxIdleTime = 5 * time.Minute
	config.HealthInterval = 10 * time.Second
	config.CleanupInterval = 1 * time.Minute
	config.ConnectionWarming = true
	config.WarmupConnections = 10
	config.LoadBalancingMode = LoadBalanceRoundRobin
	config.PreferLocalConnections = true
	config.KeepAliveEnabled = true
	config.AlertingConfig.HighLatencyThreshold = 1 * time.Second
	config.Environment = "production"
}

// applyBatchTuning applies batch processing optimized settings
func (config *AdvancedPoolConfig) applyBatchTuning() {
	config.MaxConnections = 200
	config.MaxIdleTime = 60 * time.Minute
	config.HealthInterval = 120 * time.Second
	config.CleanupInterval = 30 * time.Minute
	config.ConnectionWarming = false
	config.LoadBalancingMode = LoadBalanceWeighted
	config.EvictionStrategy = "fifo"
	config.RateLimitingEnabled = true
	config.Environment = "production"
}

// applyInteractiveTuning applies interactive workload optimized settings
func (config *AdvancedPoolConfig) applyInteractiveTuning() {
	config.MaxConnections = 25
	config.MaxIdleTime = 20 * time.Minute
	config.HealthInterval = 45 * time.Second
	config.CleanupInterval = 8 * time.Minute
	config.ConnectionWarming = true
	config.WarmupConnections = 3
	config.LoadBalancingMode = LoadBalanceRoundRobin
	config.KeepAliveEnabled = true
	config.Environment = "production"
}

// applyStandbyTuning applies standby/backup optimized settings
func (config *AdvancedPoolConfig) applyStandbyTuning() {
	config.MaxConnections = 5
	config.MaxIdleTime = 120 * time.Minute
	config.HealthInterval = 300 * time.Second
	config.CleanupInterval = 60 * time.Minute
	config.ConnectionWarming = false
	config.MetricsEnabled = false
	config.TracingEnabled = false
	config.DebugMode = false
	config.VerboseLogging = false
	config.Environment = "standby"
}

// GetConfigurationProfiles returns predefined configuration profiles
func GetConfigurationProfiles() []ConfigurationProfile {
	return []ConfigurationProfile{
		{
			Name:        "development",
			Description: "Development-friendly configuration with debug features",
			Config:      GetDevelopmentConfig(),
			Workload:    WorkloadTypeDevelopment,
		},
		{
			Name:        "production",
			Description: "Production-ready configuration with monitoring and alerting",
			Config:      GetProductionConfig(),
			Workload:    WorkloadTypeProduction,
		},
		{
			Name:        "high-throughput",
			Description: "High-throughput configuration for heavy workloads",
			Config:      GetHighThroughputConfig(),
			Workload:    WorkloadTypeHighThroughput,
		},
		{
			Name:        "low-latency",
			Description: "Low-latency configuration for real-time applications",
			Config:      GetLowLatencyConfig(),
			Workload:    WorkloadTypeLowLatency,
		},
		{
			Name:        "batch",
			Description: "Batch processing configuration for large-scale operations",
			Config:      GetBatchConfig(),
			Workload:    WorkloadTypeBatch,
		},
		{
			Name:        "interactive",
			Description: "Interactive workload configuration for user-facing applications",
			Config:      GetInteractiveConfig(),
			Workload:    WorkloadTypeInteractive,
		},
		{
			Name:        "standby",
			Description: "Standby configuration for backup and disaster recovery",
			Config:      GetStandbyConfig(),
			Workload:    WorkloadTypeStandby,
		},
	}
}

// GetDevelopmentConfig returns a development configuration
func GetDevelopmentConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeDevelopment)
	return config
}

// GetProductionConfig returns a production configuration
func GetProductionConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeProduction)
	return config
}

// GetHighThroughputConfig returns a high-throughput configuration
func GetHighThroughputConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeHighThroughput)
	return config
}

// GetLowLatencyConfig returns a low-latency configuration
func GetLowLatencyConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeLowLatency)
	return config
}

// GetBatchConfig returns a batch processing configuration
func GetBatchConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeBatch)
	return config
}

// GetInteractiveConfig returns an interactive workload configuration
func GetInteractiveConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeInteractive)
	return config
}

// GetStandbyConfig returns a standby configuration
func GetStandbyConfig() AdvancedPoolConfig {
	config := AdvancedPoolConfig{}
	config.SetDefaults()
	config.ApplyTuning(WorkloadTypeStandby)
	return config
}

// ToJSON converts the configuration to JSON
func (config *AdvancedPoolConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(config, "", "  ")
}

// FromJSON loads configuration from JSON
func (config *AdvancedPoolConfig) FromJSON(data []byte) error {
	if err := json.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for any missing values
	config.SetDefaults()

	// Validate the loaded configuration
	if err := ValidatePoolConfig(*config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	return nil
}

// Clone creates a deep copy of the configuration
func (config *AdvancedPoolConfig) Clone() AdvancedPoolConfig {
	// Create a copy by marshaling and unmarshaling
	data, _ := config.ToJSON()

	var cloned AdvancedPoolConfig
	cloned.FromJSON(data)

	return cloned
}

// Merge merges another configuration into this one, with the other taking precedence
func (config *AdvancedPoolConfig) Merge(other AdvancedPoolConfig) {
	// This would implement field-by-field merging logic
	// For now, we'll do a simple override of non-zero values

	if other.MaxConnections > 0 {
		config.MaxConnections = other.MaxConnections
	}
	if other.MaxIdleTime > 0 {
		config.MaxIdleTime = other.MaxIdleTime
	}
	if other.HealthInterval > 0 {
		config.HealthInterval = other.HealthInterval
	}
	if other.CleanupInterval > 0 {
		config.CleanupInterval = other.CleanupInterval
	}
	if other.MaxRetries > 0 {
		config.MaxRetries = other.MaxRetries
	}

	if other.EvictionStrategy != "" {
		config.EvictionStrategy = other.EvictionStrategy
	}
	if other.LoadBalancingMode != 0 {
		config.LoadBalancingMode = other.LoadBalancingMode
	}
	if other.AffinityStrategy != 0 {
		config.AffinityStrategy = other.AffinityStrategy
	}

	// Merge boolean flags
	config.ConnectionWarming = config.ConnectionWarming || other.ConnectionWarming
	config.MetricsEnabled = config.MetricsEnabled || other.MetricsEnabled
	config.TracingEnabled = config.TracingEnabled || other.TracingEnabled
	config.DebugMode = config.DebugMode || other.DebugMode

	// Merge tags
	if other.Tags != nil {
		if config.Tags == nil {
			config.Tags = make(map[string]string)
		}
		for k, v := range other.Tags {
			config.Tags[k] = v
		}
	}
}

// GetConfigSummary returns a human-readable summary of the configuration
func (config *AdvancedPoolConfig) GetConfigSummary() string {
	return fmt.Sprintf(
		"Pool Config: %d max connections, %v idle timeout, %s eviction, %s load balancing, %s environment",
		config.MaxConnections,
		config.MaxIdleTime,
		config.EvictionStrategy,
		config.LoadBalancingMode.String(),
		config.Environment,
	)
}

// CompareConfigs compares two configurations and returns differences
func CompareConfigs(config1, config2 AdvancedPoolConfig) map[string]interface{} {
	differences := make(map[string]interface{})

	if config1.MaxConnections != config2.MaxConnections {
		differences["max_connections"] = map[string]interface{}{
			"current": config1.MaxConnections,
			"new":     config2.MaxConnections,
		}
	}

	if config1.MaxIdleTime != config2.MaxIdleTime {
		differences["max_idle_time"] = map[string]interface{}{
			"current": config1.MaxIdleTime,
			"new":     config2.MaxIdleTime,
		}
	}

	if config1.EvictionStrategy != config2.EvictionStrategy {
		differences["eviction_strategy"] = map[string]interface{}{
			"current": config1.EvictionStrategy,
			"new":     config2.EvictionStrategy,
		}
	}

	// Add more field comparisons as needed

	return differences
}
