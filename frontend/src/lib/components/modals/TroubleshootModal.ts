import type {
	ConnectionDiagnostic,
	TroubleshootResult,
	QuickTroubleshootResult
} from '../../api.js';

export type { ConnectionDiagnostic, TroubleshootResult, QuickTroubleshootResult };

export interface TroubleshootModalProps {
	open?: boolean;
	result?: TroubleshootResult | null;
	serverName?: string;
	loading?: boolean;
	onclose?: () => void;
	onretry?: () => void;
	onquicktest?: () => void;
}

export interface TroubleshootModalState {
	open: boolean;
	result: TroubleshootResult | null;
	serverName: string;
	loading: boolean;
	testStartTime?: number;
	lastError?: string;
}

export class TroubleshootModalLogic {
	private state: TroubleshootModalState;
	private stateUpdateCallback?: (state: TroubleshootModalState) => void;
	private onclose?: () => void;
	private onretry?: () => void;
	private onquicktest?: () => void;

	constructor(props: TroubleshootModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			result: props.result ?? null,
			serverName: props.serverName ?? '',
			loading: props.loading ?? false
		};
		this.onclose = props.onclose;
		this.onretry = props.onretry;
		this.onquicktest = props.onquicktest;
	}

	public getState(): TroubleshootModalState {
		return this.state;
	}

	public onStateUpdate(callback: (state: TroubleshootModalState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<TroubleshootModalState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public updateProps(props: Partial<TroubleshootModalProps>): void {
		const updates: Partial<TroubleshootModalState> = {};

		if (props.open !== undefined) {
			updates.open = props.open;
			if (props.open) {
				updates.testStartTime = Date.now();
			}
		}
		if (props.result !== undefined) {
			updates.result = props.result;
			if (props.result && !props.result.success) {
				updates.lastError = 'Connection issues detected';
			}
		}
		if (props.serverName !== undefined) updates.serverName = props.serverName;
		if (props.loading !== undefined) updates.loading = props.loading;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}
		if (props.onretry !== undefined) {
			this.onretry = props.onretry;
		}
		if (props.onquicktest !== undefined) {
			this.onquicktest = props.onquicktest;
		}

		this.updateState(updates);
	}

	public handleClose(): void {
		this.onclose?.();
	}

	public handleRetry(): void {
		this.updateState({ loading: true, lastError: undefined });
		this.onretry?.();
	}

	public handleQuickTest(): void {
		this.updateState({ loading: true });
		this.onquicktest?.();
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

	public hasErrors(): boolean {
		return this.state.result?.has_errors === true;
	}

	public hasWarnings(): boolean {
		return this.state.result?.has_warnings === true;
	}

	public getResult(): TroubleshootResult | null {
		return this.state.result;
	}

	public getServerName(): string {
		return this.state.serverName;
	}

	public getTitle(): string {
		return this.state.loading ? 'Running Diagnostics...' : 'SSH Connection Troubleshooting';
	}

	public getDisplayServerName(): string {
		return this.state.serverName || 'server';
	}

	public formatStepName(step: string): string {
		// Convert snake_case to readable format
		return step
			.split('_')
			.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
			.join(' ');
	}

	public getTestDuration(): number {
		if (!this.state.testStartTime || !this.state.result?.timestamp) return 0;
		const endTime = new Date(this.state.result.timestamp).getTime();
		return Math.round((endTime - this.state.testStartTime) / 1000);
	}

	public getFormattedTimestamp(): string {
		if (!this.state.result?.timestamp) return '';
		return new Date(this.state.result.timestamp).toLocaleTimeString();
	}

	public isConnectionRefusedDetected(): boolean {
		if (!this.state.result) return false;

		return (
			this.state.result.diagnostics.some(
				(diag) =>
					diag.step === 'network_connectivity' &&
					diag.status === 'error' &&
					(diag.details?.includes('connection refused') ||
						diag.message?.includes('connection refused'))
			) || this.state.result.diagnostics.some((diag) => diag.step === 'connection_refused_analysis')
		);
	}

	public isFail2banIssueDetected(): boolean {
		if (!this.state.result) return false;

		return this.state.result.diagnostics.some(
			(diag) =>
				diag.step === 'fail2ban_check' ||
				diag.step === 'fail2ban_ban_check' ||
				diag.step === 'connection_refused_analysis' ||
				diag.message?.includes('fail2ban') ||
				diag.details?.includes('fail2ban')
		);
	}

	public getFailureCategories(): {
		network: boolean;
		auth: boolean;
		permission: boolean;
		fail2ban: boolean;
	} {
		if (!this.state.result) {
			return { network: false, auth: false, permission: false, fail2ban: false };
		}

		const categories = {
			network: false,
			auth: false,
			permission: false,
			fail2ban: false
		};

		for (const diag of this.state.result.diagnostics) {
			if (diag.status === 'error') {
				const step = diag.step.toLowerCase();
				const message = diag.message?.toLowerCase() || '';
				const details = diag.details?.toLowerCase() || '';

				if (step.includes('network') || step.includes('connectivity')) {
					categories.network = true;
				}
				if (step.includes('auth') || message.includes('auth') || details.includes('auth')) {
					categories.auth = true;
				}
				if (
					step.includes('permission') ||
					message.includes('permission') ||
					details.includes('permission')
				) {
					categories.permission = true;
				}
				if (
					step.includes('fail2ban') ||
					message.includes('fail2ban') ||
					details.includes('fail2ban') ||
					message.includes('banned') ||
					details.includes('banned')
				) {
					categories.fail2ban = true;
				}
			}
		}

		return categories;
	}

	public getPriorityActions(): string[] {
		if (!this.state.result || this.state.result.success) return [];

		const actions: string[] = [];
		const categories = this.getFailureCategories();

		if (categories.fail2ban || this.isConnectionRefusedDetected()) {
			actions.push('Check if your IP is banned by fail2ban');
			actions.push('Try connecting from a different IP address');
			actions.push('Use console access to unban your IP');
		}

		if (categories.network) {
			actions.push('Verify server is running and accessible');
			actions.push('Check firewall settings (UFW/iptables)');
			actions.push('Confirm SSH service is running');
		}

		if (categories.auth) {
			actions.push('Verify SSH key configuration');
			actions.push('Check username and authentication method');
			actions.push('Review SSH client configuration');
		}

		if (categories.permission) {
			actions.push('Check SSH key file permissions');
			actions.push('Verify user account setup');
			actions.push('Review sudo access configuration');
		}

		// If no specific categories, provide general guidance
		if (actions.length === 0) {
			actions.push('Review diagnostic details below');
			actions.push('Check server logs for more information');
			actions.push('Verify server configuration');
		}

		return actions.slice(0, 4); // Limit to top 4 actions
	}

	public getStatusSummary(): string {
		if (!this.state.result) return '';

		const { success_count, warning_count, error_count } = this.state.result;
		const total = success_count + warning_count + error_count;

		if (total === 0) return 'No diagnostics run';

		if (error_count === 0 && warning_count === 0) {
			return `All ${success_count} checks passed successfully`;
		}

		let summary = `${success_count}/${total} checks passed`;
		if (warning_count > 0) {
			summary += `, ${warning_count} warning${warning_count !== 1 ? 's' : ''}`;
		}
		if (error_count > 0) {
			summary += `, ${error_count} error${error_count !== 1 ? 's' : ''}`;
		}

		return summary;
	}

	public getDiagnosticsByStatus(status: 'success' | 'warning' | 'error'): ConnectionDiagnostic[] {
		if (!this.state.result) return [];
		return this.state.result.diagnostics.filter((diag) => diag.status === status);
	}

	public getErrorDiagnostics(): ConnectionDiagnostic[] {
		return this.getDiagnosticsByStatus('error');
	}

	public getWarningDiagnostics(): ConnectionDiagnostic[] {
		return this.getDiagnosticsByStatus('warning');
	}

	public getSuccessDiagnostics(): ConnectionDiagnostic[] {
		return this.getDiagnosticsByStatus('success');
	}

	public hasNetworkErrors(): boolean {
		return this.getErrorDiagnostics().some(
			(diag) => diag.step.includes('network') || diag.step.includes('connectivity')
		);
	}

	public hasAuthErrors(): boolean {
		return this.getErrorDiagnostics().some(
			(diag) => diag.step.includes('auth') || diag.message?.includes('auth')
		);
	}

	public hasPermissionErrors(): boolean {
		return this.getErrorDiagnostics().some(
			(diag) => diag.step.includes('permission') || diag.message?.includes('permission')
		);
	}

	public getMainErrorMessage(): string {
		if (!this.state.result || this.state.result.success) return '';

		// Look for the most critical error
		const errors = this.getErrorDiagnostics();
		if (errors.length === 0) {
			const warnings = this.getWarningDiagnostics();
			return warnings.length > 0 ? warnings[0].message : 'Unknown issue detected';
		}

		// Prioritize network connectivity errors as they're usually the root cause
		const networkError = errors.find(
			(diag) => diag.step.includes('network') || diag.step.includes('connectivity')
		);
		if (networkError) return networkError.message;

		// Then fail2ban issues
		const fail2banError = errors.find(
			(diag) => diag.step.includes('fail2ban') || diag.message?.includes('fail2ban')
		);
		if (fail2banError) return fail2banError.message;

		// Return first error if no specific priority found
		return errors[0].message;
	}

	public shouldShowQuickTest(): boolean {
		return this.state.result !== null && !this.state.loading;
	}

	public shouldShowRetry(): boolean {
		return (
			this.state.result !== null &&
			(this.state.result.has_errors || this.state.result.has_warnings) &&
			!this.state.loading
		);
	}

	public isCloseable(): boolean {
		return !this.state.loading;
	}
}
