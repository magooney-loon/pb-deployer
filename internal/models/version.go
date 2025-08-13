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
