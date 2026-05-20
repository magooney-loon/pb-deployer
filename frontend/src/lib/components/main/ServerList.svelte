<script lang="ts">
	import { onMount } from 'svelte';
	import { formatTimestamp } from '$lib/api/index.js';
	import { serversStore, appsStore } from '$lib/stores';
	import DeleteModal from '$lib/components/modals/DeleteModal.svelte';
	import ServerCreateModal from '$lib/components/modals/ServerCreateModal.svelte';
	import TroubleshootModal from '$lib/components/modals/TroubleshootModal.svelte';
	import TerminalModal from '$lib/components/modals/TerminalModal.svelte';
	import { Button, Toast, EmptyState, LoadingSpinner, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { getServerStatusBadge } from '$lib/components/partials/index.js';
	import type { Server } from '$lib/api/index.js';
	import type { ValidationResponse } from '$lib/api/index.js';

	interface ServerFormData {
		name: string;
		host: string;
		port: number;
		root_username: string;
		app_username: string;
		proxy_email: string;
	}

	onMount(async () => {
		await Promise.all([
			serversStore.initialized ? null : serversStore.load(),
			appsStore.initialized ? null : appsStore.load()
		]);
	});

	// Modal local state
	let showCreateForm = $state(false);
	let creating = $state(false);
	let successMessage = $state<string | null>(null);

	let showDeleteModal = $state(false);
	let serverToDelete = $state<Server | null>(null);
	let deleting = $state(false);

	let showTroubleshootModal = $state(false);
	let troubleshootServerId = $state<string | null>(null);
	let troubleshootResults = $state<ValidationResponse | null>(null);

	let showTerminalModal = $state(false);
	let terminalServerId = $state<string | null>(null);

	let error = $state<string | null>(null);

	// Per-server helpers
	function isSetupInProgress(id: string) {
		return !!serversStore.setupInProgress[id];
	}
	function isSecurityInProgress(id: string) {
		return !!serversStore.securityInProgress[id];
	}
	function isTroubleshootInProgress(id: string) {
		return !!serversStore.troubleshootInProgress[id];
	}
	function canSetup(server: Server) {
		return !server.setup_complete && !isSetupInProgress(server.id);
	}
	function canSecure(server: Server) {
		return server.setup_complete && !server.security_locked && !isSecurityInProgress(server.id);
	}

	async function handleCreateServer(serverData: ServerFormData) {
		creating = true;
		error = null;
		successMessage = null;
		try {
			const server = await serversStore.create({
				name: serverData.name,
				host: serverData.host,
				port: serverData.port,
				root_username: serverData.root_username,
				app_username: serverData.app_username,
				use_ssh_agent: true,
				manual_key_path: '',
				proxy_email: serverData.proxy_email || undefined
			});
			showCreateForm = false;
			successMessage = `Server "${server.name}" created successfully!`;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to create server';
		} finally {
			creating = false;
		}
	}

	function openDeleteModal(server: Server) {
		serverToDelete = server;
		showDeleteModal = true;
	}

	function closeDeleteModal() {
		showDeleteModal = false;
		setTimeout(() => {
			serverToDelete = null;
		}, 200);
	}

	async function confirmDelete(id: string) {
		deleting = true;
		const name = serverToDelete?.name || 'Server';
		try {
			await serversStore.remove(id);
			showDeleteModal = false;
			serverToDelete = null;
			successMessage = `${name} deleted successfully!`;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to delete server';
		} finally {
			deleting = false;
		}
	}

	async function handleSetup(serverId: string) {
		error = null;
		successMessage = null;
		try {
			await serversStore.setup(serverId);
			successMessage = 'Server setup completed successfully!';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Server setup failed';
		}
	}

	async function handleSecure(serverId: string) {
		error = null;
		successMessage = null;
		try {
			await serversStore.secure(serverId);
			successMessage = 'Server security hardening completed!';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Security hardening failed';
		}
	}

	async function handleTroubleshoot(serverId: string) {
		error = null;
		troubleshootServerId = serverId;
		showTroubleshootModal = true;
		try {
			const results = await serversStore.troubleshoot(serverId);
			troubleshootResults = results;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Troubleshoot failed';
		}
	}

	function closeTroubleshootModal() {
		showTroubleshootModal = false;
		setTimeout(() => {
			troubleshootResults = null;
			troubleshootServerId = null;
		}, 200);
	}

	function openTerminal(serverId: string) {
		terminalServerId = serverId;
		showTerminalModal = true;
	}

	function closeTerminal() {
		showTerminalModal = false;
		terminalServerId = null;
	}

	// Related apps for delete modal
	let relatedApps = $derived(
		serverToDelete ? appsStore.byServer(serverToDelete.id).map((a) => ({ id: a.id, name: a.name })) : []
	);
</script>

<header class="mb-8 flex items-center justify-between">
	<div>
		<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Servers</h1>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
			Manage your VPS servers and deployment infrastructure
		</p>
	</div>
	<Button
		variant="outline"
		onclick={() => (showCreateForm = true)}
		disabled={creating || deleting}
	>
		{#snippet iconSnippet()}
			<Icon name="plus" />
		{/snippet}
		Add Server
	</Button>
</header>

{#if error}
	<Toast message={error} type="error" onDismiss={() => (error = null)} />
{/if}

{#if serversStore.error}
	<Toast message={serversStore.error} type="error" onDismiss={() => serversStore.clearError()} />
{/if}

{#if successMessage}
	<Toast message={successMessage} type="success" onDismiss={() => (successMessage = null)} />
{/if}

{#if serversStore.loading}
	<LoadingSpinner text="Loading servers..." />
{:else if serversStore.servers.length === 0}
	<EmptyState
		title="No servers configured yet"
		description="Add your first server to start deploying applications"
	>
		{#snippet iconSnippet()}
			<Icon name="servers" size="h-12 w-12" class="text-gray-400" />
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
							>Server</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Status</th
						>
						<th
							class="px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400"
							>Connection</th
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
					{#each serversStore.servers as server (server.id)}
						{@const statusBadge = getServerStatusBadge(server)}
						<tr class="hover:bg-gray-50 dark:hover:bg-gray-900">
							<td class="px-6 py-4 whitespace-nowrap">
								<div>
									<div class="text-sm font-medium text-gray-900 dark:text-gray-100">
										{server.name}
									</div>
									<div class="text-sm text-gray-500 dark:text-gray-400">
										{server.host}:{server.port}
									</div>
								</div>
							</td>
							<td class="px-6 py-4 whitespace-nowrap">
								<StatusBadge status={statusBadge.text} variant={statusBadge.variant} dot />
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								<div>Root: {server.root_username}</div>
								<div>App: {server.app_username}</div>
								<div class="text-xs text-blue-600 dark:text-blue-400">SSH Agent</div>
								{#if server.proxy_email}
									<div class="text-xs text-gray-400 dark:text-gray-500">
										ACME: {server.proxy_email}
									</div>
								{/if}
							</td>
							<td class="px-6 py-4 text-sm whitespace-nowrap text-gray-500 dark:text-gray-400">
								{formatTimestamp(server.created)}
							</td>
							<td class="space-x-2 px-6 py-4 text-right text-sm font-medium whitespace-nowrap">
								{#if canSetup(server) || isSetupInProgress(server.id)}
									<Button
										variant="outline"
										color="green"
										size="sm"
										disabled={isSetupInProgress(server.id)}
										onclick={() => handleSetup(server.id)}
									>
										{#snippet iconSnippet()}
											<Icon name={isSetupInProgress(server.id) ? 'loading' : 'setup'} />
										{/snippet}
										{isSetupInProgress(server.id) ? 'Working' : 'Setup'}
									</Button>
								{/if}

								{#if canSecure(server) || isSecurityInProgress(server.id)}
									<Button
										variant="outline"
										color="yellow"
										size="sm"
										disabled={isSecurityInProgress(server.id)}
										onclick={() => handleSecure(server.id)}
									>
										{#snippet iconSnippet()}
											<Icon name={isSecurityInProgress(server.id) ? 'loading' : 'shield'} />
										{/snippet}
										{isSecurityInProgress(server.id) ? 'Working' : 'Secure'}
									</Button>
								{/if}

								{#if !canSetup(server)}
									<Button
										variant="ghost"
										color="blue"
										size="sm"
										disabled={isSetupInProgress(server.id) ||
											isSecurityInProgress(server.id) ||
											isTroubleshootInProgress(server.id)}
										onclick={() => handleTroubleshoot(server.id)}
									>
										{#snippet iconSnippet()}
											<Icon
												name={isTroubleshootInProgress(server.id) ? 'loading' : 'diagnostic'}
											/>
										{/snippet}
										{isTroubleshootInProgress(server.id) ? 'Checking' : 'Troubleshoot'}
									</Button>

									<Button
										variant="ghost"
										color="blue"
										size="sm"
										disabled={isSetupInProgress(server.id) || isSecurityInProgress(server.id)}
										onclick={() => openTerminal(server.id)}
									>
										{#snippet iconSnippet()}
											<Icon name="terminal" />
										{/snippet}
										Terminal
									</Button>
								{/if}

								<Button
									variant="ghost"
									color="red"
									size="sm"
									disabled={deleting ||
										creating ||
										isSetupInProgress(server.id) ||
										isSecurityInProgress(server.id)}
									onclick={() => openDeleteModal(server)}
								>
									{#snippet iconSnippet()}
										<Icon name="delete" />
									{/snippet}
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
			Showing {serversStore.servers.length} server{serversStore.servers.length !== 1 ? 's' : ''}
		</p>
		<Button
			variant="outline"
			size="sm"
			onclick={() => serversStore.load()}
			disabled={creating || deleting}
		>
			{#snippet iconSnippet()}
				<Icon name="refresh" />
			{/snippet}
			Refresh
		</Button>
	</div>
{/if}

<!-- Server Create Modal -->
<ServerCreateModal
	open={showCreateForm}
	{creating}
	onclose={() => (showCreateForm = false)}
	oncreate={handleCreateServer}
/>

<!-- Delete Server Modal -->
<DeleteModal
	open={showDeleteModal}
	item={serverToDelete}
	itemType="server"
	loading={deleting}
	relatedItems={relatedApps}
	relatedItemsType="apps"
	onclose={closeDeleteModal}
	onconfirm={(id) => confirmDelete(id)}
/>

<!-- Troubleshoot Modal -->
<TroubleshootModal
	open={showTroubleshootModal}
	server={troubleshootServerId
		? serversStore.servers.find((s) => s.id === troubleshootServerId) || null
		: null}
	results={troubleshootResults}
	setupInProgress={troubleshootServerId ? isSetupInProgress(troubleshootServerId) : false}
	onclose={closeTroubleshootModal}
	onsetup={(serverId) => handleSetup(serverId)}
/>

<!-- Terminal Modal -->
<TerminalModal
	open={showTerminalModal}
	server={terminalServerId
		? serversStore.servers.find((s) => s.id === terminalServerId) || null
		: null}
	onclose={closeTerminal}
/>
