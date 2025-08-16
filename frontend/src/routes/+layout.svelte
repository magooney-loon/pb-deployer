<script lang="ts">
	import '../app.css';
	import { fade } from 'svelte/transition';
	import Navigation from '$lib/components/main/Navigation.svelte';
	import { WarningBanner } from '$lib/components/partials';
	import Icon from '$lib/components/icons/Icon.svelte';
	import { onMount } from 'svelte';
	import { lockscreenState, lockScreen } from '$lib/components/main/Settings';
	import Lockscreen from './settings/components/Lockscreen.svelte';
	import SplashScreen from '$lib/components/main/SplashScreen.svelte';
	import { splashScreen, splashScreenState } from '$lib/components/main/SplashScreen';
	import Mouse from '$lib/utils/Mouse.svelte';

	let { children } = $props();

	let lockscreen = $state({ isLocked: false, isEnabled: false });
	let splashState = $derived($splashScreenState);

	onMount(() => {
		const unsubscribe = lockscreenState.subscribe((state) => {
			lockscreen = state;
		});

		splashScreen.startLoading();

		return () => {
			unsubscribe();
			splashScreen.stopLoading();
		};
	});

	function handleKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'l') {
			e.preventDefault();
			if (lockscreen.isEnabled && !lockscreen.isLocked) {
				lockScreen();
			}
		}
	}

	$effect(() => {
		if (typeof window !== 'undefined') {
			window.addEventListener('keydown', handleKeydown);
			return () => {
				window.removeEventListener('keydown', handleKeydown);
			};
		}
	});
</script>

{#if splashState.isLoading}
	<SplashScreen />
{:else if lockscreen.isEnabled && lockscreen.isLocked}
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
		<WarningBanner
			size="xs"
			message="Lockscreen Keybind: CTRL+L or CMD+L (if enabled)"
			color="blue"
		>
			{#snippet iconSnippet()}
				<Icon name="info" size="h-4 w-4" />
			{/snippet}
		</WarningBanner>
		<Navigation />
		<main in:fade class="mx-auto px-4 py-8 sm:px-6 lg:px-8">
			{@render children()}
		</main>
	</div>
{/if}

<Mouse />
