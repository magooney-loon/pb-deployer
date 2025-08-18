<script lang="ts">
	import { onMount } from 'svelte';
	import { DashboardLogic, type DashboardState } from './Dashboard.js';
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

	const logic = new DashboardLogic();
	let state = $state<DashboardState>(logic.getState());

	logic.onStateUpdate((newState) => {
		state = newState;
	});

	let metrics = $derived(logic.getMetrics());

	onMount(async () => {
		await logic.loadData();
	});
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

{#if state.error}
	<Toast message={state.error} onDismiss={() => logic.dismissError()} />
{/if}

{#if state.loading}
	<LoadingSpinner text="Loading dashboard..." />
{:else}
	<!-- Metrics Cards -->
	<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-5">
		<MetricCard title="Total Servers" value={metrics.totalServers}>
			{#snippet iconSnippet()}
				<Icon name="servers" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Ready Servers" value={metrics.readyServers.length} color="green">
			{#snippet iconSnippet()}
				<Icon name="checkmark" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Total Apps" value={metrics.totalApps}>
			{#snippet iconSnippet()}
				<Icon name="apps" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Online Apps" value={metrics.onlineApps.length} color="green">
			{#snippet iconSnippet()}
				<Icon name="green-circle" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
		<MetricCard title="Updates Available" value={metrics.updateInfo.appsWithUpdates} color="purple">
			{#snippet iconSnippet()}
				<Icon name="upload" size="h-6 w-6" />
			{/snippet}
		</MetricCard>
	</div>

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
		<!-- Recent Servers -->
		<RecentItemsCard
			title="Recent Servers"
			items={metrics.recentServers}
			viewAllHref="/servers"
			emptyState={{
				message: 'No servers configured yet',
				ctaText: 'Add your first server →',
				ctaHref: '/servers'
			}}
		>
			{#snippet children(server: Server)}
				{@const serverBadge = logic.getServerStatusBadge(server)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							{server.name}
						</span>
						<StatusBadge status={serverBadge.text} variant={serverBadge.variant} class="ml-2" />
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						{server.host}:{server.port}
					</div>
				</div>
				<div class="text-right">
					<div class="text-xs text-gray-400 dark:text-gray-500">
						Created {new Date(server.created).toLocaleDateString()}
					</div>
				</div>
			{/snippet}
		</RecentItemsCard>

		<!-- Recent Apps -->
		<RecentItemsCard
			title="Recent Applications"
			items={metrics.recentApps}
			viewAllHref="/apps"
			emptyState={{
				message: 'No apps created yet',
				ctaText: metrics.readyServers.length > 0 ? 'Create your first app →' : undefined,
				ctaHref: metrics.readyServers.length > 0 ? '/apps' : undefined,
				secondaryText: metrics.readyServers.length === 0 ? 'Set up a server first' : undefined
			}}
		>
			{#snippet children(app: App)}
				{@const appBadge = logic.getAppStatusBadge(app)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							{app.name}
						</span>
						<StatusBadge
							status="{logic.getStatusIcon(app.status)} {appBadge.text}"
							variant={appBadge.variant}
							class="ml-2"
							dot
						/>
						{#if app.latest_version && logic.hasUpdateAvailable(app.current_version, app.latest_version)}
							<StatusBadge status="Update" variant="update" size="xs" class="ml-1" />
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
					<div class="text-xs text-gray-500 dark:text-gray-400">
						{app.service_name}
					</div>
					{#if app.current_version}
						<div class="text-xs text-gray-400 dark:text-gray-500">
							v{app.current_version}
							{#if app.latest_version && app.current_version !== app.latest_version}
								<span class="text-purple-500">→ v{app.latest_version}</span>
							{/if}
						</div>
						<div class="text-xs text-gray-400 dark:text-gray-500">
							Created {new Date(app.created).toLocaleDateString()}
						</div>
					{:else}
						<div class="text-xs text-gray-400 dark:text-gray-500">
							{#if app.latest_version}
								v{app.latest_version} ready •
							{/if}Created {new Date(app.created).toLocaleDateString()}
						</div>
					{/if}
				</div>
			{/snippet}
		</RecentItemsCard>

		<!-- Recent Deployments -->
		<RecentItemsCard
			title="Recent Deployments"
			items={metrics.recentDeployments}
			viewAllHref="/deployments"
			emptyState={{
				message: 'No deployments yet',
				ctaText: metrics.totalApps > 0 ? 'Deploy an app →' : undefined,
				ctaHref: metrics.totalApps > 0 ? '/apps' : undefined,
				secondaryText: metrics.totalApps === 0 ? 'Create an app first' : undefined
			}}
		>
			{#snippet children(deployment: Deployment)}
				{@const deploymentBadge = logic.getDeploymentStatusBadge(deployment)}
				<div class="flex-1">
					<div class="flex items-center">
						<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
							Deployment #{deployment.id.slice(-8)}
						</span>
						<StatusBadge
							status={deploymentBadge.text}
							variant={deploymentBadge.variant}
							class="ml-2"
						/>
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						App ID: {deployment.app_id.slice(-8)}
					</div>
				</div>
				<div class="text-right">
					{#if deployment.version_id}
						<div class="text-xs text-gray-500 dark:text-gray-400">
							Version: {deployment.version_id.slice(-8)}
						</div>
					{/if}
					<div class="text-xs text-gray-400 dark:text-gray-500">
						{new Date(deployment.created).toLocaleDateString()} at {new Date(
							deployment.created
						).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
					</div>
				</div>
			{/snippet}
		</RecentItemsCard>
	</div>

	<!-- Status Summary -->
	{#if logic.hasData()}
		<Card title="System Status" class="mt-8">
			<div class="grid grid-cols-1 gap-6 md:grid-cols-4">
				<div>
					<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">Server Status</h4>
					<div class="space-y-2">
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Ready for deployment:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{metrics.serverStatusCounts.ready}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Setup required:</span>
							<span class="font-semibold text-amber-600 dark:text-amber-400">
								{metrics.serverStatusCounts.setupRequired}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Security available:</span>
							<span class="font-semibold text-blue-600 dark:text-blue-400">
								{metrics.serverStatusCounts.securityOptional}
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
								{metrics.appStatusCounts.online}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Offline:</span>
							<span class="font-semibold text-red-600 dark:text-red-400">
								{metrics.appStatusCounts.offline}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Unknown:</span>
							<span class="font-semibold text-gray-600 dark:text-gray-400">
								{metrics.appStatusCounts.unknown}
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
							<span class="text-gray-600 dark:text-gray-400">Apps deployed:</span>
							<span class="font-semibold text-gray-900 dark:text-gray-100">
								{metrics.deploymentInfo.appsDeployed}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Pending deployment:</span>
							<span class="font-semibold text-gray-900 dark:text-gray-100">
								{metrics.deploymentInfo.pendingDeployment}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Avg. uptime:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{metrics.deploymentInfo.averageUptime}%
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
								{metrics.updateInfo.appsWithUpdates}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Up to date:</span>
							<span class="font-semibold text-emerald-600 dark:text-emerald-400">
								{metrics.totalApps - metrics.updateInfo.appsWithUpdates}
							</span>
						</div>
						<div class="flex justify-between text-sm">
							<span class="text-gray-600 dark:text-gray-400">Coverage:</span>
							<span class="font-semibold text-gray-900 dark:text-gray-100">
								{metrics.totalApps > 0
									? Math.round(
											((metrics.totalApps - metrics.updateInfo.appsWithUpdates) /
												metrics.totalApps) *
												100
										)
									: 100}%
							</span>
						</div>
					</div>
				</div>
			</div>
		</Card>
	{/if}
{/if}
