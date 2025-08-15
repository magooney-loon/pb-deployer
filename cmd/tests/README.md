# 🧪 Test Suite Orchestrator

A comprehensive test runner that executes all tests in the pb-deployer project with detailed reporting and compact summaries.

## 🚀 Quick Start

### Basic Test Run
```bash
go run ./cmd/tests
```

### Verbose Test Run
```bash
go run ./cmd/tests -v
```

### Build and Run
```bash
go build -o test-suite ./cmd/tests
./test-suite
```

## 📋 Command Reference

| Command | Description | Example Output |
|---------|-------------|----------------|
| `go run ./cmd/tests` | 🧪 **Run All Tests** | Executes all test packages with summary |
| `go run ./cmd/tests -v` | 🔍 **Verbose Mode** | Detailed output with full test logs |
| `go build -o test-suite ./cmd/tests` | 🔨 **Build Binary** | Creates standalone test executable |
| `./test-suite` | ▶️ **Run Binary** | Execute pre-built test suite |
| `./test-suite -v` | 🔍 **Verbose Binary** | Run binary with detailed output |

## 🔧 Adding New Test Packages

To add new test packages to the suite, edit `cmd/tests/utils.go`:

```go
// GetTestPackages returns a list of test packages to run
func GetTestPackages() []string {
    return []string{
        "./internal/utils",
        "./internal/config",      // Add your new packages here
        "./internal/deployment",  //
        "./internal/docker",      //
        "./pkg/services",         //
    }
}
```

### Common Error: "undefined" functions
```
❌ Error: cmd/tests/main.go:16:2: undefined: SetVerbose
```

**Solution:** You're running a single file instead of the package. Use:
```bash
# ✅ Correct - runs entire package
go run ./cmd/tests

# ❌ Wrong - runs single file only
go run cmd/tests/main.go
```

### Tests Not Found
```
❌ Error: package ./internal/nonexistent: build constraints exclude all Go files
```

**Solution:** Check that the package path exists and contains test files (`*_test.go`).

## 🎯 Exit Codes

- **0**: All tests passed successfully
- **1**: One or more tests failed

Perfect for CI/CD integration:
```bash
go run ./cmd/tests && echo "Deploy to production!" || echo "Fix tests first!"
```

## 📁 Project Structure

```
cmd/tests/
├── main.go     # Main orchestrator entry point
├── utils.go    # Test execution utilities and formatting
└── README.md   # This file
```
