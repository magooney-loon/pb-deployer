import { writable, type Writable } from 'svelte/store';
import { browser } from '$app/environment';
import { updateAnimationPreference } from '$lib/utils/navigation.js';

const LOCKSCREEN_PASSWORD = '123a';
const STORAGE_KEY_PASSWORD = 'pb-deployer-lockscreen-password';

export interface SettingsData {
	security: SecuritySettings;
	ui: UISettings;
}

export interface SecuritySettings {
	lockscreenEnabled: boolean;
	autoLockEnabled: boolean;
	autoLockMinutes: number;
}

export interface UISettings {
	animationsEnabled: boolean;
}

const defaultSettings: SettingsData = {
	security: {
		lockscreenEnabled: false,
		autoLockEnabled: false,
		autoLockMinutes: 15
	},
	ui: {
		animationsEnabled: true
	}
};

export class SettingsService {
	private readonly STORAGE_KEY = 'pb-deployer-settings';

	private getStoredSettings(): SettingsData {
		if (!browser) return defaultSettings;

		try {
			const stored = localStorage.getItem(this.STORAGE_KEY);
			if (stored) {
				const parsed = JSON.parse(stored);
				// Merge with defaults to ensure all properties exist
				return {
					security: { ...defaultSettings.security, ...parsed.security },
					ui: { ...defaultSettings.ui, ...parsed.ui }
				};
			}
		} catch (error) {
			console.error('Failed to parse stored settings:', error);
		}

		return defaultSettings;
	}

	private saveSettings(settings: SettingsData): void {
		if (!browser) return;

		try {
			localStorage.setItem(this.STORAGE_KEY, JSON.stringify(settings));
		} catch (error) {
			console.error('Failed to save settings:', error);
			throw new Error('Failed to save settings to localStorage');
		}
	}

	async getSettings(): Promise<SettingsData> {
		await new Promise((resolve) => setTimeout(resolve, 100));
		return this.getStoredSettings();
	}

	async updateSettings(newSettings: Partial<SettingsData>): Promise<SettingsData> {
		await new Promise((resolve) => setTimeout(resolve, 200));

		const currentSettings = this.getStoredSettings();

		const updatedSettings: SettingsData = {
			security: { ...currentSettings.security, ...(newSettings.security || {}) },
			ui: { ...currentSettings.ui, ...(newSettings.ui || {}) }
		};

		this.saveSettings(updatedSettings);
		return updatedSettings;
	}
}

export const settingsService = new SettingsService();

/**
 * Update animation preference in navigation store when settings change
 */
export function updateUISettings(settings: SettingsData) {
	updateAnimationPreference(settings.ui.animationsEnabled);
}

interface LockscreenState {
	isLocked: boolean;
	isEnabled: boolean;
	autoLockEnabled: boolean;
	autoLockMinutes: number;
	lastActivity: number;
}

class LockscreenStore {
	private store: Writable<LockscreenState>;
	private autoLockTimer: ReturnType<typeof setInterval> | null = null;
	private initialized = false;

	constructor() {
		this.store = writable<LockscreenState>({
			isLocked: false,
			isEnabled: false,
			autoLockEnabled: false,
			autoLockMinutes: 15,
			lastActivity: Date.now()
		});

		if (browser) {
			this.initializeBrowserFeatures();
		}
	}

	private async initializeBrowserFeatures() {
		if (this.initialized) return;

		this.initialized = true;
		await this.loadSettings();
		this.setupActivityTracking();
		this.startAutoLockTimer();
	}

	private async loadSettings() {
		try {
			const settings = await settingsService.getSettings();

			// Update animation preference
			updateUISettings(settings);

			this.store.update((state) => ({
				...state,
				isEnabled: settings.security.lockscreenEnabled,
				autoLockEnabled: settings.security.autoLockEnabled,
				autoLockMinutes: settings.security.autoLockMinutes,
				// Lock immediately if lockscreen is enabled
				isLocked: settings.security.lockscreenEnabled
			}));
		} catch (error) {
			console.error('Failed to load lockscreen settings:', error);
		}
	}

	private setupActivityTracking() {
		if (!browser) return;

		const events = ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart', 'click'];

		const updateActivity = () => {
			this.store.update((state) => ({
				...state,
				lastActivity: Date.now()
			}));
		};

		events.forEach((event) => {
			window.addEventListener(event, updateActivity, { passive: true });
		});
	}

	private startAutoLockTimer() {
		if (this.autoLockTimer) {
			clearInterval(this.autoLockTimer);
		}

		this.autoLockTimer = setInterval(() => {
			this.store.update((state) => {
				if (state.isEnabled && state.autoLockEnabled && !state.isLocked) {
					const inactiveMinutes = (Date.now() - state.lastActivity) / 1000 / 60;

					if (inactiveMinutes >= state.autoLockMinutes) {
						return { ...state, isLocked: true };
					}
				}
				return state;
			});
		}, 10000);
	}

	subscribe(run: (value: LockscreenState) => void) {
		if (browser && !this.initialized) {
			this.initializeBrowserFeatures();
		}
		return this.store.subscribe(run);
	}

	get state() {
		let currentState: LockscreenState = {
			isLocked: false,
			isEnabled: false,
			autoLockEnabled: false,
			autoLockMinutes: 15,
			lastActivity: Date.now()
		};

		if (this.store) {
			const unsubscribe = this.store.subscribe((state) => (currentState = state));
			unsubscribe();
		}

		return currentState;
	}

	lock() {
		this.store.update((state) => {
			if (state.isEnabled) {
				return { ...state, isLocked: true };
			}
			return state;
		});
	}

	unlock(password: string): boolean {
		const storedPassword = browser ? localStorage.getItem(STORAGE_KEY_PASSWORD) : null;
		const correctPassword = storedPassword || LOCKSCREEN_PASSWORD;

		if (password === correctPassword) {
			this.store.update((state) => ({
				...state,
				isLocked: false,
				lastActivity: Date.now()
			}));
			return true;
		}
		return false;
	}

	updateSettings(settings: {
		lockscreenEnabled: boolean;
		autoLockEnabled: boolean;
		autoLockMinutes: number;
	}) {
		this.store.update((state) => {
			const newState = {
				...state,
				isEnabled: settings.lockscreenEnabled,
				autoLockEnabled: settings.autoLockEnabled,
				autoLockMinutes: settings.autoLockMinutes
			};

			// If lockscreen was just enabled, lock immediately
			if (settings.lockscreenEnabled && !state.isLocked) {
				newState.isLocked = true;
			}

			// If lockscreen was disabled, unlock
			if (!settings.lockscreenEnabled) {
				newState.isLocked = false;
			}

			return newState;
		});

		// Restart auto-lock timer with new settings
		this.startAutoLockTimer();
	}

	setPassword(newPassword: string) {
		if (browser) {
			localStorage.setItem(STORAGE_KEY_PASSWORD, newPassword);
		}
	}

	destroy() {
		if (this.autoLockTimer) {
			clearInterval(this.autoLockTimer);
		}
	}
}

// Export singleton instance - use function to avoid SSR issues
let _lockscreenStore: LockscreenStore | null = null;

export const lockscreenStore = (() => {
	if (!_lockscreenStore) {
		_lockscreenStore = new LockscreenStore();
	}
	return _lockscreenStore;
})();

export function isLocked(): boolean {
	return lockscreenStore.state.isLocked;
}

export function isLockscreenEnabled(): boolean {
	return lockscreenStore.state.isEnabled;
}

export function lockScreen() {
	lockscreenStore.lock();
}

export function unlockScreen(password: string): boolean {
	return lockscreenStore.unlock(password);
}

export function updateLockscreenSettings(settings: {
	lockscreenEnabled: boolean;
	autoLockEnabled: boolean;
	autoLockMinutes: number;
}) {
	lockscreenStore.updateSettings(settings);
}

export function setLockscreenPassword(password: string) {
	lockscreenStore.setPassword(password);
}

// Create a safe wrapper for the lockscreen state that works in SSR
export const lockscreenState = {
	subscribe: (run: (value: LockscreenState) => void) => {
		if (!browser) {
			const defaultState: LockscreenState = {
				isLocked: false,
				isEnabled: false,
				autoLockEnabled: false,
				autoLockMinutes: 15,
				lastActivity: Date.now()
			};
			run(defaultState);
			return () => {}; // Return empty unsubscribe function
		}

		return lockscreenStore.subscribe(run);
	}
};
