package models

import (
	"github.com/pocketbase/pocketbase/core"
)

// All models are now defined in their respective files:
// - Server (server.go)
// - App (app.go)
// - Version (version.go)
// - Deployment (deployment.go)

// InitializeDatabase sets up all collections and schema
func InitializeDatabase(app core.App) error {
	return RegisterCollections(app)
}
