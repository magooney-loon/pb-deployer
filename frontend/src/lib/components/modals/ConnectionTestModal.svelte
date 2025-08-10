<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { ConnectionTestModalLogic, type ConnectionTestResult } from './ConnectionTestModal.js';

	interface Props {
		open?: boolean;
		result?: ConnectionTestResult | null;
		serverName?: string;
		loading?: boolean;
		onclose?: () => void;
	}

	let { open = false, result = null, serverName = '', loading = false, onclose }: Props = $props();

	// Create logic instance
	const logic = new ConnectionTestModalLogic({ open, result, serverName, loading, onclose });
	let state = $state(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update props when they change
	$effect(() => {
		logic.updateProps({ open, result, serverName, loading, onclose });
	});
</script>

<Modal
	open={state.open}
	title={state.loading ? 'Testing Connection...' : 'Connection Test Results'}
	size="md"
	onclose={() => logic.handleClose()}
>
	{#if state.result !== null}
		{#if state.result?.success === true}
			<!-- Success State -->
			<div class="text-center">
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800"
				>
					<svg
						class="h-6 w-6 text-emerald-600 dark:text-emerald-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"
						></path>
					</svg>
				</div>
				<div class="mt-4">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
						Connection Successful!
					</h3>
					<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						Successfully connected to {state.serverName || 'the server'}
					</div>
				</div>
			</div>

			<!-- Connection Details -->
			<div class="mt-6 space-y-4">
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 font-semibold text-gray-900 dark:text-gray-100">Connection Details</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Server Host:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.connection_info?.server_host}</span
							>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Root User:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.connection_info?.username}</span
							>
						</div>
						{#if state.result?.app_user_connection}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">App User:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result?.app_user_connection}</span
								>
							</div>
						{/if}
					</div>
				</div>
			</div>
		{:else}
			<!-- Error State -->
			<div class="text-center">
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800"
				>
					<svg
						class="h-6 w-6 text-red-600 dark:text-red-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						></path>
					</svg>
				</div>
				<div class="mt-4">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Connection Failed</h3>
					<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						Could not connect to {state.serverName || 'the server'}
					</div>
				</div>
			</div>

			<!-- Error Details -->
			<div class="mt-6">
				<div class="rounded-lg bg-red-50 p-4 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800">
					<h4 class="mb-2 font-semibold text-red-700 dark:text-red-300">Error Details</h4>
					<p
						class="rounded bg-red-100 p-3 font-mono text-sm text-red-700 ring-1 ring-red-200 dark:bg-red-900 dark:text-red-300 dark:ring-red-700"
					>
						{state.result?.error || 'Unknown connection error'}
					</p>
				</div>

				<!-- Troubleshooting Tips -->
				<div
					class="mt-4 rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-2 font-semibold text-gray-900 dark:text-gray-100">Troubleshooting Tips</h4>
					<ul class="space-y-1 text-sm text-gray-600 dark:text-gray-400">
						<li>• Check that the server IP address is correct</li>
						<li>• Check firewall settings on both client and server</li>
					</ul>
				</div>
			</div>
		{/if}
	{:else if state.loading}
		<!-- Loading State -->
		<div class="py-8 text-center">
			<div class="flex items-center justify-center">
				<div
					class="h-8 w-8 animate-spin rounded-full border-b-2 border-gray-900 dark:border-gray-100"
				></div>
				<span class="ml-3 text-gray-700 dark:text-gray-300"
					>Testing connection to {state.serverName || 'the server'}...</span
				>
			</div>
		</div>
	{:else}
		<!-- No result state -->
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No test results available</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<button
				onclick={() => logic.handleClose()}
				class="rounded-lg border border-gray-200 bg-white px-4 py-2 font-medium text-gray-900 transition-colors hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900"
			>
				Close
			</button>
		</div>
	{/snippet}
</Modal>
