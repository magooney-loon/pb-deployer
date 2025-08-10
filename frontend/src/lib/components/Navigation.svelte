<script lang="ts">
	import { page } from '$app/state';
	import { themeStore } from '$lib/theme.js';
	import { NavigationLogic, type NavigationState } from './logic/Navigation.js';

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

<nav class="border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
		<div class="flex h-16 items-center justify-between">
			<!-- Logo and brand -->
			<div class="flex items-center">
				<div class="flex-shrink-0">
					<h1 class="flex items-center text-xl font-semibold text-gray-900 dark:text-gray-100">
						<span class="mr-2 text-2xl">⚡</span>
						PB Deployer
					</h1>
				</div>

				<!-- Desktop navigation -->
				<div class="hidden sm:ml-8 sm:flex sm:space-x-1">
					{#each logic.navItems as item (item.href)}
						<a
							href={item.href}
							onclick={() => logic.handleNavItemClick(item.href)}
							class="group inline-flex items-center rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200 ease-in-out
							{isActive(item.href)
								? 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100'
								: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100'}"
						>
							<span class="mr-2 text-sm transition-transform duration-200 group-hover:scale-110">
								{item.icon}
							</span>
							{item.label}
							{#if isActive(item.href)}
								<div class="ml-2 h-1.5 w-1.5 rounded-full bg-gray-900 dark:bg-gray-100"></div>
							{/if}
						</a>
					{/each}
				</div>
			</div>

			<!-- Right side items -->
			<div class="flex items-center space-x-4">
				<!-- API Status -->
				<div class="hidden items-center sm:flex">
					<span
						class="inline-flex items-center rounded-full px-3 py-1 text-xs font-medium ring-1
						{state.apiStatus === 'online'
							? 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:ring-emerald-800'
							: state.apiStatus === 'offline'
								? 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-950 dark:text-red-300 dark:ring-red-800'
								: 'bg-yellow-50 text-yellow-700 ring-yellow-200 dark:bg-yellow-950 dark:text-yellow-300 dark:ring-yellow-800'}"
					>
						<div
							class="mr-1.5 h-2 w-2 rounded-full
							{state.apiStatus === 'online'
								? 'animate-pulse bg-emerald-500'
								: state.apiStatus === 'offline'
									? 'bg-red-500'
									: 'animate-pulse bg-yellow-500'}"
						></div>
						{state.apiStatus === 'online'
							? 'API Online'
							: state.apiStatus === 'offline'
								? 'API Offline'
								: 'Checking API...'}
					</span>
				</div>

				<!-- Theme toggle -->
				<button
					onclick={() => logic.toggleTheme()}
					class="rounded-lg p-2.5 text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					title="Toggle theme"
					aria-label="Toggle dark mode"
				>
					<span class="text-lg transition-transform duration-200 hover:scale-110">
						{$themeStore === 'dark' ? '☀️' : '🌙'}
					</span>
				</button>

				<!-- Mobile menu button -->
				<button
					onclick={() => logic.toggleMobileMenu()}
					class="rounded-lg p-2.5 text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none sm:hidden dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-200 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					aria-expanded={state.mobileMenuOpen}
					aria-label="Toggle mobile menu"
				>
					<span class="text-lg">
						{state.mobileMenuOpen ? '✕' : '☰'}
					</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Mobile menu -->
	{#if state.mobileMenuOpen}
		<div
			class="border-t border-gray-200 bg-gray-50 sm:hidden dark:border-gray-800 dark:bg-gray-900"
		>
			<div class="space-y-1 px-4 py-3">
				<!-- Mobile API Status -->
				<div
					class="mb-3 flex items-center justify-between border-b border-gray-200 pb-3 dark:border-gray-700"
				>
					<span
						class="inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium ring-1
						{state.apiStatus === 'online'
							? 'bg-emerald-50 text-emerald-700 ring-emerald-200 dark:bg-emerald-950 dark:text-emerald-300 dark:ring-emerald-800'
							: state.apiStatus === 'offline'
								? 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-950 dark:text-red-300 dark:ring-red-800'
								: 'bg-yellow-50 text-yellow-700 ring-yellow-200 dark:bg-yellow-950 dark:text-yellow-300 dark:ring-yellow-800'}"
					>
						<div
							class="mr-1.5 h-2 w-2 rounded-full
							{state.apiStatus === 'online'
								? 'animate-pulse bg-emerald-500'
								: state.apiStatus === 'offline'
									? 'bg-red-500'
									: 'animate-pulse bg-yellow-500'}"
						></div>
						{state.apiStatus === 'online'
							? 'API Online'
							: state.apiStatus === 'offline'
								? 'API Offline'
								: 'Checking API...'}
					</span>
				</div>

				{#each logic.navItems as item (item.href)}
					<a
						href={item.href}
						onclick={() => logic.handleNavItemClick(item.href)}
						class="group flex items-center rounded-lg px-3 py-3 text-base font-medium transition-all duration-200
						{isActive(item.href)
							? 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100'
							: 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100'}"
					>
						<span class="mr-3 text-lg transition-transform duration-200 group-hover:scale-110">
							{item.icon}
						</span>
						{item.label}
						{#if isActive(item.href)}
							<div class="ml-auto h-2 w-2 rounded-full bg-gray-900 dark:bg-gray-100"></div>
						{/if}
					</a>
				{/each}
			</div>
		</div>
	{/if}
</nav>
