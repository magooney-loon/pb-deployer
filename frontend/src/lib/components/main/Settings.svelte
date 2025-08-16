<script lang="ts">
	import { onMount } from 'svelte';
	import { settingsService, type SettingsData, updateLockscreenSettings } from './Settings.js';
	import { Card, FormField, Button, LoadingSpinner, Toast } from '$lib/components/partials';
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	let settings: SettingsData | null = $state(null);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let successMessage = $state('');

	let formData = $state({
		lockscreenEnabled: false,
		autoLockEnabled: false,
		autoLockMinutes: 15
	});

	const autoLockOptions = [
		{ value: 5, label: '5 minutes' },
		{ value: 10, label: '10 minutes' },
		{ value: 15, label: '15 minutes' },
		{ value: 30, label: '30 minutes' },
		{ value: 60, label: '1 hour' }
	];

	onMount(async () => {
		await loadSettings();
	});

	async function loadSettings() {
		try {
			loading = true;
			error = '';

			const data = await settingsService.getSettings();
			settings = data;

			formData = {
				lockscreenEnabled: data.security.lockscreenEnabled,
				autoLockEnabled: data.security.autoLockEnabled,
				autoLockMinutes: data.security.autoLockMinutes
			};
		} catch (err) {
			error = 'Failed to load settings. Please try again.';
			console.error('Error loading settings:', err);
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		try {
			saving = true;
			error = '';
			successMessage = '';

			const updatedSettings: Partial<SettingsData> = {
				security: {
					lockscreenEnabled: formData.lockscreenEnabled,
					autoLockEnabled: formData.autoLockEnabled,
					autoLockMinutes: formData.autoLockMinutes
				}
			};

			settings = await settingsService.updateSettings(updatedSettings);

			updateLockscreenSettings({
				lockscreenEnabled: formData.lockscreenEnabled,
				autoLockEnabled: formData.autoLockEnabled,
				autoLockMinutes: formData.autoLockMinutes
			});

			successMessage = 'Settings saved successfully!';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save settings. Please try again.';
			console.error('Error saving settings:', err);
		} finally {
			saving = false;
		}
	}

	function clearError() {
		error = '';
	}

	function clearSuccess() {
		successMessage = '';
	}

	function handleSubmit(e: Event) {
		e.preventDefault();
		saveSettings();
	}

	$effect(() => {
		if (!formData.lockscreenEnabled) {
			formData.autoLockEnabled = false;
		}
	});
</script>

<header class="mb-8 flex items-center justify-between">
	<div>
		<h1 class="text-3xl font-semibold text-gray-900 dark:text-gray-100">Settings</h1>
		<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
			Configure security, notifications, and application preferences
		</p>
	</div>
</header>
{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner text="Loading settings..." size="lg" />
	</div>
{:else if error && !settings}
	<Toast message={error} onDismiss={clearError} />
{:else}
	<!-- Settings Form -->
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
{/if}
