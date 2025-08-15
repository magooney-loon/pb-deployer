<script lang="ts">
	let {
		text = 'Loading...',
		size = 'md',
		color = 'blue',
		centered = true,
		class: className = ''
	}: {
		text?: string;
		size?: 'sm' | 'md' | 'lg';
		color?: 'blue' | 'gray' | 'green' | 'red' | 'yellow';
		centered?: boolean;
		class?: string;
	} = $props();

	const sizeClasses = {
		sm: 'h-4 w-4',
		md: 'h-8 w-8',
		lg: 'h-12 w-12'
	};

	const colorClasses = {
		blue: 'border-gray-900 dark:border-gray-100',
		gray: 'border-gray-400 dark:border-gray-600',
		green: 'border-emerald-600 dark:border-emerald-400',
		red: 'border-red-500 dark:border-red-400',
		yellow: 'border-amber-500 dark:border-amber-400'
	};

	let spinnerClasses = $derived(
		`${sizeClasses[size]} animate-spin rounded-full border-b-2 ${colorClasses[color]}`
	);
	let containerClasses = $derived(
		centered ? 'flex items-center justify-center py-12' : 'flex items-center'
	);
</script>

<div class="{containerClasses} {className}">
	<div class={spinnerClasses}></div>
	{#if text}
		<span class="ml-2 text-gray-700 dark:text-gray-300">{text}</span>
	{/if}
</div>
