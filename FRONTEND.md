# Frontend Refactor — Runes Stores + Multi-App / Caddy UI

This spec covers two intertwined changes:

1. **Move the frontend to Svelte 5 runes-based singleton stores.** The current pattern (one `<Name>Logic` class per page, each constructing its own `ApiClient` and pushing updates through a callback into a `$state<...>` rune in the matching `.svelte`) doesn't share state across components and forces every page to re-fetch from scratch.
2. **Surface the multi-app + bundled-Caddy features** specified in `MULTI_APP.md`: per-app `http_port`, `proxy_email` on the server, migration UI for legacy apps, Caddy install/health surfaces, simplified app-create flow.

The store refactor comes first because the new UI relies on shared, reactive state (deployment status updates need to fan out into multiple views, migration actions need optimistic updates, etc.). Doing the UI work on top of the old `Logic`-class pattern would mean duplicating fetch logic across at least four components.

> Doc-only spec — no code changes here. Reads as a checklist for whoever picks the work up.

---

## 0. Current frontend (one-pass summary)

```
frontend/src/
├── app.css, app.html, app.d.ts
├── lib/
│   ├── api/                        # Resource CRUDs + ApiClient (PocketBase wrapper)
│   │   ├── client.ts               # `new ApiClient()` constructs one PB + 6 sub-clients
│   │   ├── apps/{crud,types}.ts
│   │   ├── servers/{crud,setup,types}.ts
│   │   ├── version/{crud,types}.ts
│   │   ├── deployment/{crud,deploy,types}.ts
│   │   └── utils.ts                # formatTimestamp, status helpers
│   ├── components/
│   │   ├── main/                   # AppList, ServerList, Dashboard, Navigation, …
│   │   │   ├── *.svelte            # presentational shell
│   │   │   └── *.ts                # `<Name>Logic` class, owns state + ApiClient
│   │   ├── modals/                 # AppCreateModal, ManageAppModal, …
│   │   └── partials/               # Button, FormField, StatusBadge, …
│   └── utils/                      # theme, navigation, view-transitions, Mouse
├── routes/
│   ├── +layout.svelte / +page.svelte / apps / servers / deployments / docs / settings
└── service-worker.js
```

State today:

- **Per-page `Logic` classes** in `lib/components/main/<Name>.ts` (`AppListLogic`, `ServerListLogic`, `DashboardLogic`, `DeploymentsListLogic`, `SettingsLogic`). Each holds an `ApiClient` and a private `state` object, exposing `getState()`/`onStateUpdate(cb)`.
- The matching `.svelte` does `let state = $state<…>(logic.getState()); logic.onStateUpdate(s => state = s);`.
- **Two legacy Svelte stores** (`splashScreenState`, `lockscreenState`) use the old `subscribe()` API in `+layout.svelte`. These predate runes.
- **No PocketBase realtime subscriptions are wired up** anywhere, even though `pb.collection(...).subscribe(...)` is available.
- Each `Logic` class instantiates its own `new ApiClient()`. Apps page and Dashboard both refetch the same `apps` collection independently.

Pain points this causes:

| Symptom                                                          | Root cause                                              |
| ---------------------------------------------------------------- | ------------------------------------------------------- |
| Deploy succeeds → Dashboard still shows old version until reload | No shared `apps` state; Dashboard never re-fetches.     |
| Two API calls for the same data on navigation                    | Each component owns its own `ApiClient` + cache.        |
| Modal closes but parent doesn't refresh unless told to           | `onrefresh` callback prop-drilled through every modal.  |
| Adding a new top-level concept (Caddy/proxy) needs new fields, callbacks, props, state shapes in 3–4 files | Logic-class boilerplate doesn't compose. |

---

## 1. Target state architecture: runes-based singleton stores

### 1.1 File layout

```
lib/
├── api/                  # unchanged — still pure HTTP/PB clients (no UI state)
└── stores/               # NEW — all `.svelte.ts` files
    ├── client.svelte.ts          # single ApiClient instance + PB subscriptions
    ├── apps.svelte.ts            # apps + derived projections
    ├── servers.svelte.ts         # servers + setup/security progress
    ├── deployments.svelte.ts     # deployments (incl. live status)
    ├── versions.svelte.ts        # versions (per-app caches)
    ├── ui.svelte.ts              # modals open/closed, toasts, lockscreen, splash
    ├── settings.svelte.ts        # user preferences (lockscreen, theme), localStorage-backed
    └── index.ts                  # barrel export
```

Components import singleton stores directly: `import { appsStore } from '$lib/stores'`.

### 1.2 Store shape (canonical pattern)

```ts
// lib/stores/apps.svelte.ts
import { api } from './client.svelte.js';
import type { App, AppRequest } from '$lib/api';

function createAppsStore() {
    let apps = $state<App[]>([]);
    let loading = $state(false);
    let error = $state<string | null>(null);
    let initialized = $state(false);

    // Derived projections — consumers read them as `appsStore.byServer(id)` etc.
    function byServer(serverId: string) {
        return apps.filter(a => a.server_id === serverId);
    }

    function byId(id: string) {
        return apps.find(a => a.id === id) ?? null;
    }

    async function load() {
        loading = true;
        error = null;
        try {
            const { apps: fresh } = await api.apps.getAppsWithLatestVersions();
            apps = fresh;
            initialized = true;
        } catch (e) {
            error = e instanceof Error ? e.message : String(e);
        } finally {
            loading = false;
        }
    }

    async function create(data: AppRequest, initialZip?: File) { /* … */ }
    async function remove(id: string) { /* optimistic + rollback on error */ }
    async function migrateProxy(id: string) { /* POST /api/apps/{id}/migrate-proxy */ }

    // Realtime hook — called once from client.svelte.ts on startup
    function _onRealtime(action: 'create' | 'update' | 'delete', record: App) {
        if (action === 'delete') {
            apps = apps.filter(a => a.id !== record.id);
        } else if (action === 'create') {
            apps = [record, ...apps];
        } else {
            apps = apps.map(a => a.id === record.id ? { ...a, ...record } : a);
        }
    }

    return {
        get apps() { return apps; },
        get loading() { return loading; },
        get error() { return error; },
        get initialized() { return initialized; },
        byServer, byId,
        load, create, remove, migrateProxy,
        _onRealtime,
    };
}

export const appsStore = createAppsStore();
```

Rules:

- **Always `.svelte.ts`**, never `.ts`, so the rune compiler runs.
- **Module-scope singleton** (`export const appsStore = createAppsStore()`). One instance per browser tab.
- **Getters expose `$state`** so consumers get reactive reads without leaking the writable binding.
- **`_onRealtime` is the only way external code writes** to the store. UI calls only `load`, `create`, `remove`, etc. — never mutates arrays directly.
- **Optimistic updates**: every `create`/`remove`/`update` mutates local state first, then awaits the network call, then either confirms (via realtime echo) or rolls back on error.
- **No `ApiClient` constructed inside stores** — they all import the single instance from `client.svelte.ts` (§1.4).

### 1.3 `ui.svelte.ts` — modal/toast bus

Replaces every component's local `showXModal` booleans.

```ts
// lib/stores/ui.svelte.ts
type ModalName =
    | 'app-create' | 'app-manage' | 'app-delete'
    | 'server-create' | 'server-delete' | 'server-troubleshoot' | 'server-terminal'
    | 'deployment-create' | 'deployment-view' | 'logs-view'
    | 'proxy-migrate';

function createUIStore() {
    let modal = $state<{ name: ModalName; payload?: unknown } | null>(null);
    let toasts = $state<Array<{ id: string; type: 'success'|'error'|'warning'|'info'; message: string }>>([]);

    function open(name: ModalName, payload?: unknown) { modal = { name, payload }; }
    function close() { modal = null; }
    function toast(type: typeof toasts[number]['type'], message: string, ttl = 5000) {
        const id = crypto.randomUUID();
        toasts = [...toasts, { id, type, message }];
        setTimeout(() => { toasts = toasts.filter(t => t.id !== id); }, ttl);
    }
    function dismiss(id: string) { toasts = toasts.filter(t => t.id !== id); }

    return {
        get modal() { return modal; },
        get toasts() { return toasts; },
        open, close, toast, dismiss,
    };
}
export const ui = createUIStore();
```

Use site:

```svelte
<Button onclick={() => ui.open('app-create')}>Add App</Button>
```

Modals mount once in `+layout.svelte` and render based on `ui.modal?.name`. Eliminates the prop-drilled `open`/`onclose`/`onrefresh` boilerplate in every page.

### 1.4 `client.svelte.ts` — shared PocketBase + realtime bridge

```ts
// lib/stores/client.svelte.ts
import { ApiClient } from '$lib/api';
import { appsStore } from './apps.svelte.js';
import { serversStore } from './servers.svelte.js';
import { deploymentsStore } from './deployments.svelte.js';
import { versionsStore } from './versions.svelte.js';

export const api = new ApiClient();
const pb = api.getPocketBase();

let realtimeStarted = false;
export async function startRealtime() {
    if (realtimeStarted) return;
    realtimeStarted = true;

    await pb.collection('apps').subscribe('*',        e => appsStore._onRealtime(e.action, e.record as any));
    await pb.collection('servers').subscribe('*',     e => serversStore._onRealtime(e.action, e.record as any));
    await pb.collection('deployments').subscribe('*', e => deploymentsStore._onRealtime(e.action, e.record as any));
    await pb.collection('versions').subscribe('*',    e => versionsStore._onRealtime(e.action, e.record as any));
}

export async function stopRealtime() {
    if (!realtimeStarted) return;
    realtimeStarted = false;
    await pb.collection('apps').unsubscribe();
    await pb.collection('servers').unsubscribe();
    await pb.collection('deployments').unsubscribe();
    await pb.collection('versions').unsubscribe();
}
```

Called once in `+layout.svelte`:

```svelte
<script>
    import { startRealtime, stopRealtime } from '$lib/stores/client.svelte.js';
    import { onMount, onDestroy } from 'svelte';
    onMount(() => { startRealtime(); });
    onDestroy(() => { stopRealtime(); });
</script>
```

Effects:

- A running deployment posting log lines updates `deploymentsStore` in real time → `DeploymentsList`, `ManageAppModal`, `Dashboard`, and `LogsModal` all show new lines without polling.
- Newly created apps via another tab show up immediately.

### 1.5 Migration from the old `<Name>Logic` pattern

For each existing `lib/components/main/<Name>.ts`:

1. Identify which slices of state are global (e.g. `apps`, `servers`, `deployments`) → move to the matching store.
2. Identify which slices are per-component view state (e.g. `showCreateForm`, form input bindings, sort/filter selection) → keep in the `.svelte` as plain `$state` locals.
3. Identify which methods are pure helpers (`getDeployedVersion(app)`, `hasUpdateAvailable(...)`) → move to `lib/utils/derive.ts` (or keep them where they live in `partials/`).
4. Delete the `<Name>.ts` file and the `onStateUpdate` plumbing in `<Name>.svelte`.

### 1.6 Legacy stores (`SplashScreen`, `Settings`/`Lockscreen`)

Both use the Svelte-3 `subscribe()` API today (`$splashScreenState`, `lockscreenState.subscribe(...)`). Convert to runes stores in the same `lib/stores/` directory:

- `splash.svelte.ts` exposes `splash.isLoading`, `splash.start()`, `splash.stop()`.
- `settings.svelte.ts` exposes `settings.lockscreen.{ isLocked, isEnabled }`, `settings.theme`, etc., and persists to `localStorage` via a `$effect.root` block.

Update `+layout.svelte` to `import { splash, settings } from '$lib/stores'` and drop the legacy imports.

---

## 2. Type updates for multi-app + Caddy

These mirror `MULTI_APP.md` §2.

### 2.1 `lib/api/apps/types.ts`

```diff
 export interface App {
     id: string;
     created: string;
     updated: string;
     name: string;
     server_id: string;
     remote_path: string;
     service_name: string;
     domain: string;
+    http_port: number;          // 8090–8999, assigned at create time
     current_version: string;
-    status: string;
+    status: 'online' | 'offline' | 'unknown' | 'needs_migration';
     latest_version?: string | undefined;
     deployed_version?: string | null;
     has_pending_deployment?: boolean;
 }

 export interface AppRequest {
     name: string;
     server_id: string;
-    remote_path: string;
-    service_name: string;
     domain: string;
+    // remote_path and service_name are now derived server-side from `name`.
+    // http_port may be supplied but defaults to auto-allocate.
+    http_port?: number;
 }
```

### 2.2 `lib/api/servers/types.ts`

```diff
 export interface Server {
     id: string;
     name: string;
     host: string;
     port: number;
     root_username: string;
     app_username: string;
     use_ssh_agent: boolean;
     manual_key_path: string;
     setup_complete: boolean;
     security_locked: boolean;
+    proxy_email: string;        // ACME email; "" → Caddy default
 }

 export interface ServerRequest {
     name: string;
     host: string;
     port: number;
     root_username: string;
     app_username: string;
     use_ssh_agent: boolean;
     manual_key_path: string;
+    proxy_email?: string;
 }
```

### 2.3 `lib/api/servers/setup.ts`

Add Caddy fields to `SetupInfo`:

```diff
 export interface SetupInfo {
     os: string;
     architecture: string;
     hostname: string;
     pocketbase_setup: boolean;
+    caddy_installed: boolean;
+    caddy_version: string;          // "" if not installed
+    caddy_status: 'running' | 'stopped' | 'failed' | 'unknown';
     installed_apps: string[];
 }
```

### 2.4 New endpoint clients

```ts
// lib/api/apps/crud.ts
async migrateProxy(id: string): Promise<{ http_port: number; status: App['status'] }> {
    return this.pb.send(`/api/apps/${id}/migrate-proxy`, { method: 'POST' });
}

async checkProxyConfig(id: string): Promise<{ valid: boolean; drift: string[] }> {
    return this.pb.send(`/api/apps/${id}/proxy-status`, { method: 'GET' });
}
```

(`proxy-status` is optional and only needed if we want a "fragment drifted" indicator in the UI; can be deferred.)

---

## 3. UI surface — what users see

### 3.1 Server-create flow

`ServerCreateModal.svelte` — one new field:

| Field          | Type     | Required | Default | Helper text                                                                 |
| -------------- | -------- | :------: | ------- | --------------------------------------------------------------------------- |
| `proxy_email`  | email    | no       | `""`    | "Email Let's Encrypt uses for ACME registration & expiry notices. Optional but recommended." |

Renders below `app_username`.

### 3.2 Server-setup wizard

`ServerList.svelte` already calls `setup` and `secure` actions per server. Wizard text needs three tweaks:

1. Step list now includes **"Installing Caddy"** and **"Configuring base Caddyfile"** (just labels in the progress UI; backend reports them via the existing `step` callbacks).
2. After setup completes, the server card surfaces a small **"Caddy: vX.X.X — running"** line under "App: pocketbase" (sourced from `SetupInfo.caddy_version` / `caddy_status`).
3. `TroubleshootModal.svelte` adds a Caddy block: install state, service state, Caddyfile validity (`caddy validate` exit code).

### 3.3 App-create flow

`AppCreateModal.svelte` shrinks. Two fields removed, one shown read-only.

| Field              | Before                | After                                                                 |
| ------------------ | --------------------- | --------------------------------------------------------------------- |
| `name`             | text input            | text input + **per-server uniqueness** check (debounce, hit `/api/apps?server_id=…&name=…`) |
| `server_id`        | select                | unchanged                                                             |
| `domain`           | text input            | text input + **per-server uniqueness** check                          |
| `remote_path`      | text input (advanced) | **removed** — server derives `/opt/pocketbase/apps/<name>`            |
| `service_name`     | text input (advanced) | **removed** — server derives `<name>` (or `pocketbase-<name>`)        |
| `http_port`        | —                     | **read-only preview**: "Will be assigned (next free: 8093)" — fetched live from `/api/apps/next-port?server_id=…` |
| `version_number`   | text input            | unchanged                                                             |
| `version_notes`    | text input            | unchanged                                                             |
| Initial ZIP        | upload                | unchanged                                                             |

Below the form, show a **"What will happen"** preview block:

```
On deploy:
  • PocketBase will run as pocketbase user on 127.0.0.1:8093
  • Caddy will route https://blog.example.com → 127.0.0.1:8093
  • A TLS certificate will be issued automatically
```

This is informational only — pulled from `appsStore` derived state.

### 3.4 App list (`AppList.svelte`)

Three changes to the table:

1. **New "Port" column** between "Server" and "Status", showing `127.0.0.1:<port>`. Mono font, click-to-copy.
2. **Status column**: render `needs_migration` as an amber `StatusBadge` with text "Needs proxy migration".
3. **Actions column**: when `status === 'needs_migration'`, replace the "Manage" button with a primary **"Migrate to proxy"** button. Clicking it opens `ProxyMigrateModal` (§3.6).

### 3.5 Manage app modal

`ManageAppModal.svelte` gains a **"Proxy"** section between "Versions" and "Deployments":

```
Proxy
├─ Public URL:    https://blog.example.com         [↗ open]
├─ Loopback:      127.0.0.1:8093                   [📋 copy]
├─ Caddy config:  /etc/caddy/conf.d/blog.caddy     [view]
└─ Status:        ✓ Active   (or)   ⚠ Drifted — fragment doesn't match expected
                                       [Reapply]
```

"View" opens a read-only modal showing the generated Caddy fragment (no editing yet; manual edits documented as not supported until an `extra_caddy_directives` field exists).

"Reapply" calls `appsStore.reapplyProxy(id)` which POSTs to a new endpoint that re-renders the fragment and reloads Caddy. (Backend addition — also worth a line in `MULTI_APP.md` §6.)

### 3.6 Proxy migration modal (new)

`lib/components/modals/ProxyMigrateModal.svelte`. Triggered from the `needs_migration` row in `AppList`. Body:

```
Migrate "blog" to Caddy reverse proxy

This will:
  1. Stop the service (downtime ~5s)
  2. Rewrite the systemd unit to bind 127.0.0.1:8093
  3. Drop privileged-port capability from the binary
  4. Add a Caddy config fragment
  5. Start the service & reload Caddy
  6. Verify https://blog.example.com responds

[Cancel]                                          [Migrate]
```

On click → `appsStore.migrateProxy(app.id)`. Show a stepper with live status sourced from a temporary deployment-like record (or just poll the app's `status` field via the existing realtime channel until it transitions out of `needs_migration`).

### 3.7 Dashboard

`Dashboard.svelte` / `.ts` add one card:

```
┌──────────────────────────┐
│ Apps needing migration   │
│           3              │
│   View list →            │
└──────────────────────────┘
```

Click navigates to `/apps?filter=needs_migration`. Implement filter via a query-param-driven `$derived` in `AppList`.

### 3.8 Per-server detail view (optional, recommended)

Currently there's no `/servers/[id]` route. With multi-app, a server detail page becomes useful:

- Apps on this server (with ports, status, links).
- Caddy status block (version, last reload, fragment count).
- Port allocation table (`8090` → `blog`, `8091` → `(free)`, `8092` → `shop`, …).
- Quick "Add app to this server" button.

Add `routes/servers/[id]/+page.svelte`. Link from each row in `ServerList`.

---

## 4. Page-by-page change list

| File                                                       | Change                                                                                          |
| ---------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| **NEW** `lib/stores/client.svelte.ts`                      | Single `ApiClient` + realtime fan-out.                                                          |
| **NEW** `lib/stores/apps.svelte.ts`                        | apps + derived projections + `migrateProxy`, `reapplyProxy`.                                    |
| **NEW** `lib/stores/servers.svelte.ts`                     | servers + setup/security/troubleshoot progress maps.                                            |
| **NEW** `lib/stores/deployments.svelte.ts`                 | deployments + per-app live log streams.                                                         |
| **NEW** `lib/stores/versions.svelte.ts`                    | versions, keyed by `app_id`.                                                                    |
| **NEW** `lib/stores/ui.svelte.ts`                          | modal + toast bus.                                                                              |
| **NEW** `lib/stores/settings.svelte.ts`                    | replaces `Settings.ts` legacy store; persists via `$effect`.                                    |
| **NEW** `lib/stores/splash.svelte.ts`                      | replaces `SplashScreen.ts` legacy store.                                                        |
| **NEW** `lib/stores/index.ts`                              | barrel.                                                                                         |
| `lib/api/client.ts`                                        | unchanged — pure HTTP client; new stores wrap it.                                               |
| `lib/api/apps/{crud,types}.ts`                             | add `http_port`, `needs_migration`, `migrateProxy()`, `nextFreePort()`, `reapplyProxy()`.       |
| `lib/api/servers/{crud,setup,types}.ts`                    | add `proxy_email`, Caddy fields in `SetupInfo`.                                                 |
| `lib/components/main/AppList.svelte`                       | drop `AppListLogic`; read from `appsStore`/`serversStore`; add Port column + migration row.     |
| `lib/components/main/AppList.ts`                           | **delete.**                                                                                     |
| `lib/components/main/ServerList.svelte`                    | drop `ServerListLogic`; new Caddy line in cards; link to `/servers/[id]`.                       |
| `lib/components/main/ServerList.ts`                        | **delete.**                                                                                     |
| `lib/components/main/Dashboard.svelte`/`.ts`               | delete `.ts`; rewire to stores; add "Apps needing migration" card.                              |
| `lib/components/main/DeploymentsList.svelte`/`.ts`         | delete `.ts`; rewire to `deploymentsStore`.                                                     |
| `lib/components/main/Settings.svelte`/`.ts`                | rewire to `settings.svelte.ts`.                                                                 |
| `lib/components/main/SplashScreen.svelte`/`.ts`            | rewire to `splash.svelte.ts`.                                                                   |
| `lib/components/main/Navigation.svelte`/`.ts`              | can stay or move route data to a tiny `routes.svelte.ts`; low priority.                        |
| `lib/components/modals/AppCreateModal.svelte`              | drop `remote_path` and `service_name`; add read-only `http_port` preview + "what will happen" block. |
| `lib/components/modals/ServerCreateModal.svelte`           | add `proxy_email` field.                                                                        |
| `lib/components/modals/ManageAppModal.svelte`              | add "Proxy" section.                                                                            |
| **NEW** `lib/components/modals/ProxyMigrateModal.svelte`   | per §3.6.                                                                                       |
| **NEW** `lib/components/modals/ProxyFragmentViewModal.svelte` | read-only Caddyfile viewer for `ManageAppModal`.                                              |
| `lib/components/modals/TroubleshootModal.svelte`           | add Caddy block.                                                                                |
| `lib/components/partials/StatusBadge.{svelte,ts}`          | add `'needs_migration'` variant (amber).                                                        |
| **NEW** `routes/servers/[id]/+page.svelte`                 | server detail per §3.8.                                                                         |
| `routes/+layout.svelte`                                    | mount global modals via `ui.modal`; call `startRealtime()`; remove legacy store imports.        |

After deletions, the entire `lib/components/main/*.ts` family (`AppList.ts`, `ServerList.ts`, `Dashboard.ts`, `DeploymentsList.ts`, plus `Modal.ts`, `Navigation.ts`, `Settings.ts`, `SplashScreen.ts`) should be reviewed — most are now redundant. Keep only files that hold pure helpers (e.g. `StatusBadge.ts` is fine).

---

## 5. Migration order (do in this order, ship incrementally)

1. **Add `lib/stores/client.svelte.ts` + `ui.svelte.ts`** alongside the existing Logic classes. New modals can use `ui.open(...)` immediately. No breaking changes yet.
2. **Convert `splash` and `settings`** legacy stores to runes — these are small and isolated. Validates the pattern.
3. **Add `appsStore`** + realtime. Convert `AppList.svelte` to read from it. Delete `AppList.ts`. Leave Dashboard and ManageAppModal as-is for now — they'll keep working through the realtime echoes.
4. **Add `serversStore`, `deploymentsStore`, `versionsStore`**, each followed by converting their consumer page.
5. **Convert Dashboard.** Now multi-page state is genuinely shared.
6. **Backend `http_port` field** lands (per `MULTI_APP.md`). Frontend types updated, App-create modal simplified, App list gets the Port column. No migration UI yet.
7. **Backend `proxy_email` + Caddy install** lands. Server-create modal updated; Setup wizard / Troubleshoot show Caddy state.
8. **Backend `/api/apps/{id}/migrate-proxy`** lands. ProxyMigrateModal + `needs_migration` UI shipped. Dashboard gets the migration card.
9. **Optional: server detail page** for port allocation visibility.

Each step keeps the app shippable.

---

## 6. Conventions & ground rules for the refactor

- **`.svelte.ts` for any module that uses `$state`/`$derived`/`$effect`.** Plain `.ts` will silently fail to compile runes. SvelteKit ships the right loader.
- **Stores return getters, not bare `$state` bindings.** Prevents external code from re-assigning `appsStore.apps = []` and breaking reactivity.
- **One ApiClient, ever.** Every store imports `api` from `client.svelte.ts`. Tests can swap it via DI later if needed.
- **Optimistic updates everywhere mutating goes through PocketBase.** Apply locally → await network → realtime echo confirms → on error, revert + `ui.toast('error', …)`.
- **No `onMount(() => store.load())` in every page.** Hoist initial loads into `+layout.svelte` (or per-route `+page.ts` loaders) so navigating between pages doesn't re-fetch. Use `if (!appsStore.initialized) appsStore.load()` for idempotency.
- **No prop-drilled `onrefresh` / `onclose` callbacks** in new modals. Read from stores, dispatch via `ui.close()`.
- **Form state stays local.** A modal's inputs live in `$state` inside the `.svelte` — only the *submission result* mutates a store.
- **Realtime is authoritative**, but optimistic writes are fine because the realtime echo will reconcile any drift within ~50ms.
- **Don't put derived UI helpers in stores.** `formatTimestamp`, `getStatusBadge`, `hasUpdateAvailable` belong in `partials/` or a `lib/utils/derive.ts`. Stores are *only* data + actions.
- **Type the realtime payloads.** PB's `record` is `RecordModel`; cast to our typed interfaces at the `_onRealtime` boundary, not deeper.

---

## 7. Open questions (resolve before implementation)

1. **Do we want a manual `http_port` override** in the create form (advanced toggle), or always auto-allocate? — Recommendation: auto-only at first, add override later if anyone asks.
2. **`needs_migration` UI gating**: should we block *all* actions on a `needs_migration` app (deploy, manage versions, view logs), or just deploy? — Recommendation: block deploy only; let users keep browsing versions/logs.
3. **Realtime subscription scope**: subscribe to `*` per collection (current spec), or use filters to only stream apps/servers the user is "watching"? — Recommendation: `*` for now, this is a single-user local-only tool.
4. **Server detail page (§3.8)** — ship in the same PR as the migration UI, or follow-up? — Recommendation: follow-up; not blocking.
5. **Old Svelte stores in `Settings.ts`/`SplashScreen.ts`** — leave the files in place and re-export from the new runes stores for a release, or delete immediately? — Recommendation: delete; nothing else imports them.

---

## 8. What this doc does NOT cover

- Server-side implementation of any new endpoint — see `MULTI_APP.md` §3, §4, §6.
- Tests. (No test harness exists in the frontend yet; out of scope here.)
- Visual design — colors/spacing stay consistent with existing Tailwind utility usage.
- i18n — strings are English-only today; same after refactor.
- Mobile/responsive — current desktop-first layout is fine for an admin tool.

When this lands: every page is reading from one source of truth, deploy progress and migration status update live across the UI, the App-create form is shorter, and Caddy install/migration is a first-class flow rather than a footnote in CLI docs.
