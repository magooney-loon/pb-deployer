import PocketBase from 'pocketbase';

export interface DeployRequest {
	app_id: string;
	version_id: string;
	deployment_id: string;
	superuser_email?: string;
	superuser_pass?: string;
}

export interface DeployResponse {
	success: boolean;
	message: string;
	deployment_id: string;
}

export interface DeployError {
	error: string;
}

export class DeploymentClient {
	private pb: PocketBase;

	constructor(pb: PocketBase) {
		this.pb = pb;
	}

	/**
	 * Deploy an application version to a server
	 */
	async deployApplication(deployRequest: DeployRequest): Promise<DeployResponse> {
		const url = `${this.pb.baseURL}/api/deploy`;

		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				Authorization: this.pb.authStore.token ? `Bearer ${this.pb.authStore.token}` : ''
			},
			body: JSON.stringify(deployRequest)
		});

		const responseText = await response.text();

		if (!response.ok) {
			let errorData;
			try {
				errorData = JSON.parse(responseText);
			} catch {
				throw new Error(`Deployment failed (${response.status})`);
			}
			throw new Error(errorData.error || 'Deployment failed');
		}

		try {
			return JSON.parse(responseText) as DeployResponse;
		} catch {
			throw new Error('Invalid response format');
		}
	}

	/**
	 * Helper method to deploy from a deployment record
	 */
	async deployFromRecord(
		deploymentId: string,
		isInitialDeploy = false,
		superuserEmail?: string,
		superuserPass?: string
	): Promise<DeployResponse> {
		const deployment = await this.pb.collection('deployments').getOne(deploymentId);

		const deployRequest: DeployRequest = {
			app_id: deployment.app,
			version_id: deployment.version,
			deployment_id: deploymentId,
			...(isInitialDeploy &&
				superuserEmail &&
				superuserPass && {
					superuser_email: superuserEmail,
					superuser_pass: superuserPass
				})
		};

		return await this.deployApplication(deployRequest);
	}

	/**
	 * Helper method to retry a failed deployment
	 */
	async retryDeployment(deploymentId: string): Promise<DeployResponse> {
		// Reset deployment status to pending before retrying
		await this.pb.collection('deployments').update(deploymentId, {
			status: 'pending',
			started_at: null,
			completed_at: null,
			logs: 'Retrying deployment...\n'
		});

		return await this.deployFromRecord(deploymentId);
	}
}
