// Package kafka provides Kafka-based receiver implementation for the receiver pattern.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/protocol"
)

const TriggerTopic = "operion.trigger"

type KafkaReceiver struct {
	sources   []protocol.SourceConfig
	eventBus  eventbus.EventBus
	logger    *slog.Logger
	consumers map[string]sarama.ConsumerGroup // key: source name
	ctx       context.Context
	cancel    context.CancelFunc
	config    protocol.ReceiverConfig
}

// NewKafkaReceiver creates a new Kafka receiver
func NewKafkaReceiver(eventBus eventbus.EventBus, logger *slog.Logger) *KafkaReceiver {
	return &KafkaReceiver{
		eventBus:  eventBus,
		logger:    logger.With("module", "kafka_receiver"),
		consumers: make(map[string]sarama.ConsumerGroup),
	}
}

func (r *KafkaReceiver) Configure(config protocol.ReceiverConfig) error {
	r.config = config

	// Filter Kafka sources
	r.sources = make([]protocol.SourceConfig, 0)
	for _, source := range config.Sources {
		if source.Type == "kafka" {
			r.sources = append(r.sources, source)
		}
	}

	return r.Validate()
}

func (r *KafkaReceiver) Validate() error {
	if len(r.sources) == 0 {
		return errors.New("no kafka sources configured")
	}

	for _, source := range r.sources {
		if source.Name == "" {
			return errors.New("kafka source name is required")
		}

		topics, ok := source.Configuration["topics"]
		if !ok {
			return fmt.Errorf("topics configuration required for kafka source %s", source.Name)
		}

		switch v := topics.(type) {
		case []interface{}:
			if len(v) == 0 {
				return fmt.Errorf("at least one topic required for kafka source %s", source.Name)
			}
		case []string:
			if len(v) == 0 {
				return fmt.Errorf("at least one topic required for kafka source %s", source.Name)
			}
		default:
			return fmt.Errorf("topics must be a list for kafka source %s", source.Name)
		}
	}

	return nil
}

func (r *KafkaReceiver) Start(ctx context.Context) error {
	r.logger.Info("Starting Kafka receiver", "sources_count", len(r.sources))
	r.ctx, r.cancel = context.WithCancel(ctx)

	for _, source := range r.sources {
		if err := r.startSource(source); err != nil {
			r.logger.Error("Failed to start Kafka source", "source", source.Name, "error", err)
			return err
		}
	}

	return nil
}

func (r *KafkaReceiver) startSource(source protocol.SourceConfig) error {
	logger := r.logger.With("source", source.Name)
	logger.Info("Starting Kafka source")

	// Get configuration
	topics := r.getTopics(source.Configuration["topics"])
	brokers := r.getBrokers(source.Configuration["brokers"])
	consumerGroup := r.getConsumerGroup(source.Configuration["consumer_group"], source.Name)

	// Configure Kafka consumer
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(brokers, consumerGroup, config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer group for source %s: %w", source.Name, err)
	}

	r.consumers[source.Name] = consumer

	// Start consuming in a goroutine
	go func() {
		defer func() {
			if err := consumer.Close(); err != nil {
				logger.Error("Error closing Kafka consumer", "error", err)
			}
		}()

		handler := &receiverConsumerHandler{
			receiver: r,
			source:   source,
			logger:   logger,
		}

		for {
			select {
			case <-r.ctx.Done():
				logger.Info("Kafka receiver context cancelled")
				return
			default:
				if err := consumer.Consume(r.ctx, topics, handler); err != nil {
					logger.Error("Kafka consumer error", "error", err)
					// Wait before retrying
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()

	// Monitor consumer errors
	go func() {
		for {
			select {
			case err := <-consumer.Errors():
				if err != nil {
					logger.Error("Kafka consumer group error", "error", err)
				}
			case <-r.ctx.Done():
				return
			}
		}
	}()

	logger.Info("Kafka source started successfully", "topics", topics, "consumer_group", consumerGroup)
	return nil
}

func (r *KafkaReceiver) Stop(ctx context.Context) error {
	r.logger.Info("Stopping Kafka receiver")

	if r.cancel != nil {
		r.cancel()
	}

	for sourceName, consumer := range r.consumers {
		if err := consumer.Close(); err != nil {
			r.logger.Error("Error closing Kafka consumer", "source", sourceName, "error", err)
		}
	}

	return nil
}

// Helper methods for configuration parsing
func (r *KafkaReceiver) getTopics(topicsConfig interface{}) []string {
	switch v := topicsConfig.(type) {
	case []interface{}:
		topics := make([]string, len(v))
		for i, topic := range v {
			topics[i] = fmt.Sprintf("%v", topic)
		}
		return topics
	case []string:
		return v
	default:
		return []string{}
	}
}

func (r *KafkaReceiver) getBrokers(brokersConfig interface{}) []string {
	if brokersConfig != nil {
		if brokersStr, ok := brokersConfig.(string); ok {
			brokers := strings.Split(brokersStr, ",")
			for i, broker := range brokers {
				brokers[i] = strings.TrimSpace(broker)
			}
			return brokers
		}
	}

	// Fall back to environment variable
	brokersStr := os.Getenv("KAFKA_BROKERS")
	if brokersStr == "" {
		brokersStr = "localhost:9092"
	}

	brokers := strings.Split(brokersStr, ",")
	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}
	return brokers
}

func (r *KafkaReceiver) getConsumerGroup(consumerGroupConfig interface{}, sourceName string) string {
	if consumerGroupConfig != nil {
		if cg, ok := consumerGroupConfig.(string); ok && cg != "" {
			return cg
		}
	}
	return fmt.Sprintf("operion-kafka-receiver-%s", sourceName)
}

// receiverConsumerHandler implements sarama.ConsumerGroupHandler
type receiverConsumerHandler struct {
	receiver *KafkaReceiver
	source   protocol.SourceConfig
	logger   *slog.Logger
}

func (h *receiverConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session started")
	return nil
}

func (h *receiverConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session ended")
	return nil
}

func (h *receiverConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		h.logger.Debug("Received Kafka message",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset,
		)

		// Parse message data (same logic as existing trigger)
		var messageData any
		var messageKey string

		if message.Key != nil {
			messageKey = string(message.Key)
		}

		// Try to parse message value as JSON
		if len(message.Value) > 0 {
			var jsonData any
			if err := json.Unmarshal(message.Value, &jsonData); err != nil {
				// If not JSON, store as raw string
				messageData = map[string]any{
					"raw_message": string(message.Value),
				}
			} else {
				messageData = jsonData
			}
		}

		// Parse headers
		headers := make(map[string]string)
		for _, header := range message.Headers {
			headers[string(header.Key)] = string(header.Value)
		}

		// Create original data (raw Kafka message format)
		originalData := map[string]any{
			"topic":     message.Topic,
			"partition": message.Partition,
			"offset":    message.Offset,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"key":       messageKey,
			"message":   messageData,
			"headers":   headers,
		}

		// Create trigger data (transformed for workflow matching)
		triggerData := map[string]any{
			"topic":   message.Topic,
			"message": messageData,
			"key":     messageKey,
		}

		// Create and publish trigger event
		triggerEvent := events.NewTriggerEvent("kafka", h.source.Name, triggerData, originalData)

		// Publish to trigger topic
		go func(event events.TriggerEvent) {
			if err := h.receiver.eventBus.Publish(context.Background(), TriggerTopic, event); err != nil {
				h.logger.Error("Failed to publish trigger event", "error", err)
			}
		}(triggerEvent)

		// Mark message as processed
		session.MarkMessage(message, "")
		
		h.logger.Debug("Published trigger event", "source", h.source.Name, "topic", message.Topic)
	}

	return nil
}