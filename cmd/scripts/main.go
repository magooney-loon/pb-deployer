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
	installDeps := flag.Bool("install", false, "Install frontend dependencies")
	buildOnly := flag.Bool("build-only", false, "Build frontend without running the server")
	runOnly := flag.Bool("run-only", false, "Run the server without building the frontend")
	production := flag.Bool("production", false, "Create a production build in dist folder")
	distDir := flag.String("dist", "dist", "Output directory for production build")
	flag.Parse()

	// Show banner
	operation := "DEVELOPMENT"
	if *production {
		operation = "PRODUCTION"
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
		if err := installDependencies(frontendDir); err != nil {
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

func installDependencies(frontendDir string) error {
	printStep("📦", "Installing frontend dependencies...")

	cmd := exec.Command("npm", "install")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed: %w", err)
	}

	printSuccess("Dependencies installed in %s", time.Since(start).Round(time.Millisecond))
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
			printInfo("  ✅ %s available", req.name)
		} else {
			printWarning("  ❌ %s missing", req.name)
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
	banner := `
 ██████╗ ██████╗       ██████╗ ███████╗██████╗ ██╗      ██████╗ ██╗   ██╗███████╗██████╗
 ██╔══██╗██╔══██╗      ██╔══██╗██╔════╝██╔══██╗██║     ██╔═══██╗╚██╗ ██╔╝██╔════╝██╔══██╗
 ██████╔╝██████╔╝█████╗██║  ██║█████╗  ██████╔╝██║     ██║   ██║ ╚████╔╝ █████╗  ██████╔╝
 ██╔═══╝ ██╔══██╗╚════╝██║  ██║██╔══╝  ██╔═══╝ ██║     ██║   ██║  ╚██╔╝  ██╔══╝  ██╔══██╗
 ██║     ██████╔╝      ██████╔╝███████╗██║     ███████╗╚██████╔╝   ██║   ███████╗██║  ██║
 ╚═╝     ╚═════╝       ╚═════╝ ╚══════╝╚═╝     ╚══════╝ ╚═════╝    ╚═╝   ╚══════╝╚═╝  ╚═╝`

	fmt.Printf("%s%s%s\n", Cyan, banner, Reset)
	fmt.Printf("%s%s                                🚀 %s BUILD SYSTEM 🚀%s\n", Bold, Yellow, operation, Reset)
	fmt.Printf("%s                                      v1.0.0%s\n\n", Gray, Reset)
}

func printHeader(title string) {
	fmt.Printf("\n%s%s%s %s %s\n", Bold, Blue, strings.Repeat("=", 20), title, strings.Repeat("=", 20))
	fmt.Printf("%s", Reset)
}

func printStep(emoji, format string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s %s %s%s\n", Gray, timestamp, Reset, emoji, message, Reset)
}

func printSuccess(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s✅ %s%s\n", Green, message, Reset)
}

func printError(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s❌ ERROR: %s%s\n", Red, message, Reset)
}

func printWarning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s⚠️  %s%s\n", Yellow, message, Reset)
}

func printInfo(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s%s%s\n", Cyan, message, Reset)
}

func printBuildSummary(duration time.Duration, isProduction bool) {
	fmt.Printf("\n%s%s", Bold, strings.Repeat("=", 60))
	fmt.Printf("\n🎯 BUILD SUMMARY")
	fmt.Printf("\n%s%s\n", strings.Repeat("=", 60), Reset)

	buildType := "Development"
	if isProduction {
		buildType = "Production"
	}

	fmt.Printf("%s📋 Build Information:%s\n", Bold, Reset)
	fmt.Printf("   • Type: %s%s%s\n", Green, buildType, Reset)
	fmt.Printf("   • Duration: %s%s%s\n", Cyan, duration.Round(time.Millisecond), Reset)
	fmt.Printf("   • Go Version: %s%s%s\n", Blue, runtime.Version(), Reset)
	fmt.Printf("   • Target: %s%s/%s%s\n", Purple, runtime.GOOS, runtime.GOARCH, Reset)

	if isProduction {
		fmt.Printf("\n%s📦 Production Artifacts:%s\n", Bold, Reset)
		fmt.Printf("   • Binary: %spb-deployer%s\n", Green, Reset)
		fmt.Printf("   • Frontend: %spb_public/%s\n", Green, Reset)
		fmt.Printf("   • Location: %sdist/%s\n", Cyan, Reset)
	} else {
		fmt.Printf("\n%s🔧 Development:%s\n", Bold, Reset)
		fmt.Printf("   • Frontend: %spb_public/%s\n", Green, Reset)
		fmt.Printf("   • Server: %sReady to start%s\n", Yellow, Reset)
	}

	fmt.Printf("\n%s⚡ Performance:%s\n", Bold, Reset)
	fmt.Printf("   • Build Rate: %s%.2f ops/sec%s\n", Cyan, 1.0/duration.Seconds(), Reset)
	fmt.Printf("   • Timestamp: %s%s%s\n", Gray, time.Now().Format("2006-01-02 15:04:05"), Reset)

	fmt.Printf("%s%s%s\n", Bold, strings.Repeat("=", 60), Reset)
}
