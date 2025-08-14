<script lang="ts">
	import { page } from '$app/state';
	import { slide } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';
	import { themeStore } from '$lib/utils/theme.js';
	import { NavigationLogic, type NavigationState } from './Navigation.js';
	import { transitionLink, getRouteTransitionName } from '$lib/utils/navigation';
	import { StatusBadge, getApiStatusBadge } from '$lib/components/partials';

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

	// Derived API status badge - reactive to state changes
	let apiStatusBadge = $derived(getApiStatusBadge(state.apiStatus));
</script>

<nav
	class="top-0 border-b border-gray-200/50 bg-white/80 backdrop-blur-lg dark:border-gray-800/50 dark:bg-gray-950/80"
>
	<div class="mx-auto px-4 sm:px-6 lg:px-8">
		<div class="flex h-14 items-center justify-between">
			<!-- Logo and brand -->
			<div class="flex items-center">
				<div class="flex-shrink-0">
					<a
						href="/"
						class="group flex items-center space-x-2"
						use:transitionLink
						style="view-transition-name: nav-logo"
					>
						<img
							alt="pb-deployer logo"
							src="/favicon.svg"
							class="logo-float h-7 w-7 transition-transform group-hover:scale-110"
						/>
						<span class="text-lg font-medium text-gray-900 dark:text-gray-100">pb-deployer</span>
					</a>
				</div>

				<!-- Desktop navigation -->
				<div class="relative ml-8 hidden sm:flex sm:items-center sm:space-x-1">
					{#each logic.navItems as item (item.href)}
						<a
							href={item.href}
							onclick={() => logic.handleNavItemClick(item.href)}
							use:transitionLink
							style="view-transition-name: nav-item-{getRouteTransitionName(item.href)}"
							class="nav-link relative flex items-center space-x-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-all duration-200
							{isActive(item.href)
								? 'text-gray-900 dark:text-gray-100'
								: 'text-gray-600 hover:bg-gray-50/50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800/30 dark:hover:text-gray-100'}"
						>
							<span
								class="text-sm transition-transform duration-200 {isActive(item.href)
									? 'scale-110'
									: ''}">{item.icon}</span
							>
							<span class="relative">
								{item.label}
								<div
									class="nav-indicator absolute -bottom-3 left-1/2 h-0.5 rounded-full bg-black transition-all duration-300 ease-out dark:bg-white
									{isActive(item.href) ? 'w-full -translate-x-1/2 opacity-100' : 'w-0 -translate-x-1/2 opacity-0'}"
								></div>
							</span>
						</a>
					{/each}
				</div>
			</div>

			<!-- Right side items -->
			<div class="flex items-center space-x-3">
				<!-- API Status -->
				<div class="hidden items-center sm:flex">
					<StatusBadge
						status={apiStatusBadge.text}
						variant={apiStatusBadge.variant}
						dot
						size="sm"
					/>
				</div>

				<!-- GitHub link -->
				<a
					href="https://github.com/magooney-loon/pb-deployer"
					target="_blank"
					rel="noopener noreferrer"
					class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-600 transition-all duration-150 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-gray-100 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					title="View on GitHub"
					aria-label="View project on GitHub"
				>
					<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
						<path
							d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
						/>
					</svg>
				</a>

				<!-- Docs link -->
				<a
					href="/docs"
					use:transitionLink
					class="flex h-8 w-8 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-600 transition-all duration-150 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-900 focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 focus:outline-none dark:border-gray-800 dark:bg-gray-950 dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-gray-100 dark:focus:ring-gray-100 dark:focus:ring-offset-gray-950"
					title="Documentation"
					aria-label="View documentation"
				>
					<svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
						/>
					</svg>
				</a>

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
			out:slide={{ duration: 100, easing: cubicOut, axis: 'y' }}
			in:slide={{ duration: 300, easing: cubicOut, axis: 'y' }}
			class="border-t border-gray-200/50 bg-white/95 backdrop-blur-lg sm:hidden dark:border-gray-800/50 dark:bg-gray-950/95"
		>
			<div class="px-4 py-3">
				<!-- Mobile API Status and Icons -->
				<div
					class="mb-3 flex items-center justify-center space-x-3 border-b border-gray-200/50 pb-3 dark:border-gray-800/50"
				>
					<!-- API Status -->
					<StatusBadge
						status="API {apiStatusBadge.text}"
						variant={apiStatusBadge.variant}
						dot
						size="sm"
					/>
				</div>

				<!-- Mobile navigation items -->
				<div class="space-y-1">
					{#each logic.navItems as item (item.href)}
						<a
							href={item.href}
							onclick={() => logic.handleNavItemClick(item.href)}
							use:transitionLink
							style="view-transition-name: mobile-nav-item-{getRouteTransitionName(item.href)}"
							class="group relative flex items-center space-x-2.5 overflow-hidden rounded-md px-3 py-2.5 text-sm font-medium transition-all duration-200
							{isActive(item.href)
								? 'bg-gray-100 text-gray-900 dark:bg-gray-800/50 dark:text-gray-100'
								: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-900/50 dark:hover:text-gray-100'}"
						>
							<span
								class="text-base transition-transform duration-200 {isActive(item.href)
									? 'scale-110'
									: ''}">{item.icon}</span
							>
							<span class="flex-1">{item.label}</span>
							<div class="flex items-center space-x-2">
								<div
									class="h-1.5 w-1.5 rounded-full bg-black transition-all duration-300 dark:bg-white
									{isActive(item.href) ? 'scale-100 opacity-100' : 'scale-0 opacity-0'}"
								></div>
							</div>
							{#if isActive(item.href)}
								<div
									class="absolute top-0 left-0 h-full w-1 bg-black transition-all duration-300 ease-out dark:bg-white"
								></div>
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

	/* Smooth nav indicator animations */
	.nav-link:hover .nav-indicator {
		width: 50%;
		opacity: 0.3;
	}

	/* Prevent layout shift during transitions */
	.nav-indicator {
		will-change: width, opacity, transform;
	}
</style>
