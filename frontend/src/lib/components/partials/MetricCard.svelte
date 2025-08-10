<script lang="ts">
	let {
		title,
		value,
		icon,
		color = 'default',
		size = 'md',
		href,
		onclick,
		class: className = ''
	}: {
		title: string;
		value: string | number;
		icon?: string;
		color?: 'default' | 'blue' | 'green' | 'red' | 'yellow' | 'purple';
		size?: 'sm' | 'md' | 'lg';
		href?: string;
		onclick?: () => void;
		class?: string;
	} = $props();

	// Color variants for accent colors
	const colorVariants = {
		default: {
			value: 'text-gray-900 dark:text-white',
			icon: 'text-gray-600 dark:text-gray-400'
		},
		blue: {
			value: 'text-blue-600 dark:text-blue-400',
			icon: 'text-blue-500 dark:text-blue-400'
		},
		green: {
			value: 'text-green-600 dark:text-green-400',
			icon: 'text-green-500 dark:text-green-400'
		},
		red: {
			value: 'text-red-600 dark:text-red-400',
			icon: 'text-red-500 dark:text-red-400'
		},
		yellow: {
			value: 'text-yellow-600 dark:text-yellow-400',
			icon: 'text-yellow-500 dark:text-yellow-400'
		},
		purple: {
			value: 'text-purple-600 dark:text-purple-400',
			icon: 'text-purple-500 dark:text-purple-400'
		}
	};

	// Size variants
	const sizeVariants = {
		sm: {
			padding: 'p-3',
			iconSize: 'text-lg',
			valueSize: 'text-base',
			spacing: 'ml-3'
		},
		md: {
			padding: 'p-5',
			iconSize: 'text-2xl',
			valueSize: 'text-lg',
			spacing: 'ml-5'
		},
		lg: {
			padding: 'p-6',
			iconSize: 'text-3xl',
			valueSize: 'text-xl',
			spacing: 'ml-6'
		}
	};

	let colors = $derived(colorVariants[color]);
	let sizes = $derived(sizeVariants[size]);
	let isClickable = $derived(!!(href || onclick));
	let cardClasses = $derived(
		`overflow-hidden rounded-lg bg-white shadow dark:bg-gray-800 dark:shadow-gray-700 ${isClickable ? 'transition-all duration-200 hover:shadow-lg dark:hover:shadow-gray-600 cursor-pointer' : ''} ${className}`
	);

	function handleClick() {
		if (onclick) {
			onclick();
		}
	}
</script>

{#if href}
	<a {href} class={cardClasses}>
		<div class={sizes.padding}>
			<div class="flex items-center">
				{#if icon}
					<div class="flex-shrink-0">
						<span class="{sizes.iconSize} {colors.icon}">{icon}</span>
					</div>
				{/if}
				<div class="{icon ? sizes.spacing : ''} w-0 flex-1">
					<dl>
						<dt class="truncate text-sm font-medium text-gray-500 dark:text-gray-400">
							{title}
						</dt>
						<dd class="{sizes.valueSize} font-medium {colors.value}">
							{value}
						</dd>
					</dl>
				</div>
			</div>
		</div>
	</a>
{:else}
	<div
		class={cardClasses}
		onclick={isClickable ? handleClick : undefined}
		role={isClickable ? 'button' : undefined}
		{...isClickable ? { tabindex: 0 } : {}}
	>
		<div class={sizes.padding}>
			<div class="flex items-center">
				{#if icon}
					<div class="flex-shrink-0">
						<span class="{sizes.iconSize} {colors.icon}">{icon}</span>
					</div>
				{/if}
				<div class="{icon ? sizes.spacing : ''} w-0 flex-1">
					<dl>
						<dt class="truncate text-sm font-medium text-gray-500 dark:text-gray-400">
							{title}
						</dt>
						<dd class="{sizes.valueSize} font-medium {colors.value}">
							{value}
						</dd>
					</dl>
				</div>
			</div>
		</div>
	</div>
{/if}
