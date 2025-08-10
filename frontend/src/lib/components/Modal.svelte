<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';
	import { ModalLogic, type ModalState } from './logic/Modal.js';

	interface Props {
		open?: boolean;
		title?: string;
		size?: 'sm' | 'md' | 'lg' | 'xl';
		closeable?: boolean;
		onclose?: () => void;
		children?: Snippet;
		footer?: Snippet;
	}

	let {
		open = false,
		title = '',
		size = 'md',
		closeable = true,
		onclose,
		children,
		footer
	}: Props = $props();

	// Create logic instance
	const logic = new ModalLogic({ open, title, size, closeable, onclose });
	let state = $state<ModalState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update props when they change
	$effect(() => {
		logic.updateProps({ open, title, size, closeable, onclose });
	});

	onMount(() => {
		return () => {
			logic.cleanup();
		};
	});
</script>

<svelte:window onkeydown={logic.getKeydownHandler()} />

{#if state.open}
	<!-- Backdrop -->
	<div
		class="bg-opacity-50 fixed inset-0 z-50 bg-black backdrop-blur-sm"
		role="presentation"
		onclick={(e) => logic.handleBackdropClick(e)}
	>
		<!-- Modal Container -->
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<div
				class="relative w-full {logic.getSizeClass()} modal-appear max-h-[90vh] overflow-hidden rounded-lg bg-white shadow-xl dark:bg-gray-800"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
			>
				<!-- Header -->
				{#if state.title || state.closeable}
					<div
						class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700"
					>
						{#if state.title}
							<h2 class="text-xl font-semibold text-gray-900 dark:text-white">
								{state.title}
							</h2>
						{:else}
							<div></div>
						{/if}

						{#if state.closeable}
							<button
								onclick={() => logic.close()}
								class="p-1 text-gray-400 transition-colors hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
								aria-label="Close modal"
							>
								<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M6 18L18 6M6 6l12 12"
									></path>
								</svg>
							</button>
						{/if}
					</div>
				{/if}

				<!-- Content -->
				<div class="max-h-[calc(90vh-8rem)] overflow-y-auto px-6 py-4">
					{@render children?.()}
				</div>

				<!-- Footer -->
				{#if footer}
					<div
						class="border-t border-gray-200 bg-gray-50 px-6 py-4 dark:border-gray-700 dark:bg-gray-900"
					>
						{@render footer?.()}
					</div>
				{/if}
			</div>
		</div>
	</div>
{/if}

<style>
	.modal-appear {
		animation: modal-appear 0.15s ease-out;
	}

	@keyframes modal-appear {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}
</style>
