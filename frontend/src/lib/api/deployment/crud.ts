import { BaseClient } from '../base.js';
import type { Deployment } from './types.js';

export class DeploymentCrudClient extends BaseClient {
	// Basic PocketBase CRUD operations for deployments
	async getDeployments() {
		console.log('Getting deployments via PocketBase...');
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				sort: '-created'
			});
			console.log('PocketBase deployments response:', records);

			// Transform to match expected format
			const result = { deployments: records || [] };
			console.log('getDeployments result:', result);
			return result;
		} catch (error) {
			console.error('Failed to get deployments:', error);
			throw error;
		}
	}

	async getDeployment(id: string): Promise<Deployment> {
		console.log('Getting deployment:', id);
		try {
			const deployment = await this.pb.collection('deployments').getOne<Deployment>(id);
			console.log('PocketBase deployment response:', deployment);
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
		console.log('Creating deployment:', data);
		try {
			const deployment = await this.pb.collection('deployments').create<Deployment>({
				...data,
				status: data.status || 'pending'
			});
			console.log('Deployment created:', deployment);
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
		console.log('Updating deployment:', id, data);
		try {
			const deployment = await this.pb.collection('deployments').update<Deployment>(id, data);
			console.log('Deployment updated:', deployment);
			return deployment;
		} catch (error) {
			console.error('Failed to update deployment:', error);
			throw error;
		}
	}

	async deleteDeployment(id: string) {
		console.log('Deleting deployment:', id);
		try {
			await this.pb.collection('deployments').delete(id);
			console.log('Deployment deleted:', id);
			return { message: 'Deployment deleted successfully' };
		} catch (error) {
			console.error('Failed to delete deployment:', error);
			throw error;
		}
	}

	async getAppDeployments(appId: string) {
		console.log('Getting deployments by app:', appId);
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				filter: `app_id = "${appId}"`,
				sort: '-created'
			});
			console.log('PocketBase deployments by app response:', records);

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
		console.log('Getting deployments by version:', versionId);
		try {
			const records = await this.pb.collection('deployments').getFullList<Deployment>({
				filter: `version_id = "${versionId}"`,
				sort: '-created'
			});
			console.log('PocketBase deployments by version response:', records);

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
