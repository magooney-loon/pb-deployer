<script lang="ts">
	import { onMount } from 'svelte';
	import { appsStore, serversStore, ui } from '$lib/stores';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import AppCreateModal from '$lib/components/modals/AppCreateModal.svelte';
	import ManageAppModal from '$lib/components/modals/ManageAppModal.svelte';
	import ProxyMigrateModal from '$lib/components/modals/ProxyMigrateModal.svelte';
	import { Button, Toast, EmptyState, LoadingSpinner, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { getAppStatusBadge, formatTimestamp, hasUpdateAvailable } from '$lib/components/partials/index.js';
	import type { App } from '$lib/api/index.js';

	interface AppFormData {
		name: string;
		server_id: string;
		domain: string;
		version_number: string;
		version_notes: string;
		initialZip?: File;
	}

	onMount(async () => {
		if (!appsStore.initialized) await appsStore.load();
		if (!serversStore.initialized) await serversStore.load();
	});

	let availableServers = $derived(serversStore.servers.filter((s) => s.setup_complete));

	// Modal local state
	let showCreateForm = $state(false);
	let creating = $state(false);

	let showDeleteModal = $state(false);
	let appToDelete = $state<App | null>(null);
	let deleting = $state(false);

	let showManageModal = $state(false);
	let appToManage = $state<App | null>(null);

	let showMigrateModal = $state(false);
	let appToMigrate = $state<App | null>(null);

	let error = $state<string | null>(null);

	async function handleCreateApp(appData: AppFormData): Promise<void> {
		creating = true;
		error = null;
		try {
			await appsStore.create({
				name: appData.name,
				server_id: appData.server_id,
				domain: appData.domain,
				version_number: appData.version_number,
				version_notes: appData.version_notes,
				initialZip: appData.initialZip
			});
			showCreateForm = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create app';
		} finally {
			creating = false;
		}
	}

	function openDeleteModal(app: App) {
		appToDelete = app;
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		setTimeout(() => {
			appToDelete = null;
		}, 200);
	}

	async function confirmDelete(id: string) {
		deleting = true;
		try {
			await appsStore.remove(id);
			showDeleteModal = false;
			appToDelete = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete app';
		} finally {
			deleting = false;
		}
	}

	function openManageModal(app: App) {
		appToManage = app;
		showManageModal = true;
	}

	function closeManageModal() {
		showManageModal = false;
		setTimeout(() => {
			appToManage = null;
		}, 200);
	}

	function openMigrateModal(app: App) {
		appToMigrate = app;
		showMigrateModal = true;
	}

	function closeMigrateModal() {
		showMigrateModal = false;
		setTimeout(() => {
			appToMigrate = null;
		}, 200);
	}

	function getServerName(serverId: string) {
		const server = serversStore.servers.find((s) => s.id === serverId);
		return server ? server.name : 'Unknown';
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text).catch(() => {});
	}
</script>

<header class="mb-8 flex items-center justify-between">
	<div>
		<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Applications</h1>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
			Manage your deployed PocketBase applications
		</p>
	</div>
	<Button
		variant="outline"
		onclick={() => (showCreateForm = true)}
		disabled={availableServers.length === 0 || creating || deleting}
	>
		{#snippet iconSnippet()}
			<Icon name="plus" />
		{/snippet}
		Add App
	</Button>
</header>

{#if availableServers.length === 0 && !showCreateForm}
	<Toast type="warning" message="No servers ready for deployment." dismissible={false} />
{/if}

{#if error}
	<Toast message={error} type="error" onDismiss={() => (error = null)} />
{/if}

{#if appsStore.error}
	<Toast message={appsStore.error} type="error" onDismiss={() => appsStore.clearError()} />
{/if}

{#if appsStore.loading}
	<LoadingSpinner text="Loading applications..." />
{:else if appsStore.apps.length === 0}
	<EmptyState
		title="No applications created yet"
		description={availableServers.length > 0
			? 'Create your first application to start deploying'
			: 'Set up a server first before creating apps'}
	>
		{#snippet iconSnippet()}
			<Icon name="apps" size="h-12 w-12" class="text-gray-400" />
		{/snippet}
	</EmptyState>
{:else}
	<div
		class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950"
	>
		<div class="overflow-x-auto">
			<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-800">
				<thead class="bg-gray-50 dark:bg-gray-900">
					<tr>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Application</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Server</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Port</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Status</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Version</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Created</th
						>
						<th
							class="px-6 py-3 text-right text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Actions</th
						>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-950">
					{#each appsStore.apps as app (app.id)}
						{@const statusBadge = getAppStatusBadge(app, app.latest_version)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div>
									<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
										{app.name}
									</div>
									<div class="text-sm text-gray-500 dark:text-gray-400">
										<a
											href="https://{app.domain}"
											target="_blank"
											class="inline-flex items-center space-x-1 text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
										>
											<span>{app.domain}</span>
											<Icon name="link" size="h-3 w-3" />
										</a>
									</div>
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="text-sm text-gray-900 dark:text-gray-100">
									{getServerName(app.server_id)}
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								{#if app.http_port}
									<button
										class="font-mono text-xs text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
										title="Click to copy"
										onclick={() => copyToClipboard(`127.0.0.1:${app.http_port}`)}
									>
										127.0.0.1:{app.http_port}
									</button>
								{:else}
									<span class="text-xs text-gray-400 dark:text-gray-500">—</span>
								{/if}
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex w-fit flex-col space-y-1">
									<StatusBadge status={statusBadge.text} variant={statusBadge.variant} dot />
									{#if app.latest_version && app.deployed_version && hasUpdateAvailable(app.deployed_version, app.latest_version)}
										<StatusBadge status="Update" variant="update" size="xs" dot />
									{/if}
								</div>
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div class="flex flex-col">
									<span class="font-medium text-gray-900 dark:text-gray-100">
										{app.deployed_version || 'Not deployed'}
									</span>
									{#if app.latest_version && app.deployed_version && app.deployed_version !== app.latest_version}
										<span class="text-xs text-purple-600 dark:text-purple-400">
											Latest: v{app.latest_version}
										</span>
									{:else if app.latest_version && !app.deployed_version}
										<span class="text-xs text-blue-600 dark:text-blue-400">
											v{app.latest_version} ready
										</span>
									{/if}
									{#if app.has_pending_deployment}
										<span class="text-xs text-amber-600 dark:text-amber-400">Pending...</span>
									{/if}
								</div>
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								{formatTimestamp(app.created)}
							</td>
							<td class="space-x-1 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								{#if app.status === 'needs_migration'}
									<Button
										variant="outline"
										color="yellow"
										size="sm"
										disabled={deleting || creating}
										onclick={() => openMigrateModal(app)}
									>
										{#snippet iconSnippet()}
											<Icon name="setup" />
										{/snippet}
										Migrate
									</Button>
								{:else}
									<Button
										variant="ghost"
										color="blue"
										size="sm"
										disabled={deleting || creating}
										onclick={() => openManageModal(app)}
									>
										{#snippet iconSnippet()}
											<Icon name="apps" />
										{/snippet}
										Manage
									</Button>
								{/if}

								<Button
									variant="ghost"
									color="red"
									size="sm"
									disabled={deleting || creating}
									onclick={() => openDeleteModal(app)}
								>
									{#snippet iconSnippet()}
										<Icon name="delete" />
									{/snippet}
									Delete
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

	<div class="mt-6 flex items-center justify-between">
		<p class="text-sm text-gray-600 dark:text-gray-400">
			Showing {appsStore.apps.length} application{appsStore.apps.length !== 1 ? 's' : ''}
		</p>
		<Button
			variant="outline"
			size="sm"
			onclick={() => appsStore.load()}
			disabled={creating || deleting}
		>
			{#snippet iconSnippet()}
				<Icon name="refresh" />
			{/snippet}
			Refresh
		</Button>
	</div>
{/if}

<!-- App Create Modal -->
<AppCreateModal
	open={showCreateForm}
	servers={serversStore.servers}
	{creating}
	onclose={() => (showCreateForm = false)}
	oncreate={handleCreateApp}
/>

<!-- Delete App Modal -->
<DeleteModal
	open={showDeleteModal}
	item={appToDelete}
	itemType="app"
	loading={deleting}
	onclose={closeDeleteModal}
	onconfirm={(id) => confirmDelete(id)}
/>

<!-- Manage App Modal -->
<ManageAppModal
	open={showManageModal}
	app={appToManage}
	onclose={closeManageModal}
	onrefresh={() => appsStore.load()}
/>

<!-- Proxy Migrate Modal -->
<ProxyMigrateModal
	open={showMigrateModal}
	app={appToMigrate}
	onclose={closeMigrateModal}
/>
