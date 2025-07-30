package webhook

import (
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/protocol"
)

type WebhookReceiverFactory struct {
	port int
}

func NewWebhookReceiverFactory(port int) *WebhookReceiverFactory {
	return &WebhookReceiverFactory{
		port: port,
	}
}

func (f *WebhookReceiverFactory) Create(config protocol.ReceiverConfig, eventBus eventbus.EventBus, logger *slog.Logger) (protocol.Receiver, error) {
	receiver := NewWebhookReceiver(eventBus, logger, f.port)
	if err := receiver.Configure(config); err != nil {
		return nil, err
	}
	return receiver, nil
}

func (f *WebhookReceiverFactory) Type() string {
	return "webhook"
}

func (f *WebhookReceiverFactory) Name() string {
	return "Webhook Receiver"
}

func (f *WebhookReceiverFactory) Description() string {
	return "Receiver for handling HTTP webhook requests and publishing trigger events"
}