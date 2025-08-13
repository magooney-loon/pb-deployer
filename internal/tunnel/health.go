package tunnel

import (
	"context"
	"fmt"
	"maps"
	"math"
	"strings"
	"sync"
	"time"
)

// HealthChecker manages health monitoring for SSH connections
type HealthChecker struct {
	client           SSHClient
	tracer           SSHTracer
	config           HealthCheckConfig
	mu               sync.RWMutex
	lastCheck        time.Time
	lastResult       HealthResult
	isRunning        bool
	stopCh           chan struct{}
	consecutiveFails int
}

// HealthResult represents the result of a health check
type HealthResult struct {
	Healthy      bool
	ResponseTime time.Duration
	Error        error
	Timestamp    time.Time
	CheckType    string
	Details      map[string]any
}

// NewHealthChecker creates a new health checker for an SSH client
func NewHealthChecker(client SSHClient, tracer SSHTracer) *HealthChecker {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &HealthChecker{
		client:    client,
		tracer:    tracer,
		config:    *DefaultHealthCheckConfig(),
		stopCh:    make(chan struct{}),
		lastCheck: time.Now(),
	}
}

// SetConfig updates the health check configuration
func (hc *HealthChecker) SetConfig(config HealthCheckConfig) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.config = config
}

// CheckHealth performs a single health check
func (hc *HealthChecker) CheckHealth(ctx context.Context) HealthResult {
	span := hc.tracer.TraceConnection(ctx, "health_check", 0, "")
	defer span.End()

	start := time.Now()
	result := HealthResult{
		Timestamp: start,
		CheckType: "basic",
		Details:   make(map[string]any),
	}

	// Check if client is connected
	if !hc.client.IsConnected() {
		result.Healthy = false
		result.Error = ErrClientNotConnected
		result.ResponseTime = time.Since(start)

		span.Event("health_check_failed", map[string]any{
			"reason":        "not_connected",
			"response_time": result.ResponseTime,
		})

		hc.updateResult(result)
		return result
	}

	// Perform a simple command to test responsiveness
	output, err := hc.client.Execute(ctx, "echo 'health_check'")
	result.ResponseTime = time.Since(start)

	if err != nil {
		result.Healthy = false
		result.Error = err
		result.Details["error"] = err.Error()

		span.Event("health_check_failed", map[string]any{
			"reason":        "command_failed",
			"error":         err.Error(),
			"response_time": result.ResponseTime,
		})
	} else if output != "health_check\n" && output != "health_check" {
		result.Healthy = false
		result.Error = fmt.Errorf("unexpected health check response: %s", output)
		result.Details["unexpected_output"] = output

		span.Event("health_check_failed", map[string]any{
			"reason":        "unexpected_output",
			"output":        output,
			"response_time": result.ResponseTime,
		})
	} else {
		result.Healthy = true
		result.Details["output"] = output

		span.Event("health_check_passed", map[string]any{
			"response_time": result.ResponseTime,
		})
	}

	// Check response time threshold
	if result.ResponseTime > hc.config.Timeout {
		result.Healthy = false
		if result.Error == nil {
			result.Error = fmt.Errorf("health check timeout: %v > %v", result.ResponseTime, hc.config.Timeout)
		}
		result.Details["timeout_exceeded"] = true
	}

	hc.updateResult(result)
	return result
}

// IsHealthy returns the current health status
func (hc *HealthChecker) IsHealthy(ctx context.Context) bool {
	hc.mu.RLock()

	// If we haven't checked recently, perform a new check
	if time.Since(hc.lastCheck) > hc.config.Interval {
		hc.mu.RUnlock()
		result := hc.CheckHealth(ctx)
		return result.Healthy
	}

	healthy := hc.lastResult.Healthy
	hc.mu.RUnlock()
	return healthy
}

// GetLastResult returns the last health check result
func (hc *HealthChecker) GetLastResult() HealthResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastResult
}

// GetHealthReport returns a comprehensive health report
func (hc *HealthChecker) GetHealthReport(ctx context.Context) HealthReport {
	span := hc.tracer.TraceConnection(ctx, "health_report", 0, "")
	defer span.End()

	hc.mu.RLock()
	lastResult := hc.lastResult
	consecutiveFails := hc.consecutiveFails
	isRunning := hc.isRunning
	hc.mu.RUnlock()

	// Perform fresh health check
	currentResult := hc.CheckHealth(ctx)

	// Get connection info if available
	var connectionInfo map[string]any
	if clientImpl, ok := hc.client.(*sshClient); ok {
		connectionInfo = clientImpl.GetConnectionInfo()
	}

	report := HealthReport{
		TotalConnections:   1,
		HealthyConnections: 0,
		FailedConnections:  1,
		CheckedAt:          time.Now(),
		Connections: []ConnectionHealth{
			{
				Key:          hc.getConnectionKey(),
				Healthy:      currentResult.Healthy,
				LastUsed:     lastResult.Timestamp,
				UseCount:     int64(consecutiveFails), // Repurpose for fail count
				ResponseTime: currentResult.ResponseTime,
				Error:        hc.formatError(currentResult.Error),
			},
		},
	}

	if currentResult.Healthy {
		report.HealthyConnections = 1
		report.FailedConnections = 0
	}

	// Add connection details if available
	if connectionInfo != nil {
		report.Connections[0].LastUsed = connectionInfo["last_used"].(time.Time)
		if host, ok := connectionInfo["host"]; ok {
			report.Connections[0].Key = fmt.Sprintf("%v", host)
		}
	}

	span.Event("health_report_generated", map[string]any{
		"healthy":           currentResult.Healthy,
		"response_time":     currentResult.ResponseTime,
		"consecutive_fails": consecutiveFails,
		"monitoring_active": isRunning,
	})

	return report
}

// StartMonitoring starts continuous health monitoring
func (hc *HealthChecker) StartMonitoring(ctx context.Context) {
	hc.mu.Lock()
	if hc.isRunning {
		hc.mu.Unlock()
		return
	}
	hc.isRunning = true
	hc.mu.Unlock()

	span := hc.tracer.TraceConnection(ctx, "health_monitoring_start", 0, "")
	defer span.End()

	span.Event("monitoring_started", map[string]any{
		"interval": hc.config.Interval,
		"timeout":  hc.config.Timeout,
	})

	go hc.monitoringLoop(ctx)
}

// StopMonitoring stops continuous health monitoring
func (hc *HealthChecker) StopMonitoring() {
	hc.mu.Lock()
	if !hc.isRunning {
		hc.mu.Unlock()
		return
	}
	hc.isRunning = false
	hc.mu.Unlock()

	close(hc.stopCh)
	hc.stopCh = make(chan struct{}) // Reset for potential restart
}

// RecoverConnection attempts to recover an unhealthy connection
func (hc *HealthChecker) RecoverConnection(ctx context.Context) error {
	span := hc.tracer.TraceConnection(ctx, "connection_recovery", 0, "")
	defer span.End()

	span.Event("recovery_started")

	// Close existing connection
	if err := hc.client.Close(); err != nil {
		span.Event("close_failed", map[string]any{
			"error": err.Error(),
		})
	}

	// Attempt to reconnect
	if err := hc.client.Connect(ctx); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("recovery failed: %w", err)
	}

	// Verify recovery with health check
	result := hc.CheckHealth(ctx)
	if !result.Healthy {
		err := fmt.Errorf("recovery verification failed: %v", result.Error)
		span.EndWithError(err)
		return err
	}

	span.Event("recovery_completed", map[string]any{
		"response_time": result.ResponseTime,
	})

	return nil
}

// monitoringLoop runs the continuous health monitoring
func (hc *HealthChecker) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			hc.mu.Lock()
			hc.isRunning = false
			hc.mu.Unlock()
			return

		case <-hc.stopCh:
			hc.mu.Lock()
			hc.isRunning = false
			hc.mu.Unlock()
			return

		case <-ticker.C:
			result := hc.CheckHealth(ctx)

			if !result.Healthy {
				hc.handleUnhealthyConnection(ctx, result)
			} else {
				hc.mu.Lock()
				hc.consecutiveFails = 0
				hc.mu.Unlock()
			}
		}
	}
}

// handleUnhealthyConnection handles an unhealthy connection
func (hc *HealthChecker) handleUnhealthyConnection(ctx context.Context, result HealthResult) {
	hc.mu.Lock()
	hc.consecutiveFails++
	consecutiveFails := hc.consecutiveFails
	enableAutoRecovery := hc.config.EnableAutoRecovery
	maxFails := hc.config.MaxConsecutiveFails
	recoveryRetries := hc.config.RecoveryRetries
	hc.mu.Unlock()

	span := hc.tracer.TraceConnection(ctx, "unhealthy_connection", 0, "")
	defer span.End()

	span.Event("unhealthy_detected", map[string]any{
		"consecutive_fails": consecutiveFails,
		"error":             hc.formatError(result.Error),
		"response_time":     result.ResponseTime,
	})

	// Attempt recovery if enabled and threshold reached
	if enableAutoRecovery && consecutiveFails >= maxFails {
		span.Event("recovery_triggered", map[string]any{
			"max_retries": recoveryRetries,
		})

		for attempt := 1; attempt <= recoveryRetries; attempt++ {
			if err := hc.RecoverConnection(ctx); err == nil {
				span.Event("recovery_successful", map[string]any{
					"attempt": attempt,
				})
				return
			} else {
				span.Event("recovery_failed", map[string]any{
					"attempt": attempt,
					"error":   err.Error(),
				})

				if attempt < recoveryRetries {
					time.Sleep(time.Duration(attempt) * time.Second)
				}
			}
		}

		span.Event("recovery_exhausted")
	}
}

// updateResult updates the stored health check result
func (hc *HealthChecker) updateResult(result HealthResult) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.lastResult = result
	hc.lastCheck = result.Timestamp
}

// getConnectionKey returns a key identifying this connection
func (hc *HealthChecker) getConnectionKey() string {
	if clientImpl, ok := hc.client.(*sshClient); ok {
		info := clientImpl.GetConnectionInfo()
		if host, ok := info["host"]; ok {
			if port, portOk := info["port"]; portOk {
				if user, userOk := info["user"]; userOk {
					return fmt.Sprintf("%v@%v:%v", user, host, port)
				}
			}
		}
	}
	return "unknown"
}

// formatError safely formats an error for reporting
func (hc *HealthChecker) formatError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// PoolHealthMonitor monitors health for multiple connections
type PoolHealthMonitor struct {
	checkers map[string]*HealthChecker
	tracer   SSHTracer
	mu       sync.RWMutex
	config   HealthCheckConfig
}

// NewPoolHealthMonitor creates a health monitor for connection pools
func NewPoolHealthMonitor(tracer SSHTracer) *PoolHealthMonitor {
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &PoolHealthMonitor{
		checkers: make(map[string]*HealthChecker),
		tracer:   tracer,
		config:   *DefaultHealthCheckConfig(),
	}
}

// AddConnection adds a connection to monitor
func (phm *PoolHealthMonitor) AddConnection(key string, client SSHClient) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	checker := NewHealthChecker(client, phm.tracer)
	checker.SetConfig(phm.config)
	phm.checkers[key] = checker
}

// RemoveConnection removes a connection from monitoring
func (phm *PoolHealthMonitor) RemoveConnection(key string) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	if checker, exists := phm.checkers[key]; exists {
		checker.StopMonitoring()
		delete(phm.checkers, key)
	}
}

// CheckAllHealth checks health of all monitored connections
func (phm *PoolHealthMonitor) CheckAllHealth(ctx context.Context) HealthReport {
	span := phm.tracer.TraceConnection(ctx, "pool_health_check", 0, "")
	defer span.End()

	phm.mu.RLock()
	checkers := make(map[string]*HealthChecker)
	maps.Copy(checkers, phm.checkers)
	phm.mu.RUnlock()

	report := HealthReport{
		TotalConnections:   len(checkers),
		HealthyConnections: 0,
		FailedConnections:  0,
		CheckedAt:          time.Now(),
		Connections:        make([]ConnectionHealth, 0, len(checkers)),
	}

	for key, checker := range checkers {
		result := checker.CheckHealth(ctx)

		connHealth := ConnectionHealth{
			Key:          key,
			Healthy:      result.Healthy,
			LastUsed:     result.Timestamp,
			ResponseTime: result.ResponseTime,
			Error:        checker.formatError(result.Error),
		}

		if result.Healthy {
			report.HealthyConnections++
		} else {
			report.FailedConnections++
		}

		report.Connections = append(report.Connections, connHealth)
	}

	span.Event("pool_health_checked", map[string]any{
		"total_connections":   report.TotalConnections,
		"healthy_connections": report.HealthyConnections,
		"failed_connections":  report.FailedConnections,
	})

	return report
}

// StartMonitoringAll starts monitoring for all connections
func (phm *PoolHealthMonitor) StartMonitoringAll(ctx context.Context) {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	for _, checker := range phm.checkers {
		checker.StartMonitoring(ctx)
	}
}

// StopMonitoringAll stops monitoring for all connections
func (phm *PoolHealthMonitor) StopMonitoringAll() {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	for _, checker := range phm.checkers {
		checker.StopMonitoring()
	}
}

// SetConfig updates configuration for all health checkers
func (phm *PoolHealthMonitor) SetConfig(config HealthCheckConfig) {
	phm.mu.Lock()
	defer phm.mu.Unlock()

	phm.config = config
	for _, checker := range phm.checkers {
		checker.SetConfig(config)
	}
}

// GetUnhealthyConnections returns keys of unhealthy connections
func (phm *PoolHealthMonitor) GetUnhealthyConnections(ctx context.Context) []string {
	phm.mu.RLock()
	defer phm.mu.RUnlock()

	var unhealthy []string
	for key, checker := range phm.checkers {
		if !checker.IsHealthy(ctx) {
			unhealthy = append(unhealthy, key)
		}
	}

	return unhealthy
}

// AdvancedHealthMonitor provides comprehensive health monitoring with metrics and predictions
type AdvancedHealthMonitor struct {
	checker   *HealthChecker
	tracer    SSHTracer
	metrics   *HealthMetrics
	predictor *HealthPredictor
	config    *AdvancedMonitoringConfig
	mu        sync.RWMutex
}

// AdvancedMonitoringConfig holds configuration for advanced monitoring
type AdvancedMonitoringConfig struct {
	MetricsRetention    time.Duration
	PredictionWindow    time.Duration
	PerformanceInterval time.Duration
	AlertThresholds     *HealthThresholds
	EnablePrediction    bool
	EnableMetrics       bool
	MetricsBufferSize   int
	PredictionSamples   int
	PerformanceTests    []PerformanceTest
}

// DefaultAdvancedMonitoringConfig returns default advanced monitoring configuration
func DefaultAdvancedMonitoringConfig() *AdvancedMonitoringConfig {
	return &AdvancedMonitoringConfig{
		MetricsRetention:    24 * time.Hour,
		PredictionWindow:    1 * time.Hour,
		PerformanceInterval: 5 * time.Minute,
		AlertThresholds:     DefaultHealthThresholds(),
		EnablePrediction:    true,
		EnableMetrics:       true,
		MetricsBufferSize:   1000,
		PredictionSamples:   50,
		PerformanceTests: []PerformanceTest{
			{Name: "latency", Command: "echo test", Timeout: 5 * time.Second},
			{Name: "cpu", Command: "cat /proc/loadavg", Timeout: 5 * time.Second},
			{Name: "memory", Command: "free -m", Timeout: 5 * time.Second},
			{Name: "disk", Command: "df -h /", Timeout: 5 * time.Second},
		},
	}
}

// HealthThresholds defines thresholds for health alerts
type HealthThresholds struct {
	MaxResponseTime     time.Duration
	MaxConsecutiveFails int
	MinSuccessRate      float64
	MaxMemoryUsage      float64
	MaxCPUUsage         float64
	MinDiskSpace        float64
}

// DefaultHealthThresholds returns default health thresholds
func DefaultHealthThresholds() *HealthThresholds {
	return &HealthThresholds{
		MaxResponseTime:     5 * time.Second,
		MaxConsecutiveFails: 5,
		MinSuccessRate:      0.95, // 95%
		MaxMemoryUsage:      0.90, // 90%
		MaxCPUUsage:         0.80, // 80%
		MinDiskSpace:        0.10, // 10% free
	}
}

// PerformanceTest defines a performance test
type PerformanceTest struct {
	Name     string
	Command  string
	Timeout  time.Duration
	Interval time.Duration
}

// HealthMetrics tracks health metrics over time
type HealthMetrics struct {
	samples         []HealthSample
	performanceData map[string][]PerformanceSample
	mu              sync.RWMutex
	maxSamples      int
	startTime       time.Time
}

// HealthSample represents a single health check sample
type HealthSample struct {
	Timestamp    time.Time
	Healthy      bool
	ResponseTime time.Duration
	Error        string
}

// PerformanceSample represents a performance metric sample
type PerformanceSample struct {
	Timestamp time.Time
	Value     float64
	Unit      string
	Details   map[string]any
}

// NewHealthMetrics creates a new health metrics tracker
func NewHealthMetrics(maxSamples int) *HealthMetrics {
	return &HealthMetrics{
		samples:         make([]HealthSample, 0, maxSamples),
		performanceData: make(map[string][]PerformanceSample),
		maxSamples:      maxSamples,
		startTime:       time.Now(),
	}
}

// AddSample adds a health sample
func (hm *HealthMetrics) AddSample(result HealthResult) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	sample := HealthSample{
		Timestamp:    result.Timestamp,
		Healthy:      result.Healthy,
		ResponseTime: result.ResponseTime,
		Error:        "",
	}

	if result.Error != nil {
		sample.Error = result.Error.Error()
	}

	// Add sample and maintain max size
	hm.samples = append(hm.samples, sample)
	if len(hm.samples) > hm.maxSamples {
		hm.samples = hm.samples[1:]
	}
}

// AddPerformanceSample adds a performance metric sample
func (hm *HealthMetrics) AddPerformanceSample(metric string, sample PerformanceSample) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.performanceData[metric] == nil {
		hm.performanceData[metric] = make([]PerformanceSample, 0, hm.maxSamples)
	}

	hm.performanceData[metric] = append(hm.performanceData[metric], sample)
	if len(hm.performanceData[metric]) > hm.maxSamples {
		hm.performanceData[metric] = hm.performanceData[metric][1:]
	}
}

// GetSuccessRate calculates success rate over a time window
func (hm *HealthMetrics) GetSuccessRate(window time.Duration) float64 {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if len(hm.samples) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-window)
	var total, successful int

	for _, sample := range hm.samples {
		if sample.Timestamp.After(cutoff) {
			total++
			if sample.Healthy {
				successful++
			}
		}
	}

	if total == 0 {
		return 0
	}

	return float64(successful) / float64(total)
}

// GetAverageResponseTime calculates average response time
func (hm *HealthMetrics) GetAverageResponseTime(window time.Duration) time.Duration {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if len(hm.samples) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-window)
	var total time.Duration
	var count int

	for _, sample := range hm.samples {
		if sample.Timestamp.After(cutoff) && sample.Healthy {
			total += sample.ResponseTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// HealthPredictor provides predictive health analysis
type HealthPredictor struct {
	metrics *HealthMetrics
	mu      sync.RWMutex
}

// NewHealthPredictor creates a new health predictor
func NewHealthPredictor(metrics *HealthMetrics) *HealthPredictor {
	return &HealthPredictor{
		metrics: metrics,
	}
}

// PredictHealthTrend predicts health trend based on historical data
func (hp *HealthPredictor) PredictHealthTrend(window time.Duration, samples int) *HealthPrediction {
	hp.mu.RLock()
	defer hp.mu.RUnlock()

	prediction := &HealthPrediction{
		Timestamp:   time.Now(),
		Window:      window,
		Confidence:  0.0,
		TrendType:   TrendStable,
		Predictions: make([]HealthForecast, 0),
		Insights:    make([]string, 0),
		Risks:       make([]RiskFactor, 0),
	}

	// Get recent samples
	recentSamples := hp.getRecentSamples(window, samples)
	if len(recentSamples) < 10 {
		prediction.Confidence = 0.1
		prediction.Insights = append(prediction.Insights, "Insufficient data for reliable prediction")
		return prediction
	}

	// Analyze trends
	successTrend := hp.analyzeSuccessTrend(recentSamples)
	responseTrend := hp.analyzeResponseTimeTrend(recentSamples)

	// Determine overall trend
	if successTrend < -0.1 || responseTrend > 0.2 {
		prediction.TrendType = TrendDegrading
	} else if successTrend > 0.1 && responseTrend < -0.1 {
		prediction.TrendType = TrendImproving
	} else {
		prediction.TrendType = TrendStable
	}

	// Calculate confidence based on data consistency
	prediction.Confidence = hp.calculateConfidence(recentSamples)

	// Generate forecasts
	prediction.Predictions = hp.generateForecasts(recentSamples, 3)

	// Identify risks
	prediction.Risks = hp.identifyRisks(recentSamples)

	// Generate insights
	prediction.Insights = hp.generateInsights(recentSamples, successTrend, responseTrend)

	return prediction
}

// getRecentSamples gets recent health samples within the window
func (hp *HealthPredictor) getRecentSamples(window time.Duration, maxSamples int) []HealthSample {
	hp.metrics.mu.RLock()
	defer hp.metrics.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	var recent []HealthSample

	for _, sample := range hp.metrics.samples {
		if sample.Timestamp.After(cutoff) {
			recent = append(recent, sample)
		}
	}

	// Limit to max samples
	if len(recent) > maxSamples {
		recent = recent[len(recent)-maxSamples:]
	}

	return recent
}

// analyzeSuccessTrend analyzes the success rate trend
func (hp *HealthPredictor) analyzeSuccessTrend(samples []HealthSample) float64 {
	if len(samples) < 2 {
		return 0
	}

	// Calculate success rate for first and second half
	mid := len(samples) / 2
	firstHalf := samples[:mid]
	secondHalf := samples[mid:]

	firstSuccess := hp.calculateSuccessRate(firstHalf)
	secondSuccess := hp.calculateSuccessRate(secondHalf)

	return secondSuccess - firstSuccess
}

// analyzeResponseTimeTrend analyzes the response time trend
func (hp *HealthPredictor) analyzeResponseTimeTrend(samples []HealthSample) float64 {
	if len(samples) < 2 {
		return 0
	}

	// Calculate average response time for first and second half
	mid := len(samples) / 2
	firstHalf := samples[:mid]
	secondHalf := samples[mid:]

	firstAvg := hp.calculateAverageResponseTime(firstHalf)
	secondAvg := hp.calculateAverageResponseTime(secondHalf)

	if firstAvg == 0 {
		return 0
	}

	return (secondAvg.Seconds() - firstAvg.Seconds()) / firstAvg.Seconds()
}

// calculateSuccessRate calculates success rate for given samples
func (hp *HealthPredictor) calculateSuccessRate(samples []HealthSample) float64 {
	if len(samples) == 0 {
		return 0
	}

	var successful int
	for _, sample := range samples {
		if sample.Healthy {
			successful++
		}
	}

	return float64(successful) / float64(len(samples))
}

// calculateAverageResponseTime calculates average response time for given samples
func (hp *HealthPredictor) calculateAverageResponseTime(samples []HealthSample) time.Duration {
	if len(samples) == 0 {
		return 0
	}

	var total time.Duration
	var count int

	for _, sample := range samples {
		if sample.Healthy {
			total += sample.ResponseTime
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// calculateConfidence calculates prediction confidence based on data consistency
func (hp *HealthPredictor) calculateConfidence(samples []HealthSample) float64 {
	if len(samples) < 10 {
		return 0.3
	}

	// Calculate variance in response times
	avg := hp.calculateAverageResponseTime(samples)
	if avg == 0 {
		return 0.5
	}

	var variance float64
	var count int
	for _, sample := range samples {
		if sample.Healthy {
			diff := sample.ResponseTime.Seconds() - avg.Seconds()
			variance += diff * diff
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	variance /= float64(count)
	stdDev := math.Sqrt(variance)

	// Lower variance = higher confidence
	coefficient := stdDev / avg.Seconds()
	confidence := math.Max(0.1, math.Min(0.95, 1.0-coefficient))

	return confidence
}

// generateForecasts generates health forecasts
func (hp *HealthPredictor) generateForecasts(samples []HealthSample, periods int) []HealthForecast {
	forecasts := make([]HealthForecast, 0, periods)

	if len(samples) == 0 {
		return forecasts
	}

	currentTime := time.Now()
	interval := 10 * time.Minute // Forecast every 10 minutes

	for i := 1; i <= periods; i++ {
		forecastTime := currentTime.Add(time.Duration(i) * interval)

		// Simple trend-based prediction
		successRate := hp.calculateSuccessRate(samples)
		avgResponseTime := hp.calculateAverageResponseTime(samples)

		forecast := HealthForecast{
			Timestamp:        forecastTime,
			PredictedHealthy: successRate > 0.8,
			SuccessRate:      successRate,
			ResponseTime:     avgResponseTime,
			Confidence:       hp.calculateConfidence(samples),
		}

		forecasts = append(forecasts, forecast)
	}

	return forecasts
}

// identifyRisks identifies potential risk factors
func (hp *HealthPredictor) identifyRisks(samples []HealthSample) []RiskFactor {
	risks := make([]RiskFactor, 0)

	if len(samples) == 0 {
		return risks
	}

	// Analyze failure patterns
	recentFailures := 0
	responseTimeIncreasing := false

	for i := len(samples) - 10; i < len(samples) && i >= 0; i++ {
		if !samples[i].Healthy {
			recentFailures++
		}
	}

	// Check for increasing response times
	if len(samples) >= 10 {
		oldAvg := hp.calculateAverageResponseTime(samples[:5])
		newAvg := hp.calculateAverageResponseTime(samples[len(samples)-5:])
		if newAvg > oldAvg*2 {
			responseTimeIncreasing = true
		}
	}

	// High failure rate risk
	if recentFailures > 3 {
		risks = append(risks, RiskFactor{
			Type:        RiskHighFailureRate,
			Severity:    RiskSeverityHigh,
			Description: fmt.Sprintf("High failure rate: %d failures in last 10 checks", recentFailures),
			Probability: 0.8,
			Impact:      RiskImpactHigh,
		})
	}

	// Performance degradation risk
	if responseTimeIncreasing {
		risks = append(risks, RiskFactor{
			Type:        RiskPerformanceDegradation,
			Severity:    RiskSeverityMedium,
			Description: "Response times are increasing significantly",
			Probability: 0.6,
			Impact:      RiskImpactMedium,
		})
	}

	return risks
}

// generateInsights generates actionable insights
func (hp *HealthPredictor) generateInsights(samples []HealthSample, successTrend, responseTrend float64) []string {
	insights := make([]string, 0)

	if len(samples) == 0 {
		return insights
	}

	// Success rate insights
	successRate := hp.calculateSuccessRate(samples)
	if successRate < 0.8 {
		insights = append(insights, fmt.Sprintf("Success rate is low (%.1f%%). Consider investigating connection stability.", successRate*100))
	}

	// Response time insights
	avgResponseTime := hp.calculateAverageResponseTime(samples)
	if avgResponseTime > 2*time.Second {
		insights = append(insights, fmt.Sprintf("Average response time is high (%.2fs). Network latency may be an issue.", avgResponseTime.Seconds()))
	}

	// Trend insights
	if successTrend < -0.1 {
		insights = append(insights, "Success rate is trending downward. Monitor for potential issues.")
	}

	if responseTrend > 0.2 {
		insights = append(insights, "Response times are increasing. Consider checking server load.")
	}

	if len(insights) == 0 {
		insights = append(insights, "Connection health appears stable with no immediate concerns.")
	}

	return insights
}

// NewAdvancedHealthMonitor creates a new advanced health monitor
func NewAdvancedHealthMonitor(checker *HealthChecker, tracer SSHTracer, config *AdvancedMonitoringConfig) *AdvancedHealthMonitor {
	if config == nil {
		config = DefaultAdvancedMonitoringConfig()
	}
	if tracer == nil {
		tracer = &NoOpTracer{}
	}

	return &AdvancedHealthMonitor{
		checker:   checker,
		tracer:    tracer,
		metrics:   NewHealthMetrics(config.MetricsBufferSize),
		predictor: NewHealthPredictor(NewHealthMetrics(config.PredictionSamples)),
		config:    config,
	}
}

// DeepHealthCheck performs comprehensive health analysis
func (ahm *AdvancedHealthMonitor) DeepHealthCheck(ctx context.Context) (*DetailedHealthReport, error) {
	span := ahm.tracer.TraceConnection(ctx, "deep_health_check", 0, "")
	defer span.End()

	report := &DetailedHealthReport{
		Timestamp:   time.Now(),
		BasicHealth: ahm.checker.CheckHealth(ctx),
		Performance: &PerformanceReport{},
		Diagnostics: make([]DiagnosticCheck, 0),
		Metrics:     ahm.getHealthMetrics(),
		Predictions: ahm.predictor.PredictHealthTrend(ahm.config.PredictionWindow, ahm.config.PredictionSamples),
		Alerts:      make([]HealthAlert, 0),
	}

	// Perform performance tests
	performance, err := ahm.runPerformanceTests(ctx)
	if err != nil {
		span.EndWithError(err)
		return report, fmt.Errorf("performance tests failed: %w", err)
	}
	report.Performance = performance

	// Run diagnostic checks
	diagnostics := ahm.runDiagnosticChecks(ctx)
	report.Diagnostics = diagnostics

	// Generate alerts based on thresholds
	alerts := ahm.generateAlerts(report)
	report.Alerts = alerts

	// Calculate overall health score
	report.OverallScore = ahm.calculateOverallScore(report)

	span.Event("deep_health_check_completed", map[string]any{
		"overall_score":    report.OverallScore,
		"alert_count":      len(alerts),
		"diagnostic_count": len(diagnostics),
	})

	return report, nil
}

// PredictiveAnalysis performs predictive analysis
func (ahm *AdvancedHealthMonitor) PredictiveAnalysis(ctx context.Context) (*HealthPrediction, error) {
	span := ahm.tracer.TraceConnection(ctx, "predictive_analysis", 0, "")
	defer span.End()

	if !ahm.config.EnablePrediction {
		return nil, fmt.Errorf("predictive analysis is disabled")
	}

	prediction := ahm.predictor.PredictHealthTrend(ahm.config.PredictionWindow, ahm.config.PredictionSamples)

	span.Event("predictive_analysis_completed", map[string]any{
		"trend_type": string(prediction.TrendType),
		"confidence": prediction.Confidence,
		"risk_count": len(prediction.Risks),
	})

	return prediction, nil
}

// AutoRecover attempts automatic recovery using specified strategy
func (ahm *AdvancedHealthMonitor) AutoRecover(ctx context.Context, strategy RecoveryStrategy) error {
	span := ahm.tracer.TraceConnection(ctx, "auto_recovery", 0, "")
	defer span.End()

	span.SetFields(map[string]any{
		"strategy": string(strategy),
	})

	switch strategy {
	case RecoveryStrategyReconnect:
		return ahm.executeReconnectStrategy(ctx)
	case RecoveryStrategyRestart:
		return ahm.executeRestartStrategy(ctx)
	case RecoveryStrategyReset:
		return ahm.executeResetStrategy(ctx)
	case RecoveryStrategyEscalate:
		return ahm.executeEscalateStrategy(ctx)
	default:
		return fmt.Errorf("unsupported recovery strategy: %s", strategy)
	}
}

// GetPerformanceMetrics retrieves current performance metrics
func (ahm *AdvancedHealthMonitor) GetPerformanceMetrics(ctx context.Context) (*PerformanceReport, error) {
	span := ahm.tracer.TraceConnection(ctx, "get_performance_metrics", 0, "")
	defer span.End()

	if !ahm.config.EnableMetrics {
		return nil, fmt.Errorf("performance metrics collection is disabled")
	}

	report, err := ahm.runPerformanceTests(ctx)
	if err != nil {
		span.EndWithError(err)
		return nil, fmt.Errorf("failed to collect performance metrics: %w", err)
	}

	span.Event("performance_metrics_collected", map[string]any{
		"test_count": len(report.Tests),
	})

	return report, nil
}

// runPerformanceTests executes performance tests and collects metrics
func (ahm *AdvancedHealthMonitor) runPerformanceTests(ctx context.Context) (*PerformanceReport, error) {
	report := &PerformanceReport{
		Timestamp:   time.Now(),
		Tests:       make([]PerformanceTestResult, 0),
		SystemInfo:  &SystemInfo{},
		NetworkInfo: &NetworkInfo{},
	}

	// Run configured performance tests
	for _, test := range ahm.config.PerformanceTests {
		testCtx, cancel := context.WithTimeout(ctx, test.Timeout)

		start := time.Now()
		output, err := ahm.checker.client.Execute(testCtx, test.Command)
		duration := time.Since(start)
		cancel()

		result := PerformanceTestResult{
			Name:      test.Name,
			Duration:  duration,
			Success:   err == nil,
			Output:    output,
			Timestamp: start,
		}

		if err != nil {
			result.Error = err.Error()
		}

		// Parse specific test results
		switch test.Name {
		case "cpu":
			result.Metrics = ahm.parseCPUMetrics(output)
		case "memory":
			result.Metrics = ahm.parseMemoryMetrics(output)
		case "disk":
			result.Metrics = ahm.parseDiskMetrics(output)
		case "latency":
			result.Metrics = map[string]float64{
				"response_time_ms": float64(duration.Nanoseconds()) / 1e6,
			}
		}

		report.Tests = append(report.Tests, result)
	}

	// Collect system information
	ahm.collectSystemInfo(ctx, report.SystemInfo)
	ahm.collectNetworkInfo(ctx, report.NetworkInfo)

	return report, nil
}

// runDiagnosticChecks performs diagnostic checks
func (ahm *AdvancedHealthMonitor) runDiagnosticChecks(ctx context.Context) []DiagnosticCheck {
	checks := make([]DiagnosticCheck, 0)

	// Connection diagnostic
	connCheck := DiagnosticCheck{
		Name:      "Connection Status",
		Category:  "connectivity",
		Timestamp: time.Now(),
		Status:    DiagnosticStatusPass,
	}

	if !ahm.checker.client.IsConnected() {
		connCheck.Status = DiagnosticStatusFail
		connCheck.Message = "SSH connection is not established"
		connCheck.Recommendations = []string{"Verify network connectivity", "Check SSH service status", "Validate credentials"}
	} else {
		connCheck.Message = "SSH connection is healthy"
	}

	checks = append(checks, connCheck)

	// Performance diagnostic
	perfCheck := ahm.runPerformanceDiagnostic(ctx)
	checks = append(checks, perfCheck)

	// Security diagnostic
	secCheck := ahm.runSecurityDiagnostic(ctx)
	checks = append(checks, secCheck)

	return checks
}

// runPerformanceDiagnostic runs performance-related diagnostics
func (ahm *AdvancedHealthMonitor) runPerformanceDiagnostic(ctx context.Context) DiagnosticCheck {
	check := DiagnosticCheck{
		Name:      "Performance Analysis",
		Category:  "performance",
		Timestamp: time.Now(),
		Status:    DiagnosticStatusPass,
	}

	// Get recent response time average
	avgResponseTime := ahm.metrics.GetAverageResponseTime(10 * time.Minute)

	if avgResponseTime > ahm.config.AlertThresholds.MaxResponseTime {
		check.Status = DiagnosticStatusWarning
		check.Message = fmt.Sprintf("Average response time (%.2fs) exceeds threshold", avgResponseTime.Seconds())
		check.Recommendations = []string{
			"Check server load",
			"Verify network latency",
			"Consider connection optimization",
		}
	} else {
		check.Message = fmt.Sprintf("Performance is healthy (avg response: %.2fs)", avgResponseTime.Seconds())
	}

	check.Details = map[string]any{
		"average_response_time": avgResponseTime.Seconds(),
		"threshold":             ahm.config.AlertThresholds.MaxResponseTime.Seconds(),
	}

	return check
}

// runSecurityDiagnostic runs security-related diagnostics
func (ahm *AdvancedHealthMonitor) runSecurityDiagnostic(ctx context.Context) DiagnosticCheck {
	check := DiagnosticCheck{
		Name:      "Security Status",
		Category:  "security",
		Timestamp: time.Now(),
		Status:    DiagnosticStatusPass,
		Message:   "Security checks passed",
	}

	// Check for authentication failures in recent samples
	ahm.metrics.mu.RLock()
	recentSamples := ahm.metrics.samples
	if len(recentSamples) > 10 {
		recentSamples = recentSamples[len(recentSamples)-10:]
	}
	ahm.metrics.mu.RUnlock()

	authFailures := 0
	for _, sample := range recentSamples {
		if !sample.Healthy && strings.Contains(strings.ToLower(sample.Error), "auth") {
			authFailures++
		}
	}

	if authFailures > 2 {
		check.Status = DiagnosticStatusFail
		check.Message = fmt.Sprintf("Multiple authentication failures detected (%d)", authFailures)
		check.Recommendations = []string{
			"Verify SSH credentials",
			"Check for unauthorized access attempts",
			"Review security logs",
		}
	}

	check.Details = map[string]any{
		"auth_failures": authFailures,
		"sample_size":   len(recentSamples),
	}

	return check
}

// generateAlerts generates health alerts based on thresholds
func (ahm *AdvancedHealthMonitor) generateAlerts(report *DetailedHealthReport) []HealthAlert {
	alerts := make([]HealthAlert, 0)

	// Check success rate
	successRate := ahm.metrics.GetSuccessRate(30 * time.Minute)
	if successRate < ahm.config.AlertThresholds.MinSuccessRate {
		alerts = append(alerts, HealthAlert{
			Type:      AlertTypeHealth,
			Severity:  AlertSeverityHigh,
			Title:     "Low Success Rate",
			Message:   fmt.Sprintf("Success rate (%.1f%%) below threshold (%.1f%%)", successRate*100, ahm.config.AlertThresholds.MinSuccessRate*100),
			Timestamp: time.Now(),
			Metadata:  map[string]any{"current_rate": successRate, "threshold": ahm.config.AlertThresholds.MinSuccessRate},
		})
	}

	// Check response time
	avgResponseTime := ahm.metrics.GetAverageResponseTime(30 * time.Minute)
	if avgResponseTime > ahm.config.AlertThresholds.MaxResponseTime {
		alerts = append(alerts, HealthAlert{
			Type:      AlertTypePerformance,
			Severity:  AlertSeverityMedium,
			Title:     "High Response Time",
			Message:   fmt.Sprintf("Average response time (%.2fs) exceeds threshold (%.2fs)", avgResponseTime.Seconds(), ahm.config.AlertThresholds.MaxResponseTime.Seconds()),
			Timestamp: time.Now(),
			Metadata:  map[string]any{"current_time": avgResponseTime.Seconds(), "threshold": ahm.config.AlertThresholds.MaxResponseTime.Seconds()},
		})
	}

	return alerts
}

// calculateOverallScore calculates an overall health score
func (ahm *AdvancedHealthMonitor) calculateOverallScore(report *DetailedHealthReport) float64 {
	score := 100.0

	// Deduct for basic health issues
	if !report.BasicHealth.Healthy {
		score -= 30
	}

	// Deduct for alerts
	for _, alert := range report.Alerts {
		switch alert.Severity {
		case AlertSeverityHigh:
			score -= 20
		case AlertSeverityMedium:
			score -= 10
		case AlertSeverityLow:
			score -= 5
		}
	}

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	return score
}

// getHealthMetrics returns current health metrics summary
func (ahm *AdvancedHealthMonitor) getHealthMetrics() *HealthMetricsSummary {
	ahm.metrics.mu.RLock()
	defer ahm.metrics.mu.RUnlock()

	summary := &HealthMetricsSummary{
		TotalSamples:    len(ahm.metrics.samples),
		SuccessRate:     ahm.metrics.GetSuccessRate(24 * time.Hour),
		AverageResponse: ahm.metrics.GetAverageResponseTime(24 * time.Hour),
		DataRetention:   ahm.config.MetricsRetention,
	}

	if len(ahm.metrics.samples) > 0 {
		summary.OldestSample = ahm.metrics.samples[0].Timestamp
		summary.LatestSample = ahm.metrics.samples[len(ahm.metrics.samples)-1].Timestamp
	}

	return summary
}

// executeReconnectStrategy implements reconnect recovery strategy
func (ahm *AdvancedHealthMonitor) executeReconnectStrategy(ctx context.Context) error {
	span := ahm.tracer.TraceConnection(ctx, "recovery_reconnect", 0, "")
	defer span.End()

	span.Event("reconnect_started")

	// Use the health checker's recovery method
	err := ahm.checker.RecoverConnection(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("reconnect strategy failed: %w", err)
	}

	span.Event("reconnect_completed")
	return nil
}

// executeRestartStrategy implements restart recovery strategy
func (ahm *AdvancedHealthMonitor) executeRestartStrategy(ctx context.Context) error {
	span := ahm.tracer.TraceConnection(ctx, "recovery_restart", 0, "")
	defer span.End()

	span.Event("restart_started")

	// Stop current monitoring
	ahm.checker.StopMonitoring()

	// Close and reconnect
	if err := ahm.checker.client.Close(); err != nil {
		span.Event("close_failed", map[string]any{"error": err.Error()})
	}

	time.Sleep(2 * time.Second) // Brief pause

	if err := ahm.checker.client.Connect(ctx); err != nil {
		span.EndWithError(err)
		return fmt.Errorf("restart strategy failed: %w", err)
	}

	// Restart monitoring
	ahm.checker.StartMonitoring(ctx)

	span.Event("restart_completed")
	return nil
}

// executeResetStrategy implements reset recovery strategy
func (ahm *AdvancedHealthMonitor) executeResetStrategy(ctx context.Context) error {
	span := ahm.tracer.TraceConnection(ctx, "recovery_reset", 0, "")
	defer span.End()

	span.Event("reset_started")

	// Reset metrics and prediction data
	ahm.metrics = NewHealthMetrics(ahm.config.MetricsBufferSize)
	ahm.predictor = NewHealthPredictor(ahm.metrics)

	// Perform restart strategy
	err := ahm.executeRestartStrategy(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("reset strategy failed: %w", err)
	}

	span.Event("reset_completed")
	return nil
}

// executeEscalateStrategy implements escalate recovery strategy
func (ahm *AdvancedHealthMonitor) executeEscalateStrategy(ctx context.Context) error {
	span := ahm.tracer.TraceConnection(ctx, "recovery_escalate", 0, "")
	defer span.End()

	span.Event("escalation_started")

	// Generate detailed report for escalation
	report, err := ahm.DeepHealthCheck(ctx)
	if err != nil {
		span.EndWithError(err)
		return fmt.Errorf("escalation failed to generate report: %w", err)
	}

	// Log escalation details
	span.Event("escalation_details", map[string]any{
		"overall_score": report.OverallScore,
		"alert_count":   len(report.Alerts),
		"failed_tests":  ahm.countFailedTests(report.Performance),
	})

	// In a real implementation, this would trigger external escalation
	// (notifications, tickets, etc.)
	span.Event("escalation_triggered")

	return fmt.Errorf("connection requires manual intervention - escalation triggered")
}

// countFailedTests counts failed performance tests
func (ahm *AdvancedHealthMonitor) countFailedTests(performance *PerformanceReport) int {
	if performance == nil {
		return 0
	}

	failed := 0
	for _, test := range performance.Tests {
		if !test.Success {
			failed++
		}
	}
	return failed
}

// parseCPUMetrics parses CPU metrics from loadavg output
func (ahm *AdvancedHealthMonitor) parseCPUMetrics(output string) map[string]float64 {
	metrics := make(map[string]float64)

	// Parse /proc/loadavg output: "0.15 0.05 0.01 1/123 456"
	var load1, load5, load15 float64
	if n, _ := fmt.Sscanf(output, "%f %f %f", &load1, &load5, &load15); n >= 3 {
		metrics["load_1min"] = load1
		metrics["load_5min"] = load5
		metrics["load_15min"] = load15
	}

	return metrics
}

// parseMemoryMetrics parses memory metrics from free output
func (ahm *AdvancedHealthMonitor) parseMemoryMetrics(output string) map[string]float64 {
	metrics := make(map[string]float64)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Mem:") {
			fields := strings.Fields(line)
			if len(fields) >= 7 {
				total := parseFloat(fields[1])
				used := parseFloat(fields[2])
				free := parseFloat(fields[3])
				available := parseFloat(fields[6])

				metrics["memory_total_mb"] = total
				metrics["memory_used_mb"] = used
				metrics["memory_free_mb"] = free
				metrics["memory_available_mb"] = available

				if total > 0 {
					metrics["memory_usage_percent"] = (used / total) * 100
				}
			}
			break
		}
	}

	return metrics
}

// parseDiskMetrics parses disk metrics from df output
func (ahm *AdvancedHealthMonitor) parseDiskMetrics(output string) map[string]float64 {
	metrics := make(map[string]float64)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "/") && !strings.Contains(line, "Filesystem") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				// Parse size, used, available
				size := parseDiskSize(fields[1])
				used := parseDiskSize(fields[2])
				available := parseDiskSize(fields[3])

				metrics["disk_total_gb"] = size / (1024 * 1024 * 1024)
				metrics["disk_used_gb"] = used / (1024 * 1024 * 1024)
				metrics["disk_available_gb"] = available / (1024 * 1024 * 1024)

				if size > 0 {
					metrics["disk_usage_percent"] = (float64(used) / float64(size)) * 100
				}
			}
			break
		}
	}

	return metrics
}

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

// parseDiskSize parses disk size with units (K, M, G, T)
func parseDiskSize(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Remove unit suffix and parse
	unit := strings.ToUpper(s[len(s)-1:])
	numStr := s
	if unit == "K" || unit == "M" || unit == "G" || unit == "T" {
		numStr = s[:len(s)-1]
	}

	val := parseFloat(numStr)

	switch unit {
	case "K":
		return val * 1024
	case "M":
		return val * 1024 * 1024
	case "G":
		return val * 1024 * 1024 * 1024
	case "T":
		return val * 1024 * 1024 * 1024 * 1024
	default:
		return val
	}
}

// collectSystemInfo collects system information
func (ahm *AdvancedHealthMonitor) collectSystemInfo(ctx context.Context, info *SystemInfo) {
	// Get system uptime
	if output, err := ahm.checker.client.Execute(ctx, "uptime -s"); err == nil {
		if uptime, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(output)); err == nil {
			info.Uptime = time.Since(uptime)
		}
	}

	// Get kernel version
	if output, err := ahm.checker.client.Execute(ctx, "uname -r"); err == nil {
		info.KernelVersion = strings.TrimSpace(output)
	}

	// Get OS info
	if output, err := ahm.checker.client.Execute(ctx, "cat /etc/os-release | grep PRETTY_NAME"); err == nil {
		if parts := strings.Split(output, "="); len(parts) == 2 {
			info.OSVersion = strings.Trim(parts[1], `"`)
		}
	}

	// Get CPU info
	if output, err := ahm.checker.client.Execute(ctx, "nproc"); err == nil {
		info.CPUCores = strings.TrimSpace(output)
	}
}

// collectNetworkInfo collects network information
func (ahm *AdvancedHealthMonitor) collectNetworkInfo(ctx context.Context, info *NetworkInfo) {
	// Get network interfaces
	if output, err := ahm.checker.client.Execute(ctx, "ip addr show | grep 'inet ' | head -5"); err == nil {
		info.Interfaces = strings.Split(strings.TrimSpace(output), "\n")
	}

	// Get network statistics
	if output, err := ahm.checker.client.Execute(ctx, "cat /proc/net/dev | grep -E '^\\s*(eth|en|wl)' | head -3"); err == nil {
		info.NetworkStats = strings.Split(strings.TrimSpace(output), "\n")
	}
}

// StartAdvancedMonitoring starts comprehensive monitoring
func (ahm *AdvancedHealthMonitor) StartAdvancedMonitoring(ctx context.Context) {
	span := ahm.tracer.TraceConnection(ctx, "start_advanced_monitoring", 0, "")
	defer span.End()

	// Start basic monitoring
	ahm.checker.StartMonitoring(ctx)

	// Start metrics collection
	if ahm.config.EnableMetrics {
		go ahm.metricsCollectionLoop(ctx)
	}

	// Start performance monitoring
	go ahm.performanceMonitoringLoop(ctx)

	span.Event("advanced_monitoring_started")
}

// metricsCollectionLoop collects health metrics continuously
func (ahm *AdvancedHealthMonitor) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(ahm.checker.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := ahm.checker.GetLastResult()
			ahm.metrics.AddSample(result)

			// Clean old samples
			ahm.cleanOldMetrics()
		}
	}
}

// performanceMonitoringLoop runs performance monitoring
func (ahm *AdvancedHealthMonitor) performanceMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(ahm.config.PerformanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if performance, err := ahm.runPerformanceTests(ctx); err == nil {
				// Store performance metrics
				ahm.storePerformanceMetrics(performance)
			}
		}
	}
}

// storePerformanceMetrics stores performance test results as metrics
func (ahm *AdvancedHealthMonitor) storePerformanceMetrics(performance *PerformanceReport) {
	for _, test := range performance.Tests {
		if test.Success && test.Metrics != nil {
			for metric, value := range test.Metrics {
				sample := PerformanceSample{
					Timestamp: test.Timestamp,
					Value:     value,
					Unit:      ahm.getMetricUnit(metric),
					Details: map[string]any{
						"test_name": test.Name,
						"duration":  test.Duration.Seconds(),
					},
				}
				ahm.metrics.AddPerformanceSample(metric, sample)
			}
		}
	}
}

// getMetricUnit returns the unit for a metric
func (ahm *AdvancedHealthMonitor) getMetricUnit(metric string) string {
	units := map[string]string{
		"response_time_ms":     "milliseconds",
		"load_1min":            "load",
		"load_5min":            "load",
		"load_15min":           "load",
		"memory_total_mb":      "megabytes",
		"memory_used_mb":       "megabytes",
		"memory_free_mb":       "megabytes",
		"memory_available_mb":  "megabytes",
		"memory_usage_percent": "percentage",
		"disk_total_gb":        "gigabytes",
		"disk_used_gb":         "gigabytes",
		"disk_available_gb":    "gigabytes",
		"disk_usage_percent":   "percentage",
	}

	if unit, exists := units[metric]; exists {
		return unit
	}
	return "unknown"
}

// cleanOldMetrics removes metrics older than retention period
func (ahm *AdvancedHealthMonitor) cleanOldMetrics() {
	ahm.metrics.mu.Lock()
	defer ahm.metrics.mu.Unlock()

	cutoff := time.Now().Add(-ahm.config.MetricsRetention)

	// Clean health samples
	var newSamples []HealthSample
	for _, sample := range ahm.metrics.samples {
		if sample.Timestamp.After(cutoff) {
			newSamples = append(newSamples, sample)
		}
	}
	ahm.metrics.samples = newSamples

	// Clean performance samples
	for metric, samples := range ahm.metrics.performanceData {
		var newPerfSamples []PerformanceSample
		for _, sample := range samples {
			if sample.Timestamp.After(cutoff) {
				newPerfSamples = append(newPerfSamples, sample)
			}
		}
		ahm.metrics.performanceData[metric] = newPerfSamples
	}
}
