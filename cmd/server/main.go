package main

import (
	"flag"
	"log"

	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/api"
	_ "pb-deployer/migrations"
)

func main() {
	devMode := flag.Bool("dev", false, "Run in developer mode")
	generateSpecsDir := flag.String("generate-specs-dir", "", "Generate OpenAPI specs into the provided directory and exit")
	generateSpecVersion := flag.String("generate-spec-version", "", "Optional API version to generate (requires --generate-specs-dir)")
	validateSpecsDir := flag.String("validate-specs-dir", "", "Validate OpenAPI specs from the provided directory and exit")
	flag.Parse()

	if *generateSpecsDir != "" {
		gen := app.NewSpecGeneratorWithInitializer(func() (*app.APIVersionManager, error) {
			return api.InitVersionedSystem(), nil
		})
		if err := gen.Generate(*generateSpecsDir, *generateSpecVersion); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *validateSpecsDir != "" {
		gen := app.NewSpecGeneratorWithInitializer(func() (*app.APIVersionManager, error) {
			return api.InitVersionedSystem(), nil
		})
		if err := gen.Validate(*validateSpecsDir); err != nil {
			log.Fatal(err)
		}
		return
	}

	initApp(*devMode)
}

func initApp(devMode bool) {
	var opts []app.Option

	if devMode {
		opts = append(opts, app.InDeveloperMode())
	} else {
		opts = append(opts, app.InNormalMode())
	}

	srv := app.New(opts...)

	app.SetupLogging(srv)

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

func registerHandlers(app core.App) {
	api.RegisterHandlers(app)
}
