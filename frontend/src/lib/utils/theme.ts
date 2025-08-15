import { writable } from 'svelte/store';
import { browser } from '$app/environment';

type Theme = 'light' | 'dark';

function initTheme(): Theme {
	if (!browser) return 'dark';

	try {
		const stored = localStorage.getItem('theme') as Theme;
		if (stored && (stored === 'light' || stored === 'dark')) {
			return stored;
		}

		if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
			return 'dark';
		}
	} catch (e) {
		// Fallback if localStorage or matchMedia aren't available
		console.warn('Theme initialization failed:', e);
	}

	return 'light';
}

function updateDOM(newTheme: Theme) {
	if (!browser || typeof document === 'undefined') return;
	try {
		document.documentElement.setAttribute('data-theme', newTheme);
	} catch (e) {
		console.warn('Failed to update theme DOM:', e);
	}
}

const theme = writable<Theme>(initTheme());

if (browser) {
	theme.subscribe((newTheme) => {
		updateDOM(newTheme);
	});
}

export const themeStore = {
	subscribe: theme.subscribe,

	set(newTheme: Theme) {
		if (browser) {
			try {
				localStorage.setItem('theme', newTheme);
			} catch (e) {
				console.warn('Failed to save theme to localStorage:', e);
			}
		}
		theme.set(newTheme);
	},

	toggle() {
		theme.update((current) => {
			const newTheme = current === 'dark' ? 'light' : 'dark';
			if (browser) {
				try {
					localStorage.setItem('theme', newTheme);
				} catch (e) {
					console.warn('Failed to save theme to localStorage:', e);
				}
			}
			return newTheme;
		});
	}
};

// Listen for system theme changes
if (browser && typeof window !== 'undefined' && window.matchMedia) {
	try {
		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		mediaQuery.addEventListener('change', (e) => {
			// Only update if user hasn't manually set a preference
			try {
				if (!localStorage.getItem('theme')) {
					const newTheme = e.matches ? 'dark' : 'light';
					theme.set(newTheme);
				}
			} catch (err) {
				console.warn('Failed to check localStorage for theme preference:', err);
			}
		});
	} catch (e) {
		console.warn('Failed to set up system theme listener:', e);
	}
}
