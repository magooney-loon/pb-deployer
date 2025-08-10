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

	// Shadow variants - Vercel-style subtle shadows
	const shadowVariants = {
		none: '',
		sm: 'shadow-sm',
		md: 'shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_1px_2px_0_rgba(0,0,0,0.05)] dark:shadow-[0_0_0_1px_rgba(255,255,255,0.05),0_1px_2px_0_rgba(0,0,0,0.3)]',
		lg: 'shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_4px_6px_-1px_rgba(0,0,0,0.1)] dark:shadow-[0_0_0_1px_rgba(255,255,255,0.05),0_4px_6px_-1px_rgba(0,0,0,0.4)]',
		xl: 'shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_10px_15px_-3px_rgba(0,0,0,0.1)] dark:shadow-[0_0_0_1px_rgba(255,255,255,0.05),0_10px_15px_-3px_rgba(0,0,0,0.4)]'
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

	// Base card styles - Vercel-inspired
	const baseStyles = 'bg-white dark:bg-gray-950 border border-gray-200 dark:border-gray-800';

	// Interactive styles - Vercel-style focus and interaction
	const interactiveStyles =
		clickable || href || onclick
			? 'transition-all duration-200 cursor-pointer focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 dark:focus:ring-gray-100'
			: '';

	// Hover styles - Subtle Vercel-style hover
	const hoverStyles = hover
		? 'hover:shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_8px_12px_-2px_rgba(0,0,0,0.1)] dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.05),0_8px_12px_-2px_rgba(0,0,0,0.4)] hover:border-gray-300 dark:hover:border-gray-700'
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
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h3>
				{/if}
				{#if subtitle}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
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
					<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h3>
				{/if}
				{#if subtitle}
					<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{subtitle}</p>
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
