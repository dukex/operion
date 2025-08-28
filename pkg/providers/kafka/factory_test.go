package kafka

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dukex/operion/pkg/protocol"
)

func TestNewKafkaProviderFactory(t *testing.T) {
	factory := NewKafkaProviderFactory()
	require.NotNil(t, factory)
	assert.IsType(t, &KafkaProviderFactory{}, factory)
}

func TestKafkaProviderFactory_ID(t *testing.T) {
	factory := NewKafkaProviderFactory()
	assert.Equal(t, "kafka", factory.ID())
}

func TestKafkaProviderFactory_Name(t *testing.T) {
	factory := NewKafkaProviderFactory()
	assert.Equal(t, "Kafka Provider", factory.Name())
}

func TestKafkaProviderFactory_Description(t *testing.T) {
	factory := NewKafkaProviderFactory()
	description := factory.Description()

	assert.NotEmpty(t, description)
	assert.Contains(t, description, "Kafka")
	assert.Contains(t, description, "consumer")
	assert.Contains(t, description, "source events")
	assert.Contains(t, description, "workflow triggering")
}

func TestKafkaProviderFactory_EventTypes(t *testing.T) {
	factory := NewKafkaProviderFactory()
	eventTypes := factory.EventTypes()

	require.Len(t, eventTypes, 1)
	assert.Equal(t, "message_received", eventTypes[0])
}

func TestKafkaProviderFactory_Create(t *testing.T) {
	factory := NewKafkaProviderFactory()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name   string
		config map[string]any
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "empty config",
			config: map[string]any{},
		},
		{
			name: "config with connection templates",
			config: map[string]any{
				"connection_templates": map[string]any{
					"local": map[string]any{
						"brokers": []string{"localhost:9092"},
					},
				},
			},
		},
		{
			name: "config with consumer settings",
			config: map[string]any{
				"consumer_config": map[string]any{
					"session_timeout":    "30s",
					"heartbeat_interval": "10s",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.Create(tt.config, logger)

			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.IsType(t, &KafkaProvider{}, provider)

			// Verify provider implements required interfaces
			assert.Implements(t, (*protocol.Provider)(nil), provider)
			assert.Implements(t, (*protocol.ProviderLifecycle)(nil), provider)

			// Verify provider has config and logger
			kafkaProvider := provider.(*KafkaProvider)
			assert.Equal(t, tt.config, kafkaProvider.config)
			assert.NotNil(t, kafkaProvider.logger)
		})
	}
}

func TestKafkaProviderFactory_Schema(t *testing.T) {
	factory := NewKafkaProviderFactory()
	schema := factory.Schema()

	// Verify basic schema structure
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])
	assert.NotEmpty(t, schema["title"])
	assert.NotEmpty(t, schema["description"])

	// Verify properties exist
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, properties)

	// Test connection_templates property
	connectionTemplates, ok := properties["connection_templates"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", connectionTemplates["type"])
	assert.NotEmpty(t, connectionTemplates["description"])

	// Verify pattern properties for connection templates
	patternProps, ok := connectionTemplates["patternProperties"].(map[string]any)
	require.True(t, ok)

	templatePattern, ok := patternProps["^[a-zA-Z0-9_-]+$"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", templatePattern["type"])

	templateProps, ok := templatePattern["properties"].(map[string]any)
	require.True(t, ok)

	// Verify brokers property
	brokers, ok := templateProps["brokers"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "array", brokers["type"])
	assert.NotEmpty(t, brokers["description"])

	// Verify security property
	security, ok := templateProps["security"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", security["type"])

	securityProps, ok := security["properties"].(map[string]any)
	require.True(t, ok)

	protocol, ok := securityProps["protocol"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", protocol["type"])

	// Verify enum values for security protocol
	enum, ok := protocol["enum"].([]string)
	require.True(t, ok)
	assert.Contains(t, enum, "PLAINTEXT")
	assert.Contains(t, enum, "SASL_SSL")

	// Test consumer_config property
	consumerConfig, ok := properties["consumer_config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", consumerConfig["type"])

	consumerProps, ok := consumerConfig["properties"].(map[string]any)
	require.True(t, ok)

	// Verify session_timeout property
	sessionTimeout, ok := consumerProps["session_timeout"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", sessionTimeout["type"])
	assert.Equal(t, "10s", sessionTimeout["default"])

	// Test performance property
	performance, ok := properties["performance"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", performance["type"])

	perfProps, ok := performance["properties"].(map[string]any)
	require.True(t, ok)

	// Verify max_processing_time property
	maxProcessingTime, ok := perfProps["max_processing_time"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", maxProcessingTime["type"])
	assert.Equal(t, "30s", maxProcessingTime["default"])

	// Verify consumer_buffer_size property
	consumerBufferSize, ok := perfProps["consumer_buffer_size"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", consumerBufferSize["type"])
	assert.Equal(t, 256, consumerBufferSize["default"])

	// Test monitoring property
	monitoring, ok := properties["monitoring"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", monitoring["type"])

	monitoringProps, ok := monitoring["properties"].(map[string]any)
	require.True(t, ok)

	// Verify metrics_enabled property
	metricsEnabled, ok := monitoringProps["metrics_enabled"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "boolean", metricsEnabled["type"])
	assert.Equal(t, true, metricsEnabled["default"])

	// Verify log_level property
	logLevel, ok := monitoringProps["log_level"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", logLevel["type"])

	logLevelEnum, ok := logLevel["enum"].([]string)
	require.True(t, ok)
	assert.Contains(t, logLevelEnum, "debug")
	assert.Contains(t, logLevelEnum, "info")
	assert.Contains(t, logLevelEnum, "warn")
	assert.Contains(t, logLevelEnum, "error")

	// Verify required fields (should be empty for top level)
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required)

	// Verify additionalProperties is false
	assert.Equal(t, false, schema["additionalProperties"])
}

func TestKafkaProviderFactory_SchemaExamples(t *testing.T) {
	factory := NewKafkaProviderFactory()
	schema := factory.Schema()

	// Verify examples exist
	examples, ok := schema["examples"].([]map[string]any)
	require.True(t, ok)
	require.NotEmpty(t, examples)

	// Test first example (simple)
	simpleExample := examples[0]
	connectionTemplates, ok := simpleExample["connection_templates"].(map[string]any)
	require.True(t, ok)

	local, ok := connectionTemplates["local"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"localhost:9092"}, local["brokers"])

	// Test second example (complex) if it exists
	if len(examples) > 1 {
		complexExample := examples[1]

		// Verify connection templates
		connectionTemplates, ok := complexExample["connection_templates"].(map[string]any)
		require.True(t, ok)

		production, ok := connectionTemplates["production"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, production["brokers"])

		// Verify security configuration
		security, ok := production["security"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "SASL_SSL", security["protocol"])

		// Verify consumer config
		consumerConfig, ok := complexExample["consumer_config"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, consumerConfig)

		// Verify performance config
		performance, ok := complexExample["performance"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, performance)

		// Verify monitoring config
		monitoring, ok := complexExample["monitoring"].(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, monitoring)
	}
}

func TestKafkaProviderFactory_InterfaceCompliance(t *testing.T) {
	factory := NewKafkaProviderFactory()

	// Verify factory implements ProviderFactory interface
	assert.Implements(t, (*protocol.ProviderFactory)(nil), factory)

	// Test all interface methods
	assert.Equal(t, "kafka", factory.ID())
	assert.NotEmpty(t, factory.Name())
	assert.NotEmpty(t, factory.Description())
	assert.NotEmpty(t, factory.EventTypes())
	assert.NotNil(t, factory.Schema())

	// Test Create method
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	provider, err := factory.Create(map[string]any{}, logger)
	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestKafkaProviderFactory_SchemaValidation(t *testing.T) {
	factory := NewKafkaProviderFactory()
	schema := factory.Schema()

	// Verify schema has required JSON Schema fields
	assert.Equal(t, "object", schema["type"])
	assert.NotEmpty(t, schema["properties"])

	// Verify all property types are valid
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	for propName, propValue := range properties {
		prop, ok := propValue.(map[string]any)
		require.True(t, ok, "Property %s should be an object", propName)

		propType, ok := prop["type"]
		require.True(t, ok, "Property %s should have a type", propName)

		// Verify type is valid JSON Schema type
		validTypes := []string{"object", "string", "integer", "boolean", "array"}
		assert.Contains(t, validTypes, propType, "Property %s has invalid type", propName)

		// If it has properties, verify they're objects
		if nestedProps, hasNestedProps := prop["properties"]; hasNestedProps {
			assert.IsType(t, map[string]any{}, nestedProps,
				"Property %s properties should be an object", propName)
		}
	}
}
