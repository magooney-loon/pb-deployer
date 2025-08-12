<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/Navigation.svelte';
	import { WarningBanner } from '$lib/components/partials';

	let { children } = $props();
</script>

<div class="min-h-screen bg-white dark:bg-gray-950">
	<WarningBanner size="xs" />
	<Navigation />

	<!-- Main content -->
	<main class="mx-auto px-4 py-8 sm:px-6 lg:px-8">
		{@render children()}
	</main>
</div>
