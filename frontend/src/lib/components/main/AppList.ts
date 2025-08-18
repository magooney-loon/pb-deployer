import { ApiClient } from '$lib/api/index.js';
import type { App, AppRequest, Server } from '$lib/api/index.js';
import type { Deployment } from '$lib/api/deployment/types.js';
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

// Extended App interface for deployment tracking
interface AppWithDeploymentStatus extends App {
	deployed_version: string | null;
	has_pending_deployment: boolean;
}

export interface AppListState {
	apps: AppWithDeploymentStatus[];
	servers: Server[];
	loading: boolean;
	error: string | null;
	showCreateForm: boolean;
	newApp: AppFormData;
	creating: boolean;
	deleting: boolean;
	showDeleteModal: boolean;
	appToDelete: AppWithDeploymentStatus | null;
	showManageModal: boolean;
	appToManage: AppWithDeploymentStatus | null;
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
			showManageModal: false,
			appToManage: null
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
			const [appsResponse, deploymentsResponse] = await Promise.all([
				this.api.apps.getAppsWithLatestVersions(),
				this.api.deployments.getDeployments()
			]);

			const apps = appsResponse.apps || [];
			const deployments = deploymentsResponse.deployments || [];

			// Enhance apps with actual deployment status
			const enhancedApps = await this.enhanceAppsWithDeploymentStatus(apps, deployments);

			this.updateState({ apps: enhancedApps });
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

	private async enhanceAppsWithDeploymentStatus(
		apps: App[],
		deployments: Deployment[]
	): Promise<AppWithDeploymentStatus[]> {
		const enhancedApps = await Promise.all(
			apps.map(async (app) => {
				try {
					// Get versions for this app
					const versionsResponse = await this.api.versions.getAppVersions(app.id);
					const versions = versionsResponse.versions || [];

					// Find the latest successful deployment for this app
					const appDeployments = deployments.filter((d) => d.app_id === app.id);
					const latestSuccessfulDeployment = appDeployments
						.filter((d) => d.status === 'success')
						.sort(
							(a, b) =>
								new Date(b.completed_at || b.created).getTime() -
								new Date(a.completed_at || a.created).getTime()
						)[0];

					let deployedVersion = null;
					if (latestSuccessfulDeployment) {
						const deployedVersionObj = versions.find(
							(v) => v.id === latestSuccessfulDeployment.version_id
						);
						deployedVersion = deployedVersionObj?.version_number || null;
					}

					// Check if there are pending deployments
					const hasPendingDeployment = appDeployments.some((d) =>
						['pending', 'running'].includes(d.status)
					);

					return {
						...app,
						deployed_version: deployedVersion,
						has_pending_deployment: hasPendingDeployment
					} as AppWithDeploymentStatus;
				} catch (error) {
					console.warn('Failed to enhance app deployment status:', error);
					return {
						...app,
						deployed_version: app.current_version,
						has_pending_deployment: false
					} as AppWithDeploymentStatus;
				}
			})
		);

		return enhancedApps;
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

			// Reload apps list to get proper deployment status
			await this.loadApps();
			this.updateState({
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

	public openManageModal(id: string): void {
		const app = this.state.apps.find((a) => a.id === id);
		if (app) {
			this.updateState({
				showManageModal: true,
				appToManage: app
			});
		}
	}

	public closeManageModal(): void {
		this.updateState({
			showManageModal: false
		});

		setTimeout(() => {
			this.updateState({
				appToManage: null
			});
		}, 200);
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

	public getDeployedVersion(app: AppWithDeploymentStatus): string {
		return app.deployed_version || 'Not deployed';
	}

	public hasPendingDeployment(app: AppWithDeploymentStatus): boolean {
		return app.has_pending_deployment;
	}
}
