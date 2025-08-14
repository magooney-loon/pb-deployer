import PocketBase from 'pocketbase';

// Singleton PocketBase instance
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

	// Health & Info endpoints
	async getHealth() {
		try {
			const response = await fetch(`${this.pb.baseURL}/api/health`);
			const data = await response.json();

			return data;
		} catch (error) {
			console.error('Health check failed:', error);
			throw error;
		}
	}

	async getApiInfo() {
		try {
			const response = await fetch(`${this.pb.baseURL}/api/info`);
			const data = await response.json();

			return data;
		} catch (error) {
			console.error('API info failed:', error);
			throw error;
		}
	}

	// Get the PocketBase instance for advanced usage
	getPocketBase(): PocketBase {
		return this.pb;
	}

	// Get base URL for custom endpoints
	protected get baseURL(): string {
		return this.pb.baseURL;
	}
}
