<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField, FileUpload } from '$lib/components/partials';
	import type { App } from '$lib/api/index.js';

	interface VersionData {
		version_number: string;
		notes: string;
		deploymentZip: File;
	}

	interface Props {
		open?: boolean;
		app?: App | null;
		uploading?: boolean;
		onclose?: () => void;
		onupload?: (versionData: VersionData) => Promise<boolean>;
	}

	let { open = false, app = null, uploading = false, onclose, onupload }: Props = $props();

	let formData = $state({
		version_number: '',
		notes: ''
	});

	let versionType = $state<'patch' | 'minor' | 'major'>('patch');
	let currentVersion = $state('0.0.0');
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

	// Update formData when nextVersion changes
	$effect(() => {
		formData.version_number = nextVersion;
	});

	// Get current version when app changes
	$effect(() => {
		if (app?.current_version) {
			currentVersion = app.current_version;
		} else {
			currentVersion = '0.0.0';
		}
	});

	let deploymentFile = $state<File | null>(null);
	let fileError = $state<string | undefined>(undefined);

	function handleClose() {
		if (!uploading) {
			resetForm();
			onclose?.();
		}
	}

	function resetForm() {
		formData = {
			version_number: '',
			notes: ''
		};
		deploymentFile = null;
		fileError = undefined;
		versionType = 'patch';
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

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (onupload && !uploading && deploymentFile && app) {
			const success = await onupload({
				version_number: formData.version_number,
				notes: formData.notes,
				deploymentZip: deploymentFile
			});

			if (success) {
				resetForm();
			}
		}
	}

	async function handleButtonClick() {
		const fakeEvent = new Event('submit');
		await handleSubmit(fakeEvent);
	}

	let isFormValid = $derived(
		formData.version_number.trim() !== '' &&
			formData.notes.trim() !== '' &&
			deploymentFile !== null &&
			!fileError &&
			!uploading
	);

	$effect(() => {
		if (!open) {
			setTimeout(() => {
				resetForm();
			}, 300);
		}
	});
</script>

{#snippet footer()}
	<div class="flex justify-end space-x-3">
		<Button
			variant="secondary"
			color="gray"
			onclick={handleClose}
			disabled={uploading}
			class="px-6 py-2"
		>
			Cancel
		</Button>
		<Button
			variant="primary"
			onclick={handleButtonClick}
			disabled={uploading || !isFormValid}
			class="min-w-[120px] px-6 py-2"
		>
			{#if uploading}
				<svg
					class="mr-2 -ml-1 h-4 w-4 animate-spin text-white"
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
				>
					<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
					></circle>
					<path
						class="opacity-75"
						fill="currentColor"
						d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
					></path>
				</svg>
				Uploading...
			{:else}
				Upload Version
			{/if}
		</Button>
	</div>
{/snippet}

<Modal
	{open}
	title="Upload New Version"
	size="xl"
	closeable={!uploading}
	onclose={handleClose}
	{footer}
>
	{#if app}
		<div class="max-h-[70vh] overflow-y-auto">
			<form
				id="deploy-form"
				onsubmit={handleSubmit}
				class="space-y-8"
				autocomplete="off"
				novalidate
			>
				<!-- App Info -->
				<div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
					<div class="flex items-center space-x-3">
						<div class="text-blue-600 dark:text-blue-400">
							<svg class="h-6 w-6" fill="currentColor" viewBox="0 0 20 20">
								<path
									d="M3 4a1 1 0 011-1h12a1 1 0 011 1v2a1 1 0 01-1 1H4a1 1 0 01-1-1V4zM3 10a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H4a1 1 0 01-1-1v-6zM14 9a1 1 0 00-1 1v6a1 1 0 001 1h2a1 1 0 001-1v-6a1 1 0 00-1-1h-2z"
								/>
							</svg>
						</div>
						<div>
							<h3 class="font-semibold text-blue-900 dark:text-blue-100">
								Uploading version for: {app.name}
							</h3>
							<p class="text-sm text-blue-700 dark:text-blue-300">
								Domain: {app.domain} • Service: {app.service_name}
							</p>
						</div>
					</div>
				</div>

				<!-- Version Information -->
				<div class="space-y-4">
					<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
							Version Information
						</h3>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Specify the version details for this upload
						</p>
					</div>

					<div class="space-y-6">
						<!-- Version Type Selection -->
						<div>
							<div class="mb-3 block text-sm font-medium text-gray-700 dark:text-gray-300">
								Version Type
							</div>
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
									<div class="mt-1 text-xs text-gray-500 dark:text-gray-400">Breaking changes</div>
									<div class="mt-2 font-mono text-sm">
										{calculateNextVersion(currentVersion, 'major')}
									</div>
								</button>
							</div>
							<div class="mt-3 rounded-lg bg-gray-50 p-3 dark:bg-gray-900">
								<div class="text-sm text-gray-600 dark:text-gray-400">
									Current version: <span class="font-mono font-semibold">{currentVersion}</span>
									→ New version:
									<span class="font-mono font-semibold text-blue-600 dark:text-blue-400"
										>{nextVersion}</span
									>
								</div>
							</div>
						</div>

						<!-- Release Notes -->
						<FormField
							id="version-notes"
							label="Release Notes"
							value={formData.notes}
							placeholder={versionType === 'patch'
								? 'Bug fixes and improvements'
								: versionType === 'minor'
									? 'New features and enhancements'
									: 'Major changes and breaking updates'}
							helperText={formData.notes.trim()
								? `${formData.notes.length}/1000 characters`
								: 'Brief description of changes in this version'}
							required
							disabled={uploading}
							oninput={(e) => (formData.notes = (e.target as HTMLInputElement).value)}
						/>
					</div>
				</div>

				<!-- File Upload -->
				<div class="space-y-4">
					<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Version Package</h3>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Upload your PocketBase distribution as a ZIP file
						</p>
					</div>

					<FileUpload
						id="deployment-zip"
						label="Upload Version ZIP"
						accept=".zip,application/zip"
						maxSize={150 * 1024 * 1024}
						required
						disabled={uploading}
						value={deploymentFile}
						errorText={fileError}
						helperText={deploymentFile
							? `File selected: ${deploymentFile.name} (${Math.round(deploymentFile.size / 1024 / 1024)}MB)`
							: 'Maximum file size: 150MB'}
						onFileSelect={handleFileSelect}
						onError={handleFileError}
					/>

					<!-- ZIP Contents Info -->
					<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-900">
						<h4 class="mb-3 font-medium text-gray-900 dark:text-gray-100">
							Required ZIP Contents:
						</h4>
						<div class="space-y-2 text-sm">
							<div class="flex items-center space-x-2">
								<span class="text-red-600 dark:text-red-400">✓</span>
								<span class="text-gray-700 dark:text-gray-300">
									<strong>PocketBase binary</strong> - The main executable file
								</span>
							</div>
							<div class="flex items-center space-x-2">
								<span class="text-green-600 dark:text-green-400">○</span>
								<span class="text-gray-600 dark:text-gray-400">
									pb_public/ - Static files and admin UI customizations
								</span>
							</div>
							<div class="flex items-center space-x-2">
								<span class="text-green-600 dark:text-green-400">○</span>
								<span class="text-gray-600 dark:text-gray-400">
									pb_migrations/ - Database migration files
								</span>
							</div>
							<div class="flex items-center space-x-2">
								<span class="text-green-600 dark:text-green-400">○</span>
								<span class="text-gray-600 dark:text-gray-400">
									pb_hooks/ - Custom JavaScript hooks
								</span>
							</div>
							<div class="flex items-center space-x-2">
								<span class="text-green-600 dark:text-green-400">○</span>
								<span class="text-gray-600 dark:text-gray-400">
									Any additional files your application needs
								</span>
							</div>
						</div>
						<p class="mt-3 text-xs text-gray-500 dark:text-gray-400">✓ = Required • ○ = Optional</p>
					</div>
				</div>

				<!-- Version Upload Info -->
				<div class="rounded-lg bg-amber-50 p-4 dark:bg-amber-900/20">
					<div class="flex items-start space-x-3">
						<svg
							class="mt-0.5 h-5 w-5 text-amber-600 dark:text-amber-400"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fill-rule="evenodd"
								d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
								clip-rule="evenodd"
							/>
						</svg>
						<div>
							<h4 class="font-medium text-amber-900 dark:text-amber-100">Version Upload</h4>
							<p class="mt-1 text-sm text-amber-800 dark:text-amber-200">
								The ZIP file will be uploaded and stored as a new version. This creates a version
								record that can be deployed later. Make sure your binary is compatible with the
								target server architecture.
							</p>
						</div>
					</div>
				</div>
			</form>
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No application selected for version upload</div>
		</div>
	{/if}
</Modal>
