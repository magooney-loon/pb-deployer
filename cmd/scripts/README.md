## ⚡ Quick Reference

| Command | Description |
|---------|-------------|
| `go run cmd/scripts/main.go` | 🔄 Build + Run |
| `go run cmd/scripts/main.go --install` | 📦 Install + Build + Run |
| `go run cmd/scripts/main.go --build-only` | 🔨 Build Only |
| `go run cmd/scripts/main.go --run-only` | ▶️ Run Only |
| `go run cmd/scripts/main.go --production` | 🚀 Production Build |
| `go run cmd/scripts/main.go --production --dist <dir>` | 📁 Custom Dist |

---

### Development Mode
```bash
go run cmd/scripts/main.go
```
Builds the frontend and runs the server in development mode.

### Fresh Install & Run
```bash
go run cmd/scripts/main.go --install
```
Installs dependencies, builds the frontend, and runs the server.

### Build Only
```bash
go run cmd/scripts/main.go --build-only
```
Only builds the frontend without running the server.

### Server Only
```bash
go run cmd/scripts/main.go --run-only
```
Only runs the server (assumes frontend is already built).

## 🚢 Production Builds

### Default Production Build
```bash
go run cmd/scripts/main.go --production
```
Creates a production build in the `dist` folder.

### Custom Output Directory
```bash
go run cmd/scripts/main.go --production --dist customfolder
```
Creates a production build in the specified `customfolder` directory.
