// Re-export all types
export type {
	Server,
	App,
	Version,
	Deployment,
	ServerRequest,
	AppRequest,
	ServerResponse,
	AppResponse,
	HealthCheckResponse,
	SetupStep,
	ConnectionInfo,
	ServerStatus
} from './types.js';

// Re-export utility functions
export {
	getStatusColor,
	getStatusIcon,
	formatTimestamp
} from './utils.js';

// Export main client class
export { ApiClient } from './client.js';

// Export individual client classes for advanced usage
export { BaseClient } from './base.js';
export { ServerClient } from './servers.js';
export { AppClient } from './apps.js';
export { RealtimeClient } from './realtime.js';

// Create and export singleton instance for backward compatibility
import { ApiClient } from './client.js';
export const api = new ApiClient();
