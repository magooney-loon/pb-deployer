import { ApiClient, type Server, type App, getStatusIcon } from '../api/index.js';

export interface DashboardState {
	servers: Server[];
	apps: App[];
	loading: boolean;
	error: string | null;
	refreshCounter: number;
	nextRefreshIn: number;
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
	private refreshInterval?: number;
	private countdownInterval?: number;
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
			error: null,
			refreshCounter: 0,
			nextRefreshIn: 30
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
				this.api.getServers(),
				this.api.getApps()
			]);

			const servers = serversResponse.servers || [];
			const apps = appsResponse.apps || [];

			this.updateState({
				servers,
				apps,
				refreshCounter: this.state.refreshCounter + 1,
				nextRefreshIn: 30
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

	public startAutoRefresh(): void {
		this.stopAutoRefresh(); // Clear any existing intervals

		// Start countdown timer (updates every second)
		this.countdownInterval = setInterval(() => {
			const nextRefreshIn = this.state.nextRefreshIn - 1;
			if (nextRefreshIn <= 0) {
				this.updateState({ nextRefreshIn: 30 });
			} else {
				this.updateState({ nextRefreshIn });
			}
		}, 1000);

		// Start refresh timer (every 30 seconds)
		this.refreshInterval = setInterval(async () => {
			await this.loadData();
		}, 30000);
	}

	public stopAutoRefresh(): void {
		if (this.refreshInterval) {
			clearInterval(this.refreshInterval);
			this.refreshInterval = undefined;
		}
		if (this.countdownInterval) {
			clearInterval(this.countdownInterval);
			this.countdownInterval = undefined;
		}
	}

	public destroy(): void {
		this.stopAutoRefresh();
	}

	public getMetrics(): DashboardMetrics {
		const { servers, apps } = this.state;

		// Calculate ready servers
		const readyServers = servers?.filter((s) => s.setup_complete) || [];

		// Calculate online apps
		const onlineApps = apps?.filter((a) => a.status === 'online') || [];

		// Get recent items (limited)
		const recentServers = servers?.slice(0, 3) || [];
		const recentApps = apps?.slice(0, 5) || [];

		// Server status counts
		const serverStatusCounts = {
			ready: readyServers.length,
			setupRequired: servers?.filter((s) => !s.setup_complete).length || 0,
			securityOptional: servers?.filter((s) => s.setup_complete && !s.security_locked).length || 0
		};

		// App status counts
		const appStatusCounts = {
			online: onlineApps.length,
			offline: apps?.filter((a) => a.status === 'offline').length || 0,
			unknown: apps?.filter((a) => a.status !== 'online' && a.status !== 'offline').length || 0
		};

		// Deployment info
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

	// Helper methods for the component
	public getStatusIcon(status: string): string {
		return getStatusIcon(status);
	}

	public hasData(): boolean {
		return (this.state.servers?.length || 0) > 0 || (this.state.apps?.length || 0) > 0;
	}
}
