import { BaseClient } from './base.js';
import type { Server, ServerRequest, ServerResponse, ServerStatus, App } from './types.js';

export class ServerClient extends BaseClient {
	// Server CRUD operations using PocketBase SDK
	async getServers() {
		console.log('Getting servers via PocketBase...');
		try {
			const records = await this.pb.collection('servers').getFullList<Server>({
				sort: '-created'
			});
			console.log('PocketBase servers response:', records);

			// Transform to match expected format
			const result = { servers: records || [] };
			console.log('getServers result:', result);
			return result;
		} catch (error) {
			console.error('Failed to get servers:', error);
			throw error;
		}
	}

	async getServer(id: string) {
		console.log('Getting server:', id);
		try {
			const server = await this.pb.collection('servers').getOne<Server>(id);
			console.log('PocketBase server response:', server);

			const response: ServerResponse = { ...server };

			// Optionally include associated apps
			try {
				const apps = await this.pb.collection('apps').getFullList<App>({
					filter: `server_id = "${id}"`
				});
				response.apps = apps;
			} catch (appsError) {
				console.warn('Failed to load apps for server:', appsError);
			}

			return response;
		} catch (error) {
			console.error('Failed to get server:', error);
			throw error;
		}
	}

	async createServer(data: ServerRequest) {
		console.log('Creating server:', data);
		try {
			const server = await this.pb.collection('servers').create<Server>(data);
			console.log('Server created:', server);
			return server;
		} catch (error) {
			console.error('Failed to create server:', error);
			throw error;
		}
	}

	async updateServer(id: string, data: Partial<ServerRequest>) {
		console.log('Updating server:', id, data);
		try {
			const server = await this.pb.collection('servers').update<Server>(id, data);
			console.log('Server updated:', server);
			return server;
		} catch (error) {
			console.error('Failed to update server:', error);
			throw error;
		}
	}

	async deleteServer(id: string) {
		console.log('Deleting server:', id);
		try {
			await this.pb.collection('servers').delete(id);
			console.log('Server deleted:', id);
			return { message: 'Server deleted successfully' };
		} catch (error) {
			console.error('Failed to delete server:', error);
			throw error;
		}
	}

	// Custom server operations (these use Go backend endpoints)
	async testServerConnection(id: string) {
		console.log('Testing server connection:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/test`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('Connection test response:', data);
			return data;
		} catch (error) {
			console.error('Connection test failed:', error);
			throw error;
		}
	}

	async runServerSetup(id: string) {
		console.log('Running server setup:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/setup`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('Server setup response:', data);
			return data;
		} catch (error) {
			console.error('Server setup failed:', error);
			throw error;
		}
	}

	async applySecurityLockdown(id: string) {
		console.log('Applying security lockdown:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/security`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('Security lockdown response:', data);
			return data;
		} catch (error) {
			console.error('Security lockdown failed:', error);
			throw error;
		}
	}

	async getServerStatus(id: string): Promise<ServerStatus> {
		console.log('Getting server status:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/status`);
			const data = await response.json();
			console.log('Server status response:', data);
			return data;
		} catch (error) {
			console.error('Get server status failed:', error);
			throw error;
		}
	}
}
