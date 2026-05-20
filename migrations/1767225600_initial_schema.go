package migrations

import (
	"fmt"

	"pb-deployer/internal/models"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if err := models.RegisterCollections(app); err != nil {
			return err
		}

		if err := backfillLegacyApps(app); err != nil {
			return err
		}

		return addPostBackfillAppIndexes(app)
	}, nil)
}

func backfillLegacyApps(app core.App) error {
	apps, err := app.FindRecordsByFilter("apps", "", "server_id, created", 0, 0)
	if err != nil {
		return err
	}

	usedPorts := make(map[string]map[int]bool)
	for _, record := range apps {
		serverID := record.GetString("server_id")
		if usedPorts[serverID] == nil {
			usedPorts[serverID] = make(map[int]bool)
		}

		if port := record.GetInt("http_port"); port > 0 {
			usedPorts[serverID][port] = true
		}
	}

	for _, record := range apps {
		serverID := record.GetString("server_id")
		if record.GetInt("http_port") > 0 {
			continue
		}

		port, err := nextFreePort(usedPorts[serverID])
		if err != nil {
			return fmt.Errorf("failed to assign http_port for app %s: %w", record.Id, err)
		}

		record.Set("http_port", port)
		record.Set("status", "needs_migration")
		usedPorts[serverID][port] = true

		if err := app.Save(record); err != nil {
			return fmt.Errorf("failed to backfill app %s: %w", record.Id, err)
		}
	}

	return nil
}

func addPostBackfillAppIndexes(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		return err
	}

	collection.AddIndex("idx_apps_port_per_server", true, "server_id, http_port", "")
	return app.Save(collection)
}

func nextFreePort(used map[int]bool) (int, error) {
	for port := 8090; port <= 8999; port++ {
		if !used[port] {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports available in [8090, 8999]")
}
