import type { Server, App } from '$lib/api/index.js';

export type StatusBadgeVariant = 'success' | 'warning' | 'error' | 'info' | 'gray';

export interface StatusBadgeResult {
	text: string;
	variant: StatusBadgeVariant;
}

export function getServerStatusBadge(server: Server): StatusBadgeResult {
	if (server.setup_complete && server.security_locked) {
		return {
			text: 'Secured',
			variant: 'info'
		};
	} else if (server.setup_complete) {
		return {
			text: 'Ready',
			variant: 'success'
		};
	} else {
		return {
			text: 'Not Setup',
			variant: 'warning'
		};
	}
}

export function getAppStatusBadge(app: App): StatusBadgeResult {
	switch (app.status) {
		case 'online':
			return {
				text: 'Online',
				variant: 'success'
			};
		case 'offline':
			return {
				text: 'Offline',
				variant: 'error'
			};
		default:
			return {
				text: 'Unknown',
				variant: 'gray'
			};
	}
}

export function getAppStatusIcon(status: string): string {
	switch (status) {
		case 'online':
			return '🟢';
		case 'offline':
			return '🔴';
		default:
			return '⚪';
	}
}

export function formatTimestamp(timestamp: string): string {
	try {
		return new Date(timestamp).toLocaleString();
	} catch {
		return timestamp;
	}
}
