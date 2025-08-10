export function getStatusColor(status: string): string {
	switch (status.toLowerCase()) {
		case 'online':
		case 'success':
		case 'completed':
			return 'green';
		case 'offline':
		case 'failed':
		case 'error':
			return 'red';
		case 'running':
		case 'pending':
		case 'starting':
			return 'yellow';
		default:
			return 'gray';
	}
}

export function getStatusIcon(status: string): string {
	switch (status.toLowerCase()) {
		case 'online':
		case 'success':
		case 'completed':
			return '✅';
		case 'offline':
		case 'failed':
		case 'error':
			return '❌';
		case 'running':
		case 'pending':
		case 'starting':
			return '🔄';
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
