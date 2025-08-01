package sqlbase

import (
	"database/sql"
	"fmt"
	"log/slog"
)

const (
	// Migration version for schema versioning
	currentSchemaVersion = 1
)

// MigrationManager handles database schema migrations
type MigrationManager struct {
	db     *sql.DB
	logger *slog.Logger
	migrations map[int]string 
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(logger *slog.Logger, db *sql.DB, migrations map[int]string) *MigrationManager {
	return &MigrationManager{
		db:        db,
		logger:    logger,
		migrations: migrations,
	}
}

// RunMigrations handles database schema creation and updates
func (m *MigrationManager) RunMigrations() error {
	m.logger.Info("Running database migrations")

	// Create schema_migrations table if it doesn't exist
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current schema version
	currentVersion, err := m.getCurrentSchemaVersion()
	if err != nil {
		return fmt.Errorf("failed to get current schema version: %w", err)
	}

	m.logger.Info("Current schema version", "version", currentVersion)

	// Apply migrations if needed
	if currentVersion < currentSchemaVersion {
		if err := m.applyMigrations(currentVersion); err != nil {
			return fmt.Errorf("failed to apply migrations: %w", err)
		}
	}

	m.logger.Info("Database migrations completed", "version", currentSchemaVersion)
	return nil
}

// createMigrationsTable creates the schema_migrations table
func (m *MigrationManager) createMigrationsTable() error {
	createMigrationsSQL := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`
	_, err := m.db.Exec(createMigrationsSQL)
	return err
}

// getCurrentSchemaVersion returns the current schema version
func (m *MigrationManager) getCurrentSchemaVersion() (int, error) {
	var version int
	err := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// applyMigrations applies all migrations from the current version to the latest
func (m *MigrationManager) applyMigrations(fromVersion int) error {
	for version, migration := range m.migrations {
		if version > fromVersion {
			m.logger.Info("Applying migration", "version", version)
			
			tx, err := m.db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction for migration %d: %w", version, err)
			}

			// Execute migration SQL
			if _, err := tx.Exec(migration); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to execute migration %d: %w", version, err)
			}

			// Record migration
			if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to record migration %d: %w", version, err)
			}

			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit migration %d: %w", version, err)
			}

			m.logger.Info("Migration applied successfully", "version", version)
		}
	}

	return nil
}