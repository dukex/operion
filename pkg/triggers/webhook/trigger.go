// Package webhook provides HTTP webhook trigger implementation with centralized server management.
package webhook

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

type Trigger struct {
	Path     string
	Method   string
	Headers  map[string]string
	Enabled  bool
	callback protocol.TriggerCallback
	logger   *slog.Logger
}

func NewTrigger(ctx context.Context, config map[string]any, logger *slog.Logger) (*Trigger, error) {
	path, ok := config["path"].(string)
	if !ok {
		path = "/webhook"
	}

	method, ok := config["method"].(string)
	if !ok {
		method = "POST"
	}

	enabled := true
	enabledVal, exists := config["enabled"]

	if exists {
		if enabledBool, ok := enabledVal.(bool); ok {
			enabled = enabledBool
		}
	}

	headers := make(map[string]string)
	headersConfig, exists := config["headers"]

	if exists {
		if headersMap, ok := headersConfig.(map[string]any); ok {
			for k, v := range headersMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	trigger := &Trigger{
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

	err := trigger.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *Trigger) Validate(_ context.Context) error {
	if t.Path == "" {
		return errors.New("webhook trigger path is required")
	}

	if t.Path[0] != '/' {
		return errors.New("webhook trigger path must start with '/'")
	}

	return nil
}

func (t *Trigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	if !t.Enabled {
		t.logger.InfoContext(ctx, "WebhookTrigger is disabled.")

		return nil
	}

	manager := GetGlobalWebhookServerManager()
	if manager == nil {
		return errors.New("global webhook server manager not initialized")
	}

	t.logger.InfoContext(ctx, "Starting WebhookTrigger")
	t.callback = callback

	handler := &Handler{
		TriggerID: t.Path,
		Callback:  callback,
		Logger:    t.logger,
	}

	err := manager.RegisterWebhook(ctx, t.Path, handler)
	if err != nil {
		return err
	}

	t.logger.InfoContext(ctx, "WebhookTrigger started", "path", t.Path)

	select {
	case <-ctx.Done():
		t.logger.InfoContext(ctx, "WebhookTrigger context cancelled")
	case <-manager.Done():
		t.logger.InfoContext(ctx, "WebhookTrigger server stopped")
	}

	return nil
}

func (t *Trigger) Stop(ctx context.Context) error {
	t.logger.InfoContext(ctx, "Stopping WebhookTrigger", "path", t.Path)

	manager := GetGlobalWebhookServerManager()
	if manager != nil {
		manager.UnregisterWebhook(ctx, t.Path)
	}

	return nil
}
