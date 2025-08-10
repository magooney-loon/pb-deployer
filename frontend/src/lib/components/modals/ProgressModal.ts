import type { SetupStep } from '../../api.js';

export interface ProgressModalProps {
	show?: boolean;
	title: string;
	progress: SetupStep[];
	onClose: () => void;
	loading?: boolean;
	operationInProgress?: boolean;
}

export interface ProgressModalState {
	show: boolean;
	title: string;
	progress: SetupStep[];
	loading: boolean;
	operationInProgress: boolean;
}

export class ProgressModalLogic {
	private state: ProgressModalState;
	private stateUpdateCallback?: (state: ProgressModalState) => void;
	private onClose: () => void;

	constructor(props: ProgressModalProps) {
		this.state = {
			show: props.show ?? false,
			title: props.title,
			progress: props.progress || [],
			loading: props.loading ?? false,
			operationInProgress: props.operationInProgress ?? false
		};
		this.onClose = props.onClose;
	}

	public getState(): ProgressModalState {
		return this.state;
	}

	public onStateUpdate(callback: (state: ProgressModalState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<ProgressModalState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public updateProps(props: Partial<ProgressModalProps>): void {
		const updates: Partial<ProgressModalState> = {};

		if (props.show !== undefined) updates.show = props.show;
		if (props.title !== undefined) updates.title = props.title;
		if (props.progress !== undefined) updates.progress = props.progress;
		if (props.loading !== undefined) updates.loading = props.loading;
		if (props.operationInProgress !== undefined)
			updates.operationInProgress = props.operationInProgress;

		if (props.onClose !== undefined) {
			this.onClose = props.onClose;
		}

		this.updateState(updates);
	}

	public show(): void {
		this.updateState({ show: true });
	}

	public hide(): void {
		this.updateState({ show: false });
	}

	public setProgress(progress: SetupStep[]): void {
		this.updateState({ progress });
	}

	public setLoading(loading: boolean): void {
		this.updateState({ loading });
	}

	public setOperationInProgress(operationInProgress: boolean): void {
		this.updateState({ operationInProgress });
	}

	public getProgressStepIcon(status: string): string {
		switch (status) {
			case 'running':
				return '🔄';
			case 'success':
				return '✅';
			case 'failed':
				return '❌';
			default:
				return '⏳';
		}
	}

	public getProgressStepColor(status: string): string {
		switch (status) {
			case 'running':
				return 'text-blue-600 dark:text-blue-400';
			case 'success':
				return 'text-green-600 dark:text-green-400';
			case 'failed':
				return 'text-red-600 dark:text-red-400';
			default:
				return 'text-gray-600 dark:text-gray-400';
		}
	}

	public getOverallProgress(): number {
		if (this.state.progress.length === 0) return 0;
		const latestStep = this.state.progress[this.state.progress.length - 1];
		return latestStep.progress_pct || 0;
	}

	public isComplete(): boolean {
		if (this.state.progress.length === 0) return false;
		const latestStep = this.state.progress[this.state.progress.length - 1];
		return latestStep.step === 'complete';
	}

	public isSuccess(): boolean {
		if (!this.isComplete()) return false;
		const latestStep = this.state.progress[this.state.progress.length - 1];
		return latestStep.status === 'success';
	}

	public isFailed(): boolean {
		if (this.state.progress.length === 0) return false;
		// Check if any step has failed, not just the latest
		return this.state.progress.some((step) => step.status === 'failed');
	}

	public isInProgress(): boolean {
		// Use explicit operation status if provided
		if (this.state.operationInProgress) return true;
		if (this.state.loading) return true;

		// If no progress data and not explicitly loading, not in progress
		if (this.state.progress.length === 0) return false;

		const latestStep = this.state.progress[this.state.progress.length - 1];

		// If operation has completed successfully, it's not in progress
		if (latestStep.step === 'complete') return false;

		// If any step has failed, operation is not in progress anymore
		if (this.state.progress.some((step) => step.status === 'failed')) return false;

		// If latest step is running, it's in progress
		return latestStep.status === 'running';
	}

	public handleClose(): void {
		if (!this.isInProgress()) {
			this.onClose();
		}
	}

	public isCloseable(): boolean {
		// Always allow closing if operation has completed (success or failure)
		if (this.isComplete() || this.isFailed()) return true;

		// Otherwise, only closeable if not in progress
		return !this.isInProgress();
	}

	public getProgressBarColor(): string {
		if (this.isSuccess()) {
			return 'bg-green-500';
		} else if (this.isFailed()) {
			return 'bg-red-500';
		} else {
			return 'bg-blue-500';
		}
	}

	public getStepBorderColor(step: SetupStep): string {
		if (step.status === 'failed') {
			return 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20';
		} else if (step.status === 'success') {
			return 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20';
		} else {
			return 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20';
		}
	}

	public formatTimestamp(timestamp: string): string {
		return new Date(timestamp).toLocaleTimeString();
	}

	public hasProgress(): boolean {
		return this.state.progress.length > 0;
	}

	public getTitle(): string {
		return this.state.title;
	}

	public isOpen(): boolean {
		return this.state.show;
	}

	public getProgress(): SetupStep[] {
		return this.state.progress;
	}

	public isLoading(): boolean {
		return this.state.loading;
	}

	public getFooterButtonClass(): string {
		if (this.isComplete()) {
			return this.isSuccess()
				? 'bg-green-600 text-white hover:bg-green-700'
				: 'bg-red-600 text-white hover:bg-red-700';
		} else {
			return 'border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600';
		}
	}

	public getFooterButtonText(): string {
		if (this.isComplete()) {
			return this.isSuccess() ? 'Done' : 'Close';
		} else if (this.isInProgress()) {
			return 'Operation in Progress...';
		} else {
			return 'Close';
		}
	}

	public isFooterButtonDisabled(): boolean {
		return this.isInProgress();
	}
}
