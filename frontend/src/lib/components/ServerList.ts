import { api, type Server, type App, type SetupStep } from './../api.js';
import type { TroubleshootResult } from './modals/TroubleshootModal.js';

export interface ServerFormData {
	name: string;
	host: string;
	port: number;
	root_username: string;
	app_username: string;
	use_ssh_agent: boolean;
	manual_key_path: string;
}

export interface TCPTestResult {
	success: boolean;
	error?: string;
	latency?: string;
}

export interface SSHTestResult {
	success: boolean;
	error?: string;
	username: string;
	auth_method?: string;
}

export interface ConnectionTestResult {
	success: boolean;
	error?: string;
	tcp_connection: TCPTestResult;
	root_ssh_connection: SSHTestResult;
	app_ssh_connection: SSHTestResult;
	overall_status: string;
}

export interface ServerListState {
	servers: Server[];
	apps: App[];
	loading: boolean;
	error: string | null;
	showCreateForm: boolean;
	testingConnection: Set<string>;
	runningSetup: Set<string>;
	applyingSecurity: Set<string>;
	troubleshooting: Set<string>;

	// Progress tracking
	setupProgress: Record<string, SetupStep[]>;
	securityProgress: Record<string, SetupStep[]>;
	setupUnsubscribers: Record<string, () => void>;
	securityUnsubscribers: Record<string, () => void>;

	// Modal states
	showConnectionModal: boolean;
	connectionTestLoading: boolean;
	connectionTestResult: ConnectionTestResult | null;
	testedServerName: string;
	showTroubleshootModal: boolean;
	troubleshootLoading: boolean;
	troubleshootResult: TroubleshootResult | null;
	troubleshootServerName: string;
	showDeleteModal: boolean;
	serverToDelete: Server | null;
	deleting: boolean;
	showSetupProgressModal: boolean;
	showSecurityProgressModal: boolean;
	currentProgressServerId: string | null;
	currentProgressServerName: string;

	// Form data
	newServer: ServerFormData;
}

export class ServerListLogic {
	private state: ServerListState;
	private stateUpdateCallback?: (state: ServerListState) => void;

	constructor() {
		this.state = this.getInitialState();
	}

	private getInitialState(): ServerListState {
		return {
			servers: [],
			apps: [],
			loading: true,
			error: null,
			showCreateForm: false,
			testingConnection: new Set(),
			runningSetup: new Set(),
			applyingSecurity: new Set(),
			troubleshooting: new Set(),
			setupProgress: {},
			securityProgress: {},
			setupUnsubscribers: {},
			securityUnsubscribers: {},
			showConnectionModal: false,
			connectionTestLoading: false,
			connectionTestResult: null,
			testedServerName: '',
			showTroubleshootModal: false,
			troubleshootLoading: false,
			troubleshootResult: null,
			troubleshootServerName: '',
			showDeleteModal: false,
			serverToDelete: null,
			deleting: false,
			showSetupProgressModal: false,
			showSecurityProgressModal: false,
			currentProgressServerId: null,
			currentProgressServerName: '',
			newServer: {
				name: '',
				host: '',
				port: 22,
				root_username: 'root',
				app_username: 'pocketbase',
				use_ssh_agent: true,
				manual_key_path: ''
			}
		};
	}

	public getState(): ServerListState {
		return this.state;
	}

	public onStateUpdate(callback: (state: ServerListState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<ServerListState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public async loadServers(): Promise<void> {
		try {
			console.log('ServerList: Starting to load servers...');
			this.updateState({ loading: true, error: null });

			const [serversResponse, appsResponse] = await Promise.all([api.getServers(), api.getApps()]);

			console.log('ServerList: API response received:', serversResponse);
			const servers = serversResponse.servers || [];
			const apps = appsResponse.apps || [];

			this.updateState({ servers, apps });
			console.log('ServerList: Servers set to:', servers);
			console.log('ServerList: Servers length:', servers.length);
		} catch (err) {
			console.error('ServerList: Error loading servers:', err);
			const error = err instanceof Error ? err.message : 'Failed to load servers';
			this.updateState({
				error,
				servers: [],
				apps: []
			});
		} finally {
			this.updateState({ loading: false });
			console.log('ServerList: Loading finished. Final servers count:', this.state.servers.length);
		}
	}

	public async createServer(): Promise<void> {
		try {
			const server = await api.createServer(this.state.newServer);
			const servers = [...this.state.servers, server];
			this.updateState({
				servers,
				showCreateForm: false
			});
			this.resetForm();
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to create server';
			this.updateState({ error });
		}
	}

	public deleteServer(id: string): void {
		const server = this.state.servers.find((s) => s.id === id);
		if (server) {
			this.updateState({
				serverToDelete: server,
				showDeleteModal: true
			});
		}
	}

	public async confirmDeleteServer(id: string): Promise<void> {
		try {
			this.updateState({ deleting: true });
			await api.deleteServer(id);
			const servers = this.state.servers.filter((s) => s.id !== id);
			this.updateState({
				servers,
				showDeleteModal: false,
				serverToDelete: null
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to delete server';
			this.updateState({ error });
		} finally {
			this.updateState({ deleting: false });
		}
	}

	public async testConnection(id: string): Promise<void> {
		const server = this.state.servers.find((s) => s.id === id);
		if (!server) return;

		// Open modal immediately with loading state
		this.updateState({
			connectionTestResult: null,
			testedServerName: server.name,
			connectionTestLoading: true,
			showConnectionModal: true
		});

		const testingConnection = new Set(this.state.testingConnection);
		testingConnection.add(id);
		this.updateState({ testingConnection });

		try {
			const result = await api.testServerConnection(id);
			this.updateState({ connectionTestResult: result });
		} catch (err) {
			const errorMsg = err instanceof Error ? err.message : 'Unknown error';
			const connectionTestResult: ConnectionTestResult = {
				success: false,
				error: errorMsg,
				tcp_connection: {
					success: false,
					error: errorMsg
				},
				root_ssh_connection: {
					success: false,
					error: errorMsg,
					username: server.root_username || 'unknown'
				},
				app_ssh_connection: {
					success: false,
					error: errorMsg,
					username: server.app_username || 'unknown'
				},
				overall_status: 'test_failed'
			};
			this.updateState({ connectionTestResult });
		} finally {
			testingConnection.delete(id);
			this.updateState({
				testingConnection,
				connectionTestLoading: false
			});
		}
	}

	public async troubleshootConnection(id: string): Promise<void> {
		const server = this.state.servers.find((s) => s.id === id);
		if (!server) return;

		// Open modal immediately with loading state
		this.updateState({
			troubleshootResult: null,
			troubleshootServerName: server.name,
			troubleshootLoading: true,
			showTroubleshootModal: true
		});

		const troubleshooting = new Set(this.state.troubleshooting);
		troubleshooting.add(id);
		this.updateState({ troubleshooting });

		try {
			const result = await api.troubleshootServer(id);
			this.updateState({ troubleshootResult: result });
		} catch (err) {
			const errorMsg = err instanceof Error ? err.message : 'Unknown error';
			const troubleshootResult: TroubleshootResult = {
				success: false,
				server_id: id,
				server_name: server.name,
				host: server.host,
				port: server.port,
				timestamp: new Date().toISOString(),
				diagnostics: [],
				summary: `Failed to run diagnostics: ${errorMsg}`,
				has_errors: true,
				has_warnings: false,
				error_count: 1,
				warning_count: 0,
				success_count: 0,
				suggestions: ['Check network connectivity', 'Try again later']
			};
			this.updateState({ troubleshootResult });
		} finally {
			troubleshooting.delete(id);
			this.updateState({
				troubleshooting,
				troubleshootLoading: false
			});
		}
	}

	public async retryTroubleshoot(): Promise<void> {
		if (!this.state.troubleshootResult) return;
		await this.troubleshootConnection(this.state.troubleshootResult.server_id);
	}

	public async quickTroubleshoot(): Promise<void> {
		if (!this.state.troubleshootResult) return;

		this.updateState({ troubleshootLoading: true });

		try {
			const result = await api.quickTroubleshootServer(this.state.troubleshootResult.server_id);
			// Convert quick result to full result format for display
			const quickDiagnostic = {
				step: 'quick_connectivity',
				status: result.status,
				message: result.message,
				suggestion: result.suggestion
			};

			const updatedResult: TroubleshootResult = {
				...this.state.troubleshootResult,
				diagnostics: [quickDiagnostic],
				success: result.success,
				has_errors: result.status === 'error',
				has_warnings: result.status === 'warning',
				error_count: result.status === 'error' ? 1 : 0,
				warning_count: result.status === 'warning' ? 1 : 0,
				success_count: result.status === 'success' ? 1 : 0,
				suggestions: result.suggestion ? [result.suggestion] : [],
				timestamp: result.timestamp
			};

			this.updateState({ troubleshootResult: updatedResult });
		} catch (err) {
			console.error('Quick troubleshoot failed:', err);
		} finally {
			this.updateState({ troubleshootLoading: false });
		}
	}

	public closeTroubleshootModal(): void {
		this.updateState({
			showTroubleshootModal: false,
			troubleshootResult: null,
			troubleshootServerName: '',
			troubleshootLoading: false
		});
	}

	public async runSetup(id: string): Promise<void> {
		try {
			const server = this.state.servers.find((s) => s.id === id);
			if (!server) return;

			const runningSetup = new Set(this.state.runningSetup);
			runningSetup.add(id);

			const setupProgress = { ...this.state.setupProgress };
			setupProgress[id] = [];

			this.updateState({
				runningSetup,
				setupProgress,
				currentProgressServerId: id,
				currentProgressServerName: server.name,
				showSetupProgressModal: true
			});

			console.log('Setup progress modal opened for server:', server.name);

			// Subscribe to setup progress
			const unsubscribe = await api.subscribeToSetupProgress(id, (step: SetupStep) => {
				console.log('Setup progress received for server', id, ':', step);
				const currentProgress = this.state.setupProgress[id] || [];
				const updatedProgress = [...currentProgress, step];
				const newSetupProgress = { ...this.state.setupProgress };
				newSetupProgress[id] = updatedProgress;

				this.updateState({ setupProgress: newSetupProgress });
				console.log('Setup progress updated:', updatedProgress.length, 'steps');

				// If setup is complete, remove from running state and refresh servers
				if (step.step === 'complete' || step.status === 'failed') {
					const runningSetup = new Set(this.state.runningSetup);
					runningSetup.delete(id);
					this.updateState({ runningSetup });
					this.loadServers(); // Refresh the list
				}
			});

			const setupUnsubscribers = { ...this.state.setupUnsubscribers };
			setupUnsubscribers[id] = unsubscribe;
			this.updateState({ setupUnsubscribers });

			// Start the setup process
			await api.runServerSetup(id);
		} catch (err) {
			alert(`Setup failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
			this.cleanupSetupProgress(id);
			this.updateState({ showSetupProgressModal: false });
		}
	}

	public async applySecurity(id: string): Promise<void> {
		if (!confirm('This will apply security lockdown to the server. Continue?')) return;

		try {
			const server = this.state.servers.find((s) => s.id === id);
			if (!server) return;

			const applyingSecurity = new Set(this.state.applyingSecurity);
			applyingSecurity.add(id);

			const securityProgress = { ...this.state.securityProgress };
			securityProgress[id] = [];

			this.updateState({
				applyingSecurity,
				securityProgress,
				currentProgressServerId: id,
				currentProgressServerName: server.name,
				showSecurityProgressModal: true
			});

			console.log('Security progress modal opened for server:', server.name);

			// Subscribe to security progress
			const unsubscribe = await api.subscribeToSecurityProgress(id, (step: SetupStep) => {
				console.log('Security progress received for server', id, ':', step);
				const currentProgress = this.state.securityProgress[id] || [];
				const updatedProgress = [...currentProgress, step];
				const newSecurityProgress = { ...this.state.securityProgress };
				newSecurityProgress[id] = updatedProgress;

				this.updateState({ securityProgress: newSecurityProgress });
				console.log('Security progress updated:', updatedProgress.length, 'steps');

				// If security is complete, remove from running state and refresh servers
				if (step.step === 'complete' || step.status === 'failed') {
					const applyingSecurity = new Set(this.state.applyingSecurity);
					applyingSecurity.delete(id);
					this.updateState({ applyingSecurity });
					this.loadServers(); // Refresh the list
				}
			});

			const securityUnsubscribers = { ...this.state.securityUnsubscribers };
			securityUnsubscribers[id] = unsubscribe;
			this.updateState({ securityUnsubscribers });

			// Start the security lockdown process
			await api.applySecurityLockdown(id);
		} catch (err) {
			alert(`Security lockdown failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
			this.cleanupSecurityProgress(id);
			this.updateState({ showSecurityProgressModal: false });
		}
	}

	private cleanupSetupProgress(id: string): void {
		const runningSetup = new Set(this.state.runningSetup);
		runningSetup.delete(id);

		const setupProgress = { ...this.state.setupProgress };
		delete setupProgress[id];

		const setupUnsubscribers = { ...this.state.setupUnsubscribers };
		if (setupUnsubscribers[id]) {
			setupUnsubscribers[id]();
			delete setupUnsubscribers[id];
		}

		this.updateState({ runningSetup, setupProgress, setupUnsubscribers });
	}

	private cleanupSecurityProgress(id: string): void {
		const applyingSecurity = new Set(this.state.applyingSecurity);
		applyingSecurity.delete(id);

		const securityProgress = { ...this.state.securityProgress };
		delete securityProgress[id];

		const securityUnsubscribers = { ...this.state.securityUnsubscribers };
		if (securityUnsubscribers[id]) {
			securityUnsubscribers[id]();
			delete securityUnsubscribers[id];
		}

		this.updateState({ applyingSecurity, securityProgress, securityUnsubscribers });
	}

	public closeSetupProgressModal(): void {
		// Check if operation is still running (not failed or complete)
		const { currentProgressServerId } = this.state;
		if (currentProgressServerId && this.state.runningSetup.has(currentProgressServerId)) {
			const currentProgress = this.state.setupProgress[currentProgressServerId] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				// Only prevent closing if operation is still running (not failed or complete)
				if (latestStep.status === 'running' && latestStep.step !== 'complete') {
					return; // Don't close
				}
			} else if (!this.isFailed()) {
				return; // Don't close if no progress yet and not failed
			}
		}

		// Clean up state when closing - now safe to clean up progress data
		if (currentProgressServerId) {
			this.cleanupSetupProgress(currentProgressServerId);
		}

		this.updateState({
			showSetupProgressModal: false,
			currentProgressServerId: null
		});
	}

	public closeSecurityProgressModal(): void {
		// Check if operation is still running (not failed or complete)
		const { currentProgressServerId } = this.state;
		if (currentProgressServerId && this.state.applyingSecurity.has(currentProgressServerId)) {
			const currentProgress = this.state.securityProgress[currentProgressServerId] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				// Only prevent closing if operation is still running (not failed or complete)
				if (latestStep.status === 'running' && latestStep.step !== 'complete') {
					return; // Don't close
				}
			} else if (!this.isFailed()) {
				return; // Don't close if no progress yet and not failed
			}
		}

		// Clean up state when closing - now safe to clean up progress data
		if (currentProgressServerId) {
			this.cleanupSecurityProgress(currentProgressServerId);
		}

		this.updateState({
			showSecurityProgressModal: false,
			currentProgressServerId: null
		});
	}

	public resetForm(): void {
		this.updateState({
			newServer: {
				name: '',
				host: '',
				port: 22,
				root_username: 'root',
				app_username: 'pocketbase',
				use_ssh_agent: true,
				manual_key_path: ''
			}
		});
	}

	public toggleCreateForm(): void {
		this.updateState({ showCreateForm: !this.state.showCreateForm });
	}

	public closeConnectionModal(): void {
		this.updateState({ showConnectionModal: false });
	}

	public closeDeleteModal(): void {
		if (!this.state.deleting) {
			this.updateState({
				showDeleteModal: false,
				serverToDelete: null
			});
		}
	}

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public updateNewServer(field: keyof ServerFormData, value: string | number | boolean): void {
		this.updateState({
			newServer: { ...this.state.newServer, [field]: value }
		});
	}

	public getServerStatusBadge(server: Server): { text: string; color: string } {
		// Check if setup is in progress
		if (this.state.runningSetup.has(server.id)) {
			const currentProgress = this.state.setupProgress[server.id] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				if (latestStep.status === 'failed') {
					return { text: 'Setup Failed', color: 'bg-red-100 text-red-800' };
				}
			}
			return { text: 'Setting Up...', color: 'bg-blue-100 text-blue-800' };
		}

		// Check if security is in progress
		if (this.state.applyingSecurity.has(server.id)) {
			const currentProgress = this.state.securityProgress[server.id] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				if (latestStep.status === 'failed') {
					return { text: 'Security Failed', color: 'bg-red-100 text-red-800' };
				}
			}
			return { text: 'Securing...', color: 'bg-purple-100 text-purple-800' };
		}

		if (!server.setup_complete) {
			return { text: 'Not Setup', color: 'bg-red-100 text-red-800' };
		} else if (server.security_locked) {
			return { text: 'Ready + Secured', color: 'bg-green-100 text-green-800' };
		} else {
			return { text: 'Ready', color: 'bg-green-100 text-green-800' };
		}
	}

	public getProgressStepIcon(status: string): string {
		switch (status) {
			case 'running':
				return '🔄';
			case 'success':
				return '✅';
			case 'failed':
				return '❌';
			default:
				return '⏳';
		}
	}

	public isFailed(): boolean {
		const { currentProgressServerId } = this.state;
		if (!currentProgressServerId) return false;

		// Check setup progress
		if (this.state.runningSetup.has(currentProgressServerId)) {
			const currentProgress = this.state.setupProgress[currentProgressServerId] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				return latestStep.status === 'failed';
			}
		}

		// Check security progress
		if (this.state.applyingSecurity.has(currentProgressServerId)) {
			const currentProgress = this.state.securityProgress[currentProgressServerId] || [];
			if (currentProgress.length > 0) {
				const latestStep = currentProgress[currentProgress.length - 1];
				return latestStep.status === 'failed';
			}
		}

		return false;
	}

	public isSetupInProgress(serverId: string | null): boolean {
		if (!serverId) return false;

		// Check if setup is marked as running
		if (!this.state.runningSetup.has(serverId)) return false;

		// Check actual progress to see if it's really still running
		const currentProgress = this.state.setupProgress[serverId] || [];
		if (currentProgress.length === 0) return true; // Just started, no progress yet

		const latestStep = currentProgress[currentProgress.length - 1];

		// If any step has failed or operation is complete, not in progress anymore
		if (latestStep.status === 'failed' || latestStep.step === 'complete') {
			return false;
		}

		// Check if any previous step failed (not just the latest)
		const hasFailedStep = currentProgress.some((step) => step.status === 'failed');
		if (hasFailedStep) {
			return false;
		}

		return true;
	}

	public isSecurityInProgress(serverId: string | null): boolean {
		if (!serverId) return false;

		// Check if security is marked as running
		if (!this.state.applyingSecurity.has(serverId)) return false;

		// Check actual progress to see if it's really still running
		const currentProgress = this.state.securityProgress[serverId] || [];
		if (currentProgress.length === 0) return true; // Just started, no progress yet

		const latestStep = currentProgress[currentProgress.length - 1];

		// If any step has failed or operation is complete, not in progress anymore
		if (latestStep.status === 'failed' || latestStep.step === 'complete') {
			return false;
		}

		// Check if any previous step failed (not just the latest)
		const hasFailedStep = currentProgress.some((step) => step.status === 'failed');
		if (hasFailedStep) {
			return false;
		}

		return true;
	}

	public async cleanup(): Promise<void> {
		// Clean up all subscriptions
		for (const unsubscribe of Object.values(this.state.setupUnsubscribers)) {
			unsubscribe();
		}
		for (const unsubscribe of Object.values(this.state.securityUnsubscribers)) {
			unsubscribe();
		}
		await api.unsubscribeFromAll();
	}
}
