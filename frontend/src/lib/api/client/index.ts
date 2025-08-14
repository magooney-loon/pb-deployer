// Export all specialized client classes
export { ServerClient } from './servers/index.js';
export { AppsClient } from './apps/index.js';
export { VersionClient } from './version/index.js';
export { DeploymentClient } from './deployment/index.js';

// Export CRUD client classes for direct use
export { ServerCrudClient } from './servers/crud.js';
export { AppsCrudClient } from './apps/crud.js';
export { VersionCrudClient } from './version/crud.js';
export { DeploymentCrudClient } from './deployment/crud.js';

// Re-export types from all clients (explicit exports to avoid conflicts)
export type { Server, ServerRequest, ServerResponse } from './servers/types.js';

export type { App, AppRequest, AppResponse } from './apps/types.js';

export type { Version } from './version/types.js';

export type { Deployment } from './deployment/types.js';

// Re-export the main composite client
export { ApiClient } from '../client.js';
