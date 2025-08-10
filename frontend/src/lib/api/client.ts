import { BaseClient } from './base.js';
import { ServerClient } from './servers.js';
import { AppClient } from './apps.js';
import { RealtimeClient } from './realtime.js';
import type {
	ServerRequest,
	AppRequest,
	Server,
	App,
	ServerResponse,
	AppResponse,
	ServerStatus,
	HealthCheckResponse,
	SetupStep
} from './types.js';

export class ApiClient extends BaseClient {
	private serverClient: ServerClient;
	private appClient: AppClient;
	private realtimeClient: RealtimeClient;

	constructor(baseUrl: string = 'http://localhost:8090') {
		super(baseUrl);

		// Initialize specialized clients with the same base URL
		this.serverClient = new ServerClient(baseUrl);
		this.appClient = new AppClient(baseUrl);
		this.realtimeClient = new RealtimeClient(baseUrl);
	}

	// Server methods - delegate to ServerClient
	async getServers() {
		return this.serverClient.getServers();
	}

	async getServer(id: string): Promise<ServerResponse> {
		return this.serverClient.getServer(id);
	}

	async createServer(data: ServerRequest): Promise<Server> {
		return this.serverClient.createServer(data);
	}

	async updateServer(id: string, data: Partial<ServerRequest>): Promise<Server> {
		return this.serverClient.updateServer(id, data);
	}

	async deleteServer(id: string) {
		return this.serverClient.deleteServer(id);
	}

	async testServerConnection(id: string) {
		return this.serverClient.testServerConnection(id);
	}

	async runServerSetup(id: string) {
		return this.serverClient.runServerSetup(id);
	}

	async applySecurityLockdown(id: string) {
		return this.serverClient.applySecurityLockdown(id);
	}

	async getServerStatus(id: string): Promise<ServerStatus> {
		return this.serverClient.getServerStatus(id);
	}

	// App methods - delegate to AppClient
	async getApps() {
		return this.appClient.getApps();
	}

	async getApp(id: string): Promise<AppResponse> {
		return this.appClient.getApp(id);
	}

	async createApp(data: AppRequest): Promise<App> {
		return this.appClient.createApp(data);
	}

	async updateApp(id: string, data: Partial<AppRequest>): Promise<App> {
		return this.appClient.updateApp(id, data);
	}

	async deleteApp(id: string) {
		return this.appClient.deleteApp(id);
	}

	async getAppsByServer(serverId: string) {
		return this.appClient.getAppsByServer(serverId);
	}

	async checkAppHealth(id: string): Promise<HealthCheckResponse> {
		return this.appClient.checkAppHealth(id);
	}

	async runAppHealthCheck(id: string): Promise<HealthCheckResponse> {
		return this.appClient.runAppHealthCheck(id);
	}

	async getAppVersions(id: string) {
		return this.appClient.getAppVersions(id);
	}

	async getAppDeployments(id: string) {
		return this.appClient.getAppDeployments(id);
	}

	// Realtime methods - delegate to RealtimeClient
	async subscribeToSetupProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		return this.realtimeClient.subscribeToSetupProgress(serverId, callback);
	}

	async subscribeToSecurityProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		return this.realtimeClient.subscribeToSecurityProgress(serverId, callback);
	}

	async unsubscribeFromAll(): Promise<void> {
		return this.realtimeClient.unsubscribeFromAll();
	}
}
