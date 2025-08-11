import type {
	ConnectionDiagnostic,
	TroubleshootResult,
	QuickTroubleshootResult,
	EnhancedTroubleshootResult,
	RecoveryStep,
	ActionableSuggestion,
	AutoFixResult
} from '../../api.js';

export type {
	ConnectionDiagnostic,
	TroubleshootResult,
	QuickTroubleshootResult,
	EnhancedTroubleshootResult,
	RecoveryStep,
	ActionableSuggestion,
	AutoFixResult
};

export interface TroubleshootModalProps {
	open?: boolean;
	result?: TroubleshootResult | null;
	enhancedResult?: EnhancedTroubleshootResult | null;
	autoFixResult?: AutoFixResult | null;
	serverName?: string;
	loading?: boolean;
	mode?: 'basic' | 'enhanced' | 'auto-fix';
	onclose?: () => void;
	onretry?: () => void;
	onquicktest?: () => void;
	onenhanced?: () => void;
	onautofix?: () => void;
}

export interface TroubleshootModalState {
	open: boolean;
	result: TroubleshootResult | null;
	enhancedResult: EnhancedTroubleshootResult | null;
	autoFixResult: AutoFixResult | null;
	serverName: string;
	loading: boolean;
	mode: 'basic' | 'enhanced' | 'auto-fix';
	testStartTime?: number;
	lastError?: string;
	currentView: 'diagnostics' | 'analysis' | 'recovery' | 'suggestions';
	showAdvanced: boolean;
}

export class TroubleshootModalLogic {
	private state: TroubleshootModalState;
	private stateUpdateCallback?: (state: TroubleshootModalState) => void;
	private onclose?: () => void;
	private onretry?: () => void;
	private onquicktest?: () => void;
	private onenhanced?: () => void;
	private onautofix?: () => void;

	constructor(props: TroubleshootModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			result: props.result ?? null,
			enhancedResult: props.enhancedResult ?? null,
			autoFixResult: props.autoFixResult ?? null,
			serverName: props.serverName ?? '',
			loading: props.loading ?? false,
			mode: props.mode ?? 'basic',
			currentView: 'diagnostics',
			showAdvanced: false
		};
		this.onclose = props.onclose;
		this.onretry = props.onretry;
		this.onquicktest = props.onquicktest;
		this.onenhanced = props.onenhanced;
		this.onautofix = props.onautofix;
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
		if (props.enhancedResult !== undefined) {
			updates.enhancedResult = props.enhancedResult;
			if (props.enhancedResult) {
				updates.mode = 'enhanced';
			}
		}
		if (props.autoFixResult !== undefined) {
			updates.autoFixResult = props.autoFixResult;
			if (props.autoFixResult) {
				updates.mode = 'auto-fix';
			}
		}
		if (props.serverName !== undefined) updates.serverName = props.serverName;
		if (props.loading !== undefined) updates.loading = props.loading;
		if (props.mode !== undefined) updates.mode = props.mode;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}
		if (props.onretry !== undefined) {
			this.onretry = props.onretry;
		}
		if (props.onquicktest !== undefined) {
			this.onquicktest = props.onquicktest;
		}
		if (props.onenhanced !== undefined) {
			this.onenhanced = props.onenhanced;
		}
		if (props.onautofix !== undefined) {
			this.onautofix = props.onautofix;
		}

		this.updateState(updates);
	}

	public handleClose(): void {
		this.updateState({ open: false });
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

	public handleEnhancedTroubleshoot(): void {
		this.updateState({ loading: true, mode: 'enhanced' });
		this.onenhanced?.();
	}

	public handleAutoFix(): void {
		this.updateState({ loading: true, mode: 'auto-fix' });
		this.onautofix?.();
	}

	public setCurrentView(view: 'diagnostics' | 'analysis' | 'recovery' | 'suggestions'): void {
		this.updateState({ currentView: view });
	}

	public toggleAdvanced(): void {
		this.updateState({ showAdvanced: !this.state.showAdvanced });
	}

	public isOpen(): boolean {
		return this.state.open;
	}

	public hasResult(): boolean {
		return (
			this.state.result !== null ||
			this.state.enhancedResult !== null ||
			this.state.autoFixResult !== null
		);
	}

	public isLoading(): boolean {
		return this.state.loading;
	}

	public isSuccess(): boolean {
		if (this.state.mode === 'enhanced' && this.state.enhancedResult) {
			return this.state.enhancedResult.success;
		}
		if (this.state.mode === 'auto-fix' && this.state.autoFixResult) {
			return this.state.autoFixResult.success;
		}
		return this.state.result?.success === true;
	}

	public hasErrors(): boolean {
		if (this.state.mode === 'enhanced' && this.state.enhancedResult) {
			return this.state.enhancedResult.has_errors;
		}
		if (this.state.mode === 'auto-fix' && this.state.autoFixResult) {
			return !this.state.autoFixResult.success;
		}
		return this.state.result?.has_errors === true;
	}

	public hasWarnings(): boolean {
		if (this.state.mode === 'enhanced' && this.state.enhancedResult) {
			return this.state.enhancedResult.has_warnings;
		}
		return this.state.result?.has_warnings === true;
	}

	public getResult(): TroubleshootResult | null {
		return this.state.result;
	}

	public getEnhancedResult(): EnhancedTroubleshootResult | null {
		return this.state.enhancedResult;
	}

	public getAutoFixResult(): AutoFixResult | null {
		return this.state.autoFixResult;
	}

	public getCurrentResult():
		| TroubleshootResult
		| EnhancedTroubleshootResult
		| AutoFixResult
		| null {
		if (this.state.mode === 'enhanced') return this.state.enhancedResult;
		if (this.state.mode === 'auto-fix') return this.state.autoFixResult;
		return this.state.result;
	}

	public getServerName(): string {
		return this.state.serverName;
	}

	public getTitle(): string {
		if (this.state.loading) {
			switch (this.state.mode) {
				case 'enhanced':
					return 'Running Enhanced Diagnostics...';
				case 'auto-fix':
					return 'Auto-Fixing Issues...';
				default:
					return 'Running Diagnostics...';
			}
		}

		switch (this.state.mode) {
			case 'enhanced':
				return 'Enhanced SSH Troubleshooting';
			case 'auto-fix':
				return 'SSH Auto-Fix Results';
			default:
				return 'SSH Connection Troubleshooting';
		}
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
		const currentResult = this.getCurrentResult();
		if (!this.state.testStartTime || !currentResult?.timestamp) return 0;
		const endTime = new Date(currentResult.timestamp).getTime();
		return Math.round((endTime - this.state.testStartTime) / 1000);
	}

	public getFormattedTimestamp(): string {
		const currentResult = this.getCurrentResult();
		if (!currentResult?.timestamp) return '';
		return new Date(currentResult.timestamp).toLocaleTimeString();
	}

	public isConnectionRefusedDetected(): boolean {
		const currentResult = this.getCurrentResult();
		if (!currentResult || !('diagnostics' in currentResult)) return false;

		return (
			currentResult.diagnostics.some(
				(diag) =>
					diag.step === 'network_connectivity' &&
					diag.status === 'error' &&
					(diag.details?.includes('connection refused') ||
						diag.message?.includes('connection refused'))
			) || currentResult.diagnostics.some((diag) => diag.step === 'connection_refused_analysis')
		);
	}

	public isFail2banIssueDetected(): boolean {
		const currentResult = this.getCurrentResult();
		if (!currentResult || !('diagnostics' in currentResult)) return false;

		return currentResult.diagnostics.some(
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
		const currentResult = this.getCurrentResult();
		if (!currentResult || !('diagnostics' in currentResult)) {
			return { network: false, auth: false, permission: false, fail2ban: false };
		}

		const categories = {
			network: false,
			auth: false,
			permission: false,
			fail2ban: false
		};

		for (const diag of currentResult.diagnostics) {
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
		const currentResult = this.getCurrentResult();

		if (this.state.mode === 'auto-fix' && this.state.autoFixResult) {
			const { fixes_applied } = this.state.autoFixResult;
			const successCount = this.state.autoFixResult.fixes.filter(
				(f) => f.status === 'success'
			).length;
			return `${successCount}/${fixes_applied} fixes applied successfully`;
		}

		if (!currentResult || !('success_count' in currentResult)) return '';

		const { success_count, warning_count, error_count } = currentResult;
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
		const currentResult = this.getCurrentResult();
		if (!currentResult || !('diagnostics' in currentResult)) return [];
		return currentResult.diagnostics.filter((diag) => diag.status === status);
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
		if (this.isSuccess()) return '';

		// For auto-fix mode, show fix-specific message
		if (this.state.mode === 'auto-fix' && this.state.autoFixResult) {
			return this.state.autoFixResult.summary;
		}

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
		return this.hasResult() && !this.state.loading && this.state.mode !== 'auto-fix';
	}

	public shouldShowRetry(): boolean {
		return this.hasResult() && (this.hasErrors() || this.hasWarnings()) && !this.state.loading;
	}

	public shouldShowEnhanced(): boolean {
		return !this.state.loading && this.state.mode === 'basic';
	}

	public shouldShowAutoFix(): boolean {
		return (
			!this.state.loading &&
			this.hasResult() &&
			this.hasErrors() &&
			this.state.mode !== 'auto-fix' &&
			this.getCanAutoFix()
		);
	}

	public isCloseable(): boolean {
		return !this.state.loading;
	}

	// Enhanced features

	public getMode(): 'basic' | 'enhanced' | 'auto-fix' {
		return this.state.mode;
	}

	public getCurrentView(): 'diagnostics' | 'analysis' | 'recovery' | 'suggestions' {
		return this.state.currentView;
	}

	public getShowAdvanced(): boolean {
		return this.state.showAdvanced;
	}

	public getAnalysis(): EnhancedTroubleshootResult['analysis'] | null {
		return this.state.enhancedResult?.analysis || null;
	}

	public getRecoveryPlan(): EnhancedTroubleshootResult['recovery_plan'] | null {
		return this.state.enhancedResult?.recovery_plan || null;
	}

	public getActionableSuggestions(): ActionableSuggestion[] {
		return this.state.enhancedResult?.actionable_suggestions || [];
	}

	public getEstimatedDuration(): string {
		return this.state.enhancedResult?.estimated_duration || '';
	}

	public getRequiredAccess(): string[] {
		return this.state.enhancedResult?.requires_access || [];
	}

	public getCanAutoFix(): boolean {
		if (this.state.enhancedResult) {
			return this.state.enhancedResult.auto_fix_available;
		}
		return this.state.result?.can_auto_fix === true;
	}

	public getSeverity(): string {
		const currentResult = this.getCurrentResult();
		if ('severity' in currentResult!) {
			return currentResult.severity || 'medium';
		}
		return 'medium';
	}

	public getClientIP(): string {
		const currentResult = this.getCurrentResult();
		if ('client_ip' in currentResult!) {
			return currentResult.client_ip || 'unknown';
		}
		return 'unknown';
	}

	public getConnectionTime(): number | null {
		const currentResult = this.getCurrentResult();
		if ('connection_time' in currentResult!) {
			return currentResult.connection_time || null;
		}
		return null;
	}

	public getPrioritySteps(): RecoveryStep[] {
		const plan = this.getRecoveryPlan();
		return plan?.steps?.filter((step: RecoveryStep) => step.required) || [];
	}

	public getOptionalSteps(): RecoveryStep[] {
		const plan = this.getRecoveryPlan();
		return plan?.steps?.filter((step: RecoveryStep) => !step.required) || [];
	}

	public getCriticalIssues(): string[] {
		const plan = this.getRecoveryPlan();
		return plan?.critical_issues || [];
	}

	public getSuccessProbability(): number {
		const plan = this.getRecoveryPlan();
		return plan?.success_probability || 0.5;
	}

	public formatDuration(ms?: number | null): string {
		if (!ms) return '';
		if (ms < 1000) return `${ms}ms`;
		const seconds = Math.round(ms / 1000);
		if (seconds < 60) return `${seconds}s`;
		const minutes = Math.floor(seconds / 60);
		const remainingSeconds = seconds % 60;
		return `${minutes}m ${remainingSeconds}s`;
	}

	public getPatternDescription(): string {
		const analysis = this.getAnalysis();
		return analysis?.description || 'No specific pattern detected';
	}

	public getImmediateAction(): string | null {
		const analysis = this.getAnalysis();
		return analysis?.immediate_action || null;
	}

	public getSuggestionsByPriority(
		priority: 'critical' | 'high' | 'medium' | 'low'
	): ActionableSuggestion[] {
		return this.getActionableSuggestions().filter((s) => s.priority === priority);
	}

	public getAutomatedSuggestions(): ActionableSuggestion[] {
		return this.getActionableSuggestions().filter((s) => s.automated);
	}

	public getManualSuggestions(): ActionableSuggestion[] {
		return this.getActionableSuggestions().filter((s) => !s.automated);
	}

	// Type guard methods
	private hasBasicSuggestions(
		result:
			| TroubleshootResult
			| EnhancedTroubleshootResult
			| AutoFixResult
			| QuickTroubleshootResult
			| null
	): result is TroubleshootResult & { suggestions: string[] } {
		return !!(result && 'suggestions' in result && Array.isArray(result.suggestions));
	}

	private hasDiagnostics(
		result:
			| TroubleshootResult
			| EnhancedTroubleshootResult
			| AutoFixResult
			| QuickTroubleshootResult
			| null
	): result is TroubleshootResult & { diagnostics: ConnectionDiagnostic[] } {
		return !!(result && 'diagnostics' in result && Array.isArray(result.diagnostics));
	}

	public getBasicSuggestions(): string[] {
		const result = this.getCurrentResult();
		return this.hasBasicSuggestions(result) ? result.suggestions : [];
	}

	public getDiagnostics(): ConnectionDiagnostic[] {
		const result = this.getCurrentResult();
		if (this.getMode() === 'auto-fix') {
			return this.getAutoFixResult()?.fixes || [];
		}
		return this.hasDiagnostics(result) ? result.diagnostics : [];
	}

	// Null-safe accessor methods
	public getAnalysisConfidence(): number {
		const analysis = this.getAnalysis();
		return analysis?.confidence ?? 0;
	}

	public getAnalysisCategory(): string {
		const analysis = this.getAnalysis();
		return analysis?.category ?? '';
	}

	public getAnalysisAutoFixable(): boolean {
		const analysis = this.getAnalysis();
		return analysis?.auto_fixable ?? false;
	}

	public getRecoveryPlanEstimatedTime(): string {
		const plan = this.getRecoveryPlan();
		return plan?.estimated_time ?? '';
	}
}
