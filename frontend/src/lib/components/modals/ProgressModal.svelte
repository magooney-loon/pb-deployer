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
		<div class="mb-4 rounded-md bg-yellow-50 p-3 dark:bg-yellow-900/20">
			<div class="flex">
				<div class="flex-shrink-0">
					<span class="text-yellow-400">⚠️</span>
				</div>
				<div class="ml-3">
					<p class="text-sm text-yellow-700 dark:text-yellow-300">
						Operation in progress. Please do not close this window until the process completes.
					</p>
				</div>
			</div>
		</div>
	{/if}

	<!-- Overall Progress Bar -->
	<div class="mb-6">
		<div class="mb-2 flex justify-between text-sm">
			<span class="text-gray-600 dark:text-gray-400">Progress</span>
			<span class="text-gray-600 dark:text-gray-400"
				>{state.progress.length > 0
					? state.progress[state.progress.length - 1]?.progress_pct || 0
					: 0}%</span
			>
		</div>
		<div class="h-2 w-full rounded-full bg-gray-200 dark:bg-gray-700">
			<div
				class="h-2 rounded-full transition-all duration-300 {state.progress.length > 0 &&
				state.progress[state.progress.length - 1]?.status === 'failed'
					? 'bg-red-500'
					: state.progress.length > 0 &&
						  state.progress[state.progress.length - 1]?.step === 'complete' &&
						  state.progress[state.progress.length - 1]?.status === 'success'
						? 'bg-green-500'
						: 'bg-blue-500'}"
				style="width: {state.progress.length > 0
					? state.progress[state.progress.length - 1]?.progress_pct || 0
					: 0}%"
			></div>
		</div>
	</div>

	<!-- Progress Steps -->
	<div class="progress-steps max-h-96 space-y-3 overflow-y-auto">
		{#if state.progress.length === 0 && state.loading}
			<div class="flex items-center space-x-3 py-3">
				<div class="h-6 w-6 animate-spin rounded-full border-b-2 border-blue-600"></div>
				<span class="text-gray-600 dark:text-gray-400">Initializing...</span>
			</div>
		{:else if state.progress.length === 0}
			<div class="py-6 text-center text-gray-500 dark:text-gray-400">
				No progress data available
			</div>
		{:else}
			{#each state.progress as step, index (step.timestamp + index)}
				<div
					class="rounded-lg border p-3 {step.status === 'failed'
						? 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20'
						: step.status === 'success'
							? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20'
							: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20'}"
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
									class="text-sm font-medium {step.status === 'running'
										? 'text-blue-600 dark:text-blue-400'
										: step.status === 'success'
											? 'text-green-600 dark:text-green-400'
											: step.status === 'failed'
												? 'text-red-600 dark:text-red-400'
												: 'text-gray-600 dark:text-gray-400'}"
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
				class="rounded-md px-4 py-2 text-sm font-medium {state.progress.length > 0 &&
				state.progress[state.progress.length - 1]?.step === 'complete'
					? state.progress[state.progress.length - 1]?.status === 'success'
						? 'bg-green-600 text-white hover:bg-green-700'
						: 'bg-red-600 text-white hover:bg-red-700'
					: 'border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'}"
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
