<script lang="ts">
	import { SvelteSet } from 'svelte/reactivity';
	import { slide } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';

	export interface AccordionSection {
		id: string;
		title: string;
		icon?: string;
		disabled?: boolean;
		[key: string]: unknown;
	}

	let {
		sections = [] as AccordionSection[],
		openSections = new SvelteSet<string>(),
		multiple = true,
		disabled = false,
		loading = {},
		contentReady = {},
		animationDuration = { in: 400, out: 300 },
		maxHeight = '400px',
		enableScroll = true,
		class: className = '',
		sectionClass = '',
		headerClass = '',
		contentClass = '',
		onToggle,
		onSectionOpen,
		onSectionClose,
		children,
		loadingContent
	}: {
		sections?: AccordionSection[];
		openSections?: SvelteSet<string>;
		multiple?: boolean;
		disabled?: boolean;
		loading?: Record<string, boolean>;
		contentReady?: Record<string, boolean>;
		animationDuration?: { in: number; out: number };
		maxHeight?: string;
		enableScroll?: boolean;
		class?: string;
		sectionClass?: string;
		headerClass?: string;
		contentClass?: string;
		onToggle?: (sectionId: string, isOpen: boolean) => void;
		onSectionOpen?: (sectionId: string) => void;
		onSectionClose?: (sectionId: string) => void;
		children?: import('svelte').Snippet<
			[section: AccordionSection, isOpen: boolean, isLoading: boolean]
		>;
		loadingContent?: import('svelte').Snippet<[section: AccordionSection]>;
	} = $props();

	function toggleSection(section: AccordionSection) {
		if (disabled || section.disabled) return;

		const isCurrentlyOpen = openSections.has(section.id);

		if (isCurrentlyOpen) {
			openSections.delete(section.id);
			onSectionClose?.(section.id);
			onToggle?.(section.id, false);
		} else {
			// If not allowing multiple sections, close others
			if (!multiple) {
				openSections.clear();
			}

			openSections.add(section.id);
			onSectionOpen?.(section.id);
			onToggle?.(section.id, true);
		}

		// Trigger reactivity
		openSections = openSections;
	}

	function handleKeydown(event: KeyboardEvent, section: AccordionSection) {
		if ((event.key === 'Enter' || event.key === ' ') && !disabled && !section.disabled) {
			event.preventDefault();
			toggleSection(section);
		}
	}

	// Base component styles
	const containerClasses = $derived(`space-y-4 ${className}`);

	const baseSectionClasses =
		'overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900';

	const baseHeaderClasses =
		'flex w-full items-center justify-between p-6 text-left transition-colors focus:outline-none';

	const baseContentClasses = 'border-t border-gray-200 dark:border-gray-800';

	const getHeaderClasses = (section: AccordionSection) => {
		let classes = baseHeaderClasses;

		if (disabled || section.disabled) {
			classes += ' cursor-not-allowed opacity-50';
		} else {
			classes += ' cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/50';
		}

		return `${classes} ${headerClass}`;
	};

	const getSectionClasses = () => {
		return `${baseSectionClasses} ${sectionClass}`;
	};

	const getContentClasses = () => {
		let classes = baseContentClasses;
		if (enableScroll) {
			classes += ` overflow-y-auto`;
		}
		return `${classes} ${contentClass}`;
	};
</script>

<div class={containerClasses}>
	{#each sections as section (section.id)}
		{#if true}
			{@const isOpen = openSections.has(section.id)}
			{@const isLoading = loading[section.id] || false}
			{@const isContentReady = contentReady[section.id] || false}

			<div class={getSectionClasses()}>
				<!-- Section Header -->
				<button
					type="button"
					class={getHeaderClasses(section)}
					disabled={disabled || section.disabled}
					onclick={() => toggleSection(section)}
					onkeydown={(e) => handleKeydown(e, section)}
					aria-expanded={isOpen}
					aria-controls="accordion-content-{section.id}"
				>
					<div class="flex items-center space-x-3">
						{#if section.icon}
							<span class="text-xl">{section.icon}</span>
						{/if}
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">{section.title}</h3>
						{#if isLoading}
							<div
								class="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"
								aria-label="Loading"
							></div>
						{/if}
					</div>

					<!-- Chevron icon -->
					<svg
						class="h-5 w-5 text-gray-500 transition-transform duration-200 {isOpen
							? 'rotate-180'
							: ''}"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
						aria-hidden="true"
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
				{#if isOpen}
					<div
						id="accordion-content-{section.id}"
						in:slide={{ duration: animationDuration.in, easing: quintOut }}
						out:slide={{ duration: animationDuration.out, easing: quintOut }}
						class={getContentClasses()}
						style={enableScroll ? `max-height: ${maxHeight}` : ''}
					>
						<div class="p-6 pt-4">
							{#if isLoading && !isContentReady && loadingContent}
								<!-- Custom loading content -->
								{@render loadingContent(section)}
							{:else if isLoading && !isContentReady}
								<!-- Default loading state -->
								<div class="flex items-center justify-center py-8">
									<div class="text-center">
										<div
											class="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"
										></div>
										<p class="text-gray-500 dark:text-gray-400">Loading content...</p>
									</div>
								</div>
							{:else if children}
								<!-- Main content -->
								{@render children(section, isOpen, isLoading)}
							{:else}
								<!-- Fallback content -->
								<p class="text-gray-500 dark:text-gray-400">No content available</p>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	{/each}
</div>
