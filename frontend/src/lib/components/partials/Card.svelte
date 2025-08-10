<script lang="ts">
	let {
		title,
		subtitle,
		padding = 'md',
		shadow = 'md',
		rounded = 'lg',
		hover = false,
		clickable = false,
		href,
		target,
		onclick,
		class: className = '',
		headerClass = '',
		bodyClass = '',
		children
	}: {
		title?: string;
		subtitle?: string;
		padding?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
		shadow?: 'none' | 'sm' | 'md' | 'lg' | 'xl';
		rounded?: 'none' | 'sm' | 'md' | 'lg' | 'xl' | 'full';
		hover?: boolean;
		clickable?: boolean;
		href?: string;
		target?: string;
		onclick?: () => void;
		class?: string;
		headerClass?: string;
		bodyClass?: string;
		children?: import('svelte').Snippet;
	} = $props();

	// Padding variants
	const paddingVariants = {
		none: '',
		sm: 'p-3',
		md: 'p-4 sm:p-6',
		lg: 'p-6 sm:p-8',
		xl: 'p-8 sm:p-10'
	};

	// Shadow variants
	const shadowVariants = {
		none: '',
		sm: 'shadow-sm dark:shadow-gray-800',
		md: 'shadow dark:shadow-gray-700',
		lg: 'shadow-lg dark:shadow-gray-600',
		xl: 'shadow-xl dark:shadow-gray-500'
	};

	// Rounded variants
	const roundedVariants = {
		none: '',
		sm: 'rounded-sm',
		md: 'rounded-md',
		lg: 'rounded-lg',
		xl: 'rounded-xl',
		full: 'rounded-full'
	};

	// Base card styles
	const baseStyles = 'bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700';

	// Interactive styles
	const interactiveStyles =
		clickable || href || onclick
			? 'transition-all duration-200 cursor-pointer focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2'
			: '';

	// Hover styles
	const hoverStyles = hover
		? 'hover:shadow-lg dark:hover:shadow-gray-600 hover:-translate-y-0.5'
		: '';

	let cardClasses = $derived(
		[
			baseStyles,
			paddingVariants[padding],
			shadowVariants[shadow],
			roundedVariants[rounded],
			interactiveStyles,
			hoverStyles,
			className
		]
			.filter(Boolean)
			.join(' ')
	);

	let hasHeader = $derived(title || subtitle);
	let isInteractive = $derived(!!(href || onclick || clickable));

	function handleClick() {
		if (onclick) {
			onclick();
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (isInteractive && (event.key === 'Enter' || event.key === ' ')) {
			event.preventDefault();
			handleClick();
		}
	}
</script>

{#if href}
	<a {href} {target} class={cardClasses} tabindex="0" onkeydown={handleKeydown}>
		{#if hasHeader}
			<div class="mb-4 {headerClass}">
				{#if title}
					<h3 class="text-lg font-medium text-gray-900 dark:text-white">{title}</h3>
				{/if}
				{#if subtitle}
					<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{subtitle}</p>
				{/if}
			</div>
		{/if}

		<div class={bodyClass}>
			{#if children}
				{@render children()}
			{/if}
		</div>
	</a>
{:else}
	<div
		class={cardClasses}
		role={isInteractive ? 'button' : undefined}
		{...isInteractive ? { tabindex: 0 } : {}}
		onclick={isInteractive ? handleClick : undefined}
		onkeydown={isInteractive ? handleKeydown : undefined}
	>
		{#if hasHeader}
			<div class="mb-4 {headerClass}">
				{#if title}
					<h3 class="text-lg font-medium text-gray-900 dark:text-white">{title}</h3>
				{/if}
				{#if subtitle}
					<p class="mt-1 text-sm text-gray-600 dark:text-gray-400">{subtitle}</p>
				{/if}
			</div>
		{/if}

		<div class={bodyClass}>
			{#if children}
				{@render children()}
			{/if}
		</div>
	</div>
{/if}
