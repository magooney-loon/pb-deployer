package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func RegisterHandlers(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		// Create API group
		// apiGroup := e.Router.Group("/api/v1")
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
		return e.Next()
	})
}
