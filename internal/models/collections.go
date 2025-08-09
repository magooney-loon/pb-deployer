package models

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// RegisterCollections creates and registers all the database collections
func RegisterCollections(app core.App) error {
	app.Logger().Info("RegisterCollections: Starting collection registration")

	// Create servers collection
	if err := createServersCollection(app); err != nil {
		app.Logger().Error("RegisterCollections: Failed to create servers collection", "error", err)
		return err
	}

	// Create apps collection
	if err := createAppsCollection(app); err != nil {
		app.Logger().Error("RegisterCollections: Failed to create apps collection", "error", err)
		return err
	}

	// Create versions collection
	if err := createVersionsCollection(app); err != nil {
		app.Logger().Error("RegisterCollections: Failed to create versions collection", "error", err)
		return err
	}

	// Create deployments collection
	if err := createDeploymentsCollection(app); err != nil {
		app.Logger().Error("RegisterCollections: Failed to create deployments collection", "error", err)
		return err
	}

	app.Logger().Info("RegisterCollections: All collections registered successfully")
	return nil
}

// createServersCollection creates the servers collection
func createServersCollection(app core.App) error {
	app.Logger().Info("createServersCollection: Starting servers collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("servers")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createServersCollection: Servers collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("servers")

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
		Name:     "host",
		Required: true,
		Max:      255,
	})

	collection.Fields.Add(&core.NumberField{
		Name: "port",
		Min:  types.Pointer(1.0),
		Max:  types.Pointer(65535.0),
	})

	collection.Fields.Add(&core.TextField{
		Name: "root_username",
		Max:  50,
	})

	collection.Fields.Add(&core.TextField{
		Name: "app_username",
		Max:  50,
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

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createServersCollection: Failed to save servers collection", "error", err)
		return err
	}

	app.Logger().Info("createServersCollection: Successfully created servers collection")
	return nil
}

// createAppsCollection creates the apps collection
func createAppsCollection(app core.App) error {
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

// createVersionsCollection creates the versions collection
func createVersionsCollection(app core.App) error {
	app.Logger().Info("createVersionsCollection: Starting versions collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("versions")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createVersionsCollection: Versions collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("versions")

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	// Add fields with minimal validation
	collection.Fields.Add(&core.TextField{
		Name:     "app_id",
		Required: true,
		Max:      15,
	})

	collection.Fields.Add(&core.TextField{
		Name: "version_number",
		Max:  50,
	})

	collection.Fields.Add(&core.FileField{
		Name:      "deployment_zip",
		MaxSelect: 1,
		MaxSize:   157286400, // 150MB
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

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createVersionsCollection: Failed to save versions collection", "error", err)
		return err
	}

	app.Logger().Info("createVersionsCollection: Successfully created versions collection")
	return nil
}

// createDeploymentsCollection creates the deployments collection
func createDeploymentsCollection(app core.App) error {
	app.Logger().Info("createDeploymentsCollection: Starting deployments collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("deployments")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createDeploymentsCollection: Deployments collection already exists")
		return nil
	}

	// Create new collection
	collection := core.NewBaseCollection("deployments")

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	// Add fields with minimal validation
	collection.Fields.Add(&core.TextField{
		Name:     "app_id",
		Required: true,
		Max:      15,
	})

	collection.Fields.Add(&core.TextField{
		Name: "version_id",
		Max:  15,
	})

	collection.Fields.Add(&core.SelectField{
		Name:   "status",
		Values: []string{"pending", "running", "success", "failed"},
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

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createDeploymentsCollection: Failed to save deployments collection", "error", err)
		return err
	}

	app.Logger().Info("createDeploymentsCollection: Successfully created deployments collection")
	return nil
}
