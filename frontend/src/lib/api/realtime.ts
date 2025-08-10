import { BaseClient } from './base.js';
import type { SetupStep } from './types.js';

export class RealtimeClient extends BaseClient {
	// Real-time subscriptions for progress updates
	async subscribeToSetupProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		const subscription = `server_setup_${serverId}`;

		return await this.pb.realtime.subscribe(subscription, (e) => {
			try {
				console.log('Raw setup progress message:', e);

				// The message object itself contains the SetupStep data
				const setupStep = e as SetupStep;

				// Validate that we have the required properties
				if (!setupStep || typeof setupStep.step !== 'string') {
					console.warn('Invalid setup step data:', setupStep);
					return;
				}

				callback(setupStep);
			} catch (error) {
				console.error('Failed to parse setup progress data:', error, 'Raw message:', e);
			}
		});
	}

	async subscribeToSecurityProgress(
		serverId: string,
		callback: (data: SetupStep) => void
	): Promise<() => void> {
		const subscription = `server_security_${serverId}`;

		return await this.pb.realtime.subscribe(subscription, (e) => {
			try {
				console.log('Raw security progress message:', e);

				// The message object itself contains the SetupStep data
				const securityStep = e as SetupStep;

				// Validate that we have the required properties
				if (!securityStep || typeof securityStep.step !== 'string') {
					console.warn('Invalid security step data:', securityStep);
					return;
				}

				callback(securityStep);
			} catch (error) {
				console.error('Failed to parse security progress data:', error, 'Raw message:', e);
			}
		});
	}

	// Unsubscribe from all realtime subscriptions
	async unsubscribeFromAll(): Promise<void> {
		await this.pb.realtime.unsubscribe();
	}
}
