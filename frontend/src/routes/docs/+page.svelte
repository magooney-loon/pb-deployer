<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import { fly } from 'svelte/transition';
	import { Accordion } from '$lib/components/partials';

	let sections: Array<{
		id: string;
		title: string;
		icon: string;
		file: string;
	}> = $state([]);

	let openSections = new SvelteSet();
	let sectionContent: Record<string, string> = $state({});
	let loadingStates: Record<string, boolean> = $state({});
	let contentReady: Record<string, boolean> = $state({});
	let showScrollTop = $state(false);

	async function loadSectionContent(sectionId: string) {
		if (sectionContent[sectionId] || loadingStates[sectionId]) return;

		const section = sections.find((s) => s.id === sectionId);
		if (!section) return;

		loadingStates = { ...loadingStates, [sectionId]: true };

		// Simulate loading delay
		await new Promise((resolve) => setTimeout(resolve, 800));

		// Generate simple test content
		const content = `This is the content for ${section.title}.

This section contains information about ${section.title.toLowerCase()}.

Key points:
• Point 1 for ${section.title}
• Point 2 for ${section.title}
• Point 3 for ${section.title}

Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.

Example code or configuration would go here.`;

		sectionContent = { ...sectionContent, [sectionId]: content };
		contentReady = { ...contentReady, [sectionId]: true };
		loadingStates = { ...loadingStates, [sectionId]: false };
	}

	function handleSectionOpen(sectionId: string) {
		loadSectionContent(sectionId);
	}

	function handleSectionToggle(sectionId: string, isOpen: boolean) {
		if (isOpen) {
			loadSectionContent(sectionId);
		}
	}

	$effect(() => {
		loadSections();
	});

	async function loadSections() {
		// Use simple test sections instead of fetching
		sections = [
			{
				id: 'getting-started',
				title: 'Getting Started',
				icon: '🚀',
				file: 'getting-started.md'
			},
			{
				id: 'installation',
				title: 'Installation',
				icon: '📦',
				file: 'installation.md'
			},
			{
				id: 'configuration',
				title: 'Configuration',
				icon: '⚙️',
				file: 'configuration.md'
			},
			{
				id: 'deployment',
				title: 'Deployment',
				icon: '🌐',
				file: 'deployment.md'
			},
			{
				id: 'api-reference',
				title: 'API Reference',
				icon: '📚',
				file: 'api-reference.md'
			}
		];
	}

	$effect(() => {
		if (typeof window !== 'undefined') {
			const handleScroll = () => {
				showScrollTop = window.scrollY > 300;
			};

			window.addEventListener('scroll', handleScroll);
			return () => window.removeEventListener('scroll', handleScroll);
		}
	});

	function scrollToTop() {
		window.scrollTo({
			top: 0,
			behavior: 'smooth'
		});
	}
</script>

<svelte:head>
	<title>Documentation</title>
	<meta
		name="description"
		content="Complete documentation for pb-deployer - PocketBase deployment tool"
	/>
</svelte:head>

<!-- Accordion-style Documentation -->
<div class="mx-auto">
	<div class="mb-8">
		<h1 class="mb-2 text-3xl font-bold text-gray-900 dark:text-white">Documentation</h1>
		<p class="text-gray-600 dark:text-gray-400">Complete guide to using pb-deployer</p>
	</div>

	<Accordion
		{sections}
		{openSections}
		loading={loadingStates}
		{contentReady}
		onSectionOpen={handleSectionOpen}
		onToggle={handleSectionToggle}
	>
		{#snippet children(section)}
			{#if sectionContent[section.id]}
				<div class="prose prose-gray dark:prose-invert max-w-none">
					<pre class="text-sm whitespace-pre-wrap">{sectionContent[section.id]}</pre>
				</div>
			{:else}
				<div class="py-4">
					<p class="text-gray-500 dark:text-gray-400">Content not available</p>
				</div>
			{/if}
		{/snippet}

		{#snippet loadingContent()}
			<div class="flex items-center justify-center py-8">
				<div class="text-center">
					<div
						class="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"
					></div>
					<p class="text-gray-500 dark:text-gray-400">Loading content...</p>
				</div>
			</div>
		{/snippet}
	</Accordion>

	<!-- Scroll to Top Button - Vercel Style -->
	{#if showScrollTop}
		<button
			onclick={scrollToTop}
			class="group fixed right-6 bottom-6 z-50 h-10 w-10 rounded-lg border border-gray-200 bg-white/80 text-gray-600 shadow-sm backdrop-blur-sm transition-all duration-200 hover:border-gray-300 hover:bg-white hover:text-gray-900 hover:shadow-md focus:ring-2 focus:ring-gray-200 focus:ring-offset-2 focus:outline-none dark:border-gray-800 dark:bg-gray-900/80 dark:text-gray-400 dark:hover:border-gray-700 dark:hover:bg-gray-900 dark:hover:text-gray-100 dark:focus:ring-gray-700"
			aria-label="Scroll to top"
			in:fly={{ y: 20, duration: 300 }}
			out:fly={{ y: 20, duration: 200 }}
		>
			<svg
				class="mx-auto h-4 w-4 transition-transform duration-200 group-hover:-translate-y-0.5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
				xmlns="http://www.w3.org/2000/svg"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M12 19V5m-7 7l7-7 7 7"
				/>
			</svg>
		</button>
	{/if}
</div>
