<script lang="ts">
	let {
		variant = 'default',
		intensity = 'medium'
	}: {
		variant?: 'default' | 'splash' | 'lockscreen';
		intensity?: 'subtle' | 'medium' | 'strong';
	} = $props();

	// Intensity styles
	const intensityStyles = {
		subtle: 'bg-gray-950/90 backdrop-blur-sm',
		medium: 'bg-gray-950/95 backdrop-blur-xl',
		strong: 'bg-gray-950/98 backdrop-blur-2xl'
	};

	// Variant-specific gradient overlays
	const variantStyles = {
		default: 'from-gray-900/30 via-transparent to-gray-800/20',
		splash: 'from-blue-950/10 via-transparent to-gray-900/20',
		lockscreen: 'from-gray-900/20 via-transparent to-blue-950/10'
	};

	let backgroundClass = $derived(`${intensityStyles[intensity]}`);
	let gradientClass = $derived(`${variantStyles[variant]}`);
</script>

<!-- Main backdrop -->
<div class="fixed inset-0 {backgroundClass}">
	<!-- Subtle gradient overlay -->
	<div class="absolute inset-0 bg-gradient-to-br {gradientClass}"></div>

	<!-- Minimal floating elements -->
	<div class="absolute inset-0 overflow-hidden">
		<!-- Top left orb -->
		<div
			class="animate-float-slow bg-gradient-radial absolute -top-32 -left-32 h-64 w-64 rounded-full from-gray-800/5 to-transparent blur-3xl"
		></div>

		<!-- Bottom right orb -->
		<div
			class="animate-float-slower bg-gradient-radial absolute -right-32 -bottom-32 h-80 w-80 rounded-full from-gray-700/5 to-transparent blur-3xl"
		></div>

		<!-- Center accent -->
		<div
			class="animate-pulse-slow bg-gradient-radial absolute top-1/2 left-1/2 h-96 w-96 -translate-x-1/2 -translate-y-1/2 rounded-full from-blue-900/3 to-transparent blur-3xl"
		></div>
	</div>

	<!-- Subtle noise texture -->
	<div class="absolute inset-0 opacity-[0.015] mix-blend-overlay">
		<div
			class="h-full w-full"
			style="background-image: url('data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iNjAiIGhlaWdodD0iNjAiIHZpZXdCb3g9IjAgMCA2MCA2MCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48ZyBmaWxsPSJub25lIiBmaWxsLXJ1bGU9ImV2ZW5vZGQiPjxnIGZpbGw9IiNmZmYiIGZpbGwtb3BhY2l0eT0iMC4xIj48Y2lyY2xlIGN4PSIyIiBjeT0iMiIgcj0iMiIvPjxjaXJjbGUgY3g9IjE4IiBjeT0iMTgiIHI9IjEiLz48Y2lyY2xlIGN4PSI0MiIgY3k9IjQyIiByPSIxIi8+PC9nPjwvZz48L3N2Zz4='); background-size: 60px 60px;"
		></div>
	</div>
</div>

<style>
	@keyframes float-slow {
		0%,
		100% {
			transform: translateY(0px) translateX(0px) rotate(0deg);
		}
		33% {
			transform: translateY(-10px) translateX(5px) rotate(1deg);
		}
		66% {
			transform: translateY(5px) translateX(-3px) rotate(-0.5deg);
		}
	}

	@keyframes float-slower {
		0%,
		100% {
			transform: translateY(0px) translateX(0px) rotate(0deg);
		}
		50% {
			transform: translateY(-15px) translateX(8px) rotate(-1deg);
		}
	}

	@keyframes pulse-slow {
		0%,
		100% {
			opacity: 0.3;
			transform: translate(-50%, -50%) scale(1);
		}
		50% {
			opacity: 0.1;
			transform: translate(-50%, -50%) scale(1.1);
		}
	}

	.animate-float-slow {
		animation: float-slow 40s ease-in-out infinite;
	}

	.animate-float-slower {
		animation: float-slower 50s ease-in-out infinite;
		animation-delay: 10s;
	}

	.animate-pulse-slow {
		animation: pulse-slow 30s ease-in-out infinite;
		animation-delay: 5s;
	}

	.bg-gradient-radial {
		background: radial-gradient(circle, var(--tw-gradient-stops));
	}
</style>
