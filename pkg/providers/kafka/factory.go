package kafka

import (
	"log/slog"

	"github.com/dukex/operion/pkg/protocol"
)

// KafkaProviderFactory creates instances of KafkaProvider.
type KafkaProviderFactory struct{}

// NewKafkaProviderFactory creates a new factory instance.
func NewKafkaProviderFactory() *KafkaProviderFactory {
	return &KafkaProviderFactory{}
}

// Create instantiates a new centralized KafkaProvider orchestrator.
func (f *KafkaProviderFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Provider, error) {
	// Create single orchestrator instance (configuration handled during Initialize)
	return &KafkaProvider{
		config: config,
		logger: logger.With("module", "kafka_provider"),
	}, nil
}

// ID returns the unique identifier for this source provider type.
func (f *KafkaProviderFactory) ID() string {
	return "kafka"
}

// Name returns a human-readable name for this source provider.
func (f *KafkaProviderFactory) Name() string {
	return "Kafka Provider"
}

// Description returns a detailed description of what this source provider does.
func (f *KafkaProviderFactory) Description() string {
	return "A centralized Kafka provider that manages Kafka consumer groups and converts incoming messages to source events for workflow triggering. Supports JSON schema validation, consumer group sharing, and automatic source registration from workflow triggers with connection detail persistence."
}

// Schema returns a JSON Schema that describes the provider configuration.
func (f *KafkaProviderFactory) Schema() map[string]any {
	return map[string]any{
		"type":        "object",
		"title":       "Kafka Provider Configuration",
		"description": "Configuration for the centralized Kafka provider that manages consumers and message processing",
		"properties": map[string]any{
			"connection_templates": map[string]any{
				"type":        "object",
				"description": "Reusable connection configurations for different Kafka clusters",
				"patternProperties": map[string]any{
					"^[a-zA-Z0-9_-]+$": map[string]any{
						"type":        "object",
						"description": "Named connection template",
						"properties": map[string]any{
							"brokers": map[string]any{
								"type":        "string",
								"description": "Comma-separated list of Kafka broker addresses",
								"examples":    []string{"kafka1:9092,kafka2:9092", "localhost:9092"},
							},
							"security": map[string]any{
								"type":        "object",
								"description": "Kafka security configuration (SASL, SSL)",
								"properties": map[string]any{
									"protocol": map[string]any{
										"type":        "string",
										"description": "Security protocol (PLAINTEXT, SASL_PLAINTEXT, SASL_SSL, SSL)",
										"enum":        []string{"PLAINTEXT", "SASL_PLAINTEXT", "SASL_SSL", "SSL"},
										"default":     "PLAINTEXT",
									},
									"sasl_mechanism": map[string]any{
										"type":        "string",
										"description": "SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)",
										"enum":        []string{"PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"},
									},
									"sasl_username": map[string]any{
										"type":        "string",
										"description": "SASL username",
									},
									"sasl_password": map[string]any{
										"type":        "string",
										"description": "SASL password (should use environment variables)",
									},
								},
								"additionalProperties": false,
							},
						},
						"required":             []string{"brokers"},
						"additionalProperties": false,
					},
				},
				"examples": []map[string]any{
					{
						"production": map[string]any{
							"brokers": "kafka1.prod.com:9092,kafka2.prod.com:9092",
							"security": map[string]any{
								"protocol":       "SASL_SSL",
								"sasl_mechanism": "SCRAM-SHA-256",
								"sasl_username":  "operion-user",
								"sasl_password":  "${KAFKA_PASSWORD}",
							},
						},
						"development": map[string]any{
							"brokers": "localhost:9092",
						},
					},
				},
			},
			"consumer_config": map[string]any{
				"type":        "object",
				"description": "Default Kafka consumer configuration applied to all consumers",
				"properties": map[string]any{
					"session_timeout": map[string]any{
						"type":        "string",
						"description": "Consumer session timeout duration (default: 10s)",
						"examples":    []string{"10s", "30s", "60s"},
						"default":     "10s",
					},
					"heartbeat_interval": map[string]any{
						"type":        "string",
						"description": "Consumer heartbeat interval duration (default: 3s)",
						"examples":    []string{"3s", "5s", "10s"},
						"default":     "3s",
					},
					"fetch_min": map[string]any{
						"type":        "integer",
						"description": "Minimum number of bytes to fetch in each request (default: 1)",
						"minimum":     1,
						"maximum":     1048576,
						"default":     1,
					},
					"fetch_max": map[string]any{
						"type":        "integer",
						"description": "Maximum number of bytes to fetch in each request (default: 1048576)",
						"minimum":     1024,
						"maximum":     52428800,
						"default":     1048576,
					},
				},
				"additionalProperties": false,
			},
			"performance": map[string]any{
				"type":        "object",
				"description": "Performance tuning configuration",
				"properties": map[string]any{
					"max_processing_time": map[string]any{
						"type":        "string",
						"description": "Maximum time to process a single message (default: 30s)",
						"examples":    []string{"30s", "1m", "5m"},
						"default":     "30s",
					},
					"consumer_buffer_size": map[string]any{
						"type":        "integer",
						"description": "Number of messages to buffer per consumer (default: 256)",
						"minimum":     1,
						"maximum":     10000,
						"default":     256,
					},
					"retry_backoff": map[string]any{
						"type":        "string",
						"description": "Backoff duration between connection retries (default: 5s)",
						"examples":    []string{"5s", "10s", "30s"},
						"default":     "5s",
					},
				},
				"additionalProperties": false,
			},
			"monitoring": map[string]any{
				"type":        "object",
				"description": "Monitoring and observability configuration",
				"properties": map[string]any{
					"metrics_enabled": map[string]any{
						"type":        "boolean",
						"description": "Enable Kafka consumer metrics collection (default: true)",
						"default":     true,
					},
					"log_level": map[string]any{
						"type":        "string",
						"description": "Log level for Kafka operations (default: info)",
						"enum":        []string{"debug", "info", "warn", "error"},
						"default":     "info",
					},
					"health_check_interval": map[string]any{
						"type":        "string",
						"description": "Interval for consumer health checks (default: 30s)",
						"examples":    []string{"30s", "1m", "5m"},
						"default":     "30s",
					},
				},
				"additionalProperties": false,
			},
		},
		"required":             []string{},
		"additionalProperties": false,
		"examples": []map[string]any{
			{
				"connection_templates": map[string]any{
					"local": map[string]any{
						"brokers": "localhost:9092",
					},
				},
			},
			{
				"connection_templates": map[string]any{
					"production": map[string]any{
						"brokers": "kafka1:9092,kafka2:9092,kafka3:9092",
						"security": map[string]any{
							"protocol":       "SASL_SSL",
							"sasl_mechanism": "SCRAM-SHA-256",
							"sasl_username":  "operion",
							"sasl_password":  "${KAFKA_PROD_PASSWORD}",
						},
					},
				},
				"consumer_config": map[string]any{
					"session_timeout":    "30s",
					"heartbeat_interval": "10s",
					"fetch_max":          2097152,
				},
				"performance": map[string]any{
					"max_processing_time":  "1m",
					"consumer_buffer_size": 512,
					"retry_backoff":        "10s",
				},
				"monitoring": map[string]any{
					"metrics_enabled":       true,
					"log_level":             "info",
					"health_check_interval": "1m",
				},
			},
		},
	}
}

// EventTypes returns a list of event types that this source provider can emit.
func (f *KafkaProviderFactory) EventTypes() []string {
	return []string{"message_received"}
}

// Ensure interface compliance.
var _ protocol.ProviderFactory = (*KafkaProviderFactory)(nil)
