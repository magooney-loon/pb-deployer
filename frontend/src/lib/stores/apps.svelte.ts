import { api } from './client.svelte.js';
import type { App } from '$lib/api/index.js';

function createAppsStore() {
	let apps = $state<App[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialized = $state(false);

	function byServer(serverId: string) {
		return apps.filter((a) => a.server_id === serverId);
	}

	function byId(id: string) {
		return apps.find((a) => a.id === id) ?? null;
	}

	async function load() {
		if (loading) return;
		loading = true;
		error = null;
		try {
			const { apps: fresh } = await api.apps.getAppsWithLatestVersions();
			apps = fresh;
			initialized = true;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function create(data: {
		name: string;
		server_id: string;
		domain: string;
		version_number?: string;
		version_notes?: string;
		initialZip?: File;
	}) {
		const pb = api.getPocketBase();
		const newApp = (await pb.send('/api/apps', {
			method: 'POST',
			body: { name: data.name, server_id: data.server_id, domain: data.domain }
		})) as { id: string; http_port: number };

		if (data.version_number) {
			await api.versions.createVersion({
				app_id: newApp.id,
				version_number: data.version_number,
				notes: data.version_notes || '',
				deployment_zip: data.initialZip
			});
		}

		await load();
		return newApp;
	}

	async function remove(id: string) {
		const prev = [...apps];
		apps = apps.filter((a) => a.id !== id);
		try {
			await api.apps.deleteApp(id);
		} catch (e) {
			apps = prev;
			throw e;
		}
	}

	async function migrateProxy(id: string) {
		const result = await api.apps.migrateProxy(id);
		apps = apps.map((a) =>
			a.id === id ? { ...a, http_port: result.http_port, status: 'online' as const } : a
		);
		return result;
	}

	function _onRealtime(action: string, record: App) {
		if (action === 'delete') {
			apps = apps.filter((a) => a.id !== record.id);
		} else if (action === 'create') {
			if (!apps.find((a) => a.id === record.id)) {
				apps = [record, ...apps];
			}
		} else {
			const exists = apps.find((a) => a.id === record.id);
			if (exists) {
				apps = apps.map((a) => (a.id === record.id ? { ...a, ...record } : a));
			} else {
				apps = [record, ...apps];
			}
		}
	}

	return {
		get apps() {
			return apps;
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
		byServer,
		byId,
		load,
		create,
		remove,
		migrateProxy,
		_onRealtime,
		clearError() {
			error = null;
		}
	};
}

export const appsStore = createAppsStore();
