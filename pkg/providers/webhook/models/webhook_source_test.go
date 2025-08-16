package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constructor Tests

func TestNewWebhookSource_ValidConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		sourceID      string
		configuration map[string]any
	}{
		{
			name:     "basic webhook source",
			sourceID: "source-123",
			configuration: map[string]any{
				"timeout": 30,
			},
		},
		{
			name:     "webhook with JSON schema",
			sourceID: "source-456",
			configuration: map[string]any{
				"json_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			name:     "minimal configuration",
			sourceID: "source-789",
			configuration: map[string]any{
				"enabled": true,
			},
		},
		{
			name:          "nil configuration",
			sourceID:      "source-nil",
			configuration: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			beforeTime := time.Now().UTC()
			source, err := NewWebhookSource(tc.sourceID, tc.configuration)
			afterTime := time.Now().UTC()

			require.NoError(t, err)
			require.NotNil(t, source)

			// Verify basic fields
			assert.Equal(t, tc.sourceID, source.ID)
			assert.True(t, source.Active)

			// Verify ExternalID is valid
			assert.NotEmpty(t, source.ExternalID.String())
			_, err = uuid.Parse(source.ExternalID.String())
			assert.NoError(t, err)

			// Verify timestamps are reasonable
			assert.True(t, source.CreatedAt.After(beforeTime) || source.CreatedAt.Equal(beforeTime))
			assert.True(t, source.CreatedAt.Before(afterTime) || source.CreatedAt.Equal(afterTime))
			assert.True(t, source.UpdatedAt.After(beforeTime) || source.UpdatedAt.Equal(beforeTime))
			assert.True(t, source.UpdatedAt.Before(afterTime) || source.UpdatedAt.Equal(afterTime))

			// Verify configuration handling
			if tc.configuration == nil {
				assert.NotNil(t, source.Configuration)
				assert.Empty(t, source.Configuration)
			} else {
				assert.Equal(t, tc.configuration, source.Configuration)
			}

			// Verify JSON schema extraction
			if schema, exists := tc.configuration["json_schema"]; exists {
				assert.True(t, source.HasJSONSchema())
				assert.Equal(t, schema, source.JSONSchema)
			} else {
				assert.False(t, source.HasJSONSchema())
			}
		})
	}
}

func TestNewWebhookSource_InvalidID(t *testing.T) {
	testCases := []struct {
		name     string
		sourceID string
	}{
		{
			name:     "empty source ID",
			sourceID: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := NewWebhookSource(tc.sourceID, map[string]any{})

			assert.Error(t, err)
			assert.Equal(t, ErrInvalidWebhookSource, err)
			assert.Nil(t, source)
		})
	}
}

// Validation Tests

func TestWebhookSource_Validate_Success(t *testing.T) {
	source := &WebhookSource{
		ID:            "test-id",
		ExternalID:    uuid.New(),
		Configuration: map[string]any{},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Active:        true,
	}

	err := source.Validate()
	assert.NoError(t, err)
}

func TestWebhookSource_Validate_MissingFields(t *testing.T) {
	testCases := []struct {
		name   string
		source *WebhookSource
	}{
		{
			name: "missing ID",
			source: &WebhookSource{
				ID:            "",
				ExternalID:    uuid.New(),
				Configuration: map[string]any{},
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
				Active:        true,
			},
		},
		{
			name: "missing ExternalID",
			source: &WebhookSource{
				ID:            "test-id",
				ExternalID:    uuid.Nil,
				Configuration: map[string]any{},
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
				Active:        true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.source.Validate()
			assert.Error(t, err)
			assert.Equal(t, ErrInvalidWebhookSource, err)
		})
	}
}

func TestWebhookSource_Validate_InvalidExternalID(t *testing.T) {
	source := &WebhookSource{
		ID:            "test-id",
		ExternalID:    uuid.Nil, // Invalid ExternalID
		Configuration: map[string]any{},
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
		Active:        true,
	}

	err := source.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidWebhookSource, err)
}

// Method Tests

func TestWebhookSource_GetWebhookURL(t *testing.T) {
	testExternalID := uuid.New()
	source := &WebhookSource{
		ExternalID: testExternalID,
	}

	expectedURL := "/webhook/" + testExternalID.String()
	assert.Equal(t, expectedURL, source.GetWebhookURL())
}

func TestWebhookSource_HasJSONSchema(t *testing.T) {
	testCases := []struct {
		name       string
		jsonSchema map[string]any
		expected   bool
	}{
		{
			name:       "nil schema",
			jsonSchema: nil,
			expected:   false,
		},
		{
			name:       "empty schema",
			jsonSchema: map[string]any{},
			expected:   false,
		},
		{
			name: "valid schema",
			jsonSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &WebhookSource{
				JSONSchema: tc.jsonSchema,
			}
			assert.Equal(t, tc.expected, source.HasJSONSchema())
		})
	}
}

func TestWebhookSource_UpdateConfiguration(t *testing.T) {
	source, err := NewWebhookSource("source-123", map[string]any{"old": "value"})
	require.NoError(t, err)

	originalUpdatedAt := source.UpdatedAt

	// Wait a small amount to ensure time difference
	time.Sleep(10 * time.Millisecond)

	newConfig := map[string]any{
		"new": "configuration",
		"json_schema": map[string]any{
			"type": "object",
		},
	}

	source.UpdateConfiguration(newConfig)

	// Verify configuration was updated
	assert.Equal(t, newConfig, source.Configuration)

	// Verify timestamp was updated
	assert.True(t, source.UpdatedAt.After(originalUpdatedAt))

	// Verify JSON schema was extracted
	assert.True(t, source.HasJSONSchema())
	assert.Equal(t, map[string]any{"type": "object"}, source.JSONSchema)
}

func TestWebhookSource_UpdateConfiguration_RemoveSchema(t *testing.T) {
	initialConfig := map[string]any{
		"json_schema": map[string]any{"type": "object"},
	}
	source, err := NewWebhookSource("source-123", initialConfig)
	require.NoError(t, err)
	assert.True(t, source.HasJSONSchema())

	// Update without schema
	newConfig := map[string]any{"other": "config"}
	source.UpdateConfiguration(newConfig)

	// Verify schema was removed
	assert.False(t, source.HasJSONSchema())
	assert.Nil(t, source.JSONSchema)
}

// JSON Marshaling Tests

func TestWebhookSource_JSONMarshaling(t *testing.T) {
	testCases := []struct {
		name   string
		source *WebhookSource
	}{
		{
			name: "complete source",
			source: &WebhookSource{
				ID:         "test-id",
				ExternalID: uuid.New(),
				JSONSchema: map[string]any{
					"type": "object",
				},
				Configuration: map[string]any{
					"timeout": 30,
				},
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
				Active:    true,
			},
		},
		{
			name: "minimal source",
			source: &WebhookSource{
				ID:            "minimal-id",
				ExternalID:    uuid.New(),
				Configuration: map[string]any{},
				CreatedAt:     time.Now().UTC(),
				UpdatedAt:     time.Now().UTC(),
				Active:        false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test marshal
			data, err := json.Marshal(tc.source)
			assert.NoError(t, err)

			// Verify timestamps are in RFC3339 format
			var jsonData map[string]any

			err = json.Unmarshal(data, &jsonData)
			assert.NoError(t, err)

			createdAtStr, ok := jsonData["created_at"].(string)
			assert.True(t, ok)

			_, err = time.Parse(time.RFC3339, createdAtStr)
			assert.NoError(t, err)

			updatedAtStr, ok := jsonData["updated_at"].(string)
			assert.True(t, ok)

			_, err = time.Parse(time.RFC3339, updatedAtStr)
			assert.NoError(t, err)

			// Test unmarshal
			var unmarshaled WebhookSource

			err = json.Unmarshal(data, &unmarshaled)
			assert.NoError(t, err)

			// Verify all fields match (compare timestamps with tolerance)
			assert.Equal(t, tc.source.ID, unmarshaled.ID)
			assert.Equal(t, tc.source.ExternalID, unmarshaled.ExternalID)
			assert.Equal(t, tc.source.JSONSchema, unmarshaled.JSONSchema)
			// Note: JSON unmarshaling converts all numbers to float64, so we need to handle this
			// For configuration comparison, we'll verify the general structure rather than exact types
			if tc.source.Configuration != nil {
				assert.Equal(t, len(tc.source.Configuration), len(unmarshaled.Configuration))

				for key := range tc.source.Configuration {
					assert.Contains(t, unmarshaled.Configuration, key)
				}
			} else {
				assert.Equal(t, tc.source.Configuration, unmarshaled.Configuration)
			}

			assert.Equal(t, tc.source.Active, unmarshaled.Active)

			// Compare timestamps with 1 second tolerance
			assert.WithinDuration(t, tc.source.CreatedAt, unmarshaled.CreatedAt, time.Second)
			assert.WithinDuration(t, tc.source.UpdatedAt, unmarshaled.UpdatedAt, time.Second)
		})
	}
}

func TestWebhookSource_JSONMarshal_EmptyTimestamps(t *testing.T) {
	source := &WebhookSource{
		ID:         "test-id",
		ExternalID: uuid.New(),
		Active:     true,
		// CreatedAt and UpdatedAt are zero values
	}

	// Should marshal without error even with zero timestamps
	data, err := json.Marshal(source)
	assert.NoError(t, err)

	var unmarshaled WebhookSource

	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, source.ID, unmarshaled.ID)
	assert.Equal(t, source.ExternalID, unmarshaled.ExternalID)
	assert.Equal(t, source.Active, unmarshaled.Active)
}

// Edge Cases and Integration Tests

func TestWebhookSource_ExternalIDUniqueness(t *testing.T) {
	const numSources = 1000

	externalIDs := make(map[uuid.UUID]bool)

	for i := range numSources {
		source, err := NewWebhookSource("source-"+string(rune(i)), map[string]any{})
		require.NoError(t, err)

		// Verify ExternalID is unique
		assert.False(t, externalIDs[source.ExternalID], "Duplicate ExternalID generated: %s", source.ExternalID)
		externalIDs[source.ExternalID] = true

		// Verify ExternalID is not nil
		assert.NotEqual(t, uuid.Nil, source.ExternalID)
	}

	assert.Len(t, externalIDs, numSources)
}

func TestWebhookSource_ConfigurationEdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		configuration map[string]any
		expectSchema  bool
	}{
		{
			name: "nested configuration",
			configuration: map[string]any{
				"nested": map[string]any{
					"key": "value",
				},
				"json_schema": map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
			expectSchema: true,
		},
		{
			name: "non-object json_schema",
			configuration: map[string]any{
				"json_schema": "not an object",
			},
			expectSchema: false,
		},
		{
			name: "nil json_schema",
			configuration: map[string]any{
				"json_schema": nil,
			},
			expectSchema: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := NewWebhookSource("source-123", tc.configuration)
			require.NoError(t, err)

			assert.Equal(t, tc.expectSchema, source.HasJSONSchema())
		})
	}
}
