import { api } from './client.svelte.js';
import type { Server, ServerRequest } from '$lib/api/index.js';
import type { ValidationResponse } from '$lib/api/index.js';

function createServersStore() {
	let servers = $state<Server[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialized = $state(false);

	let setupInProgress = $state<Record<string, boolean>>({});
	let securityInProgress = $state<Record<string, boolean>>({});
	let troubleshootInProgress = $state<Record<string, boolean>>({});
	let troubleshootResults = $state<Record<string, ValidationResponse>>({});

	function byId(id: string) {
		return servers.find((s) => s.id === id) ?? null;
	}

	async function load() {
		if (loading) return;
		loading = true;
		error = null;
		try {
			const { servers: fresh } = await api.servers.getServers();
			servers = fresh;
			initialized = true;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function create(data: ServerRequest) {
		const server = await api.servers.createServer(data);
		servers = [...servers, server];
		return server;
	}

	async function remove(id: string) {
		const prev = [...servers];
		servers = servers.filter((s) => s.id !== id);
		try {
			await api.servers.deleteServer(id);
		} catch (e) {
			servers = prev;
			throw e;
		}
	}

	async function setup(serverId: string) {
		setupInProgress = { ...setupInProgress, [serverId]: true };
		try {
			await api.setup.setupServerFromRecord(serverId);
			servers = servers.map((s) => (s.id === serverId ? { ...s, setup_complete: true } : s));
		} finally {
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
			const { [serverId]: _, ...rest } = setupInProgress;
			setupInProgress = rest;
		}
	}

	async function secure(serverId: string) {
		securityInProgress = { ...securityInProgress, [serverId]: true };
		try {
			await api.setup.secureServerFromRecord(serverId);
			servers = servers.map((s) => (s.id === serverId ? { ...s, security_locked: true } : s));
		} finally {
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
			const { [serverId]: _, ...rest } = securityInProgress;
			securityInProgress = rest;
		}
	}

	async function troubleshoot(serverId: string): Promise<ValidationResponse> {
		troubleshootInProgress = { ...troubleshootInProgress, [serverId]: true };
		try {
			const results = await api.setup.validateServerFromRecord(serverId);
			troubleshootResults = { ...troubleshootResults, [serverId]: results };
			return results;
		} finally {
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
			const { [serverId]: _, ...rest } = troubleshootInProgress;
			troubleshootInProgress = rest;
		}
	}

	function _onRealtime(action: string, record: Server) {
		if (action === 'delete') {
			servers = servers.filter((s) => s.id !== record.id);
		} else if (action === 'create') {
			if (!servers.find((s) => s.id === record.id)) {
				servers = [...servers, record];
			}
		} else {
			servers = servers.map((s) => (s.id === record.id ? { ...s, ...record } : s));
		}
	}

	return {
		get servers() {
			return servers;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		get initialized() {
			return initialized;
		},
		get setupInProgress() {
			return setupInProgress;
		},
		get securityInProgress() {
			return securityInProgress;
		},
		get troubleshootInProgress() {
			return troubleshootInProgress;
		},
		get troubleshootResults() {
			return troubleshootResults;
		},
		byId,
		load,
		create,
		remove,
		setup,
		secure,
		troubleshoot,
		_onRealtime,
		clearError() {
			error = null;
		}
	};
}

export const serversStore = createServersStore();
