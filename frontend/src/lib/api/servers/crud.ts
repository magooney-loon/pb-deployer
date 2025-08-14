import { BaseClient } from '../base.js';
import type { ServerRequest, Server, ServerResponse, App } from './types.js';

export class ServerCrudClient extends BaseClient {
	// Basic PocketBase CRUD operations for servers
	async getServers() {
		try {
			const records = await this.pb.collection('servers').getFullList<Server>({
				sort: '-created'
			});

			// Transform to match expected format
			const result = { servers: records || [] };
			return result;
		} catch (error) {
			console.error('Failed to get servers:', error);
			throw error;
		}
	}

	async getServer(id: string): Promise<ServerResponse> {
		try {
			const server = await this.pb.collection('servers').getOne<Server>(id);

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
		try {
			const server = await this.pb.collection('servers').create<Server>(data);
			return server;
		} catch (error) {
			console.error('Failed to create server:', error);
			throw error;
		}
	}

	async updateServer(id: string, data: Partial<ServerRequest>): Promise<Server> {
		try {
			const server = await this.pb.collection('servers').update<Server>(id, data);
			return server;
		} catch (error) {
			console.error('Failed to update server:', error);
			throw error;
		}
	}

	async deleteServer(id: string) {
		try {
			await this.pb.collection('servers').delete(id);
			return { message: 'Server deleted successfully' };
		} catch (error) {
			console.error('Failed to delete server:', error);
			throw error;
		}
	}
}
