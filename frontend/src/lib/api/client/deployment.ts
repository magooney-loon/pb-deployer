import { BaseClient } from '../base.js';
import type { Deployment } from '../types.js';

export class DeploymentClient extends BaseClient {
	async getAppDeployments(id: string) {
		console.log('Getting app deployments:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/deployments?app_id=${id}`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App deployments response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get app deployments:', error);
			throw error;
		}
	}

	async getDeployment(id: string): Promise<Deployment> {
		console.log('Getting deployment:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/deployments/${id}`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Deployment response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get deployment:', error);
			throw error;
		}
	}

	async createDeployment(appId: string, versionId: string): Promise<Deployment> {
		console.log('Creating deployment:', { appId, versionId });
		try {
			const response = await fetch(`${this.baseURL}/api/deployments`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					app_id: appId,
					version_id: versionId
				})
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const deployment = await response.json();
			console.log('Deployment created:', deployment);
			return deployment;
		} catch (error) {
			console.error('Failed to create deployment:', error);
			throw error;
		}
	}

	async getDeploymentLogs(id: string) {
		console.log('Getting deployment logs:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/deployments/${id}/logs`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Deployment logs response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get deployment logs:', error);
			throw error;
		}
	}

	async getDeploymentStatus(id: string) {
		console.log('Getting deployment status:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/deployments/${id}/status`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Deployment status response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get deployment status:', error);
			throw error;
		}
	}
}
