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
	ServerStatus,
	ConnectionDiagnostic,
	TroubleshootResult,
	QuickTroubleshootResult,
	EnhancedTroubleshootResult,
	RecoveryStep,
	ActionableSuggestion,
	AutoFixResult
} from './types.js';

// Re-export utility functions
export { getStatusColor, getStatusIcon, formatTimestamp } from './utils.js';

// Export main client class
export { ApiClient } from './client.js';

// Create and export singleton instance for backward compatibility
import { ApiClient } from './client.js';
export const api = new ApiClient();
