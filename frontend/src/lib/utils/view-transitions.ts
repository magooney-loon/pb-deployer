let isSupported: boolean | null = null;
let stylesInjected = false;

export function supportsViewTransitions(): boolean {
	if (isSupported !== null) {
		return isSupported;
	}

	isSupported = typeof document !== 'undefined' && 'startViewTransition' in document;
	return isSupported;
}

type ViewTransitionCallback = () => void | Promise<void>;

/**
 * Inject view transition CSS styles into the document (cached)
 */
export function injectViewTransitionStyles(): void {
	if (typeof document === 'undefined' || stylesInjected) return;

	const styles = `
		/* Enhanced View Transition API Styles */
		@view-transition {
			navigation: auto;
		}

		/* Enhanced fade transition with subtle depth */
		::view-transition-old(root),
		::view-transition-new(root) {
			animation-duration: 0.45s;
			animation-timing-function: cubic-bezier(0.25, 0.1, 0.25, 1);
			transform-origin: center;
		}

		::view-transition-old(root) {
			animation: enhanced-fade-out 0.45s cubic-bezier(0.25, 0.1, 0.25, 1);
		}

		::view-transition-new(root) {
			animation: enhanced-fade-in 0.45s cubic-bezier(0.25, 0.1, 0.25, 1);
		}

		/* Enhanced fade keyframe animations with subtle depth */
		@keyframes enhanced-fade-out {
			0% {
				opacity: 1;
				transform: scale(1);
			}
			100% {
				opacity: 0;
				transform: scale(0.98);
			}
		}

		@keyframes enhanced-fade-in {
			0% {
				opacity: 0;
				transform: scale(1.02);
			}
			100% {
				opacity: 1;
				transform: scale(1);
			}
		}

		/* Reduced motion support */
		@media (prefers-reduced-motion: reduce) {
			::view-transition-old(*),
			::view-transition-new(*) {
				animation-duration: 0.1s !important;
				animation-timing-function: ease-out !important;
			}

			/* Simple fade for reduced motion */
			@keyframes enhanced-fade-out {
				0% {
					opacity: 1;
					transform: none;
				}
				100% {
					opacity: 0;
					transform: none;
				}
			}

			@keyframes enhanced-fade-in {
				0% {
					opacity: 0;
					transform: none;
				}
				100% {
					opacity: 1;
					transform: none;
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
 * Start a view transition with the given callback
 * Falls back to immediate execution if View Transition API is not supported
 */
export function startViewTransition(callback: ViewTransitionCallback): Promise<void> {
	return new Promise<void>((resolve, reject) => {
		if (!supportsViewTransitions()) {
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
				.then(() => resolve())
				.catch((error: Error) => {
					console.warn('View transition failed:', error);
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
		if (!supportsViewTransitions()) {
			Promise.resolve(callback()).then(resolve).catch(reject);
			return;
		}

		// Add view-transition-name to elements that should animate
		const elements = document.querySelectorAll(`[data-transition="${name}"]`);
		elements.forEach((el, index) => {
			(el as HTMLElement).style.viewTransitionName = `${name}-${index}`;
		});

		startViewTransition(callback)
			.then(() => {
				// Clean up view-transition-name after transition
				elements.forEach((el) => {
					(el as HTMLElement).style.viewTransitionName = '';
				});
				resolve();
			})
			.catch(reject);
	});
}

/**
 * Check if view transitions are currently running
 */
export function isTransitionRunning(): boolean {
	if (!supportsViewTransitions()) return false;

	try {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		return (document as any).activeViewTransition !== null;
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
