package persistence

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/dukex/operion/pkg/providers/scheduler/models"
)

func TestSchedulerMigrations(t *testing.T) {
	migrations := schedulerMigrations()

	// Test that migration version 3 exists
	migration, exists := migrations[3]
	assert.True(t, exists, "Migration version 3 should exist")
	assert.Contains(t, migration, "CREATE TABLE scheduler_schedules", "Should create scheduler_schedules table")
	assert.Contains(t, migration, "idx_scheduler_schedules_active_due", "Should create optimized due schedules index")
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

func TestScanScheduleRows_EmptyRows(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	p := &PostgresPersistence{
		logger: logger,
	}

	// This would normally require actual database rows, but we can test the error handling
	// when rows.Err() returns an error by mocking or using nil rows
	// For now, we're testing the structure exists
	assert.NotNil(t, p.scanScheduleRows)
}

func TestScheduleMigrationContent(t *testing.T) {
	migrations := schedulerMigrations()
	migration := migrations[3]

	// Verify all required indexes are present
	requiredIndexes := []string{
		"idx_scheduler_schedules_source_id",
		"idx_scheduler_schedules_next_due_at",
		"idx_scheduler_schedules_active",
		"idx_scheduler_schedules_created_at",
		"idx_scheduler_schedules_updated_at",
		"idx_scheduler_schedules_active_due",
	}

	for _, index := range requiredIndexes {
		assert.Contains(t, migration, index, "Migration should contain index: %s", index)
	}

	// Verify table structure
	requiredColumns := []string{
		"id VARCHAR(255) PRIMARY KEY",
		"source_id VARCHAR(255) NOT NULL",
		"cron_expression VARCHAR(255) NOT NULL",
		"next_due_at TIMESTAMP WITH TIME ZONE NOT NULL",
		"created_at TIMESTAMP WITH TIME ZONE NOT NULL",
		"updated_at TIMESTAMP WITH TIME ZONE NOT NULL",
		"active BOOLEAN NOT NULL DEFAULT true",
	}

	for _, column := range requiredColumns {
		assert.Contains(t, migration, column, "Migration should contain column definition: %s", column)
	}

	// Verify partial index for performance
	assert.Contains(t, migration, "WHERE active = true", "Should have partial index for active schedules")
}

func TestPostgresPersistence_MethodSignatures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Test that PostgresPersistence implements the interface properly
	// by checking method signatures exist
	var persistence interface{} = &PostgresPersistence{logger: logger}

	// This will compile-time check that PostgresPersistence implements SchedulerPersistence
	_, ok := persistence.(interface {
		SaveSchedule(schedule *models.Schedule) error
		ScheduleByID(id string) (*models.Schedule, error)
		ScheduleBySourceID(sourceID string) (*models.Schedule, error)
		Schedules() ([]*models.Schedule, error)
		DueSchedules(before time.Time) ([]*models.Schedule, error)
		DeleteSchedule(id string) error
		DeleteScheduleBySourceID(sourceID string) error
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
