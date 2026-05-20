package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/dbutils"
)

func TestInitialSchemaMigrationCreatesCollections(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("NewTestApp failed: %v", err)
	}
	defer app.Cleanup()

	apps, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		t.Fatalf("apps collection missing: %v", err)
	}

	if apps.CreateRule != nil {
		t.Fatalf("apps CreateRule = %v, want nil to force /api/apps creation", *apps.CreateRule)
	}
	if apps.Fields.GetByName("http_port") == nil {
		t.Fatal("apps.http_port field missing")
	}
	if status, ok := apps.Fields.GetByName("status").(*core.SelectField); !ok {
		t.Fatal("apps.status is not a select field")
	} else {
		expected := []string{"online", "offline", "unknown"}
		if len(status.Values) != len(expected) {
			t.Fatalf("apps.status values = %v, want %v", status.Values, expected)
		}
	}
	if !hasIndex(apps, "idx_apps_name_per_server") {
		t.Fatal("idx_apps_name_per_server missing")
	}
	if !hasIndex(apps, "idx_apps_domain_per_server") {
		t.Fatal("idx_apps_domain_per_server missing")
	}
	if !hasIndex(apps, "idx_apps_port_per_server") {
		t.Fatal("idx_apps_port_per_server missing")
	}

	servers, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		t.Fatalf("servers collection missing: %v", err)
	}
	if servers.Fields.GetByName("proxy_email") == nil {
		t.Fatal("servers.proxy_email field missing")
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasIndex(collection *core.Collection, name string) bool {
	for _, index := range collection.Indexes {
		if dbutils.ParseIndex(index).IndexName == name {
			return true
		}
	}
	return false
}
