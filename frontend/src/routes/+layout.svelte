<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/Navigation.svelte';
	import { WarningBanner } from '$lib/components/partials';
	import { injectViewTransitionStyles } from '$lib/utils/view-transitions';
	import { onMount } from 'svelte';
	import { page } from '$app/state';

	let { children } = $props();

	onMount(() => {
		injectViewTransitionStyles();
	});
</script>

<div class="min-h-screen bg-white dark:bg-gray-950">
	<WarningBanner size="xs" />
	<Navigation />

	<!-- Main content -->
	<main class="mx-auto px-4 py-8 sm:px-6 lg:px-8" style="view-transition-name: main-content">
		<div style="view-transition-name: page-content-{page.route.id}">
			{@render children()}
		</div>
	</main>
</div>
