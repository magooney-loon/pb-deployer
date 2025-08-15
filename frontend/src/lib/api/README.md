# API Client

TypeScript API client for PocketBase deployment management with full CRUD operations and type safety.

Features centralized client management, resource-specific CRUD operations, and comprehensive type definitions.

## Architecture

- **Centralized Client**: Single `ApiClient` instance manages all operations
- **Resource-Specific CRUDs**: Dedicated clients for apps, servers, versions, deployments
- **Type Safety**: Full TypeScript interfaces for all data models
- **Shared PocketBase**: Single PocketBase instance across all CRUD clients
- **Error Handling**: Consistent error handling with detailed logging
- **Relational Data**: Automatic loading of related resources

## Core Features

- **Server Management**: SSH target configuration and status tracking
- **App Deployment**: Application lifecycle management with domains
- **Version Control**: ZIP package uploads with rollback support
- **Deployment Tracking**: Real-time status updates and logging
- **Relational Queries**: Automatic inclusion of related data
- **Status Utilities**: Color coding and icon helpers for UI
- **Timestamp Formatting**: Localized date/time display

## Resources

### Apps
Complete application lifecycle management with server associations.

```typescript
// Get all apps
const { apps } = await api.apps.getApps();

// Get single app with relations
const appResponse = await api.apps.getApp('app_id');
// Returns: { ...app, server?, versions?, deployments? }

// Create new app
const app = await api.apps.createApp({
    name: 'my-pocketbase-app',
    server_id: 'server_123',
    remote_path: '/opt/pocketbase/apps/my-app',
    service_name: 'my-app',
    domain: 'myapp.example.com'
});

// Update app
await api.apps.updateApp('app_id', { domain: 'newdomain.com' });

// Get apps by server
const { apps } = await api.apps.getAppsByServer('server_id');

// Delete app
await api.apps.deleteApp('app_id');
```

### Servers
SSH deployment target management with security features.

```typescript
// Get all servers
const { servers } = await api.servers.getServers();

// Get single server with apps
const serverResponse = await api.servers.getServer('server_id');
// Returns: { ...server, apps? }

// Create new server
const server = await api.servers.createServer({
    name: 'production-server',
    host: '192.168.1.100',
    port: 22,
    root_username: 'root',
    app_username: 'pbuser',
    use_ssh_agent: true,
    manual_key_path: '/path/to/key'
});

// Update server
await api.servers.updateServer('server_id', {
    setup_complete: true,
    security_locked: true
});

// Delete server
await api.servers.deleteServer('server_id');
```

### Versions
Deployment package management with ZIP file support.

```typescript
// Get all versions
const { versions } = await api.versions.getVersions();

// Get single version
const version = await api.versions.getVersion('version_id');

// Create new version
const version = await api.versions.createVersion({
    app_id: 'app_123',
    version_number: 'v1.2.3',
    notes: 'Bug fixes and improvements'
});

// Update version with deployment package
await api.versions.updateVersion('version_id', {
    deployment_zip: 'file_hash_or_path',
    notes: 'Updated deployment package'
});

// Get versions by app
const { versions } = await api.versions.getAppVersions('app_id');

// Delete version
await api.versions.deleteVersion('version_id');
```

### Deployments
Real-time deployment tracking with status updates.

```typescript
// Get all deployments
const { deployments } = await api.deployments.getDeployments();

// Get single deployment
const deployment = await api.deployments.getDeployment('deployment_id');

// Create new deployment
const deployment = await api.deployments.createDeployment({
    app_id: 'app_123',
    version_id: 'version_456',
    status: 'pending'
});

// Update deployment status
await api.deployments.updateDeployment('deployment_id', {
    status: 'running',
    started_at: new Date().toISOString(),
    logs: 'Deployment started...'
});

// Complete deployment
await api.deployments.updateDeployment('deployment_id', {
    status: 'success',
    completed_at: new Date().toISOString(),
    logs: 'Deployment completed successfully'
});

// Get deployments by app
const { deployments } = await api.deployments.getAppDeployments('app_id');

// Get deployments by version
const { deployments } = await api.deployments.getVersionDeployments('version_id');

// Delete deployment
await api.deployments.deleteDeployment('deployment_id');
```

## Type Definitions

### Core Interfaces

```typescript
// Application instance
interface App {
    id: string;
    created: string;
    updated: string;
    name: string;
    server_id: string;
    remote_path: string;
    service_name: string;
    domain: string;
    current_version: string;
    status: string;
}

// SSH deployment target
interface Server {
    id: string;
    created: string;
    updated: string;
    name: string;
    host: string;
    port: number;
    root_username: string;
    app_username: string;
    use_ssh_agent: boolean;
    manual_key_path: string;
    setup_complete: boolean;
    security_locked: boolean;
}

// Version package
interface Version {
    id: string;
    created: string;
    updated: string;
    app_id: string;
    version_number: string;
    deployment_zip: string;
    notes: string;
}

// Deployment operation
interface Deployment {
    id: string;
    created: string;
    updated: string;
    app_id: string;
    version_id: string;
    status: string;
    logs: string;
    started_at?: string;
    completed_at?: string;
}
```

### Request/Response Types

```typescript
// For creating resources
interface AppRequest {
    name: string;
    server_id: string;
    remote_path: string;
    service_name: string;
    domain: string;
}

interface ServerRequest {
    name: string;
    host: string;
    port: number;
    root_username: string;
    app_username: string;
    use_ssh_agent: boolean;
    manual_key_path: string;
}

// Enhanced responses with relations
interface AppResponse extends App {
    server?: Server;
    versions?: Version[];
    deployments?: Deployment[];
}

interface ServerResponse extends Server {
    apps?: App[];
}
```

## Utility Functions

### Status Helpers

```typescript
import { getStatusColor, getStatusIcon } from '$lib/api';

// Get status color for UI
const color = getStatusColor('online');    // 'green'
const color = getStatusColor('failed');    // 'red'
const color = getStatusColor('pending');   // 'yellow'

// Get status icon for display
const icon = getStatusIcon('success');     // '✅'
const icon = getStatusIcon('error');       // '❌'
const icon = getStatusIcon('running');     // '🔄'
```

### Timestamp Formatting

```typescript
import { formatTimestamp } from '$lib/api';

// Format ISO timestamp for display
const formatted = formatTimestamp('2024-01-15T10:30:00Z');
// Returns: "1/15/2024, 10:30:00 AM" (locale-specific)
```

## Error Handling

All CRUD operations include consistent error handling:

```typescript
try {
    const app = await api.apps.createApp(appData);
    console.log('App created:', app.name);
} catch (error) {
    console.error('Failed to create app:', error);
    // Handle error appropriately
}
```

Errors are logged with descriptive messages and the original error is re-thrown for application-level handling.

## Advanced Usage

### Custom PocketBase Configuration

```typescript
import PocketBase from 'pocketbase';
import { ApiClient } from '$lib/api';

// Custom PocketBase setup
const pb = new PocketBase('https://your-pb-instance.com');
pb.authStore.loadFromCookie(document.cookie);

// Use with ApiClient
const api = new ApiClient('https://your-pb-instance.com');
const customPb = api.getPocketBase();
```

### Batch Operations

```typescript
// Get related data efficiently
const appWithDetails = await api.apps.getApp('app_id');
// Automatically includes server, versions, and deployments

const serverWithApps = await api.servers.getServer('server_id');
// Automatically includes all associated apps
```

### Real-time Updates

The underlying PocketBase client supports real-time subscriptions:

```typescript
const pb = api.getPocketBase();

// Subscribe to deployment updates
pb.collection('deployments').subscribe('*', (e) => {
    console.log('Deployment update:', e.action, e.record);
});
```

## Collections Schema

### apps
- `name` (string): Application name
- `server_id` (relation): Target deployment server
- `remote_path` (string): Server filesystem path
- `service_name` (string): Systemd service identifier
- `domain` (string): Public domain/URL
- `current_version` (string): Active version identifier
- `status` (string): Runtime status (online/offline/unknown)

### servers
- `name` (string): Server identifier
- `host` (string): SSH hostname/IP
- `port` (number): SSH port
- `root_username` (string): Administrative user
- `app_username` (string): Application user
- `use_ssh_agent` (bool): SSH agent authentication
- `manual_key_path` (string): Private key file path
- `setup_complete` (bool): Initial setup status
- `security_locked` (bool): Security hardening status

### versions
- `app_id` (relation): Parent application
- `version_number` (string): Version identifier
- `deployment_zip` (file): Deployment package
- `notes` (text): Release notes

### deployments
- `app_id` (relation): Target application
- `version_id` (relation): Version being deployed
- `status` (string): Operation status (pending/running/success/failed)
- `logs` (text): Deployment output logs
- `started_at` (datetime): Operation start time
- `completed_at` (datetime): Operation completion time

## Best Practices

1. **Error Handling**: Always wrap API calls in try-catch blocks
2. **Type Safety**: Use provided TypeScript interfaces
3. **Resource Relations**: Leverage automatic relation loading where possible
4. **Status Display**: Use utility functions for consistent UI representation
5. **Logging**: Errors are automatically logged, but add application-specific logging as needed
6. **Performance**: Use specific getters (e.g., `getAppsByServer`) for filtered queries
