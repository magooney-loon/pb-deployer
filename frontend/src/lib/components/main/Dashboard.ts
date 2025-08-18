import { ApiClient, type Server, type App } from '$lib/api/index.js';
import type { Deployment } from '$lib/api/deployment/types.js';
import {
	getServerStatusBadge,
	getAppStatusIcon,
	getDeploymentStatusBadge,
	getAppStatusBadge,
	hasUpdateAvailable
} from '$lib/components/partials/index.js';

export interface DashboardState {
	servers: Server[];
	apps: App[];
	deployments: Deployment[];
	loading: boolean;
	error: string | null;
}

export interface DashboardMetrics {
	totalServers: number;
	readyServers: Server[];
	totalApps: number;
	onlineApps: App[];
	recentServers: Server[];
	recentApps: App[];
	recentDeployments: Deployment[];
	serverStatusCounts: {
		ready: number;
		setupRequired: number;
		securityOptional: number;
	};
	appStatusCounts: {
		online: number;
		offline: number;
		unknown: number;
	};
	deploymentInfo: {
		appsDeployed: number;
		pendingDeployment: number;
		failedDeployments: number;
	};
	updateInfo: {
		appsWithUpdates: number;
		appsNeedingUpdates: App[];
		totalUpdatesAvailable: number;
	};
}

// Extended App interface for deployment tracking
interface AppWithDeploymentStatus extends App {
	deployed_version: string | null;
	has_pending_deployment: boolean;
}

export class DashboardLogic {
	private state: DashboardState;
	private stateUpdateCallback?: (state: DashboardState) => void;
	private api: ApiClient;

	constructor() {
		this.api = new ApiClient();
		this.state = this.getInitialState();
	}

	private getInitialState(): DashboardState {
		return {
			servers: [],
			apps: [],
			deployments: [],
			loading: true,
			error: null
		};
	}

	public getState(): DashboardState {
		return this.state;
	}

	public onStateUpdate(callback: (state: DashboardState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<DashboardState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public async loadData(): Promise<void> {
		try {
			this.updateState({ loading: true, error: null });

			const [serversResponse, appsResponse, deploymentsResponse] = await Promise.all([
				this.api.servers.getServers(),
				this.api.apps.getAppsWithLatestVersions(),
				this.api.deployments.getDeployments()
			]);

			const servers = serversResponse.servers || [];
			const apps = appsResponse.apps || [];
			const deployments = deploymentsResponse.deployments || [];

			// Enhance apps with actual deployment status
			const enhancedApps = await this.enhanceAppsWithDeploymentStatus(apps, deployments);

			this.updateState({
				servers,
				apps: enhancedApps,
				deployments
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to load dashboard data';
			this.updateState({
				error,
				servers: [],
				apps: [],
				deployments: []
			});
		} finally {
			this.updateState({ loading: false });
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

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public getMetrics(): DashboardMetrics {
		const { servers, apps, deployments } = this.state;
		const enhancedApps = apps as AppWithDeploymentStatus[];

		const readyServers = servers?.filter((s) => s.setup_complete) || [];
		const onlineApps = apps?.filter((a) => a.status === 'online') || [];

		const recentServers = servers?.slice(0, 3) || [];
		const recentApps = apps?.slice(0, 5) || [];
		const recentDeployments = deployments?.slice(0, 3) || [];

		const serverStatusCounts = {
			ready: readyServers.length,
			setupRequired: servers?.filter((s) => !s.setup_complete).length || 0,
			securityOptional: servers?.filter((s) => s.setup_complete && !s.security_locked).length || 0
		};

		const appStatusCounts = {
			online: onlineApps.length,
			offline: apps?.filter((a) => a.status === 'offline').length || 0,
			unknown: apps?.filter((a) => a.status !== 'online' && a.status !== 'offline').length || 0
		};

		const failedDeployments = deployments?.filter((d) => d.status === 'failed').length || 0;

		// Use deployed_version instead of current_version for update checking
		const appsNeedingUpdates =
			enhancedApps?.filter((app) => {
				return (
					app.latest_version &&
					app.deployed_version &&
					hasUpdateAvailable(app.deployed_version, app.latest_version)
				);
			}) || [];

		return {
			totalServers: servers?.length || 0,
			readyServers,
			totalApps: apps?.length || 0,
			onlineApps,
			recentServers,
			recentApps,
			recentDeployments,
			serverStatusCounts,
			appStatusCounts,
			deploymentInfo: {
				appsDeployed: enhancedApps?.filter((app) => app.deployed_version).length || 0,
				pendingDeployment: enhancedApps?.filter((app) => app.has_pending_deployment).length || 0,
				failedDeployments: failedDeployments
			},
			updateInfo: {
				appsWithUpdates: appsNeedingUpdates.length,
				appsNeedingUpdates,
				totalUpdatesAvailable: appsNeedingUpdates.length
			}
		};
	}

	public getStatusIcon(status: string): string {
		return getAppStatusIcon(status);
	}

	public getServerStatusBadge(server: Server) {
		return getServerStatusBadge(server);
	}

	public hasData(): boolean {
		return (this.state.servers?.length || 0) > 0 || (this.state.apps?.length || 0) > 0;
	}

	public getDeploymentStatusBadge(deployment: Deployment) {
		return getDeploymentStatusBadge(deployment);
	}

	public getAppStatusBadge(app: App) {
		return getAppStatusBadge(app, app.latest_version);
	}

	public hasUpdateAvailable(currentVersion: string, latestVersion: string): boolean {
		return hasUpdateAvailable(currentVersion, latestVersion);
	}
}
