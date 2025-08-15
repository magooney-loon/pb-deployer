package models

import (
	"github.com/pocketbase/pocketbase/core"
)

func RegisterCollections(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		app.Logger().Info("RegisterCollections: Starting collection registration")

		server := NewServer()
		if err := server.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create servers collection", "error", err)
			return err
		}

		appModel := NewApp()
		if err := appModel.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create apps collection", "error", err)
			return err
		}

		version := NewVersion()
		if err := version.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create versions collection", "error", err)
			return err
		}

		deployment := NewDeployment()
		if err := deployment.CreateCollection(app); err != nil {
			app.Logger().Error("RegisterCollections: Failed to create deployments collection", "error", err)
			return err
		}

		app.Logger().Info("RegisterCollections: All collections registered successfully")
		return e.Next()
	})
}
