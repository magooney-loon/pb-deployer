import { goto } from '$app/navigation';
import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import {
	startViewTransition,
	createNamedTransition,
	isTransitionRunning
} from './view-transitions';

/**
 * Store to track whether animations are enabled
 */
export const animationsEnabled = writable(true);

/**
 * Performance monitoring
 */
let navigationStartTime = 0;
let pendingNavigations = 0;

/**
 * Debounced navigation to prevent rapid-fire calls
 */
const navigationDebounceMap = new Map<string, number>();
const NAVIGATION_DEBOUNCE_MS = 100;

/**
 * Update animation preference from settings
 */
export function updateAnimationPreference(enabled: boolean): void {
	animationsEnabled.set(enabled);
}

/**
 * Get current animation preference asynchronously (non-blocking)
 */
export async function getAnimationPreference(): Promise<boolean> {
	if (!browser) return true;

	try {
		return get(animationsEnabled);
	} catch {
		return true; // Safe fallback
	}
}

/**
 * Get current animation preference synchronously with fallback
 */
export function getAnimationPreferenceSync(): boolean {
	if (!browser) return true;

	try {
		return get(animationsEnabled);
	} catch {
		return true; // Safe fallback
	}
}

/**
 * Check if navigation should be debounced
 */
function shouldDebounceNavigation(url: string): boolean {
	const now = Date.now();
	const lastNavigation = navigationDebounceMap.get(url);

	if (lastNavigation && now - lastNavigation < NAVIGATION_DEBOUNCE_MS) {
		return true;
	}

	navigationDebounceMap.set(url, now);

	// Clean up old entries
	if (navigationDebounceMap.size > 10) {
		const entries = Array.from(navigationDebounceMap.entries());
		const oldEntries = entries.filter(([, time]) => now - time > NAVIGATION_DEBOUNCE_MS * 5);
		oldEntries.forEach(([key]) => navigationDebounceMap.delete(key));
	}

	return false;
}

/**
 * Enhanced navigation function with performance optimizations
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
		force?: boolean;
	} = {}
): Promise<void> {
	const { transitionName, skipIfRunning, force, ...gotoOptions } = options;
	const urlString = typeof url === 'string' ? url : url.toString();

	// Debounce navigation unless forced
	if (!force && shouldDebounceNavigation(urlString)) {
		console.debug('Navigation debounced:', urlString);
		return;
	}

	// Prevent navigation if transition is already running and skipIfRunning is true
	if (skipIfRunning && isTransitionRunning()) {
		console.warn('Navigation skipped: view transition already running');
		return;
	}

	// Prevent too many concurrent navigations
	if (pendingNavigations > 2) {
		console.warn('Too many pending navigations, skipping:', urlString);
		return;
	}

	pendingNavigations++;
	navigationStartTime = performance.now();

	try {
		// Check if animations are enabled (async for better performance)
		const shouldUseAnimations = await getAnimationPreference();

		if (!shouldUseAnimations) {
			// Use regular navigation without transitions
			await goto(url, gotoOptions);
			return;
		}

		// Use requestIdleCallback for better performance if available
		const runTransition = async () => {
			if (transitionName) {
				await createNamedTransition(transitionName, async () => {
					await goto(url, gotoOptions);
				});
			} else {
				await startViewTransition(async () => {
					await goto(url, gotoOptions);
				});
			}
		};

		if ('requestIdleCallback' in window) {
			await new Promise<void>((resolve, reject) => {
				requestIdleCallback(
					async () => {
						try {
							await runTransition();
							resolve();
						} catch (error) {
							reject(error);
						}
					},
					{ timeout: 1000 }
				);
			});
		} else {
			await runTransition();
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
	} finally {
		pendingNavigations = Math.max(0, pendingNavigations - 1);

		// Performance monitoring
		if (navigationStartTime > 0) {
			const duration = performance.now() - navigationStartTime;
			if (duration > 1000) {
				console.warn(`Slow navigation detected: ${duration.toFixed(2)}ms to ${urlString}`);
			}
		}
	}
}

/**
 * Enhanced link click handler with better performance
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
	let isNavigating = false;

	return async (event: MouseEvent) => {
		// Prevent double-click navigation
		if (isNavigating) {
			event.preventDefault();
			return;
		}

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
		isNavigating = true;

		try {
			options.onNavigationStart?.();

			// Use sync check for immediate response
			if (!getAnimationPreferenceSync()) {
				await goto(href, options);
			} else {
				await navigateWithTransition(href, { ...options, force: true });
			}

			options.onNavigationEnd?.();
		} catch (error) {
			const navigationError = error instanceof Error ? error : new Error(String(error));
			options.onNavigationError?.(navigationError);
			console.error('Link navigation failed:', navigationError);
		} finally {
			// Reset navigation state after a small delay
			setTimeout(() => {
				isNavigating = false;
			}, 100);
		}
	};
}

/**
 * Optimized Svelte action for transition links
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
	let handleClick: ((event: MouseEvent) => Promise<void>) | null = null;
	let cleanup: (() => void) | null = null;

	const setupLink = () => {
		if (!currentHref) {
			console.warn('transitionLink action applied to anchor without href');
			return;
		}

		// Remove old listener
		if (handleClick && cleanup) {
			cleanup();
		}

		handleClick = createTransitionLink(currentHref, options);
		node.addEventListener('click', handleClick, { passive: false });

		cleanup = () => {
			if (handleClick) {
				node.removeEventListener('click', handleClick);
			}
		};

		// Add transition attributes only if animations are enabled
		if (getAnimationPreferenceSync()) {
			node.classList.add('transition-link');
			if (options.transitionName) {
				node.setAttribute('data-transition', options.transitionName);
			}
		}
	};

	setupLink();

	return {
		destroy() {
			cleanup?.();
			node.classList.remove('transition-link');
			node.removeAttribute('data-transition');
		},
		update(newOptions: typeof options) {
			Object.assign(options, newOptions);

			const newHref = node.getAttribute('href');
			if (newHref && newHref !== currentHref) {
				currentHref = newHref;
				setupLink();
			}
		}
	};
}

/**
 * Helper function to get the current route for transition naming
 */
export function getRouteTransitionName(pathname: string): string {
	const cleanPath = pathname.replace(/\/$/, '') || 'home';
	return cleanPath.replace(/\//g, '-').replace(/^-/, '');
}

/**
 * Optimized route preloading with error handling
 */
export async function preloadRoute(href: string): Promise<void> {
	if (typeof window === 'undefined') return;

	try {
		const { preloadData } = await import('$app/navigation');
		await Promise.race([
			preloadData(href),
			new Promise((_, reject) => setTimeout(() => reject(new Error('Preload timeout')), 3000))
		]);
	} catch (error) {
		console.warn('Failed to preload route:', href, error);
	}
}

/**
 * Optimized navigation state management
 */
class NavigationState {
	private isNavigating = false;
	private currentTransition: string | null = null;
	private navigationQueue: Array<() => Promise<void>> = [];
	private processing = false;
	private maxQueueSize = 3;

	isCurrentlyNavigating(): boolean {
		return this.isNavigating;
	}

	getCurrentTransition(): string | null {
		return this.currentTransition;
	}

	async queueNavigation(navigationFn: () => Promise<void>): Promise<void> {
		// Drop oldest items if queue is too large
		if (this.navigationQueue.length >= this.maxQueueSize) {
			console.warn('Navigation queue full, dropping oldest items');
			this.navigationQueue.splice(0, this.navigationQueue.length - this.maxQueueSize + 1);
		}

		return new Promise((resolve, reject) => {
			const timeoutId = setTimeout(() => {
				reject(new Error('Navigation timeout'));
			}, 10000);

			this.navigationQueue.push(async () => {
				try {
					clearTimeout(timeoutId);
					await navigationFn();
					resolve();
				} catch (error) {
					clearTimeout(timeoutId);
					reject(error);
				}
			});

			this.processQueue();
		});
	}

	private async processQueue(): Promise<void> {
		if (this.processing || this.navigationQueue.length === 0) {
			return;
		}

		this.processing = true;
		this.isNavigating = true;

		while (this.navigationQueue.length > 0) {
			const nextNavigation = this.navigationQueue.shift();
			if (nextNavigation) {
				try {
					await nextNavigation();
				} catch (error) {
					console.error('Queued navigation failed:', error);
				}
			}
		}

		this.isNavigating = false;
		this.processing = false;
	}

	setTransition(name: string | null): void {
		this.currentTransition = name;
	}

	clear(): void {
		this.navigationQueue.length = 0;
		this.isNavigating = false;
		this.processing = false;
		this.currentTransition = null;
	}
}

export const navigationState = new NavigationState();

/**
 * Navigate with optimized queue management
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
 * Enhanced back navigation with better error handling
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
			const shouldUseAnimations = await getAnimationPreference();
			if (shouldUseAnimations) {
				await startViewTransition(async () => {
					window.history.back();
				});
			} else {
				window.history.back();
			}
		} else if (options.fallbackUrl) {
			await navigateWithTransition(options.fallbackUrl, {
				replaceState: true,
				transitionName: options.transitionName
			});
		}
	} catch (error) {
		console.error('Back navigation failed:', error);
		window.history.back();
	}
}

/**
 * Enhanced forward navigation with better error handling
 */
export async function goForwardWithTransition(): Promise<void> {
	if (typeof window === 'undefined') return;

	try {
		const shouldUseAnimations = await getAnimationPreference();
		if (shouldUseAnimations) {
			await startViewTransition(async () => {
				window.history.forward();
			});
		} else {
			window.history.forward();
		}
	} catch (error) {
		console.error('Forward navigation failed:', error);
		window.history.forward();
	}
}

/**
 * Clear all navigation state (useful for cleanup)
 */
export function clearNavigationState(): void {
	navigationState.clear();
	navigationDebounceMap.clear();
	pendingNavigations = 0;
	navigationStartTime = 0;
}

/**
 * Get navigation performance metrics
 */
export function getNavigationMetrics() {
	return {
		pendingNavigations,
		queueSize: navigationState['navigationQueue']?.length || 0,
		isNavigating: navigationState.isCurrentlyNavigating(),
		currentTransition: navigationState.getCurrentTransition()
	};
}
