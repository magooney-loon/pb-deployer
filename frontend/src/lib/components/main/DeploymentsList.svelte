<script lang="ts">
	import { onMount } from 'svelte';
	import { DeploymentsListLogic, type DeploymentsListState } from './DeploymentsList.js';
	import LogsModal from '$lib/components/modals/LogsModal.svelte';
	import DeploymentCreateModal from '$lib/components/modals/DeploymentCreateModal.svelte';
	import DeploymentModal from '$lib/components/modals/DeploymentModal.svelte';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import { Button, Toast, EmptyState, LoadingSpinner, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';

	const logic = new DeploymentsListLogic();
	let state = $state<DeploymentsListState>(logic.getState());

	logic.onStateUpdate((newState) => {
		state = newState;
	});

	onMount(async () => {
		await logic.initialize();
	});
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
		onclick={() => logic.openCreateModal()}
		disabled={state.loading || state.creating || state.deleting || state.deploying}
	>
		{#snippet iconSnippet()}
			<Icon name="rocket" />
		{/snippet}
		Deploy
	</Button>
</header>

<!-- Pending Deployments Summary -->
{#if !state.loading && state.deployments.some((d) => ['pending', 'running'].includes(d.status))}
	{@const pendingDeployments = state.deployments.filter((d) =>
		['pending', 'running'].includes(d.status)
	)}
	{@const runningDeployments = state.deployments.filter((d) => d.status === 'running')}
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
							{logic.getAppName(deployment)} v{logic.getVersionNumber(deployment)}
						</span>
						<span class="text-amber-600 capitalize dark:text-amber-400">
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

{#if state.error}
	<Toast message={state.error} type="error" onDismiss={() => logic.dismissError()} />
{/if}

{#if state.loading}
	<LoadingSpinner text="Loading deployments..." />
{:else if state.deployments.length === 0}
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
					{#each state.deployments as deployment (deployment.id)}
						{@const statusBadge = logic.getDeploymentStatusBadge(deployment)}
						{@const appName = logic.getAppName(deployment)}
						{@const appDomain = logic.getAppDomain(deployment)}
						{@const versionNumber = logic.getVersionNumber(deployment)}
						{@const versionNotes = logic.getVersionNotes(deployment)}
						{@const duration = logic.formatDuration(deployment.started_at, deployment.completed_at)}
						{@const runningDuration = logic.getRunningDuration(deployment)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div class="flex items-center">
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
								{#if logic.isDeploymentRunning(deployment) && runningDuration}
									<span class="text-blue-600 dark:text-blue-400">{runningDuration}</span>
								{:else if duration}
									<span>{duration}</span>
								{:else}
									<span class="text-gray-400">Not Deployed</span>
								{/if}
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div class="flex flex-col">
									<span
										>{deployment.started_at
											? logic.formatTimestamp(deployment.started_at)
											: 'Not started'}</span
									>
									{#if deployment.completed_at}
										<span class="text-xs text-gray-400 dark:text-gray-500">
											Completed {logic.formatTimestamp(deployment.completed_at)}
										</span>
									{/if}
								</div>
							</td>
							<td class="space-x-1 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								{#if logic.isPendingDeployment(deployment)}
									<Button
										variant="ghost"
										color="blue"
										size="sm"
										loading={logic.isDeploymentInProgress(deployment.id)}
										disabled={state.deleting ||
											state.creating ||
											state.deploying ||
											logic.isDeploymentInProgress(deployment.id)}
										onclick={() => logic.openDeployModal(deployment)}
									>
										{#snippet iconSnippet()}
											<Icon name="rocket" />
										{/snippet}
										{logic.isDeploymentInProgress(deployment.id) ? 'Deploying...' : 'Deploy'}
									</Button>

									<Button
										variant="ghost"
										color="red"
										size="sm"
										disabled={state.deleting ||
											state.creating ||
											state.deploying ||
											logic.isDeploymentInProgress(deployment.id)}
										onclick={() => logic.deleteDeployment(deployment)}
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
									onclick={() => logic.openLogsModal(deployment)}
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
			Showing {state.deployments.length} deployment{state.deployments.length !== 1 ? 's' : ''}
		</p>
		<Button
			variant="outline"
			size="sm"
			onclick={() => logic.loadDeployments()}
			disabled={state.loading || state.creating || state.deleting || state.deploying}
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
	open={state.showDeleteModal}
	item={state.deploymentToDelete}
	itemType="deployment"
	loading={state.deleting}
	onclose={() => logic.closeDeleteModal()}
	onconfirm={(id) => logic.confirmDeleteDeployment(id)}
/>

<!-- Deployment Create Modal -->
<DeploymentCreateModal
	open={state.showCreateModal}
	apps={state.apps}
	versions={state.versions}
	creating={state.creating}
	{logic}
	onclose={() => logic.closeCreateModal()}
	oncreate={(data) => logic.createDeployment(data)}
/>

<!-- Deployment Modal -->
<DeploymentModal
	open={state.showDeployModal}
	deployment={state.deploymentToDeploy}
	app={state.deploymentToDeploy ? logic.getDeploymentApp(state.deploymentToDeploy) || null : null}
	version={state.deploymentToDeploy
		? logic.getDeploymentVersion(state.deploymentToDeploy) || null
		: null}
	deployments={state.deployments}
	deploying={state.deploying}
	onclose={() => logic.closeDeployModal()}
	ondeploy={(deploymentId, isInitialDeploy, superuserEmail, superuserPass) =>
		logic.deployFromModal(deploymentId, isInitialDeploy, superuserEmail, superuserPass)}
/>

<!-- Logs Modal -->
<LogsModal
	open={state.showLogsModal}
	deployment={state.deploymentToShowLogs}
	onclose={() => logic.closeLogsModal()}
/>
