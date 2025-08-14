import { AppsCrudClient } from './apps/crud.js';
import { ServerCrudClient } from './servers/crud.js';
import { VersionCrudClient } from './version/crud.js';
import { DeploymentCrudClient } from './deployment/crud.js';
import { BaseClient } from './base.js';

export class ApiClient extends BaseClient {
	private _apps: AppsCrudClient;
	private _servers: ServerCrudClient;
	private _versions: VersionCrudClient;
	private _deployments: DeploymentCrudClient;

	constructor(baseUrl: string = 'http://localhost:8090') {
		super(baseUrl);
		this._apps = new AppsCrudClient(baseUrl);
		this._servers = new ServerCrudClient(baseUrl);
		this._versions = new VersionCrudClient(baseUrl);
		this._deployments = new DeploymentCrudClient(baseUrl);
	}

	get apps() {
		return this._apps;
	}
	get servers() {
		return this._servers;
	}
	get versions() {
		return this._versions;
	}
	get deployments() {
		return this._deployments;
	}
}
