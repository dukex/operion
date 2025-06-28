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

func (f *WebhookTriggerFactory) Create(config map[string]interface{}, logger *slog.Logger) (protocol.Trigger, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}
	trigger, err := NewWebhookTrigger(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook trigger: %w", err)
	}
	return trigger, nil
}
