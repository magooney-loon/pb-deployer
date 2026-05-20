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
	proxy_email: string;
}

export interface ServerRequest {
	name: string;
	host: string;
	port: number;
	root_username: string;
	app_username: string;
	use_ssh_agent: boolean;
	manual_key_path: string;
	proxy_email?: string;
}

export interface ServerResponse extends Server {
	apps?: App[];
}

// Import App interface for ServerResponse
import type { App } from '../apps/types.js';
export type { App };
