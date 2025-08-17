<script lang="ts">
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import Icon from '../icons/Icon.svelte';
	let {
		message,
		type = 'error',
		icon,
		iconSnippet,
		dismissible = true,
		onDismiss,
		delay = 150,
		class: className = ''
	}: {
		message: string;
		type?: 'error' | 'warning' | 'info' | 'success';
		icon?: string;
		iconSnippet?: import('svelte').Snippet;
		dismissible?: boolean;
		onDismiss?: () => void;
		delay?: number;
		class?: string;
	} = $props();

	const typeStyles = {
		error: {
			container: 'bg-red-600 text-white shadow-xl border border-red-500',
			icon: 'text-red-100',
			message: 'text-red-100',
			button: 'text-red-200 hover:text-white hover:bg-red-500/20 rounded-full'
		},
		warning: {
			container: 'bg-amber-600 text-white shadow-xl border border-amber-500',
			icon: 'text-amber-100',
			message: 'text-amber-100',
			button: 'text-amber-200 hover:text-white hover:bg-amber-500/20 rounded-full'
		},
		info: {
			container: 'bg-blue-600 text-white shadow-xl border border-blue-500',
			icon: 'text-blue-100',
			message: 'text-blue-100',
			button: 'text-blue-200 hover:text-white hover:bg-blue-500/20 rounded-full'
		},
		success: {
			container: 'bg-emerald-600 text-white shadow-xl border border-emerald-500',
			icon: 'text-emerald-100',
			message: 'text-emerald-100',
			button: 'text-emerald-200 hover:text-white hover:bg-emerald-500/20 rounded-full'
		}
	};

	const defaultIconNames = {
		error: 'error',
		warning: 'warning',
		info: 'info',
		success: 'success'
	} as const;

	let currentIconName = $derived(defaultIconNames[type]);
	let styles = $derived(typeStyles[type]);
	let isVisible = $state(false);

	// Show toast after delay
	setTimeout(() => {
		isVisible = true;
	}, delay);

	// Auto-dismiss after 5 seconds
	if (dismissible && onDismiss) {
		setTimeout(() => {
			onDismiss();
		}, 5000);
	}
</script>

{#if isVisible}
	<div
		transition:slide={{ duration: 300, easing: quintOut }}
		class="fixed right-4 bottom-4 left-4 z-50 mx-auto max-w-sm rounded-lg p-3 {styles.container} {className}"
	>
		<div class="flex items-center gap-3">
			<div class="flex-shrink-0">
				<span class="text-lg {styles.icon}">
					{#if iconSnippet}
						{@render iconSnippet()}
					{:else if icon}
						{icon}
					{:else}
						<Icon name={currentIconName} />
					{/if}
				</span>
			</div>
			<div class="min-w-0 flex-1">
				<p class="text-center text-sm font-medium {styles.message} truncate">{message}</p>
			</div>
			{#if dismissible && onDismiss}
				<button
					onclick={onDismiss}
					class="flex-shrink-0 p-1 transition-colors {styles.button}"
					aria-label="Dismiss"
				>
					<Icon name="close" size="h-4 w-4" />
				</button>
			{/if}
		</div>
	</div>
{/if}
