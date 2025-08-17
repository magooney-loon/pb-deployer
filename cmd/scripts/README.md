## 📋 Command Reference

| Command | Description | Example Output |
|---------|-------------|----------------|
| `go run cmd/scripts/main.go` | 🔄 **Development Build** | Builds frontend + starts server |
| `go run cmd/scripts/main.go --install` | 📦 **Install + Build** | Downloads deps + builds + runs |
| `go run cmd/scripts/main.go --build-only` | 🔨 **Build Only** | Just builds, doesn't run server |
| `go run cmd/scripts/main.go --run-only` | ▶️ **Run Only** | Skips build, just runs server |
| `go run cmd/scripts/main.go --production` | 🚀 **Production Build** | Creates optimized dist package |
| `go run cmd/scripts/main.go --test-only` | 🧪 **Test Suite** | Runs tests and generates reports |
| `go run cmd/scripts/main.go --production --dist <dir>` | 📁 **Custom Output** | Production build to custom dir |
| `go run cmd/scripts/main.go --help` | ❓ **Show Help** | Displays all available flags and options |

```yaml
# pb-deployer Package Metadata
# Generated automatically during production build

package:
  name: pb-deployer
  version: v1.0.0
  description: Modern deployment automation tool

build:
  timestamp: %s
  target: %s
  builder: production-script

environment:
  go_version: %s
  node_version: %s
  npm_version: %s

git:
  commit: %s
  branch: %s
  tag: %s

contents:
  binary: pb-deployer%s
  frontend: pb_public/
  tests: test-reports/

test_results:
  total_packages: 2
  status: passed
  coverage_available: true

dependencies:
  go_modules: true
  npm_packages: true

notes:
  - All tests passed during build
  - Coverage reports included
  - Production optimized build
  - Frontend statically compiled
```
