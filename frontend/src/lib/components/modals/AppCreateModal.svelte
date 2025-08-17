<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField } from '$lib/components/partials';
	import type { Server } from '$lib/api/index.js';

	interface AppFormData {
		name: string;
		server_id: string;
		domain: string;
		remote_path: string;
		service_name: string;
		version_number: string;
		version_notes: string;
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

	let availableServers = $derived(servers.filter((s) => s.setup_complete));
	let selectedServer = $derived(availableServers.find((s) => s.id === formData.server_id));

	let suggestedRemotePath = $derived(formData.name ? `/opt/pocketbase/apps/${formData.name}` : '');
	let suggestedServiceName = $derived(formData.name ? `pocketbase-${formData.name}` : '');

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
			</div>

			<!-- Deployment Configuration -->
			<div class="space-y-4">
				{#if formData.name}
					<div
						class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-600 dark:bg-gray-800/50"
					>
						<h4 class="mb-2 font-medium text-gray-900 dark:text-gray-100">
							Generated Paths Preview
						</h4>
						<div class="space-y-1 text-sm">
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Remote Path:</span>
								<code class="text-gray-900 dark:text-gray-100"
									>{formData.remote_path || suggestedRemotePath}</code
								>
							</div>
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Service Name:</span>
								<code class="text-gray-900 dark:text-gray-100"
									>{formData.service_name || suggestedServiceName}</code
								>
							</div>
						</div>
					</div>
				{/if}
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
					availableServers.length === 0}
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
