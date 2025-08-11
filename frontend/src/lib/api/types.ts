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
	status: 'success' | 'warning' | 'error';
	message: string;
	details?: string;
	suggestion?: string;
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
}

export interface QuickTroubleshootResult {
	success: boolean;
	server_id: string;
	host: string;
	port: number;
	status: 'success' | 'warning' | 'error';
	message: string;
	suggestion: string;
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
