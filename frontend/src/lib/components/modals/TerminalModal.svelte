<script lang="ts">
	import { onDestroy } from 'svelte';
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button } from '$lib/components/partials';
	import type { Server } from '$lib/api/index.js';

	interface Props {
		open: boolean;
		server: Server | null;
		onclose: () => void;
	}

	let { open, server, onclose }: Props = $props();

	let terminalEl: HTMLDivElement | undefined = $state();
	let term: import('@xterm/xterm').Terminal | undefined;
	let fitAddon: import('@xterm/addon-fit').FitAddon | undefined;
	let socket: WebSocket | undefined;
	let connected = $state(false);
	let errorMsg = $state('');
	let initialized = false;

	async function initTerminal() {
		if (!terminalEl || !server || initialized) return;
		initialized = true;

		const { Terminal } = await import('@xterm/xterm');
		const { FitAddon } = await import('@xterm/addon-fit');
		await import('@xterm/xterm/css/xterm.css');

		term = new Terminal({
			cursorBlink: true,
			cols: 220,
			rows: 40,
			theme: { background: '#0f0f0f' },
			fontFamily: 'monospace',
			fontSize: 13
		});

		fitAddon = new FitAddon();
		term.loadAddon(fitAddon);
		term.open(terminalEl);
		fitAddon.fit();

		const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const url = `${proto}//${window.location.host}/api/terminal?host=${encodeURIComponent(server.host)}&port=${server.port || 22}&user=${encodeURIComponent(server.root_username)}`;

		socket = new WebSocket(url);
		socket.binaryType = 'arraybuffer';

		socket.onopen = () => {
			connected = true;
			errorMsg = '';
		};
		socket.onclose = () => {
			connected = false;
			term?.write('\r\n\x1b[33m[Connection closed]\x1b[0m\r\n');
		};
		socket.onerror = () => {
			errorMsg = 'WebSocket connection failed';
			connected = false;
		};
		socket.onmessage = (e) => {
			term?.write(new Uint8Array(e.data as ArrayBuffer));
		};
		term.onData((data) => {
			if (socket?.readyState === WebSocket.OPEN) socket.send(data);
		});
	}

	function cleanup() {
		socket?.close();
		term?.dispose();
		term = undefined;
		socket = undefined;
		fitAddon = undefined;
		connected = false;
		errorMsg = '';
		initialized = false;
	}

	$effect(() => {
		if (open && server) {
			// defer so bind:this is resolved before we access terminalEl
			setTimeout(initTerminal, 0);
		}
		return () => {
			if (!open) cleanup();
		};
	});

	onDestroy(cleanup);

	function handleClose() {
		cleanup();
		onclose();
	}
</script>

<Modal {open} title="Terminal — {server?.name ?? ''}" size="xl" onclose={handleClose}>
	<div class="flex flex-col gap-3">
		{#if errorMsg}
			<div class="rounded border border-red-700 bg-red-950 px-3 py-2 text-sm text-red-300">
				{errorMsg}
			</div>
		{/if}
		<div
			bind:this={terminalEl}
			class="min-h-[400px] w-full overflow-hidden rounded bg-[#0f0f0f]"
		></div>
		<div class="flex items-center gap-2 text-xs text-gray-500">
			<span class="h-2 w-2 rounded-full {connected ? 'bg-green-500' : 'bg-gray-600'}"></span>
			{connected ? `Connected to ${server?.host}` : 'Disconnected'}
		</div>
	</div>

	{#snippet footer()}
		<div class="flex justify-end">
			<Button variant="outline" onclick={handleClose}>Close</Button>
		</div>
	{/snippet}
</Modal>
