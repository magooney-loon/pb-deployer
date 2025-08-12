<script lang="ts">
	import Settings from '$lib/components/Settings.svelte';
</script>

<svelte:head>
	<title>Settings</title>
	<meta name="description" content="Configure application settings and preferences" />
</svelte:head>

<div style="view-transition-name: page-title">
	<h1 class="sr-only">Settings</h1>
</div>

<div style="view-transition-name: settings-content">
	<Settings />
</div>
