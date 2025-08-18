<script lang="ts">
	import { onMount } from 'svelte';
	import { AppListLogic, type AppListState } from './AppList.js';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import AppCreateModal from '$lib/components/modals/AppCreateModal.svelte';
	import UploadVersionModal from '$lib/components/modals/UploadVersionModal.svelte';
	import { Button, Toast, EmptyState, LoadingSpinner, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';

	interface AppFormData {
		name: string;
		server_id: string;
		domain: string;
		remote_path: string;
		service_name: string;
		version_number: string;
		version_notes: string;
		initialZip?: File;
	}

	const logic = new AppListLogic();
	let state = $state<AppListState>(logic.getState());

	logic.onStateUpdate((newState) => {
		state = newState;
	});

	let availableServers = $derived(state.servers.filter((s) => s.setup_complete));

	onMount(async () => {
		await logic.initialize();
	});

	async function handleCreateApp(appData: AppFormData): Promise<void> {
		logic.updateNewApp('name', appData.name);
		logic.updateNewApp('server_id', appData.server_id);
		logic.updateNewApp('domain', appData.domain);
		logic.updateNewApp('remote_path', appData.remote_path);
		logic.updateNewApp('service_name', appData.service_name);
		logic.updateNewApp('version_number', appData.version_number);
		logic.updateNewApp('version_notes', appData.version_notes);

		await logic.createApp(appData.initialZip);
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
		onclick={() => logic.toggleCreateForm()}
		disabled={availableServers.length === 0 || state.creating || state.deleting || state.uploading}
	>
		{#snippet iconSnippet()}
			<Icon name="plus" />
		{/snippet}
		Add App
	</Button>
</header>

{#if availableServers.length === 0 && !state.showCreateForm}
	<Toast type="warning" message="No servers ready for deployment." dismissible={false} />
{/if}

{#if state.error}
	<Toast message={state.error} type="error" onDismiss={() => logic.dismissError()} />
{/if}

{#if state.loading}
	<LoadingSpinner text="Loading applications..." />
{:else if state.apps.length === 0}
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
					{#each state.apps as app (app.id)}
						{@const statusBadge = logic.getAppStatusBadge(app)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex items-center">
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
										<div class="text-xs text-gray-400 dark:text-gray-500">{app.service_name}</div>
									</div>
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="text-sm text-gray-900 dark:text-gray-100">
									{logic.getServerName(app.server_id)}
								</div>
								<div class="text-xs text-gray-500 dark:text-gray-400">{app.remote_path}</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex w-fit flex-col space-y-1">
									<StatusBadge status={statusBadge.text} variant={statusBadge.variant} dot />
									{#if app.latest_version && logic.hasUpdateAvailable(app.current_version, app.latest_version)}
										<StatusBadge status="Update Available" variant="update" size="xs" />
									{/if}
								</div>
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div class="flex flex-col">
									<span class="font-medium text-gray-900 dark:text-gray-100">
										{app.current_version || 'Not deployed'}
									</span>
									{#if app.latest_version && app.current_version !== app.latest_version}
										<span class="text-xs text-purple-600 dark:text-purple-400">
											Latest: v{app.latest_version}
										</span>
									{/if}
								</div>
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								{logic.formatTimestamp(app.created)}
							</td>
							<td class="space-x-1 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								<Button
									variant="ghost"
									color="blue"
									size="sm"
									disabled={state.deleting || state.creating || state.uploading}
									onclick={() => logic.openUploadModal(app.id)}
								>
									{#snippet iconSnippet()}
										<Icon name="upload" />
									{/snippet}
									Upload
								</Button>

								<Button
									variant="ghost"
									color="red"
									size="sm"
									disabled={state.deleting || state.creating || state.uploading}
									onclick={() => logic.deleteApp(app.id)}
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
			Showing {state.apps.length} application{state.apps.length !== 1 ? 's' : ''}
		</p>
		<Button
			variant="outline"
			size="sm"
			onclick={() => logic.loadApps()}
			disabled={state.creating || state.deleting || state.uploading}
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
	open={state.showCreateForm}
	servers={state.servers}
	creating={state.creating}
	onclose={() => logic.toggleCreateForm()}
	oncreate={handleCreateApp}
/>

<!-- Delete App Modal -->
<DeleteModal
	open={state.showDeleteModal}
	item={state.appToDelete}
	itemType="app"
	loading={state.deleting}
	onclose={() => logic.closeDeleteModal()}
	onconfirm={(id) => logic.confirmDeleteApp(id)}
/>

<!-- Upload Version Modal -->
<UploadVersionModal
	open={state.showUploadModal}
	app={state.appToUpload}
	uploading={state.uploading}
	onclose={() => logic.closeUploadModal()}
	onupload={(versionData: { version_number: string; notes: string; deploymentZip: File }) =>
		logic.uploadVersion(versionData)}
/>
