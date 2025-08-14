# API Client Structure

This directory contains the restructured API client code, organized by domain with each client having its own directory and types. No more centralized `types.ts` file - all types are imported directly from their respective client directories.

## Directory Structure

```
client/
├── README.md                 # This file
├── index.ts                  # Main exports
├── servers/
│   ├── index.ts             # ServerClient & ServerCrudClient exports
│   ├── servers.ts           # ServerClient class (custom APIs)
│   ├── crud.ts              # ServerCrudClient class (PocketBase CRUD)
│   └── types.ts             # Server-related types
├── apps/
│   ├── index.ts             # AppsClient & AppsCrudClient exports
│   ├── apps.ts              # AppsClient class (custom APIs)
│   ├── crud.ts              # AppsCrudClient class (PocketBase CRUD)
│   └── types.ts             # App-related types
├── version/
│   ├── index.ts             # VersionClient & VersionCrudClient exports
│   ├── version.ts           # VersionClient class (custom APIs)
│   ├── crud.ts              # VersionCrudClient class (PocketBase CRUD)
│   └── types.ts             # Version-related types
└── deployment/
    ├── index.ts             # DeploymentClient & DeploymentCrudClient exports
    ├── deployment.ts        # DeploymentClient class (custom APIs)
    ├── crud.ts              # DeploymentCrudClient class (PocketBase CRUD)
    └── types.ts             # Deployment-related types
```

## Usage

### Import Individual Clients

```typescript
// Full-featured clients (CRUD + custom APIs)
import { ServerClient } from './client/servers';
import { AppsClient } from './client/apps';
import { VersionClient } from './client/version';
import { DeploymentClient } from './client/deployment';

// Pure CRUD clients (PocketBase operations only)
import { ServerCrudClient } from './client/servers';
import { AppsCrudClient } from './client/apps';
import { VersionCrudClient } from './client/version';
import { DeploymentCrudClient } from './client/deployment';
```

### Import Types

```typescript
import type { Server, ServerRequest, TroubleshootResult } from './client/servers/types';
import type { App, AppRequest, HealthCheckResponse } from './client/apps/types';
import type { Version } from './client/version/types';
import type { Deployment } from './client/deployment/types';
```

### Use the Composite Client

```typescript
import { ApiClient } from './client';

const api = new ApiClient('http://localhost:8090');
await api.getServers();
await api.getApps();
```

## CRUD-Only Operations

All clients now provide only basic PocketBase CRUD operations:

### CRUD Clients (PocketBase Operations Only)
- **ServerCrudClient**: getServers, getServer, createServer, updateServer, deleteServer
- **AppsCrudClient**: getApps, getApp, createApp, updateApp, deleteApp, getAppsByServer
- **VersionCrudClient**: getVersions, getVersion, createVersion, updateVersion, deleteVersion, getAppVersions
- **DeploymentCrudClient**: getDeployments, getDeployment, createDeployment, updateDeployment, deleteDeployment, getAppDeployments, getVersionDeployments

### Wrapper Clients (Extend CRUD Clients)
- **ServerClient**: extends ServerCrudClient (CRUD operations only)
- **AppsClient**: extends AppsCrudClient (CRUD operations only)
- **VersionClient**: extends VersionCrudClient (CRUD operations only)
- **DeploymentClient**: extends DeploymentCrudClient (CRUD operations only)

All custom API operations (like testServerConnection, runServerSetup, checkAppHealth, uploadVersionFiles, etc.) have been removed.

## Type Organization

Each client directory contains its own CRUD-related types:

- **servers/types.ts**: Server, ServerRequest, ServerResponse
- **apps/types.ts**: App, AppRequest, AppResponse
- **version/types.ts**: Version
- **deployment/types.ts**: Deployment

All custom API-related types have been removed.

## Direct Imports

No more centralized `types.ts` file! All CRUD types are imported directly from their source:

```typescript
import type { Server, ServerRequest, ServerResponse } from './client/servers/types';
import type { App, AppRequest, AppResponse } from './client/apps/types';
import type { Version } from './client/version/types';
import type { Deployment } from './client/deployment/types';
```

The main API `index.ts` re-exports only CRUD-related types for convenience.

## Benefits

1. **Better Organization**: Each domain has its own directory with related types
2. **CRUD-Only Focus**: Clean, simple PocketBase operations without custom API complexity
3. **Easier Maintenance**: Changes to one client don't affect others  
4. **Direct Imports**: No more unnecessary re-exports through a central types file
5. **Type Safety**: Each client has strongly typed interfaces for CRUD operations
6. **Scalability**: Easy to add new clients without cluttering the main directory
7. **No Circular Dependencies**: Clean import paths without complex re-export chains
8. **Simplified Usage**: Pure CRUD operations make the API predictable and consistent