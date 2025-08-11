export interface Server {
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

export interface App {
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

export interface Version {
	id: string;
	created: string;
	updated: string;
	app_id: string;
	version_number: string;
	deployment_zip: string;
	notes: string;
}

export interface Deployment {
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

export interface ConnectionDiagnostic {
	step: string;
	status: 'success' | 'warning' | 'error' | 'info';
	message: string;
	details?: string;
	suggestion?: string;
	duration_ms?: number;
	timestamp?: string;
	metadata?: Record<string, unknown>;
}

export interface TroubleshootResult {
	success: boolean;
	server_id: string;
	server_name: string;
	host: string;
	port: number;
	timestamp: string;
	diagnostics: ConnectionDiagnostic[];
	summary: string;
	has_errors: boolean;
	has_warnings: boolean;
	error_count: number;
	warning_count: number;
	success_count: number;
	suggestions: string[];
	client_ip?: string;
	connection_time?: number;
	can_auto_fix?: boolean;
	next_steps?: string[];
	severity?: 'critical' | 'high' | 'medium' | 'low' | 'info';
}

export interface QuickTroubleshootResult {
	success: boolean;
	server_id: string;
	server_name?: string;
	host: string;
	port: number;
	status: 'success' | 'warning' | 'error';
	message: string;
	suggestion: string;
	timestamp: string;
	client_ip?: string;
	connection_time?: number;
	attempts?: number;
	can_auto_fix?: boolean;
	next_steps?: string[];
	severity?: 'critical' | 'high' | 'medium' | 'low' | 'info';
}

export interface EnhancedTroubleshootResult extends TroubleshootResult {
	analysis: {
		pattern_detected: string;
		confidence: number;
		auto_fixable: boolean;
		priority: string;
		category: string;
		description?: string;
		immediate_action?: string;
		error_count: number;
		warning_count: number;
		total_issues: number;
	};
	recovery_plan: {
		has_critical_issues: boolean;
		critical_issues: string[];
		estimated_time: string;
		success_probability: number;
		requires_access: string[];
		steps: RecoveryStep[];
	};
	actionable_suggestions: ActionableSuggestion[];
	estimated_duration: string;
	requires_access: string[];
	auto_fix_available: boolean;
}

export interface RecoveryStep {
	step: number;
	title: string;
	description: string;
	command?: string;
	required: boolean;
	automated: boolean;
}

export interface ActionableSuggestion {
	category: string;
	action: string;
	description: string;
	automated: boolean;
	requires: string;
	priority: string;
	command?: string;
}

export interface AutoFixResult {
	success: boolean;
	server_id: string;
	server_name: string;
	host: string;
	port: number;
	fixes_applied: number;
	fixes: ConnectionDiagnostic[];
	summary: string;
	timestamp: string;
}

export interface ServerRequest {
	name: string;
	host: string;
	port: number;
	root_username: string;
	app_username: string;
	use_ssh_agent: boolean;
	manual_key_path: string;
}

export interface AppRequest {
	name: string;
	server_id: string;
	remote_path: string;
	service_name: string;
	domain: string;
}

export interface ServerResponse extends Server {
	apps?: App[];
}

export interface AppResponse extends App {
	server?: Server;
	versions?: Version[];
	deployments?: Deployment[];
}

export interface HealthCheckResponse {
	app_id: string;
	domain: string;
	status: string;
	url: string;
	timestamp: string;
	error?: string;
	details?: string;
}

export interface SetupStep {
	step: string;
	status: string;
	message: string;
	details?: string;
	timestamp: string;
	progress_pct: number;
}

export interface ConnectionInfo {
	connected: boolean;
	server_host?: string;
	server_port?: number;
	username?: string;
	is_root?: boolean;
	server_name?: string;
	remote_addr?: string;
	local_addr?: string;
}

export interface ServerStatus {
	server_id: string;
	setup_complete: boolean;
	security_locked: boolean;
	connection: string;
	timestamp: string;
	connection_error?: string;
	setup_status?: Record<string, boolean>;
	security_status?: Record<string, boolean>;
}
