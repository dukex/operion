// Package webhook provides HTTP webhook trigger implementation with centralized server management.
package webhook

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

type WebhookTrigger struct {
	Path     string
	Method   string
	Headers  map[string]string
	Enabled  bool
	callback protocol.TriggerCallback
	logger   *slog.Logger
}

func NewWebhookTrigger(config map[string]any, logger *slog.Logger) (*WebhookTrigger, error) {
	path, ok := config["path"].(string)
	if !ok {
		path = "/webhook"
	}

	method, ok := config["method"].(string)
	if !ok {
		method = "POST"
	}

	enabled := true

	if enabledVal, exists := config["enabled"]; exists {
		if enabledBool, ok := enabledVal.(bool); ok {
			enabled = enabledBool
		}
	}

	headers := make(map[string]string)

	if headersConfig, exists := config["headers"]; exists {
		if headersMap, ok := headersConfig.(map[string]any); ok {
			for k, v := range headersMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	trigger := &WebhookTrigger{
		Path:    path,
		Method:  method,
		Headers: headers,
		Enabled: enabled,
		logger: logger.With(
			"module", "webhook_trigger",
			"path", path,
			"method", method,
			"enabled", enabled,
		),
	}

	err := trigger.Validate()
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *WebhookTrigger) Validate() error {
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
		TriggerID: t.Path,
		Callback:  callback,
		Logger:    t.logger,
	}

	err := manager.RegisterWebhook(t.Path, handler)
	if err != nil {
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
	t.logger.Info("Stopping WebhookTrigger", "path", t.Path)

	manager := GetGlobalWebhookServerManager()
	if manager != nil {
		manager.UnregisterWebhook(t.Path)
	}

	return nil
}
