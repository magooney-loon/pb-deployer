<script lang="ts">
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	let {
		message = 'Ohaithere',
		icon = '⚠️',
		dismissible = true,
		color = 'yellow',
		size = 'sm',
		class: className = '',
		onDismiss,
		delay = 150
	}: {
		message?: string;
		icon?: string;
		dismissible?: boolean;
		color?: 'yellow' | 'blue' | 'red' | 'gray';
		size?: 'xs' | 'sm';
		class?: string;
		onDismiss?: () => void;
		delay?: number;
	} = $props();

	let isDismissed = $state(true);

	// Show banner after delay
	setTimeout(() => {
		isDismissed = false;
	}, delay);

	const colorVariants = {
		yellow:
			'bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-950 dark:border-amber-800/50 dark:text-amber-200',
		blue: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-950 dark:border-blue-800/50 dark:text-blue-200',
		red: 'bg-red-50 border-red-200 text-red-800 dark:bg-red-950 dark:border-red-800/50 dark:text-red-200',
		gray: 'bg-gray-50 border-gray-200 text-gray-800 dark:bg-gray-900 dark:border-gray-700/50 dark:text-gray-200'
	};

	const sizeVariants = {
		xs: 'px-2 py-1 text-xs',
		sm: 'px-3 py-1.5 text-xs'
	};

	let bannerClasses = $derived(
		`w-full border-b ${colorVariants[color]} ${sizeVariants[size]} ${className}`
	);

	function handleDismiss() {
		isDismissed = true;
		onDismiss?.();
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && dismissible) {
			handleDismiss();
		}
	}
</script>

{#if !isDismissed}
	<div transition:slide={{ duration: 300, easing: quintOut }} class={bannerClasses} role="alert">
		<div class="mx-auto flex max-w-7xl items-center justify-between gap-2">
			<div class="flex min-w-0 flex-1 items-center gap-2">
				{#if icon}
					<span class="flex-shrink-0">{icon}</span>
				{/if}
				<p class="truncate leading-tight font-medium">
					{message}
				</p>
			</div>

			{#if dismissible}
				<button
					onclick={handleDismiss}
					onkeydown={handleKeydown}
					class="flex-shrink-0 rounded-full p-1 transition-colors hover:bg-black/5 focus:ring-2 focus:ring-current focus:ring-offset-1 focus:outline-none dark:hover:bg-white/5"
					aria-label="Dismiss warning"
				>
					<svg class="h-3 w-3" fill="currentColor" viewBox="0 0 20 20">
						<path
							fill-rule="evenodd"
							d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
							clip-rule="evenodd"
						/>
					</svg>
				</button>
			{/if}
		</div>
	</div>
{/if}
