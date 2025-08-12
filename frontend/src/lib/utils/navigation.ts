import { goto } from '$app/navigation';
import { startViewTransition } from './view-transitions';

/**
 * Enhanced navigation function that uses View Transition API for smooth page transitions
 */
export function navigateWithTransition(
	url: string | URL,
	options: {
		replaceState?: boolean;
		noScroll?: boolean;
		keepFocus?: boolean;
		invalidateAll?: boolean;
		state?: Record<string, unknown>;
	} = {}
): Promise<void> {
	return new Promise((resolve) => {
		startViewTransition(async () => {
			try {
				await goto(url, options);
				resolve();
			} catch (error) {
				console.error('Navigation failed:', error);
				resolve();
			}
		});
	});
}

/**
 * Enhanced link click handler that uses view transitions
 */
export function createTransitionLink(
	href: string,
	options: {
		replaceState?: boolean;
		noScroll?: boolean;
		keepFocus?: boolean;
		invalidateAll?: boolean;
		state?: Record<string, unknown>;
	} = {}
) {
	return (event: MouseEvent) => {
		// Only handle left clicks without modifier keys
		if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
			return;
		}

		// Let the browser handle external links
		if (href.startsWith('http') || href.startsWith('//')) {
			return;
		}

		event.preventDefault();
		navigateWithTransition(href, options);
	};
}

/**
 * Svelte action for automatically adding view transition to links
 */
export function transitionLink(
	node: HTMLAnchorElement,
	options: {
		replaceState?: boolean;
		noScroll?: boolean;
		keepFocus?: boolean;
		invalidateAll?: boolean;
		state?: Record<string, unknown>;
	} = {}
) {
	const href = node.getAttribute('href');

	if (!href) {
		return {};
	}

	const handleClick = createTransitionLink(href, options);

	node.addEventListener('click', handleClick);

	return {
		destroy() {
			node.removeEventListener('click', handleClick);
		},
		update(newOptions: typeof options) {
			// Update options if needed
			Object.assign(options, newOptions);
		}
	};
}

/**
 * Helper function to get the current route for transition naming
 */
export function getRouteTransitionName(pathname: string): string {
	// Clean up pathname and create a consistent name
	const cleanPath = pathname.replace(/\/$/, '') || 'home';
	return cleanPath.replace(/\//g, '-').replace(/^-/, '');
}
