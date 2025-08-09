<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';

	interface ConnectionTestResult {
		success: boolean;
		connection_info?: {
			server_host: string;
			username: string;
		};
		app_user_connection?: string;
		error?: string;
	}

	interface Props {
		open?: boolean;
		result?: ConnectionTestResult | null;
		serverName?: string;
		loading?: boolean;
		onclose?: () => void;
	}

	let { open = false, result = null, serverName = '', loading = false, onclose }: Props = $props();

	// Event handlers
	function handleClose() {
		onclose?.();
	}
</script>

<Modal
	{open}
	title={loading ? 'Testing Connection...' : 'Connection Test Results'}
	size="md"
	{onclose}
>
	{#if result}
		{#if result.success}
			<!-- Success State -->
			<div class="text-center">
				<div
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/20"
				>
					<svg
						class="h-6 w-6 text-green-600 dark:text-green-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"
						></path>
					</svg>
				</div>
				<div class="mt-3">
					<h3 class="text-lg font-medium text-gray-900 dark:text-white">Connection Successful!</h3>
					<div class="mt-2 text-sm text-gray-500 dark:text-gray-400">
						Successfully connected to {serverName || 'the server'}
					</div>
				</div>
			</div>

			<!-- Connection Details -->
			<div class="mt-6 space-y-4">
				<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-700">
					<h4 class="mb-3 font-medium text-gray-900 dark:text-white">Connection Details</h4>
					<div class="space-y-2 text-sm">
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Server Host:</span>
							<span class="font-mono text-gray-900 dark:text-white"
								>{result.connection_info?.server_host}</span
							>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Root User:</span>
							<span class="font-mono text-gray-900 dark:text-white"
								>{result.connection_info?.username}</span
							>
						</div>
						{#if result.app_user_connection}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">App User:</span>
								<span class="font-mono text-gray-900 dark:text-white"
									>{result.app_user_connection}</span
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
					class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/20"
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
				<div class="mt-3">
					<h3 class="text-lg font-medium text-gray-900 dark:text-white">Connection Failed</h3>
					<div class="mt-2 text-sm text-gray-500 dark:text-gray-400">
						Could not connect to {serverName || 'the server'}
					</div>
				</div>
			</div>

			<!-- Error Details -->
			<div class="mt-6">
				<div class="rounded-lg bg-red-50 p-4 dark:bg-red-900/20">
					<h4 class="mb-2 font-medium text-red-800 dark:text-red-200">Error Details</h4>
					<p
						class="rounded bg-red-100 p-2 font-mono text-sm text-red-700 dark:bg-red-900/40 dark:text-red-300"
					>
						{result.error || 'Unknown connection error'}
					</p>
				</div>

				<!-- Troubleshooting Tips -->
				<div class="mt-4 rounded-lg bg-gray-50 p-4 dark:bg-gray-700">
					<h4 class="mb-2 font-medium text-gray-900 dark:text-white">Troubleshooting Tips</h4>
					<ul class="space-y-1 text-sm text-gray-600 dark:text-gray-400">
						<li>• Check that the server IP address is correct</li>
						<li>• Check firewall settings on both client and server</li>
					</ul>
				</div>
			</div>
		{/if}
	{:else if loading}
		<!-- Loading State -->
		<div class="py-8 text-center">
			<div class="flex items-center justify-center">
				<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
				<span class="ml-3 text-gray-600 dark:text-gray-400"
					>Testing connection to {serverName || 'server'}...</span
				>
			</div>
		</div>
	{:else}
		<!-- No result state -->
		<div class="py-8 text-center">
			<div class="text-gray-500 dark:text-gray-400">No test results available</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<button
				onclick={handleClose}
				class="rounded-lg bg-gray-600 px-4 py-2 font-medium text-white transition-colors hover:bg-gray-700 dark:bg-gray-500 dark:hover:bg-gray-600"
			>
				Close
			</button>
		</div>
	{/snippet}
</Modal>
