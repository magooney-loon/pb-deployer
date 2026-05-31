package jobs

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	_ "pb-deployer/internal/migrations"
)

func TestAppHealthURL(t *testing.T) {
	tests := []struct {
		name   string
		domain string
		want   string
	}{
		{name: "plain domain", domain: "example.com", want: "https://example.com/api/health"},
		{name: "https URL", domain: "https://example.com", want: "https://example.com/api/health"},
		{name: "http URL with slash", domain: "http://example.com/", want: "http://example.com/api/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appHealthURL(tt.domain); got != tt.want {
				t.Fatalf("appHealthURL(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestCheckAppHealthStatusesUpdatesOnlineOffline(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatalf("NewTestApp failed: %v", err)
	}
	defer app.Cleanup()

	healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer healthyServer.Close()

	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer unhealthyServer.Close()

	apps, err := app.FindCollectionByNameOrId("apps")
	if err != nil {
		t.Fatalf("apps collection missing: %v", err)
	}
	servers, err := app.FindCollectionByNameOrId("servers")
	if err != nil {
		t.Fatalf("servers collection missing: %v", err)
	}

	server := core.NewRecord(servers)
	server.Set("name", "server-a")
	server.Set("host", "10.0.0.1")
	server.Set("port", 22)
	server.Set("root_username", "root")
	server.Set("app_username", "pocketbase")
	if err := app.SaveNoValidate(server); err != nil {
		t.Fatalf("save server: %v", err)
	}

	online := appRecord(apps, server.Id, "online-app", healthyServer.URL, 8090, "offline")
	offline := appRecord(apps, server.Id, "offline-app", unhealthyServer.URL, 8091, "online")
	for _, record := range []*core.Record{online, offline} {
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatalf("save app %s: %v", record.GetString("name"), err)
		}
	}

	summary, err := CheckAppHealthStatuses(app, &http.Client{Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("CheckAppHealthStatuses failed: %v", err)
	}

	if summary.Checked != 2 || summary.Online != 1 || summary.Offline != 1 || summary.Skipped != 0 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want checked=2 online=1 offline=1 skipped=0 failed=0", summary)
	}

	freshOnline, _ := app.FindRecordById("apps", online.Id)
	freshOffline, _ := app.FindRecordById("apps", offline.Id)

	if got := freshOnline.GetString("status"); got != "online" {
		t.Fatalf("online app status = %q, want online", got)
	}
	if got := freshOffline.GetString("status"); got != "offline" {
		t.Fatalf("offline app status = %q, want offline", got)
	}
}

func appRecord(collection *core.Collection, serverID string, name string, domain string, port int, status string) *core.Record {
	record := core.NewRecord(collection)
	record.Set("server_id", serverID)
	record.Set("name", name)
	record.Set("domain", domain)
	record.Set("remote_path", "/opt/pocketbase/apps/"+name)
	record.Set("service_name", "pocketbase-"+name)
	record.Set("http_port", port)
	record.Set("status", status)
	return record
}
