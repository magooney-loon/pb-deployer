import PocketBase from 'pocketbase';

let pocketBaseInstance: PocketBase | null = null;

function getPocketBaseInstance(baseUrl: string = 'http://localhost:8090'): PocketBase {
	if (!pocketBaseInstance) {
		pocketBaseInstance = new PocketBase(baseUrl);
	}
	return pocketBaseInstance;
}

export class BaseClient {
	protected pb: PocketBase;

	constructor(baseUrl: string = 'http://localhost:8090') {
		this.pb = getPocketBaseInstance(baseUrl);
	}

	// Get the PocketBase instance for advanced usage
	getPocketBase(): PocketBase {
		return this.pb;
	}

	protected get baseURL(): string {
		return this.pb.baseURL;
	}
}
