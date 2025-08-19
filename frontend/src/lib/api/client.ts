import PocketBase from 'pocketbase';
import { AppsCrudClient } from './apps/crud.js';
import { ServerCrudClient } from './servers/crud.js';
import { ServerSetupClient } from './servers/setup.js';
import { VersionCrudClient } from './version/crud.js';
import { DeploymentCrudClient } from './deployment/crud.js';
import { DeploymentClient } from './deployment/deploy.js';

export class ApiClient {
	private pb: PocketBase;
	private _apps: AppsCrudClient;
	private _servers: ServerCrudClient;
	private _setup: ServerSetupClient;
	private _versions: VersionCrudClient;
	private _deployments: DeploymentCrudClient;
	private _deploy: DeploymentClient;

	constructor(baseUrl: string = 'http://localhost:8090') {
		this.pb = new PocketBase(baseUrl);

		// Pass the shared PocketBase instance to each CRUD client
		this._apps = new AppsCrudClient(this.pb);
		this._servers = new ServerCrudClient(this.pb);
		this._setup = new ServerSetupClient(this.pb);
		this._versions = new VersionCrudClient(this.pb);
		this._deployments = new DeploymentCrudClient(this.pb);
		this._deploy = new DeploymentClient(this.pb);
	}

	get apps() {
		return this._apps;
	}

	get servers() {
		return this._servers;
	}

	get setup() {
		return this._setup;
	}

	get versions() {
		return this._versions;
	}

	get deployments() {
		return this._deployments;
	}

	get deploy() {
		return this._deploy;
	}

	getPocketBase(): PocketBase {
		return this.pb;
	}
}
