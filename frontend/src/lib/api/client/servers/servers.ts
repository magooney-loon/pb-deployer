import { ServerCrudClient } from './crud.js';

export class ServerClient extends ServerCrudClient {
	// This client now only provides CRUD operations via the inherited ServerCrudClient
	// All custom API operations have been removed
}
