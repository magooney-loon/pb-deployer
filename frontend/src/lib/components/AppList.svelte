<script lang="ts">
	import { onMount } from 'svelte';
	import { AppListLogic, type AppListState } from './logic/AppList.js';
	import {
		Button,
		ErrorAlert,
		FormField,
		EmptyState,
		LoadingSpinner,
		Card
	} from '$lib/components/partials';

	// Create logic instance
	const logic = new AppListLogic();
	let state = $state<AppListState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Make available servers reactive to state changes
	let availableServers = $derived(
		state.servers.filter((s) => s.setup_complete && s.security_locked)
	);

	onMount(async () => {
		await logic.initialize();
	});
</script>

<div class="p-6">
	<div class="mb-6 flex items-center justify-between">
		<h1 class="text-3xl font-bold text-gray-900 dark:text-white">Applications</h1>
		<Button onclick={() => logic.toggleCreateForm()} disabled={availableServers.length === 0}>
			{state.showCreateForm ? 'Cancel' : 'Add App'}
		</Button>
	</div>

	{#if availableServers.length === 0 && !state.showCreateForm}
		<ErrorAlert
			type="warning"
			title="No Ready Servers"
			message="You need at least one server with setup and security lockdown completed before you can create apps."
			dismissible={false}
		/>
	{/if}

	{#if state.error}
		<ErrorAlert message={state.error} type="error" onDismiss={() => logic.dismissError()} />
	{/if}

	{#if state.showCreateForm}
		<Card title="Add New Application" class="mb-6">
			<form
				onsubmit={(e) => {
					e.preventDefault();
					logic.createApp();
				}}
				class="space-y-4"
			>
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<FormField
						id="app-name"
						label="App Name"
						bind:value={state.newApp.name}
						placeholder="my-app"
						helperText="Used for directory and service naming"
						required
					/>

					<FormField
						id="server-select"
						label="Server"
						type="select"
						bind:value={state.newApp.server_id}
						placeholder="Select a server"
						options={availableServers.map((server) => ({
							value: server.id,
							label: `${server.name} (${server.host})`
						}))}
						required
					/>

					<FormField
						id="domain"
						label="Domain"
						bind:value={state.newApp.domain}
						placeholder="myapp.example.com"
						helperText="The domain where your app will be accessible"
						class="md:col-span-2"
						required
					/>

					<FormField
						id="remote-path"
						label="Remote Path (Optional)"
						bind:value={state.newApp.remote_path}
						placeholder="/opt/pocketbase/apps/{state.newApp.name || 'app-name'}"
					/>

					<FormField
						id="service-name"
						label="Service Name (Optional)"
						bind:value={state.newApp.service_name}
						placeholder="pocketbase-{state.newApp.name || 'app-name'}"
					/>
				</div>
				<div class="flex space-x-3">
					<Button type="submit">Create App</Button>
					<Button
						variant="secondary"
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
		<LoadingSpinner text="Loading applications..." />
	{:else if state.apps.length === 0}
		<EmptyState
			icon="📱"
			title="No applications created yet"
			description={availableServers.length > 0
				? 'Create your first application to start deploying'
				: 'Set up a server first before creating apps'}
		/>
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
						{#each state.apps as app (app.id)}
							{@const statusBadge = logic.getAppStatusBadge(app)}
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
										{logic.getServerName(app.server_id)}
									</div>
									<div class="text-xs text-gray-500 dark:text-gray-400">{app.remote_path}</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<span
										class="inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium {statusBadge.color}"
									>
										{logic.getStatusIcon(app.status)}
										{statusBadge.text}
									</span>
								</td>
								<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
									{app.current_version || 'Not deployed'}
								</td>
								<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
									{logic.formatTimestamp(app.created)}
								</td>
								<td class="space-x-2 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
									<Button
										variant="link"
										color="green"
										size="sm"
										onclick={() => logic.checkHealth(app.id)}
										disabled={state.checkingHealth.has(app.id)}
										icon={state.checkingHealth.has(app.id) ? '🔄' : '💚'}
									>
										Health
									</Button>

									<Button
										variant="link"
										size="sm"
										onclick={() => logic.openApp(app.domain)}
										icon="🔗"
									>
										Open
									</Button>

									<Button
										variant="link"
										color="red"
										size="sm"
										onclick={() => logic.deleteApp(app.id)}
										icon="🗑️"
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

		<div class="mt-4 flex items-center justify-between">
			<p class="text-sm text-gray-700 dark:text-gray-300">
				Showing {state.apps.length} application{state.apps.length !== 1 ? 's' : ''}
			</p>
			<Button variant="secondary" size="sm" icon="🔄" onclick={() => logic.loadApps()}>
				Refresh
			</Button>
		</div>
	{/if}
</div>
