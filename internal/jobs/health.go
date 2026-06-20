package jobs

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	pbjobs "github.com/magooney-loon/pb-ext/core/jobs"
	"github.com/pocketbase/pocketbase/core"
)

const AppHealthJobID = "appHealthCheck"

type HealthCheckSummary struct {
	Checked int
	Online  int
	Offline int
	Skipped int
	Failed  int
}

func Register(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		if err := RegisterAppHealthCheckJob(app); err != nil {
			app.Logger().Error("Failed to register app health check job", "error", err)
			return err
		}
		return e.Next()
	})
}

func RegisterAppHealthCheckJob(app core.App) error {
	manager := pbjobs.GetManager()
	if manager == nil {
		return fmt.Errorf("job manager not initialized")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return manager.RegisterJob(
		AppHealthJobID,
		"App Health Check",
		"Checks every deployed app health endpoint once per minute and updates app status.",
		"* * * * *",
		func(logger *pbjobs.ExecutionLogger) {
			logger.Start("App Health Check")

			summary, err := CheckAppHealthStatuses(app, client)
			if err != nil {
				logger.Error("Health check failed: %v", err)
				logger.Fail(err)
				return
			}

			logger.Statistics(map[string]interface{}{
				"checked": summary.Checked,
				"online":  summary.Online,
				"offline": summary.Offline,
				"skipped": summary.Skipped,
				"failed":  summary.Failed,
			})
			logger.Complete(fmt.Sprintf("Checked %d apps: %d online, %d offline, %d skipped, %d failed updates",
				summary.Checked, summary.Online, summary.Offline, summary.Skipped, summary.Failed))
		},
	)
}

func CheckAppHealthStatuses(app core.App, client *http.Client) (HealthCheckSummary, error) {
	var summary HealthCheckSummary
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	records, err := app.FindRecordsByFilter("apps", "", "", 0, 0)
	if err != nil {
		return summary, fmt.Errorf("failed to load apps: %w", err)
	}

	// Apps with an in-flight deployment are skipped: their service is mid-restart,
	// so a probe would race the deploy and flip status spuriously.
	active, err := activeDeploymentAppIDs(app)
	if err != nil {
		app.Logger().Warn("health check: failed to load active deployments", "error", err)
	}

	for _, record := range records {
		domain := strings.TrimSpace(record.GetString("domain"))
		status := record.GetString("status")
		if domain == "" || active[record.Id] {
			summary.Skipped++
			continue
		}

		summary.Checked++
		nextStatus := "offline"
		if isHealthy(client, appHealthURL(domain)) {
			nextStatus = "online"
			summary.Online++
		} else {
			summary.Offline++
		}

		if status == nextStatus {
			continue
		}

		record.Set("status", nextStatus)
		if err := app.Save(record); err != nil {
			summary.Failed++
			app.Logger().Warn("Failed to update app health status",
				"app_id", record.Id,
				"domain", domain,
				"status", nextStatus,
				"error", err,
			)
		}
	}

	return summary, nil
}

// activeDeploymentAppIDs returns the set of app IDs that currently have a
// pending or running deployment.
func activeDeploymentAppIDs(app core.App) (map[string]bool, error) {
	deployments, err := app.FindRecordsByFilter(
		"deployments",
		"status = 'running' || status = 'pending'",
		"", 0, 0,
	)
	if err != nil {
		return map[string]bool{}, err
	}

	active := make(map[string]bool, len(deployments))
	for _, d := range deployments {
		active[d.GetString("app_id")] = true
	}
	return active, nil
}

func isHealthy(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
}

func appHealthURL(domain string) string {
	domain = strings.TrimRight(strings.TrimSpace(domain), "/")
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return domain + "/api/health"
	}
	return "https://" + domain + "/api/health"
}
