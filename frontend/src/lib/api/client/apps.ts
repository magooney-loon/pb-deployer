import { BaseClient } from '../base.js';
import type { AppRequest, App, AppResponse, HealthCheckResponse } from '../types.js';

export class AppsClient extends BaseClient {
	async getApps() {
		console.log('Getting apps via REST API...');
		try {
			const response = await fetch(`${this.baseURL}/api/apps`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Apps API response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get apps:', error);
			throw error;
		}
	}

	async getApp(id: string): Promise<AppResponse> {
		console.log('Getting app:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App API response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get app:', error);
			throw error;
		}
	}

	async createApp(data: AppRequest): Promise<App> {
		console.log('Creating app:', data);
		try {
			const response = await fetch(`${this.baseURL}/api/apps`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(data)
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const app = await response.json();
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
			const response = await fetch(`${this.baseURL}/api/apps/${id}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(data)
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const app = await response.json();
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
			const response = await fetch(`${this.baseURL}/api/apps/${id}`, {
				method: 'DELETE'
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App deleted:', id);
			return data;
		} catch (error) {
			console.error('Failed to delete app:', error);
			throw error;
		}
	}

	async getAppsByServer(serverId: string) {
		console.log('Getting apps by server:', serverId);
		try {
			const response = await fetch(`${this.baseURL}/api/apps?server_id=${serverId}`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Apps by server response:', data);
			return {
				server_id: serverId,
				apps: data.apps || []
			};
		} catch (error) {
			console.error('Failed to get apps by server:', error);
			throw error;
		}
	}

	async checkAppHealth(id: string): Promise<HealthCheckResponse> {
		console.log('Checking app health:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/status`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
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

	async startApp(id: string): Promise<{
		app_id: string;
		service_name: string;
		action: string;
		success: boolean;
		status: string;
		message: string;
		error?: string;
		timestamp: string;
	}> {
		console.log('Starting app service:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/start`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App service started:', data);
			return data;
		} catch (error) {
			console.error('Failed to start app service:', error);
			throw error;
		}
	}

	async stopApp(id: string): Promise<{
		app_id: string;
		service_name: string;
		action: string;
		success: boolean;
		status: string;
		message: string;
		error?: string;
		timestamp: string;
	}> {
		console.log('Stopping app service:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/stop`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App service stopped:', data);
			return data;
		} catch (error) {
			console.error('Failed to stop app service:', error);
			throw error;
		}
	}

	async restartApp(id: string): Promise<{
		app_id: string;
		service_name: string;
		action: string;
		success: boolean;
		status: string;
		message: string;
		error?: string;
		timestamp: string;
	}> {
		console.log('Restarting app service:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/restart`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App service restarted:', data);
			return data;
		} catch (error) {
			console.error('Failed to restart app service:', error);
			throw error;
		}
	}
}
