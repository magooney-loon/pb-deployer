import { ApiClient, type Server, type App } from '$lib/api/index.js';
import { getServerStatusBadge, getAppStatusIcon } from '$lib/components/partials/index.js';

export interface DashboardState {
	servers: Server[];
	apps: App[];
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

			const [serversResponse, appsResponse] = await Promise.all([
				this.api.servers.getServers(),
				this.api.apps.getApps()
			]);

			const servers = serversResponse.servers || [];
			const apps = appsResponse.apps || [];

			this.updateState({
				servers,
				apps
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to load dashboard data';
			this.updateState({
				error,
				servers: [],
				apps: []
			});
		} finally {
			this.updateState({ loading: false });
		}
	}

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public getMetrics(): DashboardMetrics {
		const { servers, apps } = this.state;

		const readyServers = servers?.filter((s) => s.setup_complete) || [];

		const onlineApps = apps?.filter((a) => a.status === 'online') || [];

		const recentServers = servers?.slice(0, 3) || [];
		const recentApps = apps?.slice(0, 5) || [];

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

		return {
			totalServers: servers?.length || 0,
			readyServers,
			totalApps: apps?.length || 0,
			onlineApps,
			recentServers,
			recentApps,
			serverStatusCounts,
			appStatusCounts,
			deploymentInfo: {
				appsDeployed,
				pendingDeployment,
				averageUptime
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
}
