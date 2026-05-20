package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/dbutils"
)

func TestInitialSchemaMigrationCreatesMultiAppFields(t *testing.T) {
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
	} else if !contains(status.Values, "needs_migration") {
		t.Fatalf("apps.status values = %v, want needs_migration", status.Values)
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

func TestBackfillLegacyAppsAssignsPortsPerServerAndMarksMigration(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("NewTestApp failed: %v", err)
	}
	defer app.Cleanup()

	servers, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		t.Fatalf("servers collection missing: %v", err)
	}
	apps, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		t.Fatalf("apps collection missing: %v", err)
	}
	apps.RemoveIndex("idx_apps_port_per_server")
	if err := app.Save(apps); err != nil {
		t.Fatalf("remove port index for legacy fixture: %v", err)
	}

	serverA := core.NewRecord(servers)
	serverA.Set("name", "server-a")
	serverA.Set("host", "10.0.0.1")
	serverA.Set("port", 22)
	serverA.Set("root_username", "root")
	serverA.Set("app_username", "pocketbase")
	if err := app.SaveNoValidate(serverA); err != nil {
		t.Fatalf("save serverA: %v", err)
	}

	serverB := core.NewRecord(servers)
	serverB.Set("name", "server-b")
	serverB.Set("host", "10.0.0.2")
	serverB.Set("port", 22)
	serverB.Set("root_username", "root")
	serverB.Set("app_username", "pocketbase")
	if err := app.SaveNoValidate(serverB); err != nil {
		t.Fatalf("save serverB: %v", err)
	}

	legacyA1 := legacyAppRecord(apps, serverA.Id, "blog", "blog.example.com")
	legacyA2 := legacyAppRecord(apps, serverA.Id, "shop", "shop.example.com")
	legacyB1 := legacyAppRecord(apps, serverB.Id, "blog", "blog.staging.example.com")
	for _, record := range []*core.Record{legacyA1, legacyA2, legacyB1} {
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatalf("save legacy app %s: %v", record.GetString("name"), err)
		}
	}

	if err := backfillLegacyApps(app); err != nil {
		t.Fatalf("backfillLegacyApps failed: %v", err)
	}

	freshA1, _ := app.FindRecordById("apps", legacyA1.Id)
	freshA2, _ := app.FindRecordById("apps", legacyA2.Id)
	freshB1, _ := app.FindRecordById("apps", legacyB1.Id)

	if freshA1.GetInt("http_port") != 8090 {
		t.Fatalf("legacyA1 http_port = %d, want 8090", freshA1.GetInt("http_port"))
	}
	if freshA2.GetInt("http_port") != 8091 {
		t.Fatalf("legacyA2 http_port = %d, want 8091", freshA2.GetInt("http_port"))
	}
	if freshB1.GetInt("http_port") != 8090 {
		t.Fatalf("legacyB1 http_port = %d, want 8090 reused on different server", freshB1.GetInt("http_port"))
	}

	for _, record := range []*core.Record{freshA1, freshA2, freshB1} {
		if record.GetString("status") != "needs_migration" {
			t.Fatalf("app %s status = %q, want needs_migration", record.Id, record.GetString("status"))
		}
	}
}

func legacyAppRecord(collection *core.Collection, serverID string, name string, domain string) *core.Record {
	record := core.NewRecord(collection)
	record.Set("server_id", serverID)
	record.Set("name", name)
	record.Set("domain", domain)
	record.Set("remote_path", "/opt/pocketbase/apps/"+name)
	record.Set("service_name", "pocketbase-"+name)
	record.Set("status", "offline")
	return record
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
