package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// App represents a PocketBase application deployed on a server
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

// NewApp creates a new App instance with default values
func NewApp() *App {
	return &App{
		Status: "offline",
	}
}

// TableName returns the collection name for the App model
func (a *App) TableName() string {
	return "apps"
}

// GetHealthURL returns the health check URL for this app
func (a *App) GetHealthURL() string {
	return "https://" + a.Domain + "/api/health"
}

// IsOnline checks if the app status is online
func (a *App) IsOnline() bool {
	return a.Status == "online"
}

// CreateCollection creates the apps collection in the database
func (a *App) CreateCollection(app core.App) error {
	app.Logger().Info("createAppsCollection: Starting apps collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("apps")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createAppsCollection: Apps collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("apps")

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	// Add fields with minimal validation
	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "server_id",
		Required: true,
		Max:      15,
	})

	collection.Fields.Add(&core.TextField{
		Name: "remote_path",
		Max:  500,
	})

	collection.Fields.Add(&core.TextField{
		Name: "service_name",
		Max:  100,
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

	// Add auto-date fields
	collection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnCreate: true,
		OnUpdate: true,
	})

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createAppsCollection: Failed to save apps collection", "error", err)
		return err
	}

	app.Logger().Info("createAppsCollection: Successfully created apps collection")
	return nil
}
