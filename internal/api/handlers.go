package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.GET("/api/time", func(c *core.RequestEvent) error {
			now := time.Now()
			return c.JSON(http.StatusOK, map[string]any{
				"time": map[string]string{
					"iso":       now.Format(time.RFC3339),
					"unix":      strconv.FormatInt(now.Unix(), 10),
					"unix_nano": strconv.FormatInt(now.UnixNano(), 10),
					"utc":       now.UTC().Format(time.RFC3339),
				},
			})
		})

		// Register setup handlers on the same router
		RegisterSetupHandlers(e, app)

		return e.Next()
	})
}
