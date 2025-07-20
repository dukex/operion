// Package webhook provides HTTP webhook trigger implementation with centralized server management.
package webhook

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

type WebhookTrigger struct {
	ID         string
	Path       string
	WorkflowId string
	Enabled    bool
	callback   protocol.TriggerCallback
	logger     *slog.Logger
}

func NewWebhookTrigger(config map[string]interface{}, logger *slog.Logger) (*WebhookTrigger, error) {
	id, _ := config["id"].(string)
	workflowId, _ := config["workflow_id"].(string)
	path, ok := config["path"].(string)
	if !ok {
		path = "/webhook"
	}

	trigger := &WebhookTrigger{
		ID:         id,
		Path:       path,
		Enabled:    true,
		WorkflowId: workflowId,
		logger: logger.With(
			"module", "webhook_trigger",
			"id", id,
			"path", path,
			"workflow_id", workflowId,
		),
	}

	if err := trigger.Validate(); err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *WebhookTrigger) Validate() error {
	if t.ID == "" {
		return errors.New("webhook trigger ID is required")
	}
	if t.Path == "" {
		return errors.New("webhook trigger path is required")
	}
	if t.Path[0] != '/' {
		return errors.New("webhook trigger path must start with '/'")
	}
	return nil
}

func (t *WebhookTrigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	if !t.Enabled {
		t.logger.Info("WebhookTrigger is disabled.")
		return nil
	}

	manager := GetGlobalWebhookServerManager()
	if manager == nil {
		return errors.New("global webhook server manager not initialized")
	}

	t.logger.Info("Starting WebhookTrigger")
	t.callback = callback

	handler := &WebhookHandler{
		TriggerID: t.ID,
		Callback:  callback,
		Logger:    t.logger,
	}

	if err := manager.RegisterWebhook(t.Path, handler); err != nil {
		return err
	}

	t.logger.Info("WebhookTrigger started", "path", t.Path)

	// Wait for either context cancellation or server shutdown
	select {
	case <-ctx.Done():
		t.logger.Info("WebhookTrigger context cancelled")
	case <-manager.Done():
		t.logger.Info("WebhookTrigger server stopped")
	}

	return nil
}

func (t *WebhookTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping WebhookTrigger", "id", t.ID)

	manager := GetGlobalWebhookServerManager()
	if manager != nil {
		manager.UnregisterWebhook(t.Path)
	}

	return nil
}
