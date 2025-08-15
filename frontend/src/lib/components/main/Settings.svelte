<script lang="ts">
	import { onMount } from 'svelte';
	import { settingsService, type SettingsData, updateLockscreenSettings } from './Settings.js';
	import {
		Card,
		FormField,
		Button,
		LoadingSpinner,
		Toast,
		StatusBadge
	} from '$lib/components/partials';
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	let settings: SettingsData | null = $state(null);
	let loading = $state(true);
	let saving = $state(false);
	let testingConnection = $state(false);
	let error = $state('');
	let successMessage = $state('');
	let connectionTestResult = $state<boolean | null>(null);

	let formData = $state({
		lockscreenEnabled: false,
		autoLockEnabled: false,
		autoLockMinutes: 15,
		notificationsEnabled: false,
		telegramApiKey: '',
		chatId: '',
		notifyOnDeploy: true,
		notifyOnError: true,
		notifyOnServerStatus: false
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
				autoLockMinutes: data.security.autoLockMinutes,
				notificationsEnabled: data.notifications.enabled,
				telegramApiKey: data.notifications.telegramApiKey,
				chatId: data.notifications.chatId,
				notifyOnDeploy: data.notifications.notifyOnDeploy,
				notifyOnError: data.notifications.notifyOnError,
				notifyOnServerStatus: data.notifications.notifyOnServerStatus
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
				},
				notifications: {
					enabled: formData.notificationsEnabled,
					telegramApiKey: formData.telegramApiKey,
					chatId: formData.chatId,
					notifyOnDeploy: formData.notifyOnDeploy,
					notifyOnError: formData.notifyOnError,
					notifyOnServerStatus: formData.notifyOnServerStatus
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

	async function testTelegramConnection() {
		if (!formData.telegramApiKey || !formData.chatId) {
			error = 'Please enter both Telegram API key and Chat ID before testing.';
			return;
		}

		try {
			testingConnection = true;
			error = '';
			connectionTestResult = null;

			const result = await settingsService.testTelegramConnection(
				formData.telegramApiKey,
				formData.chatId
			);

			connectionTestResult = result;

			if (!result) {
				error = 'Connection test failed. Please check your API key and Chat ID format.';
			}
		} catch (err) {
			error = 'Failed to test Telegram connection. Please try again.';
			console.error('Error testing connection:', err);
		} finally {
			testingConnection = false;
		}
	}

	function clearTestResult() {
		connectionTestResult = null;
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

	$effect(() => {
		if (formData.telegramApiKey || formData.chatId) {
			clearTestResult();
		}
	});
</script>

<div class="space-y-8">
	<!-- Page Header -->
	<header>
		<h1 class="text-3xl font-bold text-gray-900 dark:text-gray-100">Settings</h1>
		<p class="mt-2 text-gray-600 dark:text-gray-400">
			Configure security, notifications, and application preferences
		</p>
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

			<!-- Notification Settings Card -->
			<Card
				title="Notification Settings"
				subtitle="Configure Telegram notifications for deployments and system events"
				class="space-y-6"
			>
				<!-- Enable Notifications Toggle -->
				<FormField
					id="notifications-enabled"
					label="Enable Notifications"
					type="checkbox"
					bind:checked={formData.notificationsEnabled}
					helperText="Send notifications for important events via Telegram"
				/>

				{#if formData.notificationsEnabled}
					<div
						in:slide={{ duration: 300, easing: quintOut }}
						out:slide={{ duration: 300, easing: quintOut }}
						class="ml-8 space-y-6 border-l-2 border-green-200 pl-6 dark:border-green-800"
					>
						<div class="rounded-lg bg-green-50 p-6 dark:bg-green-950/20">
							<!-- Telegram Configuration -->
							<div class="space-y-6">
								<div class="space-y-4">
									<FormField
										id="telegram-api-key"
										label="Telegram Bot API Key"
										type="password"
										bind:value={formData.telegramApiKey}
										placeholder="123456789:ABCDEFGHIJK-abcdefghijk"
										helperText="Create a bot via @BotFather on Telegram to get an API key (format: number:letters)"
										required={formData.notificationsEnabled}
									/>

									<FormField
										id="chat-id"
										label="Chat ID"
										bind:value={formData.chatId}
										placeholder="@username or -123456789 or 123456789"
										helperText="Chat ID, username (with @), or group/channel ID where notifications will be sent"
										required={formData.notificationsEnabled}
									/>
								</div>

								<!-- Connection Test Section -->
								<div
									class="rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800"
								>
									<div class="flex items-center justify-between">
										<div>
											<h4 class="text-sm font-medium text-gray-900 dark:text-gray-100">
												Test Connection
											</h4>
											<p class="text-xs text-gray-600 dark:text-gray-400">
												Verify your Telegram configuration
											</p>
										</div>
										<div class="flex items-center gap-3">
											<Button
												type="button"
												variant="outline"
												size="sm"
												loading={testingConnection}
												disabled={!formData.telegramApiKey || !formData.chatId || testingConnection}
												onclick={testTelegramConnection}
											>
												{testingConnection ? 'Testing...' : 'Test Connection'}
											</Button>

											{#if connectionTestResult !== null}
												<StatusBadge
													status={connectionTestResult ? 'Connected' : 'Failed'}
													variant={connectionTestResult ? 'success' : 'error'}
													dot
												/>
											{/if}
										</div>
									</div>
								</div>

								<!-- Notification Types -->
								<div class="space-y-4">
									<h4 class="text-sm font-medium text-gray-900 dark:text-gray-100">
										Notification Types
									</h4>
									<div class="grid gap-4 sm:grid-cols-2">
										<FormField
											id="notify-deploy"
											label="Deployment Notifications"
											type="checkbox"
											bind:checked={formData.notifyOnDeploy}
											helperText="Get notified when deployments start, complete, or fail"
										/>

										<FormField
											id="notify-error"
											label="Error Notifications"
											type="checkbox"
											bind:checked={formData.notifyOnError}
											helperText="Get notified when critical errors occur"
										/>

										<FormField
											id="notify-server-status"
											label="Server Status Notifications"
											type="checkbox"
											bind:checked={formData.notifyOnServerStatus}
											helperText="Get notified when servers go online or offline"
										/>
									</div>
								</div>
							</div>
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
</div>
