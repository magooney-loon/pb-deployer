import { ApiClient } from '$lib/api/index.js';
import { appsStore } from './apps.svelte.js';
import { serversStore } from './servers.svelte.js';
import { deploymentsStore } from './deployments.svelte.js';
import { versionsStore } from './versions.svelte.js';

export const api = new ApiClient();
const pb = api.getPocketBase();

let realtimeStarted = false;

export async function startRealtime() {
	if (realtimeStarted) return;
	realtimeStarted = true;

	await pb
		.collection('apps')
		.subscribe('*', (e) => appsStore._onRealtime(e.action, e.record as never));
	await pb
		.collection('servers')
		.subscribe('*', (e) => serversStore._onRealtime(e.action, e.record as never));
	await pb
		.collection('deployments')
		.subscribe('*', (e) => deploymentsStore._onRealtime(e.action, e.record as never));
	await pb
		.collection('versions')
		.subscribe('*', (e) => versionsStore._onRealtime(e.action, e.record as never));
}

export async function stopRealtime() {
	if (!realtimeStarted) return;
	realtimeStarted = false;
	await pb.collection('apps').unsubscribe();
	await pb.collection('servers').unsubscribe();
	await pb.collection('deployments').unsubscribe();
	await pb.collection('versions').unsubscribe();
}
