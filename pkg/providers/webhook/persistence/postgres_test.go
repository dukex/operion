//go:build integration
// +build integration

package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dukex/operion/pkg/providers/webhook/models"
)

var postgresContainer *postgres.PostgresContainer

func TestMain(m *testing.M) {
	code := m.Run()

	// Cleanup
	if postgresContainer != nil {
		_ = postgresContainer.Terminate(context.Background())
	}

	os.Exit(code)
}

// setupTestDB creates a test PostgreSQL database for testing.
func setupTestDB(t *testing.T) (*PostgresPersistence, context.Context, string) {
	ctx := context.Background()

	// Use existing container if available and running
	if postgresContainer == nil || !postgresContainer.IsRunning() {
		var err error
		postgresContainer, err = postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("operion_webhook_test"),
			postgres.WithUsername("operion"),
			postgres.WithPassword("operion"),
			postgres.BasicWaitStrategies(),
		)
		require.NoError(t, err)
	}

	databaseURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	persistence, err := NewPostgresPersistence(ctx, logger, databaseURL)
	require.NoError(t, err)

	// Clean up the table before each test
	cleanupDB(t, databaseURL)

	return persistence, ctx, databaseURL
}

func cleanupDB(t *testing.T, databaseURL string) {
	ctx := context.Background()

	db, err := sql.Open("postgres", databaseURL)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.ExecContext(ctx, "TRUNCATE TABLE webhook_sources")
	require.NoError(t, err)
}

func TestNewWebhookPostgresPersistence(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		expectError bool
	}{
		{
			name:        "valid connection",
			databaseURL: "", // Will be set by setupTestDB
			expectError: false,
		},
		{
			name:        "invalid connection string",
			databaseURL: "postgres://invalid:invalid@nonexistent:5432/nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

			if tt.databaseURL == "" {
				// Use test database
				_, _, databaseURL := setupTestDB(t)
				tt.databaseURL = databaseURL
				defer cleanupDB(t, databaseURL)
			}

			persistence, err := NewPostgresPersistence(ctx, logger, tt.databaseURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, persistence)
			} else {
				require.NoError(t, err)
				require.NotNil(t, persistence)

				// Test health check
				err = persistence.HealthCheck()
				assert.NoError(t, err)

				// Cleanup
				err = persistence.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestWebhookPersistence_SaveAndRetrieveWebhookSource(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create test webhook source with JSON schema
	config := map[string]any{
		"timeout": 30,
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"email": map[string]any{
					"type":   "string",
					"format": "email",
				},
			},
			"required": []string{"name"},
		},
	}
	source, err := models.NewWebhookSource("test-source", config)
	require.NoError(t, err)

	// Save source
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	// Retrieve by ID
	retrievedSource, err := persistence.WebhookSourceByID("test-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	// Verify source data
	assert.Equal(t, source.ID, retrievedSource.ID)
	assert.Equal(t, source.ExternalID, retrievedSource.ExternalID)
	assert.Equal(t, source.Active, retrievedSource.Active)

	// JSON schema and configuration may have type differences after JSON roundtrip
	// (e.g., []string becomes []interface{}, int becomes float64)
	assert.NotNil(t, retrievedSource.JSONSchema)
	assert.NotNil(t, retrievedSource.Configuration)

	// Verify specific values that should be preserved
	assert.Equal(t, "object", retrievedSource.JSONSchema["type"])
	assert.Equal(t, float64(30), retrievedSource.Configuration["timeout"]) // JSON unmarshals numbers as float64
	assert.True(t, !retrievedSource.CreatedAt.IsZero())
	assert.True(t, !retrievedSource.UpdatedAt.IsZero())

	// Test non-existent source
	nonExistent, err := persistence.WebhookSourceByID("non-existent")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestWebhookPersistence_WebhookSourceByExternalID(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create test webhook source
	config := map[string]any{"timeout": 30}
	source, err := models.NewWebhookSource("test-source", config)
	require.NoError(t, err)

	// Save source
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	// Test external ID lookup (critical for webhook URL resolution)
	externalSource, err := persistence.WebhookSourceByExternalID(source.ExternalID.String())
	require.NoError(t, err)
	require.NotNil(t, externalSource)
	assert.Equal(t, source.ID, externalSource.ID)
	assert.Equal(t, source.ExternalID, externalSource.ExternalID)

	// Test non-existent external ID
	randomUUID := uuid.New()
	nonExistent, err := persistence.WebhookSourceByExternalID(randomUUID.String())
	require.NoError(t, err)
	assert.Nil(t, nonExistent)

	// Test invalid external ID format
	invalidSource, err := persistence.WebhookSourceByExternalID("invalid-uuid")
	require.Error(t, err)
	assert.Nil(t, invalidSource)
	assert.Contains(t, err.Error(), "invalid external ID format")
}

func TestWebhookPersistence_ActiveWebhookSources(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create active source
	activeConfig := map[string]any{"enabled": true}
	activeSource, err := models.NewWebhookSource("active-source", activeConfig)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(activeSource)
	require.NoError(t, err)

	// Create inactive source
	inactiveConfig := map[string]any{"enabled": false}
	inactiveSource, err := models.NewWebhookSource("inactive-source", inactiveConfig)
	require.NoError(t, err)
	inactiveSource.Active = false
	err = persistence.SaveWebhookSource(inactiveSource)
	require.NoError(t, err)

	// Retrieve only active sources
	activeSources, err := persistence.ActiveWebhookSources()
	require.NoError(t, err)
	assert.Len(t, activeSources, 1)
	assert.Equal(t, "active-source", activeSources[0].ID)
	assert.True(t, activeSources[0].Active)

	// Verify all sources still exist
	allSources, err := persistence.WebhookSources()
	require.NoError(t, err)
	assert.Len(t, allSources, 2)
}

func TestWebhookPersistence_UpdateWebhookSource(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create and save initial source
	config := map[string]any{"timeout": 30}
	source, err := models.NewWebhookSource("test-source", config)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	// Get initial timestamps
	originalCreatedAt := source.CreatedAt
	originalUpdatedAt := source.UpdatedAt

	// Update source configuration
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	newConfig := map[string]any{
		"timeout": 60,
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order_id": map[string]any{"type": "string"},
			},
		},
	}
	source.UpdateConfiguration(newConfig)

	// Save updated source
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	// Retrieve and verify update
	retrievedSource, err := persistence.WebhookSourceByID("test-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	// Verify configuration and schema updates (accounting for JSON type conversions)
	assert.Equal(t, float64(60), retrievedSource.Configuration["timeout"]) // JSON converts int to float64
	assert.NotNil(t, retrievedSource.JSONSchema)
	assert.Equal(t, "object", retrievedSource.JSONSchema["type"])

	// Verify timestamps
	assert.Equal(t, originalCreatedAt.Unix(), retrievedSource.CreatedAt.Unix()) // CreatedAt should not change
	assert.True(t, retrievedSource.UpdatedAt.After(originalUpdatedAt))          // UpdatedAt should be newer
}

func TestWebhookPersistence_DeleteWebhookSource(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create and save test sources
	config1 := map[string]any{"timeout": 30}
	source1, err := models.NewWebhookSource("source-1", config1)
	require.NoError(t, err)

	config2 := map[string]any{"timeout": 60}
	source2, err := models.NewWebhookSource("source-2", config2)
	require.NoError(t, err)

	err = persistence.SaveWebhookSource(source1)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(source2)
	require.NoError(t, err)

	// Verify both sources exist
	sources, err := persistence.WebhookSources()
	require.NoError(t, err)
	assert.Len(t, sources, 2)

	// Delete one source
	err = persistence.DeleteWebhookSource("source-1")
	require.NoError(t, err)

	// Verify source was deleted
	deletedSource, err := persistence.WebhookSourceByID("source-1")
	require.NoError(t, err)
	assert.Nil(t, deletedSource)

	// Verify external ID lookup also fails
	deletedSourceByExternal, err := persistence.WebhookSourceByExternalID(source1.ExternalID.String())
	require.NoError(t, err)
	assert.Nil(t, deletedSourceByExternal)

	// Verify other source still exists
	remainingSource, err := persistence.WebhookSourceByID("source-2")
	require.NoError(t, err)
	require.NotNil(t, remainingSource)
	assert.Equal(t, "source-2", remainingSource.ID)

	// Verify total count
	sources, err = persistence.WebhookSources()
	require.NoError(t, err)
	assert.Len(t, sources, 1)
}

func TestWebhookPersistence_ComplexJSONSchemaSerialization(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create source with complex JSON schema
	config := map[string]any{
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"order": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{"type": "string", "pattern": "^[0-9a-f-]{36}$"},
						"items": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"name":     map[string]any{"type": "string"},
									"price":    map[string]any{"type": "number", "minimum": 0},
									"category": map[string]any{"type": "string", "enum": []string{"food", "drink", "dessert"}},
								},
								"required": []string{"name", "price"},
							},
							"minItems": 1,
						},
						"customer": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"email": map[string]any{"type": "string", "format": "email"},
								"phone": map[string]any{"type": "string", "pattern": "^\\+?[1-9]\\d{1,14}$"},
							},
							"anyOf": []map[string]any{
								{"required": []string{"email"}},
								{"required": []string{"phone"}},
							},
						},
					},
					"required": []string{"id", "items", "customer"},
				},
			},
			"required": []string{"order"},
		},
		"validation_options": map[string]any{
			"strict_mode":    true,
			"allow_unknown":  false,
			"error_on_extra": true,
		},
	}

	source, err := models.NewWebhookSource("complex-source", config)
	require.NoError(t, err)

	// Save and retrieve
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	retrievedSource, err := persistence.WebhookSourceByID("complex-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	// Verify all complex data was preserved (structure, not exact types due to JSON marshaling)
	assert.NotNil(t, retrievedSource.JSONSchema)
	assert.NotNil(t, retrievedSource.Configuration)

	// Test key structure preservation instead of exact equality due to JSON type conversions
	assert.Equal(t, "object", retrievedSource.JSONSchema["type"])
	assert.Contains(t, retrievedSource.JSONSchema, "properties")
	assert.Contains(t, retrievedSource.JSONSchema, "required")

	// Verify specific nested values
	require.NotNil(t, retrievedSource.JSONSchema)
	jsonSchema := retrievedSource.JSONSchema

	properties, ok := jsonSchema["properties"].(map[string]any)
	require.True(t, ok, "Properties should be map[string]any")

	orderField, ok := properties["order"].(map[string]any)
	require.True(t, ok, "Order field should be map[string]any")

	orderProps, ok := orderField["properties"].(map[string]any)
	require.True(t, ok, "Order properties should be map[string]any")

	// Test deeply nested structure preservation
	itemsField, ok := orderProps["items"].(map[string]any)
	require.True(t, ok, "Items field should be map[string]any")

	itemsItems, ok := itemsField["items"].(map[string]any)
	require.True(t, ok, "Items items should be map[string]any")

	itemsProps, ok := itemsItems["properties"].(map[string]any)
	require.True(t, ok, "Items properties should be map[string]any")

	priceField, ok := itemsProps["price"].(map[string]any)
	require.True(t, ok, "Price field should be map[string]any")

	assert.Equal(t, float64(0), priceField["minimum"])

	// Test validation options
	validationOptions, ok := retrievedSource.Configuration["validation_options"].(map[string]any)
	require.True(t, ok, "Validation options should be map[string]any")
	assert.Equal(t, true, validationOptions["strict_mode"])
	assert.Equal(t, false, validationOptions["allow_unknown"])
}

func TestWebhookPersistence_NullJSONSchemaHandling(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create source without JSON schema
	config := map[string]any{
		"timeout": 30,
		"enabled": true,
	}
	source, err := models.NewWebhookSource("no-schema-source", config)
	require.NoError(t, err)

	// Ensure no JSON schema is set
	source.JSONSchema = nil

	// Save and retrieve
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	retrievedSource, err := persistence.WebhookSourceByID("no-schema-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	// Verify JSON schema is nil/empty
	assert.Nil(t, retrievedSource.JSONSchema)
	assert.False(t, retrievedSource.HasJSONSchema())

	// Verify configuration is preserved (accounting for JSON type conversions)
	assert.Equal(t, float64(30), retrievedSource.Configuration["timeout"])
	assert.Equal(t, true, retrievedSource.Configuration["enabled"])
}

func TestWebhookPersistence_UUIDExternalIDHandling(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	testCases := []struct {
		name       string
		sourceID   string
		expectUUID bool
	}{
		{
			name:       "valid source creates random UUID",
			sourceID:   "test-source-1",
			expectUUID: true,
		},
		{
			name:       "second source gets different UUID",
			sourceID:   "test-source-2",
			expectUUID: true,
		},
	}

	var previousUUID uuid.UUID
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]any{"timeout": 30}
			source, err := models.NewWebhookSource(tc.sourceID, config)
			require.NoError(t, err)

			if tc.expectUUID {
				assert.NotEqual(t, uuid.Nil, source.ExternalID)
				assert.NotEqual(t, previousUUID, source.ExternalID) // Different from previous
				previousUUID = source.ExternalID
			}

			// Save and retrieve by external ID
			err = persistence.SaveWebhookSource(source)
			require.NoError(t, err)

			retrievedSource, err := persistence.WebhookSourceByExternalID(source.ExternalID.String())
			require.NoError(t, err)
			require.NotNil(t, retrievedSource)
			assert.Equal(t, source.ID, retrievedSource.ID)
			assert.Equal(t, source.ExternalID, retrievedSource.ExternalID)

			// Test webhook URL generation
			expectedURL := "/webhook/" + source.ExternalID.String()
			assert.Equal(t, expectedURL, source.GetWebhookURL())
		})
	}
}

func TestWebhookPersistence_HealthCheck(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Test successful health check
	err := persistence.HealthCheck()
	assert.NoError(t, err)

	// Test health check with some data
	config := map[string]any{"timeout": 30}
	source, err := models.NewWebhookSource("health-test", config)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(source)
	require.NoError(t, err)

	err = persistence.HealthCheck()
	assert.NoError(t, err)
}

func TestWebhookPersistence_PerformanceExternalIDLookup(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create a mix of webhook sources
	var externalIDs []string
	for i := 0; i < 100; i++ {
		config := map[string]any{"timeout": 30}
		source, err := models.NewWebhookSource(fmt.Sprintf("source-%d", i), config)
		require.NoError(t, err)

		if i%10 == 0 {
			source.Active = false // Some inactive
		}

		err = persistence.SaveWebhookSource(source)
		require.NoError(t, err)

		externalIDs = append(externalIDs, source.ExternalID.String())
	}

	// Measure external ID lookup performance (critical for webhook requests)
	start := time.Now()
	for _, externalID := range externalIDs {
		_, err := persistence.WebhookSourceByExternalID(externalID)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	// Performance should be well under 10ms per lookup for 100 lookups (PRP requirement)
	avgDuration := duration / time.Duration(len(externalIDs))
	assert.Less(t, avgDuration, 10*time.Millisecond, "External ID lookups should be fast")

	t.Logf("External ID lookups completed: %v total for %d lookups (avg %v per lookup)",
		duration, len(externalIDs), avgDuration)
}

func TestWebhookPersistence_ConcurrentAccess(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	const numGoroutines = 10
	const sourcesPerGoroutine = 10

	// Test concurrent writes
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()

			for j := 0; j < sourcesPerGoroutine; j++ {
				sourceID := fmt.Sprintf("concurrent-source-%d-%d", routineID, j)
				config := map[string]any{"timeout": 30}

				source, err := models.NewWebhookSource(sourceID, config)
				if err != nil {
					t.Errorf("Failed to create webhook source: %v", err)
					return
				}

				err = persistence.SaveWebhookSource(source)
				if err != nil {
					t.Errorf("Failed to save webhook source: %v", err)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all sources were saved
	allSources, err := persistence.WebhookSources()
	require.NoError(t, err)
	assert.Len(t, allSources, numGoroutines*sourcesPerGoroutine)
}

func TestWebhookPersistence_ErrorHandling(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Test saving source with duplicate ID should update, not error
	config1 := map[string]any{"timeout": 30}
	source1, err := models.NewWebhookSource("duplicate-id", config1)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(source1)
	require.NoError(t, err)

	// Saving same ID should update, not error
	config2 := map[string]any{"timeout": 60}
	source2, err := models.NewWebhookSource("duplicate-id", config2)
	require.NoError(t, err)
	err = persistence.SaveWebhookSource(source2)
	require.NoError(t, err)

	// Verify update worked
	retrieved, err := persistence.WebhookSourceByID("duplicate-id")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	// JSON unmarshals int as float64
	assert.Equal(t, float64(60), retrieved.Configuration["timeout"])
}
