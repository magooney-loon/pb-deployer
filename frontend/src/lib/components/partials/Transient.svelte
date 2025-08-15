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
			rgba(0, 0, 0, 0.02) 20%,
			rgba(0, 0, 0, 0.08) 40%,
			rgba(0, 0, 0, 0.15) 60%,
			rgba(0, 0, 0, 0.25) 80%,
			rgba(0, 0, 0, 0.4) 100%
		);
		pointer-events: none;
		z-index: 10;
		transition: opacity 0.3s ease;
		padding-bottom: 0.5rem;
		box-sizing: border-box;
	}

	/* Dark mode - subtle gray gradient */
	@media (prefers-color-scheme: dark) {
		.gradient-overlay {
			background: linear-gradient(
				to bottom,
				rgba(0, 0, 0, 0) 0%,
				rgba(20, 20, 20, 0.1) 20%,
				rgba(30, 30, 30, 0.2) 40%,
				rgba(40, 40, 40, 0.3) 60%,
				rgba(50, 50, 50, 0.4) 80%,
				rgba(60, 60, 60, 0.5) 100%
			);
		}
	}

	/* Class-based theming support */
	:global(.dark) .gradient-overlay {
		background: linear-gradient(
			to bottom,
			rgba(0, 0, 0, 0) 0%,
			rgba(20, 20, 20, 0.1) 20%,
			rgba(30, 30, 30, 0.2) 40%,
			rgba(40, 40, 40, 0.3) 60%,
			rgba(50, 50, 50, 0.4) 80%,
			rgba(60, 60, 60, 0.5) 100%
		);
	}

	:global(.light) .gradient-overlay {
		background: linear-gradient(
			to bottom,
			rgba(0, 0, 0, 0) 0%,
			rgba(0, 0, 0, 0.02) 20%,
			rgba(0, 0, 0, 0.08) 40%,
			rgba(0, 0, 0, 0.15) 60%,
			rgba(0, 0, 0, 0.25) 80%,
			rgba(0, 0, 0, 0.4) 100%
		);
	}

	/* Tailwind dark mode support */
	:global(.dark .gradient-overlay) {
		background: linear-gradient(
			to bottom,
			rgba(0, 0, 0, 0) 0%,
			rgba(20, 20, 20, 0.1) 20%,
			rgba(30, 30, 30, 0.2) 40%,
			rgba(40, 40, 40, 0.3) 60%,
			rgba(50, 50, 50, 0.4) 80%,
			rgba(60, 60, 60, 0.5) 100%
		);
	}
</style>
