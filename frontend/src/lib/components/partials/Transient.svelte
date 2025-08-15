<script lang="ts">
	let { height = '8rem', className = '' } = $props();
</script>

<div class="gradient-overlay {className}" style="--height: {height};"></div>

<style>
	.gradient-overlay {
		position: fixed;
		bottom: 0;
		left: 0;
		right: 0;
		width: 100%;
		height: var(--height);
		background: linear-gradient(
			to bottom,
			rgba(0, 0, 0, 0) 0%,
			rgba(0, 0, 0, 0.03) 10%,
			rgba(0, 0, 0, 0.06) 20%,
			rgba(0, 0, 0, 0.12) 35%,
			rgba(0, 0, 0, 0.2) 50%,
			rgba(0, 0, 0, 0.3) 65%,
			rgba(0, 0, 0, 0.42) 80%,
			rgba(0, 0, 0, 0.55) 90%,
			rgba(0, 0, 0, 0.7) 100%
		);
		pointer-events: none;
		z-index: 10;
		transition: all 0.3s ease;
		padding-bottom: 1rem;
		box-sizing: border-box;
		animation: breathing 8s ease-in-out infinite;
		transform-origin: bottom center;
		background-size: 400% 100%;
	}

	/* Data-theme attribute based theming support */
	:global([data-theme='light']) .gradient-overlay {
		background: linear-gradient(
			to bottom,
			rgba(0, 0, 0, 0) 0%,
			rgba(0, 0, 0, 0.03) 10%,
			rgba(0, 0, 0, 0.06) 20%,
			rgba(0, 0, 0, 0.12) 35%,
			rgba(0, 0, 0, 0.2) 50%,
			rgba(0, 0, 0, 0.3) 65%,
			rgba(0, 0, 0, 0.42) 80%,
			rgba(0, 0, 0, 0.55) 90%,
			rgba(0, 0, 0, 0.7) 100%
		);
	}

	:global([data-theme='dark']) .gradient-overlay {
		background: linear-gradient(
			to bottom,
			rgba(255, 255, 255, 0) 0%,
			rgba(255, 255, 255, 0.005) 10%,
			rgba(255, 255, 255, 0.01) 20%,
			rgba(255, 255, 255, 0.02) 35%,
			rgba(255, 255, 255, 0.03) 50%,
			rgba(255, 255, 255, 0.04) 65%,
			rgba(255, 255, 255, 0.06) 80%,
			rgba(255, 255, 255, 0.08) 90%,
			rgba(255, 255, 255, 0.1) 100%
		) !important;
	}

	/* Subtle looping animations */
	@keyframes breathing {
		0%,
		100% {
			opacity: 1;
			transform: scaleY(1);
		}
		50% {
			opacity: 0.85;
			transform: scaleY(0.98);
		}
	}

	@keyframes wave {
		0%,
		100% {
			background-position: 0% 0%;
		}
		25% {
			background-position: 100% 0%;
		}
		50% {
			background-position: 200% 0%;
		}
		75% {
			background-position: 100% 0%;
		}
	}

	/* Responsive adjustments */
	@media (max-width: 768px) {
		.gradient-overlay {
			height: calc(var(--height) * 0.8);
			padding-bottom: 0.75rem;
		}
	}

	@media (max-width: 480px) {
		.gradient-overlay {
			height: calc(var(--height) * 0.6);
			padding-bottom: 0.5rem;
		}
	}
</style>
