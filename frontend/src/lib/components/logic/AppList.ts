import { api, type App, type Server, getStatusIcon, formatTimestamp } from '../../api.js';

export interface AppFormData {
	name: string;
	server_id: string;
	domain: string;
	remote_path: string;
	service_name: string;
	// Version info for first-time creation
	version_number: string;
	version_notes: string;
	// File uploads
	pocketbase_binary: File | null;
	pb_public_folder: File[] | null;
}

export interface AppListState {
	apps: App[];
	servers: Server[];
	loading: boolean;
	error: string | null;
	showCreateForm: boolean;
	checkingHealth: Set<string>;
	newApp: AppFormData;
	creating: boolean;
	uploadProgress: number;
	currentStep: string;
}

export class AppListLogic {
	private state: AppListState;
	private stateUpdateCallback?: (state: AppListState) => void;

	constructor() {
		this.state = this.getInitialState();
	}

	private getInitialState(): AppListState {
		return {
			apps: [],
			servers: [],
			loading: true,
			error: null,
			showCreateForm: false,
			checkingHealth: new Set(),
			newApp: {
				name: '',
				server_id: '',
				domain: '',
				remote_path: '',
				service_name: '',
				version_number: '1.0.0',
				version_notes: 'Initial version',
				pocketbase_binary: null,
				pb_public_folder: null
			},
			creating: false,
			uploadProgress: 0,
			currentStep: ''
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
			const response = await api.getApps();
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
			const response = await api.getServers();
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
				error: null,
				currentStep: 'Creating app...',
				uploadProgress: 0
			});

			// Step 1: Create the app
			const appData = {
				name: this.state.newApp.name,
				server_id: this.state.newApp.server_id,
				domain: this.state.newApp.domain,
				remote_path:
					this.state.newApp.remote_path || `/opt/pocketbase/apps/${this.state.newApp.name}`,
				service_name: this.state.newApp.service_name || `pocketbase-${this.state.newApp.name}`
			};

			const app = await api.createApp(appData);
			this.updateState({
				currentStep: 'Creating initial version...',
				uploadProgress: 25
			});

			// Step 2: Create initial version
			const version = await api.createVersion(app.id, {
				version_number: this.state.newApp.version_number,
				notes: this.state.newApp.version_notes
			});

			this.updateState({
				currentStep: 'Uploading files...',
				uploadProgress: 50
			});

			// Step 3: Upload files
			await api.uploadVersionWithFolder(
				version.id,
				this.state.newApp.pocketbase_binary!,
				this.state.newApp.pb_public_folder!
			);

			this.updateState({
				currentStep: 'Finalizing...',
				uploadProgress: 100
			});

			// Update apps list
			const apps = [...this.state.apps, app];
			this.updateState({
				apps,
				showCreateForm: false,
				creating: false,
				currentStep: '',
				uploadProgress: 0
			});
			this.resetForm();
			return true;
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to create app';
			this.updateState({
				error,
				creating: false,
				currentStep: '',
				uploadProgress: 0
			});
			return false;
		}
	}

	public async deleteApp(id: string): Promise<void> {
		if (!confirm('Are you sure you want to delete this app?')) return;

		try {
			await api.deleteApp(id);
			const apps = this.state.apps.filter((a) => a.id !== id);
			this.updateState({ apps });
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to delete app';
			this.updateState({ error });
		}
	}

	public async checkHealth(id: string): Promise<void> {
		try {
			const checkingHealth = new Set(this.state.checkingHealth);
			checkingHealth.add(id);
			this.updateState({ checkingHealth });

			await api.runAppHealthCheck(id);
			setTimeout(async () => {
				await this.loadApps(); // Refresh to get updated status
			}, 2000);
		} catch (err) {
			alert(`Health check failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			const checkingHealth = new Set(this.state.checkingHealth);
			checkingHealth.delete(id);
			this.updateState({ checkingHealth });
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
				version_notes: 'Initial version',
				pocketbase_binary: null,
				pb_public_folder: null
			},
			creating: false,
			uploadProgress: 0,
			currentStep: ''
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
				return { text: 'Online', color: 'bg-green-100 text-green-800' };
			case 'offline':
				return { text: 'Offline', color: 'bg-red-100 text-red-800' };
			default:
				return { text: 'Unknown', color: 'bg-gray-100 text-gray-800' };
		}
	}

	public openApp(domain: string): void {
		window.open(`https://${domain}`, '_blank');
	}

	// Helper methods for the component
	public formatTimestamp(timestamp: string): string {
		return formatTimestamp(timestamp);
	}

	public getStatusIcon(status: string): string {
		return getStatusIcon(status);
	}

	// File handling methods
	public updateBinaryFile(file: File | File[] | null): void {
		const singleFile = Array.isArray(file) ? file[0] : file;
		this.updateState({
			newApp: { ...this.state.newApp, pocketbase_binary: singleFile }
		});
	}

	public updatePublicFolder(files: File | File[] | null): void {
		const fileArray = Array.isArray(files) ? files : files ? [files] : null;
		this.updateState({
			newApp: { ...this.state.newApp, pb_public_folder: fileArray }
		});
	}

	public updateVersionNumber(version: string): void {
		this.updateState({
			newApp: { ...this.state.newApp, version_number: version }
		});
	}

	public updateVersionNotes(notes: string): void {
		this.updateState({
			newApp: { ...this.state.newApp, version_notes: notes }
		});
	}

	// App service management
	public async startApp(id: string): Promise<void> {
		try {
			await api.startApp(id);
			await this.loadApps(); // Refresh to get updated status
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to start app';
			this.updateState({ error });
		}
	}

	public async stopApp(id: string): Promise<void> {
		try {
			await api.stopApp(id);
			await this.loadApps(); // Refresh to get updated status
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to stop app';
			this.updateState({ error });
		}
	}

	public async restartApp(id: string): Promise<void> {
		try {
			await api.restartApp(id);
			await this.loadApps(); // Refresh to get updated status
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to restart app';
			this.updateState({ error });
		}
	}
}
