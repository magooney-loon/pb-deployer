<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField, FileUpload } from '$lib/components/partials';
	import type { Server } from '$lib/api/index.js';

	interface AppFormData {
		name: string;
		server_id: string;
		domain: string;
		remote_path: string;
		service_name: string;
		version_number: string;
		version_notes: string;
		initialZip?: File;
	}

	interface Props {
		open?: boolean;
		servers?: Server[];
		creating?: boolean;
		onclose?: () => void;
		oncreate?: (appData: AppFormData) => Promise<void>;
	}

	let { open = false, servers = [], creating = false, onclose, oncreate }: Props = $props();

	let formData = $state<AppFormData>({
		name: '',
		server_id: '',
		domain: '',
		remote_path: '',
		service_name: '',
		version_number: '1.0.0',
		version_notes: 'Initial version'
	});

	let initialZipFile = $state<File | null>(null);
	let fileError = $state<string | undefined>(undefined);

	let availableServers = $derived(servers.filter((s) => s.setup_complete));
	let selectedServer = $derived(availableServers.find((s) => s.id === formData.server_id));

	function handleClose() {
		if (!creating) {
			resetForm();
			onclose?.();
		}
	}

	function resetForm() {
		formData = {
			name: '',
			server_id: '',
			domain: '',
			remote_path: '',
			service_name: '',
			version_number: '1.0.0',
			version_notes: 'Initial version'
		};
		initialZipFile = null;
		fileError = undefined;
	}

	function handleFileSelect(file: File | File[] | null) {
		fileError = undefined;
		if (file && !Array.isArray(file)) {
			initialZipFile = file;
			formData.initialZip = file;
		} else {
			initialZipFile = null;
			formData.initialZip = undefined;
		}
	}

	function handleFileError(error: string) {
		fileError = error;
		initialZipFile = null;
		formData.initialZip = undefined;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (oncreate && !creating) {
			await oncreate(formData);
		}
	}

	async function handleButtonClick() {
		const fakeEvent = new Event('submit');
		await handleSubmit(fakeEvent);
	}

	$effect(() => {
		if (!open) {
			// Delay to allow modal animation to complete
			setTimeout(() => {
				resetForm();
			}, 300);
		}
	});
</script>

<Modal {open} title="Add New Application" size="xl" closeable={!creating} onclose={handleClose}>
	<div class="max-h-[70vh] overflow-y-auto">
		<form
			id="app-form"
			onsubmit={handleSubmit}
			class="space-y-8"
			autocomplete="off"
			novalidate
			data-form-type="other"
		>
			<!-- App Configuration -->
			<div class="space-y-4">
				<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
						Application Configuration
					</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Basic information about your application
					</p>
				</div>

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<FormField
						id="app-name"
						label="Application Name"
						value={formData.name}
						placeholder="my-awesome-app"
						helperText="Used for directory and service naming (lowercase, no spaces)"
						required
						disabled={creating}
						oninput={(e) => (formData.name = (e.target as HTMLInputElement).value)}
					/>

					<FormField
						id="server-select"
						label="Target Server"
						type="select"
						value={formData.server_id}
						placeholder="Select a server"
						options={availableServers.map((server) => ({
							value: server.id,
							label: `${server.name} (${server.host})`
						}))}
						required
						disabled={creating}
						onchange={(e) => (formData.server_id = (e.target as HTMLSelectElement).value)}
						helperText="Choose which server to deploy this app to"
					/>

					<div class="lg:col-span-2">
						<FormField
							id="domain"
							label="Domain"
							value={formData.domain}
							placeholder="myapp.example.com"
							helperText="The domain where your app will be served"
							required
							disabled={creating}
							oninput={(e) => (formData.domain = (e.target as HTMLInputElement).value)}
						/>
					</div>
				</div>

				{#if selectedServer}
					<div
						class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-700 dark:bg-green-900/20"
					>
						<div class="flex items-center space-x-2">
							<svg
								class="h-4 w-4 text-green-600 dark:text-green-400"
								fill="currentColor"
								viewBox="0 0 20 20"
							>
								<path
									fill-rule="evenodd"
									d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
									clip-rule="evenodd"
								/>
							</svg>
							<span class="font-medium text-green-800 dark:text-green-200">Server Selected</span>
						</div>
						<p class="mt-1 text-sm text-green-700 dark:text-green-300">
							<strong>{selectedServer.name}</strong> ({selectedServer.host}) - Ready for deployment
						</p>
					</div>
				{/if}
			</div>

			<!-- Version Information -->
			<div class="space-y-4">
				<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Initial Version</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Set up the first version of your application
					</p>
				</div>

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<FormField
						id="version-number"
						label="Version Number"
						value={formData.version_number}
						placeholder="1.0.0"
						helperText="Semantic versioning recommended (major.minor.patch)"
						required
						disabled={creating}
						oninput={(e) => (formData.version_number = (e.target as HTMLInputElement).value)}
					/>

					<FormField
						id="version-notes"
						label="Version Notes"
						value={formData.version_notes}
						placeholder="Initial release"
						helperText="Brief description of this version"
						disabled={creating}
						oninput={(e) => (formData.version_notes = (e.target as HTMLInputElement).value)}
					/>
				</div>

				<!-- Initial ZIP Upload -->
				<div class="space-y-4">
					<FileUpload
						id="initial-zip"
						label="Upload Initial PocketBase Package (Optional)"
						accept=".zip,application/zip"
						maxSize={150 * 1024 * 1024}
						disabled={creating}
						value={initialZipFile}
						errorText={fileError}
						helperText="Upload your PocketBase distribution as a ZIP file (150MB max)"
						onFileSelect={handleFileSelect}
						onError={handleFileError}
					/>

					<!-- ZIP Info -->
					<div class="rounded-lg bg-gray-50 p-4 dark:bg-gray-900">
						<h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-gray-100">
							ZIP Package Contents:
						</h4>
						<div class="space-y-1 text-xs">
							<div class="flex items-center space-x-2">
								<span class="text-red-600 dark:text-red-400">✓</span>
								<span class="text-gray-700 dark:text-gray-300">
									<strong>PocketBase binary</strong> - The main executable
								</span>
							</div>
							<div class="flex items-center space-x-2">
								<span class="text-green-600 dark:text-green-400">○</span>
								<span class="text-gray-600 dark:text-gray-400">
									pb_public/, pb_migrations/, pb_hooks/, etc.
								</span>
							</div>
						</div>
						<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
							✓ = Required • ○ = Optional | You can also upload the initial version later
						</p>
					</div>
				</div>
			</div>
		</form>
	</div>

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button
				variant="secondary"
				color="gray"
				onclick={handleClose}
				disabled={creating}
				class="px-6 py-2"
			>
				Cancel
			</Button>
			<Button
				variant="primary"
				onclick={handleButtonClick}
				disabled={creating ||
					!formData.name ||
					!formData.server_id ||
					!formData.domain ||
					availableServers.length === 0 ||
					!!fileError}
				class="min-w-[120px] px-6 py-2"
			>
				{#if creating}
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
					Creating...
				{:else}
					Create Application
				{/if}
			</Button>
		</div>
	{/snippet}
</Modal>
