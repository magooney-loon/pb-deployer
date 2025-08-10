<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import type { SetupStep } from '../../api.js';

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

	function getProgressStepIcon(status: string): string {
		switch (status) {
			case 'running':
				return '🔄';
			case 'success':
				return '✅';
			case 'failed':
				return '❌';
			default:
				return '⏳';
		}
	}

	function getProgressStepColor(status: string): string {
		switch (status) {
			case 'running':
				return 'text-blue-600 dark:text-blue-400';
			case 'success':
				return 'text-green-600 dark:text-green-400';
			case 'failed':
				return 'text-red-600 dark:text-red-400';
			default:
				return 'text-gray-600 dark:text-gray-400';
		}
	}

	function getOverallProgress(): number {
		if (progress.length === 0) return 0;
		const latestStep = progress[progress.length - 1];
		return latestStep.progress_pct || 0;
	}

	function isComplete(): boolean {
		if (progress.length === 0) return false;
		const latestStep = progress[progress.length - 1];
		return latestStep.step === 'complete';
	}

	function isSuccess(): boolean {
		if (!isComplete()) return false;
		const latestStep = progress[progress.length - 1];
		return latestStep.status === 'success';
	}

	function isFailed(): boolean {
		const latestStep = progress[progress.length - 1];
		return latestStep?.status === 'failed';
	}

	function isInProgress(): boolean {
		// If operation has failed or completed, it's not in progress
		if (isFailed() || isComplete()) return false;

		// Use explicit operation status if provided
		if (operationInProgress) return true;
		if (loading) return true;
		if (progress.length === 0) return false;
		const latestStep = progress[progress.length - 1];
		return latestStep.status === 'running';
	}

	function handleClose() {
		if (!isInProgress()) {
			onClose();
		}
	}
</script>

<Modal open={show} {title} size="lg" closeable={!isInProgress()} onclose={handleClose}>
	<!-- Warning message when operation is in progress -->
	{#if isInProgress()}
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
			<span class="text-gray-600 dark:text-gray-400">{getOverallProgress()}%</span>
		</div>
		<div class="h-2 w-full rounded-full bg-gray-200 dark:bg-gray-700">
			<div
				class="h-2 rounded-full transition-all duration-300 {isSuccess()
					? 'bg-green-500'
					: isFailed()
						? 'bg-red-500'
						: 'bg-blue-500'}"
				style="width: {getOverallProgress()}%"
			></div>
		</div>
	</div>

	<!-- Progress Steps -->
	<div class="progress-steps max-h-96 space-y-3 overflow-y-auto">
		{#if progress.length === 0 && loading}
			<div class="flex items-center space-x-3 py-3">
				<div class="h-6 w-6 animate-spin rounded-full border-b-2 border-blue-600"></div>
				<span class="text-gray-600 dark:text-gray-400">Initializing...</span>
			</div>
		{:else if progress.length === 0}
			<div class="py-6 text-center text-gray-500 dark:text-gray-400">
				No progress data available
			</div>
		{:else}
			{#each progress as step, index (step.timestamp + index)}
				<div
					class="rounded-lg border p-3 {step.status === 'failed'
						? 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-900/20'
						: step.status === 'success'
							? 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20'
							: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-900/20'}"
				>
					<div class="flex items-start space-x-3">
						<span class="text-lg">
							{getProgressStepIcon(step.status)}
						</span>
						<div class="min-w-0 flex-1">
							<div class="flex items-center justify-between">
								<h4 class="text-sm font-medium {getProgressStepColor(step.status)}">
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
			{#if isComplete()}
				<button
					onclick={handleClose}
					class="rounded-md px-4 py-2 text-sm font-medium {isSuccess()
						? 'bg-green-600 text-white hover:bg-green-700'
						: 'bg-red-600 text-white hover:bg-red-700'}"
				>
					{isSuccess() ? 'Done' : 'Close'}
				</button>
			{:else if isInProgress()}
				<button
					disabled
					class="cursor-not-allowed rounded-md border border-gray-300 bg-gray-100 px-4 py-2 text-sm font-medium text-gray-400 dark:border-gray-600 dark:bg-gray-600 dark:text-gray-500"
				>
					Operation in Progress...
				</button>
			{:else}
				<button
					onclick={handleClose}
					class="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600"
				>
					Close
				</button>
			{/if}
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
