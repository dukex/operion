package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/persistence/sqlbase"
	kafkaModels "github.com/dukex/operion/pkg/providers/kafka/models"

	_ "github.com/lib/pq"
)

// PostgresPersistence implements KafkaPersistence using PostgreSQL database.
type PostgresPersistence struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresPersistence creates a new PostgreSQL persistence layer for Kafka sources.
func NewPostgresPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) (*PostgresPersistence, error) {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	err = database.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize migration manager
	migrationManager := sqlbase.NewMigrationManager(logger, database, kafkaMigrations())

	postgres := &PostgresPersistence{
		db:     database,
		logger: logger.With("component", "kafka_postgres_persistence"),
	}

	// Run migrations on initialization
	err = migrationManager.RunMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run Kafka migrations: %w", err)
	}

	logger.InfoContext(ctx, "Kafka PostgreSQL persistence initialized successfully")

	return postgres, nil
}

// SaveKafkaSource saves or updates a Kafka source in the database.
func (p *PostgresPersistence) SaveKafkaSource(source *kafkaModels.KafkaSource) error {
	ctx := context.Background()

	query := `
		INSERT INTO kafka_sources (
			id, connection_details_id, connection_details, 
			json_schema, configuration, active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) 
		DO UPDATE SET
			connection_details_id = EXCLUDED.connection_details_id,
			connection_details = EXCLUDED.connection_details,
			json_schema = EXCLUDED.json_schema,
			configuration = EXCLUDED.configuration,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`

	// Convert structs to JSON for database storage
	connectionDetailsJSON, err := kafkaModels.StructToJSON(source.ConnectionDetails)
	if err != nil {
		return fmt.Errorf("failed to serialize connection details: %w", err)
	}

	jsonSchemaJSON, err := kafkaModels.StructToJSON(source.JSONSchema)
	if err != nil {
		return fmt.Errorf("failed to serialize JSON schema: %w", err)
	}

	configurationJSON, err := kafkaModels.StructToJSON(source.Configuration)
	if err != nil {
		return fmt.Errorf("failed to serialize configuration: %w", err)
	}

	now := time.Now().UTC()
	if source.CreatedAt.IsZero() {
		source.CreatedAt = now
	}

	source.UpdatedAt = now

	_, err = p.db.ExecContext(ctx, query,
		source.ID,
		source.ConnectionDetailsID,
		connectionDetailsJSON,
		jsonSchemaJSON,
		configurationJSON,
		source.Active,
		source.CreatedAt,
		source.UpdatedAt,
	)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to save Kafka source", "source_id", source.ID, "error", err)

		return fmt.Errorf("failed to save Kafka source: %w", err)
	}

	p.logger.DebugContext(ctx, "Kafka source saved successfully", "source_id", source.ID)

	return nil
}

// KafkaSourceByID retrieves a Kafka source by its ID.
func (p *PostgresPersistence) KafkaSourceByID(id string) (*kafkaModels.KafkaSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, connection_details_id, connection_details, 
			   json_schema, configuration, active, created_at, updated_at
		FROM kafka_sources 
		WHERE id = $1
	`

	row := p.db.QueryRowContext(ctx, query, id)

	var (
		connectionDetailsJSON string
		jsonSchemaJSON        string
		configurationJSON     string
	)

	source := &kafkaModels.KafkaSource{}

	err := row.Scan(
		&source.ID,
		&source.ConnectionDetailsID,
		&connectionDetailsJSON,
		&jsonSchemaJSON,
		&configurationJSON,
		&source.Active,
		&source.CreatedAt,
		&source.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Source not found
		}

		p.logger.ErrorContext(ctx, "Failed to scan Kafka source", "source_id", id, "error", err)

		return nil, fmt.Errorf("failed to scan Kafka source: %w", err)
	}

	// Deserialize JSON fields
	if err := kafkaModels.JSONToStruct(connectionDetailsJSON, &source.ConnectionDetails); err != nil {
		return nil, fmt.Errorf("failed to deserialize connection details: %w", err)
	}

	const nullJSON = "null"
	if jsonSchemaJSON != nullJSON && jsonSchemaJSON != "" {
		if err := kafkaModels.JSONToStruct(jsonSchemaJSON, &source.JSONSchema); err != nil {
			return nil, fmt.Errorf("failed to deserialize JSON schema: %w", err)
		}
	}

	if err := kafkaModels.JSONToStruct(configurationJSON, &source.Configuration); err != nil {
		return nil, fmt.Errorf("failed to deserialize configuration: %w", err)
	}

	p.logger.DebugContext(ctx, "Kafka source retrieved successfully", "source_id", id)

	return source, nil
}

// KafkaSourceByConnectionDetailsID retrieves all Kafka sources sharing the same connection details ID.
func (p *PostgresPersistence) KafkaSourceByConnectionDetailsID(connectionDetailsID string) ([]*kafkaModels.KafkaSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, connection_details_id, connection_details, 
			   json_schema, configuration, active, created_at, updated_at
		FROM kafka_sources 
		WHERE connection_details_id = $1
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query, connectionDetailsID)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query Kafka sources by connection details ID", "connection_details_id", connectionDetailsID, "error", err)

		return nil, fmt.Errorf("failed to query Kafka sources: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	sources, err := p.scanKafkaSourceRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "Kafka sources retrieved by connection details ID", "connection_details_id", connectionDetailsID, "count", len(sources))

	return sources, nil
}

// KafkaSources retrieves all Kafka sources from the database.
func (p *PostgresPersistence) KafkaSources() ([]*kafkaModels.KafkaSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, connection_details_id, connection_details, 
			   json_schema, configuration, active, created_at, updated_at
		FROM kafka_sources 
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query all Kafka sources", "error", err)

		return nil, fmt.Errorf("failed to query Kafka sources: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	sources, err := p.scanKafkaSourceRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "All Kafka sources retrieved", "count", len(sources))

	return sources, nil
}

// ActiveKafkaSources retrieves all active Kafka sources from the database.
func (p *PostgresPersistence) ActiveKafkaSources() ([]*kafkaModels.KafkaSource, error) {
	ctx := context.Background()

	query := `
		SELECT id, connection_details_id, connection_details, 
			   json_schema, configuration, active, created_at, updated_at
		FROM kafka_sources 
		WHERE active = true
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query active Kafka sources", "error", err)

		return nil, fmt.Errorf("failed to query active Kafka sources: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	sources, err := p.scanKafkaSourceRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "Active Kafka sources retrieved", "count", len(sources))

	return sources, nil
}

// DeleteKafkaSource deletes a Kafka source from the database.
func (p *PostgresPersistence) DeleteKafkaSource(id string) error {
	ctx := context.Background()

	query := `DELETE FROM kafka_sources WHERE id = $1`

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to delete Kafka source", "source_id", id, "error", err)

		return fmt.Errorf("failed to delete Kafka source: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.logger.DebugContext(ctx, "Kafka source deletion completed", "source_id", id, "rows_affected", rowsAffected)

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

	err = p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kafka_sources").Scan(&count)
	if err != nil {
		p.logger.ErrorContext(ctx, "Database table query failed", "error", err)

		return fmt.Errorf("database table query failed: %w", err)
	}

	p.logger.DebugContext(ctx, "Database health check passed", "kafka_sources_count", count)

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

// scanKafkaSourceRows scans database rows into KafkaSource structs to reduce code duplication.
func (p *PostgresPersistence) scanKafkaSourceRows(ctx context.Context, rows *sql.Rows) ([]*kafkaModels.KafkaSource, error) {
	var sources []*kafkaModels.KafkaSource

	for rows.Next() {
		var (
			connectionDetailsJSON string
			jsonSchemaJSON        string
			configurationJSON     string
		)

		source := &kafkaModels.KafkaSource{}

		err := rows.Scan(
			&source.ID,
			&source.ConnectionDetailsID,
			&connectionDetailsJSON,
			&jsonSchemaJSON,
			&configurationJSON,
			&source.Active,
			&source.CreatedAt,
			&source.UpdatedAt,
		)
		if err != nil {
			p.logger.ErrorContext(ctx, "Failed to scan Kafka source row", "error", err)

			return nil, fmt.Errorf("failed to scan Kafka source: %w", err)
		}

		// Deserialize JSON fields
		if err := kafkaModels.JSONToStruct(connectionDetailsJSON, &source.ConnectionDetails); err != nil {
			return nil, fmt.Errorf("failed to deserialize connection details: %w", err)
		}

		const nullJSON = "null"
		if jsonSchemaJSON != nullJSON && jsonSchemaJSON != "" {
			if err := kafkaModels.JSONToStruct(jsonSchemaJSON, &source.JSONSchema); err != nil {
				return nil, fmt.Errorf("failed to deserialize JSON schema: %w", err)
			}
		}

		if err := kafkaModels.JSONToStruct(configurationJSON, &source.Configuration); err != nil {
			return nil, fmt.Errorf("failed to deserialize configuration: %w", err)
		}

		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating Kafka source rows: %w", err)
	}

	return sources, nil
}

// kafkaMigrations returns the migration scripts for Kafka-specific tables.
func kafkaMigrations() map[int]string {
	return map[int]string{
		2: `
			-- Create kafka_sources table for Kafka provider persistence
			CREATE TABLE kafka_sources (
				id VARCHAR(255) PRIMARY KEY,
				connection_details_id VARCHAR(255) NOT NULL,
				connection_details JSONB NOT NULL,
				json_schema JSONB,
				configuration JSONB NOT NULL,
				active BOOLEAN NOT NULL DEFAULT true,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);

			-- Create indexes for better query performance
			CREATE INDEX idx_kafka_sources_connection_details_id ON kafka_sources(connection_details_id);
			CREATE INDEX idx_kafka_sources_active ON kafka_sources(active);
			CREATE INDEX idx_kafka_sources_created_at ON kafka_sources(created_at);
			CREATE INDEX idx_kafka_sources_updated_at ON kafka_sources(updated_at);
		`,
	}
}
