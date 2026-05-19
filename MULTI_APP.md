# Multiple PocketBase Apps on One VPS — Design Spec

This document specifies how pb-deployer should be changed to **natively support running many PocketBase apps on the same VPS**, with **Caddy** bundled and managed by the deployer as a reverse proxy that fronts every app.

It is a design/spec, not a changelog. Each section names the file to touch, the shape of the change, and the constraint it satisfies.

> Goal: a user adds App A (`a.example.com`) and App B (`b.example.com`) to the same `Server` row, clicks deploy on each in turn, and both serve TLS-terminated traffic on the public 443 without any manual `systemd`/nginx/cert work. Removing an app cleans up its proxy config.

---

## 1. Target architecture

```
                                Internet
                                   │
                              :80 / :443
                                   │
                       ┌───────────▼───────────┐
                       │        Caddy           │  (systemd: caddy.service)
                       │  - automatic HTTPS     │
                       │  - imports conf.d/*    │
                       └────┬───────┬───────┬───┘
                            │       │       │
                  127.0.0.1:8091  :8092  :8093   (loopback only)
                            │       │       │
                       ┌────▼──┐ ┌──▼──┐ ┌──▼──┐
                       │ blog  │ │shop │ │ api │   (per-app systemd units,
                       │ pb    │ │ pb  │ │ pb  │    run as `pocketbase` user,
                       └───────┘ └─────┘ └─────┘    no privileged ports)
```

Key invariants:

- **Only Caddy binds `:80` and `:443`.** PocketBase processes bind a unique `127.0.0.1:<port>` each.
- **One Caddy config fragment per app**, at `/etc/caddy/conf.d/<app>.caddy`, written/removed by the deployer.
- **The main `/etc/caddy/Caddyfile`** is written *once* during server setup and contains a single `import /etc/caddy/conf.d/*.caddy` directive — pb-deployer never touches it again.
- **No `cap_net_bind_service`** on PocketBase binaries. The deploy step that runs `setcap` is removed.
- **No firewall rules per app.** The firewall keeps 22/80/443 open and that's it.

---

## 2. Data model changes

### 2.1 `App` (`internal/models/app.go`)

Add one field, change two indexes.

| Field         | Type   | Required | Notes                                                                 |
| ------------- | ------ | :------: | --------------------------------------------------------------------- |
| `http_port`   | int    | yes      | Loopback port the PocketBase process binds. Range `8090–8999`. Per-server unique. |

Auto-assignment rule (handled in the API layer, see §6): when creating an app, if `http_port` is empty, pick the smallest free integer in `[8090, 8999]` not yet used by another app on the **same** `server_id`. Reject if the range is exhausted.

Index changes:

| Index                       | Before                       | After                                       |
| --------------------------- | ---------------------------- | ------------------------------------------- |
| `idx_apps_name`             | **unique** `(name)`          | **non-unique** `(name)` — keep for lookups  |
| `idx_apps_name_per_server`  | —                            | **unique** `(server_id, name)`              |
| `idx_apps_domain_per_server`| —                            | **unique** `(server_id, domain)`            |
| `idx_apps_port_per_server`  | —                            | **unique** `(server_id, http_port)`         |
| `idx_apps_service_per_server`| —                           | **unique** `(server_id, service_name)`      |

Result: app names, domains, ports, and service names are unique *within* a server but reusable across servers. This makes copies of an app on a staging server possible without renaming.

### 2.2 `Server` (`internal/models/server.go`)

Add one field.

| Field            | Type | Required | Default | Notes                                              |
| ---------------- | ---- | :------: | ------- | -------------------------------------------------- |
| `proxy_email`    | str  | no       | `""`    | ACME registration email used in the global Caddyfile. Empty → Caddy uses its built-in default flow. |

No state field for "caddy installed" — this is derived from `setup_complete` (Caddy is installed during the server-setup step, see §3).

### 2.3 Migration for existing rows

Apps created before this change have:
- no `http_port`
- a systemd unit that binds 80/443 directly

On first start after upgrade:
1. Run a one-shot DB migration that fills `http_port` for any app missing one (smallest free value per server).
2. Mark those apps with `status = "needs_migration"` (new enum value, see §2.4).
3. The UI and `/api/deploy` refuse to deploy a `needs_migration` app until the user triggers a one-click "migrate to proxy" action (see §7), which rewrites the systemd unit and adds a Caddy fragment in a single transaction.

### 2.4 Status enum

Extend `apps.status` from `{online, offline, unknown}` to `{online, offline, unknown, needs_migration}`.

---

## 3. Server setup — install and configure Caddy

### 3.1 New step in `SetupManager.SetupPocketBaseServer` (`internal/tunnel/setup_manager.go`)

After `InstallEssentials`, add:

```go
err = s.InstallCaddy()
if err != nil { return fmt.Errorf("failed to install caddy: %w", err) }

err = s.WriteBaseCaddyfile(proxyEmail)
if err != nil { return fmt.Errorf("failed to write base Caddyfile: %w", err) }
```

### 3.2 `InstallCaddy()`

Mirror the existing package-manager fan-out used by `UpdateSystem()`:

- Debian/Ubuntu: add Caddy's official apt repo, `apt install -y caddy`.
- RHEL/CentOS: `yum install -y caddy` from the official COPR / RPM.
- Fedora: `dnf install -y caddy`.

Verify `which caddy` and `systemctl is-enabled caddy` return success. Add `caddy` to the `VerifySetup` essentials list.

### 3.3 `WriteBaseCaddyfile(email string)`

Idempotent — only writes if absent or content drifts. Writes:

```
{
    email {{ .Email }}              # omitted if email == ""
    admin off                       # we don't expose the admin API
}

import /etc/caddy/conf.d/*.caddy
```

Then:

- `mkdir -p /etc/caddy/conf.d` (owned `root:root`, mode `0755`).
- `systemctl enable --now caddy`.

### 3.4 New directory layout assertions in `VerifySetup`

Add `/etc/caddy`, `/etc/caddy/conf.d`, and `/etc/caddy/Caddyfile` to the list of required paths.

### 3.5 Firewall

`GetDefaultPocketBaseRules` (`internal/tunnel/security_manager.go:341`) is unchanged — Caddy already needs 22/80/443 open. Drop the `setcap` step (§4.4) and you're done.

---

## 4. Deployment pipeline changes

All changes in `internal/tunnel/deployment_manager.go`.

### 4.1 Add `HTTPPort` to `DeploymentRequest`

```go
type DeploymentRequest struct {
    // ... existing fields ...
    HTTPPort int       // 127.0.0.1:<HTTPPort> — required, non-zero
}
```

`internal/api/deploy.go` reads `appRecord.GetInt("http_port")` and passes it through.

### 4.2 Step list

Replace the 11-step pipeline with 12 steps. The new step **"Configure reverse proxy"** runs *after* the systemd unit is created and *before* the service is started, so Caddy is ready to forward traffic the moment PocketBase opens its loopback port.

```
1.  Downloading and staging deployment package
2.  Checking service status
3.  Stopping existing service
4.  Creating backup of current deployment
5.  Preparing deployment directory
6.  Installing new version
7.  Creating/updating systemd service
8.  Configuring reverse proxy            (NEW)
9.  Creating superuser (if initial deployment)
10. Starting service
11. Verifying deployment health
12. Finalizing deployment
```

### 4.3 New `createSystemdService` ExecStart

```go
ExecStart=%s serve %s --http=127.0.0.1:%d
```

(arguments: `BinaryPath`, `Domain`, `HTTPPort`)

The unit no longer needs ambient capabilities or root fallback. Delete the `useRootFallback` branch and the related logging.

### 4.4 Remove `setcap` step

Lines 396–419 in the current `swapDeployment` (the `setcap 'cap_net_bind_service=+ep'` block and the `libcap2-bin` fallback) come out entirely. `swapDeployment` ends after `chmod +x` and ownership fix-ups.

### 4.5 New `configureReverseProxy(ctx, deployCtx)` step

Writes `/etc/caddy/conf.d/<app>.caddy`:

```
{{ .Domain }} {
    encode zstd gzip
    reverse_proxy 127.0.0.1:{{ .HTTPPort }} {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
    }
}
```

Then:

```bash
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
```

Failure modes:
- `caddy validate` fails → mark deployment as failed, leave the fragment file in place under `<app>.caddy.broken` so the user can inspect; restore previous fragment from backup (see §4.7).
- `systemctl reload caddy` non-zero exit → fall back to `systemctl restart caddy` once. If that also fails, mark deployment failed and roll back.

### 4.6 New `verifyDeployment` probes

The probe list becomes:

```go
healthUrls := []struct{ url, description string }{
    {fmt.Sprintf("http://127.0.0.1:%d/api/health", req.HTTPPort), "loopback PocketBase"},
    {fmt.Sprintf("https://%s/api/health",          req.Domain),   "public HTTPS via Caddy"},
}
```

Loopback probe confirms PocketBase started. Public HTTPS probe (with `-k` retained for first-issuance edge cases) confirms Caddy is forwarding. Both must pass within the retry budget (still 15 × 2s).

### 4.7 Per-app proxy backup for rollback

Before writing a new `<app>.caddy`, copy the existing one (if any) to `/opt/pocketbase/backups/<app>-<ts>/caddy.fragment.bak`. `rollback()` restores it, then `caddy reload`. If no prior fragment existed, rollback removes the fragment outright.

### 4.8 App deletion hook

Today, deleting an `App` row only cascades to versions/deployments — nothing touches the server. Add a delete hook in `internal/api/handlers.go` (or wherever the `apps` collection is bound) that, when an app is removed:

1. SSHes in (best-effort — log a warning if the server is unreachable).
2. `systemctl stop <service> && systemctl disable <service>`.
3. `rm /etc/systemd/system/<service>.service`.
4. `rm /etc/caddy/conf.d/<app>.caddy`.
5. `systemctl daemon-reload && systemctl reload caddy`.
6. Leaves `/opt/pocketbase/apps/<app>/` and backups on disk — destructive data deletion stays manual.

---

## 5. Security manager changes

`internal/tunnel/security_manager.go`

- `GetDefaultPocketBaseRules` — **no change** (22/80/443 still correct).
- `GetCloudflareFirewallRules` — **no change** (still locks origin 80/443 to CF ranges; Caddy still listens, just behind CF).
- `HardenSSH` / `SetupFail2ban` — **no change**.

Add a small helper `EnsureCaddyAllowed()` that on hardening runs, just to be safe, asserts that 80/443 rules remain present even if the user supplied a custom rule list.

---

## 6. API changes

### 6.1 App-create handler (new — currently apps are created via the PB collection API directly)

To enforce `http_port` auto-allocation and prevent duplicates, route app creation through `v1Router.POST("/api/apps")` and:

1. Validate `name`, `domain`, `service_name` against the per-server uniqueness indexes.
2. If `http_port == 0`, pick the lowest free port in `[8090, 8999]` for that `server_id`.
3. Save the record.

Direct writes to the `apps` collection should be **disabled** (set `CreateRule = nil`) to force the API path. Note: per the current `app.go:67`, `CreateRule = types.Pointer("")` (anyone can write); flip it.

### 6.2 `/api/deploy` handler (`internal/api/deploy.go`)

- Reject if `appRecord.GetInt("http_port") == 0` with a clear error pointing the user at the migration action (§7).
- Populate `tunnel.DeploymentRequest.HTTPPort`.

### 6.3 New `/api/apps/{id}/migrate-proxy` handler

For apps in `status = "needs_migration"`:

1. Assigns an `http_port` if missing.
2. Connects via SSH, regenerates the systemd unit with `--http=127.0.0.1:<port>`, writes the Caddy fragment, reloads systemd + Caddy.
3. Optionally `setcap -r <binary>` to drop the privileged-port capability.
4. Sets `status = "online"` (or `offline` if the service can't restart).

This is the **only** non-deploy path that mutates server state. Document it loudly.

### 6.4 `/api/setup/server` handler (`internal/api/setup.go`)

Accept new optional field `proxy_email` and persist it on the `Server` record (used by `WriteBaseCaddyfile`, §3.3).

---

## 7. Migration path for existing installations

```
Existing app (pre-Caddy):
  systemd: ExecStart=/.../app serve example.com           (binds :80 + :443)
  setcap:  cap_net_bind_service=+ep on the binary
  caddy:   not installed

Step 1: Upgrade pb-deployer binary.
Step 2: Run setup again on each server  →  installs Caddy, writes base Caddyfile.
        (Setup is idempotent; existing apps keep running on :80/:443 because Caddy
         hasn't been started yet — wait: Caddy will fail to bind. See workaround.)
```

**Workaround for the bind-conflict during migration**:

The Caddy package install enables but does not necessarily start the service on Debian/Ubuntu (`systemctl enable` only). `WriteBaseCaddyfile` should **not** start Caddy if it detects any existing app on this server with `status != "needs_migration"` *and* `http_port == 0`. Instead, leave Caddy enabled-but-stopped. The first per-app `migrate-proxy` call:

1. Stops the legacy app (frees 80/443).
2. Removes `setcap` capability from its binary.
3. Rewrites the systemd unit to use `--http=127.0.0.1:<port>`.
4. Writes the Caddy fragment for it.
5. Starts the app (now binds loopback only).
6. Starts Caddy if not running, else `caddy reload`.

From there, each subsequent legacy app on the same server can migrate one at a time — Caddy is already up, the user just transitions each app off privileged ports through the same handler. The window where the migrating app is unreachable is `stop → systemd-rewrite → caddy-reload → start`, typically <5 s.

---

## 8. Edge cases the design must handle

| Case                                                         | Behavior                                                                                  |
| ------------------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| Two apps deployed concurrently to the same server            | Each writes its own `<app>.caddy` fragment; final `caddy reload` is safe to run twice.    |
| `caddy validate` fails after writing a new fragment          | Delete just-written fragment (or restore backup), no reload; deployment fails fast.       |
| User manually edits `/etc/caddy/Caddyfile`                   | Untouched by pb-deployer after initial creation; user edits persist.                      |
| User manually edits a fragment                               | Next deploy overwrites it. Document this. Provide an `extra_caddy_directives` field on `App` later if needed. |
| Port `8090` is already used by something unrelated           | `http_port` allocator probes `ss -lnt 'sport = :<port>'` once before assigning; skips occupied ports. |
| ACME challenges fail (e.g. DNS not yet pointed)              | Caddy retries internally; loopback health passes, public health fails → deployment is reported "service started, public unreachable, check DNS"; status set to `unknown` rather than failed. |
| App row deleted while service is still serving               | Delete hook (§4.8) stops + cleans up. If SSH is down, mark with `pending_cleanup` flag and retry on next successful connection to that server. |
| Server in `setup_complete = false` state                     | Deploy still rejected as today.                                                           |

---

## 9. File-by-file change list

| File                                          | Change                                                                                              |
| --------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `internal/models/app.go`                      | Add `http_port`; reshape indexes per §2.1; extend status enum.                                      |
| `internal/models/server.go`                   | Add `proxy_email`.                                                                                  |
| `internal/models/migrations.go` (new)         | One-shot fill of `http_port` for pre-existing rows; status → `needs_migration`.                     |
| `internal/tunnel/setup_manager.go`            | Add `InstallCaddy`, `WriteBaseCaddyfile`; extend `VerifySetup` with caddy checks; extend `SetupInfo`. |
| `internal/tunnel/deployment_manager.go`       | New `configureReverseProxy` step; drop `setcap` block; update systemd template; update health probes; per-app fragment backup/restore in rollback. |
| `internal/tunnel/security_manager.go`         | Add `EnsureCaddyAllowed` helper; no firewall rule changes.                                          |
| `internal/tunnel/types.go`                    | Add `HTTPPort` to `DeploymentRequest`; add `CaddyInstalled bool` to `SetupInfo`.                    |
| `internal/api/handlers.go`                    | Register `/api/apps` (create), `/api/apps/{id}/migrate-proxy`, app-delete hook.                     |
| `internal/api/setup.go`                       | Accept `proxy_email`; pass through.                                                                 |
| `internal/api/deploy.go`                      | Read `http_port`; reject if zero with migration hint; pass through.                                 |
| `internal/api/apps.go` (new)                  | App-create handler with port auto-allocation; uniqueness validation.                                |
| `frontend/...`                                | Surface `http_port` (read-only after creation), `proxy_email`, the migrate-proxy action, and "needs migration" badge. (Not detailed here.) |
| `README.md`                                   | One paragraph + link to this doc.                                                                   |

---

## 10. Caddyfile templates (canonical)

**`/etc/caddy/Caddyfile`** (written once by setup, owned `root:root`, mode `0644`):

```caddy
{
    {{ if .Email }}email {{ .Email }}{{ end }}
    admin off
}

import /etc/caddy/conf.d/*.caddy
```

**`/etc/caddy/conf.d/<app>.caddy`** (written by every deploy, owned `root:root`, mode `0644`):

```caddy
{{ .Domain }} {
    encode zstd gzip
    reverse_proxy 127.0.0.1:{{ .HTTPPort }} {
        header_up Host {host}
        header_up X-Real-IP {remote_host}
        header_up X-Forwarded-For {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    log {
        output file /opt/pocketbase/logs/{{ .AppName }}.access.log {
            roll_size 10mb
            roll_keep 5
        }
    }
}
```

Notes:
- `encode` is safe to leave on for PocketBase responses (it skips already-compressed payloads).
- `X-Forwarded-Proto` lets PocketBase generate correct `https://` URLs in emails / OAuth callbacks.
- Access logs land in `/opt/pocketbase/logs/` so they're rotated next to the existing service logs.

---

## 11. Out of scope (intentionally)

- **Multi-server load balancing.** One app still lives on one server in this design.
- **Per-app Linux user.** All apps continue to run as `Server.AppUsername`. Worth doing later for stricter isolation, but it's a separable change.
- **HTTP-only apps.** Caddy auto-issues TLS; if a user explicitly wants HTTP-only, they can edit the fragment template — not a first-class flag yet.
- **Wildcard / DNS-01 challenges.** TLS-ALPN-01 / HTTP-01 (Caddy default) covers the common case. DNS-01 requires per-provider creds and is a follow-up.
- **Caddy version pinning.** Use whatever the distro repo ships. A pinned/static binary is a future hardening step.

---

## 12. Quick reference — current code anchors this spec replaces

| Current behavior                            | File:line                                       | Replaced by                                          |
| ------------------------------------------- | ----------------------------------------------- | ---------------------------------------------------- |
| `ExecStart=<bin> serve <domain>`            | `internal/tunnel/deployment_manager.go:455`     | `ExecStart=<bin> serve <domain> --http=127.0.0.1:<port>` (§4.3) |
| `setcap cap_net_bind_service=+ep`           | `internal/tunnel/deployment_manager.go:410`     | Removed (§4.4)                                       |
| Health probes on `:80`/`:443`/`:8080`       | `internal/tunnel/deployment_manager.go:560-569` | Loopback + public HTTPS only (§4.6)                  |
| `idx_apps_name` (global unique)             | `internal/models/app.go:115`                    | Non-unique + `(server_id, name)` unique (§2.1)       |
| No `http_port` on `App`                     | `internal/models/app.go:10`                     | Added, with per-server unique index (§2.1)           |
| `setup_manager.go` essentials list          | `internal/tunnel/setup_manager.go:142`          | Adds `caddy` (§3.4)                                  |
| `CreateRule = types.Pointer("")` on apps    | `internal/models/app.go:67`                     | Tighten to force `/api/apps` creation flow (§6.1)    |

---

When this lands, the README's "one app per VPS" assumption is gone — same workflow, but you can stack as many apps as your VPS has RAM for, and Caddy handles certs and routing transparently.
