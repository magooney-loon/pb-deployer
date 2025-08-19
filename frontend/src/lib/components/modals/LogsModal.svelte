<script lang="ts">
	import type { Deployment } from '$lib/api/index.js';
	import { Button, StatusBadge } from '$lib/components/partials';
	import { getDeploymentStatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Modal from '$lib/components/main/Modal.svelte';

	interface Props {
		open: boolean;
		deployment: Deployment | null;
		onclose: () => void;
		closable?: boolean;
		autoOpened?: boolean;
	}

	let { open, deployment, onclose, closable = true, autoOpened = false }: Props = $props();

	function formatTimestamp(timestamp: string): string {
		return new Date(timestamp).toLocaleString();
	}

	async function copyLogs() {
		if (deployment?.logs) {
			try {
				await navigator.clipboard.writeText(deployment.logs);
			} catch (err) {
				console.error('Failed to copy logs:', err);
			}
		}
	}

	let modalTitle = $derived(
		deployment
			? `Deployment Logs - ${deployment.expand?.app_id?.name || 'Unknown App'} v${deployment.expand?.version_id?.version_number || 'N/A'}`
			: 'Deployment Logs'
	);
</script>

<Modal
	{open}
	title={modalTitle}
	size="xl"
	closeable={closable}
	onclose={closable ? onclose : undefined}
>
	{#if deployment}
		<!-- Auto-opened deployment indicator -->
		{#if autoOpened && !closable}
			<div
				class="mb-4 rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-900/20"
			>
				<div class="flex items-center space-x-2">
					<div class="flex-shrink-0">
						<div class="h-2 w-2 animate-pulse rounded-full bg-blue-500"></div>
					</div>
					<p class="text-sm text-blue-800 dark:text-blue-200">
						Deployment in progress - logs are being updated in real-time
					</p>
				</div>
			</div>
		{/if}

		<!-- Deployment Info -->
		<div
			class="mb-6 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800"
		>
			<div class="grid grid-cols-2 gap-4 md:grid-cols-4">
				<div>
					<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">Status</div>
					<div class="mt-1">
						<StatusBadge
							status={getDeploymentStatusBadge(deployment).text}
							variant={getDeploymentStatusBadge(deployment).variant}
						/>
					</div>
				</div>
				<div>
					<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">Started</div>
					<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
						{deployment.started_at ? formatTimestamp(deployment.started_at) : 'Not started'}
					</div>
				</div>
				<div>
					<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
						Completed
					</div>
					<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
						{deployment.completed_at ? formatTimestamp(deployment.completed_at) : 'In progress'}
					</div>
				</div>
				<div>
					<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">ID</div>
					<div
						class="mt-1 font-mono text-sm text-gray-600 select-text hover:cursor-text dark:text-gray-400"
					>
						{deployment.id}
					</div>
				</div>
			</div>
		</div>

		<!-- Logs Content -->
		{#if deployment.logs}
			<div
				class="rounded-md border border-gray-200 bg-gray-50 dark:border-gray-700 dark:bg-gray-800"
			>
				<div
					class="flex items-center justify-between border-b border-gray-200 px-4 py-2 dark:border-gray-700"
				>
					<h3 class="text-sm font-medium text-gray-900 dark:text-gray-100">Deployment Logs</h3>
					<Button variant="ghost" size="xs" onclick={copyLogs}>
						{#snippet iconSnippet()}
							<Icon name="copy" size="h-3 w-3" />
						{/snippet}
						Copy
					</Button>
				</div>
				<div class="max-h-96 overflow-y-auto">
					<pre
						class="p-4 font-mono text-xs break-words whitespace-pre-wrap text-gray-800 dark:text-gray-200">{deployment.logs}</pre>
				</div>
			</div>
		{:else}
			<div
				class="rounded-md border border-gray-200 bg-gray-50 p-8 text-center dark:border-gray-700 dark:bg-gray-800"
			>
				<Icon name="file-text" size="h-8 w-8" class="mx-auto text-gray-400" />
				<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
					No logs available for this deployment
				</p>
			</div>
		{/if}
	{/if}

	{#snippet footer()}
		<div class="flex items-center justify-between">
			{#if !closable && autoOpened}
				<div class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
					<Icon name="info" size="h-4 w-4" />
					<span>Modal will close automatically when deployment completes</span>
				</div>
			{:else}
				<div></div>
			{/if}
			<Button variant="outline" onclick={onclose} disabled={!closable}>
				{closable ? 'Close' : 'Please wait...'}
			</Button>
		</div>
	{/snippet}
</Modal>
