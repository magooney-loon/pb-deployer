<script lang="ts" generics="T extends { id: string | number; [key: string]: unknown }">
	import LoadingSpinner from './LoadingSpinner.svelte';
	import EmptyState from './EmptyState.svelte';

	interface Column {
		key: string;
		label: string;
		sortable?: boolean;
		width?: string;
		align?: 'left' | 'center' | 'right';
		class?: string;
	}

	let {
		data = [],
		columns,
		loading = false,
		emptyState,
		striped = false,
		hoverable = true,
		compact = false,
		class: className = '',
		tableClass = '',
		headerClass = '',
		bodyClass = '',
		rowClass = '',
		cellClass = '',
		children,
		actions
	}: {
		data?: T[];
		columns: Column[];
		loading?: boolean;
		emptyState?: {
			icon?: string;
			title: string;
			description?: string;
			primaryAction?: {
				text: string;
				onclick?: () => void;
				href?: string;
			};
		};
		striped?: boolean;
		hoverable?: boolean;
		compact?: boolean;
		class?: string;
		tableClass?: string;
		headerClass?: string;
		bodyClass?: string;
		rowClass?: string;
		cellClass?: string;
		children?: import('svelte').Snippet<[item: T, index: number]>;
		actions?: import('svelte').Snippet<[item: T, index: number]>;
	} = $props();

	let hasData = $derived(data && data.length > 0);
	let showEmptyState = $derived(!loading && !hasData);

	// Table styling - Vercel-inspired
	let containerClasses = $derived(
		`overflow-hidden bg-white border border-gray-200 rounded-lg dark:bg-gray-950 dark:border-gray-800 ${className}`
	);

	let scrollClasses = 'overflow-x-auto';

	let baseTableClasses = 'min-w-full divide-y divide-gray-200 dark:divide-gray-800';

	let headerRowClasses = $derived(`bg-gray-50 dark:bg-gray-900 ${headerClass}`);

	let headerCellClasses =
		'px-6 py-3 text-left text-xs font-semibold tracking-wider text-gray-600 uppercase dark:text-gray-400';

	let bodyClasses = $derived(
		`divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-950 ${bodyClass}`
	);

	let getRowClasses = (index: number) => {
		let classes = '';

		if (hoverable) {
			classes += 'hover:bg-gray-50 dark:hover:bg-gray-900 ';
		}

		if (striped && index % 2 !== 0) {
			classes += 'bg-gray-50 dark:bg-gray-900 ';
		}

		return `${classes}${rowClass}`.trim();
	};

	let getCellClasses = (column: Column) => {
		let classes = compact ? 'px-4 py-2' : 'px-6 py-4';

		if (column.align === 'center') {
			classes += ' text-center';
		} else if (column.align === 'right') {
			classes += ' text-right';
		}

		classes += ' whitespace-nowrap';

		if (column.class) {
			classes += ` ${column.class}`;
		}

		return `${classes} ${cellClass}`.trim();
	};

	let getHeaderCellClasses = (column: Column) => {
		let classes = headerCellClasses;

		if (column.align === 'center') {
			classes += ' text-center';
		} else if (column.align === 'right') {
			classes += ' text-right';
		}

		if (column.class) {
			classes += ` ${column.class}`;
		}

		return classes;
	};
</script>

{#if loading}
	<div class={containerClasses}>
		<div class="p-12">
			<LoadingSpinner text="Loading data..." />
		</div>
	</div>
{:else if showEmptyState && emptyState}
	<div class={containerClasses}>
		<EmptyState
			icon={emptyState.icon}
			title={emptyState.title}
			description={emptyState.description}
			primaryAction={emptyState.primaryAction}
			class="py-12"
		/>
	</div>
{:else if hasData}
	<div class={containerClasses}>
		<div class={scrollClasses}>
			<table class="{baseTableClasses} {tableClass}">
				<thead class={headerRowClasses}>
					<tr>
						{#each columns as column (column.key)}
							<th
								class={getHeaderCellClasses(column)}
								style={column.width ? `width: ${column.width}` : undefined}
							>
								{column.label}
							</th>
						{/each}
						{#if actions}
							<th class="{headerCellClasses} text-right">Actions</th>
						{/if}
					</tr>
				</thead>
				<tbody class={bodyClasses}>
					{#each data as item, index (item.id || index)}
						<tr class={getRowClasses(index)}>
							{#if children}
								{@render children(item, index)}
								{#if actions}
									<td
										class="{getCellClasses({
											key: 'actions',
											label: '',
											align: 'right'
										})} space-x-2 text-sm font-medium"
									>
										{@render actions(item, index)}
									</td>
								{/if}
							{:else}
								{#each columns as column (column.key)}
									<td class={getCellClasses(column)}>
										{(item as Record<string, unknown>)[column.key] || '—'}
									</td>
								{/each}
								{#if actions}
									<td
										class="{getCellClasses({
											key: 'actions',
											label: '',
											align: 'right'
										})} space-x-2 text-sm font-medium"
									>
										{@render actions(item, index)}
									</td>
								{/if}
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>
{/if}
