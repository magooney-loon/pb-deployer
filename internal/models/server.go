package models

import (
	"time"
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
