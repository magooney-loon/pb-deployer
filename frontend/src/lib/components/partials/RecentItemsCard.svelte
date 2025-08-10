<script lang="ts" generics="T extends { id: string | number }">
	import Card from './Card.svelte';

	interface EmptyState {
		message: string;
		ctaText?: string;
		ctaHref?: string;
		secondaryText?: string;
	}

	let {
		title,
		items,
		viewAllHref,
		viewAllText = 'View all →',
		emptyState,
		itemClass = 'flex items-center justify-between rounded-lg bg-gray-50 p-3 dark:bg-gray-700',
		class: className = '',
		children
	}: {
		title: string;
		items: T[];
		viewAllHref?: string;
		viewAllText?: string;
		emptyState: EmptyState;
		itemClass?: string;
		class?: string;
		children?: import('svelte').Snippet<[T, number]>;
	} = $props();

	let hasItems = $derived(items && items.length > 0);
</script>

<Card class={className}>
	<div class="mb-4 flex items-center justify-between">
		<h3 class="text-lg font-medium text-gray-900 dark:text-white">{title}</h3>
		{#if viewAllHref}
			<a
				href={viewAllHref}
				class="text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
			>
				{viewAllText}
			</a>
		{/if}
	</div>

	{#if !hasItems}
		<div class="py-6 text-center">
			<p class="text-gray-500 dark:text-gray-400">{emptyState.message}</p>
			{#if emptyState.ctaText && emptyState.ctaHref}
				<a
					href={emptyState.ctaHref}
					class="mt-2 inline-flex items-center text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
				>
					{emptyState.ctaText}
				</a>
			{/if}
			{#if emptyState.secondaryText}
				<p class="mt-2 text-xs text-gray-400 dark:text-gray-500">{emptyState.secondaryText}</p>
			{/if}
		</div>
	{:else}
		<div class="space-y-3">
			{#each items as item, index (item.id || index)}
				<div class={itemClass}>
					{#if children}
						{@render children(item, index)}
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</Card>
