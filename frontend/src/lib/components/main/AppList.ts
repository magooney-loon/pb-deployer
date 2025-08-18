import { ApiClient } from '$lib/api/index.js';
import type { App, AppRequest, Server } from '$lib/api/index.js';
import {
	getAppStatusBadge,
	getAppStatusIcon,
	formatTimestamp,
	hasUpdateAvailable
} from '$lib/components/partials/index.js';

export interface AppFormData {
	name: string;
	server_id: string;
	domain: string;
	remote_path: string;
	service_name: string;
	// Version info for first-time creation
	version_number: string;
	version_notes: string;
	initialZip?: File;
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
	uploading: boolean;
	showUploadModal: boolean;
	appToUpload: App | null;
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
			appToDelete: null,
			uploading: false,
			showUploadModal: false,
			appToUpload: null
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
			const response = await this.api.apps.getAppsWithLatestVersions();
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
		} catch {
			this.updateState({ servers: [] });
		}
	}

	public async createApp(initialZip?: File): Promise<boolean> {
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
					notes: this.state.newApp.version_notes,
					deployment_zip: initialZip
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
		// First close the modal to start the animation
		this.updateState({
			showDeleteModal: false
		});

		// Then clear the selected item after a short delay to prevent abrupt content change
		setTimeout(() => {
			this.updateState({
				appToDelete: null
			});
		}, 200);
	}

	public openUploadModal(id: string): void {
		const app = this.state.apps.find((a) => a.id === id);
		if (app) {
			this.updateState({
				showUploadModal: true,
				appToUpload: app
			});
		}
	}

	public closeUploadModal(): void {
		this.updateState({
			showUploadModal: false
		});

		setTimeout(() => {
			this.updateState({
				appToUpload: null
			});
		}, 200);
	}

	public async uploadVersion(versionData: {
		version_number: string;
		notes: string;
		deploymentZip: File;
	}): Promise<boolean> {
		if (!this.state.appToUpload) return false;

		try {
			this.updateState({ uploading: true, error: null });

			// Check if version already exists
			const versionExists = await this.api.versions.checkVersionExists(
				this.state.appToUpload.id,
				versionData.version_number
			);

			if (versionExists) {
				throw new Error(
					`Version ${versionData.version_number} already exists for this application. Please use a different version number.`
				);
			}

			// Create version with uploaded file
			await this.api.versions.createVersion({
				app_id: this.state.appToUpload.id,
				version_number: versionData.version_number,
				notes: versionData.notes,
				deployment_zip: versionData.deploymentZip
			});

			this.updateState({
				showUploadModal: false,
				uploading: false
			});

			// Refresh apps list to get updated status
			await this.loadApps();
			return true;
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to upload version';
			this.updateState({ error, uploading: false });
			return false;
		}
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

	public getAppStatusBadge(app: App) {
		return getAppStatusBadge(app, app.latest_version);
	}

	public openApp(domain: string): void {
		window.open(`https://${domain}`, '_blank');
	}

	public formatTimestamp(timestamp: string): string {
		return formatTimestamp(timestamp);
	}

	public getStatusIcon(status: string): string {
		return getAppStatusIcon(status);
	}

	public hasUpdateAvailable(currentVersion: string, latestVersion: string): boolean {
		return hasUpdateAvailable(currentVersion, latestVersion);
	}
}
