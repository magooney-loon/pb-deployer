import { BaseClient } from './base.js';
import type {
	ServerRequest,
	AppRequest,
	Server,
	App,
	Version,
	Deployment,
	ServerResponse,
	AppResponse,
	ServerStatus,
	HealthCheckResponse,
	SetupStep
} from './types.js';

export class ApiClient extends BaseClient {
	constructor(baseUrl: string = 'http://localhost:8090') {
		super(baseUrl);
	}

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

	async getApp(id: string): Promise<AppResponse> {
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

	// Real-time subscriptions for progress updates
	async subscribeToSetupProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		const subscription = `server_setup_${serverId}`;

		return await this.pb.realtime.subscribe(subscription, (e) => {
			try {
				console.log('Raw setup progress message:', e);

				// The message object itself contains the SetupStep data
				const setupStep = e as SetupStep;

				// Validate that we have the required properties
				if (!setupStep || typeof setupStep.step !== 'string') {
					console.warn('Invalid setup step data:', setupStep);
					return;
				}

				callback(setupStep);
			} catch (error) {
				console.error('Failed to parse setup progress data:', error, 'Raw message:', e);
			}
		});
	}

	async subscribeToSecurityProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		const subscription = `server_security_${serverId}`;

		return await this.pb.realtime.subscribe(subscription, (e) => {
			try {
				console.log('Raw security progress message:', e);

				// The message object itself contains the SetupStep data
				const securityStep = e as SetupStep;

				// Validate that we have the required properties
				if (!securityStep || typeof securityStep.step !== 'string') {
					console.warn('Invalid security step data:', securityStep);
					return;
				}

				callback(securityStep);
			} catch (error) {
				console.error('Failed to parse security progress data:', error, 'Raw message:', e);
			}
		});
	}

	// Unsubscribe from all realtime subscriptions
	async unsubscribeFromAll(): Promise<void> {
		await this.pb.realtime.unsubscribe();
	}
}
