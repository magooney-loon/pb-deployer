package api

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create API group
		// apiGroup := e.Router.Group("/api")

		// server.RegisterServerHandlers(app, apiGroup)

		// apps.RegisterAppsHandlers(app, apiGroup)

		// version.RegisterVersionHandlers(app, apiGroup)

		// deployment.RegisterDeploymentHandlers(app, apiGroup)

		return e.Next()
	})
}
