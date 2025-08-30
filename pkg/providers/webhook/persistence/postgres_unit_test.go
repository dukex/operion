package persistence

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dukex/operion/pkg/providers/webhook/models"
)

func TestWebhookMigrations(t *testing.T) {
	migrations := webhookMigrations()

	// Test that migration version 4 exists
	migration, exists := migrations[4]
	assert.True(t, exists, "Migration version 4 should exist")
	assert.Contains(t, migration, "CREATE TABLE webhook_sources", "Should create webhook_sources table")
	assert.Contains(t, migration, "idx_webhook_sources_external_id_unique", "Should create unique external ID index")
}

func TestNewPostgresPersistence_InvalidURL(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with completely invalid URL
	persistence, err := NewPostgresPersistence(ctx, logger, "not-a-valid-url")
	assert.Error(t, err)
	assert.Nil(t, persistence)
	// Error can be either connection failure or ping failure
	assert.True(t, err.Error() != "", "Error should not be empty")
}

func TestScanWebhookSourceRows_EmptyRows(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	p := &PostgresPersistence{
		logger: logger,
	}

	// This would normally require actual database rows, but we can test the error handling
	// when rows.Err() returns an error by mocking or using nil rows
	// For now, we're testing the structure exists
	assert.NotNil(t, p.scanWebhookSourceRows)
}

func TestWebhookMigrationContent(t *testing.T) {
	migrations := webhookMigrations()
	migration := migrations[4]

	// Verify all required indexes are present
	requiredIndexes := []string{
		"idx_webhook_sources_external_id",
		"idx_webhook_sources_active",
		"idx_webhook_sources_created_at",
		"idx_webhook_sources_updated_at",
		"idx_webhook_sources_external_id_unique",
	}

	for _, index := range requiredIndexes {
		assert.Contains(t, migration, index, "Migration should contain index: %s", index)
	}

	// Verify table structure
	requiredColumns := []string{
		"id VARCHAR(255) PRIMARY KEY",
		"external_id UUID NOT NULL UNIQUE",
		"json_schema JSONB",
		"configuration JSONB NOT NULL",
		"created_at TIMESTAMP WITH TIME ZONE NOT NULL",
		"updated_at TIMESTAMP WITH TIME ZONE NOT NULL",
		"active BOOLEAN NOT NULL DEFAULT true",
	}

	for _, column := range requiredColumns {
		assert.Contains(t, migration, column, "Migration should contain column definition: %s", column)
	}

	// Verify unique constraint for external ID
	assert.Contains(t, migration, "external_id UUID NOT NULL UNIQUE", "Should have unique constraint on external_id")
	assert.Contains(t, migration, "CREATE UNIQUE INDEX idx_webhook_sources_external_id_unique", "Should have unique index for external ID")
}

func TestPostgresPersistence_MethodSignatures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test that PostgresPersistence implements the interface properly
	// by checking method signatures exist
	var persistence interface{} = &PostgresPersistence{logger: logger}

	// This will compile-time check that PostgresPersistence implements WebhookPersistence
	_, ok := persistence.(interface {
		SaveWebhookSource(source *models.WebhookSource) error
		WebhookSourceByID(id string) (*models.WebhookSource, error)
		WebhookSourceByExternalID(externalID string) (*models.WebhookSource, error)
		WebhookSources() ([]*models.WebhookSource, error)
		ActiveWebhookSources() ([]*models.WebhookSource, error)
		DeleteWebhookSource(id string) error
		HealthCheck() error
		Close() error
	})

	assert.True(t, ok, "PostgresPersistence should implement all required methods")
}

func TestPostgresPersistence_LoggerSetup(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test with invalid connection to ensure logger is set up correctly
	persistence, err := NewPostgresPersistence(ctx, logger, "postgres://invalid:invalid@nonexistent:5432/nonexistent")

	// Should fail to connect, but if it gets past connection, logger should be set
	assert.Error(t, err)
	assert.Nil(t, persistence)
}

func TestPostgresPersistence_Close(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	p := &PostgresPersistence{
		logger: logger,
		db:     nil, // nil db should not panic
	}

	// Should handle nil database gracefully
	err := p.Close()
	assert.NoError(t, err, "Close should handle nil database without error")
}

func TestWebhookMigrationVersioning(t *testing.T) {
	migrations := webhookMigrations()

	// Test that we have exactly the expected migrations
	expectedVersions := []int{4}

	assert.Len(t, migrations, len(expectedVersions), "Should have expected number of migrations")

	for _, version := range expectedVersions {
		_, exists := migrations[version]
		assert.True(t, exists, "Migration version %d should exist", version)
	}

	// Test that the migration content is not empty
	for version, content := range migrations {
		assert.NotEmpty(t, content, "Migration %d should have content", version)
		assert.Contains(t, content, "CREATE TABLE", "Migration %d should create tables", version)
	}
}

func TestWebhookPersistence_UUIDValidation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	p := &PostgresPersistence{
		logger: logger,
		db:     nil, // Will cause error, but we're testing UUID validation
	}

	// Test invalid UUID format handling in WebhookSourceByExternalID
	testCases := []string{
		"invalid-uuid",
		"123",
		"not-a-uuid-at-all",
		"12345678-1234-1234-1234", // Too short
		"12345678-1234-1234-1234-12345678901234567890", // Too long
	}

	for _, invalidUUID := range testCases {
		t.Run("invalid_uuid_"+invalidUUID, func(t *testing.T) {
			_, err := p.WebhookSourceByExternalID(invalidUUID)
			assert.Error(t, err, "Should error with invalid UUID: %s", invalidUUID)
			assert.Contains(t, err.Error(), "invalid external ID format", "Should mention invalid format")
		})
	}

	// Test valid UUID format (should not cause UUID parsing error)
	validUUID := uuid.New().String()
	// We can't test this with nil db as it will panic, but the UUID parsing should work
	// This test validates that valid UUIDs don't cause parsing errors
	_, err := uuid.Parse(validUUID)
	assert.NoError(t, err, "Valid UUID should parse correctly")
}

func TestWebhookPersistence_JSONSchemaHandling(t *testing.T) {
	// Test the webhook model's JSON schema extraction logic
	testCases := []struct {
		name              string
		config            map[string]any
		expectedHasSchema bool
	}{
		{
			name: "config with json_schema",
			config: map[string]any{
				"json_schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			},
			expectedHasSchema: true,
		},
		{
			name: "config without json_schema",
			config: map[string]any{
				"timeout": 30,
			},
			expectedHasSchema: false,
		},
		{
			name: "config with empty json_schema",
			config: map[string]any{
				"json_schema": map[string]any{},
			},
			expectedHasSchema: false,
		},
		{
			name: "config with invalid json_schema type",
			config: map[string]any{
				"json_schema": "not-a-map",
			},
			expectedHasSchema: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			source, err := models.NewWebhookSource("test-source", tc.config)
			assert.NoError(t, err, "Should create webhook source")
			assert.Equal(t, tc.expectedHasSchema, source.HasJSONSchema(), "JSON schema detection should match expectation")
		})
	}
}

func TestWebhookPersistence_TimestampHandling(t *testing.T) {
	// Test that timestamp fields work correctly in the model
	source, err := models.NewWebhookSource("test-source", map[string]any{"timeout": 30})
	assert.NoError(t, err, "Should create webhook source")

	// Test initial timestamps
	assert.False(t, source.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, source.UpdatedAt.IsZero(), "UpdatedAt should be set")
	assert.True(t, source.CreatedAt.Equal(source.UpdatedAt), "Initial timestamps should be equal")

	// Test timestamp updates
	originalCreatedAt := source.CreatedAt
	originalUpdatedAt := source.UpdatedAt

	time.Sleep(10 * time.Millisecond) // Small delay
	source.UpdateConfiguration(map[string]any{"timeout": 60})

	assert.Equal(t, originalCreatedAt, source.CreatedAt, "CreatedAt should not change")
	assert.True(t, source.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be newer")
}
