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

	// Get available servers (setup complete)
	let availableServers = $derived(servers.filter((s) => s.setup_complete));

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

	// Reset form when modal opens/closes
	$effect(() => {
		if (!open) {
			resetForm();
		}
	});
</script>

<Modal {open} title="Add New Application" size="lg" closeable={!creating} onclose={handleClose}>
	<form onsubmit={handleSubmit} class="space-y-6">
		<!-- Basic App Information -->
		<div class="space-y-4">
			<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">App Configuration</h3>
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<FormField
					id="app-name"
					label="App Name"
					value={formData.name}
					placeholder="my-app"
					helperText="Used for directory and service naming"
					required
					disabled={creating}
					oninput={(e) => (formData.name = (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="server-select"
					label="Server"
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
				/>

				<FormField
					id="domain"
					label="Domain"
					value={formData.domain}
					placeholder="myapp.example.com"
					helperText="The domain where your app will be accessible"
					class="md:col-span-2"
					required
					disabled={creating}
					oninput={(e) => (formData.domain = (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="remote-path"
					label="Remote Path (Optional)"
					value={formData.remote_path}
					placeholder="/opt/pocketbase/apps/{formData.name || 'app-name'}"
					disabled={creating}
					oninput={(e) => (formData.remote_path = (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="service-name"
					label="Service Name (Optional)"
					value={formData.service_name}
					placeholder="pocketbase-{formData.name || 'app-name'}"
					disabled={creating}
					oninput={(e) => (formData.service_name = (e.target as HTMLInputElement).value)}
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
					value={formData.version_number}
					placeholder="1.0.0"
					helperText="Semantic versioning recommended"
					required
					disabled={creating}
					oninput={(e) => (formData.version_number = (e.target as HTMLInputElement).value)}
				/>

				<FormField
					id="version-notes"
					label="Version Notes"
					value={formData.version_notes}
					placeholder="Initial release"
					helperText="Describe this version"
					disabled={creating}
					oninput={(e) => (formData.version_notes = (e.target as HTMLInputElement).value)}
				/>
			</div>
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
					<h3 class="text-sm font-medium text-blue-800 dark:text-blue-200">CRUD Only Mode</h3>
					<div class="mt-2 text-sm text-blue-700 dark:text-blue-300">
						<p>
							This creates the app entry in the database. For file uploads and deployments, use
							external tools or the version management system.
						</p>
					</div>
				</div>
			</div>
		</div>
	</form>

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button variant="secondary" color="gray" onclick={handleClose} disabled={creating}>
				Cancel
			</Button>
			<Button
				variant="outline"
				type="submit"
				disabled={creating}
				icon={creating ? '🔄' : undefined}
			>
				{creating ? 'Creating...' : 'Create App'}
			</Button>
		</div>
	{/snippet}
</Modal>
