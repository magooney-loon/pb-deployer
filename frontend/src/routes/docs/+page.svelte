<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import { fly } from 'svelte/transition';
	import DocsSection from './components/DocsSection.svelte';

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

	let currentSection = $state('getting-started');
	let sectionContent: Record<string, string> = $state({});

	// Load markdown content for each section
	async function loadSectionContent(sectionId: string) {
		if (sectionContent[sectionId]) return; // Already loaded

		const section = sections.find((s) => s.id === sectionId);
		if (!section) return;

		try {
			// Fetch the markdown file from static folder
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
				[sectionId]: `# ${section.title}\n\nContent is being loaded...`
			};
		}
	}

	// Handle section change with proper scrolling
	function handleSectionChange(sectionId: string) {
		// Mark as manual scrolling to prevent observer interference
		isManuallyScrolling = true;
		currentSection = sectionId;
		loadSectionContent(sectionId);

		// Small delay to ensure content is rendered
		setTimeout(() => {
			const element = document.getElementById(sectionId);
			if (element) {
				element.scrollIntoView({
					behavior: 'smooth',
					block: 'start'
				});

				// Reset manual scrolling flag after scroll completes
				setTimeout(() => {
					isManuallyScrolling = false;
				}, 1000);
			}
		}, 100);
	}

	// Load initial content
	$effect(() => {
		loadSectionContent(currentSection);
	});

	// Preload all content for better UX
	$effect(() => {
		sections.forEach((section) => {
			loadSectionContent(section.id);
		});
	});

	// Enhanced intersection observer for active section tracking
	let observer: IntersectionObserver;
	let isManuallyScrolling = $state(false);
	let showScrollTop = $state(false);

	$effect(() => {
		if (typeof window !== 'undefined') {
			// Track which sections are currently visible
			const visibleSections = new SvelteMap<string, number>();

			observer = new IntersectionObserver(
				(entries) => {
					entries.forEach((entry) => {
						const sectionId = entry.target.id;

						if (entry.isIntersecting) {
							// Store the intersection ratio for this section
							visibleSections.set(sectionId, entry.intersectionRatio);
						} else {
							// Remove from visible sections
							visibleSections.delete(sectionId);
						}
					});

					// Find the section with the highest intersection ratio
					// or the first visible section if multiple are visible
					let bestSection = '';
					let maxRatio = 0;
					let topMostSection = '';
					let minTop = Infinity;

					for (const [sectionId, ratio] of visibleSections) {
						const element = document.getElementById(sectionId);
						if (element) {
							const rect = element.getBoundingClientRect();

							// Track section with highest intersection ratio
							if (ratio > maxRatio) {
								maxRatio = ratio;
								bestSection = sectionId;
							}

							// Track topmost visible section
							if (rect.top < minTop && rect.top >= -100) {
								minTop = rect.top;
								topMostSection = sectionId;
							}
						}
					}

					// Update current section based on what's most visible
					// Prefer the section with highest intersection ratio, but fall back to topmost
					const targetSection = maxRatio > 0.3 ? bestSection : topMostSection;

					if (targetSection && targetSection !== currentSection && !isManuallyScrolling) {
						currentSection = targetSection;
					}
				},
				{
					threshold: [0, 0.1, 0.3, 0.5, 0.7, 1.0],
					rootMargin: '-100px 0px -200px 0px'
				}
			);

			// Observe all sections after they're rendered
			const observeSections = () => {
				sections.forEach((section) => {
					const element = document.getElementById(section.id);
					if (element) {
						observer.observe(element);
					}
				});
			};

			// Initial observation with delay
			setTimeout(observeSections, 100);

			// Re-observe if content changes
			setTimeout(observeSections, 1000);
		}

		return () => {
			if (observer) observer.disconnect();
		};
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
	<title>Documentation - pb-deployer</title>
	<meta
		name="description"
		content="Complete documentation for pb-deployer - PocketBase deployment tool"
	/>
</svelte:head>

<div class="lg:grid lg:grid-cols-12 lg:gap-12">
	<!-- Sidebar Navigation -->
	<div class="lg:col-span-3">
		<div class="sticky top-8">
			<div class="space-y-6">
				<div>
					<h2 class="mb-4 text-sm font-medium text-gray-900 dark:text-white">Documentation</h2>
					<nav class="space-y-1">
						{#each sections as section (section.id)}
							<button
								onclick={() => handleSectionChange(section.id)}
								class="group relative flex w-full items-center space-x-3 rounded-lg px-3 py-2 text-left text-sm transition-all duration-200
										{currentSection === section.id
									? 'bg-blue-50 font-semibold text-blue-700 dark:bg-blue-950/50 dark:text-blue-300'
									: 'text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800/50 dark:hover:text-gray-100'}"
							>
								{#if currentSection === section.id}
									<div
										class="absolute top-1/2 left-0 h-4 w-0.5 -translate-y-1/2 rounded-full bg-blue-600 dark:bg-blue-400"
									></div>
								{/if}
								<span
									class="text-base transition-all duration-200 {currentSection === section.id
										? 'scale-110 opacity-100'
										: 'opacity-70 group-hover:opacity-100'}"
								>
									{section.icon}
								</span>
								<span class="font-medium">{section.title}</span>
							</button>
						{/each}
					</nav>
				</div>
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
	</div>

	<!-- Main Content -->
	<div class="lg:col-span-9">
		<div class="border-l border-gray-200 lg:pl-12 dark:border-gray-800">
			<div class="space-y-16">
				{#each sections as section (section.id)}
					{#if sectionContent[section.id]}
						<DocsSection
							id={section.id}
							title={section.title}
							icon={section.icon}
							content={sectionContent[section.id]}
						/>
					{/if}
				{/each}
			</div>
		</div>
	</div>
</div>
