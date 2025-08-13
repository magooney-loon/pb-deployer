package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// String manipulation utilities

// ShellEscape escapes a string for safe use in shell commands
func ShellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// FormatDuration formats a duration in a human-readable way
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

// Boolean conversion utilities

// BoolToYesNo converts a boolean to "yes"/"no" string (useful for SSH configs)
func BoolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// BoolToEnabledDisabled converts a boolean to "enabled"/"disabled" string
func BoolToEnabledDisabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

// Math utilities

// Max returns the maximum of two integers
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the minimum of two integers
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Slice utilities

// ContainsString checks if a string slice contains a specific string
func ContainsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ContainsService checks if a service is in the list (case-insensitive)
func ContainsService(services []string, service string) bool {
	lowerService := strings.ToLower(service)
	for _, s := range services {
		if strings.ToLower(s) == lowerService {
			return true
		}
	}
	return false
}

// RemoveFromSlice removes the first occurrence of item from slice
func RemoveFromSlice(slice []string, item string) []string {
	for i, s := range slice {
		if s == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

// UniqueStrings returns a slice with duplicate strings removed
func UniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Validation utilities

// ValidateServiceName validates a service name format
func ValidateServiceName(service string) error {
	if service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Check for invalid characters
	validName := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validName.MatchString(service) {
		return fmt.Errorf("invalid service name: %s", service)
	}

	return nil
}

// ValidateAppName validates an application name format
func ValidateAppName(name string) error {
	if len(name) < 1 || len(name) > 50 {
		return fmt.Errorf("app name must be between 1 and 50 characters")
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_') {
			return fmt.Errorf("invalid character in app name: %c", char)
		}
	}

	return nil
}

// ValidateVersionNumber validates a version number format
func ValidateVersionNumber(version string) error {
	if len(version) < 1 || len(version) > 50 {
		return fmt.Errorf("version number must be between 1 and 50 characters")
	}

	// Allow semantic versioning and other common patterns
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("version number cannot be empty or just whitespace")
	}

	return nil
}

// ValidateDomain validates a domain name format
func ValidateDomain(domain string) error {
	if len(domain) < 3 || len(domain) > 255 {
		return fmt.Errorf("domain must be between 3 and 255 characters")
	}

	// Basic domain validation
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("domain must contain at least one dot")
	}

	return nil
}

// ValidatePort validates a port number
func ValidatePort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// String formatting utilities

// TruncateString truncates a string to maxLength and adds ellipsis if needed
func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}

	if maxLength <= 3 {
		return s[:maxLength]
	}

	return s[:maxLength-3] + "..."
}

// FormatOutput formats command output with line limits
func FormatOutput(output string, maxLines int) string {
	if output == "" {
		return "(no output)"
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= maxLines {
		return output
	}

	// Show first few and last few lines with truncation indicator
	showFirst := maxLines / 2
	showLast := maxLines - showFirst - 1

	var result []string
	result = append(result, lines[:showFirst]...)
	result = append(result, fmt.Sprintf("... [%d lines truncated] ...", len(lines)-maxLines+1))
	result = append(result, lines[len(lines)-showLast:]...)

	return strings.Join(result, "\n")
}

// Each manager implements its own progress reporting using the tunnel package directly

// Configuration management utilities

// ConfigManager provides a common interface for configuration management
type ConfigManager[T any] interface {
	SetConfig(config T)
	GetConfig() T
}

// BaseConfigManager provides base configuration management functionality
type BaseConfigManager[T any] struct {
	config T
}

// SetConfig sets the configuration
func (bcm *BaseConfigManager[T]) SetConfig(config T) {
	bcm.config = config
}

// GetConfig gets the configuration
func (bcm *BaseConfigManager[T]) GetConfig() T {
	return bcm.config
}

// Retry utilities

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

// CalculateRetryDelay calculates the delay for a retry attempt with exponential backoff
func CalculateRetryDelay(attempt int, config RetryConfig) time.Duration {
	if attempt <= 0 {
		return config.BaseDelay
	}

	delay := float64(config.BaseDelay)
	for i := 0; i < attempt; i++ {
		delay *= config.Multiplier
	}

	if time.Duration(delay) > config.MaxDelay {
		return config.MaxDelay
	}

	return time.Duration(delay)
}

// File and path utilities

// SanitizeFilename sanitizes a filename by removing/replacing invalid characters
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscores
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := invalidChars.ReplaceAllString(filename, "_")

	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")

	// Ensure it's not empty
	if sanitized == "" {
		sanitized = "unnamed"
	}

	return sanitized
}

// JoinPaths safely joins path components
func JoinPaths(base string, paths ...string) string {
	result := strings.TrimSuffix(base, "/")

	for _, path := range paths {
		path = strings.Trim(path, "/")
		if path != "" {
			result += "/" + path
		}
	}

	return result
}

// Environment utilities

// IsProductionEnvironment checks if we're running in production
func IsProductionEnvironment() bool {
	// This could be enhanced to check various environment indicators
	return strings.ToLower(strings.TrimSpace(GetEnvVar("ENVIRONMENT", "development"))) == "production"
}

// GetEnvVar gets an environment variable with a default value
func GetEnvVar(key, defaultValue string) string {
	// In a real implementation, this would use os.Getenv
	// For now, return the default value
	return defaultValue
}

// Error utilities

// WrapError wraps an error with additional context
func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

// IsRetryableError checks if an error should be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

	// Common retryable error patterns
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"no route to host",
		"connection reset",
		"broken pipe",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}
