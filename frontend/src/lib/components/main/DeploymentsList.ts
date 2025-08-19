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
	showDeleteModal: boolean;
	deploymentToDelete: { id: string; name: string } | null;
	deleting: boolean;
	retrying: boolean;
	deploying: boolean;
	deployingIds: string[];
	showDeployModal: boolean;
	deploymentToDeploy: Deployment | null;
	autoOpenedLogsModal: boolean;
	logsPollingInterval: number | null;
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
		creating: false,
		showDeleteModal: false,
		deploymentToDelete: null,
		deleting: false,
		retrying: false,
		deploying: false,
		deployingIds: [],
		showDeployModal: false,
		deploymentToDeploy: null,
		autoOpenedLogsModal: false,
		logsPollingInterval: null
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
			deploymentToShowLogs: deployment,
			autoOpenedLogsModal: false
		});
	}

	closeLogsModal(): void {
		// Prevent closing if auto-opened and deployment is still in progress
		if (this.state.autoOpenedLogsModal && this.state.deploymentToShowLogs) {
			const deployment = this.state.deploymentToShowLogs;
			if (
				['pending', 'running'].includes(deployment.status) ||
				this.isDeploymentInProgress(deployment.id)
			) {
				return; // Don't close modal during active deployment
			}
		}

		// Stop polling when closing logs modal
		this.stopLogsPolling();

		// Start closing animation immediately
		this.updateState({ showLogsModal: false, autoOpenedLogsModal: false });

		// Clear deployment data after animation completes
		setTimeout(() => {
			this.updateState({ deploymentToShowLogs: null });
		}, 300);
	}

	openCreateModal(): void {
		this.updateState({ showCreateModal: true });
	}

	closeCreateModal(): void {
		// Start closing animation immediately
		this.updateState({ showCreateModal: false });

		// No additional data to clear for create modal after animation
	}

	async createDeployment(data: { app_id: string; version_id: string }): Promise<void> {
		this.updateState({ creating: true, error: null });

		try {
			// Check if there's already a pending deployment for this version
			if (this.hasPendingDeployment(data.app_id, data.version_id)) {
				throw new Error(
					'A deployment for this version is already pending or running. Please wait for it to complete.'
				);
			}

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
			const errorMessage =
				error instanceof Error ? error.message : 'Failed to create deployment. Please try again.';
			this.updateState({
				error: errorMessage,
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

	hasPendingDeployment(appId: string, versionId: string): boolean {
		return this.state.deployments.some(
			(deployment) =>
				deployment.app_id === appId &&
				deployment.version_id === versionId &&
				['pending', 'running'].includes(deployment.status)
		);
	}

	getAvailableVersionsForApp(appId: string): Version[] {
		return this.state.versions.filter((version) => {
			return version.app_id === appId && !this.hasPendingDeployment(appId, version.id);
		});
	}

	getPendingDeploymentInfo(appId: string, versionId: string): Deployment | null {
		return (
			this.state.deployments.find(
				(deployment) =>
					deployment.app_id === appId &&
					deployment.version_id === versionId &&
					['pending', 'running'].includes(deployment.status)
			) || null
		);
	}

	getVersionsWithPendingStatus(
		appId: string
	): Array<Version & { hasPending: boolean; pendingDeployment?: Deployment }> {
		return this.state.versions
			.filter((version) => version.app_id === appId)
			.map((version) => {
				const pendingDeployment = this.getPendingDeploymentInfo(appId, version.id);
				return {
					...version,
					hasPending: !!pendingDeployment,
					pendingDeployment: pendingDeployment || undefined
				};
			});
	}

	deleteDeployment(deployment: Deployment): void {
		const deploymentDisplay = {
			id: deployment.id,
			name: this.getDeploymentDisplayName(deployment)
		};

		this.updateState({
			showDeleteModal: true,
			deploymentToDelete: deploymentDisplay
		});
	}

	getDeploymentDisplayName(deployment: Deployment): string {
		const appName = deployment.expand?.app_id?.name || 'Unknown App';
		const versionNumber = deployment.expand?.version_id?.version_number || 'Unknown Version';
		return `${appName} - v${versionNumber}`;
	}

	closeDeleteModal(): void {
		// Start closing animation immediately
		this.updateState({ showDeleteModal: false });

		// Clear deployment data after animation completes
		setTimeout(() => {
			this.updateState({ deploymentToDelete: null });
		}, 300);
	}

	async confirmDeleteDeployment(deploymentId: string): Promise<void> {
		this.updateState({ deleting: true, error: null });

		try {
			await this.apiClient.deployments.deleteDeployment(deploymentId);

			// Reload deployments after deletion
			await this.loadDeployments();
			this.updateState({
				deleting: false,
				showDeleteModal: false,
				deploymentToDelete: null
			});
		} catch (error) {
			console.error('Failed to delete deployment:', error);
			this.updateState({
				error: 'Failed to delete deployment. Please try again.',
				deleting: false
			});
		}
	}

	async retryDeployment(deployment: Deployment): Promise<void> {
		this.openDeployModal(deployment);
	}

	async deployDeployment(
		deployment: Deployment,
		isInitialDeploy = false,
		superuserEmail?: string,
		superuserPass?: string
	): Promise<void> {
		this.updateState({
			deploying: true,
			error: null,
			deployingIds: [...this.state.deployingIds, deployment.id]
		});

		try {
			await this.apiClient.deploy.deployFromRecord(
				deployment.id,
				isInitialDeploy,
				superuserEmail,
				superuserPass
			);

			// Auto-open logs modal for the deployment
			this.updateState({
				showLogsModal: true,
				deploymentToShowLogs: deployment,
				autoOpenedLogsModal: true
			});
			this.startLogsPolling(deployment.id);

			// Reload deployments after starting deployment
			await this.loadDeployments();
			this.updateState({
				deploying: false,
				deployingIds: this.state.deployingIds.filter((id) => id !== deployment.id)
			});
		} catch (error) {
			console.error('Failed to deploy:', error);
			this.updateState({
				error: 'Failed to start deployment. Please try again.',
				deploying: false,
				deployingIds: this.state.deployingIds.filter((id) => id !== deployment.id)
			});
		}
	}

	isPendingDeployment(deployment: Deployment): boolean {
		return deployment.status === 'pending';
	}

	isDeploymentInProgress(deploymentId: string): boolean {
		return this.state.deployingIds.includes(deploymentId);
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

	openDeployModal(deployment: Deployment): void {
		this.updateState({
			showDeployModal: true,
			deploymentToDeploy: deployment
		});
	}

	closeDeployModal(): void {
		// Start closing animation immediately
		this.updateState({ showDeployModal: false });

		// Clear deployment data after animation completes
		setTimeout(() => {
			this.updateState({ deploymentToDeploy: null });
		}, 300);
	}

	async deployFromModal(
		deploymentId: string,
		isInitialDeploy: boolean,
		superuserEmail?: string,
		superuserPass?: string
	): Promise<void> {
		this.updateState({
			deploying: true,
			error: null,
			deployingIds: [...this.state.deployingIds, deploymentId]
		});

		try {
			// Get the deployment to auto-open logs modal
			const deployment = this.state.deployments.find((d) => d.id === deploymentId);

			await this.apiClient.deploy.deployFromRecord(
				deploymentId,
				isInitialDeploy,
				superuserEmail,
				superuserPass
			);

			// Auto-open logs modal for the deployment
			if (deployment) {
				this.updateState({
					showLogsModal: true,
					deploymentToShowLogs: deployment,
					autoOpenedLogsModal: true,
					showDeployModal: false,
					deploymentToDeploy: null
				});
				this.startLogsPolling(deploymentId);
			}

			// Reload deployments after starting deployment
			await this.loadDeployments();
			this.updateState({
				deploying: false,
				deployingIds: this.state.deployingIds.filter((id) => id !== deploymentId)
			});
		} catch (error) {
			console.error('Failed to deploy:', error);
			this.updateState({
				error: 'Failed to start deployment. Please try again.',
				deploying: false,
				deployingIds: this.state.deployingIds.filter((id) => id !== deploymentId)
			});
		}
	}

	getDeploymentApp(deployment: Deployment): App | undefined {
		return this.state.apps.find((app) => app.id === deployment.app_id);
	}

	getDeploymentVersion(deployment: Deployment): Version | undefined {
		return this.state.versions.find((version) => version.id === deployment.version_id);
	}

	private startLogsPolling(deploymentId: string): void {
		// Stop any existing polling
		this.stopLogsPolling();

		const intervalId = setInterval(async () => {
			try {
				// Reload the specific deployment to get updated logs
				const updatedDeployment = await this.apiClient.deployments.getDeployment(deploymentId);

				// Update the deployment in our state
				const updatedDeployments = this.state.deployments.map((d) =>
					d.id === deploymentId ? updatedDeployment : d
				);

				this.updateState({
					deployments: updatedDeployments,
					deploymentToShowLogs: updatedDeployment
				});

				// Stop polling if deployment is complete
				if (['success', 'failed'].includes(updatedDeployment.status)) {
					this.stopLogsPolling();
				}
			} catch (error) {
				console.error('Failed to poll deployment logs:', error);
				// Stop polling on error
				this.stopLogsPolling();
			}
		}, 1000);

		this.updateState({ logsPollingInterval: intervalId });
	}

	stopLogsPolling(): void {
		if (this.state.logsPollingInterval) {
			clearInterval(this.state.logsPollingInterval);
			this.updateState({ logsPollingInterval: null });
		}
	}

	isLogsModalClosable(): boolean {
		if (!this.state.autoOpenedLogsModal || !this.state.deploymentToShowLogs) {
			return true;
		}

		const deployment = this.state.deploymentToShowLogs;
		return !(
			['pending', 'running'].includes(deployment.status) ||
			this.isDeploymentInProgress(deployment.id)
		);
	}
}
