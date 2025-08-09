<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type App, type Server, getStatusIcon, formatTimestamp } from '../api.js';

	let apps = $state<App[]>([]);
	let servers = $state<Server[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let showCreateForm = $state(false);
	let checkingHealth = $state<Set<string>>(new Set());

	// Form data for creating new app
	let newApp = $state({
		name: '',
		server_id: '',
		domain: '',
		remote_path: '',
		service_name: ''
	});

	onMount(async () => {
		await Promise.all([loadApps(), loadServers()]);
	});

	async function loadApps() {
		try {
			loading = true;
			error = null;
			const response = await api.getApps();
			apps = response.apps || [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load apps';
			apps = [];
		} finally {
			loading = false;
		}
	}

	async function loadServers() {
		try {
			const response = await api.getServers();
			servers = response.servers || [];
		} catch (err) {
			console.error('Failed to load servers for dropdown:', err);
			servers = [];
		}
	}

	async function createApp() {
		try {
			const appData = {
				...newApp,
				remote_path: newApp.remote_path || `/opt/pocketbase/apps/${newApp.name}`,
				service_name: newApp.service_name || `pocketbase-${newApp.name}`
			};
			const app = await api.createApp(appData);
			apps = [...apps, app];
			showCreateForm = false;
			resetForm();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create app';
		}
	}

	async function deleteApp(id: string) {
		if (!confirm('Are you sure you want to delete this app?')) return;

		try {
			await api.deleteApp(id);
			apps = apps.filter((a) => a.id !== id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete app';
		}
	}

	async function checkHealth(id: string) {
		try {
			checkingHealth.add(id);
			await api.runAppHealthCheck(id);
			setTimeout(async () => {
				await loadApps(); // Refresh to get updated status
			}, 2000);
		} catch (err) {
			alert(`Health check failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
		} finally {
			checkingHealth.delete(id);
		}
	}

	function resetForm() {
		newApp = {
			name: '',
			server_id: '',
			domain: '',
			remote_path: '',
			service_name: ''
		};
	}

	function getServerName(serverId: string): string {
		const server = servers.find((s) => s.id === serverId);
		return server ? server.name : 'Unknown Server';
	}

	function getAvailableServers(): Server[] {
		return servers.filter((s) => s.setup_complete && s.security_locked);
	}

	function getAppStatusBadge(app: App) {
		switch (app.status) {
			case 'online':
				return { text: 'Online', color: 'bg-green-100 text-green-800' };
			case 'offline':
				return { text: 'Offline', color: 'bg-red-100 text-red-800' };
			default:
				return { text: 'Unknown', color: 'bg-gray-100 text-gray-800' };
		}
	}
</script>

<div class="p-6">
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-3xl font-bold text-gray-900 dark:text-white">Applications</h1>
		<button
			onclick={() => (showCreateForm = !showCreateForm)}
			class="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700"
			disabled={getAvailableServers().length === 0}
		>
			{showCreateForm ? 'Cancel' : 'Add App'}
		</button>
	</div>

	{#if getAvailableServers().length === 0 && !showCreateForm}
		<div
			class="mb-6 rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800 dark:bg-yellow-900"
		>
			<div class="flex">
				<div class="flex-shrink-0">
					<span class="text-yellow-400">⚠️</span>
				</div>
				<div class="ml-3">
					<h3 class="text-sm font-medium text-yellow-800 dark:text-yellow-200">No Ready Servers</h3>
					<div class="mt-2 text-sm text-yellow-700 dark:text-yellow-300">
						<p>
							You need at least one server with setup and security lockdown completed before you can
							create apps.
						</p>
					</div>
				</div>
			</div>
		</div>
	{/if}

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
			<h2 class="mb-4 text-xl font-semibold dark:text-white">Add New Application</h2>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					createApp();
				}}
				class="space-y-4"
			>
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<div>
						<label for="app-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>App Name</label
						>
						<input
							id="app-name"
							bind:value={newApp.name}
							type="text"
							required
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="my-app"
						/>
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
							Used for directory and service naming
						</p>
					</div>
					<div>
						<label
							for="server-select"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300">Server</label
						>
						<select
							id="server-select"
							bind:value={newApp.server_id}
							required
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
						>
							<option value="">Select a server</option>
							{#each getAvailableServers() as server (server.id)}
								<option value={server.id}>{server.name} ({server.host})</option>
							{/each}
						</select>
					</div>
					<div class="md:col-span-2">
						<label for="domain" class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Domain</label
						>
						<input
							id="domain"
							bind:value={newApp.domain}
							type="text"
							required
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="myapp.example.com"
						/>
						<p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
							The domain where your app will be accessible
						</p>
					</div>
					<div>
						<label
							for="remote-path"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Remote Path (Optional)</label
						>
						<input
							id="remote-path"
							bind:value={newApp.remote_path}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="/opt/pocketbase/apps/{newApp.name || 'app-name'}"
						/>
					</div>
					<div>
						<label
							for="service-name"
							class="block text-sm font-medium text-gray-700 dark:text-gray-300"
							>Service Name (Optional)</label
						>
						<input
							id="service-name"
							bind:value={newApp.service_name}
							type="text"
							class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
							placeholder="pocketbase-{newApp.name || 'app-name'}"
						/>
					</div>
				</div>
				<div class="flex space-x-3">
					<button
						type="submit"
						class="rounded-lg bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700"
					>
						Create App
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
			<span class="ml-2 text-gray-600 dark:text-gray-400">Loading applications...</span>
		</div>
	{:else if apps.length === 0}
		<div class="py-12 text-center">
			<p class="mb-4 text-lg text-gray-500 dark:text-gray-400">No applications created yet</p>
			{#if getAvailableServers().length > 0}
				<button
					onclick={() => (showCreateForm = true)}
					class="rounded-lg bg-blue-600 px-6 py-3 font-medium text-white hover:bg-blue-700"
				>
					Create Your First App
				</button>
			{:else}
				<p class="text-sm text-gray-400 dark:text-gray-500">
					Set up a server first before creating apps
				</p>
			{/if}
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
								>Application</th
							>
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
								>Version</th
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
						{#each apps as app (app.id)}
							{@const statusBadge = getAppStatusBadge(app)}
							<tr class="hover:bg-gray-50 dark:hover:bg-gray-700">
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="flex items-center">
										<div>
											<div class="text-sm font-medium text-gray-900 dark:text-white">
												{app.name}
											</div>
											<div class="text-sm text-gray-500 dark:text-gray-400">
												<a
													href="https://{app.domain}"
													target="_blank"
													class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
												>
													{app.domain} 🔗
												</a>
											</div>
											<div class="text-xs text-gray-400 dark:text-gray-500">{app.service_name}</div>
										</div>
									</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="text-sm text-gray-900 dark:text-white">
										{getServerName(app.server_id)}
									</div>
									<div class="text-xs text-gray-500 dark:text-gray-400">{app.remote_path}</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<span
										class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusBadge.color}"
									>
										{getStatusIcon(app.status)}
										{statusBadge.text}
									</span>
								</td>
								<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
									{app.current_version || 'Not deployed'}
								</td>
								<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
									{formatTimestamp(app.created)}
								</td>
								<td class="space-x-2 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
									<button
										onclick={() => checkHealth(app.id)}
										disabled={checkingHealth.has(app.id)}
										class="text-green-600 hover:text-green-900 disabled:opacity-50 dark:text-green-400 dark:hover:text-green-300"
										title="Check health"
									>
										{checkingHealth.has(app.id) ? '🔄' : '💚'} Health
									</button>

									<button
										onclick={() => window.open(`https://${app.domain}`, '_blank')}
										class="text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"
										title="Open app"
									>
										🔗 Open
									</button>

									<button
										onclick={() => deleteApp(app.id)}
										class="text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"
										title="Delete app"
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
				Showing {apps.length} application{apps.length !== 1 ? 's' : ''}
			</p>
			<button
				onclick={loadApps}
				class="rounded bg-gray-100 px-3 py-1 text-sm text-gray-700 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
			>
				🔄 Refresh
			</button>
		</div>
	{/if}
</div>

<style>
	input[type='text'],
	select {
		border: 1px solid #d1d5db;
		border-radius: 0.375rem;
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
	}

	input[type='text']:focus,
	select:focus {
		outline: none;
		box-shadow: 0 0 0 2px #3b82f6;
		border-color: #3b82f6;
	}

	select {
		background-color: white;
	}

	:global([data-theme='dark']) input[type='text'],
	:global([data-theme='dark']) select {
		border-color: #4b5563;
		background-color: #374151;
		color: white;
	}

	:global([data-theme='dark']) input[type='text']:focus,
	:global([data-theme='dark']) select:focus {
		border-color: #3b82f6;
	}
</style>
