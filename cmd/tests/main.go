package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	// Parse command line flags
	verbose := flag.Bool("v", false, "enable verbose output")
	flag.Parse()

	// Set verbose mode
	SetVerbose(*verbose)

	// Print header
	PrintHeader()

	// Check if go test command is available
	if err := CheckGoTestAvailable(); err != nil {
		log.Fatalf("❌ Prerequisites check failed: %v", err)
	}

	// Get list of test packages to run
	packages := GetTestPackages()

	if len(packages) == 0 {
		fmt.Println("⚠️  No test packages found to run")
		os.Exit(0)
	}

	fmt.Printf("📦 Found %d test package(s) to run\n\n", len(packages))

	// Run the full test suite
	suite := RunTestSuite(packages)

	// Print comprehensive summary
	PrintSummary(suite)

	// Exit with appropriate code
	ExitWithCode(suite)
}
