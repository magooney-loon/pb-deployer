package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type App struct {
	ID             string    `json:"id" db:"id"`
	Created        time.Time `json:"created" db:"created"`
	Updated        time.Time `json:"updated" db:"updated"`
	Name           string    `json:"name" db:"name"`
	ServerID       string    `json:"server_id" db:"server_id"`
	RemotePath     string    `json:"remote_path" db:"remote_path"`
	ServiceName    string    `json:"service_name" db:"service_name"`
	Domain         string    `json:"domain" db:"domain"` // Production domain (e.g., "myapp.example.com")
	CurrentVersion string    `json:"current_version" db:"current_version"`
	Status         string    `json:"status" db:"status"` // online/offline/unknown via /api/health ping
}

func NewApp() *App {
	return &App{
		Status: "offline",
	}
}

func (a *App) TableName() string {
	return "apps"
}

func (a *App) GetHealthURL() string {
	return "https://" + a.Domain + "/api/health"
}

func (a *App) IsOnline() bool {
	return a.Status == "online"
}

func (a *App) CreateCollection(app core.App) error {
	app.Logger().Info("createAppsCollection: Starting apps collection creation")

	existingCollection, err := app.FindCollectionByNameOrId("apps")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createAppsCollection: Apps collection already exists")
		return nil
	}

	serversCollection, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		app.Logger().Error("createAppsCollection: Servers collection not found", "error", err)
		return err
	}

	collection := core.NewBaseCollection("apps")

	collection.Fields.Add(&core.RelationField{
		Name:          "server_id",
		Required:      true,
		CollectionId:  serversCollection.Id,
		CascadeDelete: true,
	})

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "remote_path",
		Required: true,
		Max:      500,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "service_name",
		Required: true,
		Max:      100,
	})

	collection.Fields.Add(&core.TextField{
		Name: "domain",
		Max:  255,
	})

	collection.Fields.Add(&core.TextField{
		Name: "current_version",
		Max:  100,
	})

	collection.Fields.Add(&core.SelectField{
		Name:   "status",
		Values: []string{"online", "offline", "unknown"},
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnCreate: true,
		OnUpdate: true,
	})

	collection.AddIndex("idx_apps_name", true, "name", "")
	collection.AddIndex("idx_apps_server", false, "server_id", "")
	collection.AddIndex("idx_apps_domain", false, "domain", "")
	collection.AddIndex("idx_apps_status", false, "status", "")

	if err := app.Save(collection); err != nil {
		app.Logger().Error("createAppsCollection: Failed to save apps collection", "error", err)
		return err
	}

	app.Logger().Info("createAppsCollection: Successfully created apps collection")
	return nil
}
