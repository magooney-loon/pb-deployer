import { AppsCrudClient } from './apps/index.js';
import { ServerCrudClient } from './servers/index.js';
import { VersionCrudClient } from './version/index.js';
import { DeploymentCrudClient } from './deployment/index.js';
import { BaseClient } from './base.js';
import type { AppRequest } from './apps/types.js';
import type { ServerRequest } from './servers/types.js';

export class ApiClient extends BaseClient {
	private apps: AppsCrudClient;
	private servers: ServerCrudClient;
	private versions: VersionCrudClient;
	private deployments: DeploymentCrudClient;

	constructor(baseUrl: string = 'http://localhost:8090') {
		super(baseUrl);
		this.apps = new AppsCrudClient(baseUrl);
		this.servers = new ServerCrudClient(baseUrl);
		this.versions = new VersionCrudClient(baseUrl);
		this.deployments = new DeploymentCrudClient(baseUrl);
	}

	// App methods
	async getApps() {
		return this.apps.getApps();
	}

	async getApp(id: string) {
		return this.apps.getApp(id);
	}

	async createApp(data: AppRequest) {
		return this.apps.createApp(data);
	}

	async updateApp(id: string, data: Partial<AppRequest>) {
		return this.apps.updateApp(id, data);
	}

	async deleteApp(id: string) {
		return this.apps.deleteApp(id);
	}

	async getAppsByServer(serverId: string) {
		return this.apps.getAppsByServer(serverId);
	}

	// Server methods
	async getServers() {
		return this.servers.getServers();
	}

	async getServer(id: string) {
		return this.servers.getServer(id);
	}

	async createServer(data: ServerRequest) {
		return this.servers.createServer(data);
	}

	async updateServer(id: string, data: Partial<ServerRequest>) {
		return this.servers.updateServer(id, data);
	}

	async deleteServer(id: string) {
		return this.servers.deleteServer(id);
	}

	// Version methods
	async getVersions() {
		return this.versions.getVersions();
	}

	async getVersion(id: string) {
		return this.versions.getVersion(id);
	}

	async createVersion(data: { app_id: string; version_number: string; notes?: string }) {
		return this.versions.createVersion(data);
	}

	async updateVersion(
		id: string,
		data: Partial<{ version_number: string; notes: string; deployment_zip: string }>
	) {
		return this.versions.updateVersion(id, data);
	}

	async deleteVersion(id: string) {
		return this.versions.deleteVersion(id);
	}

	async getAppVersions(appId: string) {
		return this.versions.getAppVersions(appId);
	}

	// Deployment methods
	async getDeployments() {
		return this.deployments.getDeployments();
	}

	async getDeployment(id: string) {
		return this.deployments.getDeployment(id);
	}

	async createDeployment(data: { app_id: string; version_id: string; status?: string }) {
		return this.deployments.createDeployment(data);
	}

	async updateDeployment(
		id: string,
		data: Partial<{ status: string; logs: string; started_at: string; completed_at: string }>
	) {
		return this.deployments.updateDeployment(id, data);
	}

	async deleteDeployment(id: string) {
		return this.deployments.deleteDeployment(id);
	}

	async getAppDeployments(appId: string) {
		return this.deployments.getAppDeployments(appId);
	}

	async getVersionDeployments(versionId: string) {
		return this.deployments.getVersionDeployments(versionId);
	}
}
