import PocketBase from 'pocketbase';
import type { Version } from './types.js';

export class VersionCrudClient {
	private pb: PocketBase;

	constructor(pb: PocketBase) {
		this.pb = pb;
	}

	async getVersions() {
		try {
			const records = await this.pb.collection('versions').getFullList<Version>({
				sort: '-created'
			});

			const result = { versions: records || [] };
			return result;
		} catch (error) {
			console.error('Failed to get versions:', error);
			throw error;
		}
	}

	async getVersion(id: string): Promise<Version> {
		try {
			const version = await this.pb.collection('versions').getOne<Version>(id);
			return version;
		} catch (error) {
			console.error('Failed to get version:', error);
			throw error;
		}
	}

	async createVersion(data: {
		app_id: string;
		version_number: string;
		notes?: string;
	}): Promise<Version> {
		try {
			const version = await this.pb.collection('versions').create<Version>(data);
			return version;
		} catch (error) {
			console.error('Failed to create version:', error);
			throw error;
		}
	}

	async updateVersion(
		id: string,
		data: Partial<{ version_number: string; notes: string; deployment_zip: string }>
	): Promise<Version> {
		try {
			const version = await this.pb.collection('versions').update<Version>(id, data);
			return version;
		} catch (error) {
			console.error('Failed to update version:', error);
			throw error;
		}
	}

	async deleteVersion(id: string) {
		try {
			await this.pb.collection('versions').delete(id);
			return { message: 'Version deleted successfully' };
		} catch (error) {
			console.error('Failed to delete version:', error);
			throw error;
		}
	}

	async getAppVersions(appId: string) {
		try {
			const records = await this.pb.collection('versions').getFullList<Version>({
				filter: `app_id = "${appId}"`,
				sort: '-created'
			});

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
