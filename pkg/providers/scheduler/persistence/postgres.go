package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/persistence/sqlbase"
	schedulerModels "github.com/dukex/operion/pkg/providers/scheduler/models"

	_ "github.com/lib/pq"
)

// PostgresPersistence implements SchedulerPersistence using PostgreSQL database.
type PostgresPersistence struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewPostgresPersistence creates a new PostgreSQL persistence layer for scheduler.
func NewPostgresPersistence(ctx context.Context, logger *slog.Logger, databaseURL string) (*PostgresPersistence, error) {
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL database: %w", err)
	}

	err = database.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize migration manager with version 3 migrations
	migrationManager := sqlbase.NewMigrationManager(logger, database, schedulerMigrations())

	postgres := &PostgresPersistence{
		db:     database,
		logger: logger.With("component", "scheduler_postgres_persistence"),
	}

	// Run migrations on initialization
	err = migrationManager.RunMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to run Scheduler migrations: %w", err)
	}

	logger.InfoContext(ctx, "Scheduler PostgreSQL persistence initialized successfully")

	return postgres, nil
}

// SaveSchedule saves or updates a schedule in the database.
func (p *PostgresPersistence) SaveSchedule(schedule *schedulerModels.Schedule) error {
	ctx := context.Background()

	query := `
		INSERT INTO scheduler_schedules (
			id, source_id, cron_expression, next_due_at, created_at, updated_at, active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) 
		DO UPDATE SET
			source_id = EXCLUDED.source_id,
			cron_expression = EXCLUDED.cron_expression,
			next_due_at = EXCLUDED.next_due_at,
			updated_at = EXCLUDED.updated_at,
			active = EXCLUDED.active
	`

	now := time.Now().UTC()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}

	schedule.UpdatedAt = now

	_, err := p.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.SourceID,
		schedule.CronExpression,
		schedule.NextDueAt,
		schedule.CreatedAt,
		schedule.UpdatedAt,
		schedule.Active,
	)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to save schedule", "schedule_id", schedule.ID, "error", err)

		return fmt.Errorf("failed to save schedule: %w", err)
	}

	p.logger.DebugContext(ctx, "Schedule saved successfully", "schedule_id", schedule.ID, "source_id", schedule.SourceID)

	return nil
}

// ScheduleByID retrieves a schedule by its ID.
func (p *PostgresPersistence) ScheduleByID(id string) (*schedulerModels.Schedule, error) {
	ctx := context.Background()

	query := `
		SELECT id, source_id, cron_expression, next_due_at, created_at, updated_at, active
		FROM scheduler_schedules 
		WHERE id = $1
	`

	row := p.db.QueryRowContext(ctx, query, id)

	schedule := &schedulerModels.Schedule{}

	err := row.Scan(
		&schedule.ID,
		&schedule.SourceID,
		&schedule.CronExpression,
		&schedule.NextDueAt,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
		&schedule.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Schedule not found
		}

		p.logger.ErrorContext(ctx, "Failed to scan schedule", "schedule_id", id, "error", err)

		return nil, fmt.Errorf("failed to scan schedule: %w", err)
	}

	p.logger.DebugContext(ctx, "Schedule retrieved successfully", "schedule_id", id)

	return schedule, nil
}

// ScheduleBySourceID retrieves a schedule by its source ID.
func (p *PostgresPersistence) ScheduleBySourceID(sourceID string) (*schedulerModels.Schedule, error) {
	ctx := context.Background()

	query := `
		SELECT id, source_id, cron_expression, next_due_at, created_at, updated_at, active
		FROM scheduler_schedules 
		WHERE source_id = $1
		LIMIT 1
	`

	row := p.db.QueryRowContext(ctx, query, sourceID)

	schedule := &schedulerModels.Schedule{}

	err := row.Scan(
		&schedule.ID,
		&schedule.SourceID,
		&schedule.CronExpression,
		&schedule.NextDueAt,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
		&schedule.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Schedule not found
		}

		p.logger.ErrorContext(ctx, "Failed to scan schedule by source ID", "source_id", sourceID, "error", err)

		return nil, fmt.Errorf("failed to scan schedule by source ID: %w", err)
	}

	p.logger.DebugContext(ctx, "Schedule retrieved by source ID", "source_id", sourceID)

	return schedule, nil
}

// Schedules retrieves all schedules from the database.
func (p *PostgresPersistence) Schedules() ([]*schedulerModels.Schedule, error) {
	ctx := context.Background()

	query := `
		SELECT id, source_id, cron_expression, next_due_at, created_at, updated_at, active
		FROM scheduler_schedules 
		ORDER BY created_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query all schedules", "error", err)

		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	schedules, err := p.scanScheduleRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "All schedules retrieved", "count", len(schedules))

	return schedules, nil
}

// DueSchedules retrieves all schedules that are due before the specified time.
// This is the most critical method for scheduler performance.
func (p *PostgresPersistence) DueSchedules(before time.Time) ([]*schedulerModels.Schedule, error) {
	ctx := context.Background()

	query := `
		SELECT id, source_id, cron_expression, next_due_at, created_at, updated_at, active
		FROM scheduler_schedules 
		WHERE active = true AND next_due_at <= $1
		ORDER BY next_due_at ASC
	`

	rows, err := p.db.QueryContext(ctx, query, before)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to query due schedules", "before", before, "error", err)

		return nil, fmt.Errorf("failed to query due schedules: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			p.logger.ErrorContext(ctx, "Failed to close rows", "error", closeErr)
		}
	}()

	schedules, err := p.scanScheduleRows(ctx, rows)
	if err != nil {
		return nil, err
	}

	p.logger.DebugContext(ctx, "Due schedules retrieved", "before", before, "count", len(schedules))

	return schedules, nil
}

// DeleteSchedule deletes a schedule from the database.
func (p *PostgresPersistence) DeleteSchedule(id string) error {
	ctx := context.Background()

	query := `DELETE FROM scheduler_schedules WHERE id = $1`

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to delete schedule", "schedule_id", id, "error", err)

		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.logger.DebugContext(ctx, "Schedule deletion completed", "schedule_id", id, "rows_affected", rowsAffected)

	return nil
}

// DeleteScheduleBySourceID deletes a schedule by its source ID.
func (p *PostgresPersistence) DeleteScheduleBySourceID(sourceID string) error {
	ctx := context.Background()

	query := `DELETE FROM scheduler_schedules WHERE source_id = $1`

	result, err := p.db.ExecContext(ctx, query, sourceID)
	if err != nil {
		p.logger.ErrorContext(ctx, "Failed to delete schedule by source ID", "source_id", sourceID, "error", err)

		return fmt.Errorf("failed to delete schedule by source ID: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	p.logger.DebugContext(ctx, "Schedule deletion by source ID completed", "source_id", sourceID, "rows_affected", rowsAffected)

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

	err = p.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scheduler_schedules").Scan(&count)
	if err != nil {
		p.logger.ErrorContext(ctx, "Database table query failed", "error", err)

		return fmt.Errorf("database table query failed: %w", err)
	}

	p.logger.DebugContext(ctx, "Database health check passed", "scheduler_schedules_count", count)

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

// scanScheduleRows scans database rows into Schedule structs to reduce code duplication.
func (p *PostgresPersistence) scanScheduleRows(ctx context.Context, rows *sql.Rows) ([]*schedulerModels.Schedule, error) {
	var schedules []*schedulerModels.Schedule

	for rows.Next() {
		schedule := &schedulerModels.Schedule{}

		err := rows.Scan(
			&schedule.ID,
			&schedule.SourceID,
			&schedule.CronExpression,
			&schedule.NextDueAt,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&schedule.Active,
		)
		if err != nil {
			p.logger.ErrorContext(ctx, "Failed to scan schedule row", "error", err)

			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedule rows: %w", err)
	}

	return schedules, nil
}

// schedulerMigrations returns the migration scripts for Scheduler-specific tables.
func schedulerMigrations() map[int]string {
	return map[int]string{
		3: `
			-- Create scheduler_schedules table for Scheduler provider persistence
			CREATE TABLE scheduler_schedules (
				id VARCHAR(255) PRIMARY KEY,
				source_id VARCHAR(255) NOT NULL,
				cron_expression VARCHAR(255) NOT NULL,
				next_due_at TIMESTAMP WITH TIME ZONE NOT NULL,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				active BOOLEAN NOT NULL DEFAULT true
			);

			-- Create indexes for better query performance
			CREATE INDEX idx_scheduler_schedules_source_id ON scheduler_schedules(source_id);
			CREATE INDEX idx_scheduler_schedules_next_due_at ON scheduler_schedules(next_due_at);
			CREATE INDEX idx_scheduler_schedules_active ON scheduler_schedules(active);
			CREATE INDEX idx_scheduler_schedules_created_at ON scheduler_schedules(created_at);
			CREATE INDEX idx_scheduler_schedules_updated_at ON scheduler_schedules(updated_at);
			
			-- Index for efficient due schedule queries (most important query)
			CREATE INDEX idx_scheduler_schedules_active_due ON scheduler_schedules(active, next_due_at) WHERE active = true;
		`,
	}
}
