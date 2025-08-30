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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/dukex/operion/pkg/providers/scheduler/models"
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
			postgres.WithDatabase("operion_scheduler_test"),
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

	_, err = db.ExecContext(ctx, "TRUNCATE TABLE scheduler_schedules")
	require.NoError(t, err)
}

func TestNewSchedulerPostgresPersistence(t *testing.T) {
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

func TestSchedulerPersistence_SaveAndRetrieveSchedule(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create test schedule
	schedule, err := models.NewSchedule("test-schedule", "source-123", "0 * * * *")
	require.NoError(t, err)

	// Save schedule
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	// Retrieve by ID
	retrievedSchedule, err := persistence.ScheduleByID("test-schedule")
	require.NoError(t, err)
	require.NotNil(t, retrievedSchedule)

	// Verify schedule data
	assert.Equal(t, schedule.ID, retrievedSchedule.ID)
	assert.Equal(t, schedule.SourceID, retrievedSchedule.SourceID)
	assert.Equal(t, schedule.CronExpression, retrievedSchedule.CronExpression)
	assert.Equal(t, schedule.Active, retrievedSchedule.Active)
	assert.True(t, !retrievedSchedule.CreatedAt.IsZero())
	assert.True(t, !retrievedSchedule.UpdatedAt.IsZero())

	// Test non-existent schedule
	nonExistent, err := persistence.ScheduleByID("non-existent")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestSchedulerPersistence_DueSchedules(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	now := time.Now().UTC()

	// Create due schedule (past due time)
	dueSchedule, err := models.NewSchedule("due-schedule", "source-1", "* * * * *")
	require.NoError(t, err)
	dueSchedule.NextDueAt = now.Add(-1 * time.Hour)
	err = persistence.SaveSchedule(dueSchedule)
	require.NoError(t, err)

	// Create future schedule
	futureSchedule, err := models.NewSchedule("future-schedule", "source-2", "* * * * *")
	require.NoError(t, err)
	futureSchedule.NextDueAt = now.Add(1 * time.Hour)
	err = persistence.SaveSchedule(futureSchedule)
	require.NoError(t, err)

	// Create inactive due schedule
	inactiveSchedule, err := models.NewSchedule("inactive-schedule", "source-3", "* * * * *")
	require.NoError(t, err)
	inactiveSchedule.NextDueAt = now.Add(-30 * time.Minute)
	inactiveSchedule.Active = false
	err = persistence.SaveSchedule(inactiveSchedule)
	require.NoError(t, err)

	// Query due schedules
	dueSchedules, err := persistence.DueSchedules(now)
	require.NoError(t, err)
	assert.Len(t, dueSchedules, 1)
	assert.Equal(t, "due-schedule", dueSchedules[0].ID)
}

func TestSchedulerPersistence_ScheduleBySourceID(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create schedule
	schedule, err := models.NewSchedule("test-schedule", "source-123", "0 9 * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	// Retrieve by source ID
	retrievedSchedule, err := persistence.ScheduleBySourceID("source-123")
	require.NoError(t, err)
	require.NotNil(t, retrievedSchedule)
	assert.Equal(t, "test-schedule", retrievedSchedule.ID)
	assert.Equal(t, "source-123", retrievedSchedule.SourceID)

	// Test non-existent source
	nonExistent, err := persistence.ScheduleBySourceID("non-existent-source")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestSchedulerPersistence_UpdateSchedule(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create and save initial schedule
	schedule, err := models.NewSchedule("update-schedule", "source-123", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	// Get original timestamps
	originalCreatedAt := schedule.CreatedAt
	originalUpdatedAt := schedule.UpdatedAt

	// Update schedule (simulate next due time calculation)
	time.Sleep(10 * time.Millisecond) // Ensure time difference
	err = schedule.UpdateNextDueAt()
	require.NoError(t, err)

	// Save updated schedule
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	// Retrieve and verify update
	retrievedSchedule, err := persistence.ScheduleByID("update-schedule")
	require.NoError(t, err)
	require.NotNil(t, retrievedSchedule)

	// Verify timestamps
	assert.Equal(t, originalCreatedAt.Unix(), retrievedSchedule.CreatedAt.Unix()) // CreatedAt should not change
	assert.True(t, retrievedSchedule.UpdatedAt.After(originalUpdatedAt))          // UpdatedAt should be newer
}

func TestSchedulerPersistence_DeleteSchedule(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create and save schedules
	schedule1, err := models.NewSchedule("schedule-1", "source-1", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule1)
	require.NoError(t, err)

	schedule2, err := models.NewSchedule("schedule-2", "source-2", "0 9 * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule2)
	require.NoError(t, err)

	// Verify both exist
	schedules, err := persistence.Schedules()
	require.NoError(t, err)
	assert.Len(t, schedules, 2)

	// Delete one schedule
	err = persistence.DeleteSchedule("schedule-1")
	require.NoError(t, err)

	// Verify deletion
	deletedSchedule, err := persistence.ScheduleByID("schedule-1")
	require.NoError(t, err)
	assert.Nil(t, deletedSchedule)

	// Verify other schedule still exists
	remainingSchedule, err := persistence.ScheduleByID("schedule-2")
	require.NoError(t, err)
	require.NotNil(t, remainingSchedule)
	assert.Equal(t, "schedule-2", remainingSchedule.ID)
}

func TestSchedulerPersistence_DeleteScheduleBySourceID(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Create schedule
	schedule, err := models.NewSchedule("source-schedule", "source-to-delete", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	// Verify exists
	retrievedSchedule, err := persistence.ScheduleBySourceID("source-to-delete")
	require.NoError(t, err)
	require.NotNil(t, retrievedSchedule)

	// Delete by source ID
	err = persistence.DeleteScheduleBySourceID("source-to-delete")
	require.NoError(t, err)

	// Verify deletion
	deletedSchedule, err := persistence.ScheduleBySourceID("source-to-delete")
	require.NoError(t, err)
	assert.Nil(t, deletedSchedule)
}

func TestSchedulerPersistence_Schedules(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Initially should be empty
	schedules, err := persistence.Schedules()
	require.NoError(t, err)
	assert.Len(t, schedules, 0)

	// Create multiple schedules
	schedule1, err := models.NewSchedule("schedule-1", "source-1", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule1)
	require.NoError(t, err)

	schedule2, err := models.NewSchedule("schedule-2", "source-2", "0 9 * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule2)
	require.NoError(t, err)

	schedule3, err := models.NewSchedule("schedule-3", "source-3", "0 0 * * 0")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule3)
	require.NoError(t, err)

	// Retrieve all schedules
	allSchedules, err := persistence.Schedules()
	require.NoError(t, err)
	assert.Len(t, allSchedules, 3)

	// Verify schedules are returned in order of creation
	assert.Equal(t, "schedule-1", allSchedules[0].ID)
	assert.Equal(t, "schedule-2", allSchedules[1].ID)
	assert.Equal(t, "schedule-3", allSchedules[2].ID)
}

func TestSchedulerPersistence_CronExpressionEdgeCases(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	testCases := []struct {
		name           string
		cronExpression string
		description    string
	}{
		{
			name:           "every minute",
			cronExpression: "* * * * *",
			description:    "Most frequent schedule",
		},
		{
			name:           "weekday business hours",
			cronExpression: "0 9-17 * * 1-5",
			description:    "Complex business hour schedule",
		},
		{
			name:           "monthly first day",
			cronExpression: "0 0 1 * *",
			description:    "Monthly schedule",
		},
		{
			name:           "yearly schedule",
			cronExpression: "0 0 1 1 *",
			description:    "Yearly schedule",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			scheduleID := "schedule-" + tc.name
			sourceID := "source-" + tc.name

			// Create schedule with complex cron expression
			schedule, err := models.NewSchedule(scheduleID, sourceID, tc.cronExpression)
			require.NoError(t, err)

			// Save and retrieve
			err = persistence.SaveSchedule(schedule)
			require.NoError(t, err)

			retrievedSchedule, err := persistence.ScheduleByID(scheduleID)
			require.NoError(t, err)
			require.NotNil(t, retrievedSchedule)

			// Verify cron expression preserved
			assert.Equal(t, tc.cronExpression, retrievedSchedule.CronExpression)
			assert.True(t, !retrievedSchedule.NextDueAt.IsZero())
		})
	}
}

func TestSchedulerPersistence_HealthCheck(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Test successful health check
	err := persistence.HealthCheck()
	assert.NoError(t, err)

	// Test health check with some data
	schedule, err := models.NewSchedule("health-test", "source-health", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule)
	require.NoError(t, err)

	err = persistence.HealthCheck()
	assert.NoError(t, err)
}

func TestSchedulerPersistence_PerformanceQueryOptimization(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	now := time.Now().UTC()

	// Create a mix of schedules - some due, some future, some inactive
	for i := 0; i < 100; i++ {
		scheduleID := fmt.Sprintf("schedule-%d", i)
		sourceID := fmt.Sprintf("source-%d", i)

		schedule, err := models.NewSchedule(scheduleID, sourceID, "* * * * *")
		require.NoError(t, err)

		// Vary the due times and active status
		if i%3 == 0 {
			schedule.NextDueAt = now.Add(-1 * time.Hour) // Due
		} else if i%3 == 1 {
			schedule.NextDueAt = now.Add(1 * time.Hour) // Future
		} else {
			schedule.NextDueAt = now.Add(-30 * time.Minute) // Due
		}

		if i%10 == 0 {
			schedule.Active = false // Some inactive
		}

		err = persistence.SaveSchedule(schedule)
		require.NoError(t, err)
	}

	// Measure query performance for due schedules
	start := time.Now()
	dueSchedules, err := persistence.DueSchedules(now)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Greater(t, len(dueSchedules), 0)

	// Performance should be well under 50ms for 100 schedules (PRP requirement: < 50ms for 1000+ schedules)
	assert.Less(t, duration, 50*time.Millisecond, "DueSchedules query should complete quickly")

	t.Logf("DueSchedules query completed in %v for %d total schedules, returned %d due schedules",
		duration, 100, len(dueSchedules))
}

func TestSchedulerPersistence_ConcurrentAccess(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	const numGoroutines = 10
	const schedulesPerGoroutine = 10

	// Test concurrent writes
	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer func() { done <- true }()

			for j := 0; j < schedulesPerGoroutine; j++ {
				scheduleID := fmt.Sprintf("concurrent-schedule-%d-%d", routineID, j)
				sourceID := fmt.Sprintf("concurrent-source-%d-%d", routineID, j)

				schedule, err := models.NewSchedule(scheduleID, sourceID, "0 * * * *")
				if err != nil {
					t.Errorf("Failed to create schedule: %v", err)
					return
				}

				err = persistence.SaveSchedule(schedule)
				if err != nil {
					t.Errorf("Failed to save schedule: %v", err)
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all schedules were saved
	allSchedules, err := persistence.Schedules()
	require.NoError(t, err)
	assert.Len(t, allSchedules, numGoroutines*schedulesPerGoroutine)
}

func TestSchedulerPersistence_ErrorHandling(t *testing.T) {
	persistence, _, databaseURL := setupTestDB(t)
	defer persistence.Close()
	defer cleanupDB(t, databaseURL)

	// Test saving schedule with duplicate ID
	schedule1, err := models.NewSchedule("duplicate-id", "source-1", "0 * * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule1)
	require.NoError(t, err)

	// Saving same ID should update, not error
	schedule2, err := models.NewSchedule("duplicate-id", "source-2", "0 9 * * *")
	require.NoError(t, err)
	err = persistence.SaveSchedule(schedule2)
	require.NoError(t, err)

	// Verify update worked
	retrieved, err := persistence.ScheduleByID("duplicate-id")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "source-2", retrieved.SourceID)
	assert.Equal(t, "0 9 * * *", retrieved.CronExpression)
}
