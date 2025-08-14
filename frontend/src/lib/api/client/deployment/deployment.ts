import { DeploymentCrudClient } from './crud.js';

export class DeploymentClient extends DeploymentCrudClient {
	// This client now only provides CRUD operations via the inherited DeploymentCrudClient
	// All custom API operations have been removed
}
