<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField } from '$lib/components/partials';

	interface ServerFormData {
		name: string;
		host: string;
		port: number;
		root_username: string;
		app_username: string;
		use_ssh_agent: boolean;
		manual_key_path: string;
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
		app_username: 'pocketbase',
		use_ssh_agent: true,
		manual_key_path: ''
	});

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
			app_username: 'pocketbase',
			use_ssh_agent: true,
			manual_key_path: ''
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
						Configure how to connect to your server
					</p>
				</div>

				<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
					<div class="lg:col-span-2">
						<FormField
							id="server-name"
							label="Server Name"
							value={formData.name}
							placeholder="e.g., Production Server, Dev Environment"
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
						required
						disabled={creating}
						oninput={(e) => (formData.host = (e.target as HTMLInputElement).value)}
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
						helperText="Default SSH port is 22"
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
						placeholder="app"
						disabled={true}
						oninput={(e) => (formData.app_username = (e.target as HTMLInputElement).value)}
						helperText="Non-privileged user for running applications (locked)"
					/>
				</div>
			</div>

			<!-- SSH Authentication -->
			<div class="space-y-4">
				<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">SSH Authentication</h3>
					<p class="text-sm text-gray-500 dark:text-gray-400">
						Choose how to authenticate with the server
					</p>
				</div>

				<div class="space-y-4">
					<!-- SSH Agent Option -->
					<div class="rounded-lg border border-gray-200 p-4 dark:border-gray-700">
						<label class="flex cursor-pointer items-start space-x-3">
							<input
								type="checkbox"
								bind:checked={formData.use_ssh_agent}
								disabled={creating}
								class="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 disabled:opacity-50"
								autocomplete="off"
								data-form-type="other"
							/>
							<div class="flex-1">
								<div class="font-medium text-gray-900 dark:text-gray-100">Use SSH Agent</div>
								<div class="text-sm text-gray-500 dark:text-gray-400">
									Use keys loaded in your SSH agent (recommended)
								</div>
							</div>
						</label>
					</div>

					<!-- Manual Key Path (shown when SSH Agent is disabled) -->
					{#if !formData.use_ssh_agent}
						<div
							class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-900/20"
						>
							<div class="mb-3">
								<div class="flex items-center space-x-2">
									<svg
										class="h-4 w-4 text-amber-600 dark:text-amber-400"
										fill="currentColor"
										viewBox="0 0 20 20"
									>
										<path
											fill-rule="evenodd"
											d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
											clip-rule="evenodd"
										/>
									</svg>
									<span class="font-medium text-amber-800 dark:text-amber-200">
										Manual SSH Key
									</span>
								</div>
								<p class="mt-1 text-sm text-amber-700 dark:text-amber-300">
									Specify the path to your private SSH key file
								</p>
							</div>

							<FormField
								id="server-manual-key-path"
								label="Private Key Path"
								value={formData.manual_key_path}
								placeholder="/home/user/.ssh/id_rsa"
								disabled={creating}
								oninput={(e) => (formData.manual_key_path = (e.target as HTMLInputElement).value)}
								helperText="Full path to your private SSH key file"
							/>
						</div>
					{/if}
				</div>
			</div>

			<!-- Info Box -->
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
						<h4 class="font-medium text-blue-900 dark:text-blue-100">Tested On</h4>
						<p class="mt-1 text-sm text-blue-800 dark:text-blue-200">Hertzner Ubuntu VPS</p>
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
				disabled={creating || !formData.name || !formData.host}
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
					Create Server
				{/if}
			</Button>
		</div>
	{/snippet}
</Modal>
