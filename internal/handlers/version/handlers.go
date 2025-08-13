package version

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// RegisterVersionHandlers registers all version-related HTTP handlers
func RegisterVersionHandlers(app core.App, group *router.RouterGroup[*core.RequestEvent]) {
	// Version CRUD endpoints
	group.GET("/versions", func(e *core.RequestEvent) error {
		return listVersions(app, e)
	})

	group.POST("/versions", func(e *core.RequestEvent) error {
		return createVersion(app, e)
	})

	group.GET("/versions/{id}", func(e *core.RequestEvent) error {
		return getVersion(app, e)
	})

	group.PUT("/versions/{id}", func(e *core.RequestEvent) error {
		return updateVersion(app, e)
	})

	group.DELETE("/versions/{id}", func(e *core.RequestEvent) error {
		return deleteVersion(app, e)
	})

	// Version file upload/download endpoints
	group.POST("/versions/{id}/upload", func(e *core.RequestEvent) error {
		return uploadVersionZip(app, e)
	})

	group.GET("/versions/{id}/download", func(e *core.RequestEvent) error {
		return downloadVersionZip(app, e)
	})

	// App-specific version endpoints
	group.GET("/apps/{app_id}/versions", func(e *core.RequestEvent) error {
		return listAppVersions(app, e)
	})

	group.POST("/apps/{app_id}/versions", func(e *core.RequestEvent) error {
		return createAppVersion(app, e)
	})

	// Version validation endpoint
	group.POST("/versions/{id}/validate", func(e *core.RequestEvent) error {
		return validateVersion(app, e)
	})

	// Version metadata endpoints
	group.GET("/versions/{id}/metadata", func(e *core.RequestEvent) error {
		return getVersionMetadata(app, e)
	})

	group.PUT("/versions/{id}/metadata", func(e *core.RequestEvent) error {
		return updateVersionMetadata(app, e)
	})
}
