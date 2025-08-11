<script lang="ts">
	import { onMount } from 'svelte';
	import { AppListLogic, type AppListState } from './AppList.js';
	import {
		Button,
		ErrorAlert,
		FormField,
		EmptyState,
		LoadingSpinner,
		Card,
		FileUpload,
		ProgressBar
	} from '$lib/components/partials';

	// Create logic instance
	const logic = new AppListLogic();
	let state = $state<AppListState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Make available servers reactive to state changes
	let availableServers = $derived(state.servers.filter((s) => s.setup_complete));

	onMount(async () => {
		await logic.initialize();
	});
</script>

<div class="p-6">
	<div class="mb-8 flex items-center justify-between">
		<div>
			<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Applications</h1>
			<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
				Manage your deployed PocketBase applications
			</p>
		</div>
		<Button
			variant="outline"
			icon={state.showCreateForm ? 'x' : '+'}
			onclick={() => logic.toggleCreateForm()}
			disabled={availableServers.length === 0}
		>
			{state.showCreateForm ? 'Cancel' : 'Add App'}
		</Button>
	</div>

	{#if availableServers.length === 0 && !state.showCreateForm}
		<ErrorAlert
			type="warning"
			title="No Ready Servers"
			message="You need at least one server with setup completed before you can create apps."
			dismissible={false}
		/>
	{/if}

	{#if state.error}
		<ErrorAlert message={state.error} type="error" onDismiss={() => logic.dismissError()} />
	{/if}

	{#if state.showCreateForm}
		<Card title="Add New Application" class="mb-6">
			{#if state.creating}
				<div class="space-y-4">
					<div class="text-center">
						<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">
							Creating Application
						</h3>
						<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{state.currentStep}</p>
					</div>
					<ProgressBar value={state.uploadProgress} label="Progress" color="blue" animated={true} />
				</div>
			{:else}
				<form
					onsubmit={(e) => {
						e.preventDefault();
						logic.createApp();
					}}
					class="space-y-6"
				>
					<!-- Basic App Information -->
					<div class="space-y-4">
						<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">App Configuration</h3>
						<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
							<FormField
								id="app-name"
								label="App Name"
								value={state.newApp.name}
								placeholder="my-app"
								helperText="Used for directory and service naming"
								required
								oninput={(e) => logic.updateNewApp('name', (e.target as HTMLInputElement).value)}
							/>

							<FormField
								id="server-select"
								label="Server"
								type="select"
								value={state.newApp.server_id}
								placeholder="Select a server"
								options={availableServers.map((server) => ({
									value: server.id,
									label: `${server.name} (${server.host})`
								}))}
								required
								onchange={(e) =>
									logic.updateNewApp('server_id', (e.target as HTMLSelectElement).value)}
							/>

							<FormField
								id="domain"
								label="Domain"
								value={state.newApp.domain}
								placeholder="myapp.example.com"
								helperText="The domain where your app will be accessible"
								class="md:col-span-2"
								required
								oninput={(e) => logic.updateNewApp('domain', (e.target as HTMLInputElement).value)}
							/>

							<FormField
								id="remote-path"
								label="Remote Path (Optional)"
								value={state.newApp.remote_path}
								placeholder="/opt/pocketbase/apps/{state.newApp.name || 'app-name'}"
								oninput={(e) =>
									logic.updateNewApp('remote_path', (e.target as HTMLInputElement).value)}
							/>

							<FormField
								id="service-name"
								label="Service Name (Optional)"
								value={state.newApp.service_name}
								placeholder="pocketbase-{state.newApp.name || 'app-name'}"
								oninput={(e) =>
									logic.updateNewApp('service_name', (e.target as HTMLInputElement).value)}
							/>
						</div>
					</div>

					<!-- Version Information -->
					<div class="space-y-4">
						<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Initial Version</h3>
						<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
							<FormField
								id="version-number"
								label="Version Number"
								value={state.newApp.version_number}
								placeholder="1.0.0"
								helperText="Semantic versioning recommended"
								required
								oninput={(e) =>
									logic.updateNewApp('version_number', (e.target as HTMLInputElement).value)}
							/>

							<FormField
								id="version-notes"
								label="Version Notes"
								value={state.newApp.version_notes}
								placeholder="Initial release"
								helperText="Describe this version"
								oninput={(e) =>
									logic.updateNewApp('version_notes', (e.target as HTMLInputElement).value)}
							/>
						</div>
					</div>

					<!-- File Uploads -->
					<div class="space-y-4">
						<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Required Files</h3>
						<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
							<FileUpload
								id="pocketbase-binary"
								label="PocketBase Binary"
								maxSize={100 * 1024 * 1024}
								helperText="Upload the PocketBase executable file (any file type accepted)"
								value={state.newApp.pocketbase_binary}
								onFileSelect={(file) => logic.updateBinaryFile(file)}
								onError={(error) => logic.setError(error)}
								required
							/>

							<FileUpload
								id="pb-public-folder"
								label="pb_public Folder"
								directory={true}
								maxSize={50 * 1024 * 1024}
								helperText="Select your pb_public folder containing your app's frontend files"
								value={state.newApp.pb_public_folder}
								onFileSelect={(files) => logic.updatePublicFolder(files)}
								onError={(error) => logic.setError(error)}
								required
							/>
						</div>
						<div class="rounded-md bg-blue-50 p-4 dark:bg-blue-950">
							<div class="flex">
								<div class="flex-shrink-0">
									<svg class="h-5 w-5 text-blue-400" viewBox="0 0 20 20" fill="currentColor">
										<path
											fill-rule="evenodd"
											d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
											clip-rule="evenodd"
										/>
									</svg>
								</div>
								<div class="ml-3">
									<h3 class="text-sm font-medium text-blue-800 dark:text-blue-200">
										First-time setup
									</h3>
									<div class="mt-2 text-sm text-blue-700 dark:text-blue-300">
										<p>This is the first version of your app. You need to provide:</p>
										<ul class="mt-1 list-disc pl-5">
											<li>PocketBase binary - the executable file for your platform</li>
											<li>pb_public folder - your app's frontend files and subdirectories</li>
										</ul>
									</div>
								</div>
							</div>
						</div>
					</div>

					<div class="flex space-x-3">
						<Button variant="outline" type="submit" disabled={state.creating}>
							{state.creating ? 'Creating...' : 'Create App'}
						</Button>
						<Button
							variant="secondary"
							color="gray"
							onclick={() => {
								logic.toggleCreateForm();
								logic.resetForm();
							}}
							disabled={state.creating}
						>
							Cancel
						</Button>
					</div>
				</form>
			{/if}
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
								>Server</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
								>Status</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
								>Version</th
							>
							<th
								class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
								>Created</th
							>
							<th
								class="px-6 py-3 text-right text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
								>Actions</th
							>
						</tr>
					</thead>
					<tbody class="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-950">
						{#each state.apps as app (app.id)}
							{@const statusBadge = logic.getAppStatusBadge(app)}
							<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="flex items-center">
										<div>
											<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
												{app.name}
											</div>
											<div class="text-sm text-gray-500 dark:text-gray-400">
												<a
													href="https://{app.domain}"
													target="_blank"
													class="text-gray-600 underline-offset-4 hover:text-gray-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
												>
													{app.domain} 🔗
												</a>
											</div>
											<div class="text-xs text-gray-400 dark:text-gray-500">{app.service_name}</div>
										</div>
									</div>
								</td>
								<td class="px-6 py-4 whitespace-nowrap">
									<div class="text-sm text-gray-900 dark:text-gray-100">
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
								<td class="space-x-1 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
									<Button
										variant="ghost"
										color="green"
										size="sm"
										onclick={() => logic.checkHealth(app.id)}
										disabled={state.checkingHealth.has(app.id)}
										icon={state.checkingHealth.has(app.id) ? '🔄' : '💚'}
									>
										Health
									</Button>

									{#if app.status === 'offline'}
										<Button
											variant="ghost"
											color="blue"
											size="sm"
											onclick={() => logic.startApp(app.id)}
											icon="▶️"
										>
											Start
										</Button>
									{:else}
										<Button
											variant="ghost"
											color="yellow"
											size="sm"
											onclick={() => logic.stopApp(app.id)}
											icon="⏹️"
										>
											Stop
										</Button>
									{/if}

									<Button
										variant="ghost"
										color="gray"
										size="sm"
										onclick={() => logic.restartApp(app.id)}
										icon="🔄"
									>
										Restart
									</Button>

									<Button
										variant="ghost"
										size="sm"
										onclick={() => logic.openApp(app.domain)}
										icon="🔗"
									>
										Open
									</Button>

									<Button
										variant="ghost"
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

		<div class="mt-6 flex items-center justify-between">
			<p class="text-sm text-gray-600 dark:text-gray-400">
				Showing {state.apps.length} application{state.apps.length !== 1 ? 's' : ''}
			</p>
			<Button variant="outline" size="sm" icon="🔄" onclick={() => logic.loadApps()}>
				Refresh
			</Button>
		</div>
	{/if}
</div>
