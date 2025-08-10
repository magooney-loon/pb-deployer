import { BaseClient } from './base.js';
import type {
	App,
	AppRequest,
	AppResponse,
	Version,
	Deployment,
	HealthCheckResponse,
	Server
} from './types.js';

export class AppClient extends BaseClient {
	// App CRUD operations using PocketBase SDK
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

	async getApp(id: string) {
		console.log('Getting app:', id);
		try {
			const app = await this.pb.collection('apps').getOne<App>(id);
			console.log('PocketBase app response:', app);

			const response: AppResponse = { ...app };

			// Optionally include server, versions, and deployments
			try {
				const server = await this.pb.collection('servers').getOne<Server>(app.server_id);
				response.server = server;
			} catch (serverError) {
				console.warn('Failed to load server for app:', serverError);
			}

			return response;
		} catch (error) {
			console.error('Failed to get app:', error);
			throw error;
		}
	}

	async createApp(data: AppRequest) {
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

	async updateApp(id: string, data: Partial<AppRequest>) {
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
			const apps = await this.pb.collection('apps').getFullList<App>({
				filter: `server_id = "${serverId}"`,
				sort: '-created'
			});
			console.log('Apps by server response:', apps);
			return {
				server_id: serverId,
				apps: apps || []
			};
		} catch (error) {
			console.error('Failed to get apps by server:', error);
			throw error;
		}
	}

	// Custom app operations (these use Go backend endpoints)
	async checkAppHealth(id: string): Promise<HealthCheckResponse> {
		console.log('Checking app health:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/health`);
			const data = await response.json();
			console.log('App health response:', data);
			return data;
		} catch (error) {
			console.error('App health check failed:', error);
			throw error;
		}
	}

	async runAppHealthCheck(id: string): Promise<HealthCheckResponse> {
		console.log('Running app health check:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/health-check`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('App health check response:', data);
			return data;
		} catch (error) {
			console.error('App health check failed:', error);
			throw error;
		}
	}

	// Version and deployment operations
	async getAppVersions(id: string) {
		console.log('Getting app versions:', id);
		try {
			const versions = await this.pb.collection('versions').getFullList<Version>({
				filter: `app_id = "${id}"`,
				sort: '-created'
			});
			console.log('App versions response:', versions);
			return {
				app_id: id,
				versions: versions || []
			};
		} catch (error) {
			console.error('Failed to get app versions:', error);
			throw error;
		}
	}

	async getAppDeployments(id: string) {
		console.log('Getting app deployments:', id);
		try {
			const deployments = await this.pb.collection('deployments').getFullList<Deployment>({
				filter: `app_id = "${id}"`,
				sort: '-created'
			});
			console.log('App deployments response:', deployments);
			return {
				app_id: id,
				deployments: deployments || []
			};
		} catch (error) {
			console.error('Failed to get app deployments:', error);
			throw error;
		}
	}
}
