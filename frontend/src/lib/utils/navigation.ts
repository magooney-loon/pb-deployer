import { goto } from '$app/navigation';
import {
	startViewTransition,
	createNamedTransition,
	isTransitionRunning
} from './view-transitions';

/**
 * Enhanced navigation function that uses View Transition API for smooth page transitions
 */
export async function navigateWithTransition(
	url: string | URL,
	options: {
		replaceState?: boolean;
		noScroll?: boolean;
		keepFocus?: boolean;
		invalidateAll?: boolean;
		state?: Record<string, unknown>;
		transitionName?: string;
		skipIfRunning?: boolean;
	} = {}
): Promise<void> {
	const { transitionName, skipIfRunning, ...gotoOptions } = options;

	// Prevent navigation if transition is already running and skipIfRunning is true
	if (skipIfRunning && isTransitionRunning()) {
		console.warn('Navigation skipped: view transition already running');
		return;
	}

	try {
		if (transitionName) {
			await createNamedTransition(transitionName, async () => {
				await goto(url, gotoOptions);
			});
		} else {
			await startViewTransition(async () => {
				await goto(url, gotoOptions);
			});
		}
	} catch (error) {
		console.error('Navigation with transition failed:', error);
		// Fallback to regular navigation
		try {
			await goto(url, gotoOptions);
		} catch (fallbackError) {
			console.error('Fallback navigation also failed:', fallbackError);
			throw fallbackError;
		}
	}
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
		transitionName?: string;
		skipIfRunning?: boolean;
		onNavigationStart?: () => void;
		onNavigationEnd?: () => void;
		onNavigationError?: (error: Error) => void;
	} = {}
) {
	return async (event: MouseEvent) => {
		// Only handle left clicks without modifier keys
		if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
			return;
		}

		// Let the browser handle external links
		if (
			href.startsWith('http') ||
			href.startsWith('//') ||
			href.startsWith('mailto:') ||
			href.startsWith('tel:')
		) {
			return;
		}

		// Let the browser handle hash links on the same page
		if (href.startsWith('#')) {
			return;
		}

		// Check if this is the current page
		if (typeof window !== 'undefined' && window.location.pathname === href) {
			event.preventDefault();
			return;
		}

		event.preventDefault();

		try {
			options.onNavigationStart?.();
			await navigateWithTransition(href, options);
			options.onNavigationEnd?.();
		} catch (error) {
			const navigationError = error instanceof Error ? error : new Error(String(error));
			options.onNavigationError?.(navigationError);
			console.error('Link navigation failed:', navigationError);
		}
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
		transitionName?: string;
		skipIfRunning?: boolean;
		onNavigationStart?: () => void;
		onNavigationEnd?: () => void;
		onNavigationError?: (error: Error) => void;
	} = {}
) {
	let currentHref = node.getAttribute('href');

	if (!currentHref) {
		console.warn('transitionLink action applied to anchor without href');
		return {};
	}

	let handleClick = createTransitionLink(currentHref, options);

	node.addEventListener('click', handleClick);

	// Add visual feedback for transitions
	const addTransitionClass = () => {
		node.classList.add('transition-link');
		if (options.transitionName) {
			node.setAttribute('data-transition', options.transitionName);
		}
	};

	const removeTransitionClass = () => {
		node.classList.remove('transition-link');
		node.removeAttribute('data-transition');
	};

	addTransitionClass();

	return {
		destroy() {
			node.removeEventListener('click', handleClick);
			removeTransitionClass();
		},
		update(newOptions: typeof options) {
			// Remove old event listener
			node.removeEventListener('click', handleClick);

			// Update options
			Object.assign(options, newOptions);

			// Update href if it changed
			const newHref = node.getAttribute('href');
			if (newHref && newHref !== currentHref) {
				currentHref = newHref;
			}

			// Create new event listener with updated options
			if (currentHref) {
				handleClick = createTransitionLink(currentHref, options);
				node.addEventListener('click', handleClick);
			}

			// Update transition attributes
			removeTransitionClass();
			addTransitionClass();
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

/**
 * Preload a route for faster transitions
 */
export async function preloadRoute(href: string): Promise<void> {
	if (typeof window === 'undefined') return;

	try {
		// Use SvelteKit's built-in preloading
		const { preloadData } = await import('$app/navigation');
		await preloadData(href);
	} catch (error) {
		console.warn('Failed to preload route:', href, error);
	}
}

/**
 * Navigation state management
 */
class NavigationState {
	private isNavigating = false;
	private currentTransition: string | null = null;
	private navigationQueue: Array<() => Promise<void>> = [];

	isCurrentlyNavigating(): boolean {
		return this.isNavigating;
	}

	getCurrentTransition(): string | null {
		return this.currentTransition;
	}

	async queueNavigation(navigationFn: () => Promise<void>): Promise<void> {
		return new Promise((resolve, reject) => {
			this.navigationQueue.push(async () => {
				try {
					await navigationFn();
					resolve();
				} catch (error) {
					reject(error);
				}
			});

			this.processQueue();
		});
	}

	private async processQueue(): Promise<void> {
		if (this.isNavigating || this.navigationQueue.length === 0) {
			return;
		}

		this.isNavigating = true;
		const nextNavigation = this.navigationQueue.shift();

		if (nextNavigation) {
			try {
				await nextNavigation();
			} catch (error) {
				console.error('Queued navigation failed:', error);
			}
		}

		this.isNavigating = false;

		// Process next item in queue
		if (this.navigationQueue.length > 0) {
			setTimeout(() => this.processQueue(), 0);
		}
	}

	setTransition(name: string | null): void {
		this.currentTransition = name;
	}
}

export const navigationState = new NavigationState();

/**
 * Navigate with queue management to prevent overlapping transitions
 */
export async function navigateWithQueue(
	url: string | URL,
	options: Parameters<typeof navigateWithTransition>[1] = {}
): Promise<void> {
	return navigationState.queueNavigation(async () => {
		navigationState.setTransition(options.transitionName || null);
		try {
			await navigateWithTransition(url, options);
		} finally {
			navigationState.setTransition(null);
		}
	});
}

/**
 * Enhanced back navigation with transitions
 */
export async function goBackWithTransition(
	options: {
		transitionName?: string;
		fallbackUrl?: string;
	} = {}
): Promise<void> {
	if (typeof window === 'undefined') return;

	try {
		if (window.history.length > 1) {
			await startViewTransition(async () => {
				window.history.back();
			});
		} else if (options.fallbackUrl) {
			await navigateWithTransition(options.fallbackUrl, {
				replaceState: true,
				transitionName: options.transitionName
			});
		}
	} catch (error) {
		console.error('Back navigation with transition failed:', error);
		// Fallback to regular back navigation
		window.history.back();
	}
}

/**
 * Enhanced forward navigation with transitions
 */
export async function goForwardWithTransition(): Promise<void> {
	if (typeof window === 'undefined') return;

	try {
		await startViewTransition(async () => {
			window.history.forward();
		});
	} catch (error) {
		console.error('Forward navigation with transition failed:', error);
		// Fallback to regular forward navigation
		window.history.forward();
	}
}
