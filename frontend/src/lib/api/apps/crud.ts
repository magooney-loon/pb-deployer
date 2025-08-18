import PocketBase from 'pocketbase';
import type { AppRequest, App, AppResponse, Server, Version, Deployment } from './types.js';

export class AppsCrudClient {
	private pb: PocketBase;

	constructor(pb: PocketBase) {
		this.pb = pb;
	}

	async getApps() {
		try {
			const records = await this.pb.collection('apps').getFullList<App>({
				sort: '-created'
			});

			const result = { apps: records || [] };
			return result;
		} catch (error) {
			console.error('Failed to get apps:', error);
			throw error;
		}
	}

	async getApp(id: string): Promise<AppResponse> {
		try {
			const app = await this.pb.collection('apps').getOne<App>(id);

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
		try {
			const app = await this.pb.collection('apps').create<App>(data);
			return app;
		} catch (error) {
			console.error('Failed to create app:', error);
			throw error;
		}
	}

	async updateApp(id: string, data: Partial<AppRequest>): Promise<App> {
		try {
			const app = await this.pb.collection('apps').update<App>(id, data);
			return app;
		} catch (error) {
			console.error('Failed to update app:', error);
			throw error;
		}
	}

	async updateAppCurrentVersion(id: string, version: string): Promise<App> {
		try {
			const app = await this.pb.collection('apps').update<App>(id, {
				current_version: version
			});
			return app;
		} catch (error) {
			console.error('Failed to update app current version:', error);
			throw error;
		}
	}

	async deleteApp(id: string) {
		try {
			await this.pb.collection('apps').delete(id);
			return { message: 'App deleted successfully' };
		} catch (error) {
			console.error('Failed to delete app:', error);
			throw error;
		}
	}

	async getAppsByServer(serverId: string) {
		try {
			const records = await this.pb.collection('apps').getFullList<App>({
				filter: `server_id = "${serverId}"`,
				sort: '-created'
			});

			return {
				server_id: serverId,
				apps: records || []
			};
		} catch (error) {
			console.error('Failed to get apps by server:', error);
			throw error;
		}
	}

	async getLatestVersionForApp(appId: string): Promise<string | null> {
		try {
			const versions = await this.pb.collection('versions').getFullList({
				filter: `app_id = "${appId}"`,
				sort: '-created',
				limit: 1
			});

			return versions.length > 0 ? versions[0].version_number : null;
		} catch (error) {
			console.error('Failed to get latest version for app:', error);
			return null;
		}
	}

	async getAppsWithLatestVersions() {
		try {
			const apps = await this.pb.collection('apps').getFullList<App>({
				sort: '-created'
			});

			const appsWithVersions = await Promise.all(
				apps.map(async (app) => {
					const latestVersion = await this.getLatestVersionForApp(app.id);
					return {
						...app,
						latest_version: latestVersion || undefined
					};
				})
			);

			return { apps: appsWithVersions };
		} catch (error) {
			console.error('Failed to get apps with latest versions:', error);
			throw error;
		}
	}
}
