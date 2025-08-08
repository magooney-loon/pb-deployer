package models

import (
	"time"
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
