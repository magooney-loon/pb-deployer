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

export interface AppRequest {
	name: string;
	server_id: string;
	remote_path: string;
	service_name: string;
	domain: string;
}

export interface AppResponse extends App {
	server?: Server;
	versions?: Version[];
	deployments?: Deployment[];
}

// Import related interfaces
import type { Server } from '../servers/types.js';
import type { Version } from '../version/types.js';
import type { Deployment } from '../deployment/types.js';

export type { Server, Version, Deployment };
