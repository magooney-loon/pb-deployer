package main

import (
	"flag"
	"log"

	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/api"
	"pb-deployer/internal/models"
)

func main() {
	devMode := flag.Bool("dev", false, "Run in developer mode")
	flag.Parse()

	initApp(*devMode)
}

func initApp(devMode bool) {
	var srv *app.Server
	if devMode {
		srv = app.New(app.InDeveloperMode())
		log.Println("🔧 Developer mode enabled")
	} else {
		srv = app.New(app.InNormalMode())
		log.Println("🚀 Production mode")
	}

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
