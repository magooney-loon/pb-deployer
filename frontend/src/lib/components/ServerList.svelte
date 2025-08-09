<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Server, formatTimestamp } from '../api.js';

	let servers = $state<Server[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let showCreateForm = $state(false);
	let testingConnection = $state<Set<string>>(new Set());
	let runningSetup = $state<Set<string>>(new Set());
	let applyingSecurity = $state<Set<string>>(new Set());

	// Form data for creating new server
	let newServer = $state({
		name: '',
		host: '',
		port: 22,
		root_username: 'root',
		app_username: 'pocketbase',
		use_ssh_agent: true,
		manual_key_path: ''
	});

	onMount(async () => {
		await loadServers();
	});

	async function loadServers() {
		try {
			console.log('ServerList: Starting to load servers...');
			loading = true;
			error = null;
			const response = await api.getServers();
			console.log('ServerList: API response received:', response);
			servers = response.servers || [];
			console.log('ServerList: Servers set to:', servers);
			console.log('ServerList: Servers length:', servers.length);
		} catch (err) {
			console.error('ServerList: Error loading servers:', err);
			error = err instanceof Error ? err.message : 'Failed to load servers';
			servers = [];
		} finally {
			loading = false;
			console.log('ServerList: Loading finished. Final servers count:', servers.length);
		}
	}

	async function createServer() {
		try {
			const server = await api.createServer(newServer);
			servers = [...servers, server];
			showCreateForm = false;
			resetForm();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create server';
		}
	}

	async function deleteServer(id: string) {
		if (!confirm('Are you sure you want to delete this server?')) return;

		try {
			await api.deleteServer(id);
			servers = servers.filter((s) => s.id !== id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete server';
		}
	}

	async function testConnection(id: string) {
		try {
			testingConnection.add(id);
			const result = await api.testServerConnection(id);
			if (result.success) {
				alert(
					`Connection successful!\n\nServer: ${result.connection_info.server_host}\nUser: ${result.connection_info.username}\n${result.app_user_connection ? `App user: ${result.app_user_connection}` : ''}`
				);
			} else {
				alert(`Connection failed: ${result.error}`);
			}
		} catch (err) {
			alert(`Connection test failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			testingConnection.delete(id);
		}
	}

	async function runSetup(id: string) {
		try {
			runningSetup.add(id);
			await api.runServerSetup(id);
			alert('Server setup started. Check the server status for progress.');
			await loadServers(); // Refresh the list
		} catch (err) {
			alert(`Setup failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			runningSetup.delete(id);
		}
	}

	async function applySecurity(id: string) {
		if (!confirm('This will apply security lockdown to the server. Continue?')) return;

		try {
			applyingSecurity.add(id);
			await api.applySecurityLockdown(id);
			alert('Security lockdown started. Check the server status for progress.');
			await loadServers(); // Refresh the list
		} catch (err) {
			alert(`Security lockdown failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			applyingSecurity.delete(id);
		}
	}

	function resetForm() {
		newServer = {
			name: '',
			host: '',
			port: 22,
			root_username: 'root',
			app_username: 'pocketbase',
			use_ssh_agent: true,
			manual_key_path: ''
		};
	}

	function getServerStatusBadge(server: Server) {
		if (!server.setup_complete) {
			return { text: 'Not Setup', color: 'bg-red-100 text-red-800' };
		} else if (!server.security_locked) {
			return { text: 'Setup Complete', color: 'bg-yellow-100 text-yellow-800' };
		} else {
			return { text: 'Ready', color: 'bg-green-100 text-green-800' };
		}
	}
</script>

<div class="p-6">
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-3xl font-bold text-gray-900 dark:text-white">Servers</h1>
		<button
			onclick={() => (showCreateForm = !showCreateForm)}
			class="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700"
		>
			{showCreateForm ? 'Cancel' : 'Add Server'}
		</button>
	</div>

	{#if error}
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
						<p>{error}</p>
					</div>
					<div class="mt-4">
						<button
							onclick={() => (error = null)}
							class="rounded bg-red-100 px-3 py-1 text-sm text-red-800 hover:bg-red-200 dark:bg-red-800 dark:text-red-200 dark:hover:bg-red-700"
						>
							Dismiss
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}

	{#if showCreateForm}
		<div class="mb-6 rounded-lg bg-white p-6 shadow dark:bg-gray-800 dark:shadow-gray-700">
			<h2 class="mb-4 text-xl font-semibold dark:text-white">Add New Server</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					createServer();
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
							bind:value={newServer.name}
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
							bind:value={newServer.host}
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
							bind:value={newServer.port}
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
							bind:value={newServer.root_username}
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
							bind:value={newServer.app_username}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
						/>
					</div>
					<div class="flex items-center">
						<input
							id="use_ssh_agent"
							bind:checked={newServer.use_ssh_agent}
							type="checkbox"
							class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700"
						/>
						<label for="use_ssh_agent" class="ml-2 block text-sm text-gray-900 dark:text-gray-300">
							Use SSH Agent
						</label>
					</div>
				</div>
				{#if !newServer.use_ssh_agent}
					<div>
						<label
							for="manual_key_path"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Private Key Path</label
						>
						<input
							id="manual_key_path"
							bind:value={newServer.manual_key_path}
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
							showCreateForm = false;
							resetForm();
						}}
						class="rounded-lg bg-gray-600 px-4 py-2 font-medium text-white hover:bg-gray-700 dark:bg-gray-500 dark:hover:bg-gray-600"
					>
						Cancel
					</button>
				</div>
			</form>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
			<span class="ml-2 text-gray-600 dark:text-gray-400">Loading servers...</span>
		</div>
	{:else if servers.length === 0}
		<div class="py-12 text-center">
			<p class="mb-4 text-lg text-gray-500 dark:text-gray-400">No servers configured yet</p>
			<button
				onclick={() => (showCreateForm = true)}
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
						{#each servers as server (server.id)}
							{@const statusBadge = getServerStatusBadge(server)}
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
									<span
										class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusBadge.color}"
									>
										{statusBadge.text}
									</span>
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
										onclick={() => testConnection(server.id)}
										disabled={testingConnection.has(server.id)}
										class="text-blue-600 hover:text-blue-900 disabled:opacity-50 dark:text-blue-400 dark:hover:text-blue-300"
									>
										{testingConnection.has(server.id) ? '🔄' : '🔗'} Test
									</button>

									{#if !server.setup_complete}
										<button
											onclick={() => runSetup(server.id)}
											disabled={runningSetup.has(server.id)}
											class="text-green-600 hover:text-green-900 disabled:opacity-50 dark:text-green-400 dark:hover:text-green-300"
										>
											{runningSetup.has(server.id) ? '🔄' : '⚙️'} Setup
										</button>
									{:else if !server.security_locked}
										<button
											onclick={() => applySecurity(server.id)}
											disabled={applyingSecurity.has(server.id)}
											class="text-orange-600 hover:text-orange-900 disabled:opacity-50 dark:text-orange-400 dark:hover:text-orange-300"
										>
											{applyingSecurity.has(server.id) ? '🔄' : '🔒'} Secure
										</button>
									{/if}

									<button
										onclick={() => deleteServer(server.id)}
										class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
									>
										🗑️ Delete
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
				Showing {servers.length} server{servers.length !== 1 ? 's' : ''}
			</p>
			<button
				onclick={loadServers}
				class="rounded bg-gray-100 px-3 py-1 text-sm text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
			>
				🔄 Refresh
			</button>
		</div>
	{/if}
</div>

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
