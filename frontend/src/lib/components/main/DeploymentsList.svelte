<script lang="ts">
	import { onMount } from 'svelte';
	import { deploymentsStore, appsStore, versionsStore } from '$lib/stores';
	import LogsModal from '$lib/components/modals/LogsModal.svelte';
	import DeploymentCreateModal from '$lib/components/modals/DeploymentCreateModal.svelte';
	import DeploymentModal from '$lib/components/modals/DeploymentModal.svelte';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import { Button, Toast, EmptyState, LoadingSpinner, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { getDeploymentStatusBadge, formatTimestamp } from '$lib/components/partials/index.js';
	import type { Deployment } from '$lib/api/index.js';

	onMount(async () => {
		await Promise.all([
			deploymentsStore.initialized ? null : deploymentsStore.load(),
			appsStore.initialized ? null : appsStore.load(),
			versionsStore.initialized ? null : versionsStore.load()
		]);
	});

	// Modal local state
	let showCreateModal = $state(false);
	let showDeleteModal = $state(false);
	let deploymentToDelete = $state<{ id: string; name: string } | null>(null);
	let deleting = $state(false);

	let showDeployModal = $state(false);
	let deploymentToDeploy = $state<Deployment | null>(null);
	let deploying = $state(false);

	let showLogsModal = $state(false);
	let deploymentToShowLogs = $state<Deployment | null>(null);
	let autoOpenedLogsModal = $state(false);

	let error = $state<string | null>(null);

	// Keep logs modal in sync with realtime updates
	$effect(() => {
		if (deploymentToShowLogs) {
			const fresh = deploymentsStore.byId(deploymentToShowLogs.id);
			if (fresh) deploymentToShowLogs = fresh;
		}
	});

	let pendingDeployments = $derived(
		deploymentsStore.deployments.filter((d) => ['pending', 'running'].includes(d.status))
	);
	let runningDeployments = $derived(
		deploymentsStore.deployments.filter((d) => d.status === 'running')
	);

	function getAppName(deployment: Deployment) {
		return deployment.expand?.app_id?.name || 'Unknown App';
	}

	function getAppDomain(deployment: Deployment) {
		return deployment.expand?.app_id?.domain || '';
	}

	function getVersionNumber(deployment: Deployment) {
		return deployment.expand?.version_id?.version_number || 'N/A';
	}

	function getVersionNotes(deployment: Deployment) {
		return deployment.expand?.version_id?.notes || '';
	}

	function formatDuration(startedAt?: string, completedAt?: string): string | null {
		if (!startedAt || !completedAt) return null;
		const diff = new Date(completedAt).getTime() - new Date(startedAt).getTime();
		if (diff < 1000) return '< 1s';
		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) return `${hours}h ${minutes % 60}m`;
		if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
		return `${seconds}s`;
	}

	function getRunningDuration(deployment: Deployment): string | null {
		if (!deployment.started_at || deployment.status !== 'running') return null;
		const diff = Date.now() - new Date(deployment.started_at).getTime();
		const seconds = Math.floor(diff / 1000);
		const minutes = Math.floor(seconds / 60);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) return `${hours}h ${minutes % 60}m`;
		if (minutes > 0) return `${minutes}m ${seconds % 60}s`;
		return `${seconds}s`;
	}

	function getDeploymentDisplayName(deployment: Deployment) {
		return `${getAppName(deployment)} - v${getVersionNumber(deployment)}`;
	}

	function isLogsModalClosable() {
		if (!autoOpenedLogsModal || !deploymentToShowLogs) return true;
		return !(
			['pending', 'running'].includes(deploymentToShowLogs.status) ||
			deploymentsStore.isDeploying(deploymentToShowLogs.id)
		);
	}

	function openLogsModal(deployment: Deployment) {
		deploymentToShowLogs = deployment;
		autoOpenedLogsModal = false;
		showLogsModal = true;
	}

	function closeLogsModal() {
		if (!isLogsModalClosable()) return;
		showLogsModal = false;
		autoOpenedLogsModal = false;
		setTimeout(() => {
			deploymentToShowLogs = null;
		}, 300);
	}

	function openCreateModal() {
		showCreateModal = true;
	}

	function closeCreateModal() {
		showCreateModal = false;
	}

	async function createDeployment(data: { app_id: string; version_id: string }) {
		try {
			await deploymentsStore.create(data);
			showCreateModal = false;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create deployment';
		}
	}

	function openDeleteModal(deployment: Deployment) {
		deploymentToDelete = { id: deployment.id, name: getDeploymentDisplayName(deployment) };
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		setTimeout(() => {
			deploymentToDelete = null;
		}, 300);
	}

	async function confirmDelete(id: string) {
		deleting = true;
		try {
			await deploymentsStore.remove(id);
			showDeleteModal = false;
			deploymentToDelete = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete deployment';
		} finally {
			deleting = false;
		}
	}

	function openDeployModal(deployment: Deployment) {
		deploymentToDeploy = deployment;
		showDeployModal = true;
	}

	function closeDeployModal() {
		showDeployModal = false;
		setTimeout(() => {
			deploymentToDeploy = null;
		}, 300);
	}

	async function deployFromModal(
		deploymentId: string,
		isInitialDeploy: boolean,
		superuserEmail?: string,
		superuserPass?: string
	) {
		deploying = true;
		error = null;
		try {
			const deployment = deploymentsStore.byId(deploymentId);
			await deploymentsStore.deploy(deploymentId, isInitialDeploy, superuserEmail, superuserPass);

			if (deployment) {
				deploymentToShowLogs = deployment;
				autoOpenedLogsModal = true;
				showLogsModal = true;
				showDeployModal = false;
				deploymentToDeploy = null;
			}
			await deploymentsStore.load();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to start deployment';
		} finally {
			deploying = false;
		}
	}

	function hasPendingDeployment(appId: string, versionId: string): boolean {
		return deploymentsStore.deployments.some(
			(d) =>
				d.app_id === appId &&
				d.version_id === versionId &&
				['pending', 'running'].includes(d.status)
		);
	}

	function getVersionsWithPendingStatus(appId: string) {
		return versionsStore.byApp(appId).map((v) => {
			const pending = deploymentsStore.deployments.find(
				(d) =>
					d.app_id === appId && d.version_id === v.id && ['pending', 'running'].includes(d.status)
			);
			return { ...v, hasPending: !!pending, pendingDeployment: pending };
		});
	}
</script>

<header class="mb-8 flex items-center justify-between">
	<div>
		<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Deployments</h1>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
			Monitor deployment history and status
		</p>
	</div>
	<Button
		variant="outline"
		onclick={openCreateModal}
		disabled={deploymentsStore.loading || deploying || deleting}
	>
		{#snippet iconSnippet()}
			<Icon name="rocket" />
		{/snippet}
		Deploy
	</Button>
</header>

{#if !deploymentsStore.loading && pendingDeployments.length > 0}
	<div
		class="mb-6 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
	>
		<div class="flex items-center space-x-2">
			<Icon name="warning" class="text-amber-600 dark:text-amber-400" />
			<div class="flex-1">
				<h3 class="text-sm font-semibold text-amber-900 dark:text-amber-100">Active Deployments</h3>
				<p class="text-xs text-amber-800 dark:text-amber-200">
					{pendingDeployments.length} pending
					{#if runningDeployments.length > 0}
						• {runningDeployments.length} running
					{/if}
				</p>
			</div>
		</div>
		{#if pendingDeployments.length > 0}
			<div class="mt-3 space-y-1">
				{#each pendingDeployments.slice(0, 3) as deployment (deployment.id)}
					<div class="flex items-center justify-between text-xs">
						<span class="text-amber-800 dark:text-amber-200">
							{getAppName(deployment)} v{getVersionNumber(deployment)}
						</span>
						<span class="capitalize text-amber-600 dark:text-amber-400">
							{deployment.status}
						</span>
					</div>
				{/each}
				{#if pendingDeployments.length > 3}
					<div class="text-xs text-amber-700 dark:text-amber-300">
						+{pendingDeployments.length - 3} more...
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}

{#if error}
	<Toast message={error} type="error" onDismiss={() => (error = null)} />
{/if}

{#if deploymentsStore.error}
	<Toast
		message={deploymentsStore.error}
		type="error"
		onDismiss={() => deploymentsStore.clearError()}
	/>
{/if}

{#if deploymentsStore.loading}
	<LoadingSpinner text="Loading deployments..." />
{:else if deploymentsStore.deployments.length === 0}
	<EmptyState
		title="No deployments found"
		description="Create your first deployment to see deployment history here"
	>
		{#snippet iconSnippet()}
			<Icon name="rocket" size="h-12 w-12" class="text-gray-400" />
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
							>Version</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Status</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Duration</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Started</th
						>
						<th
							class="px-6 py-3 text-right text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Actions</th
						>
					</tr>
				</thead>
				<tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-950">
					{#each deploymentsStore.deployments as deployment (deployment.id)}
						{@const statusBadge = getDeploymentStatusBadge(deployment)}
						{@const appName = getAppName(deployment)}
						{@const appDomain = getAppDomain(deployment)}
						{@const versionNumber = getVersionNumber(deployment)}
						{@const versionNotes = getVersionNotes(deployment)}
						{@const duration = formatDuration(deployment.started_at, deployment.completed_at)}
						{@const runningDuration = getRunningDuration(deployment)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div>
									<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
										{appName}
									</div>
									{#if appDomain}
										<div class="text-sm text-gray-500 dark:text-gray-400">
											<a
												href="https://{appDomain}"
												target="_blank"
												class="inline-flex items-center space-x-1 text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
											>
												<span>{appDomain}</span>
												<Icon name="link" size="h-3 w-3" />
											</a>
										</div>
									{/if}
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
									v{versionNumber}
								</div>
								{#if versionNotes}
									<div class="max-w-xs truncate text-xs text-gray-500 dark:text-gray-400">
										{versionNotes}
									</div>
								{/if}
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<StatusBadge status={statusBadge.text} variant={statusBadge.variant} dot />
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								{#if deployment.status === 'running' && runningDuration}
									<span class="text-blue-600 dark:text-blue-400">{runningDuration}</span>
								{:else if duration}
									<span>{duration}</span>
								{:else}
									<span class="text-gray-400">Not Deployed</span>
								{/if}
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div class="flex flex-col">
									<span>
										{deployment.started_at ? formatTimestamp(deployment.started_at) : 'Not started'}
									</span>
									{#if deployment.completed_at}
										<span class="text-xs text-gray-400 dark:text-gray-500">
											Completed {formatTimestamp(deployment.completed_at)}
										</span>
									{/if}
								</div>
							</td>
							<td class="space-x-1 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								{#if deployment.status === 'pending'}
									<Button
										variant="ghost"
										color="blue"
										size="sm"
										disabled={deleting || deploymentsStore.isDeploying(deployment.id)}
										onclick={() => openDeployModal(deployment)}
									>
										{#snippet iconSnippet()}
											<Icon name="rocket" />
										{/snippet}
										{deploymentsStore.isDeploying(deployment.id) ? 'Deploying...' : 'Deploy'}
									</Button>

									<Button
										variant="ghost"
										color="red"
										size="sm"
										disabled={deleting || deploymentsStore.isDeploying(deployment.id)}
										onclick={() => openDeleteModal(deployment)}
									>
										{#snippet iconSnippet()}
											<Icon name="delete" />
										{/snippet}
										Delete
									</Button>
								{/if}

								<Button
									variant="ghost"
									color="blue"
									size="sm"
									onclick={() => openLogsModal(deployment)}
									disabled={!deployment.logs}
								>
									{#snippet iconSnippet()}
										<Icon name="eye" />
									{/snippet}
									Logs
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
			Showing {deploymentsStore.deployments.length} deployment{deploymentsStore.deployments.length !== 1 ? 's' : ''}
		</p>
		<Button
			variant="outline"
			size="sm"
			onclick={() => deploymentsStore.load()}
			disabled={deploymentsStore.loading || deploying || deleting}
		>
			{#snippet iconSnippet()}
				<Icon name="refresh" />
			{/snippet}
			Refresh
		</Button>
	</div>
{/if}

<!-- Delete Deployment Modal -->
<DeleteModal
	open={showDeleteModal}
	item={deploymentToDelete}
	itemType="deployment"
	loading={deleting}
	onclose={closeDeleteModal}
	onconfirm={(id) => confirmDelete(id)}
/>

<!-- Deployment Create Modal -->
<DeploymentCreateModal
	open={showCreateModal}
	apps={appsStore.apps}
	versions={versionsStore.versions}
	creating={false}
	hasPendingDeployment={(appId, versionId) => hasPendingDeployment(appId, versionId)}
	getVersionsWithPendingStatus={(appId) => getVersionsWithPendingStatus(appId)}
	onclose={closeCreateModal}
	oncreate={(data) => createDeployment(data)}
/>

<!-- Deployment Modal -->
<DeploymentModal
	open={showDeployModal}
	deployment={deploymentToDeploy}
	app={deploymentToDeploy
		? appsStore.apps.find((a) => a.id === deploymentToDeploy?.app_id) || null
		: null}
	version={deploymentToDeploy
		? versionsStore.versions.find((v) => v.id === deploymentToDeploy?.version_id) || null
		: null}
	deployments={deploymentsStore.deployments}
	{deploying}
	onclose={closeDeployModal}
	ondeploy={(deploymentId, isInitialDeploy, superuserEmail, superuserPass) =>
		deployFromModal(deploymentId, isInitialDeploy, superuserEmail, superuserPass)}
/>

<!-- Logs Modal -->
<LogsModal
	open={showLogsModal}
	deployment={deploymentToShowLogs}
	closable={isLogsModalClosable()}
	autoOpened={autoOpenedLogsModal}
	onclose={closeLogsModal}
/>
