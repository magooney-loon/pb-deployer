# Common Utilities Package

This package contains shared utility functions and helpers that are used across the pb-deployer codebase, particularly in the tunnel managers and deployment handlers.

## Overview

The utilities have been extracted from various managers to eliminate code duplication and provide a centralized location for common functionality.

## Categories

### String Manipulation Utilities

- **`ShellEscape(s string) string`** - Escapes a string for safe use in shell commands
- **`FormatDuration(d time.Duration) string`** - Formats a duration in human-readable format (e.g., "2.5m", "1.2h")
- **`TruncateString(s string, maxLength int) string`** - Truncates a string with ellipsis if needed
- **`FormatOutput(output string, maxLines int) string`** - Formats command output with line limits and truncation

### Boolean Conversion Utilities

- **`BoolToYesNo(b bool) string`** - Converts boolean to "yes"/"no" (useful for SSH configs)
- **`BoolToEnabledDisabled(b bool) string`** - Converts boolean to "enabled"/"disabled"

### Math Utilities

- **`Max(a, b int) int`** - Returns the maximum of two integers
- **`Min(a, b int) int`** - Returns the minimum of two integers

### Slice Utilities

- **`ContainsString(slice []string, item string) bool`** - Checks if string slice contains item
- **`ContainsService(services []string, service string) bool`** - Case-insensitive service name check
- **`RemoveFromSlice(slice []string, item string) []string`** - Removes first occurrence of item
- **`UniqueStrings(slice []string) []string`** - Returns slice with duplicates removed

### Validation Utilities

- **`ValidateServiceName(service string) error`** - Validates service name format
- **`ValidateAppName(name string) error`** - Validates application name format
- **`ValidateVersionNumber(version string) error`** - Validates version number format
- **`ValidateDomain(domain string) error`** - Validates domain name format
- **`ValidatePort(port int) error`** - Validates port number range

### File and Path Utilities

- **`SanitizeFilename(filename string) string`** - Sanitizes filename by removing invalid characters
- **`JoinPaths(base string, paths ...string) string`** - Safely joins path components

### Environment Utilities

- **`IsProductionEnvironment() bool`** - Checks if running in production environment
- **`GetEnvVar(key, defaultValue string) string`** - Gets environment variable with default

### Error Utilities

- **`WrapError(err error, context string) error`** - Wraps error with additional context
- **`IsRetryableError(err error) bool`** - Checks if error should be retried

### Retry Utilities

- **`RetryConfig`** - Configuration struct for retry behavior
- **`DefaultRetryConfig() RetryConfig`** - Returns sensible default retry configuration
- **`CalculateRetryDelay(attempt int, config RetryConfig) time.Duration`** - Calculates exponential backoff delay

### Configuration Management

- **`ConfigManager[T any]`** - Generic interface for configuration management
- **`BaseConfigManager[T any]`** - Base implementation of configuration management

## Usage Examples

### Shell Command Escaping
```go
import "pb-deployer/internal/utils"

// Safe shell command construction
filename := "user input.txt"
cmd := fmt.Sprintf("ls -la %s", utils.ShellEscape(filename))
// Result: ls -la 'user input.txt'
```

### Duration Formatting
```go
duration := 2*time.Minute + 30*time.Second
formatted := utils.FormatDuration(duration)
// Result: "2.5m"
```

### Service Validation
```go
if err := utils.ValidateServiceName("my-app-service"); err != nil {
    return fmt.Errorf("invalid service name: %w", err)
}
```

### Slice Operations
```go
services := []string{"nginx", "apache", "mysql"}
if utils.ContainsService(services, "NGINX") { // case-insensitive
    // nginx found
}
```

### Retry Configuration
```go
retryConfig := utils.DefaultRetryConfig()
delay := utils.CalculateRetryDelay(2, retryConfig) // 2nd retry attempt
time.Sleep(delay)
```

## Migration Notes

This package was created by extracting common utilities from:

- `pb-deployer/internal/tunnel/managers/deployment.go`
- `pb-deployer/internal/tunnel/managers/security.go`
- `pb-deployer/internal/tunnel/managers/service.go`
- `pb-deployer/internal/tunnel/managers/setup.go`
- `pb-deployer/internal/handlers/deployment/management.go`
- `pb-deployer/internal/tunnel/execution.go`

### Breaking Changes

- All utility functions are now capitalized (exported)
- `shellEscape` → `ShellEscape`
- `formatDuration` → `FormatDuration`
- `boolToYesNo` → `BoolToYesNo`
- `containsString` → `ContainsString`
- `max` → `Max`

### Progress Reporting

Each manager implements its own progress reporting using the tunnel package directly.

## Design Principles

1. **No Dependencies on Internal Packages** - This package avoids importing other internal packages to prevent circular dependencies
2. **Generic Where Possible** - Uses Go generics for type-safe configuration management
3. **Consistent Naming** - All exported functions follow Go naming conventions
4. **Error Wrapping** - Uses Go 1.13+ error wrapping for better error context
5. **Safe Defaults** - All functions provide safe defaults and validate inputs

## Testing

Each utility function should be thoroughly tested with edge cases including:
- Empty strings
- Invalid inputs
- Boundary conditions
- Special characters and encoding issues
