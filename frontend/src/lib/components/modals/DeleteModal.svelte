<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';

	interface Props {
		open?: boolean;
		item?: {
			id: string;
			name: string;
			host?: string;
			port?: number;
			setup_complete?: boolean;
			security_locked?: boolean;
			domain?: string;
			status?: string;
		} | null;
		itemType?: string;
		itemDisplayName?: string;
		loading?: boolean;
		relatedItems?: { id: string; name: string; domain?: string }[];
		relatedItemsType?: string;
		onclose?: () => void;
		onconfirm?: (itemId: string) => void;
	}

	let {
		open = false,
		item = null,
		itemType = 'item',
		itemDisplayName = '',
		loading = false,
		relatedItems = [],
		relatedItemsType = 'items',
		onclose,
		onconfirm
	}: Props = $props();

	let confirmationText = $state('');

	// Reset confirmation text when modal opens/closes or item changes
	$effect(() => {
		if (!open || !item) {
			confirmationText = '';
		}
	});

	function handleClose() {
		if (!loading) {
			confirmationText = '';
			onclose?.();
		}
	}

	function handleConfirm() {
		if (item && confirmationText === item.name && !loading) {
			onconfirm?.(item.id);
		}
	}

	// Get display name for the item
	let displayName = $derived(itemDisplayName || item?.name || 'Unknown');
</script>

<Modal
	{open}
	title="Delete {itemType.charAt(0).toUpperCase() + itemType.slice(1)}"
	size="md"
	closeable={!loading}
	onclose={handleClose}
>
	{#if item !== null}
		<div class="space-y-6">
			<!-- Warning -->
			<div class="rounded-lg bg-red-50 p-4 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800">
				<div class="flex">
					<div class="flex-shrink-0">
						<svg class="h-5 w-5 text-red-500" viewBox="0 0 20 20" fill="currentColor">
							<path
								fill-rule="evenodd"
								d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
								clip-rule="evenodd"
							/>
						</svg>
					</div>
					<div class="ml-3">
						<h3 class="text-sm font-semibold text-red-700 dark:text-red-300">
							This action cannot be undone
						</h3>
						<div class="mt-2 text-sm text-red-600 dark:text-red-400">
							<p>
								This will permanently delete the {itemType} configuration.
								{#if itemType === 'server'}
									The actual VPS server will not be affected.
								{/if}
							</p>
						</div>
					</div>
				</div>
			</div>

			<!-- Item Details -->
			<div
				class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
			>
				<h4 class="mb-3 font-semibold text-gray-900 dark:text-gray-100">
					{itemType.charAt(0).toUpperCase() + itemType.slice(1)} Details
				</h4>
				<div class="space-y-2 text-sm">
					<div class="flex justify-between">
						<span class="text-gray-600 dark:text-gray-400">Name:</span>
						<span class="font-medium text-gray-900 dark:text-gray-100">{displayName}</span>
					</div>

					{#if itemType === 'server' && item.host}
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Host:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100">
								{item.host}:{item.port || 22}
							</span>
						</div>
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Setup Status:</span>
							<span class="text-gray-900 dark:text-gray-100">
								{#if item.setup_complete && item.security_locked}
									<span
										class="rounded bg-emerald-50 px-2 py-1 text-xs text-emerald-700 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:ring-emerald-800"
									>
										Ready
									</span>
								{:else if item.setup_complete}
									<span
										class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-700 ring-1 ring-amber-200 dark:bg-amber-950 dark:text-amber-300 dark:ring-amber-800"
									>
										Setup Complete
									</span>
								{:else}
									<span
										class="rounded bg-red-50 px-2 py-1 text-xs text-red-700 ring-1 ring-red-200 dark:bg-red-950 dark:text-red-300 dark:ring-red-800"
									>
										Not Setup
									</span>
								{/if}
							</span>
						</div>
					{/if}

					{#if itemType === 'app' && item.domain}
						<div class="flex justify-between">
							<span class="text-gray-600 dark:text-gray-400">Domain:</span>
							<span class="font-mono text-gray-900 dark:text-gray-100">{item.domain}</span>
						</div>
						{#if item.status}
							<div class="flex justify-between">
								<span class="text-gray-600 dark:text-gray-400">Status:</span>
								<span class="text-gray-900 dark:text-gray-100">
									{#if item.status === 'online'}
										<span
											class="rounded bg-green-50 px-2 py-1 text-xs text-green-700 ring-1 ring-green-200 dark:bg-green-950 dark:text-green-300 dark:ring-green-800"
										>
											Online
										</span>
									{:else if item.status === 'offline'}
										<span
											class="rounded bg-red-50 px-2 py-1 text-xs text-red-700 ring-1 ring-red-200 dark:bg-red-950 dark:text-red-300 dark:ring-red-800"
										>
											Offline
										</span>
									{:else}
										<span
											class="rounded bg-gray-50 px-2 py-1 text-xs text-gray-700 ring-1 ring-gray-200 dark:bg-gray-950 dark:text-gray-300 dark:ring-gray-800"
										>
											{item.status}
										</span>
									{/if}
								</span>
							</div>
						{/if}
					{/if}
				</div>
			</div>

			<!-- Related Items Warning -->
			{#if relatedItems && relatedItems.length > 0}
				<div
					class="rounded-lg bg-amber-50 p-4 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800"
				>
					<div class="flex">
						<div class="flex-shrink-0">
							<svg class="h-5 w-5 text-amber-500" viewBox="0 0 20 20" fill="currentColor">
								<path
									fill-rule="evenodd"
									d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
									clip-rule="evenodd"
								/>
							</svg>
						</div>
						<div class="ml-3">
							<h3 class="text-sm font-semibold text-amber-700 dark:text-amber-300">
								Warning: Related {relatedItemsType.charAt(0).toUpperCase() +
									relatedItemsType.slice(1)} Found
							</h3>
							<div class="mt-2">
								<p class="text-sm text-amber-600 dark:text-amber-400">
									This {itemType} has {relatedItems.length}
									{relatedItemsType}{relatedItems.length !== 1 ? '' : ''}:
								</p>
								<ul
									class="mt-2 list-inside list-disc space-y-1 text-sm text-amber-600 dark:text-amber-400"
								>
									{#each relatedItems as relatedItem (relatedItem.id)}
										<li>
											<span class="font-medium">{relatedItem.name}</span>
											{#if relatedItem.domain}
												<span class="text-xs">({relatedItem.domain})</span>
											{/if}
										</li>
									{/each}
								</ul>
								<p class="mt-2 text-sm text-amber-600 dark:text-amber-400">
									You should delete or migrate these {relatedItemsType} before deleting the {itemType}.
								</p>
							</div>
						</div>
					</div>
				</div>
			{/if}

			<!-- Confirmation Input -->
			<div
				class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
			>
				<p class="mb-2 text-sm text-gray-700 dark:text-gray-300">
					To confirm deletion, type the {itemType} name:
					<strong class="font-mono">{displayName}</strong>
				</p>
				<input
					type="text"
					placeholder="Enter {itemType} name to confirm"
					class="mt-1 block w-full rounded-lg border-gray-200 bg-white shadow-sm transition-all duration-200 focus:border-red-500 focus:ring-2 focus:ring-red-500 focus:ring-offset-0 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100"
					bind:value={confirmationText}
					disabled={loading}
				/>
			</div>
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No {itemType} selected for deletion</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<button
				onclick={handleClose}
				disabled={loading}
				class="rounded-lg border border-gray-200 bg-white px-4 py-2 font-medium text-gray-900 transition-colors hover:border-gray-300 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900"
			>
				Cancel
			</button>
			<button
				onclick={handleConfirm}
				disabled={loading || !item || confirmationText !== displayName}
				class="rounded-lg border border-red-500 bg-red-500 px-4 py-2 font-medium text-white shadow-sm transition-colors hover:border-red-600 hover:bg-red-600 disabled:opacity-50"
			>
				{#if loading}
					<div class="flex items-center">
						<div class="mr-2 h-4 w-4 animate-spin rounded-full border-b-2 border-white"></div>
						Deleting...
					</div>
				{:else}
					Delete {itemType.charAt(0).toUpperCase() + itemType.slice(1)}
				{/if}
			</button>
		</div>
	{/snippet}
</Modal>
