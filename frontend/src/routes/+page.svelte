<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type Server, type App, getStatusIcon } from '$lib/api.js';

	let servers = $state<Server[]>([]);
	let apps = $state<App[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		await loadData();
	});

	async function loadData() {
		try {
			loading = true;
			error = null;

			const [serversResponse, appsResponse] = await Promise.all([api.getServers(), api.getApps()]);

			servers = serversResponse.servers || [];
			apps = appsResponse.apps || [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load dashboard data';
			servers = [];
			apps = [];
		} finally {
			loading = false;
		}
	}

	// Computed values for dashboard metrics
	let readyServers = $derived(servers?.filter((s) => s.setup_complete && s.security_locked) || []);
	let onlineApps = $derived(apps?.filter((a) => a.status === 'online') || []);
	let recentServers = $derived(servers?.slice(0, 3) || []);
	let recentApps = $derived(apps?.slice(0, 5) || []);
</script>

<svelte:head>
	<title>Dashboard - PB Deployer</title>
	<meta name="description" content="PocketBase deployment dashboard" />
</svelte:head>

<div class="px-4 sm:px-0">
	<div class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900">Dashboard</h1>
		<p class="mt-2 text-sm text-gray-600">Overview of your PocketBase deployment infrastructure</p>
	</div>

	{#if error}
		<div class="mb-6 rounded-lg border border-red-200 bg-red-50 p-4">
			<div class="flex">
				<div class="flex-shrink-0">
					<span class="text-red-400">❌</span>
				</div>
				<div class="ml-3">
					<h3 class="text-sm font-medium text-red-800">Error</h3>
					<div class="mt-2 text-sm text-red-700">
						<p>{error}</p>
					</div>
					<div class="mt-4">
						<button
							onclick={() => (error = null)}
							class="rounded bg-red-100 px-3 py-1 text-sm text-red-800 hover:bg-red-200"
						>
							Dismiss
						</button>
					</div>
				</div>
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
			<span class="ml-2 text-gray-600">Loading dashboard...</span>
		</div>
	{:else}
		<!-- Metrics Cards -->
		<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
			<div class="overflow-hidden rounded-lg bg-white shadow">
				<div class="p-5">
					<div class="flex items-center">
						<div class="flex-shrink-0">
							<span class="text-2xl">🖥️</span>
						</div>
						<div class="ml-5 w-0 flex-1">
							<dl>
								<dt class="truncate text-sm font-medium text-gray-500">Total Servers</dt>
								<dd class="text-lg font-medium text-gray-900">{servers?.length || 0}</dd>
							</dl>
						</div>
					</div>
				</div>
			</div>

			<div class="overflow-hidden rounded-lg bg-white shadow">
				<div class="p-5">
					<div class="flex items-center">
						<div class="flex-shrink-0">
							<span class="text-2xl">✅</span>
						</div>
						<div class="ml-5 w-0 flex-1">
							<dl>
								<dt class="truncate text-sm font-medium text-gray-500">Ready Servers</dt>
								<dd class="text-lg font-medium text-gray-900">{readyServers.length}</dd>
							</dl>
						</div>
					</div>
				</div>
			</div>

			<div class="overflow-hidden rounded-lg bg-white shadow">
				<div class="p-5">
					<div class="flex items-center">
						<div class="flex-shrink-0">
							<span class="text-2xl">📱</span>
						</div>
						<div class="ml-5 w-0 flex-1">
							<dl>
								<dt class="truncate text-sm font-medium text-gray-500">Total Apps</dt>
								<dd class="text-lg font-medium text-gray-900">{apps?.length || 0}</dd>
							</dl>
						</div>
					</div>
				</div>
			</div>

			<div class="overflow-hidden rounded-lg bg-white shadow">
				<div class="p-5">
					<div class="flex items-center">
						<div class="flex-shrink-0">
							<span class="text-2xl">🟢</span>
						</div>
						<div class="ml-5 w-0 flex-1">
							<dl>
								<dt class="truncate text-sm font-medium text-gray-500">Online Apps</dt>
								<dd class="text-lg font-medium text-gray-900">{onlineApps.length}</dd>
							</dl>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Quick Actions -->
		<div class="mb-8 rounded-lg bg-white shadow">
			<div class="px-4 py-5 sm:p-6">
				<h3 class="mb-4 text-lg font-medium text-gray-900">Quick Actions</h3>
				<div class="flex flex-col gap-4 sm:flex-row">
					<a
						href="/servers"
						class="inline-flex items-center rounded-md border border-transparent bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:outline-none"
					>
						<span class="mr-2">🖥️</span>
						Manage Servers
					</a>
					<a
						href="/apps"
						class="inline-flex items-center rounded-md border border-transparent bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-green-700 focus:ring-2 focus:ring-green-500 focus:ring-offset-2 focus:outline-none"
					>
						<span class="mr-2">📱</span>
						Manage Apps
					</a>
					<button
						onclick={loadData}
						class="inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:outline-none"
					>
						<span class="mr-2">🔄</span>
						Refresh Data
					</button>
				</div>
			</div>
		</div>

		<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
			<!-- Recent Servers -->
			<div class="rounded-lg bg-white shadow">
				<div class="px-4 py-5 sm:p-6">
					<div class="mb-4 flex items-center justify-between">
						<h3 class="text-lg font-medium text-gray-900">Recent Servers</h3>
						<a href="/servers" class="text-sm text-blue-600 hover:text-blue-800">View all →</a>
					</div>
					{#if recentServers.length === 0}
						<div class="py-6 text-center">
							<p class="text-gray-500">No servers configured yet</p>
							<a
								href="/servers"
								class="mt-2 inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
							>
								Add your first server →
							</a>
						</div>
					{:else}
						<div class="space-y-3">
							{#each recentServers as server (server.id)}
								<div class="flex items-center justify-between rounded-lg bg-gray-50 p-3">
									<div class="flex-1">
										<div class="flex items-center">
											<span class="text-sm font-medium text-gray-900">{server.name}</span>
											{#if server.setup_complete && server.security_locked}
												<span class="ml-2 rounded bg-green-100 px-2 py-1 text-xs text-green-800"
													>Ready</span
												>
											{:else if server.setup_complete}
												<span class="ml-2 rounded bg-yellow-100 px-2 py-1 text-xs text-yellow-800"
													>Setup</span
												>
											{:else}
												<span class="ml-2 rounded bg-red-100 px-2 py-1 text-xs text-red-800"
													>New</span
												>
											{/if}
										</div>
										<div class="text-xs text-gray-500">{server.host}:{server.port}</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>

			<!-- Recent Apps -->
			<div class="rounded-lg bg-white shadow">
				<div class="px-4 py-5 sm:p-6">
					<div class="mb-4 flex items-center justify-between">
						<h3 class="text-lg font-medium text-gray-900">Recent Applications</h3>
						<a href="/apps" class="text-sm text-blue-600 hover:text-blue-800">View all →</a>
					</div>
					{#if recentApps.length === 0}
						<div class="py-6 text-center">
							<p class="text-gray-500">No apps created yet</p>
							{#if readyServers.length > 0}
								<a
									href="/apps"
									class="mt-2 inline-flex items-center text-sm text-blue-600 hover:text-blue-800"
								>
									Create your first app →
								</a>
							{:else}
								<p class="mt-2 text-xs text-gray-400">Set up a server first</p>
							{/if}
						</div>
					{:else}
						<div class="space-y-3">
							{#each recentApps as app (app.id)}
								<div class="flex items-center justify-between rounded-lg bg-gray-50 p-3">
									<div class="flex-1">
										<div class="flex items-center">
											<span class="text-sm font-medium text-gray-900">{app.name}</span>
											<span class="ml-2 text-xs">
												{getStatusIcon(app.status)}
											</span>
										</div>
										<div class="text-xs text-gray-500">
											<a
												href="https://{app.domain}"
												target="_blank"
												class="text-blue-600 hover:text-blue-800"
											>
												{app.domain}
											</a>
										</div>
										{#if app.current_version}
											<div class="text-xs text-gray-400">v{app.current_version}</div>
										{/if}
									</div>
									<div class="text-right">
										<a
											href="https://{app.domain}"
											target="_blank"
											class="text-xs text-blue-600 hover:text-blue-800"
										>
											Open →
										</a>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		</div>

		<!-- Status Summary -->
		{#if (servers?.length || 0) > 0 || (apps?.length || 0) > 0}
			<div class="mt-8 rounded-lg bg-white shadow">
				<div class="px-4 py-5 sm:p-6">
					<h3 class="mb-4 text-lg font-medium text-gray-900">System Status</h3>
					<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
						<div>
							<h4 class="mb-2 text-sm font-medium text-gray-500">Server Status</h4>
							<div class="space-y-1">
								<div class="flex justify-between text-sm">
									<span>Ready for deployment:</span>
									<span class="font-medium text-green-600">{readyServers.length}</span>
								</div>
								<div class="flex justify-between text-sm">
									<span>Setup required:</span>
									<span class="font-medium text-yellow-600"
										>{servers?.filter((s) => !s.setup_complete).length || 0}</span
									>
								</div>
								<div class="flex justify-between text-sm">
									<span>Security pending:</span>
									<span class="font-medium text-orange-600"
										>{servers?.filter((s) => s.setup_complete && !s.security_locked).length ||
											0}</span
									>
								</div>
							</div>
						</div>
						<div>
							<h4 class="mb-2 text-sm font-medium text-gray-500">Application Status</h4>
							<div class="space-y-1">
								<div class="flex justify-between text-sm">
									<span>Online:</span>
									<span class="font-medium text-green-600">{onlineApps.length}</span>
								</div>
								<div class="flex justify-between text-sm">
									<span>Offline:</span>
									<span class="font-medium text-red-600"
										>{apps?.filter((a) => a.status === 'offline').length || 0}</span
									>
								</div>
								<div class="flex justify-between text-sm">
									<span>Unknown:</span>
									<span class="font-medium text-gray-600"
										>{apps?.filter((a) => a.status !== 'online' && a.status !== 'offline').length ||
											0}</span
									>
								</div>
							</div>
						</div>
						<div>
							<h4 class="mb-2 text-sm font-medium text-gray-500">Deployment Info</h4>
							<div class="space-y-1">
								<div class="flex justify-between text-sm">
									<span>Apps deployed:</span>
									<span class="font-medium"
										>{apps?.filter((a) => a.current_version).length || 0}</span
									>
								</div>
								<div class="flex justify-between text-sm">
									<span>Pending deployment:</span>
									<span class="font-medium"
										>{apps?.filter((a) => !a.current_version).length || 0}</span
									>
								</div>
								<div class="flex justify-between text-sm">
									<span>Avg. uptime:</span>
									<span class="font-medium text-green-600">
										{onlineApps.length > 0 && (apps?.length || 0) > 0
											? Math.round((onlineApps.length / (apps?.length || 1)) * 100)
											: 0}%
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>
