import { ApiClient } from '$lib/api/index.js';
import type { Server, ServerRequest } from '$lib/api/index.js';
import { getServerStatusBadge } from '$lib/components/partials/index.js';

export interface ServerFormData {
	name: string;
	host: string;
	port: number;
	root_username: string;
	app_username: string;
	use_ssh_agent: boolean;
	manual_key_path: string;
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
				app_username: 'pocketbase',
				use_ssh_agent: true,
				manual_key_path: ''
			},
			creating: false,
			deleting: false,
			showDeleteModal: false,
			serverToDelete: null,
			apps: []
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
				use_ssh_agent: this.state.newServer.use_ssh_agent,
				manual_key_path: this.state.newServer.manual_key_path
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
				app_username: 'pocketbase',
				use_ssh_agent: true,
				manual_key_path: ''
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

	public async cleanup(): Promise<void> {
		// Cleanup any resources if needed
	}
}
