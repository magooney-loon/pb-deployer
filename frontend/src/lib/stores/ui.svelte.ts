export type ModalName =
	| 'app-create'
	| 'app-manage'
	| 'app-delete'
	| 'server-create'
	| 'server-delete'
	| 'server-troubleshoot'
	| 'server-terminal'
	| 'deployment-create'
	| 'deployment-view'
	| 'logs-view'
	| 'proxy-migrate'
	| 'deploy-confirm';

type ToastType = 'success' | 'error' | 'warning' | 'info';

function createUIStore() {
	let modal = $state<{ name: ModalName; payload?: unknown } | null>(null);
	let toasts = $state<Array<{ id: string; type: ToastType; message: string }>>([]);

	function open(name: ModalName, payload?: unknown) {
		modal = { name, payload };
	}

	function close() {
		modal = null;
	}

	function toast(type: ToastType, message: string, ttl = 5000) {
		const id = crypto.randomUUID();
		toasts = [...toasts, { id, type, message }];
		setTimeout(() => {
			toasts = toasts.filter((t) => t.id !== id);
		}, ttl);
	}

	function dismiss(id: string) {
		toasts = toasts.filter((t) => t.id !== id);
	}

	return {
		get modal() {
			return modal;
		},
		get toasts() {
			return toasts;
		},
		open,
		close,
		toast,
		dismiss
	};
}

export const ui = createUIStore();
