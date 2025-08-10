package apps

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RegisterAppHandlers registers all app-related HTTP handlers
func RegisterAppsHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
	// App CRUD endpoints
	group.GET("/apps", func(e *core.RequestEvent) error {
		return listApps(app, e)
	})

	group.POST("/apps", func(e *core.RequestEvent) error {
		return createApp(app, e)
	})

	group.GET("/apps/{id}", func(e *core.RequestEvent) error {
		return getApp(app, e)
	})

	group.PUT("/apps/{id}", func(e *core.RequestEvent) error {
		return updateApp(app, e)
	})

	group.DELETE("/apps/{id}", func(e *core.RequestEvent) error {
		return deleteApp(app, e)
	})

	// App status and health endpoints
	group.GET("/apps/{id}/status", func(e *core.RequestEvent) error {
		return getAppStatus(app, e)
	})

	group.POST("/apps/{id}/health-check", func(e *core.RequestEvent) error {
		return checkAppHealth(app, e)
	})

	// App deployment endpoints
	group.POST("/apps/{id}/deploy", func(e *core.RequestEvent) error {
		return deployApp(app, e)
	})

	group.POST("/apps/{id}/rollback", func(e *core.RequestEvent) error {
		return rollbackApp(app, e)
	})

	// Service management endpoints
	group.POST("/apps/{id}/start", func(e *core.RequestEvent) error {
		return startAppService(app, e)
	})

	group.POST("/apps/{id}/stop", func(e *core.RequestEvent) error {
		return stopAppService(app, e)
	})

	group.POST("/apps/{id}/restart", func(e *core.RequestEvent) error {
		return restartAppService(app, e)
	})

	// Logs endpoint
	group.GET("/apps/{id}/logs", func(e *core.RequestEvent) error {
		return getAppLogs(app, e)
	})

	// WebSocket endpoints for real-time updates
	group.GET("/apps/{id}/deploy-ws", func(e *core.RequestEvent) error {
		return handleDeploymentWebSocket(app, e)
	})
}
