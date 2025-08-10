package server

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RegisterServerHandlers registers all server-related HTTP handlers
func RegisterServerHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
	// Connection and status endpoints
	group.POST("/servers/{id}/test", func(e *core.RequestEvent) error {
		return testServerConnection(app, e)
	})

	group.GET("/servers/{id}/status", func(e *core.RequestEvent) error {
		return getServerStatus(app, e)
	})

	// Setup endpoints
	group.POST("/servers/{id}/setup", func(e *core.RequestEvent) error {
		return runServerSetup(app, e)
	})

	// Security endpoints
	group.POST("/servers/{id}/security", func(e *core.RequestEvent) error {
		return applySecurityLockdown(app, e)
	})

	// WebSocket endpoints
	group.GET("/servers/{id}/setup-ws", func(e *core.RequestEvent) error {
		return handleSetupWebSocket(app, e)
	})

	group.GET("/servers/{id}/security-ws", func(e *core.RequestEvent) error {
		return handleSecurityWebSocket(app, e)
	})
}
