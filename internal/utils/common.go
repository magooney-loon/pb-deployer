package utils

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

func ShellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	} else {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
}

func BoolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func BoolToEnabledDisabled(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ContainsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func ContainsService(services []string, service string) bool {
	return slices.ContainsFunc(services, func(s string) bool {
		return strings.EqualFold(s, service)
	})
}

func RemoveFromSlice(slice []string, item string) []string {
	for i, s := range slice {
		if s == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

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

func ValidateServiceName(service string) error {
	if service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !validName.MatchString(service) {
		return fmt.Errorf("invalid service name: %s", service)
	}

	return nil
}

func ValidateAppName(name string) error {
	if len(name) < 1 || len(name) > 50 {
		return fmt.Errorf("app name must be between 1 and 50 characters")
	}

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

func ValidateVersionNumber(version string) error {
	if len(version) < 1 || len(version) > 50 {
		return fmt.Errorf("version number must be between 1 and 50 characters")
	}

	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("version number cannot be empty or just whitespace")
	}

	return nil
}

func ValidateDomain(domain string) error {
	if len(domain) < 3 || len(domain) > 255 {
		return fmt.Errorf("domain must be between 3 and 255 characters")
	}

	if !strings.Contains(domain, ".") {
		return fmt.Errorf("domain must contain at least one dot")
	}

	return nil
}

func ValidatePort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

func TruncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}

	if maxLength <= 3 {
		return s[:maxLength]
	}

	return s[:maxLength-3] + "..."
}

func FormatOutput(output string, maxLines int) string {
	if output == "" {
		return "(no output)"
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) <= maxLines {
		return output
	}

	showFirst := maxLines / 2
	showLast := maxLines - showFirst - 1

	var result []string
	result = append(result, lines[:showFirst]...)
	result = append(result, fmt.Sprintf("... [%d lines truncated] ...", len(lines)-maxLines+1))
	result = append(result, lines[len(lines)-showLast:]...)

	return strings.Join(result, "\n")
}

type ConfigManager[T any] interface {
	SetConfig(config T)
	GetConfig() T
}

type BaseConfigManager[T any] struct {
	config T
}

func (bcm *BaseConfigManager[T]) SetConfig(config T) {
	bcm.config = config
}

func (bcm *BaseConfigManager[T]) GetConfig() T {
	return bcm.config
}

type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
	}
}

func CalculateRetryDelay(attempt int, config RetryConfig) time.Duration {
	if attempt <= 0 {
		return config.BaseDelay
	}

	delay := float64(config.BaseDelay)
	for range attempt {
		delay *= config.Multiplier
	}

	if time.Duration(delay) > config.MaxDelay {
		return config.MaxDelay
	}

	return time.Duration(delay)
}

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

func IsProductionEnvironment() bool {
	// This could be enhanced to check various environment indicators
	return strings.ToLower(strings.TrimSpace(GetEnvVar("ENVIRONMENT", "development"))) == "production"
}

func GetEnvVar(key, defaultValue string) string {
	// In a real implementation, this would use os.Getenv
	// For now, return the default value
	return defaultValue
}

func WrapError(err error, context string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", context, err)
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

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
