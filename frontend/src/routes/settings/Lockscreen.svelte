<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly, scale } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { unlockScreen, lockscreenState } from '$lib/components/Settings.js';

	let password = $state('');
	let error = $state(false);
	let isUnlocking = $state(false);
	let showPassword = $state(false);

	// Subscribe to lockscreen state - safe for SSR
	let lockscreen = $state({ isLocked: false, isEnabled: false });

	onMount(() => {
		// Subscribe to lockscreen state changes in browser only
		const unsubscribe = lockscreenState.subscribe((state) => {
			lockscreen = state;
		});

		return unsubscribe;
	});

	async function handleUnlock() {
		if (!password) {
			shakeError();
			return;
		}

		isUnlocking = true;

		// Add a small delay for better UX
		await new Promise((resolve) => setTimeout(resolve, 300));

		const success = unlockScreen(password);

		if (!success) {
			shakeError();
			password = '';
			isUnlocking = false;
		} else {
			// Success - component will unmount as lockscreen state changes
			isUnlocking = false;
		}
	}

	function shakeError() {
		error = true;
		setTimeout(() => {
			error = false;
		}, 600);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			handleUnlock();
		}
	}

	// Focus input on mount
	let inputElement: HTMLInputElement | undefined = $state();
	$effect(() => {
		if (inputElement) {
			inputElement.focus();
		}
	});
</script>

<!-- Only show lockscreen if it's enabled and locked -->
{#if lockscreen?.isEnabled && lockscreen?.isLocked}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-white/95 backdrop-blur-xl dark:bg-gray-950/95"
		in:fade={{ duration: 300, easing: cubicOut }}
		out:fade={{ duration: 200, easing: cubicOut }}
	>
		<!-- Background gradient -->
		<div
			class="absolute inset-0 bg-gradient-to-br from-blue-50/50 via-transparent to-purple-50/50 dark:from-blue-950/20 dark:to-purple-950/20"
		></div>

		<!-- Animated orbs -->
		<div class="absolute inset-0 overflow-hidden">
			<div
				class="animate-float absolute -top-20 -left-20 h-96 w-96 rounded-full bg-gradient-to-br from-blue-400/10 to-purple-400/10 blur-3xl"
			></div>
			<div
				class="animate-float-delayed absolute -right-20 -bottom-20 h-96 w-96 rounded-full bg-gradient-to-br from-purple-400/10 to-pink-400/10 blur-3xl"
			></div>
		</div>

		<div
			class="relative w-full max-w-md px-6"
			in:fly={{ y: 20, duration: 400, delay: 100, easing: cubicOut }}
		>
			<div
				class="lockscreen-card relative overflow-hidden rounded-2xl border border-gray-200/50 bg-white/80 p-8 shadow-2xl backdrop-blur-sm dark:border-gray-800/50 dark:bg-gray-900/80 {error
					? 'animate-shake'
					: ''}"
			>
				<!-- Lock icon -->
				<div
					class="mb-6 flex justify-center"
					in:scale={{ duration: 400, delay: 200, easing: cubicOut }}
				>
					<div class="relative">
						<div
							class="absolute inset-0 animate-pulse rounded-full bg-gradient-to-br from-blue-500 to-purple-500 opacity-20 blur-xl"
						></div>
						<div
							class="relative flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-purple-500 text-white shadow-lg"
						>
							<img src="favicon.svg" alt="Icon" class="h-16 w-16" />
						</div>
					</div>
				</div>

				<!-- Title -->
				<div class="mb-6 text-center">
					<h1 class="text-2xl font-semibold text-gray-900 dark:text-gray-100">Welcome back</h1>
					<p class="mt-2 text-sm text-gray-600 dark:text-gray-400">
						Enter your password to continue
					</p>
				</div>

				<!-- Password input -->
				<div class="space-y-4">
					<div class="relative">
						<input
							bind:this={inputElement}
							bind:value={password}
							type={showPassword ? 'text' : 'password'}
							placeholder="Enter password"
							onkeydown={handleKeydown}
							disabled={isUnlocking}
							class="w-full rounded-lg border border-gray-300 bg-white px-4 py-3 pr-12 text-gray-900 placeholder-gray-400 transition-all focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 focus:outline-none disabled:opacity-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-500 dark:focus:border-blue-400 dark:focus:ring-blue-400/20"
						/>

						<!-- Show/hide password button -->
						<button
							type="button"
							onclick={() => (showPassword = !showPassword)}
							class="absolute top-1/2 right-3 -translate-y-1/2 rounded p-1 text-gray-500 transition-colors hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
							disabled={isUnlocking}
						>
							{#if showPassword}
								<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
									/>
								</svg>
							{:else}
								<svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
									/>
									<path
										stroke-linecap="round"
										stroke-linejoin="round"
										stroke-width="2"
										d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
									/>
								</svg>
							{/if}
						</button>
					</div>

					<!-- Error message -->
					{#if error}
						<div
							class="text-center text-sm text-red-600 dark:text-red-400"
							in:fade={{ duration: 200 }}
						>
							Incorrect password. Please try again.
						</div>
					{/if}

					<!-- Unlock button -->
					<button
						onclick={handleUnlock}
						disabled={isUnlocking || !password}
						class="group relative w-full overflow-hidden rounded-lg bg-gradient-to-r from-blue-500 to-purple-500 px-4 py-3 font-medium text-white shadow-lg transition-all hover:shadow-xl disabled:opacity-50"
					>
						<div
							class="absolute inset-0 bg-gradient-to-r from-blue-600 to-purple-600 opacity-0 transition-opacity group-hover:opacity-100"
						></div>
						<span class="relative flex items-center justify-center">
							{#if isUnlocking}
								<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
									<circle
										class="opacity-25"
										cx="12"
										cy="12"
										r="10"
										stroke="currentColor"
										stroke-width="4"
									></circle>
									<path
										class="opacity-75"
										fill="currentColor"
										d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
									></path>
								</svg>
							{:else}
								Unlock
							{/if}
						</span>
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	@keyframes float {
		0%,
		100% {
			transform: translateY(0) rotate(0deg);
		}
		50% {
			transform: translateY(-20px) rotate(10deg);
		}
	}

	@keyframes float-delayed {
		0%,
		100% {
			transform: translateY(0) rotate(0deg);
		}
		50% {
			transform: translateY(-30px) rotate(-10deg);
		}
	}

	.animate-float {
		animation: float 20s ease-in-out infinite;
	}

	.animate-float-delayed {
		animation: float-delayed 25s ease-in-out infinite;
		animation-delay: 5s;
	}

	.animate-shake {
		animation: shake 0.5s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
	}

	@keyframes shake {
		10%,
		90% {
			transform: translateX(-1px);
		}
		20%,
		80% {
			transform: translateX(2px);
		}
		30%,
		50%,
		70% {
			transform: translateX(-4px);
		}
		40%,
		60% {
			transform: translateX(4px);
		}
	}

	.lockscreen-card {
		animation: card-appear 0.6s cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes card-appear {
		from {
			opacity: 0;
			transform: scale(0.95) translateY(10px);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}
</style>
