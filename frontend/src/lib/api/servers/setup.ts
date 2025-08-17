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
		console.debug('[SetupClient] Starting server setup:', setupRequest);

		try {
			const url = `${this.pb.baseUrl}/api/setup/server`;
			console.debug('[SetupClient] Making setup request to:', url);

			const response = await fetch(url, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
				},
				body: JSON.stringify(setupRequest)
			});

			console.debug('[SetupClient] Setup response status:', response.status);

			const responseText = await response.text();
			console.debug('[SetupClient] Setup response text:', responseText);

			if (!response.ok) {
				let errorData;
				try {
					errorData = JSON.parse(responseText);
				} catch (parseError) {
					console.error('[SetupClient] Failed to parse error response:', parseError);
					throw new Error(
						`Setup failed with status ${response.status}. Response: ${responseText.substring(0, 200)}...`
					);
				}
				console.error('[SetupClient] Setup error response:', errorData);
				throw new Error(errorData.error || `Setup failed with status ${response.status}`);
			}

			let data;
			try {
				data = JSON.parse(responseText);
			} catch (parseError) {
				console.error('[SetupClient] Failed to parse success response:', parseError);
				throw new Error(
					`Setup response parsing failed. Response: ${responseText.substring(0, 200)}...`
				);
			}
			console.debug('[SetupClient] Setup success response:', data);
			return data as SetupResponse;
		} catch (error) {
			console.error('[SetupClient] Setup server failed:', error);
			throw error;
		}
	}

	/**
	 * Apply security hardening to a server
	 * This configures firewall, hardens SSH, and sets up fail2ban
	 */
	async secureServer(securityRequest: SecurityRequest): Promise<SecurityResponse> {
		console.debug('[SetupClient] Starting server security:', securityRequest);
		try {
			const url = `${this.pb.baseUrl}/api/setup/security`;
			console.debug('[SetupClient] Making security request to:', url);

			const response = await fetch(url, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
				},
				body: JSON.stringify(securityRequest)
			});

			console.debug('[SetupClient] Security response status:', response.status);

			const responseText = await response.text();
			console.debug('[SetupClient] Security response text:', responseText);

			if (!response.ok) {
				let errorData;
				try {
					errorData = JSON.parse(responseText);
				} catch (parseError) {
					console.error('[SetupClient] Failed to parse error response:', parseError);
					throw new Error(
						`Security setup failed with status ${response.status}. Response: ${responseText.substring(0, 200)}...`
					);
				}
				console.error('[SetupClient] Security error response:', errorData);
				throw new Error(errorData.error || `Security setup failed with status ${response.status}`);
			}

			let data;
			try {
				data = JSON.parse(responseText);
			} catch (parseError) {
				console.error('[SetupClient] Failed to parse success response:', parseError);
				throw new Error(
					`Security response parsing failed. Response: ${responseText.substring(0, 200)}...`
				);
			}
			console.debug('[SetupClient] Security success response:', data);
			return data as SecurityResponse;
		} catch (error) {
			console.error('[SetupClient] Secure server failed:', error);
			throw error;
		}
	}

	/**
	 * Validate server setup and configuration
	 */
	async validateServer(validationRequest: ValidationRequest): Promise<ValidationResponse> {
		console.debug('[SetupClient] Starting server validation:', validationRequest);
		try {
			const url = `${this.pb.baseUrl}/api/setup/validate`;
			console.debug('[SetupClient] Making validation request to:', url);

			const response = await fetch(url, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
				},
				body: JSON.stringify(validationRequest)
			});

			console.debug('[SetupClient] Validation response status:', response.status);

			const responseText = await response.text();
			console.debug('[SetupClient] Validation response text:', responseText);

			if (!response.ok) {
				let errorData;
				try {
					errorData = JSON.parse(responseText);
				} catch (parseError) {
					console.error('[SetupClient] Failed to parse error response:', parseError);
					throw new Error(
						`Validation failed with status ${response.status}. Response: ${responseText.substring(0, 200)}...`
					);
				}
				console.error('[SetupClient] Validation error response:', errorData);
				throw new Error(errorData.error || `Validation failed with status ${response.status}`);
			}

			let data;
			try {
				data = JSON.parse(responseText);
			} catch (parseError) {
				console.error('[SetupClient] Failed to parse success response:', parseError);
				throw new Error(
					`Validation response parsing failed. Response: ${responseText.substring(0, 200)}...`
				);
			}
			console.debug('[SetupClient] Validation success response:', data);
			return data as ValidationResponse;
		} catch (error) {
			console.error('[SetupClient] Validate server failed:', error);
			throw error;
		}
	}

	/**
	 * Helper method to setup server from database record
	 */
	async setupServerFromRecord(serverId: string): Promise<SetupResponse> {
		try {
			// Get server details from database
			const server = await this.pb.collection('servers').getOne(serverId);

			const setupRequest: SetupRequest = {
				host: server.host,
				port: server.port || 22,
				user: server.root_username,
				username: server.app_username,
				public_keys: [] // TODO: Get public keys from user's SSH agent or input
			};

			return await this.setupServer(setupRequest);
		} catch (error) {
			console.error('[SetupClient] Setup server from record failed:', error);
			throw error;
		}
	}

	/**
	 * Helper method to secure server from database record
	 */
	async secureServerFromRecord(serverId: string): Promise<SecurityResponse> {
		try {
			// Get server details from database
			const server = await this.pb.collection('servers').getOne(serverId);

			const securityRequest: SecurityRequest = {
				host: server.host,
				port: server.port || 22,
				user: server.root_username,
				enable_fail2ban: true
				// firewall_rules and ssh_config will use defaults if not provided
			};

			return await this.secureServer(securityRequest);
		} catch (error) {
			console.error('[SetupClient] Secure server from record failed:', error);
			throw error;
		}
	}

	/**
	 * Helper method to validate server from database record
	 */
	async validateServerFromRecord(serverId: string): Promise<ValidationResponse> {
		try {
			// Get server details from database
			const server = await this.pb.collection('servers').getOne(serverId);

			const validationRequest: ValidationRequest = {
				host: server.host,
				port: server.port || 22,
				user: server.root_username,
				username: server.app_username
			};

			return await this.validateServer(validationRequest);
		} catch (error) {
			console.error('[SetupClient] Validate server from record failed:', error);
			throw error;
		}
	}
}
