package handlers

import (
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/handlers/server"
)

// RegisterHandlers registers all API handlers with the application
func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create API group
		apiGroup := e.Router.Group("/api")

		// Register server handlers
		server.RegisterServerHandlers(app, apiGroup)

		// TODO: Register other handlers here as needed

		return e.Next()
	})
}
