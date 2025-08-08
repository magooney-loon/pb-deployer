package models

import (
	"time"
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
