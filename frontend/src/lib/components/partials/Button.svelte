<script lang="ts">
	let {
		variant = 'primary',
		color = 'blue',
		size = 'md',
		disabled = false,
		loading = false,
		href,
		target,
		icon,
		iconPosition = 'left',
		fullWidth = false,
		onclick,
		type = 'button',
		class: className = '',
		children
	}: {
		variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'link';
		color?: 'blue' | 'green' | 'red' | 'yellow' | 'gray' | 'white' | 'purple';
		size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
		disabled?: boolean;
		loading?: boolean;
		href?: string;
		target?: string;
		icon?: string;
		iconPosition?: 'left' | 'right';
		fullWidth?: boolean;
		onclick?: () => void;
		type?: 'button' | 'submit' | 'reset';
		class?: string;
		children?: import('svelte').Snippet;
	} = $props();

	// Base styles that apply to all buttons
	const baseStyles =
		'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed border';

	// Size variants
	const sizeVariants = {
		xs: 'px-2 py-1 text-xs',
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-sm',
		lg: 'px-6 py-3 text-base',
		xl: 'px-8 py-4 text-lg'
	};

	// Color and variant combinations - Vercel-inspired
	const variantStyles = {
		primary: {
			blue: 'border-black bg-black text-white hover:bg-gray-800 hover:border-gray-800 focus:ring-gray-500 disabled:hover:bg-black shadow-sm',
			green:
				'border-emerald-600 bg-emerald-600 text-white hover:bg-emerald-700 hover:border-emerald-700 focus:ring-emerald-500 disabled:hover:bg-emerald-600 shadow-sm',
			red: 'border-red-500 bg-red-500 text-white hover:bg-red-600 hover:border-red-600 focus:ring-red-500 disabled:hover:bg-red-500 shadow-sm',
			yellow:
				'border-amber-500 bg-amber-500 text-white hover:bg-amber-600 hover:border-amber-600 focus:ring-amber-500 disabled:hover:bg-amber-500 shadow-sm',
			gray: 'border-gray-600 bg-gray-600 text-white hover:bg-gray-700 hover:border-gray-700 focus:ring-gray-500 disabled:hover:bg-gray-600 shadow-sm',
			white:
				'border-gray-200 bg-white text-gray-900 hover:bg-gray-50 hover:border-gray-300 focus:ring-gray-500 disabled:hover:bg-white shadow-sm dark:border-gray-800 dark:bg-gray-950 dark:text-white dark:hover:bg-gray-900',
			purple:
				'border-violet-600 bg-violet-600 text-white hover:bg-violet-700 hover:border-violet-700 focus:ring-violet-500 disabled:hover:bg-violet-600 shadow-sm'
		},
		secondary: {
			blue: 'border-gray-200 bg-gray-50 text-gray-900 hover:bg-gray-100 hover:border-gray-300 focus:ring-gray-500 shadow-sm dark:border-gray-800 dark:bg-gray-900 dark:text-gray-100 dark:hover:bg-gray-800',
			green:
				'border-emerald-200 bg-emerald-50 text-emerald-700 hover:bg-emerald-100 hover:border-emerald-300 focus:ring-emerald-500 shadow-sm dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-100 dark:hover:bg-emerald-900',
			red: 'border-red-200 bg-red-50 text-red-700 hover:bg-red-100 hover:border-red-300 focus:ring-red-500 shadow-sm dark:border-red-800 dark:bg-red-950 dark:text-red-100 dark:hover:bg-red-900',
			yellow:
				'border-amber-200 bg-amber-50 text-amber-700 hover:bg-amber-100 hover:border-amber-300 focus:ring-amber-500 shadow-sm dark:border-amber-800 dark:bg-amber-950 dark:text-amber-100 dark:hover:bg-amber-900',
			gray: 'border-gray-200 bg-gray-50 text-gray-700 hover:bg-gray-100 hover:border-gray-300 focus:ring-gray-500 shadow-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700',
			white:
				'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 hover:border-gray-300 focus:ring-gray-500 shadow-sm dark:border-gray-800 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-900',
			purple:
				'border-violet-200 bg-violet-50 text-violet-700 hover:bg-violet-100 hover:border-violet-300 focus:ring-violet-500 shadow-sm dark:border-violet-800 dark:bg-violet-950 dark:text-violet-100 dark:hover:bg-violet-900'
		},
		outline: {
			blue: 'border-gray-300 bg-transparent text-gray-700 hover:bg-gray-50 hover:border-gray-400 focus:ring-gray-500 shadow-sm dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-900',
			green:
				'border-emerald-300 bg-transparent text-emerald-700 hover:bg-emerald-50 hover:border-emerald-400 focus:ring-emerald-500 shadow-sm dark:border-emerald-700 dark:text-emerald-300 dark:hover:bg-emerald-950',
			red: 'border-red-300 bg-transparent text-red-700 hover:bg-red-50 hover:border-red-400 focus:ring-red-500 shadow-sm dark:border-red-700 dark:text-red-300 dark:hover:bg-red-950',
			yellow:
				'border-amber-300 bg-transparent text-amber-700 hover:bg-amber-50 hover:border-amber-400 focus:ring-amber-500 shadow-sm dark:border-amber-700 dark:text-amber-300 dark:hover:bg-amber-950',
			gray: 'border-gray-300 bg-transparent text-gray-700 hover:bg-gray-50 hover:border-gray-400 focus:ring-gray-500 shadow-sm dark:border-gray-600 dark:text-gray-300 dark:hover:bg-gray-900',
			white:
				'border-gray-300 bg-transparent text-gray-700 hover:bg-gray-50 hover:border-gray-400 focus:ring-gray-500 shadow-sm dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-900',
			purple:
				'border-violet-300 bg-transparent text-violet-700 hover:bg-violet-50 hover:border-violet-400 focus:ring-violet-500 shadow-sm dark:border-violet-700 dark:text-violet-300 dark:hover:bg-violet-950'
		},
		ghost: {
			blue: 'border-transparent bg-transparent text-gray-700 hover:bg-gray-100 focus:ring-gray-500 dark:text-gray-300 dark:hover:bg-gray-800',
			green:
				'border-transparent bg-transparent text-emerald-700 hover:bg-emerald-100 focus:ring-emerald-500 dark:text-emerald-300 dark:hover:bg-emerald-950',
			red: 'border-transparent bg-transparent text-red-700 hover:bg-red-100 focus:ring-red-500 dark:text-red-300 dark:hover:bg-red-950',
			yellow:
				'border-transparent bg-transparent text-amber-700 hover:bg-amber-100 focus:ring-amber-500 dark:text-amber-300 dark:hover:bg-amber-950',
			gray: 'border-transparent bg-transparent text-gray-700 hover:bg-gray-100 focus:ring-gray-500 dark:text-gray-300 dark:hover:bg-gray-800',
			white:
				'border-transparent bg-transparent text-gray-700 hover:bg-gray-100 focus:ring-gray-500 dark:text-gray-200 dark:hover:bg-gray-800',
			purple:
				'border-transparent bg-transparent text-violet-700 hover:bg-violet-100 focus:ring-violet-500 dark:text-violet-300 dark:hover:bg-violet-950'
		},
		link: {
			blue: 'border-transparent bg-transparent text-gray-700 hover:text-gray-900 focus:ring-gray-500 underline-offset-4 hover:underline dark:text-gray-300 dark:hover:text-gray-100',
			green:
				'border-transparent bg-transparent text-emerald-700 hover:text-emerald-900 focus:ring-emerald-500 underline-offset-4 hover:underline dark:text-emerald-300 dark:hover:text-emerald-100',
			red: 'border-transparent bg-transparent text-red-700 hover:text-red-900 focus:ring-red-500 underline-offset-4 hover:underline dark:text-red-300 dark:hover:text-red-100',
			yellow:
				'border-transparent bg-transparent text-amber-700 hover:text-amber-900 focus:ring-amber-500 underline-offset-4 hover:underline dark:text-amber-300 dark:hover:text-amber-100',
			gray: 'border-transparent bg-transparent text-gray-700 hover:text-gray-900 focus:ring-gray-500 underline-offset-4 hover:underline dark:text-gray-300 dark:hover:text-gray-100',
			white:
				'border-transparent bg-transparent text-gray-700 hover:text-gray-900 focus:ring-gray-500 underline-offset-4 hover:underline dark:text-gray-200 dark:hover:text-white',
			purple:
				'border-transparent bg-transparent text-violet-700 hover:text-violet-900 focus:ring-violet-500 underline-offset-4 hover:underline dark:text-violet-300 dark:hover:text-violet-100'
		}
	};

	// Icon spacing based on size
	const iconSpacing = {
		xs: { left: 'mr-1', right: 'ml-1' },
		sm: { left: 'mr-1.5', right: 'ml-1.5' },
		md: { left: 'mr-2', right: 'ml-2' },
		lg: { left: 'mr-2.5', right: 'ml-2.5' },
		xl: { left: 'mr-3', right: 'ml-3' }
	};

	let buttonClasses = $derived(
		[
			baseStyles,
			sizeVariants[size],
			variantStyles[variant][color],
			fullWidth ? 'w-full' : '',
			className
		]
			.filter(Boolean)
			.join(' ')
	);

	let isDisabled = $derived(disabled || loading);

	function handleClick() {
		if (!isDisabled && onclick) {
			onclick();
		}
	}
</script>

{#if href}
	<a
		{href}
		{target}
		class={buttonClasses}
		class:opacity-50={isDisabled}
		class:cursor-not-allowed={isDisabled}
		class:pointer-events-none={isDisabled}
	>
		{#if loading}
			<div class="h-4 w-4 animate-spin rounded-full border-b-2 border-current"></div>
		{:else if icon && iconPosition === 'left'}
			<span class={iconSpacing[size].left}>{icon}</span>
		{/if}

		{#if children}
			{@render children()}
		{/if}

		{#if !loading && icon && iconPosition === 'right'}
			<span class={iconSpacing[size].right}>{icon}</span>
		{/if}
	</a>
{:else}
	<button {type} class={buttonClasses} disabled={isDisabled} onclick={handleClick}>
		{#if loading}
			<div class="h-4 w-4 animate-spin rounded-full border-b-2 border-current"></div>
		{:else if icon && iconPosition === 'left'}
			<span class={iconSpacing[size].left}>{icon}</span>
		{/if}

		{#if children}
			{@render children()}
		{/if}

		{#if !loading && icon && iconPosition === 'right'}
			<span class={iconSpacing[size].right}>{icon}</span>
		{/if}
	</button>
{/if}
