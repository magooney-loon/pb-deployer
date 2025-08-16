<script lang="ts">
	import { fly } from 'svelte/transition';
	import { Accordion } from '$lib/components/partials';
	import MarkdownRenderer from './components/MarkdownRenderer.svelte';

	let sections: Array<{
		id: string;
		title: string;
		icon: string;
		file: string;
	}> = $state([]);

	let sectionContent: Record<string, string> = $state({});
	let loadingStates: Record<string, boolean> = $state({});
	let contentReady: Record<string, boolean> = $state({});
	let showScrollTop = $state(false);

	async function loadSectionContent(sectionId: string) {
		if (sectionContent[sectionId] || loadingStates[sectionId]) return;

		const section = sections.find((s) => s.id === sectionId);
		if (!section) return;

		loadingStates = { ...loadingStates, [sectionId]: true };

		try {
			const response = await fetch(`/docs/${section.file}`);
			if (!response.ok) {
				throw new Error(`Failed to fetch ${section.file}: ${response.status}`);
			}
			const content = await response.text();
			sectionContent = { ...sectionContent, [sectionId]: content };
			contentReady = { ...contentReady, [sectionId]: true };
		} catch (error) {
			console.error(`Failed to load content for ${sectionId}:`, error);
			sectionContent = {
				...sectionContent,
				[sectionId]: `# ${section.title}\n\nFailed to load content. Please try again.`
			};
			contentReady = { ...contentReady, [sectionId]: true };
		} finally {
			loadingStates = { ...loadingStates, [sectionId]: false };
		}
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
		try {
			const response = await fetch('/docs/sections.json');
			if (!response.ok) {
				throw new Error(`Failed to fetch sections: ${response.status}`);
			}
			sections = await response.json();
		} catch (error) {
			console.error('Failed to load documentation sections:', error);
			// Fallback to empty array or default sections if needed
			sections = [];
		}
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
	<header class="mb-8">
		<h1 class="mb-2 text-3xl font-bold text-gray-900 dark:text-white">Documentation</h1>
		<p class="text-gray-600 dark:text-gray-400">Complete guide to using pb-deployer</p>
	</header>
	<div class="mx-auto xl:w-2/3">
		<Accordion
			{sections}
			loading={loadingStates}
			{contentReady}
			maxHeight="500px"
			enableScroll={true}
			onSectionOpen={handleSectionOpen}
			onToggle={handleSectionToggle}
		>
			{#snippet children(section)}
				{#if sectionContent[section.id]}
					<MarkdownRenderer content={sectionContent[section.id]} />
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
	</div>
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
