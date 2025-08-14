import { BaseClient } from '../../base.js';
import type { ServerRequest, Server, ServerResponse, App } from './types.js';

export class ServerCrudClient extends BaseClient {
	// Basic PocketBase CRUD operations for servers
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

	async getServer(id: string): Promise<ServerResponse> {
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

	async createServer(data: ServerRequest): Promise<Server> {
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

	async updateServer(id: string, data: Partial<ServerRequest>): Promise<Server> {
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
}
