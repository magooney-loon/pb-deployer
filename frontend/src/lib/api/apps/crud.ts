import { BaseClient } from '../base.js';
import type { AppRequest, App, AppResponse, Server, Version, Deployment } from './types.js';

export class AppsCrudClient extends BaseClient {
	// Basic PocketBase CRUD operations for apps
	async getApps() {
		console.log('Getting apps via PocketBase...');
		try {
			const records = await this.pb.collection('apps').getFullList<App>({
				sort: '-created'
			});
			console.log('PocketBase apps response:', records);

			// Transform to match expected format
			const result = { apps: records || [] };
			console.log('getApps result:', result);
			return result;
		} catch (error) {
			console.error('Failed to get apps:', error);
			throw error;
		}
	}

	async getApp(id: string): Promise<AppResponse> {
		console.log('Getting app:', id);
		try {
			const app = await this.pb.collection('apps').getOne<App>(id);
			console.log('PocketBase app response:', app);

			const response: AppResponse = { ...app };

			// Optionally include associated server
			try {
				const server = await this.pb.collection('servers').getOne(app.server_id);
				response.server = server as unknown as Server;
			} catch (serverError) {
				console.warn('Failed to load server for app:', serverError);
			}

			// Optionally include versions
			try {
				const versions = await this.pb.collection('versions').getFullList({
					filter: `app_id = "${id}"`
				});
				response.versions = versions as unknown as Version[];
			} catch (versionsError) {
				console.warn('Failed to load versions for app:', versionsError);
			}

			// Optionally include deployments
			try {
				const deployments = await this.pb.collection('deployments').getFullList({
					filter: `app_id = "${id}"`
				});
				response.deployments = deployments as unknown as Deployment[];
			} catch (deploymentsError) {
				console.warn('Failed to load deployments for app:', deploymentsError);
			}

			return response;
		} catch (error) {
			console.error('Failed to get app:', error);
			throw error;
		}
	}

	async createApp(data: AppRequest): Promise<App> {
		console.log('Creating app:', data);
		try {
			const app = await this.pb.collection('apps').create<App>(data);
			console.log('App created:', app);
			return app;
		} catch (error) {
			console.error('Failed to create app:', error);
			throw error;
		}
	}

	async updateApp(id: string, data: Partial<AppRequest>): Promise<App> {
		console.log('Updating app:', id, data);
		try {
			const app = await this.pb.collection('apps').update<App>(id, data);
			console.log('App updated:', app);
			return app;
		} catch (error) {
			console.error('Failed to update app:', error);
			throw error;
		}
	}

	async deleteApp(id: string) {
		console.log('Deleting app:', id);
		try {
			await this.pb.collection('apps').delete(id);
			console.log('App deleted:', id);
			return { message: 'App deleted successfully' };
		} catch (error) {
			console.error('Failed to delete app:', error);
			throw error;
		}
	}

	async getAppsByServer(serverId: string) {
		console.log('Getting apps by server:', serverId);
		try {
			const records = await this.pb.collection('apps').getFullList<App>({
				filter: `server_id = "${serverId}"`,
				sort: '-created'
			});
			console.log('PocketBase apps by server response:', records);

			return {
				server_id: serverId,
				apps: records || []
			};
		} catch (error) {
			console.error('Failed to get apps by server:', error);
			throw error;
		}
	}
}
