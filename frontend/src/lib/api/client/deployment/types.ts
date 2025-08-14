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
}
