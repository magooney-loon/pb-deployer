<script lang="ts">
	import type { App, Version } from '$lib/api/index.js';
	import type { DeploymentsListLogic } from '$lib/components/main/DeploymentsList.js';
	import { Button } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import Modal from '$lib/components/main/Modal.svelte';

	interface Props {
		open: boolean;
		apps: App[];
		versions: Version[];
		creating: boolean;
		logic: DeploymentsListLogic;
		onclose: () => void;
		oncreate: (data: { app_id: string; version_id: string }) => Promise<void>;
	}

	let { open, apps, versions, creating, logic, onclose, oncreate }: Props = $props();

	let selectedAppId = $state('');
	let selectedVersionId = $state('');

	// Filter versions based on selected app (excluding those with pending deployments)
	let availableVersions = $derived(
		selectedAppId ? logic.getAvailableVersionsForApp(selectedAppId) : []
	);

	// Get all versions for the app (including pending ones) to show warning
	let allVersionsForApp = $derived(
		selectedAppId ? versions.filter((v) => v.app_id === selectedAppId) : []
	);

	// Check if any versions are filtered out due to pending deployments
	let hasFilteredVersions = $derived(
		selectedAppId && allVersionsForApp.length > availableVersions.length
	);

	// Get apps that have versions available
	let availableApps = $derived(
		apps.filter((app) => versions.some((version) => version.app_id === app.id))
	);

	// Reset form when modal closes
	$effect(() => {
		if (!open) {
			selectedAppId = '';
			selectedVersionId = '';
		}
	});

	// Reset version selection when app changes
	$effect(() => {
		if (selectedAppId) {
			selectedVersionId = '';
		}
	});

	async function handleSubmit() {
		if (creating) return;

		try {
			await oncreate({
				app_id: selectedAppId,
				version_id: selectedVersionId
			});
		} catch (error) {
			console.error('Failed to create deployment:', error);
		}
	}

	function getSelectedApp(): App | undefined {
		return apps.find((app) => app.id === selectedAppId);
	}

	function getSelectedVersion(): Version | undefined {
		return versions.find((version) => version.id === selectedVersionId);
	}
</script>

<Modal {open} title="Create New Deployment" size="lg" {onclose}>
	<form onsubmit={handleSubmit} class="space-y-6">
		<!-- App Selection -->
		<div>
			<label
				for="app-select"
				class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
			>
				Application <span class="text-red-500">*</span>
			</label>
			<select
				id="app-select"
				bind:value={selectedAppId}
				disabled={creating}
				class="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none disabled:bg-gray-100 disabled:text-gray-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 dark:focus:border-blue-400 dark:disabled:bg-gray-700"
			>
				<option value="">Select an application...</option>
				{#each availableApps as app (app.id)}
					<option value={app.id}>{app.name} ({app.domain})</option>
				{/each}
			</select>
		</div>

		<!-- Version Selection -->
		<div>
			<label
				for="version-select"
				class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
			>
				Version <span class="text-red-500">*</span>
			</label>
			<select
				id="version-select"
				bind:value={selectedVersionId}
				disabled={creating || !selectedAppId}
				class="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none disabled:bg-gray-100 disabled:text-gray-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 dark:focus:border-blue-400 dark:disabled:bg-gray-700"
			>
				<option value="">
					{selectedAppId ? 'Select a version...' : 'Select an app first'}
				</option>
				{#each availableVersions as version (version.id)}
					<option value={version.id}>
						v{version.version_number}
						{#if version.notes}
							- {version.notes}
						{/if}
					</option>
				{/each}
			</select>
		</div>

		<!-- Selected Info Preview -->
		{#if selectedAppId && selectedVersionId}
			<div
				class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800"
			>
				<h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-gray-100">
					Deployment Summary
				</h4>
				<div class="space-y-1 text-sm text-gray-600 dark:text-gray-400">
					<p><span class="font-medium">App:</span> {getSelectedApp()?.name}</p>
					<p><span class="font-medium">Domain:</span> {getSelectedApp()?.domain}</p>
					<p><span class="font-medium">Version:</span> v{getSelectedVersion()?.version_number}</p>
					{#if getSelectedVersion()?.notes}
						<p><span class="font-medium">Notes:</span> {getSelectedVersion()?.notes}</p>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Warning for no apps/versions -->
		{#if availableApps.length === 0}
			<div
				class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950"
			>
				<div class="flex items-center">
					<Icon name="warning" class="mr-2 text-amber-600 dark:text-amber-400" />
					<p class="text-sm text-amber-700 dark:text-amber-300">
						No applications with uploaded versions found. Upload a version to an app first.
					</p>
				</div>
			</div>
		{/if}
		{#if selectedAppId && availableVersions.length === 0}
			<div
				class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-950"
			>
				<div class="flex items-center">
					<Icon name="warning" class="mr-2 text-amber-600 dark:text-amber-400" />
					<p class="text-sm text-amber-700 dark:text-amber-300">
						{#if hasFilteredVersions}
							All versions for this application have pending or running deployments. Please wait for
							them to complete.
						{:else}
							No versions available for this application. Upload a version first.
						{/if}
					</p>
				</div>
			</div>
		{:else if hasFilteredVersions}
			<div
				class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-950"
			>
				<div class="flex items-center">
					<Icon name="info" class="mr-2 text-blue-600 dark:text-blue-400" />
					<p class="text-sm text-blue-700 dark:text-blue-300">
						Some versions are hidden because they have pending or running deployments.
					</p>
				</div>
			</div>
		{/if}
	</form>

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button variant="outline" onclick={onclose} disabled={creating}>Cancel</Button>
			<Button
				variant="primary"
				loading={creating}
				disabled={creating ||
					!selectedAppId ||
					!selectedVersionId ||
					availableVersions.length === 0}
				onclick={handleSubmit}
			>
				{#snippet iconSnippet()}
					<Icon name="rocket" />
				{/snippet}
				{creating ? 'Creating...' : 'Create Deployment'}
			</Button>
		</div>
	{/snippet}
</Modal>
