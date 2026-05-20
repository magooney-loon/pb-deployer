<script lang="ts">
	import { onMount } from 'svelte';
	import { appsStore, serversStore, deploymentsStore } from '$lib/stores';
	import type { Server, App } from '$lib/api/index.js';
	import type { Deployment } from '$lib/api/deployment/types.js';
	import {
		Toast,
		LoadingSpinner,
		MetricCard,
		StatusBadge,
		Card,
		RecentItemsCard
	} from '$lib/components/partials/index.js';
	import Icon from '$lib/components/icons/Icon.svelte';
	import {
		getServerStatusBadge,
		getAppStatusBadge,
		getDeploymentStatusBadge,
		hasUpdateAvailable
	} from '$lib/components/partials/index.js';

	onMount(async () => {
		await Promise.all([
			appsStore.initialized ? null : appsStore.load(),
			serversStore.initialized ? null : serversStore.load(),
			deploymentsStore.initialized ? null : deploymentsStore.load()
		]);
	});

	let loading = $derived(appsStore.loading || serversStore.loading || deploymentsStore.loading);
	let error = $derived(appsStore.error || serversStore.error || deploymentsStore.error);

	let readyServers = $derived(serversStore.servers.filter((s) => s.setup_complete));
	let onlineApps = $derived(appsStore.apps.filter((a) => a.status === 'online'));

	let appsWithUpdates = $derived(
		appsStore.apps.filter(
			(a) =>
				a.latest_version &&
				a.deployed_version &&
				hasUpdateAvailable(a.deployed_version, a.latest_version)
		)
	);

	let recentServers = $derived(serversStore.servers.slice(0, 3));
	let recentApps = $derived(appsStore.apps.slice(0, 5));
	let recentDeployments = $derived(deploymentsStore.deployments.slice(0, 3));

	let serverStatusCounts = $derived({
		ready: readyServers.length,
		setupRequired: serversStore.servers.filter((s) => !s.setup_complete).length,
		securityOptional: serversStore.servers.filter((s) => s.setup_complete && !s.security_locked)
			.length
	});

	let appStatusCounts = $derived({
		online: onlineApps.length,
		offline: appsStore.apps.filter((a) => a.status === 'offline').length,
		unknown: appsStore.apps.filter((a) => a.status !== 'online' && a.status !== 'offline').length,
	});

	let failedDeployments = $derived(
		deploymentsStore.deployments.filter((d) => d.status === 'failed').length
	);
	let successfulDeployments = $derived(
		deploymentsStore.deployments.filter((d) => d.status === 'success').length
	);
	let pendingDeployments = $derived(
		deploymentsStore.deployments.filter((d) => ['pending', 'running'].includes(d.status)).length
	);

	let hasData = $derived(serversStore.servers.length > 0 || appsStore.apps.length > 0);

	function dismissError() {
		appsStore.clearError();
		serversStore.clearError();
		deploymentsStore.clearError();
	}
</script>

<header class="mb-8">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Dashboard</h1>
			<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
				Overview of your PocketBase deployment infrastructure
			</p>
		</div>
	</div>
</header>

{#if error}
	<Toast message={error} onDismiss={dismissError} />
{/if}

{#if loading}
	<LoadingSpinner text="Loading dashboard..." />
{:else}
	<!-- Metrics Cards -->
	<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-6">
		<MetricCard title="Total Servers" value={serversStore.servers.length}>
			{#snippet iconSnippet()}
				<Icon name="servers" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Ready Servers" value={readyServers.length} color="green">
			{#snippet iconSnippet()}
				<Icon name="checkmark" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Total Apps" value={appsStore.apps.length}>
			{#snippet iconSnippet()}
				<Icon name="apps" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Online Apps" value={onlineApps.length} color="green">
			{#snippet iconSnippet()}
				<Icon name="green-circle" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Updates Available" value={appsWithUpdates.length} color="purple">
			{#snippet iconSnippet()}
				<Icon name="upload" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
	</div>

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
		<!-- Recent Servers -->
		<RecentItemsCard
			title="Recent Servers"
			items={recentServers}
			viewAllHref="/servers"
			emptyState={{
				message: 'No servers configured yet',
				ctaText: 'Add your first server →',
				ctaHref: '/servers'
			}}
		>
			{#snippet children(server: Server)}
				{@const serverBadge = getServerStatusBadge(server)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							{server.name}
						</span>
						<StatusBadge status={serverBadge.text} variant={serverBadge.variant} class="ml-2" dot />
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						{server.host}:{server.port}
					</div>
				</div>
				<div class="text-right">
					<div class="text-xs text-gray-400 dark:text-gray-500">
						{new Date(server.created).toLocaleDateString()}
					</div>
				</div>
			{/snippet}
		</RecentItemsCard>

		<!-- Recent Apps -->
		<RecentItemsCard
			title="Recent Applications"
			items={recentApps}
			viewAllHref="/apps"
			emptyState={{
				message: 'No apps created yet',
				ctaText: readyServers.length > 0 ? 'Create your first app →' : undefined,
				ctaHref: readyServers.length > 0 ? '/apps' : undefined,
				secondaryText: readyServers.length === 0 ? 'Set up a server first' : undefined
			}}
		>
			{#snippet children(app: App)}
				{@const appBadge = getAppStatusBadge(app, app.latest_version)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							{app.name}
						</span>
						<StatusBadge status={appBadge.text} variant={appBadge.variant} class="ml-2" dot />
						{#if app.latest_version && app.deployed_version && hasUpdateAvailable(app.deployed_version, app.latest_version)}
							<StatusBadge status="Update" variant="update" size="xs" class="ml-1" dot />
						{/if}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
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
				<div class="text-right">
					{#if app.deployed_version}
						<div class="text-xs text-gray-500 dark:text-gray-400">
							v{app.deployed_version}
							{#if app.latest_version && app.deployed_version !== app.latest_version}
								<span class="text-purple-500">→ v{app.latest_version}</span>
							{/if}
						</div>
					{:else if app.latest_version}
						<div class="text-xs text-gray-500 dark:text-gray-400">v{app.latest_version} ready</div>
					{:else}
						<div class="text-xs text-gray-400 dark:text-gray-500">No versions</div>
					{/if}
					<div class="text-xs text-gray-400 dark:text-gray-500">
						{new Date(app.created).toLocaleDateString()}
					</div>
				</div>
			{/snippet}
		</RecentItemsCard>

		<!-- Recent Deployments -->
		<RecentItemsCard
			title="Recent Deployments"
			items={recentDeployments}
			viewAllHref="/deployments"
			emptyState={{
				message: 'No deployments yet',
				ctaText: appsStore.apps.length > 0 ? 'Deploy an app →' : undefined,
				ctaHref: appsStore.apps.length > 0 ? '/deployments' : undefined,
				secondaryText: appsStore.apps.length === 0 ? 'Create an app first' : undefined
			}}
		>
			{#snippet children(deployment: Deployment)}
				{@const deploymentBadge = getDeploymentStatusBadge(deployment)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							{deployment.expand?.app_id?.name || 'Unknown App'}
						</span>
						<StatusBadge
							status={deploymentBadge.text}
							variant={deploymentBadge.variant}
							class="ml-2"
							dot
						/>
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						{#if deployment.expand?.app_id?.domain}
							<a
								href="https://{deployment.expand.app_id.domain}"
								target="_blank"
								class="inline-flex items-center space-x-1 text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
							>
								<span>{deployment.expand.app_id.domain}</span>
								<Icon name="link" size="h-3 w-3" />
							</a>
						{:else}
							Unknown Domain
						{/if}
					</div>
				</div>
				<div class="text-right">
					{#if deployment.expand?.version_id?.version_number}
						<div class="text-xs text-gray-500 dark:text-gray-400">
							v{deployment.expand.version_id.version_number}
						</div>
					{/if}
					<div class="text-xs text-gray-400 dark:text-gray-500">
						{new Date(deployment.started_at || deployment.created).toLocaleDateString()}
					</div>
				</div>
			{/snippet}
		</RecentItemsCard>
	</div>

	<!-- Status Summary -->
	{#if hasData}
		<Card title="System Status" class="mt-8">
			<div class="grid grid-cols-1 gap-6 md:grid-cols-4">
				<div>
					<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">Server Status</h4>
					<div class="space-y-2">
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Ready for deployment:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{serverStatusCounts.ready}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Setup required:</span>
							<span class="font-semibold text-amber-600 dark:text-amber-400">
								{serverStatusCounts.setupRequired}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Security available:</span>
							<span class="font-semibold text-blue-600 dark:text-blue-400">
								{serverStatusCounts.securityOptional}
							</span>
						</div>
					</div>
				</div>
				<div>
					<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
						Application Status
					</h4>
					<div class="space-y-2">
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Online:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{appStatusCounts.online}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Offline:</span>
							<span class="font-semibold text-red-600 dark:text-red-400">
								{appStatusCounts.offline}
							</span>
						</div>
					</div>
				</div>
				<div>
					<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">Update Status</h4>
					<div class="space-y-2">
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Updates available:</span>
							<span class="font-semibold text-purple-600 dark:text-purple-400">
								{appsWithUpdates.length}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Up to date:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{appsStore.apps.length - appsWithUpdates.length}
							</span>
						</div>
					</div>
				</div>
				<div>
					<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
						Deployment Info
					</h4>
					<div class="space-y-2">
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Successful:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{successfulDeployments}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Pending:</span>
							<span class="font-semibold text-amber-600 dark:text-amber-400">
								{pendingDeployments}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Failed:</span>
							<span class="font-semibold text-red-600 dark:text-red-400">
								{failedDeployments}
							</span>
						</div>
					</div>
				</div>
			</div>
		</Card>
	{/if}
{/if}
