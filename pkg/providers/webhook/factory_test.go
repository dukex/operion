package webhook

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookProviderFactory_Create(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	testCases := []struct {
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
			name: "with port config",
			config: map[string]any{
				"port": 9000,
			},
		},
		{
			name: "with timeout config",
			config: map[string]any{
				"timeout": map[string]any{
					"read":  "45s",
					"write": "30s",
				},
			},
		},
		{
			name: "full config",
			config: map[string]any{
				"port":             8080,
				"max_request_size": 2048000,
				"timeout": map[string]any{
					"read":  "60s",
					"write": "60s",
					"idle":  "120s",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewWebhookProviderFactory()
			require.NotNil(t, factory)

			provider, err := factory.Create(tc.config, logger)
			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify it's the right type
			webhookProvider, ok := provider.(*WebhookProvider)
			assert.True(t, ok)
			assert.NotNil(t, webhookProvider)

			// Verify configuration is stored
			if tc.config != nil {
				assert.Equal(t, tc.config, webhookProvider.config)
			}

			// Verify logger is set
			assert.NotNil(t, webhookProvider.logger)
		})
	}
}

func TestWebhookProviderFactory_ID(t *testing.T) {
	factory := NewWebhookProviderFactory()
	assert.Equal(t, "webhook", factory.ID())
}

func TestWebhookProviderFactory_Name(t *testing.T) {
	factory := NewWebhookProviderFactory()
	assert.Equal(t, "Centralized Webhook", factory.Name())
}

func TestWebhookProviderFactory_Description(t *testing.T) {
	factory := NewWebhookProviderFactory()
	description := factory.Description()

	assert.NotEmpty(t, description)
	assert.Contains(t, description, "centralized webhook")
	assert.Contains(t, description, "external ID-based security")
	assert.Contains(t, description, "HTTP POST")
	assert.Contains(t, description, "source events")
}

func TestWebhookProviderFactory_EventTypes(t *testing.T) {
	factory := NewWebhookProviderFactory()
	eventTypes := factory.EventTypes()

	require.Len(t, eventTypes, 1)
	assert.Equal(t, "WebhookReceived", eventTypes[0])
}

func TestWebhookProviderFactory_Schema(t *testing.T) {
	factory := NewWebhookProviderFactory()
	schema := factory.Schema()

	// Verify schema structure
	assert.NotNil(t, schema)
	assert.Equal(t, "object", schema["type"])

	// Verify properties exist
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, properties, "port")
	assert.Contains(t, properties, "max_request_size")
	assert.Contains(t, properties, "timeout")

	// Verify port property
	portProp, ok := properties["port"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", portProp["type"])
	assert.Equal(t, 1, portProp["minimum"])
	assert.Equal(t, 65535, portProp["maximum"])
	assert.Equal(t, 8085, portProp["default"])

	// Verify examples exist
	examples, ok := portProp["examples"].([]int)
	require.True(t, ok)
	assert.Contains(t, examples, 8080)
	assert.Contains(t, examples, 8085)

	// Verify max_request_size property
	maxSizeProp, ok := properties["max_request_size"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "integer", maxSizeProp["type"])
	assert.Equal(t, 1024, maxSizeProp["minimum"])
	assert.Equal(t, 10485760, maxSizeProp["maximum"])
	assert.Equal(t, 1048576, maxSizeProp["default"])

	// Verify timeout property structure
	timeoutProp, ok := properties["timeout"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "object", timeoutProp["type"])

	timeoutProps, ok := timeoutProp["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, timeoutProps, "read")
	assert.Contains(t, timeoutProps, "write")
	assert.Contains(t, timeoutProps, "idle")

	// Verify timeout sub-properties
	readTimeout, ok := timeoutProps["read"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", readTimeout["type"])
	assert.Equal(t, "30s", readTimeout["default"])

	// Verify required fields (should be empty)
	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Empty(t, required)

	// Verify additionalProperties is false
	assert.Equal(t, false, schema["additionalProperties"])

	// Verify description exists
	description, ok := schema["description"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, description)
	assert.Contains(t, description, "webhook")

	// Verify examples exist at top level
	schemaExamples, ok := schema["examples"].([]map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, schemaExamples)

	// Check first example
	firstExample := schemaExamples[0]
	assert.Contains(t, firstExample, "port")
	assert.Contains(t, firstExample, "max_request_size")

	// Check second example with timeout config
	if len(schemaExamples) > 1 {
		secondExample := schemaExamples[1]
		assert.Contains(t, secondExample, "port")
		assert.Contains(t, secondExample, "timeout")

		timeoutExample, ok := secondExample["timeout"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, timeoutExample, "read")
		assert.Contains(t, timeoutExample, "write")
		assert.Contains(t, timeoutExample, "idle")
	}
}

func TestWebhookProviderFactory_InterfaceCompliance(t *testing.T) {
	factory := NewWebhookProviderFactory()

	// Test that all interface methods are implemented and return expected types
	assert.IsType(t, "", factory.ID())
	assert.IsType(t, "", factory.Name())
	assert.IsType(t, "", factory.Description())
	assert.IsType(t, map[string]any{}, factory.Schema())
	assert.IsType(t, []string{}, factory.EventTypes())

	// Test Create method
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	provider, err := factory.Create(map[string]any{}, logger)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewWebhookProviderFactory(t *testing.T) {
	factory1 := NewWebhookProviderFactory()
	factory2 := NewWebhookProviderFactory()

	// Each call should return a new instance
	assert.NotSame(t, factory1, factory2)

	// But they should have the same behavior
	assert.Equal(t, factory1.ID(), factory2.ID())
	assert.Equal(t, factory1.Name(), factory2.Name())
	assert.Equal(t, factory1.Description(), factory2.Description())
	assert.Equal(t, factory1.EventTypes(), factory2.EventTypes())
}
