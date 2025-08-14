<script lang="ts">
	let {
		status,
		variant = 'gray',
		size = 'sm',
		rounded = true,
		dot = false,
		customColors,
		class: className = ''
	}: {
		status: string;
		variant?: 'success' | 'warning' | 'error' | 'info' | 'gray' | 'custom';
		size?: 'xs' | 'sm' | 'md' | 'lg';
		rounded?: boolean;
		dot?: boolean;
		customColors?: {
			bg: string;
			text: string;
		};
		class?: string;
	} = $props();

	// Size variants
	const sizeVariants = {
		xs: 'px-1.5 py-0.5 text-xs',
		sm: 'px-2 py-1 text-xs',
		md: 'px-2.5 py-1.5 text-sm',
		lg: 'px-3 py-2 text-sm'
	};

	// Color variants - Vercel-style muted colors
	const colorVariants = {
		success:
			'bg-emerald-50 text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:ring-emerald-800',
		warning:
			'bg-amber-50 text-amber-700 ring-1 ring-amber-200 dark:bg-amber-950 dark:text-amber-300 dark:ring-amber-800',
		error:
			'bg-red-50 text-red-700 ring-1 ring-red-200 dark:bg-red-950 dark:text-red-300 dark:ring-red-800',
		info: 'bg-blue-50 text-blue-700 ring-1 ring-blue-200 dark:bg-blue-950 dark:text-blue-300 dark:ring-blue-800',
		gray: 'bg-gray-50 text-gray-600 ring-1 ring-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700',
		custom: customColors
			? `${customColors.bg} ${customColors.text}`
			: 'bg-gray-50 text-gray-600 ring-1 ring-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:ring-gray-700'
	};

	// Dot variants (for status indicators with dots) - Vercel-style
	const dotVariants = {
		success: 'bg-emerald-500',
		warning: 'bg-amber-500',
		error: 'bg-red-500',
		info: 'bg-blue-500',
		gray: 'bg-gray-400',
		custom: customColors?.bg || 'bg-gray-400'
	};

	let badgeClasses = $derived(
		[
			'inline-flex items-center font-medium',
			sizeVariants[size],
			colorVariants[variant],
			rounded ? 'rounded-full' : 'rounded',
			className
		]
			.filter(Boolean)
			.join(' ')
	);

	let dotClasses = $derived(['w-2 h-2 rounded-full mr-1.5', dotVariants[variant]].join(' '));
</script>

<span class={badgeClasses}>
	{#if dot}
		<span class={dotClasses}></span>
	{/if}
	{status}
</span>
