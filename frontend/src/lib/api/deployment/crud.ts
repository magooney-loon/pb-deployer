import PocketBase from 'pocketbase';
import type { Deployment } from './types.js';

export class DeploymentCrudClient {
	private pb: PocketBase;

	constructor(pb: PocketBase) {
		this.pb = pb;
	}

	async getDeployments() {
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				sort: '-created'
			});

			const result = { deployments: records || [] };
			return result;
		} catch (error) {
			console.error('Failed to get deployments:', error);
			throw error;
		}
	}

	async getDeployment(id: string): Promise<Deployment> {
		try {
			const deployment = await this.pb.collection('deployments').getOne<Deployment>(id);
			return deployment;
		} catch (error) {
			console.error('Failed to get deployment:', error);
			throw error;
		}
	}

	async createDeployment(data: {
		app_id: string;
		version_id: string;
		status?: string;
	}): Promise<Deployment> {
		try {
			const deployment = await this.pb.collection('deployments').create<Deployment>({
				...data,
				status: data.status || 'pending'
			});
			return deployment;
		} catch (error) {
			console.error('Failed to create deployment:', error);
			throw error;
		}
	}

	async updateDeployment(
		id: string,
		data: Partial<{ status: string; logs: string; started_at: string; completed_at: string }>
	): Promise<Deployment> {
		try {
			const deployment = await this.pb.collection('deployments').update<Deployment>(id, data);
			return deployment;
		} catch (error) {
			console.error('Failed to update deployment:', error);
			throw error;
		}
	}

	async deleteDeployment(id: string) {
		try {
			await this.pb.collection('deployments').delete(id);
			return { message: 'Deployment deleted successfully' };
		} catch (error) {
			console.error('Failed to delete deployment:', error);
			throw error;
		}
	}

	async getAppDeployments(appId: string) {
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				filter: `app_id = "${appId}"`,
				sort: '-created'
			});

			return {
				app_id: appId,
				deployments: records || []
			};
		} catch (error) {
			console.error('Failed to get deployments by app:', error);
			throw error;
		}
	}

	async getVersionDeployments(versionId: string) {
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				filter: `version_id = "${versionId}"`,
				sort: '-created'
			});

			return {
				version_id: versionId,
				deployments: records || []
			};
		} catch (error) {
			console.error('Failed to get deployments by version:', error);
			throw error;
		}
	}
}
