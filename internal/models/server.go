package models

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Server represents a remote server where PocketBase apps can be deployed
type Server struct {
	ID             string    `json:"id" db:"id"`
	Created        time.Time `json:"created" db:"created"`
	Updated        time.Time `json:"updated" db:"updated"`
	Name           string    `json:"name" db:"name"`
	Host           string    `json:"host" db:"host"`
	Port           int       `json:"port" db:"port"`
	RootUsername   string    `json:"root_username" db:"root_username"`
	AppUsername    string    `json:"app_username" db:"app_username"`
	UseSSHAgent    bool      `json:"use_ssh_agent" db:"use_ssh_agent"`
	ManualKeyPath  string    `json:"manual_key_path" db:"manual_key_path"`
	SetupComplete  bool      `json:"setup_complete" db:"setup_complete"`
	SecurityLocked bool      `json:"security_locked" db:"security_locked"`
}

// TableName returns the collection name for the Server model
func (s *Server) TableName() string {
	return "servers"
}

// NewServer creates a new Server instance with default values
func NewServer() *Server {
	return &Server{
		Port:           22,
		RootUsername:   "root",
		AppUsername:    "pocketbase",
		UseSSHAgent:    true,
		SetupComplete:  false,
		SecurityLocked: false,
	}
}

// GetSSHAddress returns the SSH connection address
func (s *Server) GetSSHAddress() string {
	if s.Port == 22 {
		return s.Host
	}
	return s.Host + ":" + string(rune(s.Port))
}

// IsReadyForDeployment checks if server is properly set up for deployments
func (s *Server) IsReadyForDeployment() bool {
	return s.SetupComplete && s.SecurityLocked
}

// IsSetupComplete checks if the initial server setup is finished
func (s *Server) IsSetupComplete() bool {
	return s.SetupComplete
}

// IsSecurityLocked checks if security hardening is applied
func (s *Server) IsSecurityLocked() bool {
	return s.SecurityLocked
}

// CreateCollection creates the servers collection in the database
func (s *Server) CreateCollection(app core.App) error {
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
