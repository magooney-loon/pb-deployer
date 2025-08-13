package server

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"golang.org/x/sync/errgroup"

	"pb-deployer/internal/ssh"
)

// notifySetupProgress sends setup progress updates to all subscribed clients
func notifySetupProgress(app core.App, serverID string, step ssh.SetupStep) error {
	subscription := fmt.Sprintf("server_setup_%s", serverID)
	return notifyClients(app, subscription, step)
}

// notifySecurityProgress sends security progress updates to all subscribed clients
func notifySecurityProgress(app core.App, serverID string, step ssh.SetupStep) error {
	subscription := fmt.Sprintf("server_security_%s", serverID)
	return notifyClients(app, subscription, step)
}

// notifyClients sends a message to all clients subscribed to a specific topic
func notifyClients(app core.App, subscription string, data any) error {
	// Add debugging to see what we're sending
	app.Logger().Debug("Sending realtime message",
		"subscription", subscription,
		"data", data)

	rawData, err := json.Marshal(data)
	if err != nil {
		app.Logger().Error("Failed to marshal data for realtime",
			"subscription", subscription,
			"data", data,
			"error", err)
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	app.Logger().Debug("Marshaled data",
		"subscription", subscription,
		"raw_data", string(rawData))

	message := subscriptions.Message{
		Name: subscription,
		Data: rawData,
	}

	group := new(errgroup.Group)
	chunks := app.SubscriptionsBroker().ChunkedClients(300)

	clientCount := 0
	for _, chunk := range chunks {
		group.Go(func() error {
			for _, client := range chunk {
				if !client.HasSubscription(subscription) {
					continue
				}
				clientCount++
				client.Send(message)
			}
			return nil
		})
	}

	app.Logger().Debug("Sent realtime message to clients",
		"subscription", subscription,
		"client_count", clientCount)

	return group.Wait()
}
