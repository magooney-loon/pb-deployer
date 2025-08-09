<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/Navigation.svelte';

	let { children } = $props();
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<Navigation />

	<!-- Main content -->
	<main class="mx-auto max-w-7xl py-6 sm:px-6 lg:px-8">
		{@render children()}
	</main>
</div>
