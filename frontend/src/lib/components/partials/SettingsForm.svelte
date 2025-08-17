<script lang="ts">
	import { Card, FormField, Button, Toast } from '$lib/components/partials';
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import type { SettingsData } from '$lib/components/main/Settings.js';

	let {
		settings,
		loading = false,
		saving = false,
		error = '',
		successMessage = '',
		onSave,
		onClearError,
		onClearSuccess
	}: {
		settings: SettingsData | null;
		loading?: boolean;
		saving?: boolean;
		error?: string;
		successMessage?: string;
		onSave?: (data: {
			lockscreenEnabled: boolean;
			autoLockEnabled: boolean;
			autoLockMinutes: number;
			animationsEnabled: boolean;
			mouseEffectsEnabled: boolean;
		}) => void;
		onClearError?: () => void;
		onClearSuccess?: () => void;
	} = $props();

	let formData = $state({
		lockscreenEnabled: false,
		autoLockEnabled: false,
		autoLockMinutes: 15,
		animationsEnabled: true,
		mouseEffectsEnabled: true
	});

	const autoLockOptions = [
		{ value: 5, label: '5 minutes' },
		{ value: 10, label: '10 minutes' },
		{ value: 15, label: '15 minutes' },
		{ value: 30, label: '30 minutes' },
		{ value: 60, label: '1 hour' }
	];

	// Update form data when settings change
	$effect(() => {
		if (settings) {
			formData = {
				lockscreenEnabled: settings.security.lockscreenEnabled,
				autoLockEnabled: settings.security.autoLockEnabled,
				autoLockMinutes: settings.security.autoLockMinutes,
				animationsEnabled: settings.ui.animationsEnabled,
				mouseEffectsEnabled: settings.ui.mouseEffectsEnabled
			};
		}
	});

	// Auto-disable auto-lock when lockscreen is disabled
	$effect(() => {
		if (!formData.lockscreenEnabled) {
			formData.autoLockEnabled = false;
		}
	});

	function handleSubmit(e: Event) {
		e.preventDefault();
		onSave?.({
			lockscreenEnabled: formData.lockscreenEnabled,
			autoLockEnabled: formData.autoLockEnabled,
			autoLockMinutes: formData.autoLockMinutes,
			animationsEnabled: formData.animationsEnabled,
			mouseEffectsEnabled: formData.mouseEffectsEnabled
		});
	}

	function clearError() {
		onClearError?.();
	}

	function clearSuccess() {
		onClearSuccess?.();
	}
</script>

<form onsubmit={handleSubmit} class="space-y-8">
	<!-- Global Messages -->
	{#if error}
		<Toast message={error} onDismiss={clearError} />
	{/if}

	{#if successMessage}
		<Toast message={successMessage} type="success" onDismiss={clearSuccess} />
	{/if}

	<!-- Security Settings Card -->
	<Card
		title="Security Settings"
		subtitle="Configure lockscreen and automatic locking features"
		class="space-y-6"
	>
		<!-- Main Lockscreen Toggle -->
		<FormField
			id="lockscreen-enabled"
			label="Enable Lockscreen Protection"
			type="checkbox"
			bind:checked={formData.lockscreenEnabled}
			helperText="Require authentication to access the application (Default password: 123a)"
		/>

		<!-- Auto-lock Settings (Nested) -->
		{#if formData.lockscreenEnabled}
			<div
				in:slide={{ duration: 300, easing: quintOut }}
				out:slide={{ duration: 300, easing: quintOut }}
				class="ml-8 space-y-6 border-l-2 border-blue-200 pl-6 dark:border-blue-800"
			>
				<div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-950/20">
					<FormField
						id="auto-lock-enabled"
						label="Enable Automatic Locking"
						type="checkbox"
						bind:checked={formData.autoLockEnabled}
						helperText="Automatically lock the application after a period of inactivity"
					/>

					{#if formData.autoLockEnabled}
						<div class="mt-4">
							<FormField
								id="auto-lock-minutes"
								label="Auto-lock After"
								type="select"
								bind:value={formData.autoLockMinutes}
								options={autoLockOptions}
								helperText="Time of inactivity before automatic locking"
							/>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</Card>

	<!-- UI Settings Card -->
	<Card
		title="User Interface Settings"
		subtitle="Configure visual preferences and animations"
		class="space-y-6"
	>
		<div class="space-y-6">
			<FormField
				id="animations-enabled"
				label="Enable Page Animations"
				type="checkbox"
				bind:checked={formData.animationsEnabled}
				helperText="Enable smooth page transitions and animations throughout the application"
			/>

			<FormField
				id="mouse-effects-enabled"
				label="Enable Mouse Effects"
				type="checkbox"
				bind:checked={formData.mouseEffectsEnabled}
				helperText="Enable mouse trail and click ripple effects"
			/>
		</div>
	</Card>

	<!-- Save Button Section -->
	<Card padding="lg" class="border-2 border-dashed border-gray-200 dark:border-gray-700">
		<div class="flex items-center justify-between">
			<div>
				<h3 class="text-lg font-medium text-gray-900 dark:text-gray-100">Save Configuration</h3>
				<p class="text-sm text-gray-600 dark:text-gray-400">
					Apply your changes to update the application settings
				</p>
			</div>
			<Button
				type="submit"
				loading={saving}
				disabled={saving || loading}
				color="blue"
				variant="outline"
				size="lg"
			>
				{saving ? 'Saving...' : 'Save Settings'}
			</Button>
		</div>
	</Card>
</form>
