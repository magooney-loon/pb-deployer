// Re-export CRUD-related types from client directories
export type { Server, ServerRequest, ServerResponse } from './client/servers/types.js';

export type { App, AppRequest, AppResponse } from './client/apps/types.js';

export type { Version } from './client/version/types.js';

export type { Deployment } from './client/deployment/types.js';

// Re-export utility functions
export { getStatusColor, getStatusIcon, formatTimestamp } from './utils.js';

// Export main client class
export { ApiClient } from './client.js';

// Create and export singleton instance for backward compatibility
import { ApiClient } from './client.js';
export const api = new ApiClient();
