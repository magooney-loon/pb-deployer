<script lang="ts">
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	let {
		message,
		title = 'Error',
		type = 'error',
		icon,
		dismissible = true,
		onDismiss,
		class: className = ''
	}: {
		message: string;
		title?: string;
		type?: 'error' | 'warning' | 'info' | 'success';
		icon?: string;
		dismissible?: boolean;
		onDismiss?: () => void;
		class?: string;
	} = $props();

	// Type-specific styles - Vercel-inspired minimal design
	const typeStyles = {
		error: {
			container: 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800',
			icon: 'text-red-500',
			title: 'text-red-700 dark:text-red-300',
			message: 'text-red-600 dark:text-red-400',
			button:
				'bg-red-100 text-red-700 hover:bg-red-200 ring-1 ring-red-200 hover:ring-red-300 dark:bg-red-900 dark:text-red-300 dark:hover:bg-red-800 dark:ring-red-700 dark:hover:ring-red-600'
		},
		warning: {
			container: 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800',
			icon: 'text-amber-500',
			title: 'text-amber-700 dark:text-amber-300',
			message: 'text-amber-600 dark:text-amber-400',
			button:
				'bg-amber-100 text-amber-700 hover:bg-amber-200 ring-1 ring-amber-200 hover:ring-amber-300 dark:bg-amber-900 dark:text-amber-300 dark:hover:bg-amber-800 dark:ring-amber-700 dark:hover:ring-amber-600'
		},
		info: {
			container: 'bg-gray-50 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-700',
			icon: 'text-gray-500',
			title: 'text-gray-700 dark:text-gray-300',
			message: 'text-gray-600 dark:text-gray-400',
			button:
				'bg-gray-100 text-gray-700 hover:bg-gray-200 ring-1 ring-gray-200 hover:ring-gray-300 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 dark:ring-gray-600 dark:hover:ring-gray-500'
		},
		success: {
			container: 'bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800',
			icon: 'text-emerald-500',
			title: 'text-emerald-700 dark:text-emerald-300',
			message: 'text-emerald-600 dark:text-emerald-400',
			button:
				'bg-emerald-100 text-emerald-700 hover:bg-emerald-200 ring-1 ring-emerald-200 hover:ring-emerald-300 dark:bg-emerald-900 dark:text-emerald-300 dark:hover:bg-emerald-800 dark:ring-emerald-700 dark:hover:ring-emerald-600'
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
</script>

<div
	in:slide={{ duration: 300, easing: quintOut }}
	out:slide={{ duration: 300, easing: quintOut }}
	class="mb-6 rounded-lg p-4 {styles.container} {className}"
>
	<div class="flex">
		<div class="flex-shrink-0">
			<span class={styles.icon}>{currentIcon}</span>
		</div>
		<div class="ml-3 flex-1">
			<h3 class="text-sm font-medium {styles.title}">{title}</h3>
			<div class="mt-2 text-sm {styles.message}">
				<p>{message}</p>
			</div>
			{#if dismissible && onDismiss}
				<div class="mt-4">
					<button
						onclick={onDismiss}
						class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors {styles.button}"
					>
						Dismiss
					</button>
				</div>
			{/if}
		</div>
	</div>
</div>
