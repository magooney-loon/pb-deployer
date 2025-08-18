import type { Server, App } from '$lib/api/index.js';
import type { Deployment } from '$lib/api/deployment/types.js';

export type StatusBadgeVariant = 'success' | 'warning' | 'error' | 'info' | 'gray' | 'update';

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

export function getAppStatusBadge(app: App, latestVersion?: string): StatusBadgeResult {
	// Check for version update first
	if (
		latestVersion &&
		app.current_version &&
		hasUpdateAvailable(app.current_version, latestVersion)
	) {
		return {
			text: 'Update Available',
			variant: 'update'
		};
	}

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

export function hasUpdateAvailable(currentVersion: string, latestVersion: string): boolean {
	if (!currentVersion || !latestVersion) {
		return false;
	}

	const current = parseVersion(currentVersion);
	const latest = parseVersion(latestVersion);

	// Compare major.minor.patch
	if (latest.major > current.major) return true;
	if (latest.major < current.major) return false;

	if (latest.minor > current.minor) return true;
	if (latest.minor < current.minor) return false;

	if (latest.patch > current.patch) return true;

	return false;
}

function parseVersion(version: string): { major: number; minor: number; patch: number } {
	// Remove 'v' prefix if present and clean up
	const cleaned = version.replace(/^v/, '').trim();
	const parts = cleaned.split('.').map((part) => {
		const num = parseInt(part, 10);
		return isNaN(num) ? 0 : num;
	});

	return {
		major: parts[0] || 0,
		minor: parts[1] || 0,
		patch: parts[2] || 0
	};
}

export function getAppUpdateStatus(app: App, latestVersion?: string): StatusBadgeResult | null {
	if (!latestVersion || !app.current_version) {
		return null;
	}

	if (hasUpdateAvailable(app.current_version, latestVersion)) {
		return {
			text: `v${latestVersion} Available`,
			variant: 'update'
		};
	}

	return null;
}

export function getAppStatusIcon(status: string): string {
	switch (status) {
		case 'online':
			return '';
		case 'offline':
			return '';
		default:
			return '';
	}
}

export function getDeploymentStatusBadge(deployment: Deployment): StatusBadgeResult {
	const status = deployment.status.toLowerCase();
	switch (status) {
		case 'completed':
		case 'success':
			return {
				text: 'Completed',
				variant: 'success'
			};
		case 'pending':
			return {
				text: 'Pending',
				variant: 'warning'
			};
		case 'running':
		case 'in_progress':
			return {
				text: 'Running',
				variant: 'info'
			};
		case 'failed':
		case 'error':
			return {
				text: 'Failed',
				variant: 'error'
			};
		default:
			return {
				text: 'Unknown',
				variant: 'gray'
			};
	}
}

export function formatTimestamp(timestamp: string): string {
	try {
		return new Date(timestamp).toLocaleString();
	} catch {
		return timestamp;
	}
}
