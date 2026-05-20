import { api } from './client.svelte.js';
import type { Version } from '$lib/api/index.js';

function createVersionsStore() {
	let versions = $state<Version[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialized = $state(false);

	function byApp(appId: string) {
		return versions.filter((v) => v.app_id === appId);
	}

	function byId(id: string) {
		return versions.find((v) => v.id === id) ?? null;
	}

	async function load() {
		if (loading) return;
		loading = true;
		error = null;
		try {
			const { versions: fresh } = await api.versions.getVersions();
			versions = fresh;
			initialized = true;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	function _onRealtime(action: string, record: Version) {
		if (action === 'delete') {
			versions = versions.filter((v) => v.id !== record.id);
		} else if (action === 'create') {
			if (!versions.find((v) => v.id === record.id)) {
				versions = [...versions, record];
			}
		} else {
			versions = versions.map((v) => (v.id === record.id ? { ...v, ...record } : v));
		}
	}

	return {
		get versions() {
			return versions;
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
		byApp,
		byId,
		load,
		_onRealtime,
		clearError() {
			error = null;
		}
	};
}

export const versionsStore = createVersionsStore();
