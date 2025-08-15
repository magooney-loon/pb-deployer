package main

import (
	"log"

	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/api"
	"pb-deployer/internal/models"
)

func main() {
	initApp()
}

func initApp() {
	srv := app.New()

	app.SetupLogging(srv)

	registerCollections(srv.App())
	registerHandlers(srv.App())

	srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.Bind(apis.BodyLimit(209715200))

		app.SetupRecovery(srv.App(), e)

		return e.Next()
	})

	if err := srv.Start(); err != nil {
		srv.App().Logger().Error("Fatal application error",
			"error", err,
			"uptime", srv.Stats().StartTime,
			"total_requests", srv.Stats().TotalRequests.Load(),
			"active_connections", srv.Stats().ActiveConnections.Load(),
			"last_request_time", srv.Stats().LastRequestTime.Load(),
		)
		log.Fatal(err)
	}
}

func registerCollections(app core.App) {
	models.RegisterCollections(app)
}

func registerHandlers(app core.App) {
	api.RegisterHandlers(app)
}
