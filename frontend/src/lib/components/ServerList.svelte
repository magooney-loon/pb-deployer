<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { formatTimestamp } from '../api.js';
	import { ServerListLogic, type ServerListState } from './logic/ServerList.js';
	import ConnectionTestModal from '$lib/components/modals/ConnectionTestModal.svelte';
	import DeleteServerModal from '$lib/components/modals/DeleteServerModal.svelte';
	import ProgressModal from '$lib/components/modals/ProgressModal.svelte';
	import {
		Button,
		ErrorAlert,
		FormField,
		EmptyState,
		LoadingSpinner,
		Card
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

<div class="p-6">
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Servers</h1>
			<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
				Manage your VPS servers and deployment infrastructure
			</p>
		</div>
		<Button onclick={() => logic.toggleCreateForm()}>
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
						bind:value={state.newServer.name}
						placeholder="Production Server"
						required
					/>

					<FormField
						id="host"
						label="VPS IP"
						bind:value={state.newServer.host}
						placeholder="192.168.1.100"
						required
					/>

					<FormField
						id="port"
						label="SSH Port"
						type="number"
						bind:value={state.newServer.port}
						min={1}
						max={65535}
					/>

					<FormField
						id="root_username"
						label="Root Username"
						bind:value={state.newServer.root_username}
					/>

					<FormField
						id="app_username"
						label="App Username"
						bind:value={state.newServer.app_username}
					/>

					<FormField
						id="use_ssh_agent"
						label="Use SSH Agent"
						type="checkbox"
						bind:checked={state.newServer.use_ssh_agent}
					/>
				</div>

				{#if !state.newServer.use_ssh_agent}
					<FormField
						id="manual_key_path"
						label="Private Key Path"
						bind:value={state.newServer.manual_key_path}
						placeholder="/home/user/.ssh/id_rsa"
					/>
				{/if}

				<div class="flex space-x-3">
					<Button type="submit">Create Server</Button>
					<Button
						variant="outline"
						color="gray"
						onclick={() => {
							logic.toggleCreateForm();
							logic.resetForm();
						}}
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
									<div class="space-y-1">
										<span
											class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusBadge.color}"
										>
											{statusBadge.text}
										</span>

										<!-- Setup Progress -->
										{#if server.id in state.setupProgress}
											{@const steps = state.setupProgress[server.id] || []}
											<div class="space-y-1 text-xs">
												{#each steps.slice(-3) as step, index (step.timestamp + index)}
													<div class="flex items-center space-x-1 text-blue-600 dark:text-blue-400">
														<span>{logic.getProgressStepIcon(step.status)}</span>
														<span class="max-w-32 truncate">{step.message}</span>
													</div>
												{/each}
											</div>
										{/if}

										<!-- Security Progress -->
										{#if server.id in state.securityProgress}
											{@const steps = state.securityProgress[server.id] || []}
											<div class="space-y-1 text-xs">
												{#each steps.slice(-3) as step, index (step.timestamp + index)}
													<div
														class="flex items-center space-x-1 text-purple-600 dark:text-purple-400"
													>
														<span>{logic.getProgressStepIcon(step.status)}</span>
														<span class="max-w-32 truncate">{step.message}</span>
													</div>
												{/each}
											</div>
										{/if}
									</div>
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
										size="sm"
										onclick={() => logic.testConnection(server.id)}
										disabled={state.testingConnection.has(server.id)}
									>
										{state.testingConnection.has(server.id) ? 'Testing...' : 'Test Connection'}
									</Button>

									{#if !server.setup_complete}
										<Button
											variant="ghost"
											color="green"
											size="sm"
											onclick={() => logic.runSetup(server.id)}
											disabled={state.runningSetup.has(server.id)}
										>
											{state.runningSetup.has(server.id) ? 'Setting Up...' : 'Run Setup'}
										</Button>
									{:else if !server.security_locked}
										<Button
											variant="ghost"
											color="purple"
											size="sm"
											onclick={() => logic.applySecurity(server.id)}
											disabled={state.applyingSecurity.has(server.id)}
										>
											{state.applyingSecurity.has(server.id) ? 'Securing...' : 'Apply Security'}
										</Button>
									{/if}

									<Button
										variant="ghost"
										color="red"
										size="sm"
										onclick={() => logic.deleteServer(server.id)}
										disabled={state.runningSetup.has(server.id) ||
											state.applyingSecurity.has(server.id)}
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
</div>

<!-- Connection Test Modal -->
<ConnectionTestModal
	open={state.showConnectionModal}
	result={state.connectionTestResult}
	serverName={state.testedServerName}
	loading={state.connectionTestLoading}
	onclose={() => logic.closeConnectionModal()}
/>

<!-- Delete Server Modal -->
<DeleteServerModal
	open={state.showDeleteModal}
	server={state.serverToDelete}
	apps={state.apps}
	loading={state.deleting}
	onclose={() => logic.closeDeleteModal()}
	onconfirm={(id) => logic.confirmDeleteServer(id)}
/>

<!-- Setup Progress Modal -->
<ProgressModal
	bind:show={state.showSetupProgressModal}
	title="Server Setup Progress - {state.currentProgressServerName}"
	progress={state.currentProgressServerId
		? state.setupProgress[state.currentProgressServerId] || []
		: []}
	onClose={() => logic.closeSetupProgressModal()}
	loading={state.runningSetup.has(state.currentProgressServerId || '')}
	operationInProgress={state.currentProgressServerId
		? state.runningSetup.has(state.currentProgressServerId)
		: false}
/>

<!-- Security Progress Modal -->
<ProgressModal
	bind:show={state.showSecurityProgressModal}
	title="Security Lockdown Progress - {state.currentProgressServerName}"
	progress={state.currentProgressServerId
		? state.securityProgress[state.currentProgressServerId] || []
		: []}
	onClose={() => logic.closeSecurityProgressModal()}
	loading={state.applyingSecurity.has(state.currentProgressServerId || '')}
	operationInProgress={state.currentProgressServerId
		? state.applyingSecurity.has(state.currentProgressServerId)
		: false}
/>
