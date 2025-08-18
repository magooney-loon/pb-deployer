<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField } from '$lib/components/partials';

	interface ServerFormData {
		name: string;
		host: string;
		port: number;
		root_username: string;
		app_username: string;
	}

	interface Props {
		open?: boolean;
		creating?: boolean;
		onclose?: () => void;
		oncreate?: (serverData: ServerFormData) => Promise<void>;
	}

	let { open = false, creating = false, onclose, oncreate }: Props = $props();

	let formData = $state<ServerFormData>({
		name: '',
		host: '',
		port: 22,
		root_username: 'root',
		app_username: 'pocketbase'
	});

	let hostError = $state<string | undefined>(undefined);

	function validateHost(host: string): string | undefined {
		if (!host.trim()) {
			return 'IP address or hostname is required';
		}

		const trimmedHost = host.trim();

		// Check for invalid prefixes
		if (trimmedHost.startsWith('http://') || trimmedHost.startsWith('https://')) {
			return 'Remove http:// or https:// prefix';
		}

		if (trimmedHost.startsWith('www.')) {
			return 'Remove www. prefix';
		}

		// Check if it's a valid IP address (IPv4)
		const ipv4Regex =
			/^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;

		// Check if it's a valid hostname/domain (must have TLD)
		const hostnameRegex =
			/^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$/;

		const isValidIP = ipv4Regex.test(trimmedHost);
		const isValidHostname = hostnameRegex.test(trimmedHost);

		if (!isValidIP && !isValidHostname) {
			return 'Enter a valid IP address or hostname with TLD (e.g., server.com)';
		}

		return undefined;
	}

	function handleClose() {
		if (!creating) {
			resetForm();
			onclose?.();
		}
	}

	function resetForm() {
		formData = {
			name: '',
			host: '',
			port: 22,
			root_username: 'root',
			app_username: 'pocketbase'
		};
		hostError = undefined;
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		if (oncreate && !creating) {
			await oncreate(formData);
		}
	}

	function handleHostInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		formData.host = value;
		hostError = validateHost(value);
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

	// Validate host on initial load
	$effect(() => {
		if (formData.host) {
			hostError = validateHost(formData.host);
		}
	});
</script>

<Modal {open} title="Add New Server" size="xl" closeable={!creating} onclose={handleClose}>
	<div class="max-h-[70vh] overflow-y-auto">
		<form
			id="server-form"
			onsubmit={handleSubmit}
			class="space-y-8"
			autocomplete="off"
			novalidate
			data-form-type="other"
		>
			<!-- Server Connection Details -->
			<div class="space-y-4">
				<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Server Connection</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Configure your production server connection details
					</p>
				</div>

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<div class="lg:col-span-2">
						<FormField
							id="server-name"
							label="Server Name"
							value={formData.name}
							placeholder="e.g., Production Server, Main VPS"
							required
							disabled={creating}
							oninput={(e) => (formData.name = (e.target as HTMLInputElement).value)}
							helperText="A friendly name to identify this server"
						/>
					</div>

					<FormField
						id="server-host"
						label="IP Address / Hostname"
						value={formData.host}
						placeholder="192.168.1.100 or server.example.com"
						errorText={hostError}
						required
						disabled={creating}
						oninput={handleHostInput}
						helperText="The IP address or hostname of your VPS"
					/>

					<FormField
						id="server-port"
						label="SSH Port"
						type="number"
						value={formData.port}
						min={1}
						max={65535}
						disabled={true}
						oninput={(e) => (formData.port = parseInt((e.target as HTMLInputElement).value) || 22)}
						helperText="Standard SSH port (locked to 22)"
					/>
				</div>
			</div>

			<!-- User Configuration -->
			<div class="space-y-4">
				<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">User Configuration</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Configure the user accounts for server access
					</p>
				</div>

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<FormField
						id="server-root-username"
						label="Root Username"
						value={formData.root_username}
						placeholder="root"
						disabled={true}
						oninput={(e) => (formData.root_username = (e.target as HTMLInputElement).value)}
						helperText="Username with sudo privileges (locked)"
					/>

					<FormField
						id="server-app-username"
						label="Application Username"
						value={formData.app_username}
						placeholder="pocketbase"
						disabled={true}
						oninput={(e) => (formData.app_username = (e.target as HTMLInputElement).value)}
						helperText="Non-privileged user for running applications (locked)"
					/>
				</div>
			</div>

			<!-- SSH Agent Info -->
			<div
				class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-700 dark:bg-blue-900/20"
			>
				<div class="flex items-start space-x-3">
					<svg
						class="mt-0.5 h-5 w-5 text-blue-600 dark:text-blue-400"
						fill="currentColor"
						viewBox="0 0 20 20"
					>
						<path
							fill-rule="evenodd"
							d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
							clip-rule="evenodd"
						/>
					</svg>
					<div class="flex-1">
						<h4 class="font-medium text-blue-900 dark:text-blue-100">SSH Authentication</h4>
						<p class="mt-1 text-sm text-blue-800 dark:text-blue-200">
							Ensure your SSH agent is running and has your keys loaded before setup.
						</p>
						<div class="mt-2 text-xs text-blue-700 dark:text-blue-300">
							<code class="rounded bg-blue-100 px-1 py-0.5 dark:bg-blue-800">ssh-add -l</code> to verify
							loaded keys
						</div>
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
				disabled={creating || !formData.name || !formData.host || !!hostError}
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
					Add Server
				{/if}
			</Button>
		</div>
	{/snippet}
</Modal>
