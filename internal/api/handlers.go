package api

// API_SOURCE

import (
	"github.com/magooney-loon/pb-ext/core/server"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		router := server.EnableAutoDocumentation(e)

		router.POST("/api/setup/server", func(c *core.RequestEvent) error {
			return handleServerSetup(c, app)
		})

		router.POST("/api/setup/security", func(c *core.RequestEvent) error {
			return handleServerSecurity(c, app)
		})

		router.POST("/api/setup/validate", func(c *core.RequestEvent) error {
			return handleServerValidation(c)
		})

		router.POST("/api/deploy", func(c *core.RequestEvent) error {
			return handleDeploy(c, app)
		})

		return e.Next()
	})

}
