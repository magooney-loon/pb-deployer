# pb-deployer — Agent Reference

PocketBase deployment automation tool. Go backend + SvelteKit frontend. Automates SSH-based provisioning, multi-app Caddy reverse proxy, and zero-downtime deployments.

## Project Structure

```
├── cmd/
│   ├── scripts/main.go          # CLI entry (build/run/test/production modes)
│   └── server/main.go           # PocketBase server entry point
├── internal/
│   ├── api/                     # HTTP handlers (versioned v1 API)
│   ├── jobs/                    # Cron health checks
│   ├── logger/                  # Colored console logger
│   ├── models/                  # PocketBase collection schemas + Go structs
│   └── tunnel/                  # SSH automation library
├── frontend/                    # SvelteKit 5 (runes) SPA
│   └── src/
│       ├── lib/
│       │   ├── api/             # PocketBase JS client wrappers
│       │   ├── stores/          # Svelte 5 runes singleton stores (.svelte.ts)
│       │   ├── components/      # UI components + modals + partials
│       │   └── utils/           # Theme, navigation, view transitions
│       └── routes/              # SvelteKit pages
├── migrations/                  # PocketBase DB migrations
└── pb_data/                     # PocketBase runtime data
```

## Backend Architecture

### Tunnel Package (`internal/tunnel/`)

SSH automation library for remote server management.

| File | Purpose |
|------|---------|
| `types.go` | Core interfaces: `SSHClient`, `Config`, `Result`, `Closer`, `CleanupManager`, option patterns (`ExecOption`, `UserOption`, `FileOption`) |
| `auth.go` | SSH agent authentication, host key verification, known_hosts cleanup |
| `client.go` | SSH connection management, command execution, SFTP upload/download, sudo |
| `manager.go` | System operations: create users, SSH keys, directories, systemd services, packages |
| `setup_manager.go` | Server setup: Caddy install, base Caddyfile, app cleanup on delete |
| `security_manager.go` | Firewall (ufw/firewalld/iptables), SSH hardening, fail2ban, Cloudflare rules |
| `deployment_manager.go` | 12-step deployment pipeline with rollback |

#### Key Interfaces

```go
type SSHClient interface {
    Connect() error
    Close() error
    IsConnected() bool
    Execute(cmd string, opts ...ExecOption) (*Result, error)
    ExecuteSudo(cmd string, opts ...ExecOption) (*Result, error)
    Upload(localPath, remotePath string, opts ...FileOption) error
    Download(remotePath, localPath string, opts ...FileOption) error
    Ping() error
    HostInfo() (string, error)
    SetTracer(tracer Tracer)
}
```

#### Resource Cleanup Pattern

All components support `Close()` via the `Closer` interface. Use `CleanupManager` for complex scenarios:

```go
cleanup := tunnel.NewCleanupManager()
defer cleanup.Close()
cleanup.AddCloser(client)
cleanup.AddCloser(mgr)
cleanup.Add(func() { /* custom cleanup */ })
```

### Models (`internal/models/`)

PocketBase data models with auto-created collections, cascade deletes, and optimized indexes.

#### Server (`server.go`)

```go
type Server struct {
    ID, Name, Host           string
    Port                     int       // default 22
    RootUsername             string    // default "root"
    AppUsername              string    // default "pocketbase"
    UseSSHAgent              bool      // default true
    ManualKeyPath            string
    SetupComplete            bool
    SecurityLocked           bool
    ProxyEmail               string    // ACME email for Caddy
}
```

Methods: `GetSSHAddress()`, `IsReadyForDeployment()`, `IsFullySecured()`, `IsSetupComplete()`, `IsSecurityLocked()`

#### App (`app.go`)

```go
type App struct {
    ID, Name, ServerID       string
    RemotePath, ServiceName  string
    Domain                   string
    HTTPPort                 int       // 8090-8999, per-server unique
    CurrentVersion, Status   string    // online|offline|unknown
}
```

Methods: `GetHealthURL()`, `IsOnline()`

#### Deployment (`deployment.go`)

Status: `pending` → `running` → `success`|`failed`. Methods: `MarkAsRunning()`, `MarkAsSuccess()`, `MarkAsFailed()`, `IsComplete()`, `GetDuration()`, `AppendLog()`

#### Version (`version.go`)

Links to App, supports ZIP deployment packages.

#### Relations & Cascade

```
Server (deleted) → Apps (cascade) → Versions & Deployments (cascade)
App (deleted) → Versions & Deployments (cascade)
Version (deleted) → Deployments (cascade)
```

#### Directory Structure on Target Servers

```
/opt/pocketbase/
├── apps/           # Per-app deployment directories
├── backups/        # Timestamped deployment backups
├── logs/           # Application + Caddy access logs
└── staging/        # Temporary staging during deployments
```

### API (`internal/api/`)

Versioned v1 API registered via `handlers.go`. Routes:

| Route | Handler | File |
|-------|---------|------|
| `POST /api/setup/server` | `handleServerSetup` | `setup.go` |
| `POST /api/setup/security` | `handleServerSecurity` | `setup.go` |
| `POST /api/setup/validate` | `handleServerValidation` | `setup.go` |
| `POST /api/deploy` | `handleDeploy` | `deploy.go` |
| `POST /api/apps` | `handleCreateApp` | `apps.go` |
| `GET /api/terminal` | `handleTerminal` | `terminal.go` |

Key behaviors:
- App creation goes through `/api/apps` (not direct PB write) for `http_port` auto-allocation
- `apps.CreateRule = nil` enforces this
- App delete hook cleans up remote systemd service + Caddy fragment
- WebSocket terminal for SSH passthrough

### Jobs (`internal/jobs/`)

`health.go` — Cron job checking app health every minute. Updates `status` field.

### Deployment Pipeline (12 Steps)

1. Downloading and staging deployment package
2. Checking service status
3. Stopping existing service
4. Creating backup of current deployment
5. Preparing deployment directory
6. Installing new version
7. Creating/updating systemd service (`--http=127.0.0.1:<port>`)
8. **Configuring reverse proxy** (writes Caddy fragment, validates, reloads)
9. Creating superuser (if initial deployment)
10. Starting service
11. Verifying deployment health (loopback + public HTTPS probes)
12. Finalizing deployment

Rollback restores backup + Caddy fragment on failure.

### Caddy Integration

- Installed during server setup via `InstallCaddy()` (apt/yum/dnf fan-out)
- Base Caddyfile written once by `WriteBaseCaddyfile(email)` with `import /etc/caddy/conf.d/*.caddy`
- Per-app fragments at `/etc/caddy/conf.d/<app>.caddy`
- Only Caddy binds `:80` and `:443`; PocketBase binds `127.0.0.1:<port>`
- No `setcap` on PocketBase binaries
- Firewall: 22/80/443 only

## Frontend Architecture

### Svelte 5 Runes Stores (`frontend/src/lib/stores/`)

All stores are `.svelte.ts` files (required for rune compilation). Module-scope singletons with getter-based reactive reads.

| Store | File | Purpose |
|-------|------|---------|
| `api` | `client.svelte.ts` | Single `ApiClient` instance + PocketBase realtime subscriptions |
| `appsStore` | `apps.svelte.ts` | Apps + derived projections (`byServer()`, `byId()`), CRUD |
| `serversStore` | `servers.svelte.ts` | Servers + setup/security progress |
| `deploymentsStore` | `deployments.svelte.ts` | Deployments + per-app live log streams |
| `versionsStore` | `versions.svelte.ts` | Versions keyed by `app_id` |
| `ui` | `ui.svelte.ts` | Modal/toast bus — replaces prop-drilled `onrefresh`/`onclose` |
| `settings` | `settings.svelte.ts` | User prefs (lockscreen, theme), localStorage-backed |
| `splash` | `splash.svelte.ts` | Splash screen state |

Barrel export: `import { appsStore, ui, api } from '$lib/stores'`

#### Store Conventions

- **`.svelte.ts`** for any module using `$state`/`$derived`/`$effect`
- **Getters expose `$state`** — prevents external reassignment breaking reactivity
- **One `ApiClient`** — all stores import from `client.svelte.ts`
- **Optimistic updates** — apply locally → await network → realtime echo confirms → rollback on error
- **`_onRealtime`** is the only way external code writes to stores
- **Form state stays local** in `.svelte` components — only submission results mutate stores
- **No `onMount(() => store.load())` in every page** — hoist to `+layout.svelte` or `+page.ts`

#### Realtime

Started once in `+layout.svelte` via `startRealtime()`. Subscribes to `*` on apps, servers, deployments, versions collections.

### API Client (`frontend/src/lib/api/`)

Pure HTTP/PB wrappers (no UI state). `ApiClient` constructs PocketBase instance + sub-clients for apps, servers, versions, deployments.

### Components

| Directory | Contents |
|-----------|----------|
| `components/main/` | Page components: `AppList.svelte`, `ServerList.svelte`, `Dashboard.svelte`, `DeploymentsList.svelte`, `Settings.svelte`, `Navigation.svelte`, `SplashScreen.svelte` |
| `components/modals/` | `AppCreateModal`, `ServerCreateModal`, `ManageAppModal`, `TroubleshootModal`, `TerminalModal`, `DeploymentModal`, `DeleteModal`, `LogsModal` |
| `components/partials/` | Reusable UI: `Button`, `Card`, `FormField`, `StatusBadge`, `Toast`, `DataTable`, `Accordion`, `ProgressBar`, `MetricCard`, `EmptyState`, `FileUpload`, `LoadingSpinner`, `WarningBanner` |

### Routes

| Route | Page |
|-------|------|
| `/` | Dashboard |
| `/apps` | App list |
| `/servers` | Server list |
| `/deployments` | Deployment history |
| `/settings` | Settings + lockscreen |
| `/docs` | Built-in documentation |

## Key Patterns

### App Status Flow

```
new app → "offline" → deploy → "online"
health check failure → "unknown" or "offline"
```

### Port Allocation

Range `8090–8999`. Auto-assigned smallest free port per server on app creation. Enforced unique per server via `idx_apps_port_per_server`.

### Security

- `CreateRule = nil` on apps collection forces API-layer creation
- Per-server unique indexes prevent name/domain/port/service collisions
- Cascade deletes clean up related records
- SSH agent auth with host key verification
- Firewall: ufw/firewalld/iptables with 22/80/443 rules
- SSH hardening + fail2ban during security lockdown

## Input Validation

App creation (`/api/apps`) validates `name` and `domain` before they flow into
remote shell commands (systemd unit, remote path, Caddy fragment):

- `name` must match `^[a-z0-9][a-z0-9-]{0,62}$` (DNS-label style)
- `domain` must be a valid hostname

The superuser email/password are passed to `superuser create` base64-encoded and
decoded inside the remote shell, so credentials with spaces or shell
metacharacters cannot break or inject into the command.

## Known Issues

_None currently tracked._

Resolved:
- `idx_apps_port_per_server` is now added in both `CreateCollection` and `syncAppsCollection` (`app.go`), so the per-server unique port constraint exists on freshly-created and synced databases alike.
- Base Caddyfile heredoc terminator typo (`CaddyEOF` vs `CADDYEOF`) in `setup_manager.go` fixed — the stray terminator no longer leaks into `/etc/caddy/Caddyfile`.

## Quick Commands

```bash
# Install and run
go run cmd/scripts/main.go --install

# Build
go run cmd/scripts/main.go --build

# Test mode
go run cmd/scripts/main.go --test

# Production mode
go run cmd/scripts/main.go --production
```
