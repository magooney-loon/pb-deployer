<script lang="ts">
	let {
		value = 0,
		max = 100,
		label,
		showPercentage = true,
		color = 'blue',
		size = 'md',
		animated = true,
		striped = false,
		class: className = ''
	}: {
		value?: number;
		max?: number;
		label?: string;
		showPercentage?: boolean;
		color?: 'blue' | 'green' | 'yellow' | 'red' | 'gray';
		size?: 'sm' | 'md' | 'lg';
		animated?: boolean;
		striped?: boolean;
		class?: string;
	} = $props();

	let percentage = $derived(Math.min(Math.max((value / max) * 100, 0), 100));

	const sizeStyles = {
		sm: 'h-1',
		md: 'h-2',
		lg: 'h-3'
	};

	const colorStyles = {
		blue: 'bg-blue-600 dark:bg-blue-500',
		green: 'bg-green-600 dark:bg-green-500',
		yellow: 'bg-yellow-500 dark:bg-yellow-400',
		red: 'bg-red-600 dark:bg-red-500',
		gray: 'bg-gray-600 dark:bg-gray-400'
	};

	const backgroundStyles = 'bg-gray-200 dark:bg-gray-700';

	const animationStyles = animated ? 'transition-all duration-300 ease-out' : '';

	const stripedStyles = striped
		? 'bg-gradient-to-r from-transparent via-white/20 to-transparent bg-[length:1rem_1rem] animate-pulse'
		: '';

	let barClasses = $derived(
		`${sizeStyles[size]} ${colorStyles[color]} ${animationStyles} ${stripedStyles} rounded-full`
	);

	let containerClasses = $derived(
		`w-full ${sizeStyles[size]} ${backgroundStyles} rounded-full overflow-hidden`
	);
</script>

<div class="space-y-2 {className}">
	{#if label || showPercentage}
		<div class="flex items-center justify-between text-sm">
			{#if label}
				<span class="font-medium text-gray-900 dark:text-gray-100">{label}</span>
			{/if}
			{#if showPercentage}
				<span class="text-gray-600 dark:text-gray-400">
					{Math.round(percentage)}%
				</span>
			{/if}
		</div>
	{/if}

	<div class={containerClasses}>
		<div
			class={barClasses}
			style="width: {percentage}%"
			role="progressbar"
			aria-valuenow={value}
			aria-valuemin="0"
			aria-valuemax={max}
			aria-label={label || `Progress: ${Math.round(percentage)}%`}
		></div>
	</div>
</div>
