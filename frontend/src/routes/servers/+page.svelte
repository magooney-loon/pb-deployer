<script lang="ts">
	import ServerList from '$lib/components/ServerList.svelte';
</script>

<svelte:head>
	<title>Servers - pb-deployer</title>
	<meta name="description" content="Manage your deployment servers" />
</svelte:head>

<div style="view-transition-name: page-title">
	<h1 class="sr-only">Servers</h1>
</div>

<div style="view-transition-name: servers-content">
	<ServerList />
</div>
