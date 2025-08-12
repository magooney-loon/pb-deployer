import { themeStore } from '$lib/utils/theme.js';
import { ApiClient } from '$lib/api/client.js';

export interface NavigationItem {
	href: string;
	label: string;
	icon: string;
}

export interface NavigationState {
	mobileMenuOpen: boolean;
	currentPath: string;
	apiStatus: 'online' | 'offline' | 'checking';
}

export class NavigationLogic {
	private state: NavigationState;
	private stateUpdateCallback?: (state: NavigationState) => void;
	private apiClient: ApiClient;

	// Navigation items configuration
	public readonly navItems: NavigationItem[] = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/servers', label: 'Servers', icon: '🖥️' },
		{ href: '/apps', label: 'Applications', icon: '📱' },
		{ href: '/settings', label: 'Settings', icon: '⚙️' }
	];

	constructor(initialPath: string = '/') {
		this.apiClient = new ApiClient();
		this.state = {
			mobileMenuOpen: false,
			currentPath: this.normalizePath(initialPath),
			apiStatus: 'checking'
		};

		// Start API health checking only in browser
		if (typeof window !== 'undefined') {
			this.checkApiHealth();
			this.startHealthChecking();
		}
	}

	public getState(): NavigationState {
		return this.state;
	}

	public onStateUpdate(callback: (state: NavigationState) => void): void {
		this.stateUpdateCallback = callback;
	}

	private updateState(updates: Partial<NavigationState>): void {
		this.state = { ...this.state, ...updates };
		this.stateUpdateCallback?.(this.state);
	}

	public toggleMobileMenu(): void {
		this.updateState({ mobileMenuOpen: !this.state.mobileMenuOpen });
	}

	public closeMobileMenu(): void {
		this.updateState({ mobileMenuOpen: false });
	}

	public updateCurrentPath(path: string): void {
		this.updateState({ currentPath: this.normalizePath(path) });
	}

	public isActive(path: string): boolean {
		const normalizedCurrentPath = this.normalizePath(this.state.currentPath);
		const normalizedNavPath = this.normalizePath(path);
		return normalizedCurrentPath === normalizedNavPath;
	}

	private normalizePath(path: string): string {
		// Remove trailing slash except for root path
		if (path === '/') return path;
		return path.endsWith('/') ? path.slice(0, -1) : path;
	}

	public toggleTheme(): void {
		themeStore.toggle();
	}

	// Helper method to handle navigation item click
	public handleNavItemClick(href: string): void {
		this.updateCurrentPath(href);
		this.closeMobileMenu();
	}

	// API Health checking methods
	public async checkApiHealth(): Promise<void> {
		// Only check API health in browser context
		if (typeof window === 'undefined') {
			return;
		}

		try {
			this.updateState({ apiStatus: 'checking' });

			// Use the existing API client for health check
			await this.apiClient.getHealth();
			this.updateState({ apiStatus: 'online' });
		} catch (error) {
			console.warn('API health check failed:', error);
			this.updateState({ apiStatus: 'offline' });
		}
	}

	private healthCheckInterval?: number;

	private startHealthChecking(): void {
		// Only start health checking in browser context
		if (typeof window === 'undefined') {
			return;
		}

		// Check every 30 seconds
		this.healthCheckInterval = window.setInterval(() => {
			this.checkApiHealth();
		}, 30000);
	}

	public destroy(): void {
		if (typeof window !== 'undefined' && this.healthCheckInterval) {
			clearInterval(this.healthCheckInterval);
		}
	}
}
