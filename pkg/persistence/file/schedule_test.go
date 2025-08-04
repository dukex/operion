package file_test

import (
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePersistence_ScheduleOperations(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "operion-test-schedules")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	persistence := file.NewFilePersistence(tempDir)

	// Create a test schedule
	schedule, err := models.NewSchedule("test-schedule", "test-source", "0 0 * * *")
	require.NoError(t, err)

	// Test SaveSchedule
	err = persistence.SaveSchedule(schedule)
	assert.NoError(t, err)

	// Test ScheduleByID
	retrieved, err := persistence.ScheduleByID("test-schedule")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-schedule", retrieved.ID)
	assert.Equal(t, "test-source", retrieved.SourceID)
	assert.Equal(t, "0 0 * * *", retrieved.CronExpression)

	// Test ScheduleBySourceID
	retrieved, err = persistence.ScheduleBySourceID("test-source")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-schedule", retrieved.ID)

	// Test Schedules
	schedules, err := persistence.Schedules()
	assert.NoError(t, err)
	assert.Len(t, schedules, 1)

	// Test DueSchedules - set a schedule that should be due
	schedule.NextDueAt = time.Now().UTC().Add(-1 * time.Hour) // 1 hour ago
	err = persistence.SaveSchedule(schedule)
	assert.NoError(t, err)

	dueSchedules, err := persistence.DueSchedules(time.Now().UTC())
	assert.NoError(t, err)
	assert.Len(t, dueSchedules, 1)

	// Test DeleteScheduleBySourceID
	err = persistence.DeleteScheduleBySourceID("test-source")
	assert.NoError(t, err)

	// Verify deletion
	retrieved, err = persistence.ScheduleByID("test-schedule")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestFilePersistence_ScheduleBySourceID_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "operion-test-schedules")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	persistence := file.NewFilePersistence(tempDir)

	schedule, err := persistence.ScheduleBySourceID("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, schedule)
}

func TestFilePersistence_EmptySchedules(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "operion-test-schedules")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	persistence := file.NewFilePersistence(tempDir)

	schedules, err := persistence.Schedules()
	assert.NoError(t, err)
	assert.Len(t, schedules, 0)

	dueSchedules, err := persistence.DueSchedules(time.Now().UTC())
	assert.NoError(t, err)
	assert.Len(t, dueSchedules, 0)
}