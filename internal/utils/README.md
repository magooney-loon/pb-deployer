# Utils Package

Shared utility functions for common operations across pb-deployer.

## Features

- **Shell Safety**: Proper escaping for shell commands
- **Type Safety**: Generic configuration management
- **Validation**: Comprehensive input validation
- **Error Handling**: Structured error wrapping and retry logic
- **Performance**: Zero-dependency, optimized implementations
- **Testing**: Easy to mock and test individual functions

## Core Functions

```go
// String utilities
func ShellEscape(s string) string
func TruncateString(s string, maxLength int) string
func FormatOutput(output string, maxLines int) string
func FormatDuration(d time.Duration) string

// Boolean converters
func BoolToYesNo(b bool) string
func BoolToEnabledDisabled(b bool) string

// Slice operations
func ContainsString(slice []string, item string) bool
func ContainsService(services []string, service string) bool
func RemoveFromSlice(slice []string, item string) []string
func UniqueStrings(slice []string) []string

// Validation
func ValidateServiceName(service string) error
func ValidateAppName(name string) error
func ValidateVersionNumber(version string) error
func ValidateDomain(domain string) error
func ValidatePort(port int) error

// File utilities
func SanitizeFilename(filename string) string
func JoinPaths(base string, paths ...string) string

// Environment
func IsProductionEnvironment() bool
func GetEnvVar(key, defaultValue string) string

// Error handling
func WrapError(err error, context string) error
func IsRetryableError(err error) bool

// Retry logic
type RetryConfig struct {
    MaxAttempts int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
}

func DefaultRetryConfig() RetryConfig
func CalculateRetryDelay(attempt int, config RetryConfig) time.Duration
```

## Quick Examples

```go
// Shell safety
cmd := fmt.Sprintf("ls %s", utils.ShellEscape(userInput))

// Duration formatting
fmt.Printf("Took %s", utils.FormatDuration(2*time.Minute+30*time.Second))
// Output: "2.5m"

// Validation
if err := utils.ValidateServiceName("my-app"); err != nil {
    return err
}

// Slice operations
if utils.ContainsService([]string{"nginx", "apache"}, "NGINX") {
    // Found (case-insensitive)
}

// Retry logic
config := utils.DefaultRetryConfig()
delay := utils.CalculateRetryDelay(2, config) // 2nd attempt
time.Sleep(delay)

// Error wrapping
return utils.WrapError(err, "failed to deploy application")
```

## Configuration Management

```go
type ConfigManager[T any] interface {
    Load() (T, error)
    Save(config T) error
    Validate(config T) error
}

type BaseConfigManager[T any] struct {
    filepath string
    validate func(T) error
}

func NewConfigManager[T any](filepath string, validator func(T) error) *BaseConfigManager[T]
```
