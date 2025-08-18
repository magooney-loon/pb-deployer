<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField, FileUpload } from '$lib/components/partials';
	import type { App } from '$lib/api/index.js';

	interface DeploymentData {
		version_number: string;
		notes: string;
		deploymentZip: File;
	}

	interface Props {
		open?: boolean;
		app?: App | null;
		deploying?: boolean;
		onclose?: () => void;
		ondeploy?: (deploymentData: DeploymentData) => Promise<boolean>;
	}

	let { open = false, app = null, deploying = false, onclose, ondeploy }: Props = $props();

	let formData = $state({
		version_number: '',
		notes: ''
	});

	let deploymentFile = $state<File | null>(null);
	let fileError = $state<string | undefined>(undefined);

	function handleClose() {
		if (!deploying) {
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
		if (ondeploy && !deploying && deploymentFile && app) {
			const success = await ondeploy({
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
			!fileError
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
			disabled={deploying}
			class="px-6 py-2"
		>
			Cancel
		</Button>
		<Button
			variant="primary"
			onclick={handleButtonClick}
			disabled={deploying || !isFormValid}
			class="min-w-[120px] px-6 py-2"
		>
			{#if deploying}
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
				Deploying...
			{:else}
				Deploy Application
			{/if}
		</Button>
	</div>
{/snippet}

<Modal
	{open}
	title="Deploy Application"
	size="xl"
	closeable={!deploying}
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
								Deploying to: {app.name}
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
							Specify the version details for this deployment
						</p>
					</div>

					<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
						<FormField
							id="version-number"
							label="Version Number"
							value={formData.version_number}
							placeholder="1.0.1"
							helperText="Semantic versioning recommended (major.minor.patch)"
							required
							disabled={deploying}
							oninput={(e) => (formData.version_number = (e.target as HTMLInputElement).value)}
						/>

						<FormField
							id="version-notes"
							label="Release Notes"
							value={formData.notes}
							placeholder="Bug fixes and improvements"
							helperText="Brief description of changes in this version"
							required
							disabled={deploying}
							oninput={(e) => (formData.notes = (e.target as HTMLInputElement).value)}
						/>
					</div>
				</div>

				<!-- File Upload -->
				<div class="space-y-4">
					<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
							Deployment Package
						</h3>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Upload your PocketBase distribution as a ZIP file
						</p>
					</div>

					<FileUpload
						id="deployment-zip"
						label="Upload Deployment ZIP"
						accept=".zip,application/zip"
						maxSize={180 * 1024 * 1024}
						required
						disabled={deploying}
						value={deploymentFile}
						errorText={fileError}
						helperText="Maximum file size: 180MB"
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

				<!-- Deployment Process Info -->
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
							<h4 class="font-medium text-amber-900 dark:text-amber-100">Deployment Process</h4>
							<p class="mt-1 text-sm text-amber-800 dark:text-amber-200">
								The ZIP will be uploaded to the server, extracted to the app directory, and the
								service will be restarted automatically. Make sure your binary is compatible with
								the target server architecture.
							</p>
						</div>
					</div>
				</div>
			</form>
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No application selected for deployment</div>
		</div>
	{/if}
</Modal>
