package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Deployment represents a deployment operation for a PocketBase application
type Deployment struct {
	ID          string     `json:"id" db:"id"`
	Created     time.Time  `json:"created" db:"created"`
	Updated     time.Time  `json:"updated" db:"updated"`
	AppID       string     `json:"app_id" db:"app_id"`
	VersionID   string     `json:"version_id" db:"version_id"`
	Status      string     `json:"status" db:"status"` // pending/running/success/failed
	Logs        string     `json:"logs" db:"logs"`
	StartedAt   *time.Time `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
}

// TableName returns the collection name for the Deployment model
func (d *Deployment) TableName() string {
	return "deployments"
}

// NewDeployment creates a new Deployment instance with default values
func NewDeployment() *Deployment {
	now := time.Now()
	return &Deployment{
		Status:    "pending",
		StartedAt: &now,
	}
}

// IsRunning checks if the deployment is currently running
func (d *Deployment) IsRunning() bool {
	return d.Status == "running"
}

// IsComplete checks if the deployment has finished (success or failed)
func (d *Deployment) IsComplete() bool {
	return d.Status == "success" || d.Status == "failed"
}

// IsSuccessful checks if the deployment completed successfully
func (d *Deployment) IsSuccessful() bool {
	return d.Status == "success"
}

// IsFailed checks if the deployment failed
func (d *Deployment) IsFailed() bool {
	return d.Status == "failed"
}

// GetDuration returns the deployment duration if completed
func (d *Deployment) GetDuration() *time.Duration {
	if d.StartedAt == nil || d.CompletedAt == nil {
		return nil
	}
	duration := d.CompletedAt.Sub(*d.StartedAt)
	return &duration
}

// MarkAsRunning updates the deployment status to running
func (d *Deployment) MarkAsRunning() {
	d.Status = "running"
	if d.StartedAt == nil {
		now := time.Now()
		d.StartedAt = &now
	}
}

// MarkAsSuccess updates the deployment status to success and sets completion time
func (d *Deployment) MarkAsSuccess() {
	d.Status = "success"
	now := time.Now()
	d.CompletedAt = &now
}

// MarkAsFailed updates the deployment status to failed and sets completion time
func (d *Deployment) MarkAsFailed() {
	d.Status = "failed"
	now := time.Now()
	d.CompletedAt = &now
}

// AppendLog adds a log entry to the deployment logs
func (d *Deployment) AppendLog(message string) {
	if d.Logs == "" {
		d.Logs = message
	} else {
		d.Logs += "\n" + message
	}
}

// CreateCollection creates the deployments collection in the database
func (d *Deployment) CreateCollection(app core.App) error {
	app.Logger().Info("createDeploymentsCollection: Starting deployments collection creation")

	// Check if collection already exists
	existingCollection, err := app.FindCollectionByNameOrId("deployments")
	if err == nil && existingCollection != nil {
		app.Logger().Info("createDeploymentsCollection: Deployments collection already exists")
		return nil
	}

	// Find apps collection for relation
	appsCollection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		app.Logger().Error("createDeploymentsCollection: Apps collection not found", "error", err)
		return err
	}

	// Find versions collection for relation
	versionsCollection, err := app.FindCollectionByNameOrId("versions")
	if err != nil {
		app.Logger().Error("createDeploymentsCollection: Versions collection not found", "error", err)
		return err
	}

	// Create new collection
	collection := core.NewBaseCollection("deployments")

	// Add relation fields FIRST
	collection.Fields.Add(&core.RelationField{
		Name:          "app_id",
		Required:      true,
		CollectionId:  appsCollection.Id,
		CascadeDelete: true,
	})

	collection.Fields.Add(&core.RelationField{
		Name:          "version_id",
		Required:      false, // Some deployments might not have a specific version
		CollectionId:  versionsCollection.Id,
		CascadeDelete: true,
	})

	// Set permissions to allow all operations (local-only tool)
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = types.Pointer("")
	collection.UpdateRule = types.Pointer("")
	collection.DeleteRule = types.Pointer("")

	// Add other fields

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

	// Add indexes for common queries and relations
	collection.AddIndex("idx_deployments_app", false, "app_id", "")
	collection.AddIndex("idx_deployments_version", false, "version_id", "")
	collection.AddIndex("idx_deployments_status", false, "status", "")
	collection.AddIndex("idx_deployments_app_status", false, "app_id", "status")
	collection.AddIndex("idx_deployments_created", false, "created", "")

	// Save the collection
	if err := app.Save(collection); err != nil {
		app.Logger().Error("createDeploymentsCollection: Failed to save deployments collection", "error", err)
		return err
	}

	app.Logger().Info("createDeploymentsCollection: Successfully created deployments collection")
	return nil
}
