<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, StatusBadge, LoadingSpinner, EmptyState } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import type { Server } from '$lib/api/index.js';

	interface ValidationResponse {
		valid: boolean;
		message: string;
		setup_info?: {
			os: string;
			architecture: string;
			hostname: string;
			pocketbase_setup: boolean;
			installed_apps: string[];
		};
		error?: string;
	}

	interface Props {
		open: boolean;
		server: Server | null;
		results: ValidationResponse | null;
		setupInProgress?: boolean;
		onclose: () => void;
		onsetup?: (serverId: string) => Promise<void>;
	}

	let { open, server, results, setupInProgress = false, onclose, onsetup }: Props = $props();

	function formatTimestamp(): string {
		return new Date().toLocaleString();
	}

	function getValidationStatusBadge(isValid: boolean): {
		text: string;
		variant: 'success' | 'error' | 'warning' | 'info' | 'gray' | 'update' | 'custom';
	} {
		return isValid
			? { text: 'Valid', variant: 'success' }
			: { text: 'Issues Found', variant: 'error' };
	}

	async function copyResults() {
		if (results) {
			try {
				const resultText = JSON.stringify(results, null, 2);
				await navigator.clipboard.writeText(resultText);
			} catch (err) {
				console.error('Failed to copy results:', err);
			}
		}
	}

	async function handleSetup() {
		if (server?.id && onsetup) {
			try {
				await onsetup(server.id);
				onclose();
			} catch (error) {
				console.error('Setup failed:', error);
			}
		}
	}

	let modalTitle = $derived(
		server ? `Troubleshoot Results - ${server.name}` : 'Troubleshoot Results'
	);
</script>

<Modal {open} title={modalTitle} size="xl" {onclose}>
	{#if server}
		{#if results}
			<!-- Server Info Header -->
			<div
				class="mb-6 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800"
			>
				<div class="flex items-center justify-between">
					<div class="flex items-center space-x-3">
						<div class="text-blue-600 dark:text-blue-400">
							<Icon name="servers" />
						</div>
						<div>
							<h3 class="font-semibold text-gray-900 dark:text-gray-100">
								{server.name}
							</h3>
							<p class="text-sm text-gray-600 dark:text-gray-400">
								{server.host}:{server.port} • Checked at {formatTimestamp()}
							</p>
						</div>
					</div>
					<StatusBadge
						status={getValidationStatusBadge(results.valid).text}
						variant={getValidationStatusBadge(results.valid).variant}
					/>
				</div>
			</div>

			<!-- Validation Results -->
			<div class="space-y-6">
				<!-- Overall Status -->
				<div
					class="rounded-lg border p-4 {results.valid
						? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950'
						: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950'}"
				>
					<div class="flex items-start space-x-3">
						<div class="flex-shrink-0">
							{#if results.valid}
								<Icon name="check" class="text-green-600 dark:text-green-400" />
							{:else}
								<Icon name="warning" class="text-red-600 dark:text-red-400" />
							{/if}
						</div>
						<div class="flex-1">
							<h3
								class="font-semibold {results.valid
									? 'text-green-900 dark:text-green-100'
									: 'text-red-900 dark:text-red-100'}"
							>
								{results.valid ? 'Server Validation Passed' : 'Server Validation Failed'}
							</h3>
							<p
								class="mt-1 text-sm {results.valid
									? 'text-green-800 dark:text-green-200'
									: 'text-red-800 dark:text-red-200'}"
							>
								{results.message}
							</p>
							{#if results.error}
								<div
									class="mt-3 rounded-md border border-red-300 bg-red-100 p-3 dark:border-red-700 dark:bg-red-900/50"
								>
									<h4 class="font-medium text-red-900 dark:text-red-100">Error Details:</h4>
									<p class="mt-1 text-sm text-red-800 dark:text-red-200">{results.error}</p>
								</div>
							{/if}
						</div>
					</div>
				</div>

				<!-- System Information -->
				{#if results.setup_info}
					<div
						class="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-950"
					>
						<div class="mb-4 flex items-center justify-between">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
								System Information
							</h3>
							<Button variant="ghost" size="xs" onclick={copyResults}>
								{#snippet iconSnippet()}
									<Icon name="copy" size="h-3 w-3" />
								{/snippet}
								Copy All
							</Button>
						</div>

						<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
							<div class="space-y-3">
								<div>
									<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
										Operating System
									</div>
									<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
										{results.setup_info.os || 'Unknown'}
									</div>
								</div>

								<div>
									<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
										Architecture
									</div>
									<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
										{results.setup_info.architecture || 'Unknown'}
									</div>
								</div>

								<div>
									<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
										Hostname
									</div>
									<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
										{results.setup_info.hostname || 'Unknown'}
									</div>
								</div>
							</div>

							<div class="space-y-3">
								<div>
									<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
										PocketBase Setup
									</div>
									<div class="mt-1">
										<StatusBadge
											status={results.setup_info.pocketbase_setup ? 'Complete' : 'Not Setup'}
											variant={results.setup_info.pocketbase_setup ? 'success' : 'warning'}
											size="xs"
										/>
									</div>
								</div>

								<div>
									<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
										Installed Applications
									</div>
									<div class="mt-1">
										{#if results.setup_info.installed_apps && results.setup_info.installed_apps.length > 0}
											<div class="space-y-1">
												{#each results.setup_info.installed_apps as app (app)}
													<span
														class="inline-block rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200"
													>
														{app}
													</span>
												{/each}
											</div>
										{:else}
											<span class="text-sm text-gray-500 dark:text-gray-400">No apps installed</span
											>
										{/if}
									</div>
								</div>
							</div>
						</div>
					</div>
				{/if}

				<!-- Connection Details -->
				<div
					class="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-950"
				>
					<h3 class="mb-4 text-lg font-semibold text-gray-900 dark:text-gray-100">
						Connection Details
					</h3>
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<div>
							<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
								SSH Connection
							</div>
							<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
								{server.root_username}@{server.host}:{server.port}
							</div>
						</div>
						<div>
							<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
								App User
							</div>
							<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
								{server.app_username}
							</div>
						</div>
					</div>
				</div>

				<!-- Recommendations -->
				{#if !results.valid}
					<div
						class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950"
					>
						<div class="flex items-start space-x-3">
							<div class="flex-shrink-0">
								<Icon name="lightbulb" class="text-amber-600 dark:text-amber-400" />
							</div>
							<div class="flex-1">
								<h3 class="font-semibold text-amber-900 dark:text-amber-100">
									Troubleshooting Recommendations
								</h3>
								<div class="mt-2 space-y-2 text-sm text-amber-800 dark:text-amber-200">
									<div class="flex items-start space-x-2">
										<span class="mt-1.5 h-1 w-1 rounded-full bg-amber-600 dark:bg-amber-400"></span>
										<span
											>Verify SSH connectivity: <code
												class="rounded bg-amber-100 px-1 py-0.5 text-xs dark:bg-amber-900"
												>ssh {server.root_username}@{server.host}</code
											></span
										>
									</div>
									<div class="flex items-start space-x-2">
										<span class="mt-1.5 h-1 w-1 rounded-full bg-amber-600 dark:bg-amber-400"></span>
										<span
											>Check SSH agent: <code
												class="rounded bg-amber-100 px-1 py-0.5 text-xs dark:bg-amber-900"
												>ssh-add -l</code
											></span
										>
									</div>
									<div class="flex items-start space-x-2">
										<span class="mt-1.5 h-1 w-1 rounded-full bg-amber-600 dark:bg-amber-400"></span>
										<span>Ensure server is accessible and credentials are correct</span>
									</div>
									{#if results.setup_info && !results.setup_info.pocketbase_setup}
										<div class="flex items-start space-x-2">
											<span class="mt-1.5 h-1 w-1 rounded-full bg-amber-600 dark:bg-amber-400"
											></span>
											<span>Run server setup to configure PocketBase environment</span>
										</div>
									{/if}
								</div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{:else}
			<!-- Loading State -->
			<div class="space-y-6">
				<!-- Server Info Header -->
				<div
					class="mb-6 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800"
				>
					<div class="flex items-center space-x-3">
						<div class="text-blue-600 dark:text-blue-400">
							<Icon name="servers" />
						</div>
						<div>
							<h3 class="font-semibold text-gray-900 dark:text-gray-100">
								{server.name}
							</h3>
							<p class="text-sm text-gray-600 dark:text-gray-400">
								{server.host}:{server.port} • Running diagnostics...
							</p>
						</div>
					</div>
				</div>

				<!-- Loading Content -->
				<LoadingSpinner text="Running server diagnostics..." size="lg" />

				<div
					class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-700 dark:bg-blue-900/20"
				>
					<div class="flex items-start space-x-3">
						<div class="flex-shrink-0">
							<Icon name="info" class="text-blue-600 dark:text-blue-400" />
						</div>
						<div class="flex-1">
							<h3 class="font-semibold text-blue-900 dark:text-blue-100">Checking Server Status</h3>
							<div class="mt-2 space-y-1 text-sm text-blue-800 dark:text-blue-200">
								<p>• Verifying SSH connectivity</p>
								<p>• Checking system information</p>
								<p>• Validating PocketBase setup</p>
								<p>• Scanning installed applications</p>
							</div>
						</div>
					</div>
				</div>
			</div>
		{/if}
	{:else}
		<EmptyState
			title="No server selected"
			description="Select a server to run troubleshoot diagnostics"
		>
			{#snippet iconSnippet()}
				<Icon name="servers" size="h-12 w-12" class="text-gray-400" />
			{/snippet}
		</EmptyState>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button variant="outline" onclick={onclose} disabled={setupInProgress}>Close</Button>
			{#if results && !results.valid && server && onsetup}
				<Button
					variant="primary"
					onclick={handleSetup}
					disabled={setupInProgress}
					loading={setupInProgress}
				>
					{#snippet iconSnippet()}
						<Icon name="setup" />
					{/snippet}
					{setupInProgress ? 'Setting up...' : 'Run Setup'}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>
