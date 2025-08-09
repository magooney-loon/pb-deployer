package handlers

import (
	"github.com/pocketbase/pocketbase/core"
)

// RegisterHandlers registers all API handlers with the application
func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create API group
		apiGroup := e.Router.Group("/api")

		// Register server handlers
		RegisterServerHandlers(app, apiGroup)

		// TODO: Register other handlers (apps, deployments, etc.) here as needed

		return e.Next()
	})
}
