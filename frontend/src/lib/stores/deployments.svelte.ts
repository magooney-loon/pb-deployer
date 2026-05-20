import { api } from './client.svelte.js';
import type { Deployment } from '$lib/api/index.js';

function createDeploymentsStore() {
	let deployments = $state<Deployment[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let initialized = $state(false);
	let deployingIds = $state<string[]>([]);

	function byApp(appId: string) {
		return deployments.filter((d) => d.app_id === appId);
	}

	function byId(id: string) {
		return deployments.find((d) => d.id === id) ?? null;
	}

	function isDeploying(id: string) {
		return deployingIds.includes(id);
	}

	async function load() {
		if (loading) return;
		loading = true;
		error = null;
		try {
			const { deployments: fresh } = await api.deployments.getDeployments();
			deployments = fresh;
			initialized = true;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	async function create(data: { app_id: string; version_id: string }) {
		await api.deployments.createDeployment({ ...data, status: 'pending' });
		await load();
	}

	async function remove(id: string) {
		const prev = [...deployments];
		deployments = deployments.filter((d) => d.id !== id);
		try {
			await api.deployments.deleteDeployment(id);
		} catch (e) {
			deployments = prev;
			throw e;
		}
	}

	async function deploy(
		deploymentId: string,
		isInitialDeploy: boolean,
		superuserEmail?: string,
		superuserPass?: string
	) {
		deployingIds = [...deployingIds, deploymentId];
		try {
			await api.deploy.deployFromRecord(deploymentId, isInitialDeploy, superuserEmail, superuserPass);
		} finally {
			deployingIds = deployingIds.filter((id) => id !== deploymentId);
		}
	}

	function _onRealtime(action: string, record: Deployment) {
		if (action === 'delete') {
			deployments = deployments.filter((d) => d.id !== record.id);
		} else if (action === 'create') {
			if (!deployments.find((d) => d.id === record.id)) {
				deployments = [record, ...deployments];
			}
		} else {
			const exists = deployments.find((d) => d.id === record.id);
			if (exists) {
				deployments = deployments.map((d) => (d.id === record.id ? { ...d, ...record } : d));
			} else {
				deployments = [record, ...deployments];
			}
		}
	}

	return {
		get deployments() {
			return deployments;
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
		get deployingIds() {
			return deployingIds;
		},
		byApp,
		byId,
		isDeploying,
		load,
		create,
		remove,
		deploy,
		_onRealtime,
		clearError() {
			error = null;
		}
	};
}

export const deploymentsStore = createDeploymentsStore();
