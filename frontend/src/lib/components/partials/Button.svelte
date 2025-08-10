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
		'inline-flex items-center justify-center font-medium rounded-md transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed';

	// Size variants
	const sizeVariants = {
		xs: 'px-2 py-1 text-xs',
		sm: 'px-3 py-1.5 text-sm',
		md: 'px-4 py-2 text-sm',
		lg: 'px-6 py-3 text-base',
		xl: 'px-8 py-4 text-lg'
	};

	// Color and variant combinations
	const variantStyles = {
		primary: {
			blue: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 disabled:hover:bg-blue-600',
			green:
				'bg-green-600 text-white hover:bg-green-700 focus:ring-green-500 disabled:hover:bg-green-600',
			red: 'bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 disabled:hover:bg-red-600',
			yellow:
				'bg-yellow-600 text-white hover:bg-yellow-700 focus:ring-yellow-500 disabled:hover:bg-yellow-600',
			gray: 'bg-gray-600 text-white hover:bg-gray-700 focus:ring-gray-500 disabled:hover:bg-gray-600',
			white:
				'bg-white text-gray-900 hover:bg-gray-50 focus:ring-gray-500 border border-gray-300 disabled:hover:bg-white',
			purple:
				'bg-purple-600 text-white hover:bg-purple-700 focus:ring-purple-500 disabled:hover:bg-purple-600'
		},
		secondary: {
			blue: 'bg-blue-100 text-blue-700 hover:bg-blue-200 focus:ring-blue-500 dark:bg-blue-900 dark:text-blue-200 dark:hover:bg-blue-800',
			green:
				'bg-green-100 text-green-700 hover:bg-green-200 focus:ring-green-500 dark:bg-green-900 dark:text-green-200 dark:hover:bg-green-800',
			red: 'bg-red-100 text-red-700 hover:bg-red-200 focus:ring-red-500 dark:bg-red-900 dark:text-red-200 dark:hover:bg-red-800',
			yellow:
				'bg-yellow-100 text-yellow-700 hover:bg-yellow-200 focus:ring-yellow-500 dark:bg-yellow-900 dark:text-yellow-200 dark:hover:bg-yellow-800',
			gray: 'bg-gray-100 text-gray-700 hover:bg-gray-200 focus:ring-gray-500 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600',
			white:
				'bg-white text-gray-700 hover:bg-gray-50 focus:ring-gray-500 border border-gray-300 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700 dark:border-gray-600',
			purple:
				'bg-purple-100 text-purple-700 hover:bg-purple-200 focus:ring-purple-500 dark:bg-purple-900 dark:text-purple-200 dark:hover:bg-purple-800'
		},
		outline: {
			blue: 'border border-blue-600 text-blue-600 hover:bg-blue-50 focus:ring-blue-500 dark:text-blue-400 dark:border-blue-400 dark:hover:bg-blue-900',
			green:
				'border border-green-600 text-green-600 hover:bg-green-50 focus:ring-green-500 dark:text-green-400 dark:border-green-400 dark:hover:bg-green-900',
			red: 'border border-red-600 text-red-600 hover:bg-red-50 focus:ring-red-500 dark:text-red-400 dark:border-red-400 dark:hover:bg-red-900',
			yellow:
				'border border-yellow-600 text-yellow-600 hover:bg-yellow-50 focus:ring-yellow-500 dark:text-yellow-400 dark:border-yellow-400 dark:hover:bg-yellow-900',
			gray: 'border border-gray-600 text-gray-600 hover:bg-gray-50 focus:ring-gray-500 dark:text-gray-400 dark:border-gray-400 dark:hover:bg-gray-700',
			white:
				'border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-gray-500 dark:border-gray-600 dark:text-gray-200 dark:hover:bg-gray-700',
			purple:
				'border border-purple-600 text-purple-600 hover:bg-purple-50 focus:ring-purple-500 dark:text-purple-400 dark:border-purple-400 dark:hover:bg-purple-900'
		},
		ghost: {
			blue: 'text-blue-600 hover:bg-blue-50 focus:ring-blue-500 dark:text-blue-400 dark:hover:bg-blue-900',
			green:
				'text-green-600 hover:bg-green-50 focus:ring-green-500 dark:text-green-400 dark:hover:bg-green-900',
			red: 'text-red-600 hover:bg-red-50 focus:ring-red-500 dark:text-red-400 dark:hover:bg-red-900',
			yellow:
				'text-yellow-600 hover:bg-yellow-50 focus:ring-yellow-500 dark:text-yellow-400 dark:hover:bg-yellow-900',
			gray: 'text-gray-600 hover:bg-gray-50 focus:ring-gray-500 dark:text-gray-400 dark:hover:bg-gray-700',
			white:
				'text-gray-700 hover:bg-gray-50 focus:ring-gray-500 dark:text-gray-200 dark:hover:bg-gray-700',
			purple:
				'text-purple-600 hover:bg-purple-50 focus:ring-purple-500 dark:text-purple-400 dark:hover:bg-purple-900'
		},
		link: {
			blue: 'text-blue-600 hover:text-blue-800 focus:ring-blue-500 dark:text-blue-400 dark:hover:text-blue-300',
			green:
				'text-green-600 hover:text-green-800 focus:ring-green-500 dark:text-green-400 dark:hover:text-green-300',
			red: 'text-red-600 hover:text-red-800 focus:ring-red-500 dark:text-red-400 dark:hover:text-red-300',
			yellow:
				'text-yellow-600 hover:text-yellow-800 focus:ring-yellow-500 dark:text-yellow-400 dark:hover:text-yellow-300',
			gray: 'text-gray-600 hover:text-gray-800 focus:ring-gray-500 dark:text-gray-400 dark:hover:text-gray-200',
			white:
				'text-gray-700 hover:text-gray-900 focus:ring-gray-500 dark:text-gray-200 dark:hover:text-white',
			purple:
				'text-purple-600 hover:text-purple-800 focus:ring-purple-500 dark:text-purple-400 dark:hover:text-purple-300'
		}
	};

	// Icon spacing based on size
	const iconSpacing = {
		xs: 'space-x-1',
		sm: 'space-x-1.5',
		md: 'space-x-2',
		lg: 'space-x-2.5',
		xl: 'space-x-3'
	};

	let buttonClasses = $derived(
		[
			baseStyles,
			sizeVariants[size],
			variantStyles[variant][color],
			fullWidth ? 'w-full' : '',
			icon ? iconSpacing[size] : '',
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
			<span>{icon}</span>
		{/if}

		{#if children}
			{@render children()}
		{/if}

		{#if !loading && icon && iconPosition === 'right'}
			<span>{icon}</span>
		{/if}
	</a>
{:else}
	<button {type} class={buttonClasses} disabled={isDisabled} onclick={handleClick}>
		{#if loading}
			<div class="h-4 w-4 animate-spin rounded-full border-b-2 border-current"></div>
		{:else if icon && iconPosition === 'left'}
			<span>{icon}</span>
		{/if}

		{#if children}
			{@render children()}
		{/if}

		{#if !loading && icon && iconPosition === 'right'}
			<span>{icon}</span>
		{/if}
	</button>
{/if}
