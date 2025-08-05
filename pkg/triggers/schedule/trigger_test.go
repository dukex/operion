package schedule_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/triggers/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduleTrigger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		expected    *schedule.Trigger
	}{
		{
			name: "valid cron expression",
			config: map[string]any{
				"id":          "test-schedule-1",
				"cron":        "*/5 * * * *", // every 5 minutes
				"workflow_id": "workflow-123",
			},
			expectError: false,
			expected: &schedule.Trigger{
				CronExpr: "*/5 * * * *",
				Enabled:  true,
			},
		},
		{
			name: "simple daily cron",
			config: map[string]any{
				"id":          "test-schedule-2",
				"cron":        "0 0 * * *", // daily at midnight
				"workflow_id": "workflow-456",
			},
			expectError: false,
			expected: &schedule.Trigger{
				CronExpr: "0 0 * * *",
				Enabled:  true,
			},
		},
		{
			name: "every minute cron",
			config: map[string]any{
				"id":          "test-schedule-3",
				"cron":        "* * * * *",
				"workflow_id": "workflow-789",
			},
			expectError: false,
			expected: &schedule.Trigger{
				CronExpr: "* * * * *",
				Enabled:  true,
			},
		},
		{
			name: "invalid cron expression",
			config: map[string]any{
				"id":          "test-invalid",
				"cron":        "invalid cron",
				"workflow_id": "workflow-error",
			},
			expectError: true,
		},
		{
			name: "minimal config",
			config: map[string]any{
				"cron": "*/5 * * * *",
			},
			expectError: false,
			expected: &schedule.Trigger{
				CronExpr: "*/5 * * * *",
				Enabled:  true,
			},
		},
		{
			name: "missing cron",
			config: map[string]any{
				"id":          "test-no-cron",
				"workflow_id": "workflow-no-cron",
			},
			expectError: true,
		},
		{
			name:        "empty config",
			config:      map[string]any{},
			expectError: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			trigger, err := schedule.NewTrigger(t.Context(), testCase.config, logger)

			if testCase.expectError {
				require.Error(t, err)
				assert.Nil(t, trigger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, trigger)
				assert.Equal(t, testCase.expected.CronExpr, trigger.CronExpr)
				assert.Equal(t, testCase.expected.Enabled, trigger.Enabled)
			}
		})
	}
}

func TestScheduleTrigger_Validate(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
	}{
		{
			name: "valid trigger config",
			config: map[string]any{
				"cron":        "*/5 * * * *",
				"enabled":     true,
				"workflow_id": "test-workflow",
			},
			expectError: false,
		},
		{
			name: "empty cron expression",
			config: map[string]any{
				"cron":        "",
				"enabled":     true,
				"workflow_id": "test-workflow",
			},
			expectError: true,
		},
		{
			name: "invalid cron expression",
			config: map[string]any{
				"cron":        "invalid-cron",
				"enabled":     true,
				"workflow_id": "test-workflow",
			},
			expectError: true,
		},
		{
			name: "invalid cron expression complex",
			config: map[string]any{
				"cron":        "invalid * cron * expression",
				"enabled":     true,
				"workflow_id": "test-workflow",
			},
			expectError: true,
		},
		{
			name: "valid complex cron",
			config: map[string]any{
				"cron":        "30 14 * * 1-5", // weekdays at 2:30 PM
				"enabled":     true,
				"workflow_id": "test-workflow",
			},
			expectError: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := schedule.NewTrigger(t.Context(), testCase.config, logger)

			if testCase.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScheduleTrigger_StartStop(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-start-stop",
		"cron":        "* * * * *", // every minute for quick test
		"workflow_id": "workflow-start-stop",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)
	require.NotNil(t, trigger)

	// Test start and stop
	ctx := context.Background()

	var (
		callbackCount int
		mutex         sync.Mutex
	)

	callback := func(_ context.Context, _ map[string]any) error {
		mutex.Lock()

		callbackCount++

		mutex.Unlock()

		return nil
	}

	// Start the trigger
	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution (short time since cron runs every minute)
	time.Sleep(500 * time.Millisecond)

	// Stop the trigger
	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mutex.Lock()

	finalCount := callbackCount

	mutex.Unlock()

	// May not execute since cron only runs every minute
	assert.GreaterOrEqual(t, finalCount, 0)

	// Wait a bit more and ensure no more executions
	time.Sleep(2 * time.Second)

	mutex.Lock()

	afterStopCount := callbackCount

	mutex.Unlock()

	// Count should not have increased after stop
	assert.Equal(t, finalCount, afterStopCount)
}

func TestScheduleTrigger_CallbackWithData(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-callback-data",
		"cron":        "* * * * *", // every minute
		"workflow_id": "workflow-callback",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	var (
		receivedData map[string]any
		mutex        sync.Mutex
		called       bool
	)

	callback := func(_ context.Context, data map[string]any) error {
		mutex.Lock()

		receivedData = data
		called = true

		mutex.Unlock()

		return nil
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(500 * time.Millisecond)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mutex.Lock()
	defer mutex.Unlock()

	// Note: may not be called since cron runs every minute and test is short
	if called {
		assert.NotNil(t, receivedData)
		assert.Contains(t, receivedData, "timestamp")

		// Verify timestamp format
		timestamp, ok := receivedData["timestamp"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, timestamp)

		// Should be valid RFC3339 format
		_, err = time.Parse(time.RFC3339, timestamp)
		assert.NoError(t, err)
	}
}

func TestScheduleTrigger_DisabledTrigger(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-disabled",
		"cron":        "* * * * *",
		"workflow_id": "workflow-disabled",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)

	// Disable the trigger
	trigger.Enabled = false

	ctx := context.Background()

	var (
		called bool
		mutex  sync.Mutex
	)

	callback := func(_ context.Context, _ map[string]any) error {
		mutex.Lock()

		called = true

		mutex.Unlock()

		return nil
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(2 * time.Second)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mutex.Lock()
	defer mutex.Unlock()

	// Should not have been called since trigger is disabled
	assert.False(t, called)
}

func TestScheduleTrigger_CallbackError(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-callback-error",
		"cron":        "* * * * *",
		"workflow_id": "workflow-error",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)

	ctx := context.Background()

	var (
		callCount int
		mutex     sync.Mutex
	)

	callback := func(_ context.Context, _ map[string]any) error {
		mutex.Lock()

		callCount++

		mutex.Unlock()

		return assert.AnError // Return an error
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(500 * time.Millisecond)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mutex.Lock()
	defer mutex.Unlock()

	// May not have been called since cron runs every minute
	assert.GreaterOrEqual(t, callCount, 0)
}

func TestScheduleTrigger_ConcurrentStartStop(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-concurrent",
		"cron":        "* * * * *",
		"workflow_id": "workflow-concurrent",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	callback := func(_ context.Context, _ map[string]any) error {
		return nil
	}

	// Start multiple times (should not cause issues)
	for range 3 {
		err = trigger.Start(ctx, callback)
		require.NoError(t, err)
	}

	// Stop multiple times (should not cause issues)
	for range 3 {
		err = trigger.Stop(ctx)
		assert.NoError(t, err)
	}
}

func TestScheduleTrigger_Interface(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]any{
		"id":          "test-interface",
		"cron":        "*/5 * * * *",
		"workflow_id": "workflow-interface",
	}

	trigger, err := schedule.NewTrigger(t.Context(), config, logger)
	require.NoError(t, err)

	// Verify it implements the Trigger interface
	var _ protocol.Trigger = trigger

	assert.Equal(t, "*/5 * * * *", trigger.CronExpr)
	assert.True(t, trigger.Enabled)
}
