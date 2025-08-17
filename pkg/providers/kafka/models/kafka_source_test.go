package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKafkaSource(t *testing.T) {
	tests := []struct {
		name          string
		sourceID      string
		configuration map[string]any
		expectError   bool
		expectedTopic string
	}{
		{
			name:     "valid basic configuration",
			sourceID: "test-source-1",
			configuration: map[string]any{
				"topic":   "orders",
				"brokers": "localhost:9092",
			},
			expectError:   false,
			expectedTopic: "orders",
		},
		{
			name:     "valid configuration with consumer group",
			sourceID: "test-source-2",
			configuration: map[string]any{
				"topic":          "events",
				"brokers":        "kafka1:9092,kafka2:9092",
				"consumer_group": "operion-events",
			},
			expectError:   false,
			expectedTopic: "events",
		},
		{
			name:     "valid configuration with JSON schema",
			sourceID: "test-source-3",
			configuration: map[string]any{
				"topic":   "notifications",
				"brokers": "localhost:9092",
				"json_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{"type": "string"},
					},
					"required": []string{"message"},
				},
			},
			expectError:   false,
			expectedTopic: "notifications",
		},
		{
			name:     "valid configuration with additional kafka config",
			sourceID: "test-source-4",
			configuration: map[string]any{
				"topic":   "logs",
				"brokers": "localhost:9092",
				"kafka_config": map[string]any{
					"security.protocol": "SASL_SSL",
					"sasl.mechanism":    "PLAIN",
				},
			},
			expectError:   false,
			expectedTopic: "logs",
		},
		{
			name:          "empty source ID",
			sourceID:      "",
			configuration: map[string]any{"topic": "test", "brokers": "localhost:9092"},
			expectError:   true,
		},
		{
			name:     "missing topic",
			sourceID: "test-source-5",
			configuration: map[string]any{
				"brokers": "localhost:9092",
			},
			expectError: true,
		},
		{
			name:     "missing brokers",
			sourceID: "test-source-6",
			configuration: map[string]any{
				"topic": "test",
			},
			expectError: true,
		},
		{
			name:     "empty topic",
			sourceID: "test-source-7",
			configuration: map[string]any{
				"topic":   "",
				"brokers": "localhost:9092",
			},
			expectError: true,
		},
		{
			name:     "empty brokers",
			sourceID: "test-source-8",
			configuration: map[string]any{
				"topic":   "test",
				"brokers": "",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewKafkaSource(tt.sourceID, tt.configuration)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, source)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, source)

			// Verify basic fields
			assert.Equal(t, tt.sourceID, source.ID)
			assert.True(t, source.Active)
			assert.NotEmpty(t, source.ConnectionDetailsID)
			assert.Equal(t, tt.expectedTopic, source.ConnectionDetails.Topic)

			// Verify timestamps
			assert.False(t, source.CreatedAt.IsZero())
			assert.False(t, source.UpdatedAt.IsZero())
			assert.True(t, source.UpdatedAt.Equal(source.CreatedAt))

			// Verify configuration is preserved
			assert.Equal(t, tt.configuration, source.Configuration)

			// Verify JSON schema extraction
			if schema, exists := tt.configuration["json_schema"]; exists {
				assert.Equal(t, schema, source.JSONSchema)
				assert.True(t, source.HasJSONSchema())
			} else {
				assert.False(t, source.HasJSONSchema())
			}
		})
	}
}

func TestKafkaSource_Validate(t *testing.T) {
	tests := []struct {
		name        string
		source      *KafkaSource
		expectError bool
	}{
		{
			name: "valid source",
			source: &KafkaSource{
				ID:                  "valid-source",
				ConnectionDetailsID: "test-connection-id",
				ConnectionDetails: ConnectionDetails{
					Topic:   "test-topic",
					Brokers: "localhost:9092",
				},
			},
			expectError: false,
		},
		{
			name: "empty ID",
			source: &KafkaSource{
				ID:                  "",
				ConnectionDetailsID: "test-connection-id",
				ConnectionDetails: ConnectionDetails{
					Topic:   "test-topic",
					Brokers: "localhost:9092",
				},
			},
			expectError: true,
		},
		{
			name: "empty connection details ID",
			source: &KafkaSource{
				ID:                  "test-source",
				ConnectionDetailsID: "",
				ConnectionDetails: ConnectionDetails{
					Topic:   "test-topic",
					Brokers: "localhost:9092",
				},
			},
			expectError: true,
		},
		{
			name: "empty topic",
			source: &KafkaSource{
				ID:                  "test-source",
				ConnectionDetailsID: "test-connection-id",
				ConnectionDetails: ConnectionDetails{
					Topic:   "",
					Brokers: "localhost:9092",
				},
			},
			expectError: true,
		},
		{
			name: "empty brokers",
			source: &KafkaSource{
				ID:                  "test-source",
				ConnectionDetailsID: "test-connection-id",
				ConnectionDetails: ConnectionDetails{
					Topic:   "test-topic",
					Brokers: "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.source.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKafkaSource_GetConsumerGroup(t *testing.T) {
	tests := []struct {
		name                string
		consumerGroup       string
		connectionDetailsID string
		expected            string
	}{
		{
			name:          "explicit consumer group",
			consumerGroup: "operion-orders",
			expected:      "operion-orders",
		},
		{
			name:                "fallback to connection details ID",
			consumerGroup:       "",
			connectionDetailsID: "abc123",
			expected:            "operion-kafka-abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := &KafkaSource{
				ConnectionDetailsID: tt.connectionDetailsID,
				ConnectionDetails: ConnectionDetails{
					ConsumerGroup: tt.consumerGroup,
				},
			}

			result := source.GetConsumerGroup()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKafkaSource_UpdateConfiguration(t *testing.T) {
	// Create initial source
	initialConfig := map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
	}
	source, err := NewKafkaSource("test-source", initialConfig)
	require.NoError(t, err)

	initialUpdatedAt := source.UpdatedAt
	initialConnectionDetailsID := source.ConnectionDetailsID

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Update configuration
	newConfig := map[string]any{
		"topic":          "events",
		"brokers":        "kafka1:9092,kafka2:9092",
		"consumer_group": "operion-events",
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"event_type": map[string]any{"type": "string"},
			},
		},
	}

	err = source.UpdateConfiguration(newConfig)
	require.NoError(t, err)

	// Verify changes
	assert.Equal(t, newConfig, source.Configuration)
	assert.Equal(t, "events", source.ConnectionDetails.Topic)
	assert.Equal(t, "kafka1:9092,kafka2:9092", source.ConnectionDetails.Brokers)
	assert.Equal(t, "operion-events", source.ConnectionDetails.ConsumerGroup)
	assert.True(t, source.HasJSONSchema())
	assert.Equal(t, newConfig["json_schema"], source.JSONSchema)
	assert.True(t, source.UpdatedAt.After(initialUpdatedAt))
	assert.NotEqual(t, initialConnectionDetailsID, source.ConnectionDetailsID) // Should change due to different config
}

func TestKafkaSource_CanShareConsumerWith(t *testing.T) {
	config1 := map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
	}
	source1, err := NewKafkaSource("source-1", config1)
	require.NoError(t, err)

	// Same configuration should share consumer
	source2, err := NewKafkaSource("source-2", config1)
	require.NoError(t, err)
	assert.True(t, source1.CanShareConsumerWith(source2))

	// Different configuration should not share consumer
	config2 := map[string]any{
		"topic":   "events",
		"brokers": "localhost:9092",
	}
	source3, err := NewKafkaSource("source-3", config2)
	require.NoError(t, err)
	assert.False(t, source1.CanShareConsumerWith(source3))
}

func TestKafkaSource_JSONMarshaling(t *testing.T) {
	config := map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
		"json_schema": map[string]any{
			"type": "object",
		},
	}

	// Create source
	source, err := NewKafkaSource("test-source", config)
	require.NoError(t, err)

	// Marshal to JSON
	jsonData, err := json.Marshal(source)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaledSource KafkaSource

	err = json.Unmarshal(jsonData, &unmarshaledSource)
	require.NoError(t, err)

	// Verify fields
	assert.Equal(t, source.ID, unmarshaledSource.ID)
	assert.Equal(t, source.ConnectionDetailsID, unmarshaledSource.ConnectionDetailsID)
	assert.Equal(t, source.ConnectionDetails, unmarshaledSource.ConnectionDetails)
	assert.Equal(t, source.JSONSchema, unmarshaledSource.JSONSchema)
	assert.Equal(t, source.Configuration, unmarshaledSource.Configuration)
	assert.Equal(t, source.Active, unmarshaledSource.Active)

	// Timestamps should be equal (within a reasonable margin due to JSON precision)
	assert.True(t, source.CreatedAt.Unix() == unmarshaledSource.CreatedAt.Unix())
	assert.True(t, source.UpdatedAt.Unix() == unmarshaledSource.UpdatedAt.Unix())
}

func TestGenerateConnectionDetailsID(t *testing.T) {
	details1 := ConnectionDetails{
		Topic:         "orders",
		Brokers:       "localhost:9092",
		ConsumerGroup: "operion-orders",
	}

	details2 := ConnectionDetails{
		Topic:         "orders",
		Brokers:       "localhost:9092",
		ConsumerGroup: "operion-orders",
	}

	details3 := ConnectionDetails{
		Topic:         "events",
		Brokers:       "localhost:9092",
		ConsumerGroup: "operion-orders",
	}

	schema := map[string]any{
		"type": "object",
	}

	// Same details should generate same ID
	id1 := generateConnectionDetailsID(details1, nil)
	id2 := generateConnectionDetailsID(details2, nil)
	assert.Equal(t, id1, id2)

	// Different details should generate different ID
	id3 := generateConnectionDetailsID(details3, nil)
	assert.NotEqual(t, id1, id3)

	// Same details with schema should generate different ID than without schema
	id4 := generateConnectionDetailsID(details1, schema)
	assert.NotEqual(t, id1, id4)

	// Verify ID format (should be hex string of reasonable length)
	assert.Len(t, id1, 32) // 16 bytes = 32 hex chars
	assert.Regexp(t, "^[0-9a-f]+$", id1)
}

func TestStructToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: "null",
		},
		{
			name: "simple struct",
			input: ConnectionDetails{
				Topic:   "orders",
				Brokers: "localhost:9092",
			},
			expected: `{"topic":"orders","brokers":"localhost:9092","consumer_group":""}`,
		},
		{
			name: "map with nested values",
			input: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "string"},
				},
			},
			expected: `{"properties":{"id":{"type":"string"}},"type":"object"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StructToJSON(tt.input)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, result)
		})
	}
}

func TestJSONToStruct(t *testing.T) {
	t.Run("unmarshal to struct", func(t *testing.T) {
		jsonStr := `{"topic":"orders","brokers":"localhost:9092","consumer_group":"test-group"}`

		var connDetails ConnectionDetails

		err := JSONToStruct(jsonStr, &connDetails)
		require.NoError(t, err)

		assert.Equal(t, "orders", connDetails.Topic)
		assert.Equal(t, "localhost:9092", connDetails.Brokers)
		assert.Equal(t, "test-group", connDetails.ConsumerGroup)
	})

	t.Run("unmarshal to map", func(t *testing.T) {
		jsonStr := `{"type":"object","properties":{"id":{"type":"string"}}}`

		var result map[string]any

		err := JSONToStruct(jsonStr, &result)
		require.NoError(t, err)

		assert.Equal(t, "object", result["type"])
		properties := result["properties"].(map[string]any)
		idField := properties["id"].(map[string]any)
		assert.Equal(t, "string", idField["type"])
	})

	t.Run("null string", func(t *testing.T) {
		var result map[string]any

		err := JSONToStruct("null", &result)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty string", func(t *testing.T) {
		var result map[string]any

		err := JSONToStruct("", &result)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var result map[string]any

		err := JSONToStruct("invalid json", &result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal JSON")
	})
}

func TestJSONSerializationRoundTrip(t *testing.T) {
	originalData := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"order_id": map[string]any{
				"type":    "string",
				"pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
			},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":    map[string]any{"type": "string"},
						"price": map[string]any{"type": "number", "minimum": 0},
					},
					"required": []string{"id", "price"},
				},
			},
		},
		"required": []string{"order_id", "items"},
	}

	// Serialize to JSON
	jsonStr, err := StructToJSON(originalData)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonStr)
	assert.NotEqual(t, "null", jsonStr)

	// Deserialize back
	var deserializedData map[string]any

	err = JSONToStruct(jsonStr, &deserializedData)
	require.NoError(t, err)

	// Verify data integrity
	assert.Equal(t, originalData["type"], deserializedData["type"])

	// Convert slice types for comparison since JSON unmarshaling creates []interface{} not []string
	origRequired := originalData["required"].([]string)
	deserRequired := deserializedData["required"].([]interface{})
	assert.Len(t, deserRequired, len(origRequired))

	for i, req := range origRequired {
		assert.Equal(t, req, deserRequired[i].(string))
	}

	// Verify nested structures
	origProperties := originalData["properties"].(map[string]any)
	deserProperties := deserializedData["properties"].(map[string]any)

	origOrderId := origProperties["order_id"].(map[string]any)
	deserOrderId := deserProperties["order_id"].(map[string]any)
	assert.Equal(t, origOrderId["pattern"], deserOrderId["pattern"])

	origItems := origProperties["items"].(map[string]any)
	deserItems := deserProperties["items"].(map[string]any)
	assert.Equal(t, origItems["type"], deserItems["type"])
}
