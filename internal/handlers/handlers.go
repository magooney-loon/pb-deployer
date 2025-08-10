package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/handlers/apps"
	"pb-deployer/internal/handlers/deployment"
	"pb-deployer/internal/handlers/server"
	"pb-deployer/internal/handlers/version"
)

// RegisterHandlers registers all API handlers with the application
func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create API group
		apiGroup := e.Router.Group("/api")

		// Register server handlers
		server.RegisterServerHandlers(app, apiGroup)

		// Register app handlers
		apps.RegisterAppsHandlers(app, apiGroup)

		// Register version handlers
		version.RegisterVersionHandlers(app, apiGroup)

		// Register deployment handlers
		deployment.RegisterDeploymentHandlers(app, apiGroup)

		return e.Next()
	})
}
