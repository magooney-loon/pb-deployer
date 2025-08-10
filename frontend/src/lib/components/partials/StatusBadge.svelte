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

	// Color variants
	const colorVariants = {
		success: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
		warning: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
		error: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
		info: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
		gray: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200',
		custom: customColors
			? `${customColors.bg} ${customColors.text}`
			: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
	};

	// Dot variants (for status indicators with dots)
	const dotVariants = {
		success: 'bg-green-400',
		warning: 'bg-yellow-400',
		error: 'bg-red-400',
		info: 'bg-blue-400',
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
