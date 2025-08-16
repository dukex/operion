// Package kafka provides Kafka topic-based trigger implementation.
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
	"github.com/operion-flow/interfaces"
)

type Trigger struct {
	Topic         string
	ConsumerGroup string
	Brokers       []string
	consumer      sarama.ConsumerGroup
	callback      interfaces.TriggerCallback
	logger        *slog.Logger
}

func NewTrigger(ctx context.Context, config map[string]any, logger *slog.Logger) (*Trigger, error) {
	topic, ok := config["topic"].(string)
	if !ok || topic == "" {
		return nil, errors.New("kafka trigger topic is required")
	}

	consumerGroup, _ := config["consumer_group"].(string)
	if consumerGroup == "" {
		consumerGroup = "operion-triggers-" + "default"
	}

	// Get brokers from config or environment
	brokersStr, _ := config["brokers"].(string)
	if brokersStr == "" {
		brokersStr = os.Getenv("KAFKA_BROKERS")
		if brokersStr == "" {
			brokersStr = "localhost:9092"
		}
	}

	brokers := strings.Split(brokersStr, ",")
	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}

	trigger := &Trigger{
		Topic:         topic,
		ConsumerGroup: consumerGroup,
		Brokers:       brokers,
		logger: logger.With(
			"module", "kafka_trigger",
			"topic", topic,
			"consumer_group", consumerGroup,
			"brokers", brokers,
		),
	}

	err := trigger.Validate(ctx)
	if err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *Trigger) Validate(_ context.Context) error {
	if t.Topic == "" {
		return errors.New("kafka trigger topic is required")
	}

	if len(t.Brokers) == 0 {
		return errors.New("kafka trigger brokers are required")
	}

	return nil
}

const kafkaSessionTimeout = 10 * time.Second
const kafkaHeartbeatInterval = 3 * time.Second
const kafkaRetryInterval = 5 * time.Second

func (t *Trigger) Start(ctx context.Context, callback interfaces.TriggerCallback) error {
	t.logger.InfoContext(ctx, "Starting Kafka trigger")
	t.callback = callback
	newCtx, cancel := context.WithCancel(ctx)

	// Configure Kafka consumer
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Session.Timeout = kafkaSessionTimeout
	config.Consumer.Group.Heartbeat.Interval = kafkaHeartbeatInterval
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(t.Brokers, t.ConsumerGroup, config)
	if err != nil {
		cancel()
		t.logger.ErrorContext(ctx, "Failed to create Kafka consumer group", "error", err)

		return fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	t.consumer = consumer

	// Start consuming
	go t.consuming(newCtx, cancel)

	// Monitor consumer errors
	go t.monitorConsumerErrors(newCtx)

	return nil
}

func (t *Trigger) Stop(ctx context.Context) error {
	t.logger.InfoContext(ctx, "Stopping Kafka trigger")

	if t.consumer != nil {
		err := t.consumer.Close()
		if err != nil {
			t.logger.ErrorContext(ctx, "Error closing Kafka consumer", "error", err)

			return err
		}
	}

	return nil
}

func (t *Trigger) consuming(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		err := t.consumer.Close()
		if err != nil {
			t.logger.ErrorContext(ctx, "Error closing Kafka consumer", "error", err)
		}

		cancel()
	}()

	handler := &consumerGroupHandler{
		trigger: t,
	}

	for {
		select {
		case <-ctx.Done():
			t.logger.InfoContext(ctx, "Kafka trigger context cancelled")
			cancel()

			return
		default:
			err := t.consumer.Consume(ctx, []string{t.Topic}, handler)
			if err != nil {
				t.logger.ErrorContext(ctx, "Kafka consumer error", "error", err)
				time.Sleep(kafkaRetryInterval)
			}
		}
	}
}

func (t *Trigger) monitorConsumerErrors(ctx context.Context) {
	for {
		select {
		case err := <-t.consumer.Errors():
			if err != nil {
				t.logger.ErrorContext(ctx, "Kafka consumer group error", "error", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

type consumerGroupHandler struct {
	trigger *Trigger
}

func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.trigger.logger.InfoContext(session.Context(), "Kafka consumer group session started")

	return nil
}

func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.trigger.logger.InfoContext(session.Context(), "Kafka consumer group session ended")

	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ctx := session.Context()

	for message := range claim.Messages() {
		h.trigger.logger.DebugContext(ctx, "Received Kafka message",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset,
		)

		// Parse message data
		var (
			messageData any
			messageKey  string
		)

		if message.Key != nil {
			messageKey = string(message.Key)
		}

		// Try to parse message value as JSON
		if len(message.Value) > 0 {
			var jsonData any

			err := json.Unmarshal(message.Value, &jsonData)
			if err != nil {
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

		// Create trigger data according to spec
		triggerData := map[string]any{
			"topic":     message.Topic,
			"partition": message.Partition,
			"offset":    message.Offset,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"key":       messageKey,
			"message":   messageData,
			"headers":   headers,
		}

		// Execute workflow callback
		go func(data map[string]any) {
			err := h.trigger.callback(ctx, data)
			if err != nil {
				h.trigger.logger.ErrorContext(ctx, "Error executing workflow for Kafka trigger", "error", err)
			}
		}(triggerData)

		// Mark message as processed
		session.MarkMessage(message, "")
	}

	return nil
}
