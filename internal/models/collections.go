package models

import (
	"github.com/pocketbase/pocketbase/core"
)

// RegisterCollections creates and registers all the database collections
func RegisterCollections(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		app.Logger().Info("RegisterCollections: Starting collection registration")

		// Create servers collection
		server := NewServer()
		if err := server.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create servers collection", "error", err)
			return err
		}

		// Create apps collection
		appModel := NewApp()
		if err := appModel.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create apps collection", "error", err)
			return err
		}

		// Create versions collection
		version := NewVersion()
		if err := version.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create versions collection", "error", err)
			return err
		}

		// Create deployments collection
		deployment := NewDeployment()
		if err := deployment.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create deployments collection", "error", err)
			return err
		}

		app.Logger().Info("RegisterCollections: All collections registered successfully")
		return e.Next()
	})
}
