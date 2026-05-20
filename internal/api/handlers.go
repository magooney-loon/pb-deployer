package api

// API_SOURCE

import (
	"github.com/magooney-loon/pb-ext/core/server/api"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(pbApp core.App) {
	RegisterAppDeleteHook(pbApp)

	versionManager := InitVersionedSystem()

	pbApp.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := versionManager.RegisterAllVersionRoutes(e); err != nil {
			return err
		}
		return e.Next()
	})

	versionManager.RegisterWithServer(pbApp)
}

// InitVersionedSystem builds the API version manager.
// Exported so main.go can reuse it for spec generation (pass nil app — handlers won't be called).
func InitVersionedSystem() *api.APIVersionManager {
	v1Config := &api.APIDocsConfig{
		Title:         "pb-deployer",
		Version:       "1.0.0",
		Description:   "PocketBase deployment management API",
		Status:        "stable",
		Enabled:       true,
		PublicSwagger: true,
	}

	return api.InitializeVersionedSystemWithRoutes(map[string]*api.VersionSetup{
		"v1": {
			Config: v1Config,
			Routes: registerV1Routes,
		},
	}, "v1")
}

func registerV1Routes(router *api.VersionedAPIRouter) {
	router.POST("/api/setup/server", handleServerSetup)
	router.POST("/api/setup/security", handleServerSecurity)
	router.POST("/api/setup/validate", handleServerValidation)
	router.POST("/api/deploy", handleDeploy)
	router.POST("/api/apps", handleCreateApp)
	router.GET("/api/terminal", handleTerminal)
}
