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

	// Size variants
	const sizeClasses = {
		sm: 'h-4 w-4',
		md: 'h-8 w-8',
		lg: 'h-12 w-12'
	};

	// Color variants
	const colorClasses = {
		blue: 'border-blue-600',
		gray: 'border-gray-600',
		green: 'border-green-600',
		red: 'border-red-600',
		yellow: 'border-yellow-600'
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
		<span class="ml-2 text-gray-600 dark:text-gray-400">{text}</span>
	{/if}
</div>
