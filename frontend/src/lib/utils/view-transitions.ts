// Firefox slacking..
// https://developer.mozilla.org/en-US/docs/Web/API/View_Transition_API#css.at-rules.view-transition

let isSupported: boolean | null = null;
let stylesInjected = false;

export function supportsViewTransitions(): boolean {
	if (isSupported !== null) {
		return isSupported;
	}

	isSupported = typeof document !== 'undefined' && 'startViewTransition' in document;
	return isSupported;
}

/**
 * Check if user prefers reduced motion
 */
function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined') return false;

	return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

type ViewTransitionCallback = () => void | Promise<void>;

/**
 * Inject view transition CSS styles into the document (cached)
 */
export function injectViewTransitionStyles(): void {
	if (typeof document === 'undefined' || stylesInjected) return;

	// Double-check if styles already exist in DOM
	if (document.getElementById('view-transition-styles')) {
		stylesInjected = true;
		return;
	}

	const styles = `
		/* Enhanced View Transition API Styles */
		@view-transition {
			navigation: auto;
		}

		/* Enhanced fade transition with subtle visual enhancement */
		::view-transition-old(root),
		::view-transition-new(root) {
			animation-duration: 0.45s;
			animation-timing-function: cubic-bezier(0.25, 0.46, 0.45, 0.94);
			transform-origin: center;
		}

		::view-transition-old(root) {
			animation: enhanced-fade-out 0.45s cubic-bezier(0.25, 0.46, 0.45, 0.94);
		}

		::view-transition-new(root) {
			animation: enhanced-fade-in 0.45s cubic-bezier(0.25, 0.46, 0.45, 0.94);
		}

		/* Enhanced fade keyframe animations with subtle brightness effect */
		@keyframes enhanced-fade-out {
			0% {
				opacity: 1;
				filter: brightness(1) saturate(1);
			}
			100% {
				opacity: 0;
				filter: brightness(0.95) saturate(0.9);
			}
		}

		@keyframes enhanced-fade-in {
			0% {
				opacity: 0;
				filter: brightness(1.05) saturate(1.1);
			}
			100% {
				opacity: 1;
				filter: brightness(1) saturate(1);
			}
		}

		/* Reduced motion support - disable transitions completely */
		@media (prefers-reduced-motion: reduce) {
			::view-transition-old(*),
			::view-transition-new(*) {
				animation-duration: 0s !important;
				animation-timing-function: ease-out !important;
			}

			/* No transition for reduced motion */
			@keyframes enhanced-fade-out {
				0%, 100% {
					opacity: 0;
					filter: none;
				}
			}

			@keyframes enhanced-fade-in {
				0%, 100% {
					opacity: 1;
					filter: none;
				}
			}
		}

		/* Hardware acceleration for smooth transitions */
		::view-transition-old(root),
		::view-transition-new(root) {
			backface-visibility: hidden;
			will-change: transform, opacity;
		}

		/* Performance optimizations */
		::view-transition-group(*) {
			contain: layout style paint;
		}
	`;

	const styleElement = document.createElement('style');
	styleElement.id = 'view-transition-styles';
	styleElement.textContent = styles;
	document.head.appendChild(styleElement);
	stylesInjected = true;
}

/**
 * Clean up will-change properties to avoid performance issues
 */
function cleanupWillChange(): void {
	if (typeof document === 'undefined') return;

	const elements = document.querySelectorAll('[style*="will-change"]');
	elements.forEach((el) => {
		const element = el as HTMLElement;
		if (element.style.willChange === 'transform, opacity') {
			element.style.willChange = '';
		}
	});
} /**
 * Start a view transition with the given callback
 * Falls back to immediate execution if View Transition API is not supported
 */
export function startViewTransition(callback: ViewTransitionCallback): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		if (!supportsViewTransitions() || prefersReducedMotion()) {
			Promise.resolve(callback()).then(resolve).catch(reject);
			return;
		}

		try {
			// eslint-disable-next-line @typescript-eslint/no-explicit-any
			const transition = (document as any).startViewTransition(async () => {
				try {
					await callback();
				} catch (error) {
					console.error('View transition callback failed:', error);
					throw error;
				}
			});

			// Handle transition completion
			transition.finished
				.then(() => {
					cleanupWillChange();
					resolve();
				})
				.catch((error: Error) => {
					console.warn('View transition failed:', error);
					cleanupWillChange();
					resolve(); // Still resolve to not break navigation
				});
		} catch (error) {
			console.error('Failed to start view transition:', error);
			// Fallback to immediate execution
			Promise.resolve(callback()).then(resolve).catch(reject);
		}
	});
}

/**
 * Initialize view transitions by injecting styles if supported
 * Call this once when your app starts
 */
export function initViewTransitions(): void {
	if (supportsViewTransitions()) {
		injectViewTransitionStyles();
	}
}

/**
 * Create a named view transition for specific elements
 */
export function createNamedTransition(
	name: string,
	callback: ViewTransitionCallback
): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		if (!supportsViewTransitions() || prefersReducedMotion()) {
			Promise.resolve(callback()).then(resolve).catch(reject);
			return;
		}

		// Cache elements and add view-transition-name
		const elements = Array.from(
			document.querySelectorAll(`[data-transition="${name}"]`)
		) as HTMLElement[];

		if (elements.length === 0) {
			Promise.resolve(callback()).then(resolve).catch(reject);
			return;
		}

		elements.forEach((el, index) => {
			el.style.viewTransitionName = `${name}-${index}`;
		});

		startViewTransition(callback)
			.then(() => {
				// Clean up view-transition-name after transition
				elements.forEach((el) => {
					el.style.viewTransitionName = '';
				});
				resolve();
			})
			.catch((error) => {
				// Clean up on error too
				elements.forEach((el) => {
					el.style.viewTransitionName = '';
				});
				reject(error);
			});
	});
}

/**
 * Check if view transitions are currently running
 */
export function isTransitionRunning(): boolean {
	if (!supportsViewTransitions()) return false;

	try {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const activeTransition = (document as any).activeViewTransition;
		return activeTransition !== null && activeTransition !== undefined;
	} catch {
		return false;
	}
}

/**
 * Skip the current view transition
 */
export function skipTransition(): void {
	if (!supportsViewTransitions()) return;

	try {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const activeTransition = (document as any).activeViewTransition;
		if (activeTransition) {
			activeTransition.skipTransition();
		}
	} catch (error) {
		console.warn('Failed to skip transition:', error);
	}
}
