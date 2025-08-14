import { BaseClient } from '../../base.js';
import type { Version } from './types.js';

export class VersionCrudClient extends BaseClient {
	// Basic PocketBase CRUD operations for versions
	async getVersions() {
		console.log('Getting versions via PocketBase...');
		try {
			const records = await this.pb.collection('versions').getFullList<Version>({
				sort: '-created'
			});
			console.log('PocketBase versions response:', records);

			// Transform to match expected format
			const result = { versions: records || [] };
			console.log('getVersions result:', result);
			return result;
		} catch (error) {
			console.error('Failed to get versions:', error);
			throw error;
		}
	}

	async getVersion(id: string): Promise<Version> {
		console.log('Getting version:', id);
		try {
			const version = await this.pb.collection('versions').getOne<Version>(id);
			console.log('PocketBase version response:', version);
			return version;
		} catch (error) {
			console.error('Failed to get version:', error);
			throw error;
		}
	}

	async createVersion(data: { app_id: string; version_number: string; notes?: string }): Promise<Version> {
		console.log('Creating version:', data);
		try {
			const version = await this.pb.collection('versions').create<Version>(data);
			console.log('Version created:', version);
			return version;
		} catch (error) {
			console.error('Failed to create version:', error);
			throw error;
		}
	}

	async updateVersion(id: string, data: Partial<{ version_number: string; notes: string; deployment_zip: string }>): Promise<Version> {
		console.log('Updating version:', id, data);
		try {
			const version = await this.pb.collection('versions').update<Version>(id, data);
			console.log('Version updated:', version);
			return version;
		} catch (error) {
			console.error('Failed to update version:', error);
			throw error;
		}
	}

	async deleteVersion(id: string) {
		console.log('Deleting version:', id);
		try {
			await this.pb.collection('versions').delete(id);
			console.log('Version deleted:', id);
			return { message: 'Version deleted successfully' };
		} catch (error) {
			console.error('Failed to delete version:', error);
			throw error;
		}
	}

	async getAppVersions(appId: string) {
		console.log('Getting versions by app:', appId);
		try {
			const records = await this.pb.collection('versions').getFullList<Version>({
				filter: `app_id = "${appId}"`,
				sort: '-created'
			});
			console.log('PocketBase versions by app response:', records);

			return {
				app_id: appId,
				versions: records || []
			};
		} catch (error) {
			console.error('Failed to get versions by app:', error);
			throw error;
		}
	}
}
