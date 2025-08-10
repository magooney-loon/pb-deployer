<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { formatTimestamp } from '../api.js';
	import { ServerListLogic, type ServerListState } from './logic/ServerList.js';
	import ConnectionTestModal from '$lib/components/modals/ConnectionTestModal.svelte';
	import DeleteServerModal from '$lib/components/modals/DeleteServerModal.svelte';
	import ProgressModal from '$lib/components/modals/ProgressModal.svelte';

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
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-3xl font-bold text-gray-900 dark:text-white">Servers</h1>
		<button
			onclick={() => logic.toggleCreateForm()}
			class="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700"
		>
			{state.showCreateForm ? 'Cancel' : 'Add Server'}
		</button>
	</div>

	{#if state.error}
		<div
			class="mb-6 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900"
		>
			<div class="flex">
				<div class="flex-shrink-0">
					<span class="text-red-400">❌</span>
				</div>
				<div class="ml-3">
					<h3 class="text-sm font-medium text-red-800 dark:text-red-200">Error</h3>
					<div class="mt-2 text-sm text-red-700 dark:text-red-300">
						<p>{state.error}</p>
					</div>
					<div class="mt-4">
						<button
							onclick={() => logic.dismissError()}
							class="rounded bg-red-100 px-3 py-1 text-sm text-red-800 hover:bg-red-200 dark:bg-red-800 dark:text-red-200 dark:hover:bg-red-700"
						>
							Dismiss
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}

	{#if state.showCreateForm}
		<div class="mb-6 rounded-lg bg-white p-6 shadow dark:bg-gray-800 dark:shadow-gray-700">
			<h2 class="mb-4 text-xl font-semibold dark:text-white">Add New Server</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					logic.createServer();
				}}
				class="space-y-4"
			>
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<div>
						<label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Name</label
						>
						<input
							id="name"
							bind:value={state.newServer.name}
							type="text"
							required
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="Production Server"
						/>
					</div>
					<div>
						<label for="host" class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>VPS IP</label
						>
						<input
							id="host"
							bind:value={state.newServer.host}
							type="text"
							required
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="192.168.1.100"
						/>
					</div>
					<div>
						<label for="port" class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>SSH Port</label
						>
						<input
							id="port"
							bind:value={state.newServer.port}
							type="number"
							min="1"
							max="65535"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div>
						<label
							for="root_username"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Root Username</label
						>
						<input
							id="root_username"
							bind:value={state.newServer.root_username}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div>
						<label
							for="app_username"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300">App Username</label
						>
						<input
							id="app_username"
							bind:value={state.newServer.app_username}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div class="flex items-center">
						<input
							id="use_ssh_agent"
							bind:checked={state.newServer.use_ssh_agent}
							type="checkbox"
							class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700"
						/>
						<label for="use_ssh_agent" class="ml-2 block text-sm text-gray-900 dark:text-gray-300">
							Use SSH Agent
						</label>
					</div>
				</div>
				{#if !state.newServer.use_ssh_agent}
					<div>
						<label
							for="manual_key_path"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Private Key Path</label
						>
						<input
							id="manual_key_path"
							bind:value={state.newServer.manual_key_path}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="/home/user/.ssh/id_rsa"
						/>
					</div>
				{/if}
				<div class="flex space-x-3">
					<button
						type="submit"
						class="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700"
					>
						Create Server
					</button>
					<button
						type="button"
						onclick={() => {
							logic.toggleCreateForm();
							logic.resetForm();
						}}
						class="rounded-lg bg-gray-600 px-4 py-2 font-medium text-white hover:bg-gray-700 dark:bg-gray-500 dark:hover:bg-gray-600"
					>
						Cancel
					</button>
				</div>
			</form>
		</div>
	{/if}

	{#if state.loading}
		<div class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
			<span class="ml-2 text-gray-600 dark:text-gray-400">Loading servers...</span>
		</div>
	{:else if state.servers.length === 0}
		<div class="py-12 text-center">
			<p class="mb-4 text-lg text-gray-500 dark:text-gray-400">No servers configured yet</p>
			<button
				onclick={() => logic.toggleCreateForm()}
				class="rounded-lg bg-blue-600 px-6 py-3 font-medium text-white hover:bg-blue-700"
			>
				Add Your First Server
			</button>
		</div>
	{:else}
		<div
			class="overflow-hidden bg-white shadow sm:rounded-lg dark:bg-gray-800 dark:shadow-gray-700"
		>
			<div class="overflow-x-auto">
				<table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
					<thead class="bg-gray-50 dark:bg-gray-700">
						<tr>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400"
								>Server</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400"
								>Status</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400"
								>Connection</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400"
								>Created</th
							>
							<th
								class="px-6 py-3 text-right text-xs font-medium tracking-wider text-gray-500 uppercase dark:text-gray-400"
								>Actions</th
							>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-700 dark:bg-gray-800">
						{#each state.servers as server (server.id)}
							{@const statusBadge = logic.getServerStatusBadge(server)}
							<tr class="hover:bg-gray-50 dark:hover:bg-gray-700">
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="flex items-center">
										<div>
											<div class="text-sm font-medium text-gray-900 dark:text-white">
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
									<button
										onclick={() => logic.testConnection(server.id)}
										disabled={state.testingConnection.has(server.id)}
										class="text-blue-600 hover:text-blue-900 disabled:opacity-50 dark:text-blue-400 dark:hover:text-blue-300"
									>
										{state.testingConnection.has(server.id) ? 'Testing...' : 'Test Connection'}
									</button>

									{#if !server.setup_complete}
										<button
											onclick={() => logic.runSetup(server.id)}
											disabled={state.runningSetup.has(server.id)}
											class="text-green-600 hover:text-green-900 disabled:opacity-50 dark:text-green-400 dark:hover:text-green-300"
										>
											{state.runningSetup.has(server.id) ? 'Setting Up...' : 'Run Setup'}
										</button>
									{:else if !server.security_locked}
										<button
											onclick={() => logic.applySecurity(server.id)}
											disabled={state.applyingSecurity.has(server.id)}
											class="text-purple-600 hover:text-purple-900 disabled:opacity-50 dark:text-purple-400 dark:hover:text-purple-300"
										>
											{state.applyingSecurity.has(server.id) ? 'Securing...' : 'Apply Security'}
										</button>
									{/if}

									<button
										onclick={() => logic.deleteServer(server.id)}
										disabled={state.runningSetup.has(server.id) ||
											state.applyingSecurity.has(server.id)}
										class="text-red-600 hover:text-red-900 disabled:opacity-50 dark:text-red-400 dark:hover:text-red-300"
									>
										Delete
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-700 dark:text-gray-300">
				Showing {state.servers.length} server{state.servers.length !== 1 ? 's' : ''}
			</p>
			<button
				onclick={() => logic.loadServers()}
				class="rounded bg-gray-100 px-3 py-1 text-sm text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
			>
				🔄 Refresh
			</button>
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

<style>
	input[type='text'],
	input[type='number'] {
		border: 1px solid #d1d5db;
		border-radius: 0.375rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
	}

	input[type='text']:focus,
	input[type='number']:focus {
		outline: none;
		box-shadow: 0 0 0 2px #3b82f6;
		border-color: #3b82f6;
	}

	:global([data-theme='dark']) input[type='text'],
	:global([data-theme='dark']) input[type='number'] {
		border-color: #4b5563;
		background-color: #374151;
		color: white;
	}

	:global([data-theme='dark']) input[type='text']:focus,
	:global([data-theme='dark']) input[type='number']:focus {
		border-color: #3b82f6;
	}
</style>
