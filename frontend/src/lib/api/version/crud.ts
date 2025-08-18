import PocketBase from 'pocketbase';
import type { Version } from './types.js';

interface PocketBaseError {
	status: number;
	message?: string;
	data?: {
		data?: Record<string, unknown>;
	};
}

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
		deployment_zip?: File;
	}): Promise<Version> {
		try {
			console.log('Creating version with data:', {
				app_id: data.app_id,
				version_number: data.version_number,
				notes: data.notes,
				has_file: !!data.deployment_zip,
				file_size: data.deployment_zip?.size,
				file_type: data.deployment_zip?.type
			});

			// Check for duplicate version first
			const existingVersions = await this.checkVersionExists(data.app_id, data.version_number);
			if (existingVersions) {
				throw new Error(
					`Version ${data.version_number} already exists for this application. Please use a different version number.`
				);
			}

			// Validate file if provided
			if (data.deployment_zip) {
				const maxSize = 150 * 1024 * 1024; // 150MB
				if (data.deployment_zip.size > maxSize) {
					throw new Error(
						`File size (${Math.round(data.deployment_zip.size / 1024 / 1024)}MB) exceeds maximum allowed size (150MB)`
					);
				}

				if (!data.deployment_zip.type.includes('zip')) {
					throw new Error('File must be a ZIP archive');
				}
			}

			const formData = new FormData();
			formData.append('app_id', data.app_id);
			formData.append('version_number', data.version_number);

			if (data.notes) {
				formData.append('notes', data.notes);
			}

			if (data.deployment_zip) {
				formData.append('deployment_zip', data.deployment_zip);
			}

			console.log('Sending FormData to PocketBase...');
			const version = await this.pb.collection('versions').create<Version>(formData);
			console.log('Version created successfully:', version.id);

			// Update app's current_version field
			try {
				await this.pb.collection('apps').update(data.app_id, {
					current_version: data.version_number
				});
				console.log('Updated app current_version to:', data.version_number);
			} catch (updateError) {
				console.warn('Failed to update app current_version:', updateError);
				// Don't throw here as the version was created successfully
			}

			return version;
		} catch (error) {
			console.error('Failed to create version:', error);

			// Handle our own validation errors first
			if (error instanceof Error && error.message.includes('already exists')) {
				throw error;
			}

			// Handle specific PocketBase errors
			if (error && typeof error === 'object' && 'data' in error) {
				const pbError = error as PocketBaseError;

				// Check for file size or validation errors
				if (pbError.status === 400) {
					if (pbError.message?.includes('deployment_zip')) {
						throw new Error(
							'File upload failed. Please ensure the file is a valid ZIP archive under 150MB.'
						);
					}
					throw new Error(pbError.message || 'Validation failed. Please check your input data.');
				}

				// Server errors - provide more specific guidance
				if (pbError.status === 500) {
					throw new Error(
						'Server error during version creation. This might be due to a database issue or file processing error. Please try again with a different version number or file.'
					);
				}

				throw new Error(pbError.message || 'Failed to create version');
			}

			if (error instanceof Error) {
				throw error;
			}
			throw new Error('An unexpected error occurred during version creation');
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

	async checkVersionExists(appId: string, versionNumber: string): Promise<boolean> {
		try {
			const records = await this.pb.collection('versions').getFullList<Version>({
				filter: `app_id = "${appId}" && version_number = "${versionNumber}"`,
				requestKey: null // Disable caching for this check
			});

			console.log(`Version check for ${versionNumber} in app ${appId}:`, records.length > 0);
			return records.length > 0;
		} catch (error) {
			console.error('Failed to check version existence:', error);
			// If we can't check, assume it doesn't exist and let the server handle conflicts
			return false;
		}
	}
}
