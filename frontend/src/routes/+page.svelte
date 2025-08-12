<script lang="ts">
	import Dashboard from '$lib/components/Dashboard.svelte';
</script>

<svelte:head>
	<title>Dashboard</title>
	<meta name="description" content="PocketBase deployment dashboard" />
</svelte:head>

<div style="view-transition-name: page-title">
	<h1 class="sr-only">Dashboard</h1>
</div>

<div style="view-transition-name: dashboard-content">
	<Dashboard />
</div>
