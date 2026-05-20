import { browser } from '$app/environment';
import { updateAnimationPreference } from '$lib/utils/navigation.js';

const LOCKSCREEN_PASSWORD = '123a';
const STORAGE_KEY_PASSWORD = 'pb-deployer-lockscreen-password';
const STORAGE_KEY_SETTINGS = 'pb-deployer-settings';

export interface LockscreenSettings {
	isLocked: boolean;
	isEnabled: boolean;
	autoLockEnabled: boolean;
	autoLockMinutes: number;
}

export interface UISettings {
	animationsEnabled: boolean;
	mouseEffectsEnabled: boolean;
}

export interface SettingsData {
	security: {
		lockscreenEnabled: boolean;
		autoLockEnabled: boolean;
		autoLockMinutes: number;
	};
	ui: UISettings;
}

function createSettingsStore() {
	let isLocked = $state(false);
	let isEnabled = $state(false);
	let autoLockEnabled = $state(false);
	let autoLockMinutes = $state(15);
	let lastActivity = $state(Date.now());
	let animationsEnabled = $state(true);
	let mouseEffectsEnabled = $state(true);

	let initialized = false;
	let autoLockTimer: ReturnType<typeof setInterval> | null = null;

	function loadFromStorage() {
		if (!browser) return;
		try {
			const stored = localStorage.getItem(STORAGE_KEY_SETTINGS);
			if (stored) {
				const parsed = JSON.parse(stored);
				isEnabled = parsed.security?.lockscreenEnabled ?? false;
				autoLockEnabled = parsed.security?.autoLockEnabled ?? false;
				autoLockMinutes = parsed.security?.autoLockMinutes ?? 15;
				animationsEnabled = parsed.ui?.animationsEnabled ?? true;
				mouseEffectsEnabled = parsed.ui?.mouseEffectsEnabled ?? true;
			}
		} catch {
			// ignore parse errors
		}
		if (isEnabled) isLocked = true;
		updateAnimationPreference(animationsEnabled);
	}

	function saveToStorage() {
		if (!browser) return;
		try {
			localStorage.setItem(
				STORAGE_KEY_SETTINGS,
				JSON.stringify({
					security: { lockscreenEnabled: isEnabled, autoLockEnabled, autoLockMinutes },
					ui: { animationsEnabled, mouseEffectsEnabled }
				})
			);
		} catch {
			// ignore write errors
		}
	}

	function setupActivityTracking() {
		if (!browser) return;
		const events = ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart', 'click'];
		const handler = () => {
			lastActivity = Date.now();
		};
		events.forEach((e) => window.addEventListener(e, handler, { passive: true }));
	}

	function startAutoLockTimer() {
		if (autoLockTimer) clearInterval(autoLockTimer);
		autoLockTimer = setInterval(() => {
			if (isEnabled && autoLockEnabled && !isLocked) {
				const inactiveMinutes = (Date.now() - lastActivity) / 1000 / 60;
				if (inactiveMinutes >= autoLockMinutes) isLocked = true;
			}
		}, 10000);
	}

	function initialize() {
		if (initialized || !browser) return;
		initialized = true;
		loadFromStorage();
		setupActivityTracking();
		startAutoLockTimer();
	}

	function lock() {
		if (isEnabled) isLocked = true;
	}

	function unlock(password: string): boolean {
		if (!browser) return false;
		const storedPassword = localStorage.getItem(STORAGE_KEY_PASSWORD);
		const correct = storedPassword || LOCKSCREEN_PASSWORD;
		if (password === correct) {
			isLocked = false;
			lastActivity = Date.now();
			return true;
		}
		return false;
	}

	function updateLockscreen(s: {
		lockscreenEnabled: boolean;
		autoLockEnabled: boolean;
		autoLockMinutes: number;
	}) {
		isEnabled = s.lockscreenEnabled;
		autoLockEnabled = s.autoLockEnabled;
		autoLockMinutes = s.autoLockMinutes;
		if (s.lockscreenEnabled && !isLocked) isLocked = true;
		if (!s.lockscreenEnabled) isLocked = false;
		saveToStorage();
		startAutoLockTimer();
	}

	function updateUI(prefs: { animationsEnabled?: boolean; mouseEffectsEnabled?: boolean }) {
		if (prefs.animationsEnabled !== undefined) {
			animationsEnabled = prefs.animationsEnabled;
			updateAnimationPreference(animationsEnabled);
		}
		if (prefs.mouseEffectsEnabled !== undefined) mouseEffectsEnabled = prefs.mouseEffectsEnabled;
		saveToStorage();
	}

	function setPassword(password: string) {
		if (browser) localStorage.setItem(STORAGE_KEY_PASSWORD, password);
	}

	return {
		get lockscreen() {
			return { isLocked, isEnabled, autoLockEnabled, autoLockMinutes };
		},
		get ui() {
			return { animationsEnabled, mouseEffectsEnabled };
		},
		initialize,
		lock,
		unlock,
		updateLockscreen,
		updateUI,
		setPassword
	};
}

export const settings = createSettingsStore();
