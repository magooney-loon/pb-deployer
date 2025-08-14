import type { Server, App } from '$lib/api/index.js';

// Common status badge variant type
export type StatusBadgeVariant = 'success' | 'warning' | 'error' | 'info' | 'gray';

// Status badge result interface
export interface StatusBadgeResult {
	text: string;
	variant: StatusBadgeVariant;
}

/**
 * Get consistent server status badge information
 */
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

/**
 * Get consistent app status badge information
 */
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

/**
 * Get consistent API status badge information
 */
export function getApiStatusBadge(status: 'online' | 'offline' | 'checking'): StatusBadgeResult {
	switch (status) {
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
		case 'checking':
		default:
			return {
				text: 'Checking...',
				variant: 'warning'
			};
	}
}

/**
 * Get status icon for apps (emoji representation)
 */
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

/**
 * Format timestamp consistently across the app
 */
export function formatTimestamp(timestamp: string): string {
	try {
		return new Date(timestamp).toLocaleString();
	} catch {
		return timestamp;
	}
}
