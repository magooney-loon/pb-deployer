package migrations

import (
	"pb-deployer/internal/models"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		return models.RegisterCollections(app)
	}, nil)
}
