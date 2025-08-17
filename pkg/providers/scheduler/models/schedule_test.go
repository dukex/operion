package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Constructor Tests

func TestNewSchedule_ValidCronExpression(t *testing.T) {
	testCases := []struct {
		name           string
		id             string
		sourceID       string
		cronExpression string
	}{
		{
			name:           "every minute",
			id:             "sched-1",
			sourceID:       "source-123",
			cronExpression: "* * * * *",
		},
		{
			name:           "every 5 minutes",
			id:             "sched-2",
			sourceID:       "source-456",
			cronExpression: "*/5 * * * *",
		},
		{
			name:           "daily at midnight",
			id:             "sched-3",
			sourceID:       "source-789",
			cronExpression: "0 0 * * *",
		},
		{
			name:           "weekly on monday at 9am",
			id:             "sched-4",
			sourceID:       "source-abc",
			cronExpression: "0 9 * * 1",
		},
		{
			name:           "monthly on first day at noon",
			id:             "sched-5",
			sourceID:       "source-def",
			cronExpression: "0 12 1 * *",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			beforeTime := time.Now().UTC()
			schedule, err := NewSchedule(tc.id, tc.sourceID, tc.cronExpression)
			afterTime := time.Now().UTC()

			require.NoError(t, err)
			require.NotNil(t, schedule)

			// Verify basic fields
			assert.Equal(t, tc.id, schedule.ID)
			assert.Equal(t, tc.sourceID, schedule.SourceID)
			assert.Equal(t, tc.cronExpression, schedule.CronExpression)
			assert.True(t, schedule.Active)

			// Verify timestamps are reasonable
			assert.True(t, schedule.CreatedAt.After(beforeTime) || schedule.CreatedAt.Equal(beforeTime))
			assert.True(t, schedule.CreatedAt.Before(afterTime) || schedule.CreatedAt.Equal(afterTime))

			// UpdatedAt will be set by calculateNextDueAt, so it should be >= CreatedAt
			assert.True(t, schedule.UpdatedAt.After(schedule.CreatedAt) || schedule.UpdatedAt.Equal(schedule.CreatedAt))

			// Verify NextDueAt is in the future and reasonable
			assert.True(t, schedule.NextDueAt.After(beforeTime))

			// Set reasonable maximum based on cron type
			var maxExpected time.Time

			switch tc.cronExpression {
			case "0 9 * * 1": // weekly
				maxExpected = beforeTime.Add(8 * 24 * time.Hour) // within 8 days
			case "0 12 1 * *": // monthly
				maxExpected = beforeTime.Add(32 * 24 * time.Hour) // within 32 days
			default: // daily or more frequent
				maxExpected = beforeTime.Add(25 * time.Hour) // within 25 hours
			}

			assert.True(t, schedule.NextDueAt.Before(maxExpected),
				"NextDueAt %v should be before %v for cron %s",
				schedule.NextDueAt, maxExpected, tc.cronExpression)
		})
	}
}

func TestNewSchedule_InvalidCronExpression(t *testing.T) {
	testCases := []struct {
		name           string
		cronExpression string
	}{
		{
			name:           "empty expression",
			cronExpression: "",
		},
		{
			name:           "too few fields",
			cronExpression: "* *",
		},
		{
			name:           "too many fields",
			cronExpression: "* * * * * * *",
		},
		{
			name:           "invalid minute range",
			cronExpression: "60 * * * *",
		},
		{
			name:           "invalid hour range",
			cronExpression: "* 24 * * *",
		},
		{
			name:           "invalid day of month",
			cronExpression: "* * 32 * *",
		},
		{
			name:           "invalid month",
			cronExpression: "* * * 13 *",
		},
		{
			name:           "invalid day of week",
			cronExpression: "* * * * 8",
		},
		{
			name:           "malformed expression",
			cronExpression: "invalid cron",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schedule, err := NewSchedule("test-id", "source-123", tc.cronExpression)

			assert.Error(t, err)
			assert.Nil(t, schedule)
		})
	}
}

func TestNewSchedule_EmptySourceID(t *testing.T) {
	schedule, err := NewSchedule("test-id", "", "* * * * *")

	require.NoError(t, err) // Constructor doesn't validate, only cron parsing fails
	require.NotNil(t, schedule)

	// But validation should fail
	err = schedule.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSchedule, err)
}

func TestNewSchedule_EmptyID(t *testing.T) {
	schedule, err := NewSchedule("", "source-123", "* * * * *")

	require.NoError(t, err) // Constructor doesn't validate, only cron parsing fails
	require.NotNil(t, schedule)

	// But validation should fail
	err = schedule.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSchedule, err)
}

// Next Due Time Tests

func TestSchedule_UpdateNextDueAt_Success(t *testing.T) {
	// Create schedule with every-minute cron
	schedule, err := NewSchedule("test-id", "source-123", "* * * * *")
	require.NoError(t, err)

	originalUpdatedAt := schedule.UpdatedAt

	// Wait a small amount to ensure time difference
	time.Sleep(10 * time.Millisecond)

	err = schedule.UpdateNextDueAt()
	require.NoError(t, err)

	// UpdatedAt should definitely be updated
	assert.True(t, schedule.UpdatedAt.After(originalUpdatedAt))

	// For every-minute cron, NextDueAt should be within the next minute
	now := time.Now().UTC()
	maxExpected := now.Add(61 * time.Second) // Extra second for timing
	assert.True(t, schedule.NextDueAt.Before(maxExpected) || schedule.NextDueAt.Equal(maxExpected))
}

func TestSchedule_UpdateNextDueAt_InvalidCron(t *testing.T) {
	// Create schedule manually with invalid cron (bypass constructor validation)
	schedule := &Schedule{
		ID:             "test-id",
		SourceID:       "source-123",
		CronExpression: "invalid cron expression",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Active:         true,
	}

	err := schedule.UpdateNextDueAt()
	assert.Error(t, err)
}

// IsDue Tests

func TestSchedule_IsDue_ActiveAndDue(t *testing.T) {
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	schedule := &Schedule{
		ID:        "test-id",
		SourceID:  "source-123",
		NextDueAt: pastTime,
		Active:    true,
	}

	isDue := schedule.IsDue(time.Now().UTC())
	assert.True(t, isDue)
}

func TestSchedule_IsDue_ActiveButNotDue(t *testing.T) {
	futureTime := time.Now().UTC().Add(1 * time.Hour)
	schedule := &Schedule{
		ID:        "test-id",
		SourceID:  "source-123",
		NextDueAt: futureTime,
		Active:    true,
	}

	isDue := schedule.IsDue(time.Now().UTC())
	assert.False(t, isDue)
}

func TestSchedule_IsDue_InactiveAndDue(t *testing.T) {
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	schedule := &Schedule{
		ID:        "test-id",
		SourceID:  "source-123",
		NextDueAt: pastTime,
		Active:    false,
	}

	isDue := schedule.IsDue(time.Now().UTC())
	assert.False(t, isDue) // Inactive schedules are never due
}

func TestSchedule_IsDue_ExactTime(t *testing.T) {
	now := time.Now().UTC()
	schedule := &Schedule{
		ID:        "test-id",
		SourceID:  "source-123",
		NextDueAt: now,
		Active:    true,
	}

	isDue := schedule.IsDue(now)
	assert.True(t, isDue) // Exact time should be considered due
}

// Validation Tests

func TestSchedule_Validate_Success(t *testing.T) {
	schedule := &Schedule{
		ID:             "test-id",
		SourceID:       "source-123",
		CronExpression: "0 * * * *", // every hour
		NextDueAt:      time.Now().UTC().Add(1 * time.Hour),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Active:         true,
	}

	err := schedule.Validate()
	assert.NoError(t, err)
}

func TestSchedule_Validate_MissingID(t *testing.T) {
	schedule := &Schedule{
		ID:             "",
		SourceID:       "source-123",
		CronExpression: "0 * * * *",
		NextDueAt:      time.Now().UTC().Add(1 * time.Hour),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Active:         true,
	}

	err := schedule.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSchedule, err)
}

func TestSchedule_Validate_MissingSourceID(t *testing.T) {
	schedule := &Schedule{
		ID:             "test-id",
		SourceID:       "",
		CronExpression: "0 * * * *",
		NextDueAt:      time.Now().UTC().Add(1 * time.Hour),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Active:         true,
	}

	err := schedule.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSchedule, err)
}

func TestSchedule_Validate_MissingCronExpression(t *testing.T) {
	schedule := &Schedule{
		ID:             "test-id",
		SourceID:       "source-123",
		CronExpression: "",
		NextDueAt:      time.Now().UTC().Add(1 * time.Hour),
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
		Active:         true,
	}

	err := schedule.Validate()
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidSchedule, err)
}

func TestSchedule_Validate_InvalidCronExpression(t *testing.T) {
	testCases := []struct {
		name           string
		cronExpression string
	}{
		{
			name:           "malformed",
			cronExpression: "invalid cron",
		},
		{
			name:           "wrong number of fields",
			cronExpression: "* * * *", // missing day of week
		},
		{
			name:           "invalid range",
			cronExpression: "60 * * * *", // minute 60 doesn't exist
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			schedule := &Schedule{
				ID:             "test-id",
				SourceID:       "source-123",
				CronExpression: tc.cronExpression,
				NextDueAt:      time.Now().UTC().Add(1 * time.Hour),
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
				Active:         true,
			}

			err := schedule.Validate()
			assert.Error(t, err)
			// Error should be from cron parsing, not ErrInvalidSchedule
			assert.NotEqual(t, ErrInvalidSchedule, err)
		})
	}
}

// Edge Cases and Integration Tests

func TestSchedule_CronExpressionExamples(t *testing.T) {
	testCases := []struct {
		name           string
		cronExpression string
		expectedNext   func(now time.Time) time.Time
	}{
		{
			name:           "every minute",
			cronExpression: "* * * * *",
			expectedNext: func(now time.Time) time.Time {
				return now.Truncate(time.Minute).Add(time.Minute)
			},
		},
		{
			name:           "every hour",
			cronExpression: "0 * * * *",
			expectedNext: func(now time.Time) time.Time {
				next := now.Truncate(time.Hour).Add(time.Hour)

				return next
			},
		},
		{
			name:           "daily at midnight",
			cronExpression: "0 0 * * *",
			expectedNext: func(now time.Time) time.Time {
				next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)

				return next
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now().UTC()
			schedule, err := NewSchedule("test-id", "source-123", tc.cronExpression)
			require.NoError(t, err)

			expected := tc.expectedNext(now)

			// Allow some tolerance for timing differences
			tolerance := 2 * time.Second

			diff := schedule.NextDueAt.Sub(expected)
			if diff < 0 {
				diff = -diff
			}

			assert.True(t, diff <= tolerance,
				"NextDueAt %v should be within %v of expected %v (diff: %v) for cron %s",
				schedule.NextDueAt, tolerance, expected, diff, tc.cronExpression)
		})
	}
}
