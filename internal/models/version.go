package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

type Version struct {
	ID            string    `json:"id" db:"id"`
	Created       time.Time `json:"created" db:"created"`
	Updated       time.Time `json:"updated" db:"updated"`
	AppID         string    `json:"app_id" db:"app_id"`
	VersionNum    string    `json:"version_number" db:"version_number"`
	DeploymentZip string    `json:"deployment_zip" db:"deployment_zip"` // Version zip containing binary and pb_public folder
	Notes         string    `json:"notes" db:"notes"`
}

func (v *Version) TableName() string {
	return "versions"
}

func NewVersion() *Version {
	return &Version{}
}

func (v *Version) HasDeploymentZip() bool {
	return v.DeploymentZip != ""
}

func (v *Version) HasNotes() bool {
	return v.Notes != ""
}

func (v *Version) GetVersionString() string {
	if v.VersionNum == "" {
		return "unknown"
	}
	return v.VersionNum
}

func (v *Version) CreateCollection(app core.App) error {
	app.Logger().Info("createVersionsCollection: Starting versions collection creation")

	existingCollection, err := app.FindCollectionByNameOrId("versions")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createVersionsCollection: Versions collection already exists")
		return nil
	}

	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		app.Logger().Error("createVersionsCollection: Apps collection not found", "error", err)
		return err
	}

	collection := core.NewBaseCollection("versions")

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

	collection.Fields.Add(&core.AutodateField{
		Name:     "created",
		OnCreate: true,
	})

	collection.Fields.Add(&core.AutodateField{
		Name:     "updated",
		OnCreate: true,
		OnUpdate: true,
	})

	collection.AddIndex("idx_versions_app", false, "app_id", "")
	collection.AddIndex("idx_versions_version", false, "version_number", "")

	if err := app.Save(collection); err != nil {
		app.Logger().Error("createVersionsCollection: Failed to save versions collection", "error", err)
		return err
	}

	app.Logger().Info("createVersionsCollection: Successfully created versions collection")
	return nil
}
