package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GetTestPackages returns a list of test packages to run
func GetTestPackages() []string {
	return []string{
		"./internal/tunnel",
		// Add more packages here as they are created
	}
}

// Global flags
var (
	Verbose bool = false
)

// TestResult represents the result of running a test package
type TestResult struct {
	Package  string
	Passed   int
	Failed   int
	Skipped  int
	Duration time.Duration
	Success  bool
	Output   []string
	Errors   []string
}

// TestSuite represents the overall test suite results
type TestSuite struct {
	Results     []TestResult
	TotalPassed int
	TotalFailed int
	TotalTests  int
	Duration    time.Duration
	Success     bool
}

// RunTestPackage executes tests for a specific package and returns the result
func RunTestPackage(packagePath string) TestResult {
	result := TestResult{
		Package: packagePath,
		Output:  []string{},
		Errors:  []string{},
	}

	start := time.Now()

	if Verbose {
		fmt.Printf("  └─ Executing: go test -v %s\n", packagePath)
	}

	// Run go test command
	cmd := exec.Command("go", "test", "-v", packagePath)

	// Get command output
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)

	if Verbose {
		fmt.Printf("  └─ Command completed in %v\n", result.Duration.Round(time.Millisecond))
	}

	if err != nil {
		result.Success = false
		errorMsg := fmt.Sprintf("Command failed: %v", err)
		result.Errors = append(result.Errors, errorMsg)
		if Verbose {
			fmt.Printf("  └─ Error: %s\n", errorMsg)
		}
	} else {
		result.Success = true
	}

	// Parse test output
	parseTestOutput(string(output), &result)

	return result
}

// parseTestOutput parses the go test output to extract test statistics
func parseTestOutput(output string, result *TestResult) {
	lines := strings.Split(output, "\n")

	// Regex patterns for parsing test output
	testPassRegex := regexp.MustCompile(`^\s*--- PASS: (\w+)`)
	testFailRegex := regexp.MustCompile(`^\s*--- FAIL: (\w+)`)
	testSkipRegex := regexp.MustCompile(`^\s*--- SKIP: (\w+)`)

	for _, line := range lines {
		result.Output = append(result.Output, line)

		if testPassRegex.MatchString(line) {
			result.Passed++
		} else if testFailRegex.MatchString(line) {
			result.Failed++
			result.Errors = append(result.Errors, line)
		} else if testSkipRegex.MatchString(line) {
			result.Skipped++
		} else if strings.Contains(line, "FAIL") && strings.Contains(line, "exit status") {
			result.Success = false
		}
	}

	// If we have failures, mark as unsuccessful
	if result.Failed > 0 {
		result.Success = false
	}
}

// PrintTestResult prints a formatted result for a single test package
func PrintTestResult(result TestResult) {
	status := "✅ PASS"
	if !result.Success {
		status = "❌ FAIL"
	}

	fmt.Printf("%-20s %s  (%d passed, %d failed, %d skipped) [%v]\n",
		result.Package, status, result.Passed, result.Failed, result.Skipped, result.Duration.Round(time.Millisecond))

	// Print errors if any
	if len(result.Errors) > 0 && result.Failed > 0 {
		for _, err := range result.Errors {
			if strings.Contains(err, "FAIL:") {
				fmt.Printf("  └─ %s\n", strings.TrimSpace(err))
			}
		}
	}

	// Show full output in verbose mode
	if Verbose && len(result.Output) > 0 {
		fmt.Printf("  └─ Full output:\n")
		for _, line := range result.Output {
			if strings.TrimSpace(line) != "" {
				fmt.Printf("     %s\n", line)
			}
		}
	}
}

// PrintSummary prints the overall test suite summary
func PrintSummary(suite TestSuite) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("                    TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))

	overallStatus := "✅ ALL TESTS PASSED"
	if !suite.Success {
		overallStatus = "❌ SOME TESTS FAILED"
	}

	fmt.Printf("Overall Status: %s\n", overallStatus)
	fmt.Printf("Total Tests:    %d\n", suite.TotalTests)
	fmt.Printf("Passed:         %d\n", suite.TotalPassed)
	fmt.Printf("Failed:         %d\n", suite.TotalFailed)
	fmt.Printf("Duration:       %v\n", suite.Duration.Round(time.Millisecond))
	fmt.Printf("Packages:       %d\n", len(suite.Results))

	if !suite.Success {
		fmt.Println("\nFailed Packages:")
		for _, result := range suite.Results {
			if !result.Success {
				fmt.Printf("  - %s (%d failures)\n", result.Package, result.Failed)
			}
		}
	}

	fmt.Println(strings.Repeat("=", 60))
}

// RunTestSuite runs all specified test packages and returns the overall results
func RunTestSuite(packages []string) TestSuite {
	suite := TestSuite{
		Results: make([]TestResult, 0, len(packages)),
		Success: true,
	}

	start := time.Now()

	fmt.Println("🚀 Starting Test Suite Execution")
	fmt.Println(strings.Repeat("-", 60))

	for i, pkg := range packages {
		fmt.Printf("[%d/%d] Running tests for %s...\n", i+1, len(packages), pkg)

		result := RunTestPackage(pkg)
		suite.Results = append(suite.Results, result)

		// Update totals
		suite.TotalPassed += result.Passed
		suite.TotalFailed += result.Failed
		suite.TotalTests += result.Passed + result.Failed + result.Skipped

		if !result.Success {
			suite.Success = false
		}

		PrintTestResult(result)

		if Verbose && i < len(packages)-1 {
			fmt.Println()
		}
	}

	suite.Duration = time.Since(start)
	return suite
}

// ExitWithCode exits the program with appropriate exit code based on test results
func ExitWithCode(suite TestSuite) {
	if suite.Success {
		fmt.Println("✅ All tests completed successfully!")
		os.Exit(0)
	} else {
		fmt.Println("❌ Test suite failed!")
		os.Exit(1)
	}
}

// CheckGoTestAvailable verifies that go test command is available
func CheckGoTestAvailable() error {
	if Verbose {
		fmt.Println("🔍 Checking if go command is available...")
	}

	cmd := exec.Command("go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go command not available: %v", err)
	}

	if Verbose {
		fmt.Printf("  └─ Go version: %s", string(output))
	}

	return nil
}

// PrintHeader prints a nice header for the test suite
func PrintHeader() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("               PB-DEPLOYER TEST SUITE")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Started at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	if Verbose {
		fmt.Printf("Verbose mode: enabled\n")
	}
	fmt.Println()
}

// SetVerbose sets the verbose flag for detailed logging
func SetVerbose(verbose bool) {
	Verbose = verbose
}
