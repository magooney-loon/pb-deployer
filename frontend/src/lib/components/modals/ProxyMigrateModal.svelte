<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button } from '$lib/components/partials';
	import { appsStore, ui } from '$lib/stores';
	import type { App } from '$lib/api/index.js';

	interface Props {
		open?: boolean;
		app?: App | null;
		onclose?: () => void;
	}

	let { open = false, app = null, onclose }: Props = $props();

	let migrating = $state(false);
	let error = $state<string | null>(null);
	let success = $state(false);

	$effect(() => {
		if (!open) {
			setTimeout(() => {
				migrating = false;
				error = null;
				success = false;
			}, 300);
		}
	});

	async function handleMigrate() {
		if (!app || migrating) return;
		migrating = true;
		error = null;
		try {
			await appsStore.migrateProxy(app.id);
			success = true;
			ui.toast('success', `"${app.name}" migrated to Caddy proxy successfully`);
			setTimeout(() => {
				onclose?.();
			}, 1000);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Migration failed';
		} finally {
			migrating = false;
		}
	}
</script>

<Modal
	{open}
	title={app ? `Migrate "${app.name}" to Caddy proxy` : 'Migrate to proxy'}
	size="md"
	closeable={!migrating}
	onclose={onclose}
>
	{#if app}
		<div class="space-y-4">
			{#if success}
				<div
					class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-700 dark:bg-green-900/20"
				>
					<p class="text-sm font-medium text-green-800 dark:text-green-200">
						Migration completed successfully!
					</p>
				</div>
			{:else}
				<p class="text-sm text-gray-600 dark:text-gray-400">
					This will migrate <strong>{app.name}</strong> from direct port binding to the Caddy reverse
					proxy.
				</p>

				<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-900">
					<p class="mb-2 text-sm font-medium text-gray-900 dark:text-gray-100">This will:</p>
					<ol class="space-y-1 text-sm text-gray-600 dark:text-gray-400">
						<li>1. Stop the service briefly (~5s downtime)</li>
						<li>2. Rewrite the systemd unit to bind <code class="font-mono">127.0.0.1:{app.http_port || '&lt;auto&gt;'}</code></li>
						<li>3. Add a Caddy config fragment for <code class="font-mono">{app.domain}</code></li>
						<li>4. Start the service and reload Caddy</li>
						<li>5. Verify <code class="font-mono">https://{app.domain}</code> responds</li>
					</ol>
				</div>

				{#if error}
					<div
						class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-700 dark:bg-red-900/20"
					>
						<p class="text-sm text-red-800 dark:text-red-200">{error}</p>
					</div>
				{/if}
			{/if}
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button
				variant="secondary"
				color="gray"
				onclick={onclose}
				disabled={migrating}
				class="px-6 py-2"
			>
				{success ? 'Close' : 'Cancel'}
			</Button>
			{#if !success}
				<Button
					variant="primary"
					onclick={handleMigrate}
					disabled={migrating || !app}
					class="min-w-[120px] px-6 py-2"
				>
					{#if migrating}
						<svg
							class="mr-2 -ml-1 h-4 w-4 animate-spin text-white"
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
						>
							<circle
								class="opacity-25"
								cx="12"
								cy="12"
								r="10"
								stroke="currentColor"
								stroke-width="4"
							></circle>
							<path
								class="opacity-75"
								fill="currentColor"
								d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
							></path>
						</svg>
						Migrating...
					{:else}
						Migrate
					{/if}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>
