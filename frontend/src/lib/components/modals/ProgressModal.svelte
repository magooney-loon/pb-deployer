<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { SetupStep } from '../../api.js';
	import { ProgressModalLogic } from './ProgressModal.js';

	interface Props {
		show?: boolean;
		title: string;
		progress: SetupStep[];
		onClose: () => void;
		loading?: boolean;
		operationInProgress?: boolean;
	}

	let {
		show = $bindable(false),
		title,
		progress,
		onClose,
		loading = false,
		operationInProgress = false
	}: Props = $props();

	// Create logic instance
	const logic = new ProgressModalLogic({
		show,
		title,
		progress,
		onClose,
		loading,
		operationInProgress
	});
	let state = $state(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update props when they change
	$effect(() => {
		logic.updateProps({ show, title, progress, onClose, loading, operationInProgress });
	});

	// Update bindable show prop
	$effect(() => {
		show = state.show;
	});
</script>

<Modal
	open={state.show}
	title={state.title}
	size="lg"
	closeable={!state.operationInProgress && !state.loading}
	onclose={() => logic.handleClose()}
>
	<!-- Warning message when operation is in progress -->
	{#if state.operationInProgress || state.loading || (state.progress.length > 0 && state.progress[state.progress.length - 1]?.status === 'running')}
		<div
			class="mb-4 rounded-lg bg-amber-50 p-4 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800"
		>
			<div class="flex">
				<div class="flex-shrink-0">
					<span class="text-amber-500">⚠️</span>
				</div>
				<div class="ml-3">
					<p class="text-sm text-amber-700 dark:text-amber-300">
						Operation in progress. Please do not close this window until the process completes.
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Overall Progress Bar -->
	<div class="mb-6">
		<div class="mb-3 flex justify-between text-sm">
			<span class="text-gray-700 dark:text-gray-300">Progress</span>
			<span class="font-medium text-gray-900 dark:text-gray-100"
				>{state.progress.length > 0
					? state.progress[state.progress.length - 1]?.progress_pct || 0
					: 0}%</span
			>
		</div>
		<div class="h-2 w-full rounded-full bg-gray-200 dark:bg-gray-800">
			<div
				class="h-2 rounded-full transition-all duration-300 {state.progress.length > 0 &&
				state.progress[state.progress.length - 1]?.status === 'failed'
					? 'bg-red-500'
					: state.progress.length > 0 &&
						  state.progress[state.progress.length - 1]?.step === 'complete' &&
						  state.progress[state.progress.length - 1]?.status === 'success'
						? 'bg-emerald-500'
						: 'bg-gray-900 dark:bg-gray-100'}"
				style="width: {state.progress.length > 0
					? state.progress[state.progress.length - 1]?.progress_pct || 0
					: 0}%"
			></div>
		</div>
	</div>

	<!-- Progress Steps -->
	<div class="progress-steps max-h-96 space-y-3 overflow-y-auto">
		{#if state.progress.length === 0 && state.loading}
			<div class="flex items-center space-x-3 py-4">
				<div
					class="h-6 w-6 animate-spin rounded-full border-b-2 border-gray-900 dark:border-gray-100"
				></div>
				<span class="text-gray-700 dark:text-gray-300">Initializing...</span>
			</div>
		{:else if state.progress.length === 0}
			<div class="py-8 text-center text-gray-600 dark:text-gray-400">
				No progress data available
			</div>
		{:else}
			{#each state.progress as step, index (step.timestamp + index)}
				<div
					class="rounded-lg p-4 {step.status === 'failed'
						? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
						: step.status === 'success'
							? 'bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800'
							: 'bg-gray-50 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800'}"
				>
					<div class="flex items-start space-x-3">
						<span class="text-lg">
							{step.status === 'running'
								? '🔄'
								: step.status === 'success'
									? '✅'
									: step.status === 'failed'
										? '❌'
										: '⏳'}
						</span>
						<div class="min-w-0 flex-1">
							<div class="flex items-center justify-between">
								<h4
									class="text-sm font-semibold {step.status === 'running'
										? 'text-gray-900 dark:text-gray-100'
										: step.status === 'success'
											? 'text-emerald-700 dark:text-emerald-300'
											: step.status === 'failed'
												? 'text-red-700 dark:text-red-300'
												: 'text-gray-700 dark:text-gray-300'}"
								>
									{step.message}
								</h4>
								<span class="text-xs text-gray-500 dark:text-gray-400">
									{new Date(step.timestamp).toLocaleTimeString()}
								</span>
							</div>
							{#if step.details}
								<p class="mt-1 text-xs text-gray-600 dark:text-gray-400">
									{step.details}
								</p>
							{/if}
							<div class="mt-1 text-xs text-gray-500 dark:text-gray-500">
								Step: {step.step}
							</div>
						</div>
					</div>
				</div>
			{/each}
		{/if}
	</div>

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<button
				onclick={() => logic.handleClose()}
				disabled={state.operationInProgress ||
					state.loading ||
					(state.progress.length > 0 &&
						state.progress[state.progress.length - 1]?.status === 'running')}
				class="rounded-lg px-4 py-2 text-sm font-medium transition-colors {state.progress.length >
					0 && state.progress[state.progress.length - 1]?.step === 'complete'
					? state.progress[state.progress.length - 1]?.status === 'success'
						? 'border border-emerald-500 bg-emerald-500 text-white shadow-sm hover:border-emerald-600 hover:bg-emerald-600'
						: 'border border-red-500 bg-red-500 text-white shadow-sm hover:border-red-600 hover:bg-red-600'
					: 'border border-gray-200 bg-white text-gray-900 hover:border-gray-300 hover:bg-gray-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900'} disabled:opacity-50"
			>
				{state.progress.length > 0 && state.progress[state.progress.length - 1]?.step === 'complete'
					? state.progress[state.progress.length - 1]?.status === 'success'
						? 'Done'
						: 'Close'
					: state.operationInProgress ||
						  state.loading ||
						  (state.progress.length > 0 &&
								state.progress[state.progress.length - 1]?.status === 'running')
						? 'Operation in Progress...'
						: 'Close'}
			</button>
		</div>
	{/snippet}
</Modal>

<style>
	/* Custom scrollbar for progress steps */
	:global(.progress-steps::-webkit-scrollbar) {
		width: 6px;
	}

	:global(.progress-steps::-webkit-scrollbar-track) {
		background: rgb(243 244 246);
		border-radius: 3px;
	}

	:global(.progress-steps::-webkit-scrollbar-thumb) {
		background: rgb(156 163 175);
		border-radius: 3px;
	}

	:global(.progress-steps::-webkit-scrollbar-thumb:hover) {
		background: rgb(107 114 128);
	}

	:global([data-theme='dark']) :global(.progress-steps::-webkit-scrollbar-track) {
		background: rgb(55 65 81);
	}

	:global([data-theme='dark']) :global(.progress-steps::-webkit-scrollbar-thumb) {
		background: rgb(107 114 128);
	}

	:global([data-theme='dark']) :global(.progress-steps::-webkit-scrollbar-thumb:hover) {
		background: rgb(156 163 175);
	}
</style>
