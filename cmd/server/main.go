package main

import (
	"log"
	"os/exec"
	"runtime"
	"time"

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

		go func() {
			time.Sleep(690 * time.Millisecond)
			openBrowser("http://localhost:8090")
		}()

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
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := models.RegisterCollections(e.App); err != nil {
			app.Logger().Error("Failed to initialize database collections", "error", err)
		}
		return e.Next()
	})
}

func registerHandlers(app core.App) {
	api.RegisterHandlers(app)
}

func openBrowser(url string) {
	if runtime.GOOS != "linux" {
		return
	}

	if err := exec.Command("xdg-open", url).Start(); err == nil {
		log.Printf("Browser launched for URL: %s", url)
	} else {
		log.Printf("Failed to open browser for URL: %s", url)
	}
}
