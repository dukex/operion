package eventbus

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dukex/operion/pkg/channels/kafka"
	"github.com/dukex/operion/pkg/events"
)

const SourceEventsTopic = "operion.source-events"

// kafkaSourceEventBus implements SourceEventBus using Kafka via Watermill
type kafkaSourceEventBus struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	handlers   []SourceEventHandler
	logger     *slog.Logger
}

// NewKafkaSourceEventBus creates a new Kafka-based source event bus
func NewKafkaSourceEventBus(logger *slog.Logger) (SourceEventBus, error) {
	pub, sub, err := kafka.CreateChannel(watermill.NewSlogLogger(logger), "source-events")
	if err != nil {
		return nil, err
	}

	return &kafkaSourceEventBus{
		publisher:  pub,
		subscriber: sub,
		handlers:   make([]SourceEventHandler, 0),
		logger:     logger.With("module", "kafka-source-event-bus"),
	}, nil
}

// PublishSourceEvent publishes a source event to Kafka
func (k *kafkaSourceEventBus) PublishSourceEvent(ctx context.Context, sourceEvent *events.SourceEvent) error {
	if err := sourceEvent.Validate(); err != nil {
		return err
	}

	// Serialize the source event to JSON
	payload, err := json.Marshal(sourceEvent)
	if err != nil {
		k.logger.Error("Failed to marshal source event", "error", err, "source_id", sourceEvent.SourceID)
		return err
	}

	// Create Watermill message
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata.Set("key", sourceEvent.SourceID) // Required for Kafka partitioning
	msg.Metadata.Set("source_id", sourceEvent.SourceID)
	msg.Metadata.Set("provider_id", sourceEvent.ProviderID)
	msg.Metadata.Set("event_type", string(sourceEvent.EventType))

	k.logger.Debug("Publishing source event to Kafka",
		"source_id", sourceEvent.SourceID,
		"provider_id", sourceEvent.ProviderID,
		"event_type", sourceEvent.EventType,
		"topic", SourceEventsTopic)

	// Publish to Kafka
	err = k.publisher.Publish(SourceEventsTopic, msg)
	if err != nil {
		k.logger.Error("Failed to publish source event to Kafka", "error", err)
		return err
	}

	k.logger.Debug("Successfully published source event to Kafka")
	return nil
}

// HandleSourceEvents registers a handler for source events
func (k *kafkaSourceEventBus) HandleSourceEvents(handler SourceEventHandler) error {
	k.handlers = append(k.handlers, handler)
	k.logger.Debug("Registered source event handler", "total_handlers", len(k.handlers))
	return nil
}

// SubscribeToSourceEvents starts consuming source events from Kafka
func (k *kafkaSourceEventBus) SubscribeToSourceEvents(ctx context.Context) error {
	if len(k.handlers) == 0 {
		k.logger.Warn("No handlers registered for source events")
		return nil
	}

	k.logger.Info("Starting Kafka source event subscription", "topic", SourceEventsTopic)

	// Subscribe to the source events topic
	messages, err := k.subscriber.Subscribe(ctx, SourceEventsTopic)
	if err != nil {
		k.logger.Error("Failed to subscribe to Kafka topic", "error", err, "topic", SourceEventsTopic)
		return err
	}

	// Process messages in a goroutine
	go func() {
		for msg := range messages {
			k.logger.Debug("Received message from Kafka", "message_id", msg.UUID)

			// Deserialize the source event
			var sourceEvent events.SourceEvent
			if err := json.Unmarshal(msg.Payload, &sourceEvent); err != nil {
				k.logger.Error("Failed to unmarshal source event", "error", err, "message_id", msg.UUID)
				msg.Nack()
				continue
			}

			k.logger.Info("Processing source event from Kafka",
				"source_id", sourceEvent.SourceID,
				"provider_id", sourceEvent.ProviderID,
				"event_type", sourceEvent.EventType)

			// Call all registered handlers
			success := true
			for _, handler := range k.handlers {
				if err := handler(ctx, &sourceEvent); err != nil {
					k.logger.Error("Source event handler failed",
						"error", err,
						"source_id", sourceEvent.SourceID,
						"handler_index", len(k.handlers))
					success = false
				}
			}

			// Acknowledge or reject the message
			if success {
				msg.Ack()
				k.logger.Debug("Successfully processed source event", "source_id", sourceEvent.SourceID)
			} else {
				msg.Nack()
				k.logger.Error("Failed to process source event", "source_id", sourceEvent.SourceID)
			}
		}
	}()

	k.logger.Info("Kafka source event subscription started successfully")
	return nil
}

// Close shuts down the Kafka source event bus
func (k *kafkaSourceEventBus) Close() error {
	k.logger.Info("Closing Kafka source event bus")

	var publisherErr, subscriberErr error

	if k.publisher != nil {
		publisherErr = k.publisher.Close()
		if publisherErr != nil {
			k.logger.Error("Failed to close Kafka publisher", "error", publisherErr)
		}
	}

	if k.subscriber != nil {
		subscriberErr = k.subscriber.Close()
		if subscriberErr != nil {
			k.logger.Error("Failed to close Kafka subscriber", "error", subscriberErr)
		}
	}

	// Return the first error encountered, if any
	if publisherErr != nil {
		return publisherErr
	}
	return subscriberErr
}
