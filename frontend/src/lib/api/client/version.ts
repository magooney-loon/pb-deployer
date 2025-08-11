import { BaseClient } from '../base.js';
import type { Version } from '../types.js';

export class VersionClient extends BaseClient {
	async getAppVersions(id: string) {
		console.log('Getting app versions:', id);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${id}/versions`);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('App versions response:', data);
			return data;
		} catch (error) {
			console.error('Failed to get app versions:', error);
			throw error;
		}
	}

	async createVersion(
		appId: string,
		data: { version_number: string; notes?: string }
	): Promise<Version> {
		console.log('Creating version for app:', appId, data);
		try {
			const response = await fetch(`${this.baseURL}/api/apps/${appId}/versions`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					app_id: appId,
					...data
				})
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const version = await response.json();
			console.log('Version created:', version);
			return version;
		} catch (error) {
			console.error('Failed to create version:', error);
			throw error;
		}
	}

	async uploadVersionFiles(
		versionId: string,
		binaryFile: File,
		publicFiles: File[]
	): Promise<{
		message: string;
		version_id: string;
		binary_file: string;
		binary_size: number;
		public_files_count: number;
		public_total_size: number;
		deployment_file: string;
		deployment_size: number;
		uploaded_at: string;
	}> {
		console.log('Uploading version files:', versionId);
		try {
			const formData = new FormData();
			formData.append('pocketbase_binary', binaryFile);

			// Append all public files
			for (const file of publicFiles) {
				formData.append('pb_public_files', file);
			}

			const response = await fetch(`${this.baseURL}/api/versions/${versionId}/upload`, {
				method: 'POST',
				body: formData
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.error || `HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			console.log('Version files uploaded:', data);
			return data;
		} catch (error) {
			console.error('Failed to upload version files:', error);
			throw error;
		}
	}

	// Convenience method for uploading version with folder structure
	async uploadVersionWithFolder(
		versionId: string,
		binaryFile: File,
		folderFiles: FileList | File[]
	): Promise<{
		message: string;
		version_id: string;
		binary_file: string;
		binary_size: number;
		public_files_count: number;
		public_total_size: number;
		deployment_file: string;
		deployment_size: number;
		uploaded_at: string;
	}> {
		console.log('Uploading version with folder structure:', versionId);

		// Convert FileList to Array if needed
		const publicFiles = Array.isArray(folderFiles) ? folderFiles : Array.from(folderFiles);

		// Validate that we have files
		if (publicFiles.length === 0) {
			throw new Error('No public folder files provided');
		}

		// Use the existing uploadVersionFiles method
		return await this.uploadVersionFiles(versionId, binaryFile, publicFiles);
	}

	// Helper method to validate folder structure for pb_public
	validatePublicFolderStructure(files: File[]): {
		valid: boolean;
		errors: string[];
		warnings: string[];
	} {
		const errors: string[] = [];
		const warnings: string[] = [];

		// Check for common required files
		const hasIndexHtml = files.some(
			(f) => f.webkitRelativePath?.endsWith('index.html') || f.name === 'index.html'
		);
		if (!hasIndexHtml) {
			warnings.push('No index.html found - make sure your app has a main entry point');
		}

		// Check for suspicious files that shouldn't be in public folder
		const suspiciousFiles = files.filter((f) => {
			const name = f.name.toLowerCase();
			return name.includes('.env') || name.includes('config') || name.includes('secret');
		});

		if (suspiciousFiles.length > 0) {
			warnings.push(
				`Found potentially sensitive files: ${suspiciousFiles.map((f) => f.name).join(', ')}`
			);
		}

		// Check total size
		const totalSize = files.reduce((sum, f) => sum + f.size, 0);
		if (totalSize > 50 * 1024 * 1024) {
			// 50MB
			errors.push(
				`Total folder size (${Math.round(totalSize / (1024 * 1024))}MB) exceeds 50MB limit`
			);
		}

		return {
			valid: errors.length === 0,
			errors,
			warnings
		};
	}
}
