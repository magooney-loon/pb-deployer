<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { themeStore } from '$lib/theme.js';

	let { children } = $props();

	function isActive(path: string): boolean {
		return $page.url.pathname === path;
	}
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<!-- Navigation -->
	<nav class="bg-white shadow dark:bg-gray-800 dark:shadow-gray-700">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 justify-between">
				<div class="flex">
					<div class="flex flex-shrink-0 items-center">
						<h1 class="text-xl font-bold text-gray-900 dark:text-white">PB Deployer</h1>
					</div>
					<div class="hidden sm:ml-6 sm:flex sm:space-x-8">
						<a
							href="/"
							class="border-b-2 border-transparent px-1 py-2 text-sm font-medium whitespace-nowrap text-gray-500 transition-colors hover:border-gray-300 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:text-gray-200
							{isActive('/') ? 'border-blue-500 text-blue-600 dark:text-blue-400' : ''}"
						>
							Dashboard
						</a>
						<a
							href="/servers"
							class="border-b-2 border-transparent px-1 py-2 text-sm font-medium whitespace-nowrap text-gray-500 transition-colors hover:border-gray-300 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:text-gray-200
							{isActive('/servers') ? 'border-blue-500 text-blue-600 dark:text-blue-400' : ''}"
						>
							Servers
						</a>
						<a
							href="/apps"
							class="border-b-2 border-transparent px-1 py-2 text-sm font-medium whitespace-nowrap text-gray-500 transition-colors hover:border-gray-300 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:text-gray-200
							{isActive('/apps') ? 'border-blue-500 text-blue-600 dark:text-blue-400' : ''}"
						>
							Applications
						</a>
					</div>
				</div>
				<div class="hidden space-x-4 sm:ml-6 sm:flex sm:items-center">
					<button
						onclick={() => themeStore.toggle()}
						class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
						title="Toggle theme"
					>
						{$themeStore === 'dark' ? '☀️' : '🌙'}
					</button>
					<div class="text-sm text-gray-500 dark:text-gray-400">
						<span
							class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"
						>
							🟢 API Online
						</span>
					</div>
				</div>
			</div>
		</div>

		<!-- Mobile menu -->
		<div class="sm:hidden">
			<div class="space-y-1 pt-2 pb-3">
				<div class="flex items-center justify-between px-4 pb-2">
					<button
						onclick={() => themeStore.toggle()}
						class="rounded-lg p-2 text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700"
						title="Toggle theme"
					>
						{$themeStore === 'dark' ? '☀️' : '🌙'}
					</button>
					<span
						class="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"
					>
						🟢 API Online
					</span>
				</div>
				<a
					href="/"
					class="block border-l-4 py-2 pr-4 pl-3 text-base font-medium
					{isActive('/')
						? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
						: 'border-transparent text-gray-500 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200'}"
				>
					Dashboard
				</a>
				<a
					href="/servers"
					class="block border-l-4 py-2 pr-4 pl-3 text-base font-medium
					{isActive('/servers')
						? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
						: 'border-transparent text-gray-500 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200'}"
				>
					Servers
				</a>
				<a
					href="/apps"
					class="block border-l-4 py-2 pr-4 pl-3 text-base font-medium
					{isActive('/apps')
						? 'border-blue-500 bg-blue-50 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
						: 'border-transparent text-gray-500 hover:border-gray-300 hover:bg-gray-50 hover:text-gray-700 dark:text-gray-300 dark:hover:border-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-200'}"
				>
					Applications
				</a>
			</div>
		</div>
	</nav>

	<!-- Main content -->
	<main class="mx-auto max-w-7xl py-6 sm:px-6 lg:px-8">
		{@render children()}
	</main>
</div>
