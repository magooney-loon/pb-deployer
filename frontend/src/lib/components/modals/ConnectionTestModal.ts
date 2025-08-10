export interface TCPTestResult {
	success: boolean;
	error?: string;
	latency?: string;
}

export interface SSHTestResult {
	success: boolean;
	error?: string;
	username: string;
	auth_method?: string;
}

export interface ConnectionTestResult {
	success: boolean;
	error?: string;
	tcp_connection: TCPTestResult;
	root_ssh_connection: SSHTestResult;
	app_ssh_connection: SSHTestResult;
	overall_status: string;
}

export interface ConnectionTestModalProps {
	open?: boolean;
	result?: ConnectionTestResult | null;
	serverName?: string;
	loading?: boolean;
	onclose?: () => void;
}

export interface ConnectionTestModalState {
	open: boolean;
	result: ConnectionTestResult | null;
	serverName: string;
	loading: boolean;
}

export class ConnectionTestModalLogic {
	private state: ConnectionTestModalState;
	private stateUpdateCallback?: (state: ConnectionTestModalState) => void;
	private onclose?: () => void;

	constructor(props: ConnectionTestModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			result: props.result ?? null,
			serverName: props.serverName ?? '',
			loading: props.loading ?? false
		};
		this.onclose = props.onclose;
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

		if (props.open !== undefined) updates.open = props.open;
		if (props.result !== undefined) updates.result = props.result;
		if (props.serverName !== undefined) updates.serverName = props.serverName;
		if (props.loading !== undefined) updates.loading = props.loading;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}

		this.updateState(updates);
	}

	public handleClose(): void {
		this.onclose?.();
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
		return [
			'Check that the server IP address is correct',
			'Check firewall settings on both client and server'
		];
	}
}
