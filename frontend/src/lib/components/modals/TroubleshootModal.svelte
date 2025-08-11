<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import {
		TroubleshootModalLogic,
		type TroubleshootResult,
		type EnhancedTroubleshootResult,
		type AutoFixResult
	} from './TroubleshootModal.js';

	interface Props {
		open?: boolean;
		result?: TroubleshootResult | null;
		enhancedResult?: EnhancedTroubleshootResult | null;
		autoFixResult?: AutoFixResult | null;
		serverName?: string;
		loading?: boolean;
		mode?: 'basic' | 'enhanced' | 'auto-fix';
		onclose?: () => void;
		onretry?: () => void;
		onquicktest?: () => void;
		onenhanced?: () => void;
		onautofix?: () => void;
	}

	let {
		open = false,
		result = null,
		enhancedResult = null,
		autoFixResult = null,
		serverName = '',
		loading = false,
		mode = 'basic',
		onclose,
		onretry,
		onquicktest,
		onenhanced,
		onautofix
	}: Props = $props();

	// Create logic instance
	const logic = new TroubleshootModalLogic({
		open,
		result,
		enhancedResult,
		autoFixResult,
		serverName,
		loading,
		mode,
		onclose,
		onretry,
		onquicktest,
		onenhanced,
		onautofix
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
			enhancedResult,
			autoFixResult,
			serverName,
			loading,
			mode,
			onclose,
			onretry,
			onquicktest,
			onenhanced,
			onautofix
		});
	});
</script>

<Modal open={state.open} title={logic.getTitle()} size="xl" onclose={() => logic.handleClose()}>
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
	{:else if logic.hasResult()}
		<!-- Mode Selector for Enhanced Features -->
		{#if logic.getMode() === 'enhanced' || logic.getMode() === 'auto-fix'}
			<div class="mb-4 flex space-x-1 rounded-lg bg-gray-100 p-1 dark:bg-gray-800">
				<button
					onclick={() => logic.setCurrentView('diagnostics')}
					class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors {logic.getCurrentView() ===
					'diagnostics'
						? 'bg-white text-gray-900 shadow dark:bg-gray-700 dark:text-gray-100'
						: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'}"
				>
					Diagnostics
				</button>
				{#if logic.getMode() === 'enhanced'}
					<button
						onclick={() => logic.setCurrentView('analysis')}
						class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors {logic.getCurrentView() ===
						'analysis'
							? 'bg-white text-gray-900 shadow dark:bg-gray-700 dark:text-gray-100'
							: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'}"
					>
						Analysis
					</button>
					<button
						onclick={() => logic.setCurrentView('recovery')}
						class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors {logic.getCurrentView() ===
						'recovery'
							? 'bg-white text-gray-900 shadow dark:bg-gray-700 dark:text-gray-100'
							: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'}"
					>
						Recovery
					</button>
					<button
						onclick={() => logic.setCurrentView('suggestions')}
						class="flex-1 rounded-md px-3 py-2 text-sm font-medium transition-colors {logic.getCurrentView() ===
						'suggestions'
							? 'bg-white text-gray-900 shadow dark:bg-gray-700 dark:text-gray-100'
							: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'}"
					>
						Actions
					</button>
				{/if}
			</div>
		{/if}

		<!-- Results Header -->
		<div class="mb-6 text-center">
			<div
				class="mx-auto flex h-12 w-12 items-center justify-center rounded-full {logic.isSuccess()
					? 'bg-emerald-50 ring-1 ring-emerald-200 dark:bg-emerald-950 dark:ring-emerald-800'
					: logic.hasErrors()
						? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
						: 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800'}"
			>
				{#if logic.isSuccess()}
					<svg
						class="h-6 w-6 text-emerald-600 dark:text-emerald-400"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"
						></path>
					</svg>
				{:else if logic.hasErrors()}
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
					{logic.isSuccess()
						? logic.getMode() === 'auto-fix'
							? 'Auto-Fix Completed'
							: 'All Diagnostics Passed'
						: logic.hasErrors()
							? logic.getMode() === 'auto-fix'
								? 'Auto-Fix Issues'
								: 'Connection Issues Found'
							: 'Warnings Detected'}
				</h3>
				<div class="mt-2 text-sm text-gray-600 dark:text-gray-400">
					{logic.getCurrentResult()?.host}:{logic.getCurrentResult()?.port}
					{#if logic.getMode() !== 'auto-fix'}
						• {logic.getStatusSummary()}
					{:else}
						• {logic.getStatusSummary()}
					{/if}
					{#if logic.getClientIP() !== 'unknown'}
						• Client IP: {logic.getClientIP()}
					{/if}
				</div>
			</div>
		</div>

		<!-- Content Views -->
		{#if logic.getCurrentView() === 'analysis' && logic.getMode() === 'enhanced'}
			<!-- Enhanced Analysis View -->
			<div class="space-y-6">
				<!-- Pattern Analysis -->
				{#if logic.getAnalysis()}
					<div
						class="rounded-lg bg-blue-50 p-4 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800"
					>
						<h4 class="mb-2 text-sm font-semibold text-blue-900 dark:text-blue-100">
							🔍 Issue Analysis
						</h4>
						<div class="space-y-2 text-sm text-blue-800 dark:text-blue-200">
							<p><strong>Pattern:</strong> {logic.getPatternDescription()}</p>
							<p>
								<strong>Confidence:</strong>
								{Math.round(logic.getAnalysisConfidence() * 100)}%
							</p>
							<p><strong>Category:</strong> {logic.getAnalysisCategory()}</p>
							<p>
								<strong>Auto-fixable:</strong>
								{logic.getAnalysisAutoFixable() ? 'Yes' : 'No'}
							</p>
							{#if logic.getImmediateAction()}
								<p><strong>Immediate Action:</strong> {logic.getImmediateAction()}</p>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Performance Metrics -->
				{#if logic.getConnectionTime()}
					<div
						class="rounded-lg bg-gray-50 p-4 ring-1 ring-gray-200 dark:bg-gray-900 dark:ring-gray-800"
					>
						<h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
							⏱️ Performance
						</h4>
						<div class="grid grid-cols-2 gap-4 text-sm">
							<div>
								<span class="text-gray-600 dark:text-gray-400">Connection Time:</span>
								<span class="ml-2 font-mono">{logic.formatDuration(logic.getConnectionTime())}</span
								>
							</div>
							<div>
								<span class="text-gray-600 dark:text-gray-400">Estimated Duration:</span>
								<span class="ml-2 font-mono">{logic.getEstimatedDuration()}</span>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{:else if logic.getCurrentView() === 'recovery' && logic.getMode() === 'enhanced'}
			<!-- Recovery Plan View -->
			<div class="space-y-6">
				{#if logic.getRecoveryPlan()}
					<!-- Recovery Overview -->
					<div
						class="rounded-lg bg-amber-50 p-4 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800"
					>
						<h4 class="mb-2 text-sm font-semibold text-amber-900 dark:text-amber-100">
							🛠️ Recovery Plan
						</h4>
						<div class="grid grid-cols-2 gap-4 text-sm text-amber-800 dark:text-amber-200">
							<div>
								<span class="text-amber-600 dark:text-amber-400">Success Probability:</span>
								<span class="ml-2 font-semibold"
									>{Math.round(logic.getSuccessProbability() * 100)}%</span
								>
							</div>
							<div>
								<span class="text-amber-600 dark:text-amber-400">Estimated Time:</span>
								<span class="ml-2 font-semibold">{logic.getRecoveryPlanEstimatedTime()}</span>
							</div>
						</div>
						{#if logic.getRequiredAccess().length > 0}
							<div class="mt-3">
								<span class="text-amber-600 dark:text-amber-400">Required Access:</span>
								<span class="ml-2">{logic.getRequiredAccess().join(', ')}</span>
							</div>
						{/if}
					</div>

					<!-- Critical Issues -->
					{#if logic.getCriticalIssues().length > 0}
						<div
							class="rounded-lg bg-red-50 p-4 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800"
						>
							<h4 class="mb-2 text-sm font-semibold text-red-900 dark:text-red-100">
								🚨 Critical Issues
							</h4>
							<ul class="space-y-1 text-sm text-red-800 dark:text-red-200">
								{#each logic.getCriticalIssues() as issue, i (i)}
									<li class="flex items-start">
										<span class="mt-0.5 mr-2">•</span>
										<span>{issue.replace(/_/g, ' ')}</span>
									</li>
								{/each}
							</ul>
						</div>
					{/if}

					<!-- Recovery Steps -->
					{#if logic.getPrioritySteps().length > 0}
						<div
							class="rounded-lg bg-green-50 p-4 ring-1 ring-green-200 dark:bg-green-950 dark:ring-green-800"
						>
							<h4 class="mb-3 text-sm font-semibold text-green-900 dark:text-green-100">
								🎯 Required Steps
							</h4>
							<div class="space-y-3">
								{#each logic.getPrioritySteps() as step, i (i)}
									<div class="flex items-start space-x-3">
										<span
											class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-green-100 text-xs font-semibold text-green-800 dark:bg-green-800 dark:text-green-100"
										>
											{step.step}
										</span>
										<div class="min-w-0 flex-1">
											<h5 class="text-sm font-semibold text-green-900 dark:text-green-100">
												{step.title}
											</h5>
											<p class="text-sm text-green-700 dark:text-green-300">{step.description}</p>
											{#if step.command}
												<div class="mt-2 rounded bg-gray-900 p-2">
													<code class="text-xs text-green-400">{step.command}</code>
												</div>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<!-- Optional Steps -->
					{#if logic.getOptionalSteps().length > 0}
						<div
							class="rounded-lg bg-blue-50 p-4 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800"
						>
							<h4 class="mb-3 text-sm font-semibold text-blue-900 dark:text-blue-100">
								💡 Optional Steps
							</h4>
							<div class="space-y-2">
								{#each logic.getOptionalSteps() as step, i (i)}
									<div class="text-sm">
										<span class="font-medium text-blue-800 dark:text-blue-200">{step.title}:</span>
										<span class="text-blue-700 dark:text-blue-300">{step.description}</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{/if}
			</div>
		{:else if logic.getCurrentView() === 'suggestions' && logic.getMode() === 'enhanced'}
			<!-- Actionable Suggestions View -->
			<div class="space-y-6">
				<!-- Automated Suggestions -->
				{#if logic.getAutomatedSuggestions().length > 0}
					<div
						class="rounded-lg bg-green-50 p-4 ring-1 ring-green-200 dark:bg-green-950 dark:ring-green-800"
					>
						<h4 class="mb-3 text-sm font-semibold text-green-900 dark:text-green-100">
							🤖 Automated Fixes
						</h4>
						<div class="space-y-2">
							{#each logic.getAutomatedSuggestions() as suggestion, i (i)}
								<div class="flex items-start space-x-3">
									<span class="text-green-500">✓</span>
									<div class="min-w-0 flex-1">
										<p class="text-sm font-medium text-green-900 dark:text-green-100">
											{suggestion.action}
										</p>
										{#if suggestion.command}
											<code class="mt-1 block rounded bg-gray-900 p-1 text-xs text-green-400"
												>{suggestion.command}</code
											>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Manual Suggestions by Priority -->
				{#each ['critical', 'high', 'medium', 'low'] as priority, i (i)}
					{#if logic.getSuggestionsByPriority(priority as 'critical' | 'high' | 'medium' | 'low').length > 0}
						<div
							class="rounded-lg {priority === 'critical'
								? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
								: priority === 'high'
									? 'bg-orange-50 ring-1 ring-orange-200 dark:bg-orange-950 dark:ring-orange-800'
									: 'bg-blue-50 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800'} p-4"
						>
							<h4
								class="mb-3 text-sm font-semibold {priority === 'critical'
									? 'text-red-900 dark:text-red-100'
									: priority === 'high'
										? 'text-orange-900 dark:text-orange-100'
										: 'text-blue-900 dark:text-blue-100'}"
							>
								{priority === 'critical' ? '🚨' : priority === 'high' ? '⚠️' : '💡'}
								{priority.charAt(0).toUpperCase() + priority.slice(1)} Priority
							</h4>
							<div class="space-y-2">
								{#each logic.getSuggestionsByPriority(priority as 'critical' | 'high' | 'medium' | 'low') as suggestion, j (j)}
									<div class="text-sm">
										<p
											class="font-medium {priority === 'critical'
												? 'text-red-900 dark:text-red-100'
												: priority === 'high'
													? 'text-orange-900 dark:text-orange-100'
													: 'text-blue-900 dark:text-blue-100'}"
										>
											{suggestion.action}
										</p>
										<p
											class="text-xs {priority === 'critical'
												? 'text-red-700 dark:text-red-300'
												: priority === 'high'
													? 'text-orange-700 dark:text-orange-300'
													: 'text-blue-700 dark:text-blue-300'}"
										>
											{suggestion.description}
										</p>
										{#if suggestion.command}
											<code class="mt-1 block rounded bg-gray-900 p-1 text-xs text-green-400"
												>{suggestion.command}</code
											>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{/each}
			</div>
		{:else}
			<!-- Default Diagnostics View -->
			{#if logic.getCurrentResult()?.summary && logic.getMode() !== 'auto-fix'}
				<div
					class="mb-6 rounded-lg bg-blue-50 p-4 ring-1 ring-blue-200 dark:bg-blue-950 dark:ring-blue-800"
				>
					<h4 class="mb-2 text-sm font-semibold text-blue-900 dark:text-blue-100">Summary</h4>
					<div class="text-sm whitespace-pre-line text-blue-800 dark:text-blue-200">
						{logic.getCurrentResult()?.summary || ''}
					</div>
				</div>
			{/if}

			<!-- Priority Suggestions -->
			{#if logic.getBasicSuggestions().length > 0 && logic.getMode() === 'basic'}
				<div
					class="mb-6 rounded-lg {logic.hasErrors()
						? 'bg-red-50 ring-1 ring-red-200 dark:bg-red-950 dark:ring-red-800'
						: 'bg-amber-50 ring-1 ring-amber-200 dark:bg-amber-950 dark:ring-amber-800'}"
				>
					<div class="p-4">
						<h4
							class="mb-3 flex items-center text-sm font-semibold {logic.hasErrors()
								? 'text-red-900 dark:text-red-100'
								: 'text-amber-900 dark:text-amber-100'}"
						>
							<span class="mr-2">{logic.hasErrors() ? '🚨' : '💡'}</span>
							{logic.hasErrors() ? 'Recommended Actions' : 'Suggestions'}
						</h4>
						<ul
							class="space-y-2 text-sm {logic.hasErrors()
								? 'text-red-800 dark:text-red-200'
								: 'text-amber-800 dark:text-amber-200'}"
						>
							{#each logic.getBasicSuggestions().slice(0, 3) as suggestion, i (i)}
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
				<div class="flex items-center justify-between">
					<h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
						{logic.getMode() === 'auto-fix' ? 'Applied Fixes' : 'Diagnostic Results'}
					</h4>
					{#if logic.getMode() === 'enhanced'}
						<button
							onclick={() => logic.toggleAdvanced()}
							class="text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
						>
							{logic.getShowAdvanced() ? 'Hide' : 'Show'} Advanced
						</button>
					{/if}
				</div>
				<div class="diagnostic-steps max-h-96 space-y-3 overflow-y-auto">
					{#each logic.getDiagnostics() as diagnostic, i (i)}
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

									<!-- Enhanced metadata display -->
									{#if logic.getShowAdvanced() && diagnostic.metadata}
										<div class="mt-2 rounded bg-gray-100 p-2 dark:bg-gray-800">
											<div class="text-xs text-gray-600 dark:text-gray-400">
												{#if diagnostic.duration_ms}
													<span class="mr-3"
														>Duration: {logic.formatDuration(diagnostic.duration_ms)}</span
													>
												{/if}
												{#if diagnostic.timestamp}
													<span class="mr-3"
														>Time: {new Date(diagnostic.timestamp).toLocaleTimeString()}</span
													>
												{/if}
												{#if diagnostic.metadata && Object.keys(diagnostic.metadata).length > 0}
													<details class="mt-1">
														<summary class="cursor-pointer text-xs text-blue-600 dark:text-blue-400"
															>Metadata</summary
														>
														<pre
															class="mt-1 text-xs text-gray-500 dark:text-gray-400">{JSON.stringify(
																diagnostic.metadata,
																null,
																2
															)}</pre>
													</details>
												{/if}
											</div>
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
			{#if logic.hasErrors()}
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
				{#if logic.shouldShowQuickTest() && onquicktest}
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

				{#if logic.shouldShowEnhanced() && onenhanced}
					<button
						onclick={() => logic.handleEnhancedTroubleshoot()}
						disabled={state.loading}
						class="rounded-lg border border-purple-200 bg-purple-50 px-4 py-2 text-sm font-medium text-purple-900 transition-colors hover:border-purple-300 hover:bg-purple-100 disabled:opacity-50 dark:border-purple-800 dark:bg-purple-950 dark:text-purple-100 dark:hover:bg-purple-900"
					>
						Enhanced Scan
					</button>
				{/if}

				{#if logic.shouldShowAutoFix() && onautofix}
					<button
						onclick={() => logic.handleAutoFix()}
						disabled={state.loading}
						class="rounded-lg border border-green-200 bg-green-50 px-4 py-2 text-sm font-medium text-green-900 transition-colors hover:border-green-300 hover:bg-green-100 disabled:opacity-50 dark:border-green-800 dark:bg-green-950 dark:text-green-100 dark:hover:bg-green-900"
					>
						Auto Fix
					</button>
				{/if}

				{#if logic.shouldShowRetry()}
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
					class="rounded-lg border border-gray-200 bg-white px-4 py-2 text-sm font-medium text-gray-900 transition-colors hover:border-gray-300 hover:bg-gray-50 disabled:opacity-50 dark:border-gray-800 dark:bg-gray-950 dark:text-gray-100 dark:hover:bg-gray-900"
					disabled={!logic.isCloseable()}
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
