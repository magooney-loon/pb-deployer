<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField, FileUpload, StatusBadge } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import DeleteModal from './DeleteModal.svelte';
	import { ApiClient } from '$lib/api/index.js';
	import type { App, Version, Deployment } from '$lib/api/index.js';

	interface Props {
		open?: boolean;
		app?: App | null;
		onclose?: () => void;
		onrefresh?: () => Promise<void>;
	}

	let { open = false, app = null, onclose, onrefresh }: Props = $props();

	const api = new ApiClient();

	let loading = $state(false);
	let uploading = $state(false);
	let deleting = $state(false);
	let versions = $state<Version[]>([]);
	let deployments = $state<Deployment[]>([]);
	let error = $state<string | undefined>(undefined);

	// Upload form state
	let showUploadForm = $state(false);
	let uploadFormData = $state({
		version_number: '',
		notes: ''
	});
	let versionType = $state<'patch' | 'minor' | 'major'>('patch');
	let currentVersion = $state('0.0.0');
	let deploymentFile = $state<File | null>(null);
	let fileError = $state<string | undefined>(undefined);

	// Delete modal state
	let showDeleteModal = $state(false);
	let versionToDelete = $state<Version | null>(null);

	// Derived values
	let nextVersion = $derived(calculateNextVersion(currentVersion, versionType));

	function parseVersion(version: string): [number, number, number] {
		const parts = version.split('.').map(Number);
		return [parts[0] || 0, parts[1] || 0, parts[2] || 0];
	}

	function calculateNextVersion(current: string, type: 'patch' | 'minor' | 'major'): string {
		const [major, minor, patch] = parseVersion(current);

		switch (type) {
			case 'patch':
				return `${major}.${minor}.${patch + 1}`;
			case 'minor':
				return `${major}.${minor + 1}.0`;
			case 'major':
				return `${major + 1}.0.0`;
			default:
				return `${major}.${minor}.${patch + 1}`;
		}
	}

	function formatDate(dateString: string): string {
		try {
			return new Date(dateString).toLocaleDateString();
		} catch {
			return dateString;
		}
	}

	async function loadVersions() {
		if (!app?.id) return;

		try {
			loading = true;
			error = undefined;

			// Load both versions and deployments
			const [versionsResponse, deploymentsResponse] = await Promise.all([
				api.versions.getAppVersions(app.id),
				api.deployments.getAppDeployments(app.id)
			]);

			versions = versionsResponse.versions || [];
			deployments = deploymentsResponse.deployments || [];

			// Update current version based on latest successful deployment
			const latestSuccessfulDeployment = deployments
				.filter((d) => d.status === 'success')
				.sort(
					(a, b) =>
						new Date(b.completed_at || b.created).getTime() -
						new Date(a.completed_at || a.created).getTime()
				)[0];

			if (latestSuccessfulDeployment) {
				const deployedVersion = versions.find(
					(v) => v.id === latestSuccessfulDeployment.version_id
				);
				if (deployedVersion) {
					currentVersion = deployedVersion.version_number;
				}
			} else if (app.current_version && app.current_version !== '0.0.0') {
				currentVersion = app.current_version;
			} else if (versions.length > 0) {
				currentVersion = versions[0].version_number;
			} else {
				currentVersion = '0.0.0';
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load versions';
			versions = [];
			deployments = [];
		} finally {
			loading = false;
		}
	}

	function resetUploadForm() {
		showUploadForm = false;
		uploadFormData = {
			version_number: '',
			notes: ''
		};
		versionType = 'patch';
		deploymentFile = null;
		fileError = undefined;
	}

	function handleFileSelect(file: File | File[] | null) {
		fileError = undefined;
		if (file && !Array.isArray(file)) {
			deploymentFile = file;
		} else {
			deploymentFile = null;
		}
	}

	function handleFileError(error: string) {
		fileError = error;
		deploymentFile = null;
	}

	async function handleUploadSubmit() {
		if (!app?.id || uploading || !deploymentFile) {
			console.warn('Upload blocked:', {
				hasApp: !!app?.id,
				uploading,
				hasFile: !!deploymentFile
			});
			return;
		}

		try {
			uploading = true;
			error = undefined;

			console.log('Starting version upload:', {
				app_id: app.id,
				version_number: nextVersion,
				file_name: deploymentFile.name,
				file_size: deploymentFile.size
			});

			// Check if version already exists
			console.log('Checking if version exists:', nextVersion);
			const versionExists = await api.versions.checkVersionExists(app.id, nextVersion);
			console.log('Version exists check result:', versionExists);

			if (versionExists) {
				throw new Error(
					`Version ${nextVersion} already exists for this application. Please use a different version type.`
				);
			}

			// Create version
			console.log('Creating version...');
			const result = await api.versions.createVersion({
				app_id: app.id,
				version_number: nextVersion,
				notes: uploadFormData.notes,
				deployment_zip: deploymentFile
			});
			console.log('Version created successfully:', result);

			resetUploadForm();
			await loadVersions();
			await onrefresh?.();
		} catch (err) {
			console.error('Upload failed:', err);
			error = err instanceof Error ? err.message : 'Failed to upload version';
		} finally {
			uploading = false;
		}
	}

	function openDeleteModal(version: Version) {
		// Check if this version has any successful deployments
		const hasSuccessfulDeployment = deployments.some(
			(d) => d.version_id === version.id && d.status === 'success'
		);

		// Check if this version has any pending/running deployments
		const hasPendingDeployment = deployments.some(
			(d) => d.version_id === version.id && ['pending', 'running'].includes(d.status)
		);

		if (hasSuccessfulDeployment) {
			error = 'Cannot delete a version that has been successfully deployed';
			console.warn('Attempted to delete deployed version:', version.version_number);
			return;
		}

		if (hasPendingDeployment) {
			error = 'Cannot delete a version with pending deployments';
			console.warn('Attempted to delete version with pending deployment:', version.version_number);
			return;
		}

		console.log('Opening delete modal for version:', version.version_number);
		versionToDelete = version;
		showDeleteModal = true;
	}

	async function handleDeleteConfirm(versionId: string) {
		try {
			deleting = true;
			error = undefined;
			console.log('Deleting version:', versionId);
			await api.versions.deleteVersion(versionId);
			console.log('Version deleted successfully');
			await loadVersions();
			await onrefresh?.();
			versionToDelete = null;
			showDeleteModal = false;
		} catch (err) {
			console.error('Delete failed:', err);
			error = err instanceof Error ? err.message : 'Failed to delete version';
		} finally {
			deleting = false;
		}
	}

	function canDeleteVersion(version: Version): boolean {
		// Check if this version has any successful deployments
		const hasSuccessfulDeployment = deployments.some(
			(d) => d.version_id === version.id && d.status === 'success'
		);

		// Check if this version has any pending/running deployments
		const hasPendingDeployment = deployments.some(
			(d) => d.version_id === version.id && ['pending', 'running'].includes(d.status)
		);

		return !hasSuccessfulDeployment && !hasPendingDeployment;
	}

	function getVersionStatus(version: Version): {
		text: string;
		variant: 'success' | 'error' | 'gray' | 'warning' | 'info' | 'update' | 'custom';
		isDeployed: boolean;
		isPending: boolean;
	} {
		const successfulDeployment = deployments.find(
			(d) => d.version_id === version.id && d.status === 'success'
		);
		const pendingDeployment = deployments.find(
			(d) => d.version_id === version.id && ['pending', 'running'].includes(d.status)
		);
		const failedDeployment = deployments.find(
			(d) => d.version_id === version.id && d.status === 'failed'
		);

		if (successfulDeployment) {
			return { text: 'Deployed', variant: 'success', isDeployed: true, isPending: false };
		} else if (pendingDeployment) {
			return {
				text: pendingDeployment.status === 'running' ? 'Deploying' : 'Pending',
				variant: 'warning',
				isDeployed: false,
				isPending: true
			};
		} else if (failedDeployment) {
			return { text: 'Deploy Failed', variant: 'error', isDeployed: false, isPending: false };
		} else {
			return { text: 'Ready', variant: 'gray', isDeployed: false, isPending: false };
		}
	}

	// Update form data when nextVersion changes
	$effect(() => {
		uploadFormData.version_number = nextVersion;
	});

	// Load versions when app changes
	$effect(() => {
		if (open && app) {
			loadVersions();
		}
	});

	// Reset state when modal closes
	$effect(() => {
		if (!open) {
			setTimeout(() => {
				resetUploadForm();
				error = undefined;
				versions = [];
				deployments = [];
			}, 300);
		}
	});

	function handleClose() {
		if (!uploading && !deleting) {
			onclose?.();
		}
	}
</script>

<Modal
	{open}
	title="Manage App Versions"
	size="xl"
	closeable={!uploading && !deleting}
	onclose={handleClose}
>
	{#if app}
		<div class="space-y-6">
			<!-- App Info -->
			<div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
				<div class="flex items-center space-x-3">
					<div class="text-blue-600 dark:text-blue-400">
						<Icon name="apps" />
					</div>
					<div>
						<h3 class="font-semibold text-blue-900 dark:text-blue-100">
							{app.name}
						</h3>
						<p class="text-sm text-blue-700 dark:text-blue-300">
							Domain: {app.domain} • Current: v{currentVersion || 'None'}
						</p>
					</div>
				</div>
			</div>

			<!-- Error Display -->
			{#if error}
				<div class="rounded-lg bg-red-50 p-4 dark:bg-red-900/20">
					<div class="flex items-center space-x-2">
						<Icon name="warning" class="text-red-600 dark:text-red-400" />
						<p class="text-sm text-red-700 dark:text-red-300">{error}</p>
					</div>
				</div>
			{/if}

			<!-- Upload New Version Section -->
			<div class="space-y-4">
				<div class="flex items-center justify-between">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Versions</h3>
					<Button
						variant="outline"
						onclick={() => (showUploadForm = !showUploadForm)}
						disabled={loading || uploading || deleting}
					>
						{#snippet iconSnippet()}
							<Icon name={showUploadForm ? 'close' : 'plus'} />
						{/snippet}
						{showUploadForm ? 'Cancel' : 'Upload New Version'}
					</Button>
				</div>

				<!-- Upload Form -->
				{#if showUploadForm}
					<div
						class="rounded-lg border border-gray-200 bg-gray-50 p-6 dark:border-gray-700 dark:bg-gray-900"
					>
						<div class="space-y-6">
							<!-- Version Type Selection -->
							<fieldset>
								<legend class="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
									Version Type
								</legend>
								<div class="grid grid-cols-3 gap-3">
									<button
										type="button"
										class="flex flex-col items-center rounded-lg border p-4 transition-colors {versionType ===
										'patch'
											? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
											: 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'}"
										disabled={uploading}
										onclick={() => (versionType = 'patch')}
									>
										<div class="font-semibold">Patch</div>
										<div class="mt-1 text-xs text-gray-500 dark:text-gray-400">Bug fixes</div>
										<div class="mt-2 font-mono text-sm">
											{calculateNextVersion(currentVersion, 'patch')}
										</div>
									</button>
									<button
										type="button"
										class="flex flex-col items-center rounded-lg border p-4 transition-colors {versionType ===
										'minor'
											? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
											: 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'}"
										disabled={uploading}
										onclick={() => (versionType = 'minor')}
									>
										<div class="font-semibold">Minor</div>
										<div class="mt-1 text-xs text-gray-500 dark:text-gray-400">New features</div>
										<div class="mt-2 font-mono text-sm">
											{calculateNextVersion(currentVersion, 'minor')}
										</div>
									</button>
									<button
										type="button"
										class="flex flex-col items-center rounded-lg border p-4 transition-colors {versionType ===
										'major'
											? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900/20 dark:text-blue-300'
											: 'border-gray-200 hover:border-gray-300 dark:border-gray-700 dark:hover:border-gray-600'}"
										disabled={uploading}
										onclick={() => (versionType = 'major')}
									>
										<div class="font-semibold">Major</div>
										<div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
											Breaking changes
										</div>
										<div class="mt-2 font-mono text-sm">
											{calculateNextVersion(currentVersion, 'major')}
										</div>
									</button>
								</div>
								<div class="mt-3 rounded-lg bg-gray-50 p-3 dark:bg-gray-800">
									<div class="text-sm text-gray-600 dark:text-gray-400">
										Current version: <span class="font-mono font-semibold">{currentVersion}</span>
										→ New version:
										<span class="font-mono font-semibold text-blue-600 dark:text-blue-400"
											>{nextVersion}</span
										>
									</div>
								</div>
							</fieldset>

							<!-- Release Notes -->
							<FormField
								id="upload-notes"
								label="Release Notes"
								value={uploadFormData.notes}
								placeholder={versionType === 'patch'
									? 'Bug fixes and improvements'
									: versionType === 'minor'
										? 'New features and enhancements'
										: 'Major changes and breaking updates'}
								disabled={uploading}
								oninput={(e) => (uploadFormData.notes = (e.target as HTMLInputElement).value)}
							/>

							<!-- File Upload -->
							<FileUpload
								id="upload-deployment-zip"
								label="Deployment ZIP"
								accept=".zip,application/zip"
								maxSize={150 * 1024 * 1024}
								required
								disabled={uploading}
								value={deploymentFile}
								errorText={fileError}
								helperText={deploymentFile
									? `File selected: ${deploymentFile.name} (${Math.round(deploymentFile.size / 1024 / 1024)}MB)`
									: 'Upload your PocketBase distribution as a ZIP file (max 150MB)'}
								onFileSelect={handleFileSelect}
								onError={handleFileError}
							/>

							<!-- Upload Button -->
							<div class="flex justify-end">
								<Button
									variant="primary"
									onclick={handleUploadSubmit}
									disabled={uploading || !deploymentFile || !!fileError}
									loading={uploading}
								>
									{#snippet iconSnippet()}
										<Icon name="upload" />
									{/snippet}
									{uploading ? 'Uploading...' : 'Upload Version'}
								</Button>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Versions List -->
			<div class="space-y-4">
				{#if loading}
					<div class="flex items-center justify-center py-8">
						<div class="text-sm text-gray-500 dark:text-gray-400">Loading versions...</div>
					</div>
				{:else if versions.length === 0}
					<div
						class="rounded-lg border border-gray-200 bg-gray-50 p-8 text-center dark:border-gray-700 dark:bg-gray-900"
					>
						<Icon name="apps" size="h-12 w-12" class="mx-auto text-gray-400" />
						<h3 class="mt-4 text-lg font-medium text-gray-900 dark:text-gray-100">
							No versions uploaded
						</h3>
						<p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
							Upload your first version to start deploying this application
						</p>
					</div>
				{:else}
					<div class="space-y-3">
						{#each versions as version (version.id)}
							{@const versionStatus = getVersionStatus(version)}
							<div
								class="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-950"
							>
								<div class="flex items-center justify-between">
									<div class="flex items-center space-x-4">
										<div>
											<div class="flex items-center space-x-2">
												<span class="font-mono font-semibold text-gray-900 dark:text-gray-100">
													v{version.version_number}
												</span>
												<StatusBadge
													status={versionStatus.text}
													variant={versionStatus.variant}
													size="xs"
												/>
											</div>
											{#if version.notes}
												<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{version.notes}</p>
											{/if}
											<div
												class="mt-2 flex items-center space-x-4 text-xs text-gray-500 dark:text-gray-400"
											>
												<span>Uploaded {formatDate(version.created)}</span>
												{#if version.deployment_zip}
													<span>• ZIP included</span>
												{/if}
											</div>
										</div>
									</div>
									<div class="flex items-center space-x-2">
										{#if canDeleteVersion(version)}
											<Button
												variant="ghost"
												color="red"
												size="sm"
												onclick={() => openDeleteModal(version)}
												disabled={deleting || uploading}
											>
												{#snippet iconSnippet()}
													<Icon name="delete" />
												{/snippet}
												Delete
											</Button>
										{:else}
											<div class="text-xs text-gray-500 dark:text-gray-400">Currently deployed</div>
										{/if}
									</div>
								</div>
							</div>
						{/each}
					</div>

					<div class="flex items-center justify-between pt-4">
						<p class="text-sm text-gray-600 dark:text-gray-400">
							{versions.length} version{versions.length !== 1 ? 's' : ''} total
						</p>
						<Button
							variant="ghost"
							size="sm"
							onclick={loadVersions}
							disabled={loading || uploading || deleting}
						>
							{#snippet iconSnippet()}
								<Icon name="refresh" />
							{/snippet}
							Refresh
						</Button>
					</div>
				{/if}
			</div>
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No application selected</div>
		</div>
	{/if}
</Modal>

<!-- Delete Version Modal -->
<DeleteModal
	open={showDeleteModal}
	item={versionToDelete
		? { id: versionToDelete.id, name: `v${versionToDelete.version_number}` }
		: null}
	itemType="version"
	loading={deleting}
	onclose={() => {
		showDeleteModal = false;
		versionToDelete = null;
	}}
	onconfirm={(id) => handleDeleteConfirm(id)}
/>
