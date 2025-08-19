<script lang="ts">
	import Modal from '$lib/components/main/Modal.svelte';
	import { Button, FormField } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import type { App, Version, Deployment } from '$lib/api/index.js';

	interface Props {
		open: boolean;
		deployment: Deployment | null;
		app: App | null;
		version: Version | null;
		deployments: Deployment[];
		deploying?: boolean;
		onclose: () => void;
		ondeploy: (
			deploymentId: string,
			isInitialDeploy: boolean,
			superuserEmail?: string,
			superuserPass?: string
		) => Promise<void>;
	}

	let {
		open,
		deployment,
		app,
		version,
		deployments = [],
		deploying = false,
		onclose,
		ondeploy
	}: Props = $props();

	let superuserEmail = $state('');
	let superuserPassword = $state('');
	let emailError = $state<string | undefined>(undefined);
	let passwordError = $state<string | undefined>(undefined);

	// Check if this is an initial deployment (no successful deployments for this app)
	let isInitialDeployment = $derived(() => {
		if (!app?.id) return false;
		return !deployments.some((d) => d.app_id === app.id && d.status === 'success');
	});

	function validateEmail(email: string): string | undefined {
		if (!email.trim()) {
			return 'Admin email is required for initial deployment';
		}

		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		if (!emailRegex.test(email.trim())) {
			return 'Please enter a valid email address';
		}

		return undefined;
	}

	function validatePassword(password: string): string | undefined {
		if (!password.trim()) {
			return 'Admin password is required for initial deployment';
		}

		if (password.length < 8) {
			return 'Password must be at least 8 characters long';
		}

		return undefined;
	}

	function handleClose() {
		if (!deploying) {
			resetForm();
			onclose();
		}
	}

	function resetForm() {
		superuserEmail = '';
		superuserPassword = '';
		emailError = undefined;
		passwordError = undefined;
	}

	function handleEmailInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		superuserEmail = value;
		emailError = validateEmail(value);
	}

	function handlePasswordInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		superuserPassword = value;
		passwordError = validatePassword(value);
	}

	async function handleDeploy() {
		if (!deployment?.id || deploying) return;

		const isInitial = isInitialDeployment();

		// Validate credentials for initial deployment
		if (isInitial) {
			emailError = validateEmail(superuserEmail);
			passwordError = validatePassword(superuserPassword);

			if (emailError || passwordError) {
				return;
			}
		}

		try {
			await ondeploy(
				deployment.id,
				isInitial,
				isInitial ? superuserEmail : undefined,
				isInitial ? superuserPassword : undefined
			);
		} catch (error) {
			console.error('Deployment failed:', error);
		}
	}

	// Reset form when modal opens/closes
	$effect(() => {
		if (!open) {
			setTimeout(() => {
				resetForm();
			}, 300);
		}
	});

	let modalTitle = $derived(
		!app || !version ? 'Deploy Application' : `Deploy ${app.name} v${version.version_number}`
	);

	let canDeploy = $derived(() => {
		if (deploying || !deployment) return false;

		if (isInitialDeployment()) {
			return !emailError && !passwordError && superuserEmail && superuserPassword;
		}

		return true;
	});
</script>

<Modal {open} title={modalTitle} size="lg" closeable={!deploying} onclose={handleClose}>
	{#if deployment && app && version}
		<div class="space-y-6">
			<!-- Deployment Summary -->
			<div
				class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-gray-700 dark:bg-gray-800"
			>
				<h3 class="mb-3 text-lg font-semibold text-gray-900 dark:text-gray-100">
					Deployment Summary
				</h3>
				<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
					<div>
						<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
							Application
						</div>
						<div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
							{app.name}
						</div>
					</div>
					<div>
						<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
							Version
						</div>
						<div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
							v{version.version_number}
						</div>
					</div>
					<div>
						<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">Domain</div>
						<div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">
							<a
								href="https://{app.domain}"
								target="_blank"
								class="text-blue-600 underline-offset-4 hover:underline dark:text-blue-400"
							>
								{app.domain}
							</a>
						</div>
					</div>
					<div>
						<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
							Deployment Type
						</div>
						<div class="mt-1">
							{#if isInitialDeployment()}
								<span
									class="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200"
								>
									<Icon name="rocket" size="h-3 w-3" class="mr-1" />
									Initial Deployment
								</span>
							{:else}
								<span
									class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"
								>
									<Icon name="refresh" size="h-3 w-3" class="mr-1" />
									Update Deployment
								</span>
							{/if}
						</div>
					</div>
				</div>
				{#if version.notes}
					<div class="mt-4">
						<div class="text-xs font-medium text-gray-500 uppercase dark:text-gray-400">
							Release Notes
						</div>
						<div class="mt-1 text-sm text-gray-900 dark:text-gray-100">
							{version.notes}
						</div>
					</div>
				{/if}
			</div>

			<!-- Initial Deployment Notice -->
			{#if isInitialDeployment()}
				<div
					class="rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-700 dark:bg-blue-900/20"
				>
					<div class="flex items-start space-x-3">
						<div class="flex-shrink-0">
							<Icon name="info" class="text-blue-600 dark:text-blue-400" />
						</div>
						<div class="flex-1">
							<h3 class="font-semibold text-blue-900 dark:text-blue-100">
								Initial Deployment Setup
							</h3>
							<p class="mt-1 text-sm text-blue-800 dark:text-blue-200">
								This is the first deployment for this application. You need to provide admin
								credentials to set up the initial PocketBase admin user.
							</p>
						</div>
					</div>
				</div>

				<!-- Superuser Credentials Form -->
				<div class="space-y-4">
					<div class="border-b border-gray-200 pb-2 dark:border-gray-700">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">Admin User Setup</h3>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							Configure the initial admin user for your PocketBase instance
						</p>
					</div>

					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<FormField
							id="superuser-email"
							label="Admin Email"
							type="email"
							value={superuserEmail}
							placeholder="admin@example.com"
							errorText={emailError}
							required
							disabled={deploying}
							oninput={handleEmailInput}
							helperText="Email for the PocketBase admin user"
						/>

						<FormField
							id="superuser-password"
							label="Admin Password"
							type="password"
							value={superuserPassword}
							placeholder="••••••••"
							errorText={passwordError}
							required
							disabled={deploying}
							oninput={handlePasswordInput}
							helperText="Password for the PocketBase admin user (min. 8 characters)"
						/>
					</div>

					<div
						class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-700 dark:bg-amber-900/20"
					>
						<div class="flex items-start space-x-3">
							<div class="flex-shrink-0">
								<Icon name="warning" class="text-amber-600 dark:text-amber-400" />
							</div>
							<div class="flex-1">
								<h4 class="font-medium text-amber-900 dark:text-amber-100">Security Note</h4>
								<p class="mt-1 text-sm text-amber-800 dark:text-amber-200">
									These credentials will be used to create the initial admin user. Make sure to use
									a strong password and store these credentials securely.
								</p>
							</div>
						</div>
					</div>
				</div>
			{:else}
				<!-- Regular Deployment Confirmation -->
				<div
					class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-700 dark:bg-green-900/20"
				>
					<div class="flex items-start space-x-3">
						<div class="flex-shrink-0">
							<Icon name="check" class="text-green-600 dark:text-green-400" />
						</div>
						<div class="flex-1">
							<h3 class="font-semibold text-green-900 dark:text-green-100">Ready to Deploy</h3>
							<p class="mt-1 text-sm text-green-800 dark:text-green-200">
								This will update your existing application to version {version.version_number}. The
								deployment process will handle the update automatically.
							</p>
						</div>
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<div class="py-8 text-center">
			<div class="text-gray-600 dark:text-gray-400">No deployment data available</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end space-x-3">
			<Button variant="outline" onclick={handleClose} disabled={deploying}>Cancel</Button>
			<Button variant="primary" onclick={handleDeploy} disabled={!canDeploy} loading={deploying}>
				{#snippet iconSnippet()}
					<Icon name="rocket" />
				{/snippet}
				{deploying ? 'Deploying...' : isInitialDeployment() ? 'Deploy & Setup' : 'Deploy'}
			</Button>
		</div>
	{/snippet}
</Modal>
