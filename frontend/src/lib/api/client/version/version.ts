import { VersionCrudClient } from './crud.js';

export class VersionClient extends VersionCrudClient {
	// This client now only provides CRUD operations via the inherited VersionCrudClient
	// All custom API operations have been removed
}
