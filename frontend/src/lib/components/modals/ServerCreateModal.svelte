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
		app_username: 'app',
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
			app_username: 'app',
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

	// Reset form when modal opens/closes
	$effect(() => {
		if (!open) {
			resetForm();
		}
	});
</script>

<Modal {open} title="Add New Server" size="lg" closeable={!creating} onclose={handleClose}>
	<form onsubmit={handleSubmit} class="space-y-4">
		<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
			<FormField
				id="server-name"
				label="Name"
				value={formData.name}
				placeholder="Production Server"
				required
				disabled={creating}
				oninput={(e) => (formData.name = (e.target as HTMLInputElement).value)}
			/>

			<FormField
				id="server-host"
				label="VPS IP"
				value={formData.host}
				placeholder="192.168.1.100"
				required
				disabled={creating}
				oninput={(e) => (formData.host = (e.target as HTMLInputElement).value)}
			/>

			<FormField
				id="server-port"
				label="SSH Port"
				type="number"
				value={formData.port}
				min={1}
				max={65535}
				disabled={creating}
				oninput={(e) => (formData.port = parseInt((e.target as HTMLInputElement).value) || 22)}
			/>

			<FormField
				id="server-root-username"
				label="Root Username"
				value={formData.root_username}
				disabled={creating}
				oninput={(e) => (formData.root_username = (e.target as HTMLInputElement).value)}
			/>

			<FormField
				id="server-app-username"
				label="App Username"
				value={formData.app_username}
				disabled={creating}
				oninput={(e) => (formData.app_username = (e.target as HTMLInputElement).value)}
			/>

			<FormField
				id="server-use-ssh-agent"
				label="Use SSH Agent"
				type="checkbox"
				checked={formData.use_ssh_agent}
				disabled={creating}
				onchange={(e) => (formData.use_ssh_agent = (e.target as HTMLInputElement).checked)}
			/>
		</div>

		{#if !formData.use_ssh_agent}
			<FormField
				id="server-manual-key-path"
				label="Private Key Path"
				value={formData.manual_key_path}
				placeholder="/home/user/.ssh/id_rsa"
				disabled={creating}
				oninput={(e) => (formData.manual_key_path = (e.target as HTMLInputElement).value)}
			/>
		{/if}
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
				{creating ? 'Creating...' : 'Create Server'}
			</Button>
		</div>
	{/snippet}
</Modal>
