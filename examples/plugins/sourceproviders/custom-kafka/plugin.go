package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/IBM/sarama"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

// TopicConfig represents configuration for a single Kafka topic.
type TopicConfig struct {
	Name     string `json:"name"      validate:"required"`
	SourceID string `json:"source_id" validate:"required"`
}

// KafkaConfig represents the full configuration for the Custom Kafka source provider.
type KafkaConfig struct {
	Brokers       []string      `json:"brokers"`
	ConsumerGroup string        `json:"consumer_group"`
	Topics        []TopicConfig `json:"topics"`
}

// CustomKafkaSourceProvider implements a Kafka consumer that transforms messages to SourceEvents.
type CustomKafkaSourceProvider struct {
	config        map[string]any
	logger        *slog.Logger
	brokers       []string
	topics        []TopicConfig
	consumerGroup string
	consumer      sarama.ConsumerGroup
	callback      protocol.SourceEventCallback
	started       bool
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// Start begins consuming from configured Kafka topics.
func (p *CustomKafkaSourceProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}

	p.callback = callback
	p.logger.Info("Starting Custom Kafka source provider",
		"brokers", p.brokers,
		"topics", len(p.topics),
		"consumer_group", p.consumerGroup)

	// Create Kafka consumer
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumerGroup(p.brokers, p.consumerGroup, config)
	if err != nil {
		return fmt.Errorf("failed to create Kafka consumer group: %w", err)
	}

	p.consumer = consumer
	p.started = true

	// Create cancellable context
	consumerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	// Extract topic names
	topicNames := make([]string, len(p.topics))
	for i, topic := range p.topics {
		topicNames[i] = topic.Name
	}

	// Start consuming in goroutine
	p.wg.Add(1)

	go p.consumeMessages(consumerCtx, topicNames)

	p.logger.Info("Custom Kafka source provider started successfully")

	return nil
}

// Stop gracefully shuts down the Kafka consumer.
func (p *CustomKafkaSourceProvider) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return nil
	}

	p.logger.Info("Stopping Custom Kafka source provider")

	if p.cancel != nil {
		p.cancel()
	}

	if p.consumer != nil {
		if err := p.consumer.Close(); err != nil {
			p.logger.Error("Error closing Kafka consumer", "error", err)
		}
	}

	p.wg.Wait()
	p.started = false
	p.logger.Info("Custom Kafka source provider stopped successfully")

	return nil
}

// Validate checks if the provider configuration is valid.
func (p *CustomKafkaSourceProvider) Validate() error {
	if len(p.brokers) == 0 {
		return errors.New("at least one Kafka broker is required")
	}

	if len(p.topics) == 0 {
		return errors.New("at least one topic configuration is required")
	}

	if p.consumerGroup == "" {
		return errors.New("consumer group is required")
	}

	return nil
}

// consumeMessages handles the actual message consumption in a goroutine.
func (p *CustomKafkaSourceProvider) consumeMessages(ctx context.Context, topics []string) {
	defer p.wg.Done()

	handler := &kafkaConsumerGroupHandler{
		provider: p,
		logger:   p.logger,
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := p.consumer.Consume(ctx, topics, handler); err != nil {
				p.logger.Error("Error from consumer", "error", err)

				return
			}
		}
	}
}

// publishKafkaMessage transforms a Kafka message to a SourceEvent and publishes it.
func (p *CustomKafkaSourceProvider) publishKafkaMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	// Find the source ID for this topic
	var sourceID string

	for _, topicConfig := range p.topics {
		if topicConfig.Name == message.Topic {
			sourceID = topicConfig.SourceID

			break
		}
	}

	if sourceID == "" {
		p.logger.Warn("No source ID configured for topic", "topic", message.Topic)

		return nil
	}

	// Transform Kafka headers to map
	headers := make(map[string]string)
	for _, header := range message.Headers {
		headers[string(header.Key)] = string(header.Value)
	}

	// Create event data with Kafka message details
	eventData := map[string]any{
		"topic":     message.Topic,
		"partition": message.Partition,
		"offset":    message.Offset,
		"timestamp": message.Timestamp,
		"headers":   headers,
	}

	// Add key if present
	if message.Key != nil {
		eventData["key"] = string(message.Key)
	}

	// Add message value (transformation function left empty as requested)
	if message.Value != nil {
		eventData["value"] = string(message.Value)
		// TODO: Add custom transformation logic here
		// This is where you would transform the message.Value to your desired format
	}

	// Publish source event
	return p.callback(ctx, sourceID, "custom-kafka", "MessageReceived", eventData)
}

// Initialize sets up the provider with required dependencies.
func (p *CustomKafkaSourceProvider) Initialize(ctx context.Context, deps protocol.Dependencies) error {
	p.logger = deps.Logger.With("module", "custom_kafka_provider")

	return nil
}

// Configure sets up the provider based on workflow definitions.
func (p *CustomKafkaSourceProvider) Configure(workflows []*models.Workflow) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("Configuring Custom Kafka provider with workflows", "workflow_count", len(workflows))
	// For this provider, configuration is handled through the factory's Create method
	// Individual workflows don't affect the Kafka consumer configuration
	return nil
}

// Prepare performs final preparation before starting.
func (p *CustomKafkaSourceProvider) Prepare(ctx context.Context) error {
	return p.Validate()
}

// kafkaConsumerGroupHandler implements sarama.ConsumerGroupHandler.
type kafkaConsumerGroupHandler struct {
	provider *CustomKafkaSourceProvider
	logger   *slog.Logger
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (h *kafkaConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Debug("Kafka consumer group session setup")

	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (h *kafkaConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Debug("Kafka consumer group session cleanup")

	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (h *kafkaConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		h.logger.Debug("Received Kafka message",
			"topic", message.Topic,
			"partition", message.Partition,
			"offset", message.Offset)

		// Publish the message as a source event
		if err := h.provider.publishKafkaMessage(session.Context(), message); err != nil {
			h.logger.Error("Failed to publish Kafka message as source event", "error", err)

			continue
		}

		// Mark message as processed
		session.MarkMessage(message, "")
	}

	return nil
}

// CustomKafkaSourceProviderFactory creates instances of CustomKafkaSourceProvider.
type CustomKafkaSourceProviderFactory struct{}

// NewCustomKafkaSourceProviderFactory creates a new factory instance.
func NewCustomKafkaSourceProviderFactory() *CustomKafkaSourceProviderFactory {
	return &CustomKafkaSourceProviderFactory{}
}

// Create instantiates a new CustomKafkaSourceProvider with configuration loaded from a JSON file.
// The file path is specified via the CUSTOM_KAFKA_CONFIG environment variable.
func (f *CustomKafkaSourceProviderFactory) Create(config map[string]any, logger *slog.Logger) (protocol.SourceProvider, error) {
	provider := &CustomKafkaSourceProvider{
		config: config,
		logger: logger.With("module", "custom_kafka"),
	}

	// Load configuration from JSON file specified in environment variable
	configFilePath := os.Getenv("CUSTOM_KAFKA_CONFIG")
	if configFilePath == "" {
		return nil, errors.New("CUSTOM_KAFKA_CONFIG environment variable is required and must point to a JSON configuration file")
	}

	// Make path absolute if it's relative
	if !filepath.IsAbs(configFilePath) {
		// Get the directory where this plugin is located
		pluginDir := filepath.Dir(os.Args[0]) // Fallback to executable directory
		if pluginPath := os.Getenv("PLUGINS_PATH"); pluginPath != "" {
			pluginDir = filepath.Join(pluginPath, "sourceproviders", "custom-kafka")
		}
		configFilePath = filepath.Join(pluginDir, configFilePath)
	}

	logger.Info("Loading Custom Kafka configuration from file", "config_file", configFilePath)

	// Read the configuration file
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file '%s': %w", configFilePath, err)
	}

	// Parse JSON configuration
	var kafkaConfig KafkaConfig
	if err := json.Unmarshal(configData, &kafkaConfig); err != nil {
		return nil, fmt.Errorf("failed to parse JSON configuration: %w", err)
	}

	// Apply configuration with defaults
	if len(kafkaConfig.Brokers) == 0 {
		provider.brokers = []string{"localhost:9092"} // Default
	} else {
		provider.brokers = kafkaConfig.Brokers
	}

	if kafkaConfig.ConsumerGroup == "" {
		provider.consumerGroup = "cg-operion-custom-kafka" // Default
	} else {
		provider.consumerGroup = kafkaConfig.ConsumerGroup
	}

	if len(kafkaConfig.Topics) == 0 {
		return nil, errors.New("at least one topic configuration is required")
	}
	provider.topics = kafkaConfig.Topics

	logger.Info("Custom Kafka configuration loaded successfully",
		"brokers", provider.brokers,
		"consumer_group", provider.consumerGroup,
		"topics_count", len(provider.topics))

	return provider, nil
}

// ID returns the unique identifier for this source provider type.
func (f *CustomKafkaSourceProviderFactory) ID() string {
	return "custom-kafka"
}

// Name returns a human-readable name for this source provider.
func (f *CustomKafkaSourceProviderFactory) Name() string {
	return "Custom Kafka"
}

// Description returns a detailed description of what this source provider does.
func (f *CustomKafkaSourceProviderFactory) Description() string {
	return "A custom Kafka source provider that consumes messages from multiple Kafka topics and transforms them into source events for workflow triggering. Supports custom message transformation and configurable topic-to-source mapping."
}

// Schema returns a JSON Schema that describes the configuration structure.
func (f *CustomKafkaSourceProviderFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"brokers": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "List of Kafka broker addresses (default: [\"localhost:9092\"])",
				"examples":    [][]string{{"localhost:9092"}, {"broker1:9092", "broker2:9092"}},
				"default":     []string{"localhost:9092"},
			},
			"consumer_group": map[string]any{
				"type":        "string",
				"description": "Kafka consumer group ID (default: \"operion-custom-kafka\")",
				"examples":    []string{"operion-custom-kafka", "my-workflows", "data-processing"},
				"default":     "operion-custom-kafka",
			},
			"topics": map[string]any{
				"type":        "array",
				"description": "List of Kafka topics to consume from with their source ID mappings",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type":        "string",
							"description": "Kafka topic name to consume from",
							"examples":    []string{"user-events", "order-updates", "notifications"},
						},
						"source_id": map[string]any{
							"type":        "string",
							"description": "Source ID to use for events from this topic",
							"examples":    []string{"user-topic-source", "order-source", "notification-source"},
						},
					},
					"required":             []string{"name", "source_id"},
					"additionalProperties": false,
				},
				"minItems": 1,
			},
		},
		"required":             []string{"topics"},
		"additionalProperties": false,
		"examples": []map[string]any{
			{
				"brokers":        []string{"localhost:9092"},
				"consumer_group": "operion-custom-kafka",
				"topics": []map[string]any{
					{
						"name":      "user-events",
						"source_id": "user-topic-source",
					},
					{
						"name":      "order-updates",
						"source_id": "order-source",
					},
				},
			},
			{
				"brokers":        []string{"broker1:9092", "broker2:9092"},
				"consumer_group": "my-workflow-group",
				"topics": []map[string]any{
					{
						"name":      "notifications",
						"source_id": "notification-source",
					},
				},
			},
		},
	}
}

// EventTypes returns a list of event types that this source provider can emit.
func (f *CustomKafkaSourceProviderFactory) EventTypes() []string {
	return []string{"MessageReceived"}
}

// Ensure interface compliance.
var _ protocol.SourceProviderFactory = (*CustomKafkaSourceProviderFactory)(nil)
var _ protocol.SourceProvider = (*CustomKafkaSourceProvider)(nil)
var _ protocol.ProviderLifecycle = (*CustomKafkaSourceProvider)(nil)

// GetSourceProvider returns the factory instance.
func GetSourceProvider() protocol.SourceProviderFactory {
	return &CustomKafkaSourceProviderFactory{}
}

// SourceProvider is the exported factory variable that the plugin system will load.
var SourceProvider = GetSourceProvider()
