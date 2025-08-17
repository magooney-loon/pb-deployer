import PocketBase from 'pocketbase';

export interface SetupInfo {
	os: string;
	architecture: string;
	hostname: string;
	pocketbase_setup: boolean;
	installed_apps: string[];
}

export interface FirewallRule {
	port: number;
	protocol: string;
	source?: string;
	action: string;
	description: string;
}

export interface SSHConfig {
	password_auth: boolean;
	root_login: boolean;
	pubkey_auth: boolean;
	max_auth_tries: number;
	client_alive_interval: number;
	client_alive_count_max: number;
	allow_users?: string[];
	allow_groups?: string[];
}

export interface SetupRequest {
	host: string;
	port: number;
	user: string;
	username: string;
	public_keys: string[];
}

export interface SecurityRequest {
	host: string;
	port: number;
	user: string;
	firewall_rules?: FirewallRule[];
	ssh_config?: SSHConfig;
	enable_fail2ban: boolean;
}

export interface ValidationRequest {
	host: string;
	port: number;
	user: string;
	username: string;
}

export interface SetupResponse {
	success: boolean;
	message: string;
	setup_info: SetupInfo;
}

export interface SecurityResponse {
	success: boolean;
	message: string;
	applied_config: {
		firewall_rules: FirewallRule[];
		ssh_hardened: boolean;
		fail2ban_enabled: boolean;
	};
}

export interface ValidationResponse {
	valid: boolean;
	message: string;
	setup_info?: SetupInfo;
	error?: string;
}

export interface ValidationError {
	error: string;
}

export class ServerSetupClient {
	private pb: PocketBase;

	constructor(pb: PocketBase) {
		this.pb = pb;
	}

	/**
	 * Setup a server for PocketBase deployment
	 * This creates the necessary users, directories, and installs essential packages
	 */
	async setupServer(setupRequest: SetupRequest): Promise<SetupResponse> {
		const url = `${this.pb.baseURL}/api/setup/server`;

		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
			},
			body: JSON.stringify(setupRequest)
		});

		const responseText = await response.text();

		if (!response.ok) {
			let errorData;
			try {
				errorData = JSON.parse(responseText);
			} catch {
				throw new Error(`Setup failed (${response.status})`);
			}
			throw new Error(errorData.error || 'Setup failed');
		}

		try {
			return JSON.parse(responseText) as SetupResponse;
		} catch {
			throw new Error('Invalid response format');
		}
	}

	/**
	 * Apply security hardening to a server
	 * This configures firewall, hardens SSH, and sets up fail2ban
	 */
	async secureServer(securityRequest: SecurityRequest): Promise<SecurityResponse> {
		const url = `${this.pb.baseURL}/api/setup/security`;

		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
			},
			body: JSON.stringify(securityRequest)
		});

		const responseText = await response.text();

		if (!response.ok) {
			let errorData;
			try {
				errorData = JSON.parse(responseText);
			} catch {
				throw new Error(`Security setup failed (${response.status})`);
			}
			throw new Error(errorData.error || 'Security setup failed');
		}

		try {
			return JSON.parse(responseText) as SecurityResponse;
		} catch {
			throw new Error('Invalid response format');
		}
	}

	/**
	 * Validate server setup and configuration
	 */
	async validateServer(validationRequest: ValidationRequest): Promise<ValidationResponse> {
		const url = `${this.pb.baseURL}/api/setup/validate`;

		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
			},
			body: JSON.stringify(validationRequest)
		});

		const responseText = await response.text();

		if (!response.ok) {
			let errorData;
			try {
				errorData = JSON.parse(responseText);
			} catch {
				throw new Error(`Validation failed (${response.status})`);
			}
			throw new Error(errorData.error || 'Validation failed');
		}

		try {
			return JSON.parse(responseText) as ValidationResponse;
		} catch {
			throw new Error('Invalid response format');
		}
	}

	/**
	 * Helper method to setup server from database record
	 */
	async setupServerFromRecord(serverId: string): Promise<SetupResponse> {
		const server = await this.pb.collection('servers').getOne(serverId);

		const setupRequest: SetupRequest = {
			host: server.host,
			port: server.port || 22,
			user: server.root_username,
			username: server.app_username,
			public_keys: [] // TODO: Get public keys from user's SSH agent or input
		};

		return await this.setupServer(setupRequest);
	}

	/**
	 * Helper method to secure server from database record
	 */
	async secureServerFromRecord(serverId: string): Promise<SecurityResponse> {
		const server = await this.pb.collection('servers').getOne(serverId);

		const securityRequest: SecurityRequest = {
			host: server.host,
			port: server.port || 22,
			user: server.root_username,
			enable_fail2ban: true
			// firewall_rules and ssh_config will use defaults if not provided
		};

		return await this.secureServer(securityRequest);
	}

	/**
	 * Helper method to validate server from database record
	 */
	async validateServerFromRecord(serverId: string): Promise<ValidationResponse> {
		const server = await this.pb.collection('servers').getOne(serverId);

		const validationRequest: ValidationRequest = {
			host: server.host,
			port: server.port || 22,
			user: server.root_username,
			username: server.app_username
		};

		return await this.validateServer(validationRequest);
	}
}
