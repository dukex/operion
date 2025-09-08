// Package kafka provides Apache Kafka integration for event messaging.
package kafka

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

type kafkaEventBus struct {
	logger   *slog.Logger
	writer   *kafkago.Writer
	reader   *kafkago.Reader
	handlers map[events.EventType]eventbus.EventHandler
}

func NewEventBus(ctx context.Context, logger *slog.Logger) (eventbus.EventBus, error) {
	brokersStr := os.Getenv("KAFKA_BROKERS")

	splitBrokers := strings.Split(brokersStr, ",")
	if len(splitBrokers) == 0 || (len(splitBrokers) == 1 && splitBrokers[0] == "") {
		return nil, errors.New("no Kafka brokers configured")
	}

	writer := kafkago.NewWriter(kafkago.WriterConfig{
		Brokers: splitBrokers,
		Topic:   events.Topic,
	})

	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "cg-operion-event-bus"
	}

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: splitBrokers,
		Topic:   events.Topic,
		GroupID: groupID,
	})

	return &kafkaEventBus{
		logger:   logger,
		writer:   writer,
		reader:   reader,
		handlers: make(map[events.EventType]eventbus.EventHandler),
	}, nil
}

func (k *kafkaEventBus) Publish(ctx context.Context, key string, event eventbus.Event) error {
	return publishEvent(ctx, k.logger, k.writer, key, event)
}

func (k *kafkaEventBus) Subscribe(ctx context.Context) error {
	k.logger.InfoContext(ctx, "Subscribing to events")

	go consumeEvents(ctx, k.logger, k.reader, k.handlers)

	return nil
}

func (k *kafkaEventBus) Close(ctx context.Context) error {
	k.logger.InfoContext(ctx, "Closing Kafka event bus")

	if err := k.writer.Close(); err != nil {
		k.logger.ErrorContext(ctx, "Failed to close Kafka writer", "error", err)

		return err
	}

	if err := k.reader.Close(); err != nil {
		k.logger.ErrorContext(ctx, "Failed to close Kafka reader", "error", err)

		return err
	}

	return nil
}

func (k *kafkaEventBus) GenerateID(ctx context.Context) string {
	id, err := uuid.NewV7()
	if err != nil {
		k.logger.ErrorContext(ctx, "Failed to generate V7 uuid", "error", err)

		return uuid.NewString()
	}

	k.logger.DebugContext(ctx, "Generated new ID", "id", id.String())

	return id.String()
}

func (k *kafkaEventBus) Handle(ctx context.Context, eventType events.EventType, handler eventbus.EventHandler) error {
	k.logger.DebugContext(ctx, "Handling message", "message", string(eventType))

	k.handlers[eventType] = handler

	return nil
}
