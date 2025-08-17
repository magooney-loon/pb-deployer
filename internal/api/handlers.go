package api

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {

		e.Router.POST("/api/setup/server", func(c *core.RequestEvent) error {
			return handleServerSetup(c, app)
		})

		e.Router.POST("/api/setup/security", func(c *core.RequestEvent) error {
			return handleServerSecurity(c, app)
		})

		e.Router.POST("/api/setup/validate", func(c *core.RequestEvent) error {
			return handleServerValidation(c)
		})

		return e.Next()
	})
}
