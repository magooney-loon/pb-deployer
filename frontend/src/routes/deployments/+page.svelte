<script lang="ts">
	import DeploymentsList from '$lib/components/main/DeploymentsList.svelte';
</script>

<svelte:head>
	<title>Deployments</title>
	<meta name="description" content="Manage your production deployments" />
</svelte:head>

<div style="view-transition-name: page-title">
	<h1 class="sr-only">Deployments</h1>
</div>

<div style="view-transition-name: deployments-content">
	<DeploymentsList />
</div>
