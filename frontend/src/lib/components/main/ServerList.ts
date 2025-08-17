import { ApiClient } from '$lib/api/index.js';
import type { Server, ServerRequest } from '$lib/api/index.js';
import { getServerStatusBadge } from '$lib/components/partials/index.js';

export interface ServerFormData {
	name: string;
	host: string;
	port: number;
	root_username: string;
	app_username: string;
}

export interface ServerListState {
	servers: Server[];
	loading: boolean;
	error: string | null;
	successMessage: string | null;
	showCreateForm: boolean;
	newServer: ServerFormData;
	creating: boolean;
	deleting: boolean;
	showDeleteModal: boolean;
	serverToDelete: Server | null;
	apps: { id: string; name: string; domain?: string }[]; // For delete modal to show related apps
	// Setup and security operations
	setupInProgress: string | null; // Server ID being setup
	securityInProgress: string | null; // Server ID being secured
	validationInProgress: string | null; // Server ID being validated
	setupError: string | null;
	securityError: string | null;
	validationError: string | null;
}

export class ServerListLogic {
	private state: ServerListState;
	private stateUpdateCallback?: (state: ServerListState) => void;
	private api: ApiClient;

	constructor() {
		this.api = new ApiClient();
		this.state = this.getInitialState();
	}

	private getInitialState(): ServerListState {
		return {
			servers: [],
			loading: true,
			error: null,
			successMessage: null,
			showCreateForm: false,
			newServer: {
				name: '',
				host: '',
				port: 22,
				root_username: 'root',
				app_username: 'pocketbase'
			},
			creating: false,
			deleting: false,
			showDeleteModal: false,
			serverToDelete: null,
			apps: [],
			// Setup and security operations
			setupInProgress: null,
			securityInProgress: null,
			validationInProgress: null,
			setupError: null,
			securityError: null,
			validationError: null
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
			this.updateState({ loading: true, error: null });
			const response = await this.api.servers.getServers();
			const servers = response.servers || [];
			this.updateState({ servers });
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to load servers';
			this.updateState({ error, servers: [] });
		} finally {
			this.updateState({ loading: false });
		}
	}

	public async createServer(): Promise<void> {
		try {
			this.updateState({ creating: true, error: null, successMessage: null });

			const serverData: ServerRequest = {
				name: this.state.newServer.name,
				host: this.state.newServer.host,
				port: this.state.newServer.port,
				root_username: this.state.newServer.root_username,
				app_username: this.state.newServer.app_username,
				use_ssh_agent: true,
				manual_key_path: ''
			};

			const server = await this.api.servers.createServer(serverData);
			const servers = [...this.state.servers, server];

			this.updateState({
				servers,
				showCreateForm: false,
				creating: false,
				successMessage: `Server "${server.name}" created successfully!`
			});
			this.resetForm();
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to create server';
			this.updateState({ error, creating: false });
		}
	}

	public deleteServer(id: string): void {
		const server = this.state.servers.find((s) => s.id === id);
		if (server) {
			this.updateState({
				showDeleteModal: true,
				serverToDelete: server
			});
			this.loadRelatedApps(id);
		}
	}

	private async loadRelatedApps(serverId: string): Promise<void> {
		try {
			const response = await this.api.apps.getAppsByServer(serverId);
			this.updateState({ apps: response.apps || [] });
		} catch {
			this.updateState({ apps: [] });
		}
	}

	public async confirmDeleteServer(id: string): Promise<void> {
		try {
			this.updateState({ deleting: true, error: null, successMessage: null });
			const serverName = this.state.serverToDelete?.name || 'Server';
			await this.api.servers.deleteServer(id);

			const servers = this.state.servers.filter((s) => s.id !== id);
			this.updateState({
				servers,
				showDeleteModal: false,
				serverToDelete: null,
				deleting: false,
				apps: [],
				successMessage: `${serverName} deleted successfully!`
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to delete server';
			this.updateState({ error, deleting: false });
		}
	}

	public resetForm(): void {
		this.updateState({
			newServer: {
				name: '',
				host: '',
				port: 22,
				root_username: 'root',
				app_username: 'pocketbase'
			}
		});
	}

	public toggleCreateForm(): void {
		this.updateState({ showCreateForm: !this.state.showCreateForm });
	}

	public closeDeleteModal(): void {
		// First close the modal to start the animation
		this.updateState({
			showDeleteModal: false
		});

		// Then clear the selected item after a short delay to prevent abrupt content change
		setTimeout(() => {
			this.updateState({
				serverToDelete: null,
				apps: []
			});
		}, 200);
	}

	public dismissError(): void {
		this.updateState({ error: null });
	}

	public dismissSuccess(): void {
		this.updateState({ successMessage: null });
	}

	public updateNewServer(field: keyof ServerFormData, value: string | number | boolean): void {
		this.updateState({
			newServer: { ...this.state.newServer, [field]: value }
		});
	}

	public getServerStatusBadge(server: Server) {
		return getServerStatusBadge(server);
	}

	public async setupServer(serverId: string): Promise<void> {
		try {
			this.updateState({
				setupInProgress: serverId,
				setupError: null,
				validationError: null,
				error: null,
				successMessage: null
			});

			const response = await this.api.setup.setupServerFromRecord(serverId);

			// Update server in local state
			const servers = this.state.servers.map((server) =>
				server.id === serverId ? { ...server, setup_complete: true } : server
			);

			this.updateState({
				servers,
				setupInProgress: null,
				successMessage: `Server setup completed successfully! ${response.setup_info.os} (${response.setup_info.architecture})`
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to setup server';

			this.updateState({
				setupInProgress: null,
				setupError: error,
				error
			});
		}
	}

	public async secureServer(serverId: string): Promise<void> {
		try {
			this.updateState({
				securityInProgress: serverId,
				securityError: null,
				error: null,
				successMessage: null
			});

			const response = await this.api.setup.secureServerFromRecord(serverId);

			// Update server in local state
			const servers = this.state.servers.map((server) =>
				server.id === serverId ? { ...server, security_locked: true } : server
			);

			this.updateState({
				servers,
				securityInProgress: null,
				successMessage: `Server security hardening completed! ${response.applied_config.firewall_rules.length} firewall rules applied.`
			});
		} catch (err) {
			const error = err instanceof Error ? err.message : 'Failed to secure server';
			this.updateState({
				securityInProgress: null,
				securityError: error,
				error
			});
		}
	}

	public isServerSetupInProgress(serverId: string): boolean {
		return this.state.setupInProgress === serverId;
	}

	public isServerSecurityInProgress(serverId: string): boolean {
		return this.state.securityInProgress === serverId;
	}

	public canSetupServer(server: Server): boolean {
		return !server.setup_complete && !this.isServerSetupInProgress(server.id);
	}

	public canSecureServer(server: Server): boolean {
		return (
			server.setup_complete &&
			!server.security_locked &&
			!this.isServerSecurityInProgress(server.id)
		);
	}

	public dismissSetupError(): void {
		this.updateState({ setupError: null });
	}

	public dismissSecurityError(): void {
		this.updateState({ securityError: null });
	}

	public dismissValidationError(): void {
		this.updateState({ validationError: null });
	}

	public async cleanup(): Promise<void> {
		// Cleanup any resources if needed
	}
}
