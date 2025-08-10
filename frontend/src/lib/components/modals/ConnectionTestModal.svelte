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
		{#if state.result?.success === true || state.result?.overall_status === 'healthy_secured'}
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
						All Connections Successful!
					</h3>
					<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						{#if state.result?.overall_status === 'healthy_secured'}
							TCP and App SSH connections to {state.serverName || 'the server'} are working (Security
							Locked)
						{:else}
							TCP, Root SSH, and App SSH connections to {state.serverName || 'the server'} are working
						{/if}
					</div>
				</div>
			</div>

			<!-- Connection Details -->
			<div class="mt-6 space-y-4">
				<!-- TCP Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span class="mr-2 text-emerald-500">✓</span>
						TCP Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Status:</span>
							<span class="font-medium text-emerald-600 dark:text-emerald-400">Connected</span>
						</div>
						{#if state.result?.tcp_connection?.latency}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Latency:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.tcp_connection.latency}</span
								>
							</div>
						{/if}
					</div>
				</div>

				<!-- Root SSH Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span
							class="mr-2 {state.result?.overall_status === 'healthy_secured'
								? 'text-orange-500'
								: 'text-emerald-500'}"
						>
							{state.result?.overall_status === 'healthy_secured' ? '🔒' : '✓'}
						</span>
						Root SSH Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Username:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.root_ssh_connection?.username}</span
							>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Status:</span>
							<span
								class="font-medium {state.result?.overall_status === 'healthy_secured'
									? 'text-orange-600 dark:text-orange-400'
									: 'text-emerald-600 dark:text-emerald-400'}"
							>
								{state.result?.overall_status === 'healthy_secured'
									? 'Disabled (Security Locked)'
									: 'Connected'}
							</span>
						</div>
						{#if state.result?.root_ssh_connection?.auth_method && state.result?.overall_status !== 'healthy_secured'}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Auth Method:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.root_ssh_connection.auth_method}</span
								>
							</div>
						{/if}
						{#if state.result?.overall_status === 'healthy_secured'}
							<div class="mt-2">
								<p
									class="rounded bg-orange-100 p-2 text-xs text-orange-700 ring-1 ring-orange-200 dark:bg-orange-900 dark:text-orange-300 dark:ring-orange-700"
								>
									Root SSH access has been disabled as part of security hardening. This is expected
									behavior.
								</p>
							</div>
						{/if}
					</div>
				</div>

				<!-- App SSH Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span class="mr-2 text-emerald-500">✓</span>
						App SSH Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Username:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.app_ssh_connection?.username}</span
							>
						</div>
						{#if state.result?.app_ssh_connection?.auth_method}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Auth Method:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.app_ssh_connection.auth_method}</span
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
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
						Connection Issues Detected
					</h3>
					<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						Status: {state.result?.overall_status || 'Unknown error'}
					</div>
				</div>
			</div>

			<!-- Connection Test Details -->
			<div class="mt-6 space-y-4">
				<!-- TCP Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span
							class="mr-2 {state.result?.tcp_connection?.success
								? 'text-emerald-500'
								: 'text-red-500'}"
						>
							{state.result?.tcp_connection?.success ? '✓' : '✗'}
						</span>
						TCP Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Status:</span>
							<span
								class="font-medium {state.result?.tcp_connection?.success
									? 'text-emerald-600 dark:text-emerald-400'
									: 'text-red-600 dark:text-red-400'}"
							>
								{state.result?.tcp_connection?.success ? 'Connected' : 'Failed'}
							</span>
						</div>
						{#if state.result?.tcp_connection?.latency}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Latency:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.tcp_connection.latency}</span
								>
							</div>
						{/if}
						{#if state.result?.tcp_connection?.error}
							<div class="mt-2">
								<p
									class="rounded bg-red-100 p-2 font-mono text-xs text-red-700 ring-1 ring-red-200 dark:bg-red-900 dark:text-red-300 dark:ring-red-700"
								>
									{state.result.tcp_connection.error}
								</p>
							</div>
						{/if}
					</div>
				</div>

				<!-- Root SSH Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span
							class="mr-2 {state.result?.root_ssh_connection?.success
								? 'text-emerald-500'
								: 'text-red-500'}"
						>
							{state.result?.root_ssh_connection?.success ? '✓' : '✗'}
						</span>
						Root SSH Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Username:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.root_ssh_connection?.username}</span
							>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Status:</span>
							<span
								class="font-medium {state.result?.root_ssh_connection?.success
									? 'text-emerald-600 dark:text-emerald-400'
									: 'text-red-600 dark:text-red-400'}"
							>
								{state.result?.root_ssh_connection?.success ? 'Connected' : 'Failed'}
							</span>
						</div>
						{#if state.result?.root_ssh_connection?.auth_method}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Auth Method:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.root_ssh_connection.auth_method}</span
								>
							</div>
						{/if}
						{#if state.result?.root_ssh_connection?.error}
							<div class="mt-2">
								<p
									class="rounded bg-red-100 p-2 font-mono text-xs text-red-700 ring-1 ring-red-200 dark:bg-red-900 dark:text-red-300 dark:ring-red-700"
								>
									{state.result.root_ssh_connection.error}
								</p>
							</div>
						{/if}
					</div>
				</div>

				<!-- App SSH Connection -->
				<div
					class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
				>
					<h4 class="mb-3 flex items-center font-semibold text-gray-900 dark:text-gray-100">
						<span
							class="mr-2 {state.result?.app_ssh_connection?.success
								? 'text-emerald-500'
								: 'text-red-500'}"
						>
							{state.result?.app_ssh_connection?.success ? '✓' : '✗'}
						</span>
						App SSH Connection
					</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Username:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100"
								>{state.result?.app_ssh_connection?.username}</span
							>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Status:</span>
							<span
								class="font-medium {state.result?.app_ssh_connection?.success
									? 'text-emerald-600 dark:text-emerald-400'
									: 'text-red-600 dark:text-red-400'}"
							>
								{state.result?.app_ssh_connection?.success ? 'Connected' : 'Failed'}
							</span>
						</div>
						{#if state.result?.app_ssh_connection?.auth_method}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Auth Method:</span>
								<span class="font-mono text-gray-900 dark:text-gray-100"
									>{state.result.app_ssh_connection.auth_method}</span
								>
							</div>
						{/if}
						{#if state.result?.app_ssh_connection?.error}
							<div class="mt-2">
								<p
									class="rounded bg-red-100 p-2 font-mono text-xs text-red-700 ring-1 ring-red-200 dark:bg-red-900 dark:text-red-300 dark:ring-red-700"
								>
									{state.result.app_ssh_connection.error}
								</p>
							</div>
						{/if}
					</div>
				</div>

				<!-- Troubleshooting Tips -->
				<div
					class="mt-4 rounded-lg bg-blue-50 p-4 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800"
				>
					<h4 class="mb-2 font-semibold text-blue-900 dark:text-blue-100">Troubleshooting Tips</h4>
					<ul class="space-y-1 text-sm text-blue-800 dark:text-blue-200">
						{#if state.result?.overall_status === 'healthy_secured' || state.result?.overall_status === 'app_ssh_failed'}
							<li>• For security-locked servers, only app user SSH access is available</li>
							<li>• Verify app user SSH keys are properly configured</li>
							<li>• Check that the app user has sudo privileges for deployment operations</li>
							<li>• Root SSH access is intentionally disabled after security hardening</li>
						{:else}
							<li>• Check that the server IP address and port are correct</li>
							<li>• Verify SSH keys are properly configured and accessible</li>
							<li>• Check firewall settings on both client and server</li>
							<li>• Ensure the specified usernames exist on the server</li>
							<li>• Check SSH service is running on the server</li>
						{/if}
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
