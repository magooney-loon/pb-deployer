export interface TCPTestResult {
	success: boolean;
	error?: string;
	latency?: string;
	retry_count?: number;
}

export interface SSHTestResult {
	success: boolean;
	error?: string;
	username: string;
	auth_method?: string;
	retry_count?: number;
	connection_time?: string;
}

export interface ConnectionTestResult {
	success: boolean;
	error?: string;
	tcp_connection: TCPTestResult;
	root_ssh_connection: SSHTestResult;
	app_ssh_connection: SSHTestResult;
	overall_status: string;
	test_duration?: number;
	timestamp?: string;
}

export interface ConnectionDiagnostic {
	step: string;
	status: 'success' | 'warning' | 'error';
	message: string;
	details?: string;
	suggestion?: string;
}

export interface ConnectionHealth {
	healthy: boolean;
	last_check: string;
	response_time?: number;
	error_count: number;
}

export interface ConnectionTestModalProps {
	open?: boolean;
	result?: ConnectionTestResult | null;
	serverName?: string;
	loading?: boolean;
	onclose?: () => void;
	onretry?: () => void;
	retryCount?: number;
	maxRetries?: number;
}

export interface ConnectionTestModalState {
	open: boolean;
	result: ConnectionTestResult | null;
	serverName: string;
	loading: boolean;
	retryCount: number;
	maxRetries: number;
	testStartTime?: number;
	lastError?: string;
	autoRetryEnabled: boolean;
}

export class ConnectionTestModalLogic {
	private state: ConnectionTestModalState;
	private stateUpdateCallback?: (state: ConnectionTestModalState) => void;
	private onclose?: () => void;
	private onretry?: () => void;
	private retryTimeout?: number;

	constructor(props: ConnectionTestModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			result: props.result ?? null,
			serverName: props.serverName ?? '',
			loading: props.loading ?? false,
			retryCount: props.retryCount ?? 0,
			maxRetries: props.maxRetries ?? 3,
			autoRetryEnabled: false
		};
		this.onclose = props.onclose;
		this.onretry = props.onretry;
	}

	public getState(): ConnectionTestModalState {
		return this.state;
	}

	public onStateUpdate(callback: (state: ConnectionTestModalState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<ConnectionTestModalState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public updateProps(props: Partial<ConnectionTestModalProps>): void {
		const updates: Partial<ConnectionTestModalState> = {};

		if (props.open !== undefined) {
			updates.open = props.open;
			if (props.open) {
				updates.testStartTime = Date.now();
			}
		}
		if (props.result !== undefined) {
			updates.result = props.result;
			// Reset retry count on new result
			if (props.result && !props.result.success) {
				updates.lastError = props.result.error;
			}
		}
		if (props.serverName !== undefined) updates.serverName = props.serverName;
		if (props.loading !== undefined) {
			updates.loading = props.loading;
			// Clear any pending retry when loading starts
			if (props.loading && this.retryTimeout) {
				clearTimeout(this.retryTimeout);
				this.retryTimeout = undefined;
			}
		}
		if (props.retryCount !== undefined) updates.retryCount = props.retryCount;
		if (props.maxRetries !== undefined) updates.maxRetries = props.maxRetries;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}
		if (props.onretry !== undefined) {
			this.onretry = props.onretry;
		}

		this.updateState(updates);
	}

	public handleClose(): void {
		// Clear any pending retry
		if (this.retryTimeout) {
			clearTimeout(this.retryTimeout);
			this.retryTimeout = undefined;
		}
		this.onclose?.();
	}

	public handleRetry(): void {
		if (this.state.retryCount >= this.state.maxRetries) {
			return;
		}

		this.updateState({
			retryCount: this.state.retryCount + 1,
			loading: true,
			lastError: undefined
		});

		this.onretry?.();
	}

	public enableAutoRetry(): void {
		this.updateState({ autoRetryEnabled: true });
		this.scheduleAutoRetry();
	}

	public disableAutoRetry(): void {
		this.updateState({ autoRetryEnabled: false });
		if (this.retryTimeout) {
			clearTimeout(this.retryTimeout);
			this.retryTimeout = undefined;
		}
	}

	private scheduleAutoRetry(): void {
		if (!this.state.autoRetryEnabled || this.state.retryCount >= this.state.maxRetries) {
			return;
		}

		// Schedule retry with exponential backoff
		const delay = Math.min(1000 * Math.pow(2, this.state.retryCount), 30000); // Max 30s

		this.retryTimeout = setTimeout(() => {
			if (this.state.autoRetryEnabled && !this.state.loading) {
				this.handleRetry();
			}
		}, delay);
	}

	public isOpen(): boolean {
		return this.state.open;
	}

	public hasResult(): boolean {
		return this.state.result !== null;
	}

	public isLoading(): boolean {
		return this.state.loading;
	}

	public isSuccess(): boolean {
		return this.state.result?.success === true;
	}

	public isError(): boolean {
		return this.state.result?.success === false;
	}

	public getResult(): ConnectionTestResult | null {
		return this.state.result;
	}

	public getServerName(): string {
		return this.state.serverName;
	}

	public getTitle(): string {
		return this.state.loading ? 'Testing Connection...' : 'Connection Test Results';
	}

	public getTCPConnection(): TCPTestResult | undefined {
		return this.state.result?.tcp_connection;
	}

	public getRootSSHConnection(): SSHTestResult | undefined {
		return this.state.result?.root_ssh_connection;
	}

	public getAppSSHConnection(): SSHTestResult | undefined {
		return this.state.result?.app_ssh_connection;
	}

	public getOverallStatus(): string | undefined {
		return this.state.result?.overall_status;
	}

	public getError(): string {
		return this.state.result?.error || 'Unknown connection error';
	}

	public getDisplayServerName(): string {
		return this.state.serverName || 'the server';
	}

	public getTroubleshootingTips(): string[] {
		const tips = [
			'Check that the server IP address is correct',
			'Check firewall settings on both client and server'
		];

		// Add specific tips based on the error
		if (this.state.result && !this.state.result.success) {
			const error = this.state.result.error?.toLowerCase() || '';

			if (error.includes('timeout')) {
				tips.push('Connection is timing out - check network connectivity');
				tips.push('Verify the server is running and accessible');
			}

			if (error.includes('refused')) {
				tips.push('Connection refused - check if SSH service is running');
				tips.push('Verify the correct port number (usually 22)');
			}

			if (error.includes('authentication') || error.includes('permission')) {
				tips.push('Check SSH key configuration and permissions');
				tips.push('Verify username and authentication method');
			}

			if (error.includes('host key')) {
				tips.push('Host key verification failed - check known_hosts file');
				tips.push('Consider accepting the new host key if this is expected');
			}

			if (this.state.result.overall_status === 'app_ssh_failed') {
				tips.push('App user SSH access failed - check key setup for the application user');
				tips.push('Verify sudo permissions are configured correctly');
			}
		}

		return tips;
	}

	public getRetryCount(): number {
		return this.state.retryCount;
	}

	public getMaxRetries(): number {
		return this.state.maxRetries;
	}

	public canRetry(): boolean {
		return this.state.retryCount < this.state.maxRetries && !this.state.loading;
	}

	public isAutoRetryEnabled(): boolean {
		return this.state.autoRetryEnabled;
	}

	public getTestDuration(): number {
		if (!this.state.testStartTime) return 0;
		const endTime = this.state.result?.timestamp
			? new Date(this.state.result.timestamp).getTime()
			: Date.now();
		return Math.round((endTime - this.state.testStartTime) / 1000);
	}

	public getConnectionSummary(): string {
		if (!this.state.result) return '';

		const { tcp_connection, overall_status } = this.state.result;

		let summary = `Connection test completed in ${this.getTestDuration()}s. `;

		if (tcp_connection.success) {
			summary += `TCP connection successful (${tcp_connection.latency}). `;
		} else {
			summary += 'TCP connection failed. ';
		}

		if (overall_status === 'healthy_secured') {
			summary += 'Server is security-locked with working app user access.';
		} else if (overall_status === 'healthy') {
			summary += 'All connections are working properly.';
		} else {
			summary += `Status: ${overall_status}.`;
		}

		return summary;
	}

	public getDetailedErrorInfo(): string[] {
		if (!this.state.result || this.state.result.success) return [];

		const errors: string[] = [];
		const { tcp_connection, root_ssh_connection, app_ssh_connection } = this.state.result;

		if (!tcp_connection.success && tcp_connection.error) {
			errors.push(`TCP: ${tcp_connection.error}`);
		}

		if (!root_ssh_connection.success && root_ssh_connection.error) {
			errors.push(`Root SSH: ${root_ssh_connection.error}`);
		}

		if (!app_ssh_connection.success && app_ssh_connection.error) {
			errors.push(`App SSH: ${app_ssh_connection.error}`);
		}

		return errors;
	}

	public shouldShowRetryButton(): boolean {
		return this.canRetry() && this.isError() && !this.isLoading();
	}

	public shouldShowAutoRetryOption(): boolean {
		return this.isError() && this.canRetry();
	}
}
