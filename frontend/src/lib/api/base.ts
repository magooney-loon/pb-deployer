import PocketBase from 'pocketbase';

export class BaseClient {
	protected pb: PocketBase;

	constructor(baseUrl: string = 'http://localhost:8090') {
		this.pb = new PocketBase(baseUrl);
		console.log('PocketBase client initialized with URL:', baseUrl);
	}

	// Health & Info endpoints
	async getHealth() {
		console.log('API Request: GET /api/health');
		try {
			const response = await fetch(`${this.pb.baseURL}/api/health`);
			const data = await response.json();
			console.log('Health check response:', data);
			return data;
		} catch (error) {
			console.error('Health check failed:', error);
			throw error;
		}
	}

	async getApiInfo() {
		console.log('API Request: GET /api/info');
		try {
			const response = await fetch(`${this.pb.baseURL}/api/info`);
			const data = await response.json();
			console.log('API info response:', data);
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
