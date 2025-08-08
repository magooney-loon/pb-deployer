package models

import (
	"time"
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
