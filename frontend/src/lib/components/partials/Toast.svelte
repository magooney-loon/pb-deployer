<script lang="ts">
	let {
		message,
		type = 'error',
		icon,
		dismissible = true,
		onDismiss,
		class: className = ''
	}: {
		message: string;
		type?: 'error' | 'warning' | 'info' | 'success';
		icon?: string;
		dismissible?: boolean;
		onDismiss?: () => void;
		class?: string;
	} = $props();

	// Compact toast-style design
	const typeStyles = {
		error: {
			container: 'bg-red-600/50 backdrop-blur-sm text-white shadow-xl border border-red-500',
			icon: 'text-red-100',
			message: 'text-red-100',
			button: 'text-red-200 hover:text-white hover:bg-red-500/20 rounded-full'
		},
		warning: {
			container: 'bg-amber-600/50 backdrop-blur-sm text-white shadow-xl border border-amber-500',
			icon: 'text-amber-100',
			message: 'text-amber-100',
			button: 'text-amber-200 hover:text-white hover:bg-amber-500/20 rounded-full'
		},
		info: {
			container: 'bg-blue-600/50 backdrop-blur-sm text-white shadow-xl border border-blue-500',
			icon: 'text-blue-100',
			message: 'text-blue-100',
			button: 'text-blue-200 hover:text-white hover:bg-blue-500/20 rounded-full'
		},
		success: {
			container:
				'bg-emerald-600/50 backdrop-blur-sm text-white shadow-xl border border-emerald-500',
			icon: 'text-emerald-100',
			message: 'text-emerald-100',
			button: 'text-emerald-200 hover:text-white hover:bg-emerald-500/20 rounded-full'
		}
	};

	const defaultIcons = {
		error: '❌',
		warning: '⚠️',
		info: 'ℹ️',
		success: '✅'
	};

	let currentIcon = $derived(icon || defaultIcons[type]);
	let styles = $derived(typeStyles[type]);
	let isVisible = $state(true);
</script>

{#if isVisible}
	<div
		class="fixed bottom-4 left-1/2 z-50 mx-4 w-full max-w-sm -translate-x-1/2 rounded-lg p-3 md:right-4 md:left-auto md:mx-0 md:translate-x-0 {styles.container} {className}"
	>
		<div class="flex items-center gap-3">
			<div class="flex-shrink-0">
				<span class="text-lg {styles.icon}">{currentIcon}</span>
			</div>
			<div class="min-w-0 flex-1">
				<p class="text-sm font-medium {styles.message} truncate">{message}</p>
			</div>
			{#if dismissible && onDismiss}
				<button
					onclick={onDismiss}
					class="flex-shrink-0 p-1 transition-colors {styles.button}"
					aria-label="Dismiss"
				>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
			{/if}
		</div>
	</div>
{/if}
