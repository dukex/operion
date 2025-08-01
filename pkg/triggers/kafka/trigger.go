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
	"github.com/dukex/operion/pkg/protocol"
)

type KafkaTrigger struct {
	Topic         string
	ConsumerGroup string
	Brokers       []string
	triggerID     string
	consumer      sarama.ConsumerGroup
	callback      protocol.TriggerCallback
	logger        *slog.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewKafkaTrigger(config map[string]any, logger *slog.Logger) (*KafkaTrigger, error) {
	topic, ok := config["topic"].(string)
	if !ok || topic == "" {
		return nil, errors.New("kafka trigger topic is required")
	}

	// Get consumer group from config or generate default
	consumerGroup, _ := config["consumer_group"].(string)
	if consumerGroup == "" {
		// This will be set when we have access to trigger ID
		consumerGroup = fmt.Sprintf("operion-triggers-%s", "default")
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

	trigger := &KafkaTrigger{
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

	if err := trigger.Validate(); err != nil {
		return nil, err
	}

	return trigger, nil
}

func (t *KafkaTrigger) Validate() error {
	if t.Topic == "" {
		return errors.New("kafka trigger topic is required")
	}
	if len(t.Brokers) == 0 {
		return errors.New("kafka trigger brokers are required")
	}
	return nil
}

func (t *KafkaTrigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	t.logger.Info("Starting Kafka trigger")
	t.callback = callback
	t.ctx, t.cancel = context.WithCancel(ctx)

	// Configure Kafka consumer
	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(t.Brokers, t.ConsumerGroup, config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	t.consumer = consumer

	// Start consuming in a goroutine
	go func() {
		defer func() {
			if err := consumer.Close(); err != nil {
				t.logger.Error("Error closing Kafka consumer", "error", err)
			}
		}()

		handler := &consumerGroupHandler{
			trigger: t,
			logger:  t.logger,
		}

		for {
			select {
			case <-t.ctx.Done():
				t.logger.Info("Kafka trigger context cancelled")
				return
			default:
				if err := consumer.Consume(t.ctx, []string{t.Topic}, handler); err != nil {
					t.logger.Error("Kafka consumer error", "error", err)
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
					t.logger.Error("Kafka consumer group error", "error", err)
				}
			case <-t.ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (t *KafkaTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping Kafka trigger")

	if t.cancel != nil {
		t.cancel()
	}

	if t.consumer != nil {
		if err := t.consumer.Close(); err != nil {
			t.logger.Error("Error closing Kafka consumer", "error", err)
			return err
		}
	}

	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	trigger *KafkaTrigger
	logger  *slog.Logger
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session started")
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session ended")
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		h.logger.Debug("Received Kafka message",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset,
		)

		// Parse message data
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
			if err := h.trigger.callback(context.Background(), data); err != nil {
				h.logger.Error("Error executing workflow for Kafka trigger", "error", err)
			}
		}(triggerData)

		// Mark message as processed
		session.MarkMessage(message, "")
	}

	return nil
}
