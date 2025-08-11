import { BaseClient } from './base.js';
import type {
	ServerRequest,
	AppRequest,
	Server,
	App,
	Version,
	ServerResponse,
	AppResponse,
	ServerStatus,
	HealthCheckResponse,
	SetupStep,
	EnhancedTroubleshootResult,
	AutoFixResult
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

	async troubleshootServer(id: string) {
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

	async quickTroubleshootServer(id: string) {
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

	// App CRUD operations using REST API
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

	async getAppVersions(id: string) {
		console.log('Getting app versions:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/versions`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App versions response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get app versions:', error);
			throw error;
		}
	}

	async getAppDeployments(id: string) {
		console.log('Getting app deployments:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/deployments?app_id=${id}`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App deployments response:', data);
			return data;
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

	// Version management
	async createVersion(
		appId: string,
		data: { version_number: string; notes?: string }
	): Promise<Version> {
		console.log('Creating version for app:', appId, data);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${appId}/versions`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					app_id: appId,
					...data
				})
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const version = await response.json();
			console.log('Version created:', version);
			return version;
		} catch (error) {
			console.error('Failed to create version:', error);
			throw error;
		}
	}

	async uploadVersionFiles(
		versionId: string,
		binaryFile: File,
		publicFiles: File[]
	): Promise<{
		message: string;
		version_id: string;
		binary_file: string;
		binary_size: number;
		public_files_count: number;
		public_total_size: number;
		deployment_file: string;
		deployment_size: number;
		uploaded_at: string;
	}> {
		console.log('Uploading version files:', versionId);
		try {
			const formData = new FormData();
			formData.append('pocketbase_binary', binaryFile);

			// Append all public files
			for (const file of publicFiles) {
				formData.append('pb_public_files', file);
			}

			const response = await fetch(`${this.baseURL}/api/versions/${versionId}/upload`, {
				method: 'POST',
				body: formData
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Version files uploaded:', data);
			return data;
		} catch (error) {
			console.error('Failed to upload version files:', error);
			throw error;
		}
	}

	// Convenience method for uploading version with folder structure
	async uploadVersionWithFolder(
		versionId: string,
		binaryFile: File,
		folderFiles: FileList | File[]
	): Promise<{
		message: string;
		version_id: string;
		binary_file: string;
		binary_size: number;
		public_files_count: number;
		public_total_size: number;
		deployment_file: string;
		deployment_size: number;
		uploaded_at: string;
	}> {
		console.log('Uploading version with folder structure:', versionId);

		// Convert FileList to Array if needed
		const publicFiles = Array.isArray(folderFiles) ? folderFiles : Array.from(folderFiles);

		// Validate that we have files
		if (publicFiles.length === 0) {
			throw new Error('No public folder files provided');
		}

		// Use the existing uploadVersionFiles method
		return await this.uploadVersionFiles(versionId, binaryFile, publicFiles);
	}

	// Helper method to validate folder structure for pb_public
	validatePublicFolderStructure(files: File[]): {
		valid: boolean;
		errors: string[];
		warnings: string[];
	} {
		const errors: string[] = [];
		const warnings: string[] = [];

		// Check for common required files
		const hasIndexHtml = files.some(
			(f) => f.webkitRelativePath?.endsWith('index.html') || f.name === 'index.html'
		);
		if (!hasIndexHtml) {
			warnings.push('No index.html found - make sure your app has a main entry point');
		}

		// Check for suspicious files that shouldn't be in public folder
		const suspiciousFiles = files.filter((f) => {
			const name = f.name.toLowerCase();
			return name.includes('.env') || name.includes('config') || name.includes('secret');
		});

		if (suspiciousFiles.length > 0) {
			warnings.push(
				`Found potentially sensitive files: ${suspiciousFiles.map((f) => f.name).join(', ')}`
			);
		}

		// Check total size
		const totalSize = files.reduce((sum, f) => sum + f.size, 0);
		if (totalSize > 50 * 1024 * 1024) {
			// 50MB
			errors.push(
				`Total folder size (${Math.round(totalSize / (1024 * 1024))}MB) exceeds 50MB limit`
			);
		}

		return {
			valid: errors.length === 0,
			errors,
			warnings
		};
	}

	// App service management
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
