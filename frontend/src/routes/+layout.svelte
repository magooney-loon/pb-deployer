<script lang="ts">
	import '../app.css';
	import { fade } from 'svelte/transition';
	import Navigation from '$lib/components/main/Navigation.svelte';
	import { WarningBanner, Transient } from '$lib/components/partials';
	import { onMount } from 'svelte';
	import { lockscreenState, lockScreen } from '$lib/components/main/Settings';
	import Lockscreen from './settings/components/Lockscreen.svelte';
	import SplashScreen from '$lib/components/main/SplashScreen.svelte';
	import { splashScreen, splashScreenState } from '$lib/components/main/SplashScreen';

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

	// Add keyboard shortcut to lock screen (Ctrl+L or Cmd+L)
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
	<div>
		<WarningBanner size="xs" />
		<WarningBanner
			size="xs"
			message="Lockscreen Keybind: CTRL+L or CMD+L (if enabled)"
			color="blue"
			icon="ℹ️"
		/>
		<Navigation />
		<main in:fade class="mx-auto px-4 py-8 sm:px-6 lg:px-8">
			<div>
				{@render children()}
			</div>
		</main>
		<Transient />
	</div>
{/if}
