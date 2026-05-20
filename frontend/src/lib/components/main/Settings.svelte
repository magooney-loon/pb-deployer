<script lang="ts">
	import { onMount } from 'svelte';
	import { settings as settingsStore, type SettingsData } from '$lib/stores/settings.svelte.js';
	import { LoadingSpinner, Toast, SettingsForm } from '$lib/components/partials';

	let settingsData: SettingsData | null = $state(null);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let successMessage = $state('');

	onMount(() => {
		loadSettings();
	});

	function loadSettings() {
		try {
			loading = true;
			error = '';
			settingsStore.initialize();
			settingsData = {
				security: {
					lockscreenEnabled: settingsStore.lockscreen.isEnabled,
					autoLockEnabled: settingsStore.lockscreen.autoLockEnabled,
					autoLockMinutes: settingsStore.lockscreen.autoLockMinutes
				},
				ui: {
					animationsEnabled: settingsStore.ui.animationsEnabled,
					mouseEffectsEnabled: settingsStore.ui.mouseEffectsEnabled
				}
			};
		} catch (err) {
			error = 'Failed to load settings. Please try again.';
			console.error('Error loading settings:', err);
		} finally {
			loading = false;
		}
	}

	function saveSettings(data: {
		lockscreenEnabled: boolean;
		autoLockEnabled: boolean;
		autoLockMinutes: number;
		animationsEnabled: boolean;
		mouseEffectsEnabled: boolean;
	}) {
		try {
			saving = true;
			error = '';
			successMessage = '';

			const {
				lockscreenEnabled,
				autoLockEnabled,
				autoLockMinutes,
				animationsEnabled,
				mouseEffectsEnabled
			} = data;

			const updatedSettings: SettingsData = {
				security: {
					lockscreenEnabled,
					autoLockEnabled,
					autoLockMinutes
				},
				ui: {
					animationsEnabled,
					mouseEffectsEnabled
				}
			};

			settingsStore.updateLockscreen(updatedSettings.security);
			settingsStore.updateUI(updatedSettings.ui);
			settingsData = {
				security: {
					lockscreenEnabled: settingsStore.lockscreen.isEnabled,
					autoLockEnabled: settingsStore.lockscreen.autoLockEnabled,
					autoLockMinutes: settingsStore.lockscreen.autoLockMinutes
				},
				ui: {
					animationsEnabled: settingsStore.ui.animationsEnabled,
					mouseEffectsEnabled: settingsStore.ui.mouseEffectsEnabled
				}
			};

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
{:else if error && !settingsData}
	<Toast message={error} onDismiss={clearError} />
{:else}
	<SettingsForm
		settings={settingsData}
		{loading}
		{saving}
		{error}
		{successMessage}
		onSave={saveSettings}
		onClearError={clearError}
		onClearSuccess={clearSuccess}
	/>
{/if}
