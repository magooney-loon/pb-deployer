import { themeStore } from '$lib/utils/theme.js';

export interface NavigationItem {
	href: string;
	label: string;
	icon: string;
}

export interface NavigationState {
	mobileMenuOpen: boolean;
	currentPath: string;
}

export class NavigationLogic {
	private state: NavigationState;
	private stateUpdateCallback?: (state: NavigationState) => void;

	public readonly navItems: NavigationItem[] = [
		{ href: '/', label: 'Dashboard', icon: '📊' },
		{ href: '/servers', label: 'Servers', icon: '🖥️' },
		{ href: '/apps', label: 'Applications', icon: '📱' },
		{ href: '/settings', label: 'Settings', icon: '⚙️' }
	];

	constructor(initialPath: string = '/') {
		this.state = {
			mobileMenuOpen: false,
			currentPath: this.normalizePath(initialPath)
		};
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

	public handleNavItemClick(href: string): void {
		this.updateCurrentPath(href);
		this.closeMobileMenu();
	}
}
