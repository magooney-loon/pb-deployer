<script lang="ts">
	import '../app.css';
	import Navigation from '$lib/components/main/Navigation.svelte';
	import { WarningBanner } from '$lib/components/partials';

	import { onMount } from 'svelte';
	import { lockscreenState, lockScreen } from '$lib/components/main/Settings';
	import Lockscreen from './settings/components/Lockscreen.svelte';

	let { children } = $props();

	let lockscreen = $state({ isLocked: false, isEnabled: false });

	onMount(() => {
		const unsubscribe = lockscreenState.subscribe((state) => {
			lockscreen = state;
		});

		return () => {
			unsubscribe();
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

{#if lockscreen.isEnabled && lockscreen.isLocked}
	<Lockscreen />
{/if}

<div
	class={lockscreen.isEnabled && lockscreen.isLocked ? 'invisible' : ''}
	aria-hidden={lockscreen.isEnabled && lockscreen.isLocked}
>
	<WarningBanner size="xs" />
	<Navigation />

	<main class="mx-auto px-4 py-8 sm:px-6 lg:px-8">
		<div>
			{@render children()}
		</div>
	</main>
</div>
