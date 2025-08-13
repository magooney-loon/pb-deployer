package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Version represents a version of a deployed PocketBase application
type Version struct {
	ID            string    `json:"id" db:"id"`
	Created       time.Time `json:"created" db:"created"`
	Updated       time.Time `json:"updated" db:"updated"`
	AppID         string    `json:"app_id" db:"app_id"`
	VersionNum    string    `json:"version_number" db:"version_number"`
	DeploymentZip string    `json:"deployment_zip" db:"deployment_zip"` // Single zip containing binary and pb_public folder
	Notes         string    `json:"notes" db:"notes"`
}

// TableName returns the collection name for the Version model
func (v *Version) TableName() string {
	return "versions"
}

// NewVersion creates a new Version instance with default values
func NewVersion() *Version {
	return &Version{}
}

// HasDeploymentZip checks if this version has a deployment zip file
func (v *Version) HasDeploymentZip() bool {
	return v.DeploymentZip != ""
}

// HasNotes checks if this version has release notes
func (v *Version) HasNotes() bool {
	return v.Notes != ""
}

// GetVersionString returns a formatted version string
func (v *Version) GetVersionString() string {
	if v.VersionNum == "" {
		return "unknown"
	}
	return v.VersionNum
}

// CreateCollection creates the versions collection in the database
func (v *Version) CreateCollection(app core.App) error {
	app.Logger().Info("createVersionsCollection: Starting versions collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("versions")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createVersionsCollection: Versions collection already exists")
		return nil
	}

	// Find apps collection for relation
	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		app.Logger().Error("createVersionsCollection: Apps collection not found", "error", err)
		return err
	}

	// Create new collection
	collection := core.NewBaseCollection("versions")

	// Add relation field to app FIRST
	collection.Fields.Add(&core.RelationField{
		Name:          "app_id",
		Required:      true,
		CollectionId:  appsCollection.Id,
		CascadeDelete: true,
	})

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	// Add other fields
	collection.Fields.Add(&core.TextField{
		Name:     "version_number",
		Required: true,
		Max:      50,
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

	// Add indexes for common queries and relations
	collection.AddIndex("idx_versions_app", false, "app_id", "")
	collection.AddIndex("idx_versions_version", false, "version_number", "")
	collection.AddIndex("idx_versions_app_version", true, "app_id", "version_number")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createVersionsCollection: Failed to save versions collection", "error", err)
		return err
	}

	app.Logger().Info("createVersionsCollection: Successfully created versions collection")
	return nil
}
