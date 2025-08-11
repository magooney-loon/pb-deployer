package main

import (
	"log"

	app "github.com/magooney-loon/pb-ext/core"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pb-deployer/internal/handlers"
	"pb-deployer/internal/models"
	"pb-deployer/internal/ssh"
)

func main() {
	initApp()
}

func initApp() {
	srv := app.New()

	app.SetupLogging(srv)

	// Initialize SSH connection management
	srv.App().Logger().Info("Initializing SSH connection management system...")
	connectionManager := ssh.GetConnectionManager()

	// Get initial connection status
	initialStatus := connectionManager.GetConnectionStatus()
	srv.App().Logger().Info("Initial SSH connection pool status",
		"active_connections", len(initialStatus))

	// Clean up any existing stale SSH connections on startup
	srv.App().Logger().Info("Performing startup cleanup of stale SSH connections...")
	cleanedCount := connectionManager.CleanupConnections()
	if cleanedCount > 0 {
		srv.App().Logger().Info("Cleaned up stale SSH connections on startup", "count", cleanedCount)
	} else {
		srv.App().Logger().Info("No stale SSH connections found during startup cleanup")
	}

	// Setup graceful shutdown for SSH connections
	srv.App().OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		srv.App().Logger().Info("Graceful shutdown initiated - cleaning up SSH connections...")

		// Get final connection status before shutdown
		finalStatus := connectionManager.GetConnectionStatus()
		srv.App().Logger().Info("Final SSH connection pool status before shutdown",
			"active_connections", len(finalStatus))

		// Shutdown connection manager
		connectionManager.Shutdown()
		srv.App().Logger().Info("SSH connection manager shutdown complete")
		return e.Next()
	})

	srv.App().Logger().Info("SSH connection management system initialized successfully")

	registerCollections(srv.App())
	registerHandlers(srv.App())

	srv.App().OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Add global body size limit middleware (200MB)
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
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := models.InitializeDatabase(e.App); err != nil {
			app.Logger().Error("Failed to initialize database collections", "error", err)
		}
		return e.Next()
	})
}

func registerHandlers(app core.App) {
	handlers.RegisterHandlers(app)
}
