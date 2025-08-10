<script lang="ts">
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

	// Type-specific styles
	const typeStyles = {
		error: {
			container: 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900',
			icon: 'text-red-400',
			title: 'text-red-800 dark:text-red-200',
			message: 'text-red-700 dark:text-red-300',
			button:
				'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-800 dark:text-red-200 dark:hover:bg-red-700'
		},
		warning: {
			container: 'border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-900',
			icon: 'text-yellow-400',
			title: 'text-yellow-800 dark:text-yellow-200',
			message: 'text-yellow-700 dark:text-yellow-300',
			button:
				'bg-yellow-100 text-yellow-800 hover:bg-yellow-200 dark:bg-yellow-800 dark:text-yellow-200 dark:hover:bg-yellow-700'
		},
		info: {
			container: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900',
			icon: 'text-blue-400',
			title: 'text-blue-800 dark:text-blue-200',
			message: 'text-blue-700 dark:text-blue-300',
			button:
				'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-800 dark:text-blue-200 dark:hover:bg-blue-700'
		},
		success: {
			container: 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900',
			icon: 'text-green-400',
			title: 'text-green-800 dark:text-green-200',
			message: 'text-green-700 dark:text-green-300',
			button:
				'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-800 dark:text-green-200 dark:hover:bg-green-700'
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

<div class="mb-6 rounded-lg border p-4 {styles.container} {className}">
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
					<button onclick={onDismiss} class="rounded px-3 py-1 text-sm {styles.button}">
						Dismiss
					</button>
				</div>
			{/if}
		</div>
	</div>
</div>
