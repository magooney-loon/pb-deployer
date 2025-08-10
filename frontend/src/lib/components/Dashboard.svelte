<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { DashboardLogic, type DashboardState } from './logic/Dashboard.js';
	import type { Server, App } from '../api.js';
	import {
		ErrorAlert,
		LoadingSpinner,
		MetricCard,
		StatusBadge,
		Card,
		RecentItemsCard
	} from './partials/index.js';

	// Create logic instance
	const logic = new DashboardLogic();
	let state = $state<DashboardState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Get computed metrics
	let metrics = $derived(logic.getMetrics());

	onMount(async () => {
		await logic.loadData();
		logic.startAutoRefresh();
	});

	onDestroy(() => {
		logic.destroy();
	});
</script>

<div class="px-4 sm:px-0">
	<div class="mb-8">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Dashboard</h1>
				<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
					Overview of your PocketBase deployment infrastructure
				</p>
			</div>
			{#if !state.loading && state.refreshCounter > 0}
				<div class="text-right">
					<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
						Refreshes: {state.refreshCounter}
					</div>
					<div class="text-xs text-gray-500 dark:text-gray-400">
						Next refresh: {state.nextRefreshIn}s
					</div>
				</div>
			{/if}
		</div>
	</div>

	{#if state.error}
		<ErrorAlert message={state.error} onDismiss={() => logic.dismissError()} />
	{/if}

	{#if state.loading}
		<LoadingSpinner text="Loading dashboard..." />
	{:else}
		<!-- Metrics Cards -->
		<div class="mb-8 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
			<MetricCard title="Total Servers" value={metrics.totalServers} icon="🖥️" />
			<MetricCard
				title="Ready Servers"
				value={metrics.readyServers.length}
				icon="✅"
				color="green"
			/>
			<MetricCard title="Total Apps" value={metrics.totalApps} icon="📱" />
			<MetricCard title="Online Apps" value={metrics.onlineApps.length} icon="🟢" color="green" />
		</div>

		<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
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
					<div class="flex-1">
						<div class="flex items-center">
							<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
								{server.name}
							</span>
							{#if server.setup_complete}
								{#if server.security_locked}
									<StatusBadge status="Ready + Secured" variant="success" class="ml-2" />
								{:else}
									<StatusBadge status="Ready" variant="success" class="ml-2" />
								{/if}
							{:else}
								<StatusBadge status="New" variant="error" class="ml-2" />
							{/if}
						</div>
						<div class="text-xs text-gray-500 dark:text-gray-400">
							{server.host}:{server.port}
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
					<div class="flex-1">
						<div class="flex items-center">
							<span class="text-sm font-medium text-gray-900 dark:text-gray-100">
								{app.name}
							</span>
							<span class="ml-2 text-xs">
								{logic.getStatusIcon(app.status)}
							</span>
						</div>
						<div class="text-xs text-gray-500 dark:text-gray-400">
							<a
								href="https://{app.domain}"
								target="_blank"
								class="text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
							>
								{app.domain}
							</a>
						</div>
						{#if app.current_version}
							<div class="text-xs text-gray-400 dark:text-gray-500">
								v{app.current_version}
							</div>
						{/if}
					</div>
					<div class="text-right">
						<a
							href="https://{app.domain}"
							target="_blank"
							class="text-xs text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
						>
							Open →
						</a>
					</div>
				{/snippet}
			</RecentItemsCard>
		</div>

		<!-- Status Summary -->
		{#if logic.hasData()}
			<Card title="System Status" class="mt-8">
				<div class="grid grid-cols-1 gap-6 md:grid-cols-3">
					<div>
						<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
							Server Status
						</h4>
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
				</div>
			</Card>
		{/if}
	{/if}
</div>
