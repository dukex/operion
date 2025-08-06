package models

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	requiredTag = "required"
	minTag      = "min"
)

// Source Model Tests

func TestSource_Validation_ValidSource(t *testing.T) {
	source := &Source{
		ID:         "source-123",
		ProviderID: "scheduler",
		OwnerID:    "user-456",
		Configuration: map[string]any{
			"key1": "value1",
			"key2": 42,
		},
	}

	validate := validator.New()
	err := validate.Struct(source)
	assert.NoError(t, err)
}

func TestSource_Validation_MissingID(t *testing.T) {
	source := &Source{
		ID:         "", // Missing ID
		ProviderID: "scheduler",
		OwnerID:    "user-456",
		Configuration: map[string]any{
			"key1": "value1",
		},
	}

	validate := validator.New()
	err := validate.Struct(source)
	assert.Error(t, err)

	// Check that the error is for the ID field
	validationErrors := func() validator.ValidationErrors {
		var target validator.ValidationErrors

		_ = errors.As(err, &target)

		return target
	}()
	found := false

	for _, fieldErr := range validationErrors {
		if fieldErr.Field() == "ID" && fieldErr.Tag() == requiredTag {
			found = true

			break
		}
	}

	assert.True(t, found, "Should have validation error for required ID field")
}

func TestSource_Validation_MissingProviderID(t *testing.T) {
	source := &Source{
		ID:         "source-123",
		ProviderID: "", // Missing ProviderID
		OwnerID:    "user-456",
		Configuration: map[string]any{
			"key1": "value1",
		},
	}

	validate := validator.New()
	err := validate.Struct(source)
	assert.Error(t, err)

	// Check that the error is for the ProviderID field
	validationErrors := func() validator.ValidationErrors {
		var target validator.ValidationErrors

		_ = errors.As(err, &target)

		return target
	}()
	found := false

	for _, fieldErr := range validationErrors {
		if fieldErr.Field() == "ProviderID" && fieldErr.Tag() == requiredTag {
			found = true

			break
		}
	}

	assert.True(t, found, "Should have validation error for required ProviderID field")
}

func TestSource_Validation_MissingOwnerID(t *testing.T) {
	source := &Source{
		ID:         "source-123",
		ProviderID: "scheduler",
		OwnerID:    "", // Missing OwnerID
		Configuration: map[string]any{
			"key1": "value1",
		},
	}

	validate := validator.New()
	err := validate.Struct(source)
	assert.Error(t, err)

	// Check that the error is for the OwnerID field
	validationErrors := func() validator.ValidationErrors {
		var target validator.ValidationErrors

		_ = errors.As(err, &target)

		return target
	}()
	found := false

	for _, fieldErr := range validationErrors {
		if fieldErr.Field() == "OwnerID" && fieldErr.Tag() == requiredTag {
			found = true

			break
		}
	}

	assert.True(t, found, "Should have validation error for required OwnerID field")
}

func TestSource_Validation_InvalidProviderID(t *testing.T) {
	testCases := []struct {
		name       string
		providerID string
	}{
		{
			name:       "too short",
			providerID: "ab", // Less than minimum 3 characters
		},
		{
			name:       "single character",
			providerID: "a",
		},
		{
			name:       "two characters",
			providerID: "xy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &Source{
				ID:         "source-123",
				ProviderID: tc.providerID,
				OwnerID:    "user-456",
				Configuration: map[string]any{
					"key1": "value1",
				},
			}

			validate := validator.New()
			err := validate.Struct(source)
			assert.Error(t, err)

			// Check that the error is for the ProviderID field with min validation
			validationErrors := func() validator.ValidationErrors {
				var target validator.ValidationErrors

				_ = errors.As(err, &target)

				return target
			}()
			found := false

			for _, fieldErr := range validationErrors {
				if fieldErr.Field() == "ProviderID" && fieldErr.Tag() == minTag {
					found = true

					break
				}
			}

			assert.True(t, found, "Should have validation error for ProviderID min length")
		})
	}
}

func TestSource_Validation_ValidProviderID(t *testing.T) {
	testCases := []struct {
		name       string
		providerID string
	}{
		{
			name:       "exactly 3 characters",
			providerID: "abc",
		},
		{
			name:       "common provider",
			providerID: "scheduler",
		},
		{
			name:       "webhook provider",
			providerID: "webhook",
		},
		{
			name:       "custom provider",
			providerID: "my-custom-provider",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &Source{
				ID:         "source-123",
				ProviderID: tc.providerID,
				OwnerID:    "user-456",
				Configuration: map[string]any{
					"key1": "value1",
				},
			}

			validate := validator.New()
			err := validate.Struct(source)
			assert.NoError(t, err)
		})
	}
}

func TestSource_JSONSerialization(t *testing.T) {
	original := &Source{
		ID:         "source-123",
		ProviderID: "scheduler",
		OwnerID:    "user-456",
		Configuration: map[string]any{
			"cron_expression": "0 * * * *",
			"enabled":         true,
			"retry_count":     3,
			"metadata": map[string]any{
				"name": "hourly schedule",
				"tags": []string{"prod", "important"},
			},
		},
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), `"id":"source-123"`)
	assert.Contains(t, string(jsonData), `"provider_id":"scheduler"`)
	assert.Contains(t, string(jsonData), `"owner_id":"user-456"`)

	// Deserialize from JSON
	var deserialized Source

	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.ProviderID, deserialized.ProviderID)
	assert.Equal(t, original.OwnerID, deserialized.OwnerID)

	// Verify configuration deep equality
	assert.Equal(t, original.Configuration["cron_expression"], deserialized.Configuration["cron_expression"])
	assert.Equal(t, original.Configuration["enabled"], deserialized.Configuration["enabled"])
	assert.Equal(t, float64(3), deserialized.Configuration["retry_count"]) // JSON unmarshal converts numbers to float64

	// Verify nested objects
	originalMetadata := original.Configuration["metadata"].(map[string]any)
	deserializedMetadata := deserialized.Configuration["metadata"].(map[string]any)
	assert.Equal(t, originalMetadata["name"], deserializedMetadata["name"])
}

func TestSource_EmptyConfiguration(t *testing.T) {
	testCases := []struct {
		name          string
		configuration map[string]any
	}{
		{
			name:          "nil configuration",
			configuration: nil,
		},
		{
			name:          "empty configuration",
			configuration: map[string]any{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source := &Source{
				ID:            "source-123",
				ProviderID:    "scheduler",
				OwnerID:       "user-456",
				Configuration: tc.configuration,
			}

			validate := validator.New()
			err := validate.Struct(source)
			assert.NoError(t, err) // Configuration is optional

			// JSON serialization should work
			jsonData, err := json.Marshal(source)
			require.NoError(t, err)

			var deserialized Source

			err = json.Unmarshal(jsonData, &deserialized)
			require.NoError(t, err)

			// Nil configuration becomes empty map after JSON roundtrip
			if tc.configuration == nil {
				assert.Nil(t, deserialized.Configuration) // JSON preserves nil
			} else {
				assert.Equal(t, tc.configuration, deserialized.Configuration)
			}
		})
	}
}

// Provider Model Tests

func TestProvider_Validation_ValidProvider(t *testing.T) {
	provider := &Provider{
		ID:          "scheduler",
		Description: "Cron-based scheduling provider for time-based workflow triggers",
	}

	validate := validator.New()
	err := validate.Struct(provider)
	assert.NoError(t, err)
}

func TestProvider_Validation_MissingID(t *testing.T) {
	provider := &Provider{
		ID:          "", // Missing ID
		Description: "Some description",
	}

	validate := validator.New()
	err := validate.Struct(provider)
	assert.Error(t, err)

	// Check that the error is for the ID field
	validationErrors := func() validator.ValidationErrors {
		var target validator.ValidationErrors

		_ = errors.As(err, &target)

		return target
	}()
	found := false

	for _, fieldErr := range validationErrors {
		if fieldErr.Field() == "ID" && fieldErr.Tag() == requiredTag {
			found = true

			break
		}
	}

	assert.True(t, found, "Should have validation error for required ID field")
}

func TestProvider_Validation_EmptyDescription(t *testing.T) {
	provider := &Provider{
		ID:          "test-provider",
		Description: "", // Empty description (should be valid since it's optional)
	}

	validate := validator.New()
	err := validate.Struct(provider)
	assert.NoError(t, err) // Description is optional
}

func TestProvider_JSONSerialization(t *testing.T) {
	testCases := []struct {
		name     string
		provider *Provider
	}{
		{
			name: "with description",
			provider: &Provider{
				ID:          "scheduler",
				Description: "Cron-based scheduling provider",
			},
		},
		{
			name: "without description",
			provider: &Provider{
				ID:          "webhook",
				Description: "",
			},
		},
		{
			name: "complex provider",
			provider: &Provider{
				ID:          "github",
				Description: "GitHub webhook integration for repository events including pushes, pull requests, and issues",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Serialize to JSON
			jsonData, err := json.Marshal(tc.provider)
			require.NoError(t, err)
			assert.Contains(t, string(jsonData), `"id":"`+tc.provider.ID+`"`)

			// Deserialize from JSON
			var deserialized Provider

			err = json.Unmarshal(jsonData, &deserialized)
			require.NoError(t, err)

			// Verify all fields match
			assert.Equal(t, tc.provider.ID, deserialized.ID)
			assert.Equal(t, tc.provider.Description, deserialized.Description)
		})
	}
}

func TestProvider_CommonProviderIDs(t *testing.T) {
	commonProviders := []struct {
		id          string
		description string
	}{
		{
			id:          "scheduler",
			description: "Time-based scheduling using cron expressions",
		},
		{
			id:          "webhook",
			description: "HTTP webhook endpoints for external system integration",
		},
		{
			id:          "kafka",
			description: "Apache Kafka message consumption",
		},
		{
			id:          "github",
			description: "GitHub repository events",
		},
		{
			id:          "gitlab",
			description: "GitLab repository and pipeline events",
		},
		{
			id:          "slack",
			description: "Slack message and event integration",
		},
	}

	for _, providerInfo := range commonProviders {
		t.Run(providerInfo.id, func(t *testing.T) {
			provider := &Provider{
				ID:          providerInfo.id,
				Description: providerInfo.description,
			}

			validate := validator.New()
			err := validate.Struct(provider)
			assert.NoError(t, err)

			// Verify serialization works
			jsonData, err := json.Marshal(provider)
			require.NoError(t, err)

			var deserialized Provider

			err = json.Unmarshal(jsonData, &deserialized)
			require.NoError(t, err)

			assert.Equal(t, provider.ID, deserialized.ID)
			assert.Equal(t, provider.Description, deserialized.Description)
		})
	}
}
