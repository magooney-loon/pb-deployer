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
	}

	let { open, deployment, onclose }: Props = $props();

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

<Modal {open} title={modalTitle} size="xl" {onclose}>
	{#if deployment}
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
					<div class="mt-1 font-mono text-xs text-gray-600 dark:text-gray-400">
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
		<div class="flex justify-end">
			<Button variant="outline" onclick={onclose}>Close</Button>
		</div>
	{/snippet}
</Modal>
