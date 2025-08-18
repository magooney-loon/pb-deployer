import { ApiClient } from '$lib/api/index.js';
import type { Deployment } from '$lib/api/index.js';
import { getDeploymentStatusBadge, formatTimestamp } from '$lib/components/partials/index.js';
import type { App, Version } from '$lib/api/index.js';

export interface DeploymentsListState {
	deployments: Deployment[];
	apps: App[];
	versions: Version[];
	loading: boolean;
	error: string | null;
	showLogsModal: boolean;
	deploymentToShowLogs: Deployment | null;
	showCreateModal: boolean;
	creating: boolean;
}

export class DeploymentsListLogic {
	private apiClient: ApiClient;
	private state: DeploymentsListState = {
		deployments: [],
		apps: [],
		versions: [],
		loading: false,
		error: null,
		showLogsModal: false,
		deploymentToShowLogs: null,
		showCreateModal: false,
		creating: false
	};

	private stateUpdateCallbacks: ((state: DeploymentsListState) => void)[] = [];

	constructor() {
		this.apiClient = new ApiClient();
	}

	getState(): DeploymentsListState {
		return { ...this.state };
	}

	onStateUpdate(callback: (state: DeploymentsListState) => void): void {
		this.stateUpdateCallbacks.push(callback);
	}

	private updateState(updates: Partial<DeploymentsListState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallbacks.forEach((callback) => callback(this.getState()));
	}

	async initialize(): Promise<void> {
		await Promise.all([this.loadDeployments(), this.loadApps(), this.loadVersions()]);
	}

	async loadDeployments(): Promise<void> {
		this.updateState({ loading: true, error: null });

		try {
			const result = await this.apiClient.deployments.getDeployments();
			this.updateState({
				deployments: result.deployments,
				loading: false
			});
		} catch (error) {
			console.error('Failed to load deployments:', error);
			this.updateState({
				error: 'Failed to load deployments. Please try again.',
				loading: false
			});
		}
	}

	dismissError(): void {
		this.updateState({ error: null });
	}

	async loadApps(): Promise<void> {
		try {
			const result = await this.apiClient.apps.getApps();
			this.updateState({ apps: result.apps });
		} catch (error) {
			console.error('Failed to load apps:', error);
		}
	}

	async loadVersions(): Promise<void> {
		try {
			const result = await this.apiClient.versions.getVersions();
			this.updateState({ versions: result.versions });
		} catch (error) {
			console.error('Failed to load versions:', error);
		}
	}

	getDeploymentStatusBadge(deployment: Deployment): {
		text: string;
		variant: 'success' | 'warning' | 'error' | 'info' | 'gray' | 'update';
	} {
		return getDeploymentStatusBadge(deployment);
	}

	formatTimestamp(timestamp: string): string {
		return formatTimestamp(timestamp);
	}

	formatDuration(startedAt?: string, completedAt?: string): string | null {
		if (!startedAt || !completedAt) {
			return null;
		}

		const start = new Date(startedAt);
		const end = new Date(completedAt);
		const diff = end.getTime() - start.getTime();

		if (diff < 1000) {
			return '< 1s';
		}

		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);

		if (hours > 0) {
			return `${hours}h ${minutes % 60}m`;
		} else if (minutes > 0) {
			return `${minutes}m ${seconds % 60}s`;
		} else {
			return `${seconds}s`;
		}
	}

	getAppName(deployment: Deployment): string {
		return deployment.expand?.app_id?.name || 'Unknown App';
	}

	getAppDomain(deployment: Deployment): string {
		return deployment.expand?.app_id?.domain || '';
	}

	getVersionNumber(deployment: Deployment): string {
		return deployment.expand?.version_id?.version_number || 'N/A';
	}

	getVersionNotes(deployment: Deployment): string {
		return deployment.expand?.version_id?.notes || '';
	}

	openLogsModal(deployment: Deployment): void {
		this.updateState({
			showLogsModal: true,
			deploymentToShowLogs: deployment
		});
	}

	closeLogsModal(): void {
		this.updateState({
			showLogsModal: false,
			deploymentToShowLogs: null
		});
	}

	openCreateModal(): void {
		this.updateState({ showCreateModal: true });
	}

	closeCreateModal(): void {
		this.updateState({ showCreateModal: false });
	}

	async createDeployment(data: { app_id: string; version_id: string }): Promise<void> {
		this.updateState({ creating: true, error: null });

		try {
			await this.apiClient.deployments.createDeployment({
				app_id: data.app_id,
				version_id: data.version_id,
				status: 'pending'
			});

			// Reload deployments after creation
			await this.loadDeployments();
			this.updateState({ creating: false, showCreateModal: false });
		} catch (error) {
			console.error('Failed to create deployment:', error);
			this.updateState({
				error: 'Failed to create deployment. Please try again.',
				creating: false
			});
		}
	}

	getAvailableApps(): App[] {
		return this.state.apps.filter((app) => app.current_version); // Only apps that have been deployed
	}

	getVersionsForApp(appId: string): Version[] {
		return this.state.versions.filter((version) => version.app_id === appId);
	}

	isDeploymentComplete(deployment: Deployment): boolean {
		return deployment.status === 'success' || deployment.status === 'failed';
	}

	isDeploymentRunning(deployment: Deployment): boolean {
		return deployment.status === 'running';
	}

	getRunningDuration(deployment: Deployment): string | null {
		if (!deployment.started_at || deployment.status !== 'running') {
			return null;
		}

		const start = new Date(deployment.started_at);
		const now = new Date();
		const diff = now.getTime() - start.getTime();

		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);

		if (hours > 0) {
			return `${hours}h ${minutes % 60}m`;
		} else if (minutes > 0) {
			return `${minutes}m ${seconds % 60}s`;
		} else {
			return `${seconds}s`;
		}
	}
}
