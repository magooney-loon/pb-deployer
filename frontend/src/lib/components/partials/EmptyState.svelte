<script lang="ts">
	import Button from './Button.svelte';

	let {
		icon,
		iconSnippet,
		title,
		description,
		primaryAction,
		secondaryText,
		size = 'md',
		class: className = ''
	}: {
		icon?: string;
		iconSnippet?: import('svelte').Snippet;
		title: string;
		description?: string;
		primaryAction?: {
			text: string;
			onclick?: () => void;
			href?: string;
			variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'link';
			color?: 'blue' | 'green' | 'red' | 'yellow' | 'gray' | 'white' | 'purple';
		};
		secondaryText?: string;
		size?: 'sm' | 'md' | 'lg';
		class?: string;
	} = $props();

	const sizeVariants = {
		sm: {
			container: 'py-8',
			icon: 'text-3xl mb-3',
			title: 'text-base',
			description: 'text-sm',
			spacing: 'space-y-2'
		},
		md: {
			container: 'py-12',
			icon: 'text-4xl mb-4',
			title: 'text-lg',
			description: 'text-base',
			spacing: 'space-y-3'
		},
		lg: {
			container: 'py-16',
			icon: 'text-5xl mb-6',
			title: 'text-xl',
			description: 'text-lg',
			spacing: 'space-y-4'
		}
	};

	let sizeConfig = $derived(sizeVariants[size]);
</script>

<div class="text-center {sizeConfig.container} {className}">
	{#if icon || iconSnippet}
		<div class="flex justify-center text-gray-500 dark:text-gray-400 {sizeConfig.icon}">
			{#if iconSnippet}
				{@render iconSnippet()}
			{:else}
				{icon}
			{/if}
		</div>
	{/if}

	<div class={sizeConfig.spacing}>
		<h3 class="font-semibold text-gray-900 dark:text-gray-100 {sizeConfig.title}">
			{title}
		</h3>

		{#if description}
			<p class="text-gray-600 dark:text-gray-400 {sizeConfig.description}">
				{description}
			</p>
		{/if}

		{#if primaryAction}
			<div class="mt-4">
				<Button
					variant={primaryAction.variant || 'primary'}
					color={primaryAction.color || 'blue'}
					size={size === 'sm' ? 'md' : 'lg'}
					href={primaryAction.href}
					onclick={primaryAction.onclick}
				>
					{primaryAction.text}
				</Button>
			</div>
		{/if}

		{#if secondaryText}
			<p class="mt-3 text-xs text-gray-500 dark:text-gray-500">
				{secondaryText}
			</p>
		{/if}
	</div>
</div>
