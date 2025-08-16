<script lang="ts">
	import { onMount } from 'svelte';

	interface Position {
		x: number;
		y: number;
	}

	interface TrailDot extends Position {
		id: number;
		targetX: number;
		targetY: number;
		velocityX: number;
		velocityY: number;
		angle: number;
		orbitRadius: number;
	}

	interface Ripple extends Position {
		id: number;
		scale: number;
		opacity: number;
		timestamp: number;
	}

	let mousePos = $state<Position>({ x: 0, y: 0 });
	let trail = $state<TrailDot[]>([]);
	let ripples = $state<Ripple[]>([]);
	let mounted = $state<boolean>(false);
	let isTouchDevice = $state<boolean>(false);
	let animationId = 0;

	const TRAIL_LENGTH: number = 7;
	const SPRING_STIFFNESS: number = 0.18;
	const SPRING_DAMPING: number = 0.09;
	const ORBIT_AMPLITUDE: number = 0.45;
	const RIPPLE_DURATION: number = 270;

	function handleMousemove(event: MouseEvent): void {
		mousePos.x = event.clientX;
		mousePos.y = event.clientY;
	}

	function handleMousedown(event: MouseEvent): void {
		// Create Vercel-style multiple ring effect
		const baseTime = Date.now();
		const newRipples: Ripple[] = [];

		// Create 3 concentric rings with staggered timing
		for (let i = 0; i < 3; i++) {
			newRipples.push({
				id: baseTime + i + Math.random(),
				x: event.clientX,
				y: event.clientY,
				scale: 0,
				opacity: 1,
				timestamp: baseTime + i * 50 // Stagger by 50ms
			});
		}

		ripples = [...ripples, ...newRipples];
	}

	function updateTrail(): void {
		if (!mounted) return;

		const currentTime = Date.now();
		const timeOffset = currentTime * 0.001;

		// Update trail dots - mutate in place for better performance
		for (let i = 0; i < trail.length; i++) {
			const dot = trail[i];
			let targetX, targetY;

			if (i === 0) {
				// First dot follows mouse cursor directly
				targetX = mousePos.x;
				targetY = mousePos.y;
			} else {
				// Each subsequent dot follows the previous dot with subtle offset
				const prevDot = trail[i - 1];
				const offsetX = Math.sin(timeOffset + i) * (dot.orbitRadius * 0.5);
				const offsetY = Math.cos(timeOffset + i) * (dot.orbitRadius * 0.5);
				targetX = prevDot.x + offsetX;
				targetY = prevDot.y + offsetY;
			}

			// Spring physics for smooth following
			const deltaX = targetX - dot.x;
			const deltaY = targetY - dot.y;

			dot.velocityX += deltaX * SPRING_STIFFNESS;
			dot.velocityY += deltaY * SPRING_STIFFNESS;

			dot.velocityX *= SPRING_DAMPING;
			dot.velocityY *= SPRING_DAMPING;

			dot.x += dot.velocityX;
			dot.y += dot.velocityY;
		}

		// Update ripples - filter and update in place
		for (let i = ripples.length - 1; i >= 0; i--) {
			const ripple = ripples[i];
			const elapsed = currentTime - ripple.timestamp;
			const progress = Math.max(0, elapsed / RIPPLE_DURATION);

			if (progress >= 1) {
				ripples.splice(i, 1);
			} else {
				// Vercel-style easing and scaling
				const easeOut = 1 - Math.pow(1 - progress, 3);
				ripple.scale = easeOut * 1.8;
				ripple.opacity = (1 - progress) * (0.4 - progress * 0.3);
			}
		}

		// Ensure we always have trail dots
		while (trail.length < TRAIL_LENGTH) {
			const index = trail.length;
			trail.push({
				id: currentTime + index,
				x: mousePos.x,
				y: mousePos.y,
				targetX: mousePos.x,
				targetY: mousePos.y,
				velocityX: 0,
				velocityY: 0,
				angle: (index / TRAIL_LENGTH) * Math.PI * 2,
				orbitRadius: ORBIT_AMPLITUDE * (index / TRAIL_LENGTH)
			});
		}

		// Trigger reactivity
		trail = trail;
		ripples = ripples;

		animationId = requestAnimationFrame(updateTrail);
	}

	onMount(() => {
		// Detect touch devices and disable mouse effects
		isTouchDevice =
			'ontouchstart' in window ||
			navigator.maxTouchPoints > 0 ||
			/Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);

		if (isTouchDevice) {
			return;
		}

		mounted = true;

		// Listen to mouse events on window with passive listeners for better performance
		window.addEventListener('mousemove', handleMousemove, { passive: true });
		window.addEventListener('mousedown', handleMousedown, { passive: true });

		// Initialize trail
		for (let i = 0; i < TRAIL_LENGTH; i++) {
			trail.push({
				id: Date.now() + i,
				x: window.innerWidth / 2,
				y: window.innerHeight / 2,
				targetX: window.innerWidth / 2,
				targetY: window.innerHeight / 2,
				velocityX: 0,
				velocityY: 0,
				angle: (i / TRAIL_LENGTH) * Math.PI * 2,
				orbitRadius: ORBIT_AMPLITUDE * (i / TRAIL_LENGTH)
			});
		}

		// Start animation loop
		animationId = requestAnimationFrame(updateTrail);

		return () => {
			mounted = false;
			cancelAnimationFrame(animationId);
			window.removeEventListener('mousemove', handleMousemove);
			window.removeEventListener('mousedown', handleMousedown);
		};
	});
</script>

{#if !isTouchDevice}
	<div class="mouse-container">
		<!-- Trail dots -->
		{#each trail as dot, index (dot.id)}
			<div
				class="trail-dot"
				style="
				left: {dot.x - 3}px;
				top: {dot.y - 3}px;
				opacity: {((TRAIL_LENGTH - index) / TRAIL_LENGTH) * 0.45};
				transform: scale({((TRAIL_LENGTH - index) / TRAIL_LENGTH) * 0.45});
			"
			></div>
		{/each}

		<!-- Ripple effects -->
		{#each ripples as ripple (ripple.id)}
			<div
				class="ripple"
				style="
				left: {ripple.x}px;
				top: {ripple.y}px;
				opacity: {ripple.opacity};
				transform: translate(-50%, -50%) scale({ripple.scale});
			"
			></div>
		{/each}
	</div>
{/if}

<style>
	.mouse-container {
		position: fixed;
		top: 0;
		left: 0;
		width: 100vw;
		height: 100vh;
		pointer-events: none;
		z-index: 9999;
	}

	.trail-dot {
		position: absolute;
		width: 6px;
		height: 6px;
		background: linear-gradient(45deg, #3b82f6, #8b5cf6, transparent 69%);
		border-radius: 50%;
		pointer-events: none;
		box-shadow: 0 0 6px rgba(59, 130, 246, 0.3);
	}

	.ripple {
		position: absolute;
		width: 16px;
		height: 16px;
		border: 1px solid rgba(255, 255, 255, 0.6);
		border-radius: 50%;
		pointer-events: none;
		background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 60%);
		box-shadow:
			0 0 0 1px rgba(0, 0, 0, 0.05),
			0 1px 3px rgba(0, 0, 0, 0.1);
	}
</style>
