package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	Bold   = "\033[1m"
)

func main() {
	// Define command line flags
	installDeps := flag.Bool("install", false, "Install project dependencies")
	buildOnly := flag.Bool("build-only", false, "Build frontend without running the server")
	runOnly := flag.Bool("run-only", false, "Run the server without building the frontend")
	production := flag.Bool("production", false, "Create a production build in dist folder")
	testOnly := flag.Bool("test-only", false, "Run test suite and generate reports only")
	distDir := flag.String("dist", "dist", "Output directory for production build")
	help := flag.Bool("help", false, "Show help and usage information")
	flag.Parse()

	// Show help if requested
	if *help {
		showHelp()
		return
	}

	// Show banner
	operation := "DEVELOPMENT"
	if *production {
		operation = "PRODUCTION"
	} else if *testOnly {
		operation = "TESTING"
	}
	printBanner(operation)

	startTime := time.Now()

	printStep("🔍", "Checking system requirements...")
	if err := checkSystemRequirements(); err != nil {
		printError("System requirements check failed: %v", err)
		os.Exit(1)
	}
	printSuccess("System requirements satisfied!")

	// Get the root directory of the project
	rootDir, err := os.Getwd()
	if err != nil {
		printError("Failed to get current directory: %v", err)
		os.Exit(1)
	}

	printStep("📁", "Project root: %s", rootDir)

	// Handle test-only mode
	if *testOnly {
		if err := testOnlyMode(rootDir, *distDir); err != nil {
			printError("Test suite failed: %v", err)
			os.Exit(1)
		}
		printTestSummary(time.Since(startTime))
		return
	}

	// Handle production build
	if *production {
		if err := productionBuild(rootDir, *installDeps, *distDir); err != nil {
			printError("Production build failed: %v", err)
			os.Exit(1)
		}
		printBuildSummary(time.Since(startTime), true)
		return
	}

	// If not in run-only mode, build the frontend
	if !*runOnly {
		if err := buildFrontend(rootDir, *installDeps); err != nil {
			printError("Frontend build failed: %v", err)
			os.Exit(1)
		}
	}

	// Run the server unless in build-only mode
	if !*buildOnly {
		if err := runServer(rootDir); err != nil {
			printError("Server startup failed: %v", err)
			os.Exit(1)
		}
	}

	if *buildOnly {
		printBuildSummary(time.Since(startTime), false)
	}
	printSuccess("🎉 All operations completed successfully!")
}

func productionBuild(rootDir string, installDeps bool, distDir string) error {
	printHeader("🚀 PRODUCTION BUILD")

	// Create output directory
	outputDir := filepath.Join(rootDir, distDir)
	printStep("🧹", "Cleaning output directory...")

	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("failed to clean dist directory: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}
	printSuccess("Output directory prepared: %s", outputDir)

	// Build frontend
	if err := buildFrontendProduction(rootDir, installDeps); err != nil {
		return fmt.Errorf("frontend production build failed: %w", err)
	}

	// Copy to dist
	if err := copyFrontendToDist(rootDir, outputDir); err != nil {
		return fmt.Errorf("failed to copy frontend to dist: %w", err)
	}

	// Run test suite and generate report
	if err := runTestSuiteAndGenerateReport(rootDir, outputDir); err != nil {
		return fmt.Errorf("test suite failed: %w", err)
	}

	// Build server binary
	if err := buildServerBinary(rootDir, outputDir); err != nil {
		return fmt.Errorf("failed to build server binary: %w", err)
	}

	printSuccess("✅ Production build completed! Files are in '%s'", distDir)
	return nil
}

func buildFrontend(rootDir string, installDeps bool) error {
	printHeader("🔨 FRONTEND BUILD")

	frontendDir := filepath.Join(rootDir, "frontend")

	if err := validateFrontendSetup(frontendDir); err != nil {
		return err
	}

	if installDeps {
		if err := installDependencies(rootDir, frontendDir); err != nil {
			return err
		}
	}

	if err := buildFrontendCore(frontendDir); err != nil {
		return err
	}

	return copyFrontendToPbPublic(rootDir, frontendDir)
}

func buildFrontendProduction(rootDir string, installDeps bool) error {
	printStep("🏗️", "Building frontend for production...")
	return buildFrontend(rootDir, installDeps)
}

func validateFrontendSetup(frontendDir string) error {
	printStep("🔍", "Validating frontend setup...")

	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		return fmt.Errorf("frontend directory not found at %s", frontendDir)
	}

	packageJSON := filepath.Join(frontendDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json not found at %s", packageJSON)
	}

	printSuccess("Frontend setup validated")
	return nil
}

func installDependencies(rootDir, frontendDir string) error {
	printStep("📦", "Installing dependencies...")

	// Install Go dependencies first
	printStep("🏗️", "Installing Go dependencies...")

	// Run go mod tidy first
	printStep("🧹", "Tidying Go modules...")
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}
	printSuccess("Go modules tidied in %s", time.Since(start).Round(time.Millisecond))

	// Run go mod download
	printStep("⬇️", "Downloading Go modules...")
	cmd = exec.Command("go", "mod", "download")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start = time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go mod download failed: %w", err)
	}
	printSuccess("Go modules downloaded in %s", time.Since(start).Round(time.Millisecond))

	// Install frontend dependencies
	printStep("📦", "Installing frontend dependencies...")
	cmd = exec.Command("npm", "install")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start = time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	printSuccess("Frontend dependencies installed in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func buildFrontendCore(frontendDir string) error {
	printStep("⚙️", "Building frontend...")

	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm run build failed: %w", err)
	}

	printSuccess("Frontend built in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func copyFrontendToPbPublic(rootDir, frontendDir string) error {
	printStep("📂", "Copying frontend build to pb_public...")

	pbPublicDir := filepath.Join(rootDir, "pb_public")

	if err := os.RemoveAll(pbPublicDir); err != nil {
		return fmt.Errorf("failed to clean pb_public: %w", err)
	}

	if err := os.MkdirAll(pbPublicDir, 0755); err != nil {
		return fmt.Errorf("failed to create pb_public: %w", err)
	}

	buildDir := findBuildDirectory(frontendDir)

	start := time.Now()
	cmd := exec.Command("cp", "-r", buildDir+"/.", pbPublicDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy build files: %w", err)
	}

	printSuccess("Frontend copied to pb_public in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func copyFrontendToDist(rootDir, outputDir string) error {
	printStep("📁", "Copying frontend to dist...")

	pbPublicDir := filepath.Join(outputDir, "pb_public")
	if err := os.MkdirAll(pbPublicDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist pb_public: %w", err)
	}

	frontendDir := filepath.Join(rootDir, "frontend")
	buildDir := findBuildDirectory(frontendDir)

	start := time.Now()
	cmd := exec.Command("cp", "-r", buildDir+"/.", pbPublicDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy to dist: %w", err)
	}

	printSuccess("Frontend copied to dist in %s", time.Since(start).Round(time.Millisecond))
	return nil
}

func runTestSuiteAndGenerateReport(rootDir, outputDir string) error {
	printStep("🧪", "Running test suite...")

	// Create test reports directory
	reportsDir := filepath.Join(outputDir, "test-reports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create test reports directory: %w", err)
	}

	start := time.Now()

	// Run the test suite with JSON output for detailed report
	reportFile := filepath.Join(reportsDir, "test-report.json")
	cmd := exec.Command("go", "test", "-json", "./...")
	cmd.Dir = rootDir

	// Capture output for both display and file
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Save the failed test output anyway
		if writeErr := os.WriteFile(reportFile, output, 0644); writeErr != nil {
			printWarning("Failed to write test report: %v", writeErr)
		}
		return fmt.Errorf("test suite failed: %w", err)
	}

	// Write JSON report
	if err := os.WriteFile(reportFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write test report: %w", err)
	}

	// Run our custom test suite for summary
	summaryFile := filepath.Join(reportsDir, "test-summary.txt")
	cmd = exec.Command("go", "run", "./cmd/tests")
	cmd.Dir = rootDir

	summaryOutput, err := cmd.CombinedOutput()
	if err != nil {
		// Save the failed summary anyway
		if writeErr := os.WriteFile(summaryFile, summaryOutput, 0644); writeErr != nil {
			printWarning("Failed to write test summary: %v", writeErr)
		}
		return fmt.Errorf("test suite summary failed: %w", err)
	}

	// Write summary report
	if err := os.WriteFile(summaryFile, summaryOutput, 0644); err != nil {
		return fmt.Errorf("failed to write test summary: %w", err)
	}

	// Generate coverage report
	coverageFile := filepath.Join(reportsDir, "coverage.out")
	htmlCoverageFile := filepath.Join(reportsDir, "coverage.html")

	cmd = exec.Command("go", "test", "-coverprofile="+coverageFile, "./...")
	cmd.Dir = rootDir
	if err := cmd.Run(); err != nil {
		printWarning("Failed to generate coverage report: %v", err)
	} else {
		// Generate HTML coverage report
		cmd = exec.Command("go", "tool", "cover", "-html="+coverageFile, "-o", htmlCoverageFile)
		cmd.Dir = rootDir
		if err := cmd.Run(); err != nil {
			printWarning("Failed to generate HTML coverage report: %v", err)
		}
	}

	duration := time.Since(start)
	printSuccess("Test suite completed in %s", duration.Round(time.Millisecond))
	printSuccess("Test reports generated in '%s'", filepath.Join(outputDir, "test-reports"))

	return nil
}

func testOnlyMode(rootDir, distDir string) error {
	printHeader("🧪 TEST SUITE EXECUTION")

	// Create output directory for test reports
	outputDir := filepath.Join(rootDir, distDir)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Run test suite and generate report
	if err := runTestSuiteAndGenerateReport(rootDir, outputDir); err != nil {
		return fmt.Errorf("test suite failed: %w", err)
	}

	printSuccess("✅ Test suite completed! Reports are in '%s/test-reports'", distDir)
	return nil
}

func buildServerBinary(rootDir, outputDir string) error {
	printStep("🏗️", "Building server binary...")

	binaryName := "pb-deployer"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	outputPath := filepath.Join(outputDir, binaryName)

	start := time.Now()
	cmd := exec.Command("go", "build", "-o", outputPath, filepath.Join(rootDir, "cmd/server/main.go"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	// Get binary size
	if stat, err := os.Stat(outputPath); err == nil {
		sizeMB := float64(stat.Size()) / 1024 / 1024
		printSuccess("Server binary built in %s (%.2f MB)", time.Since(start).Round(time.Millisecond), sizeMB)
	} else {
		printSuccess("Server binary built in %s", time.Since(start).Round(time.Millisecond))
	}

	return nil
}

func runServer(rootDir string) error {
	printHeader("🚀 STARTING SERVER")

	cmd := exec.Command("go", "run", filepath.Join(rootDir, "cmd/server/main.go"), "serve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	printStep("🌐", "Server starting...")
	return cmd.Run()
}

func findBuildDirectory(frontendDir string) string {
	possibleDirs := []string{"build", "dist", "static"}

	for _, dir := range possibleDirs {
		buildDir := filepath.Join(frontendDir, dir)
		if _, err := os.Stat(buildDir); err == nil {
			return buildDir
		}
	}

	log.Fatalf("Could not find frontend build directory in: %v", possibleDirs)
	return ""
}

func checkSystemRequirements() error {
	requirements := []struct {
		name    string
		command string
		args    []string
	}{
		{"Go", "go", []string{"version"}},
		{"Node.js", "node", []string{"--version"}},
		{"npm", "npm", []string{"--version"}},
		{"Git", "git", []string{"--version"}},
	}

	missing := make([]string, 0)

	for _, req := range requirements {
		if checkCommand(req.command, req.args...) {
			printInfo("%s available", req.name)
		} else {
			printWarning("%s missing", req.name)
			missing = append(missing, req.name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %v", missing)
	}

	return nil
}

func checkCommand(command string, args ...string) bool {
	cmd := exec.Command(command, args...)
	return cmd.Run() == nil
}

// Visual output functions
func printBanner(operation string) {
	fmt.Printf("\n%s▲ pb-deployer%s %sv1.0.0%s\n", Bold, Reset, Gray, Reset)
	fmt.Printf("%s%s%s\n\n", Gray, strings.ToLower(operation), Reset)
}

func printHeader(title string) {
	fmt.Printf("\n%s%s%s\n", Bold, title, Reset)
}

func printStep(emoji, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s %s\n", emoji, message)
}

func printSuccess(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s✓%s %s\n", Green, Reset, message)
}

func printError(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s✗ Error:%s %s\n", Red, Reset, message)
}

func printWarning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠ Warning:%s %s\n", Yellow, Reset, message)
}

func printInfo(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%sℹ%s %s\n", Cyan, Reset, message)
}

func printBuildSummary(duration time.Duration, isProduction bool) {
	buildType := "Development"
	if isProduction {
		buildType = "Production"
	}

	fmt.Printf("\n%sBuild Complete%s\n", Bold, Reset)
	fmt.Printf("%s%s%s\n", Gray, strings.Repeat("─", 14), Reset)

	fmt.Printf("\n%sType:%s     %s%s%s\n", Gray, Reset, Green, buildType, Reset)
	fmt.Printf("%sDuration:%s %s%s%s\n", Gray, Reset, Cyan, duration.Round(time.Millisecond), Reset)
	fmt.Printf("%sTarget:%s   %s%s/%s%s\n", Gray, Reset, Purple, runtime.GOOS, runtime.GOARCH, Reset)

	if isProduction {
		fmt.Printf("\n%sOutput:%s\n", Gray, Reset)
		fmt.Printf("  %spb-deployer%s binary\n", Green, Reset)
		fmt.Printf("  %spb_public/%s directory\n", Green, Reset)
		fmt.Printf("  %stest-reports/%s directory\n", Green, Reset)
		fmt.Printf("  %sdist/%s location\n", Cyan, Reset)
	}

	fmt.Printf("\n")
}

func printTestSummary(duration time.Duration) {
	fmt.Printf("\n%sTest Suite Complete%s\n", Bold, Reset)
	fmt.Printf("%s%s%s\n", Gray, strings.Repeat("─", 19), Reset)

	fmt.Printf("\n%sType:%s     %sTesting%s\n", Gray, Reset, Green, Reset)
	fmt.Printf("%sDuration:%s %s%s%s\n", Gray, Reset, Cyan, duration.Round(time.Millisecond), Reset)
	fmt.Printf("%sTarget:%s   %s%s/%s%s\n", Gray, Reset, Purple, runtime.GOOS, runtime.GOARCH, Reset)

	fmt.Printf("\n%sOutput:%s\n", Gray, Reset)
	fmt.Printf("  %stest-summary.txt%s report\n", Green, Reset)
	fmt.Printf("  %stest-report.json%s detailed data\n", Green, Reset)
	fmt.Printf("  %scoverage.html%s interactive report\n", Green, Reset)
	fmt.Printf("  %scoverage.out%s coverage data\n", Green, Reset)

	fmt.Printf("\n")
}

func showHelp() {
	fmt.Printf("\n%s▲ pb-deployer%s %sv1.0.0%s\n", Bold, Reset, Gray, Reset)
	fmt.Printf("%sModern deployment automation tool%s\n\n", Gray, Reset)

	fmt.Printf("%sUSAGE:%s\n", Bold, Reset)
	fmt.Printf("  go run ./cmd/scripts [options]\n\n")

	fmt.Printf("%sOPTIONS:%s\n", Bold, Reset)
	fmt.Printf("  %s--help%s          Show this help message\n", Green, Reset)
	fmt.Printf("  %s--install%s       Install all project dependencies (Go + npm)\n", Green, Reset)
	fmt.Printf("  %s--production%s    Create production build with all assets\n", Green, Reset)
	fmt.Printf("  %s--test-only%s     Run test suite and generate reports only\n", Green, Reset)
	fmt.Printf("  %s--build-only%s    Build frontend without starting server\n", Green, Reset)
	fmt.Printf("  %s--run-only%s      Start server without building frontend\n", Green, Reset)
	fmt.Printf("  %s--dist%s DIR      Output directory for production build (default: dist)\n", Green, Reset)

	fmt.Printf("\n%sEXAMPLES:%s\n", Bold, Reset)
	fmt.Printf("  %s# Development - build frontend and start server%s\n", Gray, Reset)
	fmt.Printf("  go run ./cmd/scripts\n\n")

	fmt.Printf("  %s# Install dependencies first%s\n", Gray, Reset)
	fmt.Printf("  go run ./cmd/scripts --install\n\n")

	fmt.Printf("  %s# Production build with tests%s\n", Gray, Reset)
	fmt.Printf("  go run ./cmd/scripts --production\n\n")

	fmt.Printf("  %s# Run test suite only%s\n", Gray, Reset)
	fmt.Printf("  go run ./cmd/scripts --test-only\n\n")

	fmt.Printf("  %s# Custom dist directory%s\n", Gray, Reset)
	fmt.Printf("  go run ./cmd/scripts --production --dist=release\n\n")

	fmt.Printf("%sPRODUCTION OUTPUT:%s\n", Bold, Reset)
	fmt.Printf("  %spb-deployer%s       - Server binary\n", Green, Reset)
	fmt.Printf("  %spb_public/%s        - Frontend assets\n", Green, Reset)
	fmt.Printf("  %stest-reports/%s     - Test results and coverage\n", Green, Reset)
	fmt.Printf("    %s├── test-summary.txt%s   - Human-readable summary\n", Cyan, Reset)
	fmt.Printf("    %s├── test-report.json%s   - Detailed JSON results\n", Cyan, Reset)
	fmt.Printf("    %s├── coverage.html%s      - Interactive coverage report\n", Cyan, Reset)
	fmt.Printf("    %s└── coverage.out%s       - Coverage profile data\n", Cyan, Reset)

	fmt.Printf("\n%sREQUIREMENTS:%s\n", Bold, Reset)
	fmt.Printf("  %s• Go%s 1.21+ for backend development\n", Green, Reset)
	fmt.Printf("  %s• Node.js%s 18+ for frontend build process\n", Green, Reset)
	fmt.Printf("  %s• npm%s for frontend dependency management\n", Green, Reset)
	fmt.Printf("  %s• Git%s for version control operations\n", Green, Reset)

	fmt.Printf("\n")
}
