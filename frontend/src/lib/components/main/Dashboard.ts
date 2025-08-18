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
		averageUptime: number;
	};
	updateInfo: {
		appsWithUpdates: number;
		appsNeedingUpdates: App[];
		totalUpdatesAvailable: number;
	};
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

			this.updateState({
				servers,
				apps,
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

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public getMetrics(): DashboardMetrics {
		const { servers, apps, deployments } = this.state;

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

		const appsDeployed = apps?.filter((a) => a.current_version).length || 0;
		const pendingDeployment = apps?.filter((a) => !a.current_version).length || 0;
		const averageUptime =
			onlineApps.length > 0 && (apps?.length || 0) > 0
				? Math.round((onlineApps.length / (apps?.length || 1)) * 100)
				: 0;

		const appsNeedingUpdates =
			apps?.filter(
				(app) =>
					app.latest_version &&
					app.current_version &&
					hasUpdateAvailable(app.current_version, app.latest_version)
			) || [];

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
				appsDeployed,
				pendingDeployment,
				averageUptime
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
