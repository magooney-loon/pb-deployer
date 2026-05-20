<script lang="ts">
	import '../app.css';
	import { fade } from 'svelte/transition';
	import Navigation from '$lib/components/main/Navigation.svelte';
	import { WarningBanner } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { onMount, onDestroy } from 'svelte';
	import { splash, settings, startRealtime, stopRealtime } from '$lib/stores';
	import Lockscreen from './settings/components/Lockscreen.svelte';
	import SplashScreen from '$lib/components/main/SplashScreen.svelte';
	import Mouse from '$lib/utils/Mouse.svelte';

	let { children } = $props();

	onMount(() => {
		settings.initialize();
		splash.start();
		startRealtime();
	});

	onDestroy(() => {
		stopRealtime();
	});
</script>

{#if splash.isLoading}
	<SplashScreen />
{:else if settings.lockscreen.isEnabled && settings.lockscreen.isLocked}
	<Lockscreen />
{:else}
	<div class="svg-grid relative">
		<WarningBanner
			size="xs"
			message="Always close this application using Ctrl+C to prevent data loss and ensure proper cleanup."
			color="yellow"
		>
			{#snippet iconSnippet()}
				<Icon name="warning" size="h-4 w-4" />
			{/snippet}
		</WarningBanner>
		<Navigation />
		<main in:fade class="mx-auto px-4 py-8 sm:px-6 lg:px-8">
			{@render children()}
		</main>
	</div>
{/if}

<Mouse />
