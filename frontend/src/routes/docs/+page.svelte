<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import { fly, slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import MarkdownRenderer from './components/MarkdownRenderer.svelte';

	// Define sections with their metadata
	const sections = [
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
		},
		{
			id: 'troubleshooting',
			title: 'Troubleshooting',
			icon: '🔧',
			file: 'troubleshooting.md'
		}
	];

	let openSections = new SvelteSet(['']);
	let sectionContent: Record<string, string> = $state({});
	let loadingContent: Record<string, boolean> = $state({});
	let showScrollTop = $state(false);

	// Load markdown content for a specific section
	async function loadSectionContent(sectionId: string) {
		if (sectionContent[sectionId] || loadingContent[sectionId]) return;

		const section = sections.find((s) => s.id === sectionId);
		if (!section) return;

		loadingContent = { ...loadingContent, [sectionId]: true };

		try {
			const response = await fetch(`/docs/${section.file}`);
			if (!response.ok) {
				throw new Error(`Failed to fetch ${section.file}: ${response.status}`);
			}
			const content = await response.text();
			sectionContent = { ...sectionContent, [sectionId]: content };
		} catch (error) {
			console.error(`Failed to load content for ${sectionId}:`, error);
			sectionContent = {
				...sectionContent,
				[sectionId]: `# ${section.title}\n\nFailed to load content. Please try again.`
			};
		} finally {
			loadingContent = { ...loadingContent, [sectionId]: false };
		}
	}

	// Toggle section open/closed
	function toggleSection(sectionId: string) {
		if (openSections.has(sectionId)) {
			openSections.delete(sectionId);
		} else {
			openSections.add(sectionId);
			// Load content when opening
			loadSectionContent(sectionId);
		}
	}

	// Load initial content for getting-started
	$effect(() => {
		loadSectionContent('getting-started');
	});

	// Scroll to top functionality
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

	<div class="space-y-4">
		{#each sections as section (section.id)}
			<div
				class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"
			>
				<!-- Section Header -->
				<button
					onclick={() => toggleSection(section.id)}
					class="flex w-full items-center justify-between p-6 text-left transition-colors hover:bg-gray-50 dark:hover:bg-gray-800/50"
				>
					<div class="flex items-center space-x-3">
						<span class="text-xl">{section.icon}</span>
						<h2 class="text-lg font-semibold text-gray-900 dark:text-white">{section.title}</h2>
						{#if loadingContent[section.id]}
							<div
								class="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"
							></div>
						{/if}
					</div>
					<svg
						class="h-5 w-5 text-gray-500 transition-transform duration-200 {openSections.has(
							section.id
						)
							? 'rotate-180'
							: ''}"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M19 9l-7 7-7-7"
						/>
					</svg>
				</button>

				<!-- Section Content -->
				{#if openSections.has(section.id)}
					<div
						class="border-t border-gray-200 dark:border-gray-800"
						transition:slide={{ duration: 300, easing: quintOut }}
					>
						<div class="p-6 pt-4">
							{#if sectionContent[section.id]}
								<MarkdownRenderer content={sectionContent[section.id]} />
							{:else if loadingContent[section.id]}
								<div class="flex items-center justify-center py-8">
									<div class="text-center">
										<div
											class="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"
										></div>
										<p class="text-gray-500 dark:text-gray-400">Loading content...</p>
									</div>
								</div>
							{:else}
								<div class="py-4">
									<p class="text-gray-500 dark:text-gray-400">Content not available</p>
								</div>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		{/each}
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
