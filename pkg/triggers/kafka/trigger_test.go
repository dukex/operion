package kafka

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewKafkaTrigger(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		envVars     map[string]string
		expectError bool
		expected    *KafkaTrigger
	}{
		{
			name: "valid configuration with topic",
			config: map[string]interface{}{
				"id":          "test-kafka-1",
				"topic":       "test-topic",
				"workflow_id": "workflow-123",
			},
			expectError: false,
			expected: &KafkaTrigger{
				ID:            "test-kafka-1",
				Topic:         "test-topic",
				WorkflowID:    "workflow-123",
				ConsumerGroup: "operion-triggers-test-kafka-1",
				Brokers:       []string{"localhost:9092"},
				Enabled:       true,
			},
		},
		{
			name: "configuration with custom consumer group",
			config: map[string]interface{}{
				"id":             "test-kafka-2",
				"topic":          "orders",
				"consumer_group": "operion-orders",
				"workflow_id":    "workflow-456",
			},
			expectError: false,
			expected: &KafkaTrigger{
				ID:            "test-kafka-2",
				Topic:         "orders",
				WorkflowID:    "workflow-456",
				ConsumerGroup: "operion-orders",
				Brokers:       []string{"localhost:9092"},
				Enabled:       true,
			},
		},
		{
			name: "configuration with custom brokers",
			config: map[string]interface{}{
				"id":          "test-kafka-3",
				"topic":       "events",
				"brokers":     "kafka1:9092,kafka2:9092",
				"workflow_id": "workflow-789",
			},
			expectError: false,
			expected: &KafkaTrigger{
				ID:            "test-kafka-3",
				Topic:         "events",
				WorkflowID:    "workflow-789",
				ConsumerGroup: "operion-triggers-test-kafka-3",
				Brokers:       []string{"kafka1:9092", "kafka2:9092"},
				Enabled:       true,
			},
		},
		{
			name: "configuration with environment variable brokers",
			config: map[string]interface{}{
				"id":          "test-kafka-4",
				"topic":       "notifications",
				"workflow_id": "workflow-111",
			},
			envVars: map[string]string{
				"KAFKA_BROKERS": "env-kafka1:9092,env-kafka2:9092",
			},
			expectError: false,
			expected: &KafkaTrigger{
				ID:            "test-kafka-4",
				Topic:         "notifications",
				WorkflowID:    "workflow-111",
				ConsumerGroup: "operion-triggers-test-kafka-4",
				Brokers:       []string{"env-kafka1:9092", "env-kafka2:9092"},
				Enabled:       true,
			},
		},
		{
			name: "missing id",
			config: map[string]interface{}{
				"topic":       "test-topic",
				"workflow_id": "workflow-123",
			},
			expectError: true,
		},
		{
			name: "missing topic",
			config: map[string]interface{}{
				"id":          "test-kafka-5",
				"workflow_id": "workflow-123",
			},
			expectError: true,
		},
		{
			name: "missing workflow_id",
			config: map[string]interface{}{
				"id":    "test-kafka-6",
				"topic": "test-topic",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables if provided
			if tt.envVars != nil {
				for key, value := range tt.envVars {
					os.Setenv(key, value)
					defer os.Unsetenv(key)
				}
			}

			trigger, err := NewKafkaTrigger(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, trigger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, trigger)
				assert.Equal(t, tt.expected.ID, trigger.ID)
				assert.Equal(t, tt.expected.Topic, trigger.Topic)
				assert.Equal(t, tt.expected.WorkflowID, trigger.WorkflowID)
				assert.Equal(t, tt.expected.ConsumerGroup, trigger.ConsumerGroup)
				assert.Equal(t, tt.expected.Brokers, trigger.Brokers)
				assert.Equal(t, tt.expected.Enabled, trigger.Enabled)
				assert.NotNil(t, trigger.logger)
			}
		})
	}
}

func TestKafkaTrigger_GetMethods(t *testing.T) {
	config := map[string]interface{}{
		"id":             "test-get-methods",
		"topic":          "test-topic",
		"consumer_group": "test-group",
		"workflow_id":    "workflow-test",
		"brokers":        "kafka1:9092,kafka2:9092",
	}

	trigger, err := NewKafkaTrigger(config)
	require.NoError(t, err)

	assert.Equal(t, "test-get-methods", trigger.GetID())
	assert.Equal(t, "kafka", trigger.GetType())

	retrievedConfig := trigger.GetConfig()
	assert.Equal(t, "test-get-methods", retrievedConfig["id"])
	assert.Equal(t, "test-topic", retrievedConfig["topic"])
	assert.Equal(t, "test-group", retrievedConfig["consumer_group"])
	assert.Equal(t, "workflow-test", retrievedConfig["workflow_id"])
	assert.Equal(t, "kafka1:9092,kafka2:9092", retrievedConfig["brokers"])
	assert.True(t, retrievedConfig["enabled"].(bool))
}

func TestKafkaTrigger_Validate(t *testing.T) {
	tests := []struct {
		name        string
		trigger     *KafkaTrigger
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid trigger",
			trigger: &KafkaTrigger{
				ID:         "valid-trigger",
				Topic:      "test-topic",
				WorkflowID: "workflow-123",
				Brokers:    []string{"localhost:9092"},
			},
			expectError: false,
		},
		{
			name: "missing ID",
			trigger: &KafkaTrigger{
				Topic:      "test-topic",
				WorkflowID: "workflow-123",
				Brokers:    []string{"localhost:9092"},
			},
			expectError: true,
			errorMsg:    "kafka trigger ID is required",
		},
		{
			name: "missing topic",
			trigger: &KafkaTrigger{
				ID:         "missing-topic",
				WorkflowID: "workflow-123",
				Brokers:    []string{"localhost:9092"},
			},
			expectError: true,
			errorMsg:    "kafka trigger topic is required",
		},
		{
			name: "missing workflow_id",
			trigger: &KafkaTrigger{
				ID:      "missing-workflow",
				Topic:   "test-topic",
				Brokers: []string{"localhost:9092"},
			},
			expectError: true,
			errorMsg:    "kafka trigger workflow_id is required",
		},
		{
			name: "missing brokers",
			trigger: &KafkaTrigger{
				ID:         "missing-brokers",
				Topic:      "test-topic",
				WorkflowID: "workflow-123",
				Brokers:    []string{},
			},
			expectError: true,
			errorMsg:    "kafka trigger brokers are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKafkaTrigger_StartStop(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-start-stop",
		"topic":       "test-topic",
		"workflow_id": "workflow-test",
		"brokers":     "non-existent-broker:9092", // This will fail connection but won't error on Start
	}

	trigger, err := NewKafkaTrigger(config)
	require.NoError(t, err)

	callCount := 0
	callback := func(ctx context.Context, data map[string]interface{}) error {
		callCount++
		// Verify the trigger data structure
		assert.Equal(t, "test-start-stop", data["trigger_id"])
		assert.Equal(t, "kafka", data["trigger_type"])
		assert.NotEmpty(t, data["timestamp"])
		return nil
	}

	// Start the trigger - this should not error even if Kafka is not available
	// because the consumer group creation might be delayed
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// We expect this to potentially fail since we're using a non-existent broker
	// But the interface should handle it gracefully
	err = trigger.Start(ctx, callback)
	// Don't assert on error here since Kafka connection might fail, which is expected

	// Stop the trigger
	err = trigger.Stop(context.Background())
	assert.NoError(t, err)
}

func TestKafkaTrigger_DisabledTrigger(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-disabled",
		"topic":       "test-topic",
		"workflow_id": "workflow-test",
	}

	trigger, err := NewKafkaTrigger(config)
	require.NoError(t, err)

	// Disable the trigger
	trigger.Enabled = false

	callCount := 0
	callback := func(ctx context.Context, data map[string]interface{}) error {
		callCount++
		return nil
	}

	// Start should not error but should not actually start consuming
	err = trigger.Start(context.Background(), callback)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop
	err = trigger.Stop(context.Background())
	require.NoError(t, err)

	// Should not have been called since it's disabled
	assert.Equal(t, 0, callCount, "Disabled trigger should not execute")
}

func TestKafkaTrigger_EnvironmentVariablePriority(t *testing.T) {
	// Test that environment variable takes precedence over config
	os.Setenv("KAFKA_BROKERS", "env-broker1:9092,env-broker2:9092")
	defer os.Unsetenv("KAFKA_BROKERS")

	config := map[string]interface{}{
		"id":          "test-env-priority",
		"topic":       "test-topic",
		"workflow_id": "workflow-test",
		"brokers":     "config-broker:9092", // This should be ignored
	}

	trigger, err := NewKafkaTrigger(config)
	require.NoError(t, err)

	// Should use environment variable, not config
	expected := []string{"env-broker1:9092", "env-broker2:9092"}
	assert.Equal(t, expected, trigger.Brokers)
}

func TestConvertHeaders(t *testing.T) {
	headers := []*sarama.RecordHeader{
		{Key: []byte("header1"), Value: []byte("value1")},
		{Key: []byte("header2"), Value: []byte("value2")},
	}

	result := convertHeaders(headers)

	expected := map[string]string{
		"header1": "value1",
		"header2": "value2",
	}

	assert.Equal(t, expected, result)
}

func TestGetKafkaTriggerSchema(t *testing.T) {
	schema := GetKafkaTriggerSchema()

	assert.Equal(t, "kafka", schema.Type)
	assert.Equal(t, "Kafka Topic", schema.Name)
	assert.Contains(t, schema.Description, "Trigger workflow when messages")

	// Check required fields
	assert.Contains(t, schema.Schema.Required, "topic")

	// Check properties
	assert.Contains(t, schema.Schema.Properties, "topic")
	assert.Contains(t, schema.Schema.Properties, "consumer_group")
	assert.Contains(t, schema.Schema.Properties, "brokers")
	assert.Contains(t, schema.Schema.Properties, "workflow_id")

	// Check topic property
	topicProp := schema.Schema.Properties["topic"]
	assert.Equal(t, "string", topicProp.Type)
	assert.Contains(t, topicProp.Description, "Kafka topic name")
}

// Mock tests would go here if we wanted to test actual Kafka message processing
// For now, we focus on testing the configuration and lifecycle management