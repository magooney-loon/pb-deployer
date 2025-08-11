<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { TroubleshootModalLogic, type TroubleshootResult } from './TroubleshootModal.js';

	interface Props {
		open?: boolean;
		result?: TroubleshootResult | null;
		serverName?: string;
		loading?: boolean;
		onclose?: () => void;
		onretry?: () => void;
		onquicktest?: () => void;
	}

	let {
		open = false,
		result = null,
		serverName = '',
		loading = false,
		onclose,
		onretry,
		onquicktest
	}: Props = $props();

	// Create logic instance
	const logic = new TroubleshootModalLogic({
		open,
		result,
		serverName,
		loading,
		onclose,
		onretry,
		onquicktest
	});
	let state = $state(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update props when they change
	$effect(() => {
		logic.updateProps({
			open,
			result,
			serverName,
			loading,
			onclose,
			onretry,
			onquicktest
		});
	});
</script>

<Modal
	open={state.open}
	title={state.loading ? 'Running Diagnostics...' : 'SSH Connection Troubleshooting'}
	size="lg"
	onclose={() => logic.handleClose()}
>
	{#if state.loading}
		<!-- Loading State -->
		<div class="py-8 text-center">
			<div class="flex items-center justify-center">
				<div
					class="h-8 w-8 animate-spin rounded-full border-b-2 border-gray-900 dark:border-gray-100"
				></div>
				<span class="ml-3 text-gray-700 dark:text-gray-300">
					Analyzing connection to {state.serverName || 'server'}...
				</span>
			</div>
			<p class="mt-2 text-xs text-gray-500 dark:text-gray-400">This may take up to 30 seconds...</p>
		</div>
	{:else if state.result}
		<!-- Results Header -->
		<div class="mb-6 text-center">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-full {state.result.success
					? 'bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800'
					: state.result.has_errors
						? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
						: 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800'}"
			>
				{#if state.result.success}
					<svg
						class="h-6 w-6 text-emerald-600 dark:text-emerald-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"
						></path>
					</svg>
				{:else if state.result.has_errors}
					<svg
						class="h-6 w-6 text-red-600 dark:text-red-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						></path>
					</svg>
				{:else}
					<svg
						class="h-6 w-6 text-amber-600 dark:text-amber-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16.5c-.77.833.192 2.5 1.732 2.5z"
						></path>
					</svg>
				{/if}
			</div>
			<div class="mt-4">
				<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
					{state.result.success
						? 'All Diagnostics Passed'
						: state.result.has_errors
							? 'Connection Issues Found'
							: 'Warnings Detected'}
				</h3>
				<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
					{state.result.host}:{state.result.port} •
					{state.result.success_count} passed,
					{state.result.warning_count} warnings,
					{state.result.error_count} errors
				</div>
			</div>
		</div>

		<!-- Quick Summary -->
		{#if state.result.summary}
			<div
				class="mb-6 rounded-lg bg-blue-50 p-4 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800"
			>
				<h4 class="mb-2 text-sm font-semibold text-blue-900 dark:text-blue-100">Summary</h4>
				<div class="text-sm whitespace-pre-line text-blue-800 dark:text-blue-200">
					{state.result.summary}
				</div>
			</div>
		{/if}

		<!-- Priority Suggestions -->
		{#if state.result.suggestions && state.result.suggestions.length > 0}
			<div
				class="mb-6 rounded-lg {state.result.has_errors
					? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
					: 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800'}"
			>
				<div class="p-4">
					<h4
						class="mb-3 flex items-center text-sm font-semibold {state.result.has_errors
							? 'text-red-900 dark:text-red-100'
							: 'text-amber-900 dark:text-amber-100'}"
					>
						<span class="mr-2">{state.result.has_errors ? '🚨' : '💡'}</span>
						{state.result.has_errors ? 'Recommended Actions' : 'Suggestions'}
					</h4>
					<ul
						class="space-y-2 text-sm {state.result.has_errors
							? 'text-red-800 dark:text-red-200'
							: 'text-amber-800 dark:text-amber-200'}"
					>
						{#each state.result.suggestions.slice(0, 3) as suggestion, i (i)}
							<li class="flex items-start">
								<span class="mt-0.5 mr-2 flex-shrink-0">•</span>
								<span>{suggestion}</span>
							</li>
						{/each}
					</ul>
				</div>
			</div>
		{/if}

		<!-- Diagnostic Steps -->
		<div class="space-y-3">
			<h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">Diagnostic Results</h4>
			<div class="max-h-96 space-y-3 overflow-y-auto">
				{#each state.result.diagnostics as diagnostic, i (i)}
					<div
						class="rounded-lg p-4 {diagnostic.status === 'error'
							? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
							: diagnostic.status === 'warning'
								? 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800'
								: 'bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800'}"
					>
						<div class="flex items-start space-x-3">
							<span
								class="text-lg {diagnostic.status === 'error'
									? 'text-red-500'
									: diagnostic.status === 'warning'
										? 'text-amber-500'
										: 'text-emerald-500'}"
							>
								{diagnostic.status === 'error'
									? '❌'
									: diagnostic.status === 'warning'
										? '⚠️'
										: '✅'}
							</span>
							<div class="min-w-0 flex-1">
								<div class="flex items-center justify-between">
									<h5
										class="text-sm font-semibold {diagnostic.status === 'error'
											? 'text-red-900 dark:text-red-100'
											: diagnostic.status === 'warning'
												? 'text-amber-900 dark:text-amber-100'
												: 'text-emerald-900 dark:text-emerald-100'}"
									>
										{logic.formatStepName(diagnostic.step)}: {diagnostic.message}
									</h5>
								</div>
								{#if diagnostic.details}
									<div class="mt-2">
										<p
											class="text-xs whitespace-pre-line {diagnostic.status === 'error'
												? 'text-red-700 dark:text-red-300'
												: diagnostic.status === 'warning'
													? 'text-amber-700 dark:text-amber-300'
													: 'text-emerald-700 dark:text-emerald-300'}"
										>
											{diagnostic.details}
										</p>
									</div>
								{/if}
								{#if diagnostic.suggestion}
									<div class="mt-2">
										<div
											class="rounded bg-white p-2 {diagnostic.status === 'error'
												? 'ring-1 ring-red-300 dark:bg-red-900 dark:ring-red-700'
												: diagnostic.status === 'warning'
													? 'ring-1 ring-amber-300 dark:bg-amber-900 dark:ring-amber-700'
													: 'ring-1 ring-emerald-300 dark:bg-emerald-900 dark:ring-emerald-700'}"
										>
											<p
												class="text-xs {diagnostic.status === 'error'
													? 'text-red-800 dark:text-red-200'
													: diagnostic.status === 'warning'
														? 'text-amber-800 dark:text-amber-200'
														: 'text-emerald-800 dark:text-emerald-200'}"
											>
												<strong>💡 Suggestion:</strong>
												{diagnostic.suggestion}
											</p>
										</div>
									</div>
								{/if}
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>

		<!-- Additional Actions -->
		{#if state.result.has_errors}
			<div
				class="mt-6 rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
			>
				<h4 class="mb-3 text-sm font-semibold text-gray-900 dark:text-gray-100">
					🔧 Additional Actions
				</h4>
				<div class="space-y-2 text-sm text-gray-600 dark:text-gray-400">
					<p>• Run a quick connectivity test to check current status</p>
					<p>• Use console access to run server-side diagnostics</p>
					<p>• Try connecting from a different IP address</p>
					{#if logic.isConnectionRefusedDetected()}
						<p>• Wait 10-15 minutes for temporary fail2ban bans to expire</p>
					{/if}
				</div>
			</div>
		{/if}
	{:else}
		<!-- No result state -->
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No diagnostic results available</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex items-center justify-between">
			<div class="flex items-center space-x-3">
				{#if state.result && (state.result.has_errors || state.result.has_warnings)}
					<span class="text-xs text-gray-500 dark:text-gray-400">
						Last check: {logic.getFormattedTimestamp()}
					</span>
				{/if}
			</div>

			<div class="flex space-x-3">
				{#if onquicktest && state.result}
					<button
						onclick={() => logic.handleQuickTest()}
						disabled={state.loading}
						class="rounded-lg border border-blue-200 bg-blue-50 px-4 py-2 text-sm font-medium text-blue-900 transition-colors hover:border-blue-300 hover:bg-blue-100 disabled:opacity-50 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-100 dark:hover:bg-blue-900"
					>
						{#if state.loading}
							Quick Testing...
						{:else}
							Quick Test
						{/if}
					</button>
				{/if}

				{#if onretry && state.result && (state.result.has_errors || state.result.has_warnings)}
					<button
						onclick={() => logic.handleRetry()}
						disabled={state.loading}
						class="rounded-lg border border-orange-200 bg-orange-50 px-4 py-2 text-sm font-medium text-orange-900 transition-colors hover:border-orange-300 hover:bg-orange-100 disabled:opacity-50 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-100 dark:hover:bg-orange-900"
					>
						{#if state.loading}
							Re-running...
						{:else}
							Run Again
						{/if}
					</button>
				{/if}

				<button
					onclick={() => logic.handleClose()}
					class="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-900 transition-colors hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900"
				>
					Close
				</button>
			</div>
		</div>
	{/snippet}
</Modal>

<style>
	/* Custom scrollbar for diagnostic steps */
	:global(.diagnostic-steps::-webkit-scrollbar) {
		width: 6px;
	}

	:global(.diagnostic-steps::-webkit-scrollbar-track) {
		background: rgb(243 244 246);
		border-radius: 3px;
	}

	:global(.diagnostic-steps::-webkit-scrollbar-thumb) {
		background: rgb(156 163 175);
		border-radius: 3px;
	}

	:global(.diagnostic-steps::-webkit-scrollbar-thumb:hover) {
		background: rgb(107 114 128);
	}

	:global([data-theme='dark']) :global(.diagnostic-steps::-webkit-scrollbar-track) {
		background: rgb(55 65 81);
	}

	:global([data-theme='dark']) :global(.diagnostic-steps::-webkit-scrollbar-thumb) {
		background: rgb(107 114 128);
	}

	:global([data-theme='dark']) :global(.diagnostic-steps::-webkit-scrollbar-thumb:hover) {
		background: rgb(156 163 175);
	}
</style>
