package utils

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "'simple'"},
		{"with spaces", "'with spaces'"},
		{"with'quote", "'with'\"'\"'quote'"},
		{"", "''"},
		{"multiple'quotes'here", "'multiple'\"'\"'quotes'\"'\"'here'"},
	}

	for _, test := range tests {
		result := ShellEscape(test.input)
		if result != test.expected {
			t.Errorf("ShellEscape(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{30 * time.Second, "30.0s"},
		{90 * time.Second, "1.5m"},
		{2*time.Hour + 30*time.Minute, "2.5h"},
		{45 * time.Minute, "45.0m"},
		{0, "0.0s"},
	}

	for _, test := range tests {
		result := FormatDuration(test.input)
		if result != test.expected {
			t.Errorf("FormatDuration(%v) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestBoolToYesNo(t *testing.T) {
	if BoolToYesNo(true) != "yes" {
		t.Error("BoolToYesNo(true) should return 'yes'")
	}
	if BoolToYesNo(false) != "no" {
		t.Error("BoolToYesNo(false) should return 'no'")
	}
}

func TestBoolToEnabledDisabled(t *testing.T) {
	if BoolToEnabledDisabled(true) != "enabled" {
		t.Error("BoolToEnabledDisabled(true) should return 'enabled'")
	}
	if BoolToEnabledDisabled(false) != "disabled" {
		t.Error("BoolToEnabledDisabled(false) should return 'disabled'")
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 5},
		{3, 5, 5},
		{5, 5, 5},
		{-1, -5, -1},
		{0, 0, 0},
	}

	for _, test := range tests {
		result := Max(test.a, test.b)
		if result != test.expected {
			t.Errorf("Max(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{5, 3, 3},
		{3, 5, 3},
		{5, 5, 5},
		{-1, -5, -5},
		{0, 0, 0},
	}

	for _, test := range tests {
		result := Min(test.a, test.b)
		if result != test.expected {
			t.Errorf("Min(%d, %d) = %d, expected %d", test.a, test.b, result, test.expected)
		}
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	if !ContainsString(slice, "banana") {
		t.Error("ContainsString should find 'banana'")
	}

	if ContainsString(slice, "grape") {
		t.Error("ContainsString should not find 'grape'")
	}

	if ContainsString([]string{}, "anything") {
		t.Error("ContainsString should return false for empty slice")
	}
}

func TestContainsService(t *testing.T) {
	services := []string{"nginx", "Apache", "MySQL"}

	// Case-insensitive matching
	if !ContainsService(services, "NGINX") {
		t.Error("ContainsService should find 'NGINX' (case-insensitive)")
	}

	if !ContainsService(services, "apache") {
		t.Error("ContainsService should find 'apache' (case-insensitive)")
	}

	if ContainsService(services, "redis") {
		t.Error("ContainsService should not find 'redis'")
	}
}

func TestRemoveFromSlice(t *testing.T) {
	slice := []string{"a", "b", "c", "b", "d"}
	result := RemoveFromSlice(slice, "b")

	expected := []string{"a", "c", "b", "d"} // Only first occurrence removed
	if len(result) != len(expected) {
		t.Errorf("RemoveFromSlice length mismatch: got %d, expected %d", len(result), len(expected))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("RemoveFromSlice[%d] = %q, expected %q", i, result[i], v)
		}
	}

	// Test removing non-existent item
	unchanged := RemoveFromSlice(slice, "z")
	if len(unchanged) != len(slice) {
		t.Error("RemoveFromSlice should not change slice when item not found")
	}
}

func TestUniqueStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	result := UniqueStrings(input)

	expected := []string{"a", "b", "c", "d"}
	if len(result) != len(expected) {
		t.Errorf("UniqueStrings length mismatch: got %d, expected %d", len(result), len(expected))
	}

	// Check all expected items are present
	for _, item := range expected {
		if !ContainsString(result, item) {
			t.Errorf("UniqueStrings missing expected item: %q", item)
		}
	}
}

func TestValidateServiceName(t *testing.T) {
	validNames := []string{"nginx", "my-app", "service_01", "app.service"}
	for _, name := range validNames {
		if err := ValidateServiceName(name); err != nil {
			t.Errorf("ValidateServiceName(%q) should be valid, got error: %v", name, err)
		}
	}

	invalidNames := []string{"", "invalid@name", "name with spaces", "name/with/slashes"}
	for _, name := range invalidNames {
		if err := ValidateServiceName(name); err == nil {
			t.Errorf("ValidateServiceName(%q) should be invalid", name)
		}
	}
}

func TestValidateAppName(t *testing.T) {
	validNames := []string{"myapp", "my-app", "app_01", "MyApp123"}
	for _, name := range validNames {
		if err := ValidateAppName(name); err != nil {
			t.Errorf("ValidateAppName(%q) should be valid, got error: %v", name, err)
		}
	}

	invalidNames := []string{"", "app@invalid", "name with spaces", "toolongname" + strings.Repeat("x", 50)}
	for _, name := range invalidNames {
		if err := ValidateAppName(name); err == nil {
			t.Errorf("ValidateAppName(%q) should be invalid", name)
		}
	}
}

func TestValidatePort(t *testing.T) {
	validPorts := []int{22, 80, 443, 8080, 65535}
	for _, port := range validPorts {
		if err := ValidatePort(port); err != nil {
			t.Errorf("ValidatePort(%d) should be valid, got error: %v", port, err)
		}
	}

	invalidPorts := []int{0, -1, 65536, 100000}
	for _, port := range invalidPorts {
		if err := ValidatePort(port); err == nil {
			t.Errorf("ValidatePort(%d) should be invalid", port)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input     string
		maxLength int
		expected  string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exact", 5, "exact"},
		{"toolong", 3, "too"},
		{"", 5, ""},
	}

	for _, test := range tests {
		result := TruncateString(test.input, test.maxLength)
		if result != test.expected {
			t.Errorf("TruncateString(%q, %d) = %q, expected %q", test.input, test.maxLength, result, test.expected)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-file.txt", "normal-file.txt"},
		{"file with spaces.txt", "file with spaces.txt"},
		{"invalid<file>.txt", "invalid_file_.txt"},
		{"file/with\\slashes", "file_with_slashes"},
		{"", "unnamed"},
		{"...dotted...", "dotted"},
	}

	for _, test := range tests {
		result := SanitizeFilename(test.input)
		if result != test.expected {
			t.Errorf("SanitizeFilename(%q) = %q, expected %q", test.input, result, test.expected)
		}
	}
}

func TestIsRetryableError(t *testing.T) {
	retryableErrors := []error{
		errors.New("connection refused"),
		errors.New("operation timeout"),
		errors.New("network unreachable"),
		errors.New("Connection reset by peer"),
	}

	for _, err := range retryableErrors {
		if !IsRetryableError(err) {
			t.Errorf("IsRetryableError should return true for: %v", err)
		}
	}

	nonRetryableErrors := []error{
		errors.New("permission denied"),
		errors.New("file not found"),
		errors.New("invalid syntax"),
	}

	for _, err := range nonRetryableErrors {
		if IsRetryableError(err) {
			t.Errorf("IsRetryableError should return false for: %v", err)
		}
	}

	// Test nil error
	if IsRetryableError(nil) {
		t.Error("IsRetryableError should return false for nil error")
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxAttempts <= 0 {
		t.Error("DefaultRetryConfig should have positive MaxAttempts")
	}

	if config.BaseDelay <= 0 {
		t.Error("DefaultRetryConfig should have positive BaseDelay")
	}

	if config.MaxDelay <= config.BaseDelay {
		t.Error("DefaultRetryConfig MaxDelay should be greater than BaseDelay")
	}

	if config.Multiplier <= 1.0 {
		t.Error("DefaultRetryConfig Multiplier should be greater than 1.0")
	}
}

func TestCalculateRetryDelay(t *testing.T) {
	config := RetryConfig{
		BaseDelay:  1 * time.Second,
		MaxDelay:   10 * time.Second,
		Multiplier: 2.0,
	}

	// Test exponential backoff
	delay1 := CalculateRetryDelay(0, config)
	delay2 := CalculateRetryDelay(1, config)
	delay3 := CalculateRetryDelay(2, config)

	if delay1 != 1*time.Second {
		t.Errorf("First delay should be base delay, got %v", delay1)
	}

	if delay2 != 2*time.Second {
		t.Errorf("Second delay should be 2x base delay, got %v", delay2)
	}

	if delay3 != 4*time.Second {
		t.Errorf("Third delay should be 4x base delay, got %v", delay3)
	}

	// Test max delay cap
	largeDelay := CalculateRetryDelay(10, config)
	if largeDelay != config.MaxDelay {
		t.Errorf("Large delay should be capped at MaxDelay, got %v", largeDelay)
	}
}
