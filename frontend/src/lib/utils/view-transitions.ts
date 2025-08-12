/**
 * View Transition API utilities for smooth page transitions
 */

// Check if the browser supports View Transition API
export function supportsViewTransitions(): boolean {
	return typeof document !== 'undefined' && 'startViewTransition' in document;
}

// Type for view transition callback
type ViewTransitionCallback = () => void | Promise<void>;

/**
 * Start a view transition with the given callback
 * Falls back to immediate execution if View Transition API is not supported
 */
export function startViewTransition(callback: ViewTransitionCallback): void {
	if (!supportsViewTransitions()) {
		// Fallback for browsers that don't support View Transition API
		callback();
		return;
	}

	// Use the View Transition API with simple callback
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	(document as any).startViewTransition(callback);
}

/**
 * Create a view transition name for an element
 * Useful for creating smooth transitions between specific elements
 */
export function setViewTransitionName(element: HTMLElement, name: string): void {
	if (element && supportsViewTransitions()) {
		element.style.viewTransitionName = name;
	}
}

/**
 * Remove view transition name from an element
 */
export function removeViewTransitionName(element: HTMLElement): void {
	if (element) {
		element.style.viewTransitionName = '';
	}
}

/**
 * CSS helper to add view transition classes
 */
export const viewTransitionStyles = `
	/* Default fade transition for all elements */
	::view-transition-old(root),
	::view-transition-new(root) {
		animation-duration: 0.3s;
		animation-timing-function: ease-in-out;
	}

	/* Slide transition for main content */
	::view-transition-old(main-content) {
		transform: translateX(-100%);
		opacity: 0;
	}

	::view-transition-new(main-content) {
		transform: translateX(0);
		opacity: 1;
	}

	/* Navigation highlight transition */
	::view-transition-old(nav-item),
	::view-transition-new(nav-item) {
		animation-duration: 0.2s;
	}

	/* Page title transition */
	::view-transition-old(page-title),
	::view-transition-new(page-title) {
		animation-duration: 0.4s;
		animation-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
	}

	/* Card transitions */
	::view-transition-old(card),
	::view-transition-new(card) {
		animation-duration: 0.35s;
		animation-timing-function: ease-out;
	}

	/* Reduced motion support */
	@media (prefers-reduced-motion: reduce) {
		::view-transition-old(*),
		::view-transition-new(*) {
			animation-duration: 0.1s !important;
		}
	}
`;

/**
 * Inject view transition styles into the document
 */
export function injectViewTransitionStyles(): void {
	if (typeof document === 'undefined' || !supportsViewTransitions()) {
		return;
	}

	const styleId = 'view-transition-styles';

	// Don't inject if already exists
	if (document.getElementById(styleId)) {
		return;
	}

	const style = document.createElement('style');
	style.id = styleId;
	style.textContent = viewTransitionStyles;
	document.head.appendChild(style);
}
