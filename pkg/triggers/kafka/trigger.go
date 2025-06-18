package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
)

type KafkaTrigger struct {
	ID            string
	Topic         string
	ConsumerGroup string
	WorkflowID    string
	Brokers       []string
	Enabled       bool
	
	// Kafka components
	consumer       sarama.ConsumerGroup
	callback       models.TriggerCallback
	ctx            context.Context
	cancel         context.CancelFunc
	logger         *log.Entry
}

func NewKafkaTrigger(config map[string]interface{}) (*KafkaTrigger, error) {
	id, _ := config["id"].(string)
	topic, _ := config["topic"].(string)
	consumerGroup, _ := config["consumer_group"].(string)
	workflowID, _ := config["workflow_id"].(string)
	
	// Get broker hosts from environment variable or config
	var brokers []string
	if brokersEnv := os.Getenv("KAFKA_BROKERS"); brokersEnv != "" {
		brokers = strings.Split(brokersEnv, ",")
	} else if brokersConfig, ok := config["brokers"].(string); ok {
		brokers = strings.Split(brokersConfig, ",")
	} else {
		brokers = []string{"localhost:9092"} // default
	}
	
	// Default consumer group if not provided
	if consumerGroup == "" {
		consumerGroup = fmt.Sprintf("operion-triggers-%s", id)
	}

	trigger := &KafkaTrigger{
		ID:            id,
		Topic:         topic,
		ConsumerGroup: consumerGroup,
		WorkflowID:    workflowID,
		Brokers:       brokers,
		Enabled:       true,
		logger: log.WithFields(log.Fields{
			"module":         "kafka_trigger",
			"id":             id,
			"topic":          topic,
			"consumer_group": consumerGroup,
			"workflow_id":    workflowID,
			"brokers":        brokers,
		}),
	}
	
	if err := trigger.Validate(); err != nil {
		return nil, err
	}
	
	return trigger, nil
}

func (t *KafkaTrigger) GetID() string   { return t.ID }
func (t *KafkaTrigger) GetType() string { return "kafka" }
func (t *KafkaTrigger) GetConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":             t.ID,
		"topic":          t.Topic,
		"consumer_group": t.ConsumerGroup,
		"workflow_id":    t.WorkflowID,
		"brokers":        strings.Join(t.Brokers, ","),
		"enabled":        t.Enabled,
	}
}

func (t *KafkaTrigger) Validate() error {
	if t.ID == "" {
		return errors.New("kafka trigger ID is required")
	}
	if t.Topic == "" {
		return errors.New("kafka trigger topic is required")
	}
	if t.WorkflowID == "" {
		return errors.New("kafka trigger workflow_id is required")
	}
	if len(t.Brokers) == 0 {
		return errors.New("kafka trigger brokers are required")
	}
	return nil
}

// GetKafkaTriggerSchema returns the JSON Schema for Kafka Trigger configuration
func GetKafkaTriggerSchema() *models.RegisteredComponent {
	return &models.RegisteredComponent{
		Type:        "kafka",
		Name:        "Kafka Topic",
		Description: "Trigger workflow when messages are received on a Kafka topic",
		Schema: &models.JSONSchema{
			Type:        "object",
			Title:       "Kafka Trigger Configuration",
			Description: "Configuration for Kafka topic-based triggering",
			Properties: map[string]*models.Property{
				"topic": {
					Type:        "string",
					Description: "Kafka topic name to subscribe to",
				},
				"consumer_group": {
					Type:        "string",
					Description: "Kafka consumer group ID (auto-generated if not provided)",
				},
				"brokers": {
					Type:        "string",
					Description: "Comma-separated list of Kafka broker addresses (uses KAFKA_BROKERS env if not provided)",
					Default:     "localhost:9092",
				},
				"workflow_id": {
					Type:        "string",
					Description: "ID of the workflow to trigger",
				},
			},
			Required: []string{"topic"},
		},
	}
}

func (t *KafkaTrigger) Start(ctx context.Context, callback models.TriggerCallback) error {
	if !t.Enabled {
		t.logger.Info("KafkaTrigger is disabled")
		return nil
	}
	
	t.logger.Info("Starting KafkaTrigger")
	t.callback = callback
	t.ctx, t.cancel = context.WithCancel(ctx)
	
	// Setup Kafka consumer
	config := sarama.NewConfig()
	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Version = sarama.V2_6_0_0
	
	consumer, err := sarama.NewConsumerGroup(t.Brokers, t.ConsumerGroup, config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}
	
	t.consumer = consumer
	
	// Start consuming in a goroutine
	go t.consume()
	
	// Start error handling goroutine
	go t.handleErrors()
	
	t.logger.Info("KafkaTrigger started successfully")
	return nil
}

func (t *KafkaTrigger) consume() {
	handler := &ConsumerGroupHandler{
		trigger: t,
		logger:  t.logger,
	}
	
	for {
		select {
		case <-t.ctx.Done():
			t.logger.Info("Kafka consumer context cancelled")
			return
		default:
			err := t.consumer.Consume(t.ctx, []string{t.Topic}, handler)
			if err != nil {
				t.logger.Errorf("Error from consumer: %v", err)
				// Wait before retrying
				time.Sleep(5 * time.Second)
			}
		}
	}
}

func (t *KafkaTrigger) handleErrors() {
	for {
		select {
		case <-t.ctx.Done():
			return
		case err := <-t.consumer.Errors():
			t.logger.Errorf("Kafka consumer error: %v", err)
		}
	}
}

func (t *KafkaTrigger) Stop(ctx context.Context) error {
	t.logger.Info("Stopping KafkaTrigger")
	
	if t.cancel != nil {
		t.cancel()
	}
	
	if t.consumer != nil {
		if err := t.consumer.Close(); err != nil {
			t.logger.Errorf("Error closing Kafka consumer: %v", err)
			return err
		}
	}
	
	t.logger.Info("KafkaTrigger stopped")
	return nil
}

// ConsumerGroupHandler handles Kafka messages
type ConsumerGroupHandler struct {
	trigger *KafkaTrigger
	logger  *log.Entry
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session started")
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Kafka consumer group session ended")
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-h.trigger.ctx.Done():
			return nil
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}
			
			h.logger.WithFields(log.Fields{
				"topic":     message.Topic,
				"partition": message.Partition,
				"offset":    message.Offset,
			}).Info("Received Kafka message")
			
			// Process the message
			if err := h.processMessage(message); err != nil {
				h.logger.Errorf("Error processing message: %v", err)
			}
			
			// Mark message as processed
			session.MarkMessage(message, "")
		}
	}
}

func (h *ConsumerGroupHandler) processMessage(message *sarama.ConsumerMessage) error {
	// Parse message data
	var messageData map[string]interface{}
	if err := json.Unmarshal(message.Value, &messageData); err != nil {
		// If JSON parsing fails, use raw message
		messageData = map[string]interface{}{
			"raw_message": string(message.Value),
		}
	}
	
	// Create trigger data
	triggerData := map[string]interface{}{
		"trigger_id":   h.trigger.ID,
		"trigger_type": "kafka",
		"topic":        message.Topic,
		"partition":    message.Partition,
		"offset":       message.Offset,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"message_data": messageData,
		"headers":      convertHeaders(message.Headers),
	}
	
	// Add message key if present
	if message.Key != nil {
		triggerData["message_key"] = string(message.Key)
	}
	
	// Call the trigger callback
	go func() {
		if err := h.trigger.callback(context.Background(), triggerData); err != nil {
			h.logger.Errorf("Error executing workflow callback: %v", err)
		}
	}()
	
	return nil
}

func convertHeaders(headers []*sarama.RecordHeader) map[string]string {
	result := make(map[string]string)
	for _, header := range headers {
		result[string(header.Key)] = string(header.Value)
	}
	return result
}
