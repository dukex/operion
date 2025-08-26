package webhook

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	webhookModels "github.com/dukex/operion/pkg/providers/webhook/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	webhookTriggerType = "webhook"
)

// MockSourceEventCallback for testing.
type MockSourceEventCallback struct {
	mock.Mock
}

func (m *MockSourceEventCallback) Call(ctx context.Context, sourceID, providerID, eventType string, eventData map[string]any) error {
	args := m.Called(ctx, sourceID, providerID, eventType, eventData)

	return args.Error(0)
}

// Test helpers

func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

func createTestWorkflow(id string, triggerNodes []*models.WorkflowNode) *models.Workflow {
	return &models.Workflow{
		ID:          id,
		Name:        "Test Workflow " + id,
		Status:      models.WorkflowStatusPublished,
		Nodes:       triggerNodes,
		Connections: []*models.Connection{},
		Variables:   map[string]any{},
		Metadata:    map[string]any{},
	}
}

func createWebhookTriggerNode(id, sourceID string, config map[string]any) *models.WorkflowNode {
	var sourceIDPtr *string
	if sourceID != "" {
		sourceIDPtr = &sourceID
	}

	providerID := webhookTriggerType
	eventType := "webhook_request"

	return &models.WorkflowNode{
		ID:         id,
		Type:       "trigger:webhook",
		Category:   models.CategoryTypeTrigger,
		Name:       "Webhook Trigger " + id,
		Config:     config,
		SourceID:   sourceIDPtr,
		ProviderID: &providerID,
		EventType:  &eventType,
		Enabled:    true,
	}
}

// Constructor and Initialization Tests

func TestWebhookProvider_Initialize(t *testing.T) {
	testCases := []struct {
		name         string
		config       map[string]any
		envVars      map[string]string
		expectedPort int
	}{
		{
			name:         "default port when no config",
			config:       map[string]any{},
			envVars:      map[string]string{},
			expectedPort: 8085,
		},
		{
			name: "port from config as int",
			config: map[string]any{
				"port": 9000,
			},
			envVars:      map[string]string{},
			expectedPort: 9000,
		},
		{
			name: "port from config as string",
			config: map[string]any{
				"port": "9001",
			},
			envVars:      map[string]string{},
			expectedPort: 9001,
		},
		{
			name:         "port from environment variable",
			config:       map[string]any{},
			envVars:      map[string]string{"WEBHOOK_PORT": "9002"},
			expectedPort: 9002,
		},
		{
			name: "config overrides environment",
			config: map[string]any{
				"port": 9003,
			},
			envVars:      map[string]string{"WEBHOOK_PORT": "9004"},
			expectedPort: 9003,
		},
		{
			name:         "invalid port in env falls back to default",
			config:       map[string]any{},
			envVars:      map[string]string{"WEBHOOK_PORT": "invalid"},
			expectedPort: 8085,
		},
		{
			name: "invalid port in config falls back to env",
			config: map[string]any{
				"port": "invalid",
			},
			envVars:      map[string]string{"WEBHOOK_PORT": "9005"},
			expectedPort: 9005,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set persistence URL for all tests
			t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_test")

			// Set environment variables
			for key, value := range tc.envVars {
				t.Setenv(key, value)
			}

			provider := &WebhookProvider{
				config: tc.config,
			}

			ctx := context.Background()
			deps := protocol.Dependencies{
				Logger: createTestLogger(),
			}

			err := provider.Initialize(ctx, deps)
			require.NoError(t, err)

			assert.Equal(t, tc.expectedPort, provider.port)
			assert.NotNil(t, provider.server)
			assert.NotNil(t, provider.webhookPersistence)
			assert.NotNil(t, provider.logger)
		})
	}
}

// Configure Tests

func TestWebhookProvider_Configure(t *testing.T) {
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
			name: "workflow without webhook triggers",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					{
						ID:         "trigger-1",
						Type:       "trigger:scheduler",
						Category:   models.CategoryTypeTrigger,
						Name:       "Non-webhook Trigger",
						Config:     map[string]any{},
						ProviderID: func(s string) *string { return &s }("scheduler"),
						SourceID:   func(s string) *string { return &s }("source-123"),
						EventType:  func(s string) *string { return &s }("schedule_due"),
						Enabled:    true,
					},
				}),
			},
			expected: 0,
		},
		{
			name: "single webhook trigger",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createWebhookTriggerNode("trigger-1", "source-123", map[string]any{}),
				}),
			},
			expected: 1,
		},
		{
			name: "multiple webhook triggers different sources",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createWebhookTriggerNode("trigger-1", "source-123", map[string]any{}),
					createWebhookTriggerNode("trigger-2", "source-456", map[string]any{}),
				}),
			},
			expected: 2,
		},
		{
			name: "multiple webhook triggers same source",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createWebhookTriggerNode("trigger-1", "source-123", map[string]any{}),
					createWebhookTriggerNode("trigger-2", "source-123", map[string]any{}),
				}),
			},
			expected: 1, // Should not create duplicate
		},
		{
			name: "inactive workflow ignored",
			workflows: []*models.Workflow{
				{
					ID:     "workflow-1",
					Status: models.WorkflowStatusDraft,
					Nodes: []*models.WorkflowNode{
						createWebhookTriggerNode("trigger-1", "source-123", map[string]any{}),
					},
				},
			},
			expected: 0,
		},
		{
			name: "webhook trigger without source_id generates UUID",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createWebhookTriggerNode("trigger-1", "", map[string]any{}), // No source ID
				}),
			},
			expected: 1, // Now generates UUID and creates source
		},
		{
			name: "webhook with JSON schema configuration",
			workflows: []*models.Workflow{
				createTestWorkflow("workflow-1", []*models.WorkflowNode{
					createWebhookTriggerNode("trigger-1", "source-schema", map[string]any{
						"json_schema": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
							},
							"required": []string{"name"},
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
			t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_configure_test")

			provider := &WebhookProvider{
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
			sources, err := provider.webhookPersistence.WebhookSources()
			require.NoError(t, err)
			assert.Len(t, sources, tt.expected)

			// Verify source properties for webhook triggers
			for _, workflow := range tt.workflows {
				if workflow.Status != models.WorkflowStatusPublished {
					continue
				}

				// Filter trigger nodes from all nodes
				for _, node := range workflow.Nodes {
					if node.Category != models.CategoryTypeTrigger {
						continue
					}

					trigger := node // rename for compatibility
					if trigger.ProviderID != nil && *trigger.ProviderID == webhookTriggerType && trigger.SourceID != nil && *trigger.SourceID != "" {
						found := false

						for _, source := range sources {
							if source.ID == *trigger.SourceID {
								found = true

								assert.Equal(t, trigger.Config, source.Configuration)
								assert.NotEmpty(t, source.ExternalID.String())
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

// Lifecycle Tests

func TestWebhookProvider_Validate(t *testing.T) {
	testCases := []struct {
		name      string
		provider  *WebhookProvider
		wantError bool
	}{
		{
			name: "valid provider",
			provider: &WebhookProvider{
				server: &WebhookServer{},
				port:   8080,
			},
			wantError: false,
		},
		{
			name: "nil server",
			provider: &WebhookProvider{
				server: nil,
				port:   8080,
			},
			wantError: true,
		},
		{
			name: "invalid port - zero",
			provider: &WebhookProvider{
				server: &WebhookServer{},
				port:   0,
			},
			wantError: true,
		},
		{
			name: "invalid port - negative",
			provider: &WebhookProvider{
				server: &WebhookServer{},
				port:   -1,
			},
			wantError: true,
		},
		{
			name: "invalid port - too high",
			provider: &WebhookProvider{
				server: &WebhookServer{},
				port:   65536,
			},
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.provider.Validate()
			if tc.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookProvider_Prepare(t *testing.T) {
	// Set up persistence for test
	t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_prepare_test")

	provider := &WebhookProvider{
		logger: createTestLogger(),
	}

	// Initialize the provider to set up persistence and server
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	// Add a test source to persistence
	source, err := webhookModels.NewWebhookSource("source-123", map[string]any{})
	require.NoError(t, err)

	err = provider.webhookPersistence.SaveWebhookSource(source)
	require.NoError(t, err)

	err = provider.Prepare(ctx)
	assert.NoError(t, err)
}

func TestWebhookProvider_Prepare_NilServer(t *testing.T) {
	provider := &WebhookProvider{
		server: nil,
	}

	ctx := context.Background()
	err := provider.Prepare(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "webhook server not initialized")
}

// Start/Stop Tests

func TestWebhookProvider_StartStop(t *testing.T) {
	// Set up persistence for test
	t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_startstop_test")

	provider := &WebhookProvider{
		config: map[string]any{"port": 0}, // Use port 0 for testing to get random available port
		logger: createTestLogger(),
	}

	// Initialize
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	// Prepare
	err = provider.Prepare(ctx)
	require.NoError(t, err)

	// Start
	mockCallback := &MockSourceEventCallback{}
	err = provider.Start(ctx, mockCallback.Call)
	require.NoError(t, err)
	assert.True(t, provider.started)

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

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

// Helper Methods Tests

func TestWebhookProvider_GetWebhookURL(t *testing.T) {
	// Set up persistence for test
	t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_url_test")

	provider := &WebhookProvider{
		logger: createTestLogger(),
	}

	// Initialize the provider to set up persistence
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	// Add test source
	source, err := webhookModels.NewWebhookSource("source-123", map[string]any{})
	require.NoError(t, err)

	err = provider.webhookPersistence.SaveWebhookSource(source)
	require.NoError(t, err)

	// Test existing source
	url := provider.GetWebhookURL("source-123")
	assert.Equal(t, "/webhook/"+source.ExternalID.String(), url)

	// Test non-existing source
	url = provider.GetWebhookURL("non-existing")
	assert.Empty(t, url)
}

func TestWebhookProvider_GetRegisteredSources(t *testing.T) {
	// Set up persistence for test
	t.Setenv("WEBHOOK_PERSISTENCE_URL", "file://"+t.TempDir()+"/webhook_sources_test")

	provider := &WebhookProvider{
		logger: createTestLogger(),
	}

	// Initialize the provider to set up persistence
	ctx := context.Background()
	deps := protocol.Dependencies{Logger: createTestLogger()}
	err := provider.Initialize(ctx, deps)
	require.NoError(t, err)

	// Add test sources
	source1, err := webhookModels.NewWebhookSource("source-1", map[string]any{})
	require.NoError(t, err)
	source2, err := webhookModels.NewWebhookSource("source-2", map[string]any{})
	require.NoError(t, err)

	err = provider.webhookPersistence.SaveWebhookSource(source1)
	require.NoError(t, err)
	err = provider.webhookPersistence.SaveWebhookSource(source2)
	require.NoError(t, err)

	// Get registered sources
	sources := provider.GetRegisteredSources()

	// Should return sources from persistence
	assert.Len(t, sources, 2)
	assert.Contains(t, sources, source1.ExternalID.String())
	assert.Contains(t, sources, source2.ExternalID.String())
}

// Port Configuration Tests

func TestWebhookProvider_getWebhookPort(t *testing.T) {
	testCases := []struct {
		name         string
		config       map[string]any
		envVar       string
		expectedPort int
	}{
		{
			name:         "default port",
			config:       map[string]any{},
			envVar:       "",
			expectedPort: 8085,
		},
		{
			name:         "config port as int",
			config:       map[string]any{"port": 9000},
			envVar:       "",
			expectedPort: 9000,
		},
		{
			name:         "config port as string",
			config:       map[string]any{"port": "9001"},
			envVar:       "",
			expectedPort: 9001,
		},
		{
			name:         "env var port",
			config:       map[string]any{},
			envVar:       "9002",
			expectedPort: 9002,
		},
		{
			name:         "config overrides env",
			config:       map[string]any{"port": 9003},
			envVar:       "9004",
			expectedPort: 9003,
		},
		{
			name:         "invalid config falls back to env",
			config:       map[string]any{"port": "invalid"},
			envVar:       "9005",
			expectedPort: 9005,
		},
		{
			name:         "invalid config and env falls back to default",
			config:       map[string]any{"port": "invalid"},
			envVar:       "invalid",
			expectedPort: 8085,
		},
		{
			name:         "port out of range - too high",
			config:       map[string]any{"port": 70000},
			envVar:       "",
			expectedPort: 8085,
		},
		{
			name:         "port out of range - negative",
			config:       map[string]any{"port": -1},
			envVar:       "",
			expectedPort: 8085,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVar != "" {
				t.Setenv("WEBHOOK_PORT", tc.envVar)
			}

			provider := &WebhookProvider{
				config: tc.config,
			}

			port := provider.getWebhookPort()
			assert.Equal(t, tc.expectedPort, port)
		})
	}
}
