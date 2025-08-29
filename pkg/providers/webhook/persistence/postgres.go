package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/persistence/sqlbase"
	webhookModels "github.com/dukex/operion/pkg/providers/webhook/models"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

// PostgresPersistence implements WebhookPersistence using PostgreSQL database.
type PostgresPersistence struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresPersistence creates a new PostgreSQL persistence layer for webhook.
func NewPostgresPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) (*PostgresPersistence, error) {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	err = database.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize migration manager with version 4 migrations
	migrationManager := sqlbase.NewMigrationManager(logger, database, webhookMigrations())

	postgres := &PostgresPersistence{
		db:     database,
		logger: logger.With("component", "webhook_postgres_persistence"),
	}

	// Run migrations on initialization
	err = migrationManager.RunMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run Webhook migrations: %w", err)
	}

	logger.InfoContext(ctx, "Webhook PostgreSQL persistence initialized successfully")

	return postgres, nil
}

// SaveWebhookSource saves or updates a webhook source in the database.
func (p *PostgresPersistence) SaveWebhookSource(source *webhookModels.WebhookSource) error {
	ctx := context.Background()

	query := `
		INSERT INTO webhook_sources (
			id, external_id, json_schema, configuration, 
			created_at, updated_at, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) 
		DO UPDATE SET
			external_id = EXCLUDED.external_id,
			json_schema = EXCLUDED.json_schema,
			configuration = EXCLUDED.configuration,
			updated_at = EXCLUDED.updated_at,
			active = EXCLUDED.active
	`

	// Handle optional JSON schema
	var jsonSchemaJSON sql.NullString

	if len(source.JSONSchema) > 0 {
		jsonBytes, err := json.Marshal(source.JSONSchema)
		if err != nil {
			p.logger.ErrorContext(ctx, "Failed to serialize JSON schema", "source_id", source.ID, "error", err)

			return fmt.Errorf("failed to serialize JSON schema: %w", err)
		}

		jsonSchemaJSON = sql.NullString{String: string(jsonBytes), Valid: true}
	}

	// Handle required configuration
	configurationJSON, err := json.Marshal(source.Configuration)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to serialize configuration", "source_id", source.ID, "error", err)

		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}

	source.UpdatedAt = now

	_, err = p.db.ExecContext(ctx, query,
		source.ID,
		source.ExternalID,
		jsonSchemaJSON,
		string(configurationJSON),
		source.CreatedAt,
		source.UpdatedAt,
		source.Active,
	)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to save webhook source", "source_id", source.ID, "error", err)

		return fmt.Errorf("failed to save webhook source: %w", err)
	}

	p.logger.DebugContext(ctx, "Webhook source saved successfully", "source_id", source.ID, "external_id", source.ExternalID)

	return nil
}

// WebhookSourceByID retrieves a webhook source by its ID.
func (p *PostgresPersistence) WebhookSourceByID(id string) (*webhookModels.WebhookSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, external_id, json_schema, configuration, created_at, updated_at, active
		FROM webhook_sources 
		WHERE id = $1
	`

	row := p.db.QueryRowContext(ctx, query, id)

	var (
		jsonSchemaJSON    sql.NullString
		configurationJSON string
	)

	source := &webhookModels.WebhookSource{}

	err := row.Scan(
		&source.ID,
		&source.ExternalID,
		&jsonSchemaJSON,
		&configurationJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
		&source.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Source not found
		}

		p.logger.ErrorContext(ctx, "Failed to scan webhook source", "source_id", id, "error", err)

		return nil, fmt.Errorf("failed to scan webhook source: %w", err)
	}

	// Deserialize JSON fields
	if jsonSchemaJSON.Valid && jsonSchemaJSON.String != "" {
		if err := json.Unmarshal([]byte(jsonSchemaJSON.String), &source.JSONSchema); err != nil {
			return nil, fmt.Errorf("failed to deserialize JSON schema: %w", err)
		}
	}

	if err := json.Unmarshal([]byte(configurationJSON), &source.Configuration); err != nil {
		return nil, fmt.Errorf("failed to deserialize configuration: %w", err)
	}

	p.logger.DebugContext(ctx, "Webhook source retrieved successfully", "source_id", id)

	return source, nil
}

// WebhookSourceByExternalID retrieves a webhook source by its external ID (for URL resolution).
// This is the most critical method for webhook URL resolution performance.
func (p *PostgresPersistence) WebhookSourceByExternalID(externalID string) (*webhookModels.WebhookSource, error) {
	ctx := context.Background()

	// Parse external ID as UUID
	externalUUID, err := uuid.Parse(externalID)
	if err != nil {
		p.logger.ErrorContext(ctx, "Invalid external ID format", "external_id", externalID, "error", err)

		return nil, fmt.Errorf("invalid external ID format: %w", err)
	}

	query := `
		SELECT id, external_id, json_schema, configuration, created_at, updated_at, active
		FROM webhook_sources 
		WHERE external_id = $1
	`

	row := p.db.QueryRowContext(ctx, query, externalUUID)

	var (
		jsonSchemaJSON    sql.NullString
		configurationJSON string
	)

	source := &webhookModels.WebhookSource{}

	err = row.Scan(
		&source.ID,
		&source.ExternalID,
		&jsonSchemaJSON,
		&configurationJSON,
		&source.CreatedAt,
		&source.UpdatedAt,
		&source.Active,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Source not found
		}

		p.logger.ErrorContext(ctx, "Failed to scan webhook source by external ID", "external_id", externalID, "error", err)

		return nil, fmt.Errorf("failed to scan webhook source: %w", err)
	}

	// Deserialize JSON fields
	if jsonSchemaJSON.Valid && jsonSchemaJSON.String != "" {
		if err := json.Unmarshal([]byte(jsonSchemaJSON.String), &source.JSONSchema); err != nil {
			return nil, fmt.Errorf("failed to deserialize JSON schema: %w", err)
		}
	}

	if err := json.Unmarshal([]byte(configurationJSON), &source.Configuration); err != nil {
		return nil, fmt.Errorf("failed to deserialize configuration: %w", err)
	}

	p.logger.DebugContext(ctx, "Webhook source retrieved by external ID", "external_id", externalID, "source_id", source.ID)

	return source, nil
}

// WebhookSources retrieves all webhook sources from the database.
func (p *PostgresPersistence) WebhookSources() ([]*webhookModels.WebhookSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, external_id, json_schema, configuration, created_at, updated_at, active
		FROM webhook_sources 
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query all webhook sources", "error", err)

		return nil, fmt.Errorf("failed to query webhook sources: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	sources, err := p.scanWebhookSourceRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "All webhook sources retrieved", "count", len(sources))

	return sources, nil
}

// ActiveWebhookSources retrieves all active webhook sources from the database.
func (p *PostgresPersistence) ActiveWebhookSources() ([]*webhookModels.WebhookSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, external_id, json_schema, configuration, created_at, updated_at, active
		FROM webhook_sources 
		WHERE active = true
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query active webhook sources", "error", err)

		return nil, fmt.Errorf("failed to query active webhook sources: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	sources, err := p.scanWebhookSourceRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "Active webhook sources retrieved", "count", len(sources))

	return sources, nil
}

// DeleteWebhookSource deletes a webhook source from the database.
func (p *PostgresPersistence) DeleteWebhookSource(id string) error {
	ctx := context.Background()

	query := `DELETE FROM webhook_sources WHERE id = $1`

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to delete webhook source", "source_id", id, "error", err)

		return fmt.Errorf("failed to delete webhook source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.logger.DebugContext(ctx, "Webhook source deletion completed", "source_id", id, "rows_affected", rowsAffected)

	return nil
}

// HealthCheck verifies the database connection is healthy.
func (p *PostgresPersistence) HealthCheck() error {
	ctx := context.Background()

	err := p.db.PingContext(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, "Database health check failed", "error", err)

		return fmt.Errorf("database health check failed: %w", err)
	}

	// Test with a simple query
	var count int

	err = p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM webhook_sources").Scan(&count)
	if err != nil {
		p.logger.ErrorContext(ctx, "Database table query failed", "error", err)

		return fmt.Errorf("database table query failed: %w", err)
	}

	p.logger.DebugContext(ctx, "Database health check passed", "webhook_sources_count", count)

	return nil
}

// Close closes the database connection.
func (p *PostgresPersistence) Close() error {
	ctx := context.Background()

	if p.db != nil {
		err := p.db.Close()
		if err != nil {
			p.logger.ErrorContext(ctx, "Failed to close database connection", "error", err)

			return fmt.Errorf("failed to close database connection: %w", err)
		}

		p.logger.InfoContext(ctx, "Database connection closed successfully")
	}

	return nil
}

// scanWebhookSourceRows scans database rows into WebhookSource structs to reduce code duplication.
func (p *PostgresPersistence) scanWebhookSourceRows(ctx context.Context, rows *sql.Rows) ([]*webhookModels.WebhookSource, error) {
	var sources []*webhookModels.WebhookSource

	for rows.Next() {
		var (
			jsonSchemaJSON    sql.NullString
			configurationJSON string
		)

		source := &webhookModels.WebhookSource{}

		err := rows.Scan(
			&source.ID,
			&source.ExternalID,
			&jsonSchemaJSON,
			&configurationJSON,
			&source.CreatedAt,
			&source.UpdatedAt,
			&source.Active,
		)
		if err != nil {
			p.logger.ErrorContext(ctx, "Failed to scan webhook source row", "error", err)

			return nil, fmt.Errorf("failed to scan webhook source: %w", err)
		}

		// Deserialize JSON fields
		if jsonSchemaJSON.Valid && jsonSchemaJSON.String != "" {
			if err := json.Unmarshal([]byte(jsonSchemaJSON.String), &source.JSONSchema); err != nil {
				return nil, fmt.Errorf("failed to deserialize JSON schema: %w", err)
			}
		}

		if err := json.Unmarshal([]byte(configurationJSON), &source.Configuration); err != nil {
			return nil, fmt.Errorf("failed to deserialize configuration: %w", err)
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating webhook source rows: %w", err)
	}

	return sources, nil
}

// webhookMigrations returns the migration scripts for Webhook-specific tables.
func webhookMigrations() map[int]string {
	return map[int]string{
		4: `
			-- Create webhook_sources table for Webhook provider persistence
			CREATE TABLE webhook_sources (
				id VARCHAR(255) PRIMARY KEY,
				external_id UUID NOT NULL UNIQUE,
				json_schema JSONB,
				configuration JSONB NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				active BOOLEAN NOT NULL DEFAULT true
			);

			-- Create indexes for better query performance
			CREATE INDEX idx_webhook_sources_external_id ON webhook_sources(external_id);
			CREATE INDEX idx_webhook_sources_active ON webhook_sources(active);
			CREATE INDEX idx_webhook_sources_created_at ON webhook_sources(created_at);
			CREATE INDEX idx_webhook_sources_updated_at ON webhook_sources(updated_at);
			
			-- Unique index for external ID lookups (critical for webhook URL resolution)
			CREATE UNIQUE INDEX idx_webhook_sources_external_id_unique ON webhook_sources(external_id);
		`,
	}
}
