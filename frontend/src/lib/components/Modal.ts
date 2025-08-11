export interface ModalProps {
	open?: boolean;
	title?: string;
	size?: 'sm' | 'md' | 'lg' | 'xl';
	closeable?: boolean;
	onclose?: () => void;
}

export interface ModalState {
	open: boolean;
	title: string;
	size: 'sm' | 'md' | 'lg' | 'xl';
	closeable: boolean;
}

export class ModalLogic {
	private state: ModalState;
	private stateUpdateCallback?: (state: ModalState) => void;
	private onclose?: () => void;
	private keydownHandler?: (event: KeyboardEvent) => void;

	// Size classes configuration
	public readonly sizeClasses = {
		sm: 'max-w-md',
		md: 'max-w-lg',
		lg: 'max-w-2xl',
		xl: 'max-w-4xl'
	};

	constructor(props: ModalProps = {}) {
		this.state = {
			open: props.open ?? false,
			title: props.title ?? '',
			size: props.size ?? 'md',
			closeable: props.closeable ?? true
		};
		this.onclose = props.onclose;
		this.setupKeydownHandler();
	}

	public getState(): ModalState {
		return this.state;
	}

	public onStateUpdate(callback: (state: ModalState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<ModalState>): void {
		const previousOpen = this.state.open;
		this.state = { ...this.state, ...updates };

		// Handle body scroll when open state changes
		if (previousOpen !== this.state.open) {
			this.handleBodyScroll(this.state.open);
		}

		this.stateUpdateCallback?.(this.state);
	}

	private setupKeydownHandler(): void {
		this.keydownHandler = (event: KeyboardEvent) => {
			if (event.key === 'Escape' && this.state.closeable && this.state.open) {
				this.close();
			}
		};
	}

	public getKeydownHandler(): (event: KeyboardEvent) => void {
		return this.keydownHandler!;
	}

	private handleBodyScroll(open: boolean): void {
		if (typeof document !== 'undefined') {
			if (open) {
				document.body.style.overflow = 'hidden';
			} else {
				document.body.style.overflow = '';
			}
		}
	}

	public open(): void {
		this.updateState({ open: true });
	}

	public close(): void {
		if (this.state.closeable) {
			this.updateState({ open: false });
			this.onclose?.();
		}
	}

	public updateProps(props: Partial<ModalProps>): void {
		const updates: Partial<ModalState> = {};

		if (props.open !== undefined) updates.open = props.open;
		if (props.title !== undefined) updates.title = props.title;
		if (props.size !== undefined) updates.size = props.size;
		if (props.closeable !== undefined) updates.closeable = props.closeable;

		if (props.onclose !== undefined) {
			this.onclose = props.onclose;
		}

		this.updateState(updates);
	}

	public getSizeClass(): string {
		return this.sizeClasses[this.state.size];
	}

	public isOpen(): boolean {
		return this.state.open;
	}

	public isCloseable(): boolean {
		return this.state.closeable;
	}

	public getTitle(): string {
		return this.state.title;
	}

	public getSize(): 'sm' | 'md' | 'lg' | 'xl' {
		return this.state.size;
	}

	// Cleanup method to restore body scroll
	public cleanup(): void {
		if (typeof document !== 'undefined') {
			document.body.style.overflow = '';
		}
	}

	// Handle backdrop click
	public handleBackdropClick(event: MouseEvent): void {
		// Only close if clicking the backdrop itself, not the modal content
		if (this.state.closeable && event.target === event.currentTarget) {
			this.close();
		}
	}
}
