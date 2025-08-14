import { AppsCrudClient } from './crud.js';

export class AppsClient extends AppsCrudClient {
	// This client now only provides CRUD operations via the inherited AppsCrudClient
	// All custom API operations have been removed
}
