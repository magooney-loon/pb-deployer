package models

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// RegisterCollections creates and registers all the database collections
func RegisterCollections(app core.App) error {
	// Create servers collection
	if err := createServersCollection(app); err != nil {
		return err
	}

	// Create apps collection
	if err := createAppsCollection(app); err != nil {
		return err
	}

	// Create versions collection
	if err := createVersionsCollection(app); err != nil {
		return err
	}

	// Create deployments collection
	if err := createDeploymentsCollection(app); err != nil {
		return err
	}

	return nil
}

// createServersCollection creates the servers collection
func createServersCollection(app core.App) error {
	// Check if collection already exists
	existingCollection, _ := app.FindCollectionByNameOrId("servers")
	if existingCollection != nil {
		app.Logger().Info("Servers collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("servers")

	// Add fields
	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Min:      1,
		Max:      255,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "host",
		Required: true,
		Min:      1,
		Max:      255,
	})

	collection.Fields.Add(&core.NumberField{
		Name:     "port",
		Required: true,
		Min:      types.Pointer(1.0),
		Max:      types.Pointer(65535.0),
	})

	collection.Fields.Add(&core.TextField{
		Name:     "root_username",
		Required: true,
		Min:      1,
		Max:      50,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "app_username",
		Required: true,
		Min:      1,
		Max:      50,
	})

	collection.Fields.Add(&core.BoolField{
		Name: "use_ssh_agent",
	})

	collection.Fields.Add(&core.TextField{
		Name: "manual_key_path",
		Max:  500,
	})

	collection.Fields.Add(&core.BoolField{
		Name: "setup_complete",
	})

	collection.Fields.Add(&core.BoolField{
		Name: "security_locked",
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

	// Add index for name (unique)
	collection.AddIndex("idx_servers_name", true, "name", "")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("Failed to create servers collection", "error", err)
		return err
	}

	app.Logger().Info("Created servers collection")
	return nil
}

// createAppsCollection creates the apps collection
func createAppsCollection(app core.App) error {
	// Check if collection already exists
	existingCollection, _ := app.FindCollectionByNameOrId("apps")
	if existingCollection != nil {
		app.Logger().Info("Apps collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("apps")

	// Find servers collection for relation
	serversCollection, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		return err
	}

	// Add fields
	collection.Fields.Add(&core.TextField{
		Name:     "name",
		Required: true,
		Min:      1,
		Max:      255,
	})

	collection.Fields.Add(&core.RelationField{
		Name:          "server_id",
		Required:      true,
		CollectionId:  serversCollection.Id,
		CascadeDelete: true,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "remote_path",
		Required: true,
		Min:      1,
		Max:      500,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "service_name",
		Required: true,
		Min:      1,
		Max:      100,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "domain",
		Required: true,
		Min:      1,
		Max:      255,
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

	// Add indexes
	collection.AddIndex("idx_apps_server", false, "server_id", "")
	collection.AddIndex("idx_apps_domain", true, "domain", "")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("Failed to create apps collection", "error", err)
		return err
	}

	app.Logger().Info("Created apps collection")
	return nil
}

// createVersionsCollection creates the versions collection
func createVersionsCollection(app core.App) error {
	// Check if collection already exists
	existingCollection, _ := app.FindCollectionByNameOrId("versions")
	if existingCollection != nil {
		app.Logger().Info("Versions collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("versions")

	// Find apps collection for relation
	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		return err
	}

	// Add fields
	collection.Fields.Add(&core.RelationField{
		Name:          "app_id",
		Required:      true,
		CollectionId:  appsCollection.Id,
		CascadeDelete: true,
	})

	collection.Fields.Add(&core.TextField{
		Name:     "version_number",
		Required: true,
		Min:      1,
		Max:      50,
	})

	collection.Fields.Add(&core.FileField{
		Name:      "deployment_zip",
		Required:  true,
		MaxSelect: 1,
		MaxSize:   157286400, // 150MB (binary + static files)
		MimeTypes: []string{"application/zip"},
	})

	collection.Fields.Add(&core.TextField{
		Name: "notes",
		Max:  1000,
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

	// Add indexes
	collection.AddIndex("idx_versions_app", false, "app_id", "")
	collection.AddIndex("idx_versions_app_version", true, "app_id", "version_number")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("Failed to create versions collection", "error", err)
		return err
	}

	app.Logger().Info("Created versions collection")
	return nil
}

// createDeploymentsCollection creates the deployments collection
func createDeploymentsCollection(app core.App) error {
	// Check if collection already exists
	existingCollection, _ := app.FindCollectionByNameOrId("deployments")
	if existingCollection != nil {
		app.Logger().Info("Deployments collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("deployments")

	// Find apps and versions collections for relations
	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		return err
	}

	versionsCollection, err := app.FindCollectionByNameOrId("versions")
	if err != nil {
		return err
	}

	// Add fields
	collection.Fields.Add(&core.RelationField{
		Name:          "app_id",
		Required:      true,
		CollectionId:  appsCollection.Id,
		CascadeDelete: true,
	})

	collection.Fields.Add(&core.RelationField{
		Name:         "version_id",
		Required:     true,
		CollectionId: versionsCollection.Id,
	})

	collection.Fields.Add(&core.SelectField{
		Name:     "status",
		Required: true,
		Values:   []string{"pending", "running", "success", "failed"},
	})

	collection.Fields.Add(&core.TextField{
		Name: "logs",
		Max:  100000, // 100KB of logs
	})

	collection.Fields.Add(&core.DateField{
		Name: "started_at",
	})

	collection.Fields.Add(&core.DateField{
		Name: "completed_at",
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

	// Add indexes
	collection.AddIndex("idx_deployments_app", false, "app_id", "")
	collection.AddIndex("idx_deployments_status", false, "status", "")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("Failed to create deployments collection", "error", err)
		return err
	}

	app.Logger().Info("Created deployments collection")
	return nil
}
