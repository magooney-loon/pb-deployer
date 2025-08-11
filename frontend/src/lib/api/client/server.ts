import { BaseClient } from '../base.js';
import type {
	ServerRequest,
	Server,
	ServerResponse,
	ServerStatus,
	SetupStep,
	TroubleshootResult,
	QuickTroubleshootResult,
	EnhancedTroubleshootResult,
	AutoFixResult,
	App
} from '../types.js';

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

	async troubleshootServer(id: string): Promise<TroubleshootResult> {
		console.log('Running server troubleshooting:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/troubleshoot`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('Troubleshoot response:', data);
			return data;
		} catch (error) {
			console.error('Troubleshooting failed:', error);
			throw error;
		}
	}

	async quickTroubleshootServer(id: string): Promise<QuickTroubleshootResult> {
		console.log('Running quick server troubleshoot:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${id}/quick-troubleshoot`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				}
			});
			const data = await response.json();
			console.log('Quick troubleshoot response:', data);
			return data;
		} catch (error) {
			console.error('Quick troubleshooting failed:', error);
			throw error;
		}
	}

	async enhancedTroubleshootServer(serverId: string): Promise<EnhancedTroubleshootResult> {
		const response = await fetch(`${this.baseURL}/api/servers/${serverId}/troubleshoot/enhanced`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		if (!response.ok) {
			throw new Error(`Enhanced troubleshooting failed: ${response.statusText}`);
		}

		return response.json();
	}

	async autoFixServerIssues(serverId: string): Promise<AutoFixResult> {
		const response = await fetch(`${this.baseURL}/api/servers/${serverId}/auto-fix`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			}
		});

		if (!response.ok) {
			throw new Error(`Auto-fix failed: ${response.statusText}`);
		}

		return response.json();
	}

	async getServerStatus(serverId: string): Promise<ServerStatus> {
		console.log('Getting server status:', serverId);
		try {
			const response = await fetch(`${this.baseURL}/api/servers/${serverId}/status`);
			const data = await response.json();
			console.log('Server status response:', data);
			return data;
		} catch (error) {
			console.error('Get server status failed:', error);
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
