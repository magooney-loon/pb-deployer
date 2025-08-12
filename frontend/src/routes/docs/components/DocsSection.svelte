<script lang="ts">
	import MarkdownRenderer from './MarkdownRenderer.svelte';

	interface Props {
		id: string;
		title: string;
		icon: string;
		content: string;
		class?: string;
	}

	let { id, title, icon, content, class: className = '' }: Props = $props();
</script>

<section {id} class="scroll-mt-20 {className}">
	<div class="space-y-8">
		<!-- Section Header -->
		<div class="flex items-center space-x-3 border-b border-gray-100 pb-6 dark:border-gray-800">
			<span class="text-2xl opacity-80">{icon}</span>
			<h2 class="text-3xl font-bold tracking-tight text-black dark:text-white">{title}</h2>
		</div>

		<!-- Content -->
		<div class="max-w-none">
			<MarkdownRenderer {content} />
		</div>
	</div>
</section>
