export interface Deployment {
	id: string;
	created: string;
	updated: string;
	app_id: string;
	version_id: string;
	status: string;
	logs: string;
	started_at?: string;
	completed_at?: string;
	// Expanded relations
	expand?: {
		app_id?: {
			id: string;
			name: string;
			domain: string;
			service_name: string;
			server_id: string;
			remote_path: string;
			current_version?: string;
			status: string;
		};
		version_id?: {
			id: string;
			app_id: string;
			version_number: string;
			notes: string;
			deployment_zip?: string;
		};
	};
}
