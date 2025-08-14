export function supportsViewTransitions(): boolean {
	return typeof document !== 'undefined' && 'startViewTransition' in document;
}

type ViewTransitionCallback = () => void | Promise<void>;

/**
 * Inject view transition CSS styles into the document
 */
export function injectViewTransitionStyles(): void {
	if (typeof document === 'undefined') return;

	// Check if styles are already injected
	if (document.getElementById('view-transition-styles')) return;

	const styles = `
		/* Enhanced View Transition API Styles */
		@view-transition {
			navigation: auto;
		}

		/* Enhanced fade transition with subtle depth */
		::view-transition-old(root),
		::view-transition-new(root) {
			animation-duration: 0.3s;
			animation-timing-function: cubic-bezier(0.25, 0.1, 0.25, 1);
			transform-origin: center;
		}

		::view-transition-old(root) {
			animation: enhanced-fade-out 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
		}

		::view-transition-new(root) {
			animation: enhanced-fade-in 0.3s cubic-bezier(0.25, 0.1, 0.25, 1);
		}

		/* Enhanced fade keyframe animations with subtle depth */
		@keyframes enhanced-fade-out {
			0% {
				opacity: 1;
				transform: scale(1) translateY(0);
				filter: blur(0px);
			}
			100% {
				opacity: 0;
				transform: scale(0.98) translateY(-5px);
				filter: blur(1px);
			}
		}

		@keyframes enhanced-fade-in {
			0% {
				opacity: 0;
				transform: scale(1.02) translateY(5px);
				filter: blur(1px);
			}
			100% {
				opacity: 1;
				transform: scale(1) translateY(0);
				filter: blur(0px);
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
				}
				100% {
					opacity: 0;
				}
			}

			@keyframes enhanced-fade-in {
				0% {
					opacity: 0;
				}
				100% {
					opacity: 1;
				}
			}
		}

		/* Hardware acceleration for smooth transitions */
		::view-transition-old(root),
		::view-transition-new(root) {
			backface-visibility: hidden;
			will-change: transform, opacity, filter;
		}
	`;

	const styleElement = document.createElement('style');
	styleElement.id = 'view-transition-styles';
	styleElement.textContent = styles;
	document.head.appendChild(styleElement);
}

/**
 * Start a view transition with the given callback
 * Falls back to immediate execution if View Transition API is not supported
 */
export function startViewTransition(callback: ViewTransitionCallback): void {
	if (!supportsViewTransitions()) {
		callback();
		return;
	}

	// Ensure styles are injected before starting transition
	injectViewTransitionStyles();

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	(document as any).startViewTransition(callback);
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
