<script lang="ts">
	import ProgressBar from '../partials/ProgressBar.svelte';
	import Background from '../partials/Background.svelte';
	import { splashScreenState } from './SplashScreen';

	let state = $derived($splashScreenState);
</script>

<div class="fixed inset-0 z-50">
	<Background variant="splash" intensity="strong" />

	<div class="relative flex h-full items-center justify-center">
		<div class="flex flex-col items-center space-y-8 px-4">
			<!-- Gopher Logo -->
			<div class="animate-pulse">
				<img src="/favicon.svg" alt="PB Deployer" class="h-24 w-24 sm:h-32 sm:w-32" />
			</div>

			<!-- App Name -->
			<div class="text-center">
				<h1 class="mb-2 text-2xl font-bold text-white sm:text-3xl">PB Deployer</h1>
				<p class="text-sm text-gray-300 sm:text-base">Loading...</p>
			</div>

			<!-- Progress Bar -->
			<div class="w-48 sm:w-64">
				<ProgressBar
					value={state.progress}
					max={100}
					color="blue"
					size="sm"
					animated={true}
					showPercentage={false}
				/>
			</div>
		</div>
	</div>
</div>
