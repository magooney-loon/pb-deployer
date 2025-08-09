<script lang="ts">
	import { page } from '$app/state';
	import { themeStore } from '$lib/theme.js';

	function isActive(path: string): boolean {
		return page.url.pathname === path;
	}

	// Mobile menu toggle
	let mobileMenuOpen = $state(false);

	function toggleMobileMenu() {
		mobileMenuOpen = !mobileMenuOpen;
	}

	// Navigation items
	const navItems = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/servers', label: 'Servers', icon: '🖥️' },
		{ href: '/apps', label: 'Applications', icon: '📱' }
	];
</script>

<nav
	class="border-b border-gray-200 bg-white shadow-sm dark:border-gray-700 dark:bg-gray-800 dark:shadow-gray-700"
>
	<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
		<div class="flex h-16 items-center justify-between">
			<!-- Logo and brand -->
			<div class="flex items-center">
				<div class="flex-shrink-0">
					<h1 class="flex items-center text-xl font-bold text-gray-900 dark:text-white">
						<span class="mr-2 text-2xl">⚡</span>
						PB Deployer
					</h1>
				</div>

				<!-- Desktop navigation -->
				<div class="hidden sm:ml-8 sm:flex sm:space-x-1">
					{#each navItems as item (item.href)}
						<a
							href={item.href}
							class="group inline-flex items-center rounded-lg px-3 py-2 text-sm font-medium transition-all duration-200 ease-in-out
							{isActive(item.href)
								? 'bg-blue-50 text-blue-700 shadow-sm dark:bg-blue-900/50 dark:text-blue-300'
								: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700/50 dark:hover:text-white'}"
						>
							<span class="mr-2 text-sm transition-transform duration-200 group-hover:scale-110">
								{item.icon}
							</span>
							{item.label}
							{#if isActive(item.href)}
								<div class="ml-2 h-1.5 w-1.5 rounded-full bg-blue-500 dark:bg-blue-400"></div>
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
						class="inline-flex items-center rounded-full bg-green-50 px-3 py-1 text-xs font-medium text-green-700 ring-1 ring-green-600/20 dark:bg-green-500/10 dark:text-green-400 dark:ring-green-500/20"
					>
						<div class="mr-1.5 h-2 w-2 animate-pulse rounded-full bg-green-500"></div>
						API Online
					</span>
				</div>

				<!-- Theme toggle -->
				<button
					onclick={() => themeStore.toggle()}
					class="rounded-lg p-2.5 text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:outline-none dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200 dark:focus:ring-offset-gray-800"
					title="Toggle theme"
					aria-label="Toggle dark mode"
				>
					<span class="text-lg transition-transform duration-200 hover:scale-110">
						{$themeStore === 'dark' ? '☀️' : '🌙'}
					</span>
				</button>

				<!-- Mobile menu button -->
				<button
					onclick={toggleMobileMenu}
					class="rounded-lg p-2.5 text-gray-500 transition-all duration-200 hover:bg-gray-100 hover:text-gray-700 focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 focus:outline-none sm:hidden dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200 dark:focus:ring-offset-gray-800"
					aria-expanded={mobileMenuOpen}
					aria-label="Toggle mobile menu"
				>
					<span class="text-lg">
						{mobileMenuOpen ? '✕' : '☰'}
					</span>
				</button>
			</div>
		</div>
	</div>

	<!-- Mobile menu -->
	{#if mobileMenuOpen}
		<div
			class="border-t border-gray-200 bg-gray-50 sm:hidden dark:border-gray-700 dark:bg-gray-800/50"
		>
			<div class="space-y-1 px-4 py-3">
				<!-- Mobile API Status -->
				<div
					class="mb-3 flex items-center justify-between border-b border-gray-200 pb-3 dark:border-gray-600"
				>
					<span
						class="inline-flex items-center rounded-full bg-green-50 px-2.5 py-1 text-xs font-medium text-green-700 ring-1 ring-green-600/20 dark:bg-green-500/10 dark:text-green-400 dark:ring-green-500/20"
					>
						<div class="mr-1.5 h-2 w-2 animate-pulse rounded-full bg-green-500"></div>
						API Online
					</span>
				</div>

				{#each navItems as item (item.href)}
					<a
						href={item.href}
						onclick={() => {
							mobileMenuOpen = false;
						}}
						class="group flex items-center rounded-lg px-3 py-3 text-base font-medium transition-all duration-200
						{isActive(item.href)
							? 'bg-blue-50 text-blue-700 shadow-sm dark:bg-blue-900/50 dark:text-blue-300'
							: 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-700/50 dark:hover:text-white'}"
					>
						<span class="mr-3 text-lg transition-transform duration-200 group-hover:scale-110">
							{item.icon}
						</span>
						{item.label}
						{#if isActive(item.href)}
							<div class="ml-auto h-2 w-2 rounded-full bg-blue-500 dark:bg-blue-400"></div>
						{/if}
					</a>
				{/each}
			</div>
		</div>
	{/if}
</nav>
