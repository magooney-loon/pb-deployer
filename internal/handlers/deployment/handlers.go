package deployment

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RegisterDeploymentHandlers registers all deployment-related HTTP handlers
func RegisterDeploymentHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
	// Deployment listing and details
	group.GET("/deployments", func(e *core.RequestEvent) error {
		return listDeployments(app, e)
	})

	group.GET("/deployments/{id}", func(e *core.RequestEvent) error {
		return getDeployment(app, e)
	})

	// Deployment status and logs
	group.GET("/deployments/{id}/status", func(e *core.RequestEvent) error {
		return getDeploymentStatus(app, e)
	})

	group.GET("/deployments/{id}/logs", func(e *core.RequestEvent) error {
		return getDeploymentLogs(app, e)
	})

	// Deployment control
	group.POST("/deployments/{id}/cancel", func(e *core.RequestEvent) error {
		return cancelDeployment(app, e)
	})

	group.POST("/deployments/{id}/retry", func(e *core.RequestEvent) error {
		return retryDeployment(app, e)
	})

	// App-specific deployment endpoints
	group.GET("/apps/{app_id}/deployments", func(e *core.RequestEvent) error {
		return listAppDeployments(app, e)
	})

	group.GET("/apps/{app_id}/deployments/latest", func(e *core.RequestEvent) error {
		return getLatestAppDeployment(app, e)
	})

	// WebSocket endpoints for real-time updates
	group.GET("/deployments/{id}/ws", func(e *core.RequestEvent) error {
		return handleDeploymentProgressWebSocket(app, e)
	})

	// Deployment statistics and analytics
	group.GET("/deployments/stats", func(e *core.RequestEvent) error {
		return getDeploymentStats(app, e)
	})

	// Bulk operations
	group.POST("/deployments/cleanup", func(e *core.RequestEvent) error {
		return cleanupOldDeployments(app, e)
	})
}
