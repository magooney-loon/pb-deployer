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

	const colorVariants = {
		default: {
			value: 'text-gray-900 dark:text-gray-100',
			icon: 'text-gray-600 dark:text-gray-400'
		},
		blue: {
			value: 'text-gray-900 dark:text-gray-100',
			icon: 'text-gray-600 dark:text-gray-400'
		},
		green: {
			value: 'text-emerald-700 dark:text-emerald-300',
			icon: 'text-emerald-600 dark:text-emerald-400'
		},
		red: {
			value: 'text-red-700 dark:text-red-300',
			icon: 'text-red-600 dark:text-red-400'
		},
		yellow: {
			value: 'text-amber-700 dark:text-amber-300',
			icon: 'text-amber-600 dark:text-amber-400'
		},
		purple: {
			value: 'text-violet-700 dark:text-violet-300',
			icon: 'text-violet-600 dark:text-violet-400'
		}
	};

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
		`overflow-hidden rounded-lg bg-white border border-gray-200 shadow-sm dark:bg-gray-950 dark:border-gray-800 ${isClickable ? 'transition-all duration-200 hover:shadow-[0_0_0_1px_rgba(0,0,0,0.05),0_4px_6px_-1px_rgba(0,0,0,0.1)] hover:border-gray-300 dark:hover:shadow-[0_0_0_1px_rgba(255,255,255,0.05),0_4px_6px_-1px_rgba(0,0,0,0.4)] dark:hover:border-gray-700 cursor-pointer' : ''} ${className}`
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
						<dt class="truncate text-sm font-medium text-gray-600 dark:text-gray-400">
							{title}
						</dt>
						<dd class="{sizes.valueSize} font-semibold {colors.value}">
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
						<dt class="truncate text-sm font-medium text-gray-600 dark:text-gray-400">
							{title}
						</dt>
						<dd class="{sizes.valueSize} font-semibold {colors.value}">
							{value}
						</dd>
					</dl>
				</div>
			</div>
		</div>
	</div>
{/if}
