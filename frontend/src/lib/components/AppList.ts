import { ApiClient } from '../api/index.js';
import type { App, AppRequest, Server } from '../api/index.js';

export interface AppFormData {
	name: string;
	server_id: string;
	domain: string;
	remote_path: string;
	service_name: string;
	// Version info for first-time creation
	version_number: string;
	version_notes: string;
}

export interface AppListState {
	apps: App[];
	servers: Server[];
	loading: boolean;
	error: string | null;
	showCreateForm: boolean;
	newApp: AppFormData;
	creating: boolean;
	deleting: boolean;
	showDeleteModal: boolean;
	appToDelete: App | null;
}

export class AppListLogic {
	private state: AppListState;
	private stateUpdateCallback?: (state: AppListState) => void;
	private api: ApiClient;

	constructor() {
		this.api = new ApiClient();
		this.state = this.getInitialState();
	}

	private getInitialState(): AppListState {
		return {
			apps: [],
			servers: [],
			loading: true,
			error: null,
			showCreateForm: false,
			newApp: {
				name: '',
				server_id: '',
				domain: '',
				remote_path: '',
				service_name: '',
				version_number: '1.0.0',
				version_notes: 'Initial version'
			},
			creating: false,
			deleting: false,
			showDeleteModal: false,
			appToDelete: null
		};
	}

	public getState(): AppListState {
		return this.state;
	}

	public onStateUpdate(callback: (state: AppListState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<AppListState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public setError(error: string): void {
		this.updateState({ error });
	}

	public async initialize(): Promise<void> {
		await Promise.all([this.loadApps(), this.loadServers()]);
	}

	public async loadApps(): Promise<void> {
		try {
			this.updateState({ loading: true, error: null });
			const response = await this.api.apps.getApps();
			const apps = response.apps || [];
			this.updateState({ apps });
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to load apps';
			this.updateState({ error, apps: [] });
		} finally {
			this.updateState({ loading: false });
		}
	}

	public async loadServers(): Promise<void> {
		try {
			const response = await this.api.servers.getServers();
			const servers = response.servers || [];
			this.updateState({ servers });
		} catch (err) {
			console.error('Failed to load servers for dropdown:', err);
			this.updateState({ servers: [] });
		}
	}

	public async createApp(): Promise<boolean> {
		try {
			this.updateState({
				creating: true,
				error: null
			});

			// Step 1: Create the app
			const appData: AppRequest = {
				name: this.state.newApp.name,
				server_id: this.state.newApp.server_id,
				domain: this.state.newApp.domain,
				remote_path:
					this.state.newApp.remote_path || `/opt/pocketbase/apps/${this.state.newApp.name}`,
				service_name: this.state.newApp.service_name || `pocketbase-${this.state.newApp.name}`
			};

			const app = await this.api.apps.createApp(appData);

			// Step 2: Create initial version (optional)
			if (this.state.newApp.version_number) {
				await this.api.versions.createVersion({
					app_id: app.id,
					version_number: this.state.newApp.version_number,
					notes: this.state.newApp.version_notes
				});
			}

			// Update apps list
			const apps = [...this.state.apps, app];
			this.updateState({
				apps,
				showCreateForm: false,
				creating: false
			});
			this.resetForm();
			return true;
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to create app';
			this.updateState({
				error,
				creating: false
			});
			return false;
		}
	}

	public deleteApp(id: string): void {
		const app = this.state.apps.find((a) => a.id === id);
		if (app) {
			this.updateState({
				showDeleteModal: true,
				appToDelete: app
			});
		}
	}

	public async confirmDeleteApp(id: string): Promise<void> {
		try {
			this.updateState({ deleting: true, error: null });
			await this.api.apps.deleteApp(id);

			const apps = this.state.apps.filter((a) => a.id !== id);
			this.updateState({
				apps,
				showDeleteModal: false,
				appToDelete: null,
				deleting: false
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to delete app';
			this.updateState({ error, deleting: false });
		}
	}

	public closeDeleteModal(): void {
		this.updateState({
			showDeleteModal: false,
			appToDelete: null
		});
	}

	public resetForm(): void {
		this.updateState({
			newApp: {
				name: '',
				server_id: '',
				domain: '',
				remote_path: '',
				service_name: '',
				version_number: '1.0.0',
				version_notes: 'Initial version'
			},
			creating: false
		});
	}

	public toggleCreateForm(): void {
		this.updateState({ showCreateForm: !this.state.showCreateForm });
	}

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public updateNewApp(field: keyof AppFormData, value: string): void {
		this.updateState({
			newApp: { ...this.state.newApp, [field]: value }
		});
	}

	public getServerName(serverId: string): string {
		const server = this.state.servers.find((s) => s.id === serverId);
		return server ? server.name : 'Unknown Server';
	}

	public getAvailableServers(): Server[] {
		return this.state.servers.filter((s) => s.setup_complete);
	}

	public getAppStatusBadge(app: App): { text: string; color: string } {
		switch (app.status) {
			case 'online':
				return {
					text: 'Online',
					color: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
				};
			case 'offline':
				return {
					text: 'Offline',
					color: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
				};
			default:
				return {
					text: 'Unknown',
					color: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-200'
				};
		}
	}

	public openApp(domain: string): void {
		window.open(`https://${domain}`, '_blank');
	}

	// Helper methods for the component
	public formatTimestamp(timestamp: string): string {
		try {
			return new Date(timestamp).toLocaleString();
		} catch {
			return timestamp;
		}
	}

	public getStatusIcon(status: string): string {
		switch (status) {
			case 'online':
				return '🟢';
			case 'offline':
				return '🔴';
			default:
				return '⚪';
		}
	}
}
