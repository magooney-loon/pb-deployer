<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { Server, App } from '$lib/api.js';
	import { DeleteServerModalLogic } from './DeleteServerModal.js';

	interface Props {
		open?: boolean;
		server?: Server | null;
		apps?: App[];
		loading?: boolean;
		onclose?: () => void;
		onconfirm?: (serverId: string) => void;
	}

	let {
		open = false,
		server = null,
		apps = [],
		loading = false,
		onclose,
		onconfirm
	}: Props = $props();

	// Create logic instance
	const logic = new DeleteServerModalLogic({ open, server, apps, loading, onclose, onconfirm });
	let state = $state(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update props when they change
	$effect(() => {
		logic.updateProps({ open, server, apps, loading, onclose, onconfirm });
	});
</script>

<Modal
	open={state.open}
	title="Delete Server"
	size="md"
	closeable={!state.loading}
	onclose={() => logic.handleClose()}
>
	{#if state.server !== null}
		<div class="space-y-6">
			<!-- Warning -->
			<div class="rounded-lg bg-red-50 p-4 dark:bg-red-900/20">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
							<path
								fill-rule="evenodd"
								d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
								clip-rule="evenodd"
							/>
						</svg>
					</div>
					<div class="ml-3">
						<h3 class="text-sm font-medium text-red-800 dark:text-red-200">
							This action cannot be undone
						</h3>
						<div class="mt-2 text-sm text-red-700 dark:text-red-300">
							<p>
								This will permanently delete the server configuration. The actual VPS server will
								not be affected.
							</p>
						</div>
					</div>
				</div>
			</div>

			<!-- Server Details -->
			<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-700">
				<h4 class="mb-3 font-medium text-gray-900 dark:text-white">Server Details</h4>
				<div class="space-y-2 text-sm">
					<div class="flex justify-between">
						<span class="text-gray-600 dark:text-gray-400">Name:</span>
						<span class="font-medium text-gray-900 dark:text-white">{state.server?.name}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600 dark:text-gray-400">Host:</span>
						<span class="font-mono text-gray-900 dark:text-white"
							>{state.server?.host}:{state.server?.port}</span
						>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600 dark:text-gray-400">Setup Status:</span>
						<span class="text-gray-900 dark:text-white">
							{#if state.server}
								{#if state.server.setup_complete && state.server.security_locked}
									<span
										class="rounded bg-green-100 px-2 py-1 text-xs text-green-800 dark:bg-green-900 dark:text-green-200"
									>
										Ready
									</span>
								{:else if state.server.setup_complete}
									<span
										class="rounded bg-yellow-100 px-2 py-1 text-xs text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
									>
										Setup Complete
									</span>
								{:else}
									<span
										class="rounded bg-red-100 px-2 py-1 text-xs text-red-800 dark:bg-red-900 dark:text-red-200"
									>
										Not Setup
									</span>
								{/if}
							{:else}
								<span class="bg-gray-100 text-gray-800">Unknown</span>
							{/if}
						</span>
					</div>
				</div>
			</div>

			<!-- Associated Apps Warning -->
			{#if state.server && state.apps.filter((app) => app.server_id === state.server!.id).length > 0}
				<div class="rounded-lg bg-orange-50 p-4 dark:bg-orange-900/20">
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-orange-400" viewBox="0 0 20 20" fill="currentColor">
								<path
									fill-rule="evenodd"
									d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-medium text-orange-800 dark:text-orange-200">
								Warning: Apps Deployed on This Server
							</h3>
							<div class="mt-2">
								<p class="text-sm text-orange-700 dark:text-orange-300">
									This server has {state.apps.filter((app) => app.server_id === state.server!.id)
										.length} app{state.apps.filter((app) => app.server_id === state.server!.id)
										.length !== 1
										? 's'
										: ''} deployed on it:
								</p>
								<ul
									class="mt-2 list-inside list-disc space-y-1 text-sm text-orange-700 dark:text-orange-300"
								>
									{#each state.apps.filter((app) => app.server_id === state.server!.id) as app (app.id)}
										<li>
											<span class="font-medium">{app.name}</span>
											{#if app.domain}
												<span class="text-xs">({app.domain})</span>
											{/if}
										</li>
									{/each}
								</ul>
								<p class="mt-2 text-sm text-orange-700 dark:text-orange-300">
									You should delete or migrate these apps before deleting the server.
								</p>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Confirmation Input -->
			<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-700">
				<p class="mb-2 text-sm text-gray-700 dark:text-gray-300">
					To confirm deletion, type the server name: <strong class="font-mono"
						>{state.server?.name}</strong
					>
				</p>
				<input
					type="text"
					placeholder="Enter server name to confirm"
					class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-red-500 focus:ring-red-500 dark:border-gray-600 dark:bg-gray-800 dark:text-white"
					value={state.confirmationText}
					oninput={(e) => logic.updateConfirmationText(e.currentTarget.value)}
					disabled={state.loading}
				/>
			</div>
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-500 dark:text-gray-400">No server selected for deletion</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<button
				onclick={() => logic.handleClose()}
				disabled={state.loading}
				class="rounded-lg bg-gray-600 px-4 py-2 font-medium text-white transition-colors hover:bg-gray-700 disabled:opacity-50 dark:bg-gray-500 dark:hover:bg-gray-600"
			>
				Cancel
			</button>
			<button
				onclick={() => logic.handleConfirm()}
				disabled={state.loading || !state.server || state.confirmationText !== state.server!.name}
				class="rounded-lg bg-red-600 px-4 py-2 font-medium text-white transition-colors hover:bg-red-700 disabled:opacity-50"
			>
				{#if state.loading}
					<div class="flex items-center">
						<div class="mr-2 h-4 w-4 animate-spin rounded-full border-b-2 border-white"></div>
						Deleting...
					</div>
				{:else}
					Delete Server
				{/if}
			</button>
		</div>
	{/snippet}
</Modal>
