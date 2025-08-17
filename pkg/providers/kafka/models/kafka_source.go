package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidKafkaSource is returned when Kafka source validation fails.
var ErrInvalidKafkaSource = errors.New("invalid kafka source")

// ConnectionDetails represents Kafka connection configuration.
type ConnectionDetails struct {
	// Topic is the Kafka topic to consume from
	Topic string `json:"topic" validate:"required"`

	// ConsumerGroup is the Kafka consumer group ID
	ConsumerGroup string `json:"consumer_group"`

	// Brokers is the comma-separated list of Kafka broker addresses
	Brokers string `json:"brokers" validate:"required"`

	// Additional Kafka client configuration
	Config map[string]any `json:"config,omitempty"`
}

// KafkaSource represents a Kafka consumer configuration with external ID-based mapping.
// Each Kafka source maps connection details and optional JSON schema for message validation.
type KafkaSource struct {
	// ID is the internal source identifier used in workflows
	ID string `json:"id" validate:"required"`

	// ExternalID is the external UUID used for source identification
	ExternalID uuid.UUID `json:"external_id" validate:"required"`

	// ConnectionDetailsID is a hash of the connection details for sharing consumers
	ConnectionDetailsID string `json:"connection_details_id" validate:"required"`

	// ConnectionDetails contains Kafka connection configuration
	ConnectionDetails ConnectionDetails `json:"connection_details" validate:"required"`

	// JSONSchema contains optional JSON schema for message validation
	JSONSchema map[string]any `json:"json_schema,omitempty"`

	// Configuration contains source-specific settings from trigger configuration
	Configuration map[string]any `json:"configuration"`

	// CreatedAt is the timestamp when this source was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when this source was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Active indicates if this Kafka source is active and should consume messages
	Active bool `json:"active"`
}

// NewKafkaSource creates a new Kafka source with the given parameters.
// Automatically generates a random UUID for external access and sets timestamps.
func NewKafkaSource(sourceID string, configuration map[string]any) (*KafkaSource, error) {
	if sourceID == "" {
		return nil, ErrInvalidKafkaSource
	}

	if configuration == nil {
		configuration = make(map[string]any)
	}

	now := time.Now().UTC()

	// Extract connection details from configuration
	connectionDetails, err := extractConnectionDetails(configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to extract connection details: %w", err)
	}

	// Generate connection details ID for consumer sharing
	connectionDetailsID := generateConnectionDetailsID(connectionDetails, extractJSONSchema(configuration))

	source := &KafkaSource{
		ID:                  sourceID,
		ExternalID:          uuid.New(),
		ConnectionDetailsID: connectionDetailsID,
		ConnectionDetails:   connectionDetails,
		Configuration:       configuration,
		CreatedAt:           now,
		UpdatedAt:           now,
		Active:              true,
	}

	// Extract optional JSON schema from configuration
	if schema := extractJSONSchema(configuration); schema != nil {
		source.JSONSchema = schema
	}

	return source, nil
}

// extractConnectionDetails extracts and validates connection details from configuration.
func extractConnectionDetails(config map[string]any) (ConnectionDetails, error) {
	var details ConnectionDetails

	// Extract topic
	if topicVal, exists := config["topic"]; exists {
		if topic, ok := topicVal.(string); ok && topic != "" {
			details.Topic = topic
		} else {
			return details, errors.New("topic must be a non-empty string")
		}
	} else {
		return details, errors.New("topic is required in connection details")
	}

	// Extract brokers
	if brokersVal, exists := config["brokers"]; exists {
		if brokers, ok := brokersVal.(string); ok && brokers != "" {
			details.Brokers = brokers
		} else {
			return details, errors.New("brokers must be a non-empty string")
		}
	} else {
		return details, errors.New("brokers is required in connection details")
	}

	// Extract optional consumer group
	if consumerGroupVal, exists := config["consumer_group"]; exists {
		if consumerGroup, ok := consumerGroupVal.(string); ok {
			details.ConsumerGroup = consumerGroup
		}
	}

	// Extract additional config
	if configVal, exists := config["kafka_config"]; exists {
		if kafkaConfig, ok := configVal.(map[string]any); ok {
			details.Config = kafkaConfig
		}
	}

	return details, nil
}

// extractJSONSchema extracts JSON schema from configuration.
func extractJSONSchema(config map[string]any) map[string]any {
	if schemaVal, exists := config["json_schema"]; exists {
		if schema, ok := schemaVal.(map[string]any); ok {
			return schema
		}
	}
	return nil
}

// generateConnectionDetailsID generates a deterministic ID based on connection details and schema.
func generateConnectionDetailsID(details ConnectionDetails, schema map[string]any) string {
	// Create a deterministic string representation
	var parts []string
	parts = append(parts, details.Topic)
	parts = append(parts, details.Brokers)
	parts = append(parts, details.ConsumerGroup)

	// Include schema in the hash if present
	if schema != nil {
		schemaBytes, _ := json.Marshal(schema)
		parts = append(parts, string(schemaBytes))
	}

	// Include additional config if present
	if details.Config != nil {
		configBytes, _ := json.Marshal(details.Config)
		parts = append(parts, string(configBytes))
	}

	// Generate hash
	combined := strings.Join(parts, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes for shorter ID
}

// Validate performs validation on the Kafka source structure.
func (ks *KafkaSource) Validate() error {
	if ks.ID == "" {
		return ErrInvalidKafkaSource
	}

	if ks.ExternalID == uuid.Nil {
		return ErrInvalidKafkaSource
	}

	if ks.ConnectionDetailsID == "" {
		return ErrInvalidKafkaSource
	}

	// Validate connection details
	if ks.ConnectionDetails.Topic == "" {
		return errors.New("topic is required")
	}

	if ks.ConnectionDetails.Brokers == "" {
		return errors.New("brokers is required")
	}

	return nil
}

// GetConsumerGroup returns the consumer group to use, with fallback generation.
func (ks *KafkaSource) GetConsumerGroup() string {
	if ks.ConnectionDetails.ConsumerGroup != "" {
		return ks.ConnectionDetails.ConsumerGroup
	}
	// Fallback to connection details ID as specified in PRP
	return "operion-kafka-" + ks.ConnectionDetailsID
}

// HasJSONSchema returns true if this Kafka source has JSON schema validation configured.
func (ks *KafkaSource) HasJSONSchema() bool {
	return len(ks.JSONSchema) > 0
}

// UpdateConfiguration updates the Kafka source configuration and timestamp.
func (ks *KafkaSource) UpdateConfiguration(config map[string]any) error {
	// Extract new connection details
	newConnectionDetails, err := extractConnectionDetails(config)
	if err != nil {
		return err
	}

	// Update connection details and regenerate ID
	ks.ConnectionDetails = newConnectionDetails
	ks.ConnectionDetailsID = generateConnectionDetailsID(newConnectionDetails, extractJSONSchema(config))
	ks.Configuration = config
	ks.UpdatedAt = time.Now().UTC()

	// Update JSON schema if present
	if schema := extractJSONSchema(config); schema != nil {
		ks.JSONSchema = schema
	} else {
		ks.JSONSchema = nil
	}

	return nil
}

// CanShareConsumerWith checks if this source can share a consumer with another source.
func (ks *KafkaSource) CanShareConsumerWith(other *KafkaSource) bool {
	return ks.ConnectionDetailsID == other.ConnectionDetailsID
}

// MarshalJSON implements the json.Marshaler interface.
func (ks *KafkaSource) MarshalJSON() ([]byte, error) {
	type Alias KafkaSource

	return json.Marshal(&struct {
		*Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias:     (*Alias)(ks),
		CreatedAt: ks.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ks.UpdatedAt.Format(time.RFC3339),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ks *KafkaSource) UnmarshalJSON(data []byte) error {
	type Alias KafkaSource

	aux := &struct {
		*Alias
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias: (*Alias)(ks),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if aux.CreatedAt != "" {
		ks.CreatedAt, err = time.Parse(time.RFC3339, aux.CreatedAt)
		if err != nil {
			return err
		}
	}

	if aux.UpdatedAt != "" {
		ks.UpdatedAt, err = time.Parse(time.RFC3339, aux.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}
