import { BaseClient } from './base.js';
import { ServerClient } from './client/server.js';
import { AppsClient } from './client/apps.js';
import { VersionClient } from './client/version.js';
import { DeploymentClient } from './client/deployment.js';
import type {
	ServerRequest,
	AppRequest,
	Server,
	App,
	Version,
	ServerResponse,
	AppResponse,
	ServerStatus,
	HealthCheckResponse,
	SetupStep,
	TroubleshootResult,
	QuickTroubleshootResult,
	EnhancedTroubleshootResult,
	AutoFixResult,
	Deployment
} from './types.js';

export class ApiClient extends BaseClient {
	private serverClient: ServerClient;
	private appsClient: AppsClient;
	private versionClient: VersionClient;
	private deploymentClient: DeploymentClient;

	constructor(baseUrl: string = 'http://localhost:8090') {
		super(baseUrl);
		this.serverClient = new ServerClient(baseUrl);
		this.appsClient = new AppsClient(baseUrl);
		this.versionClient = new VersionClient(baseUrl);
		this.deploymentClient = new DeploymentClient(baseUrl);
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

	async troubleshootServer(id: string): Promise<TroubleshootResult> {
		return this.serverClient.troubleshootServer(id);
	}

	async quickTroubleshootServer(id: string): Promise<QuickTroubleshootResult> {
		return this.serverClient.quickTroubleshootServer(id);
	}

	async enhancedTroubleshootServer(serverId: string): Promise<EnhancedTroubleshootResult> {
		return this.serverClient.enhancedTroubleshootServer(serverId);
	}

	async autoFixServerIssues(serverId: string): Promise<AutoFixResult> {
		return this.serverClient.autoFixServerIssues(serverId);
	}

	async getServerStatus(serverId: string): Promise<ServerStatus> {
		return this.serverClient.getServerStatus(serverId);
	}

	async subscribeToSetupProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		return this.serverClient.subscribeToSetupProgress(serverId, callback);
	}

	async subscribeToSecurityProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		return this.serverClient.subscribeToSecurityProgress(serverId, callback);
	}

	async unsubscribeFromAll(): Promise<void> {
		return this.serverClient.unsubscribeFromAll();
	}

	// App methods - delegate to AppsClient
	async getApps() {
		return this.appsClient.getApps();
	}

	async getApp(id: string): Promise<AppResponse> {
		return this.appsClient.getApp(id);
	}

	async createApp(data: AppRequest): Promise<App> {
		return this.appsClient.createApp(data);
	}

	async updateApp(id: string, data: Partial<AppRequest>): Promise<App> {
		return this.appsClient.updateApp(id, data);
	}

	async deleteApp(id: string) {
		return this.appsClient.deleteApp(id);
	}

	async getAppsByServer(serverId: string) {
		return this.appsClient.getAppsByServer(serverId);
	}

	async checkAppHealth(id: string): Promise<HealthCheckResponse> {
		return this.appsClient.checkAppHealth(id);
	}

	async runAppHealthCheck(id: string): Promise<HealthCheckResponse> {
		return this.appsClient.runAppHealthCheck(id);
	}

	async startApp(id: string) {
		return this.appsClient.startApp(id);
	}

	async stopApp(id: string) {
		return this.appsClient.stopApp(id);
	}

	async restartApp(id: string) {
		return this.appsClient.restartApp(id);
	}

	// Version methods - delegate to VersionClient
	async getAppVersions(id: string) {
		return this.versionClient.getAppVersions(id);
	}

	async createVersion(
		appId: string,
		data: { version_number: string; notes?: string }
	): Promise<Version> {
		return this.versionClient.createVersion(appId, data);
	}

	async uploadVersionFiles(versionId: string, binaryFile: File, publicFiles: File[]) {
		return this.versionClient.uploadVersionFiles(versionId, binaryFile, publicFiles);
	}

	async uploadVersionWithFolder(
		versionId: string,
		binaryFile: File,
		folderFiles: FileList | File[]
	) {
		return this.versionClient.uploadVersionWithFolder(versionId, binaryFile, folderFiles);
	}

	validatePublicFolderStructure(files: File[]) {
		return this.versionClient.validatePublicFolderStructure(files);
	}

	// Deployment methods - delegate to DeploymentClient
	async getAppDeployments(id: string) {
		return this.deploymentClient.getAppDeployments(id);
	}

	async getDeployment(id: string): Promise<Deployment> {
		return this.deploymentClient.getDeployment(id);
	}

	async createDeployment(appId: string, versionId: string): Promise<Deployment> {
		return this.deploymentClient.createDeployment(appId, versionId);
	}

	async getDeploymentLogs(id: string) {
		return this.deploymentClient.getDeploymentLogs(id);
	}

	async getDeploymentStatus(id: string) {
		return this.deploymentClient.getDeploymentStatus(id);
	}

	// Direct access to specialized clients for advanced usage
	get servers() {
		return this.serverClient;
	}

	get apps() {
		return this.appsClient;
	}

	get versions() {
		return this.versionClient;
	}

	get deployments() {
		return this.deploymentClient;
	}
}
