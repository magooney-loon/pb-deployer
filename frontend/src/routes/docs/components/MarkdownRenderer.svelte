<script lang="ts">
	import { markdownParser } from './markdown.js';

	interface Props {
		content: string;
		class?: string;
	}

	let { content, class: className = '' }: Props = $props();

	let parsedContent = $derived(markdownParser.parse(content));
</script>

<div class="markdown-content {className}">
	<!-- eslint-disable-next-line svelte/no-at-html-tags -->
	{@html parsedContent}
</div>

<style>
	/* Base styles for markdown content */
	:global(.markdown-content) {
		line-height: 1.7;
		font-size: 1rem;
	}

	/* Ensure proper spacing between elements */
	:global(.markdown-content > *:first-child) {
		margin-top: 0 !important;
	}

	:global(.markdown-content > *:last-child) {
		margin-bottom: 0 !important;
	}

	/* Code block refinements */
	:global(.markdown-content pre) {
		margin: 0;
		border-radius: 0;
	}

	:global(.markdown-content code) {
		font-family:
			'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
	}

	/* List refinements */
	:global(.markdown-content ul ul),
	:global(.markdown-content ol ol) {
		margin-top: 0.5rem;
		margin-bottom: 0.5rem;
		padding-left: 1rem;
	}

	/* Link hover effects */
	:global(.markdown-content a:hover) {
		text-decoration-thickness: 2px;
	}

	/* Table last row border removal */
	:global(.markdown-content table tr:last-child td) {
		border-bottom: none;
	}

	/* Blockquote paragraph margins */
	:global(.markdown-content blockquote p) {
		margin: 0;
	}

	/* Callout paragraph margins */
	:global(.markdown-content .callout p) {
		margin: 0;
	}
</style>
