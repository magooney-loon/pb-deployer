package api

// API_SOURCE

import (
	"github.com/magooney-loon/pb-ext/core/server/api"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(pbApp core.App) {
	v1Config := &api.APIDocsConfig{
		Title:       "pb-deployer legacy",
		Version:     "1.0.0",
		Description: "legacy devops routes",
		Status:      "stable",
		Enabled:     true,
	}

	versions := map[string]*api.APIDocsConfig{
		"v1": v1Config,
	}
	versionManager := api.InitializeVersionedSystem(versions, "v1") // v1 is default/stable

	pbApp.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Get version-specific routers
		v1Router, err := versionManager.GetVersionRouter("v1", e)
		if err != nil {
			return err
		}

		v1Router.POST("/api/setup/server", func(c *core.RequestEvent) error {
			return handleServerSetup(c, pbApp)
		})

		v1Router.POST("/api/setup/security", func(c *core.RequestEvent) error {
			return handleServerSecurity(c, pbApp)
		})

		v1Router.POST("/api/setup/validate", func(c *core.RequestEvent) error {
			return handleServerValidation(c)
		})

		v1Router.POST("/api/deploy", func(c *core.RequestEvent) error {
			return handleDeploy(c, pbApp)
		})

		return e.Next()
	})

	versionManager.RegisterWithServer(pbApp)
}
