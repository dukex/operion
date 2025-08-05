package webhook

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

func NewWebhookTriggerFactory() protocol.TriggerFactory {
	return &WebhookTriggerFactory{}
}

type WebhookTriggerFactory struct{}

func (f *WebhookTriggerFactory) ID() string {
	return "webhook"
}

func (f *WebhookTriggerFactory) Name() string {
	return "Webhook"
}

func (f *WebhookTriggerFactory) Description() string {
	return "Trigger workflow execution via HTTP webhook endpoints"
}

func (f *WebhookTriggerFactory) Schema() map[string]any {
	return map[string]any{
		"type":        "object",
		"title":       "Webhook Trigger Configuration",
		"description": "Configuration for HTTP webhook-based workflow triggering",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "URL path for the webhook endpoint (e.g., '/webhooks/github')",
				"pattern":     `^/.*`,
				"examples":    []string{"/webhooks/github", "/api/events/user", "/triggers/payment"},
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method to accept for this webhook",
				"enum":        []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
				"default":     "POST",
				"examples":    []string{"POST", "PUT", "GET"},
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Required headers for webhook validation",
				"additionalProperties": map[string]any{
					"type": "string",
				},
				"examples": []map[string]any{
					{"Authorization": "Bearer token123"},
					{"X-API-Key": "secret-key", "Content-Type": "application/json"},
				},
			},
			"enabled": map[string]any{
				"type":        "boolean",
				"description": "Whether this webhook trigger is active",
				"default":     true,
				"examples":    []bool{true, false},
			},
		},
		"required": []string{"path"},
		"examples": []map[string]any{
			{
				"path":   "/webhooks/github",
				"method": "POST",
				"headers": map[string]string{
					"X-GitHub-Event": "push",
				},
			},
			{
				"path":    "/api/payments/webhook",
				"method":  "POST",
				"enabled": true,
			},
			{
				"path": "/webhooks/simple",
			},
		},
	}
}

func (f *WebhookTriggerFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	trigger, err := NewWebhookTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook trigger: %w", err)
	}

	return trigger, nil
}
