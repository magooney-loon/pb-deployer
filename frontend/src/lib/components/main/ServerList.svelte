<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { formatTimestamp } from '$lib/api/index.js';
	import { ServerListLogic, type ServerListState } from './ServerList.js';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import {
		Button,
		ErrorAlert,
		FormField,
		EmptyState,
		LoadingSpinner,
		Card,
		StatusBadge
	} from '$lib/components/partials';

	// Create logic instance
	const logic = new ServerListLogic();
	let state = $state<ServerListState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	onMount(async () => {
		await logic.loadServers();
	});

	onDestroy(async () => {
		await logic.cleanup();
	});
</script>

<div class="mb-8 flex items-center justify-between">
	<div>
		<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Servers</h1>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
			Manage your VPS servers and deployment infrastructure
		</p>
	</div>
	<Button
		variant="outline"
		onclick={() => logic.toggleCreateForm()}
		icon={state.showCreateForm ? '✕' : '+'}
	>
		{state.showCreateForm ? 'Cancel' : 'Add Server'}
	</Button>
</div>

{#if state.error}
	<ErrorAlert message={state.error} type="error" onDismiss={() => logic.dismissError()} />
{/if}

{#if state.showCreateForm}
	<Card title="Add New Server" class="mb-6">
		<form
			onsubmit={(e) => {
				e.preventDefault();
				logic.createServer();
			}}
			class="space-y-4"
		>
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<FormField
					id="name"
					label="Name"
					value={state.newServer.name}
					placeholder="Production Server"
					required
					oninput={(e) => logic.updateNewServer('name', (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="host"
					label="VPS IP"
					value={state.newServer.host}
					placeholder="192.168.1.100"
					required
					oninput={(e) => logic.updateNewServer('host', (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="port"
					label="SSH Port"
					type="number"
					value={state.newServer.port}
					min={1}
					max={65535}
					oninput={(e) =>
						logic.updateNewServer('port', parseInt((e.target as HTMLInputElement).value) || 22)}
				/>

				<FormField
					id="root_username"
					label="Root Username"
					value={state.newServer.root_username}
					oninput={(e) =>
						logic.updateNewServer('root_username', (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="app_username"
					label="App Username"
					value={state.newServer.app_username}
					oninput={(e) =>
						logic.updateNewServer('app_username', (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="use_ssh_agent"
					label="Use SSH Agent"
					type="checkbox"
					checked={state.newServer.use_ssh_agent}
					onchange={(e) =>
						logic.updateNewServer('use_ssh_agent', (e.target as HTMLInputElement).checked)}
				/>
			</div>

			{#if !state.newServer.use_ssh_agent}
				<FormField
					id="manual_key_path"
					label="Private Key Path"
					value={state.newServer.manual_key_path}
					placeholder="/home/user/.ssh/id_rsa"
					oninput={(e) =>
						logic.updateNewServer('manual_key_path', (e.target as HTMLInputElement).value)}
				/>
			{/if}

			<div class="flex space-x-3">
				<Button
					variant="outline"
					type="submit"
					disabled={state.creating}
					icon={state.creating ? '🔄' : undefined}
				>
					{state.creating ? 'Creating...' : 'Create Server'}
				</Button>
				<Button
					variant="outline"
					color="gray"
					onclick={() => {
						logic.toggleCreateForm();
						logic.resetForm();
					}}
					disabled={state.creating}
				>
					Cancel
				</Button>
			</div>
		</form>
	</Card>
{/if}

{#if state.loading}
	<LoadingSpinner text="Loading servers..." />
{:else if state.servers.length === 0}
	<EmptyState
		icon="🖥️"
		title="No servers configured yet"
		description="Add your first server to start deploying applications"
	/>
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
							>Server</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Status</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Connection</th
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
					{#each state.servers as server (server.id)}
						{@const statusBadge = logic.getServerStatusBadge(server)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex items-center">
									<div>
										<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
											{server.name}
										</div>
										<div class="text-sm text-gray-500 dark:text-gray-400">
											{server.host}:{server.port}
										</div>
									</div>
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<StatusBadge status={statusBadge.text} variant={statusBadge.variant} />
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div>Root: {server.root_username}</div>
								<div>App: {server.app_username}</div>
								{#if server.use_ssh_agent}
									<div class="text-xs text-blue-600 dark:text-blue-400">SSH Agent</div>
								{:else}
									<div class="text-xs text-gray-400 dark:text-gray-500">Manual Key</div>
								{/if}
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								{formatTimestamp(server.created)}
							</td>
							<td class="space-x-2 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								<Button
									variant="ghost"
									color="red"
									size="sm"
									onclick={() => logic.deleteServer(server.id)}
									icon="🗑️"
								>
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
			Showing {state.servers.length} server{state.servers.length !== 1 ? 's' : ''}
		</p>
		<Button variant="outline" size="sm" icon="🔄" onclick={() => logic.loadServers()}>
			Refresh
		</Button>
	</div>
{/if}

<!-- Delete Server Modal -->
<DeleteModal
	open={state.showDeleteModal}
	item={state.serverToDelete}
	itemType="server"
	loading={state.deleting}
	relatedItems={state.apps}
	relatedItemsType="apps"
	onclose={() => logic.closeDeleteModal()}
	onconfirm={(id) => logic.confirmDeleteServer(id)}
/>
