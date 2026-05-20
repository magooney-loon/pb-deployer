function createSplashStore() {
	let isLoading = $state(true);
	let progress = $state(0);
	let timer: ReturnType<typeof setInterval> | null = null;

	function start() {
		stop();
		isLoading = true;
		progress = 0;
		const duration = 900;
		const interval = 16;
		const totalSteps = duration / interval;
		let step = 0;

		timer = setInterval(() => {
			step++;
			progress = (step / totalSteps) * 100;
			if (step >= totalSteps) complete();
		}, interval);
	}

	function complete() {
		stop();
		isLoading = false;
		progress = 100;
	}

	function stop() {
		if (timer !== null) {
			clearInterval(timer);
			timer = null;
		}
	}

	return {
		get isLoading() {
			return isLoading;
		},
		get progress() {
			return progress;
		},
		start,
		stop,
		complete
	};
}

export const splash = createSplashStore();
