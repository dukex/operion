package kafka

import (
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/protocol"
)

type KafkaReceiverFactory struct{}

func NewKafkaReceiverFactory() *KafkaReceiverFactory {
	return &KafkaReceiverFactory{}
}

func (f *KafkaReceiverFactory) Create(config protocol.ReceiverConfig, eventBus eventbus.EventBus, logger *slog.Logger) (protocol.Receiver, error) {
	receiver := NewKafkaReceiver(eventBus, logger)
	if err := receiver.Configure(config); err != nil {
		return nil, err
	}
	return receiver, nil
}

func (f *KafkaReceiverFactory) Type() string {
	return "kafka"
}

func (f *KafkaReceiverFactory) Name() string {
	return "Kafka Receiver"
}

func (f *KafkaReceiverFactory) Description() string {
	return "Receiver for consuming messages from Kafka topics and publishing trigger events"
}