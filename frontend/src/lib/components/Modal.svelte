<script lang="ts">
	import { onMount } from 'svelte';
	import type { Snippet } from 'svelte';

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

	// Handle escape key
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && closeable && open) {
			close();
		}
	}

	function close() {
		onclose?.();
	}

	// Manage body scroll
	$effect(() => {
		if (open) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}

		return () => {
			document.body.style.overflow = '';
		};
	});

	onMount(() => {
		return () => {
			// Cleanup: restore scroll when component unmounts
			document.body.style.overflow = '';
		};
	});

	// Size classes
	const sizeClasses = {
		sm: 'max-w-md',
		md: 'max-w-lg',
		lg: 'max-w-2xl',
		xl: 'max-w-4xl'
	};
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
	<!-- Backdrop -->
	<div class="bg-opacity-50 fixed inset-0 z-50 bg-black backdrop-blur-sm" role="presentation">
		<!-- Modal Container -->
		<div class="fixed inset-0 z-50 flex items-center justify-center p-4">
			<div
				class="relative w-full {sizeClasses[
					size
				]} modal-appear max-h-[90vh] overflow-hidden rounded-lg bg-white shadow-xl dark:bg-gray-800"
				role="dialog"
				aria-modal="true"
				tabindex="-1"
			>
				<!-- Header -->
				{#if title || closeable}
					<div
						class="flex items-center justify-between border-b border-gray-200 px-6 py-4 dark:border-gray-700"
					>
						{#if title}
							<h2 class="text-xl font-semibold text-gray-900 dark:text-white">
								{title}
							</h2>
						{:else}
							<div></div>
						{/if}

						{#if closeable}
							<button
								onclick={close}
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
