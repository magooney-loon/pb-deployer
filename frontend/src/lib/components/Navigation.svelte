<script lang="ts">
	import { page } from '$app/state';
	import { themeStore } from '$lib/theme.js';
	import { NavigationLogic, type NavigationState } from './Navigation.js';

	// Create logic instance
	const logic = new NavigationLogic(page.url.pathname);
	let state = $state<NavigationState>(logic.getState());

	// Update state when logic changes
	logic.onStateUpdate((newState) => {
		state = newState;
	});

	// Update path when page changes
	$effect(() => {
		logic.updateCurrentPath(page.url.pathname);
	});

	// Helper function to check if a path is active using reactive state
	function isActive(path: string): boolean {
		const normalizePath = (p: string) => (p === '/' ? p : p.endsWith('/') ? p.slice(0, -1) : p);
		return normalizePath(state.currentPath) === normalizePath(path);
	}
</script>

<nav
	class="sticky top-0 z-50 border-b border-gray-200/50 bg-white/80 backdrop-blur-lg dark:border-gray-800/50 dark:bg-gray-950/80"
>
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
		<div class="flex h-14 items-center justify-between">
			<!-- Logo and brand -->
			<div class="flex items-center">
				<div class="flex-shrink-0">
					<a href="/" class="group flex items-center space-x-2">
						<img
							alt="pb-deployer logo"
							src="/favicon.svg"
							class="logo-float h-7 w-7 transition-transform group-hover:scale-110"
						/>
						<span class="text-lg font-medium text-gray-900 dark:text-gray-100">pb-deployer</span>
					</a>
				</div>

				<!-- Desktop navigation -->
				<div class="ml-8 hidden sm:flex sm:items-center sm:space-x-1">
					{#each logic.navItems as item (item.href)}
						<a
							href={item.href}
							onclick={() => logic.handleNavItemClick(item.href)}
							class="relative flex items-center space-x-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-all duration-150
							{isActive(item.href)
								? 'text-gray-900 dark:text-gray-100'
								: 'text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100'}"
						>
							<span class="text-sm">{item.icon}</span>
							<span>{item.label}</span>
							{#if isActive(item.href)}
								<div
									class="absolute -bottom-3 left-1/2 h-0.5 w-6 -translate-x-1/2 rounded-full bg-black dark:bg-white"
								></div>
							{/if}
						</a>
					{/each}
				</div>
			</div>

			<!-- Right side items -->
			<div class="flex items-center space-x-3">
				<!-- API Status -->
				<div class="hidden items-center sm:flex">
					<div
						class="flex items-center space-x-1.5 rounded-full border px-2.5 py-1 text-xs font-medium
						{state.apiStatus === 'online'
							? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-950/50 dark:text-emerald-300'
							: state.apiStatus === 'offline'
								? 'border-red-200 bg-red-50 text-red-700 dark:border-red-800/50 dark:bg-red-950/50 dark:text-red-300'
								: 'border-yellow-200 bg-yellow-50 text-yellow-700 dark:border-yellow-800/50 dark:bg-yellow-950/50 dark:text-yellow-300'}"
					>
						<div
							class="h-1.5 w-1.5 rounded-full
							{state.apiStatus === 'online'
								? 'bg-emerald-500'
								: state.apiStatus === 'offline'
									? 'bg-red-500'
									: 'animate-pulse bg-yellow-500'}"
						></div>
						<span>
							{state.apiStatus === 'online'
								? 'Online'
								: state.apiStatus === 'offline'
									? 'Offline'
									: 'Checking...'}
						</span>
					</div>
				</div>

				<!-- Theme toggle -->
				<button
					onclick={() => logic.toggleTheme()}
					class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-600 transition-all duration-150 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-gray-100 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					title="Toggle theme"
					aria-label="Toggle dark mode"
				>
					<span class="text-sm">
						{$themeStore === 'dark' ? '☀️' : '🌙'}
					</span>
				</button>

				<!-- Mobile menu button -->
				<button
					onclick={() => logic.toggleMobileMenu()}
					class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-600 transition-all duration-150 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none sm:hidden dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-gray-100 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					aria-expanded={state.mobileMenuOpen}
					aria-label="Toggle mobile menu"
				>
					<span
						class="text-sm transition-transform duration-150 {state.mobileMenuOpen
							? 'rotate-90'
							: ''}"
					>
						{state.mobileMenuOpen ? '✕' : '☰'}
					</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Mobile menu -->
	{#if state.mobileMenuOpen}
		<div
			class="border-t border-gray-200/50 bg-white/95 backdrop-blur-lg sm:hidden dark:border-gray-800/50 dark:bg-gray-950/95"
		>
			<div class="px-4 py-3">
				<!-- Mobile API Status -->
				<div
					class="mb-3 flex items-center justify-center border-b border-gray-200/50 pb-3 dark:border-gray-800/50"
				>
					<div
						class="flex items-center space-x-1.5 rounded-full border px-2.5 py-1 text-xs font-medium
						{state.apiStatus === 'online'
							? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800/50 dark:bg-emerald-950/50 dark:text-emerald-300'
							: state.apiStatus === 'offline'
								? 'border-red-200 bg-red-50 text-red-700 dark:border-red-800/50 dark:bg-red-950/50 dark:text-red-300'
								: 'border-yellow-200 bg-yellow-50 text-yellow-700 dark:border-yellow-800/50 dark:bg-yellow-950/50 dark:text-yellow-300'}"
					>
						<div
							class="h-1.5 w-1.5 rounded-full
							{state.apiStatus === 'online'
								? 'bg-emerald-500'
								: state.apiStatus === 'offline'
									? 'bg-red-500'
									: 'animate-pulse bg-yellow-500'}"
						></div>
						<span>
							{state.apiStatus === 'online'
								? 'API Online'
								: state.apiStatus === 'offline'
									? 'API Offline'
									: 'Checking API...'}
						</span>
					</div>
				</div>

				<!-- Mobile navigation items -->
				<div class="space-y-1">
					{#each logic.navItems as item (item.href)}
						<a
							href={item.href}
							onclick={() => logic.handleNavItemClick(item.href)}
							class="group flex items-center space-x-2.5 rounded-md px-3 py-2.5 text-sm font-medium transition-all duration-150
							{isActive(item.href)
								? 'bg-gray-100 text-gray-900 dark:bg-gray-800/50 dark:text-gray-100'
								: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-900/50 dark:hover:text-gray-100'}"
						>
							<span class="text-base">{item.icon}</span>
							<span class="flex-1">{item.label}</span>
							{#if isActive(item.href)}
								<div class="h-1.5 w-1.5 rounded-full bg-black dark:bg-white"></div>
							{/if}
						</a>
					{/each}
				</div>
			</div>
		</div>
	{/if}
</nav>

<style>
	@keyframes float {
		0%,
		100% {
			transform: translateY(0px) scale(1);
		}
		33% {
			transform: translateY(-2px) scale(1.02);
		}
		66% {
			transform: translateY(1px) scale(0.98);
		}
	}

	.logo-float {
		animation: float 3s ease-in-out infinite;
	}
</style>
