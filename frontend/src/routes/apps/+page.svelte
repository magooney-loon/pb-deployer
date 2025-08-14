<script lang="ts">
	import AppList from '$lib/components/main/AppList.svelte';
</script>

<svelte:head>
	<title>Applications</title>
	<meta name="description" content="Manage your PocketBase applications" />
</svelte:head>

<div style="view-transition-name: page-title">
	<h1 class="sr-only">Applications</h1>
</div>

<div style="view-transition-name: apps-content">
	<AppList />
</div>
