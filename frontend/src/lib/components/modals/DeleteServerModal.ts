import type { Server, App } from '../../api.js';

export interface DeleteServerModalProps {
	open?: boolean;
	server?: Server | null;
	apps?: App[];
	loading?: boolean;
	onclose?: () => void;
	onconfirm?: (serverId: string) => void;
}

export interface DeleteServerModalState {
	open: boolean;
	server: Server | null;
	apps: App[];
	loading: boolean;
	confirmationText: string;
}

export class DeleteServerModalLogic {
	private state: DeleteServerModalState;
	private stateUpdateCallback?: (state: DeleteServerModalState) => void;
	private onclose?: () => void;
	private onconfirm?: (serverId: string) => void;

	constructor(props: DeleteServerModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			server: props.server ?? null,
			apps: props.apps ?? [],
			loading: props.loading ?? false,
			confirmationText: ''
		};
		this.onclose = props.onclose;
		this.onconfirm = props.onconfirm;
	}

	public getState(): DeleteServerModalState {
		return this.state;
	}

	public onStateUpdate(callback: (state: DeleteServerModalState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<DeleteServerModalState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public updateProps(props: Partial<DeleteServerModalProps>): void {
		const updates: Partial<DeleteServerModalState> = {};

		if (props.open !== undefined) updates.open = props.open;
		if (props.server !== undefined) updates.server = props.server;
		if (props.apps !== undefined) updates.apps = props.apps;
		if (props.loading !== undefined) updates.loading = props.loading;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}

		if (props.onconfirm !== undefined) {
			this.onconfirm = props.onconfirm;
		}

		this.updateState(updates);
	}

	public handleClose(): void {
		if (!this.state.loading) {
			this.onclose?.();
		}
	}

	public handleConfirm(): void {
		if (this.state.server && !this.state.loading && this.isConfirmationValid()) {
			this.onconfirm?.(this.state.server.id);
		}
	}

	public updateConfirmationText(text: string): void {
		this.updateState({ confirmationText: text });
	}

	public resetConfirmationText(): void {
		this.updateState({ confirmationText: '' });
	}

	public isOpen(): boolean {
		return this.state.open;
	}

	public isLoading(): boolean {
		return this.state.loading;
	}

	public hasServer(): boolean {
		return this.state.server !== null;
	}

	public getServer(): Server | null {
		return this.state.server;
	}

	public getConfirmationText(): string {
		return this.state.confirmationText;
	}

	public isConfirmationValid(): boolean {
		return this.state.server ? this.state.confirmationText === this.state.server.name : false;
	}

	public getAssociatedApps(): App[] {
		return this.state.server
			? this.state.apps.filter((app) => app.server_id === this.state.server!.id)
			: [];
	}

	public hasAssociatedApps(): boolean {
		return this.getAssociatedApps().length > 0;
	}

	public getAssociatedAppsCount(): number {
		return this.getAssociatedApps().length;
	}

	public getServerStatus(): {
		text: string;
		colorClass: string;
	} {
		if (!this.state.server) {
			return { text: 'Unknown', colorClass: 'bg-gray-100 text-gray-800' };
		}

		const { server } = this.state;

		if (server.setup_complete && server.security_locked) {
			return {
				text: 'Ready',
				colorClass:
					'rounded bg-green-100 px-2 py-1 text-xs text-green-800 dark:bg-green-900 dark:text-green-200'
			};
		} else if (server.setup_complete) {
			return {
				text: 'Setup Complete',
				colorClass:
					'rounded bg-yellow-100 px-2 py-1 text-xs text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
			};
		} else {
			return {
				text: 'Not Setup',
				colorClass:
					'rounded bg-red-100 px-2 py-1 text-xs text-red-800 dark:bg-red-900 dark:text-red-200'
			};
		}
	}

	public isCloseable(): boolean {
		return !this.state.loading;
	}

	public isConfirmButtonEnabled(): boolean {
		return !this.state.loading && this.state.server !== null && this.isConfirmationValid();
	}

	public getConfirmButtonText(): string {
		return this.state.loading ? 'Deleting...' : 'Delete Server';
	}
}
