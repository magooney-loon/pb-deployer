import { writable } from 'svelte/store';

interface SplashScreenState {
	isLoading: boolean;
	progress: number;
}

const initialState: SplashScreenState = {
	isLoading: true,
	progress: 0
};

export const splashScreenState = writable<SplashScreenState>(initialState);

export class SplashScreenManager {
	private static instance: SplashScreenManager;
	private progressTimer: number | null = null;
	private readonly loadingDuration = 750; // 0.75 seconds
	private readonly updateInterval = 16; // ~60fps

	static getInstance(): SplashScreenManager {
		if (!SplashScreenManager.instance) {
			SplashScreenManager.instance = new SplashScreenManager();
		}
		return SplashScreenManager.instance;
	}

	startLoading(): void {
		this.stopLoading(); // Clear any existing timer

		splashScreenState.set({
			isLoading: true,
			progress: 0
		});

		const totalSteps = this.loadingDuration / this.updateInterval;
		let currentStep = 0;

		this.progressTimer = window.setInterval(() => {
			currentStep++;
			const progress = (currentStep / totalSteps) * 100;

			splashScreenState.update((state) => ({
				...state,
				progress
			}));

			if (currentStep >= totalSteps) {
				this.completeLoading();
			}
		}, this.updateInterval);
	}

	completeLoading(): void {
		this.stopLoading();
		splashScreenState.update((state) => ({
			...state,
			isLoading: false,
			progress: 100
		}));
	}

	stopLoading(): void {
		if (this.progressTimer !== null) {
			clearInterval(this.progressTimer);
			this.progressTimer = null;
		}
	}

	reset(): void {
		this.stopLoading();
		splashScreenState.set(initialState);
	}
}

export const splashScreen = SplashScreenManager.getInstance();
