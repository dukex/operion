package kafka

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	kafkaModels "github.com/dukex/operion/pkg/providers/kafka/models"
)

const (
	kafkaProviderType = "kafka"
)

// MockSourceEventCallback for testing.
type MockSourceEventCallback struct {
	mock.Mock
}

func (m *MockSourceEventCallback) Call(ctx context.Context, sourceID, providerID, eventType string, eventData map[string]any) error {
	args := m.Called(ctx, sourceID, providerID, eventType, eventData)

	return args.Error(0)
}

// Test helpers.
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestWorkflow(id string, triggerNodes []*models.WorkflowNode) *models.Workflow {
	return &models.Workflow{
		ID:          id,
		Name:        "Test Workflow " + id,
		Status:      models.WorkflowStatusActive,
		Nodes:       triggerNodes,
		Connections: []*models.Connection{},
		Variables:   map[string]any{},
		Metadata:    map[string]any{},
	}
}

func createKafkaTriggerNode(id, sourceID string, config map[string]any) *models.WorkflowNode {
	var sourceIDPtr *string
	if sourceID != "" {
		sourceIDPtr = &sourceID
	}

	providerID := kafkaProviderType
	eventType := "kafka_message"

	return &models.WorkflowNode{
		ID:         id,
		NodeType:   "trigger:kafka",
		Category:   models.CategoryTypeTrigger,
		Name:       "Kafka Trigger " + id,
		Config:     config,
		SourceID:   sourceIDPtr,
		ProviderID: &providerID,
		EventType:  &eventType,
		Enabled:    true,
	}
}

// Constructor and Initialization Tests

func TestKafkaProvider_Initialize(t *testing.T) {
	testCases := []struct {
		name        string
		config      map[string]any
		envVars     map[string]string
		expectError bool
	}{
		{
			name:   "valid initialization with file persistence",
			config: map[string]any{},
			envVars: map[string]string{
				"KAFKA_PERSISTENCE_URL": "file://" + t.TempDir() + "/kafka_test",
			},
			expectError: false,
		},
		{
			name:    "missing persistence URL",
			config:  map[string]any{},
			envVars: map[string]string{
				// No KAFKA_PERSISTENCE_URL
			},
			expectError: true,
		},
		{
			name:   "invalid persistence scheme",
			config: map[string]any{},
			envVars: map[string]string{
				"KAFKA_PERSISTENCE_URL": "redis://localhost:6379",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			provider := &KafkaProvider{
				config: tc.config,
			}

			ctx := context.Background()
			deps := protocol.Dependencies{
				Logger: createTestLogger(),
			}

			err := provider.Initialize(ctx, deps)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, provider.persistence)
				assert.NotNil(t, provider.logger)
				assert.NotNil(t, provider.consumers)
			}
		})
	}
}

// Configure Tests

func TestKafkaProvider_Configure(t *testing.T) {
	tests := []struct {
		name      string
		workflows []*models.Workflow
		expected  int // expected number of sources created
	}{
		{
			name:      "no workflows",
			workflows: []*models.Workflow{},
			expected:  0,
		},
		{
			name: "workflow without kafka triggers",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					{
						ID:         "trigger-1",
						NodeType:   "trigger:webhook",
						Category:   models.CategoryTypeTrigger,
						Name:       "Webhook Trigger",
						ProviderID: &[]string{"webhook"}[0],
						SourceID:   &[]string{"source-123"}[0],
						EventType:  &[]string{"WebhookReceived"}[0],
						Enabled:    true,
					},
				}),
			},
			expected: 0,
		},
		{
			name: "single kafka trigger",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createKafkaTriggerNode("trigger-1", "source-123", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
					}),
				}),
			},
			expected: 1,
		},
		{
			name: "multiple kafka triggers different sources",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createKafkaTriggerNode("trigger-1", "source-123", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
					}),
					createKafkaTriggerNode("trigger-2", "source-456", map[string]any{
						"topic":   "events",
						"brokers": "localhost:9092",
					}),
				}),
			},
			expected: 2,
		},
		{
			name: "multiple kafka triggers same source",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createKafkaTriggerNode("trigger-1", "source-123", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
					}),
					createKafkaTriggerNode("trigger-2", "source-123", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
					}),
				}),
			},
			expected: 1, // Should not create duplicate
		},
		{
			name: "inactive workflow ignored",
			workflows: []*models.Workflow{
				{
					ID:     "workflow-1",
					Status: models.WorkflowStatusInactive,
					Nodes: []*models.WorkflowNode{
						createKafkaTriggerNode("trigger-1", "source-123", map[string]any{
							"topic":   "orders",
							"brokers": "localhost:9092",
						}),
					},
					Connections: []*models.Connection{},
					Variables:   map[string]any{},
					Metadata:    map[string]any{},
				},
			},
			expected: 0,
		},
		{
			name: "kafka trigger without source_id generates UUID",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createKafkaTriggerNode("trigger-1", "", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
					}),
				}),
			},
			expected: 1, // Now generates UUID and creates source
		},
		{
			name: "kafka trigger with JSON schema configuration",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createKafkaTriggerNode("trigger-1", "source-schema", map[string]any{
						"topic":   "orders",
						"brokers": "localhost:9092",
						"json_schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"order_id": map[string]any{"type": "string"},
							},
							"required": []string{"order_id"},
						},
					}),
				}),
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up persistence for Configure test
			t.Setenv("KAFKA_PERSISTENCE_URL", "file://"+t.TempDir()+"/kafka_configure_test")

			provider := &KafkaProvider{
				config: map[string]any{},
				logger: createTestLogger(),
			}

			// Initialize the provider to set up persistence
			ctx := context.Background()
			deps := protocol.Dependencies{Logger: createTestLogger()}
			err := provider.Initialize(ctx, deps)
			require.NoError(t, err)

			_, err = provider.Configure(tt.workflows)
			require.NoError(t, err)

			// Check expected sources count from persistence
			sources, err := provider.persistence.KafkaSources()
			require.NoError(t, err)
			assert.Len(t, sources, tt.expected)

			// Verify source properties for kafka triggers
			for _, workflow := range tt.workflows {
				if workflow.Status != models.WorkflowStatusActive {
					continue
				}

				// Filter trigger nodes from all nodes
				for _, node := range workflow.Nodes {
					if node.Category != models.CategoryTypeTrigger {
						continue
					}

					trigger := node // rename for compatibility
					if trigger.ProviderID != nil && *trigger.ProviderID == kafkaProviderType && trigger.SourceID != nil && *trigger.SourceID != "" {
						found := false

						for _, source := range sources {
							if source.ID == *trigger.SourceID {
								found = true

								assert.Equal(t, trigger.Config, source.Configuration)
								assert.True(t, source.Active)

								break
							}
						}
						// If we expected sources to be created, they should be found
						if tt.expected > 0 {
							assert.True(t, found, "Source not found for source_id: %s", trigger.SourceID)
						}
					}
				}
			}
		})
	}
}

// Consumer Manager Tests

func TestKafkaProvider_UpdateConsumerManagers(t *testing.T) {
	// Set up persistence for test
	t.Setenv("KAFKA_PERSISTENCE_URL", "file://"+t.TempDir()+"/kafka_consumer_test")

	provider := &KafkaProvider{
		logger: createTestLogger(),
	}

	// Initialize the provider to set up persistence
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	// Create test workflows with sources that should share consumers
	workflows := []*models.Workflow{
		createTestWorkflow("workflow-1", []*models.WorkflowNode{
			createKafkaTriggerNode("trigger-1", "source-1", map[string]any{
				"topic":   "orders",
				"brokers": "localhost:9092",
			}),
			createKafkaTriggerNode("trigger-2", "source-2", map[string]any{
				"topic":   "orders",
				"brokers": "localhost:9092",
			}),
		}),
		createTestWorkflow("workflow-2", []*models.WorkflowNode{
			createKafkaTriggerNode("trigger-3", "source-3", map[string]any{
				"topic":   "events",
				"brokers": "localhost:9092",
			}),
		}),
	}

	_, err = provider.Configure(workflows)
	require.NoError(t, err)

	// Verify consumer managers were created
	assert.Len(t, provider.consumers, 2) // Two different connection details

	// Verify consumer manager contains correct sources
	var ordersManager, eventsManager *ConsumerManager

	for _, manager := range provider.consumers {
		switch manager.connectionDetails.Topic {
		case "orders":
			ordersManager = manager
		case "events":
			eventsManager = manager
		}
	}

	require.NotNil(t, ordersManager)
	require.NotNil(t, eventsManager)

	// Orders manager should have 2 sources (sharing same connection details)
	assert.Len(t, ordersManager.sources, 2)
	assert.Contains(t, ordersManager.sources, "source-1")
	assert.Contains(t, ordersManager.sources, "source-2")

	// Events manager should have 1 source
	assert.Len(t, eventsManager.sources, 1)
	assert.Contains(t, eventsManager.sources, "source-3")
}

// Lifecycle Tests

func TestKafkaProvider_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		provider    *KafkaProvider
		expectError bool
	}{
		{
			name: "valid provider",
			provider: &KafkaProvider{
				persistence: &mockKafkaPersistence{},
			},
			expectError: false,
		},
		{
			name: "nil persistence",
			provider: &KafkaProvider{
				persistence: nil,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.Validate()
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKafkaProvider_Prepare(t *testing.T) {
	// Set up persistence for test
	t.Setenv("KAFKA_PERSISTENCE_URL", "file://"+t.TempDir()+"/kafka_prepare_test")

	provider := &KafkaProvider{
		logger: createTestLogger(),
	}

	// Initialize the provider to set up persistence
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	err = provider.Prepare(ctx)
	assert.NoError(t, err)
}

func TestKafkaProvider_Prepare_NilPersistence(t *testing.T) {
	provider := &KafkaProvider{
		persistence: nil,
	}

	ctx := context.Background()
	err := provider.Prepare(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persistence not initialized")
}

// Start/Stop Tests (Note: These don't actually start Kafka consumers due to testing complexity)

func TestKafkaProvider_StartStopWithoutConsumers(t *testing.T) {
	// Set up persistence for test
	t.Setenv("KAFKA_PERSISTENCE_URL", "file://"+t.TempDir()+"/kafka_startstop_test")

	provider := &KafkaProvider{
		config: map[string]any{},
		logger: createTestLogger(),
	}

	// Initialize and prepare
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	err = provider.Prepare(ctx)
	require.NoError(t, err)

	// Start with no consumers should succeed
	mockCallback := &MockSourceEventCallback{}
	err = provider.Start(ctx, mockCallback.Call)
	require.NoError(t, err)
	assert.True(t, provider.started)

	// Start again should be no-op
	err = provider.Start(ctx, mockCallback.Call)
	assert.NoError(t, err)

	// Stop
	err = provider.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, provider.started)

	// Stop again should be no-op
	err = provider.Stop(ctx)
	assert.NoError(t, err)
}

// Message Processing Tests

func TestKafkaConsumerGroupHandler_ProcessMessage(t *testing.T) {
	provider := &KafkaProvider{
		logger: createTestLogger(),
	}

	manager := &ConsumerManager{
		sources: make(map[string]*kafkaModels.KafkaSource),
		logger:  createTestLogger(),
	}

	handler := &kafkaConsumerGroupHandler{
		provider: provider,
		manager:  manager,
	}

	// Create test source without JSON schema
	source, err := kafkaModels.NewKafkaSource("test-source", map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
	})
	require.NoError(t, err)

	manager.sources["test-source"] = source

	// Mock callback
	mockCallback := &MockSourceEventCallback{}
	provider.callback = mockCallback.Call

	// Create test Kafka message
	message := &sarama.ConsumerMessage{
		Topic:     "orders",
		Partition: 0,
		Offset:    123,
		Key:       []byte("order-key"),
		Value:     []byte(`{"order_id": "12345", "amount": 100}`),
		Headers:   []*sarama.RecordHeader{},
	}

	// Setup mock expectation
	mockCallback.On("Call",
		mock.Anything,
		"test-source",
		"kafka",
		"message_received",
		mock.AnythingOfType("map[string]interface {}"),
	).Return(nil)

	// Process message
	ctx := context.Background()
	err = handler.processMessage(ctx, "test-source", source, message)
	require.NoError(t, err)

	// Verify callback was called
	mockCallback.AssertExpectations(t)

	// Verify event data structure
	call := mockCallback.Calls[0]
	eventData := call.Arguments[4].(map[string]any)
	assert.Equal(t, "orders", eventData["topic"])
	assert.Equal(t, int32(0), eventData["partition"])
	assert.Equal(t, int64(123), eventData["offset"])
	assert.Equal(t, "order-key", eventData["key"])
	assert.NotEmpty(t, eventData["timestamp"])

	// Verify message was parsed as JSON
	messageData := eventData["message"].(map[string]any)
	assert.Equal(t, "12345", messageData["order_id"])
	assert.Equal(t, float64(100), messageData["amount"])
}

func TestKafkaConsumerGroupHandler_ProcessMessageWithSchema(t *testing.T) {
	provider := &KafkaProvider{
		logger: createTestLogger(),
	}

	manager := &ConsumerManager{
		sources: make(map[string]*kafkaModels.KafkaSource),
		logger:  createTestLogger(),
	}

	handler := &kafkaConsumerGroupHandler{
		provider: provider,
		manager:  manager,
	}

	// Create test source with JSON schema
	source, err := kafkaModels.NewKafkaSource("test-source-schema", map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
				"amount":   map[string]any{"type": "number"},
			},
			"required": []string{"order_id"},
		},
	})
	require.NoError(t, err)

	manager.sources["test-source-schema"] = source

	// Mock callback
	mockCallback := &MockSourceEventCallback{}
	provider.callback = mockCallback.Call

	tests := []struct {
		name           string
		messageValue   string
		expectCallback bool
	}{
		{
			name:           "valid message should trigger callback",
			messageValue:   `{"order_id": "12345", "amount": 100}`,
			expectCallback: true,
		},
		{
			name:           "invalid message should be discarded",
			messageValue:   `{"amount": 100}`, // Missing required order_id
			expectCallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockCallback.ExpectedCalls = nil
			mockCallback.Calls = nil

			if tt.expectCallback {
				mockCallback.On("Call",
					mock.Anything,
					"test-source-schema",
					"kafka",
					"message_received",
					mock.AnythingOfType("map[string]interface {}"),
				).Return(nil)
			}

			// Create test Kafka message
			message := &sarama.ConsumerMessage{
				Topic:     "orders",
				Partition: 0,
				Offset:    123,
				Key:       []byte("order-key"),
				Value:     []byte(tt.messageValue),
				Headers:   []*sarama.RecordHeader{},
			}

			// Process message
			ctx := context.Background()
			err = handler.processMessage(ctx, "test-source-schema", source, message)
			require.NoError(t, err)

			if tt.expectCallback {
				mockCallback.AssertExpectations(t)
			} else {
				mockCallback.AssertNotCalled(t, "Call")
			}
		})
	}
}

// Mock types for testing

type mockKafkaPersistence struct {
	mock.Mock
}

func (m *mockKafkaPersistence) SaveKafkaSource(source *kafkaModels.KafkaSource) error {
	args := m.Called(source)

	return args.Error(0)
}

func (m *mockKafkaPersistence) KafkaSourceByID(id string) (*kafkaModels.KafkaSource, error) {
	args := m.Called(id)

	return args.Get(0).(*kafkaModels.KafkaSource), args.Error(1)
}

func (m *mockKafkaPersistence) KafkaSourceByConnectionDetailsID(connectionDetailsID string) ([]*kafkaModels.KafkaSource, error) {
	args := m.Called(connectionDetailsID)

	return args.Get(0).([]*kafkaModels.KafkaSource), args.Error(1)
}

func (m *mockKafkaPersistence) KafkaSources() ([]*kafkaModels.KafkaSource, error) {
	args := m.Called()

	return args.Get(0).([]*kafkaModels.KafkaSource), args.Error(1)
}

func (m *mockKafkaPersistence) ActiveKafkaSources() ([]*kafkaModels.KafkaSource, error) {
	args := m.Called()

	return args.Get(0).([]*kafkaModels.KafkaSource), args.Error(1)
}

func (m *mockKafkaPersistence) DeleteKafkaSource(id string) error {
	args := m.Called(id)

	return args.Error(0)
}

func (m *mockKafkaPersistence) HealthCheck() error {
	args := m.Called()

	return args.Error(0)
}

func (m *mockKafkaPersistence) Close() error {
	args := m.Called()

	return args.Error(0)
}

// Removed mockConsumerMessage - using sarama.ConsumerMessage directly for testing
