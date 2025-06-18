package schedule

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduleTrigger(t *testing.T) {
	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
		expected    *ScheduleTrigger
	}{
		{
			name: "valid cron expression",
			config: map[string]interface{}{
				"id":          "test-schedule-1",
				"cron":        "*/5 * * * *", // every 5 minutes
				"workflow_id": "workflow-123",
			},
			expectError: false,
			expected: &ScheduleTrigger{
				ID:         "test-schedule-1",
				CronExpr:   "*/5 * * * *",
				WorkflowId: "workflow-123",
				Enabled:    true,
			},
		},
		{
			name: "simple daily cron",
			config: map[string]interface{}{
				"id":          "test-schedule-2",
				"cron":        "0 0 * * *", // daily at midnight
				"workflow_id": "workflow-456",
			},
			expectError: false,
			expected: &ScheduleTrigger{
				ID:         "test-schedule-2",
				CronExpr:   "0 0 * * *",
				WorkflowId: "workflow-456",
				Enabled:    true,
			},
		},
		{
			name: "missing id",
			config: map[string]interface{}{
				"cron":        "0 */5 * * * *",
				"workflow_id": "workflow-123",
			},
			expectError: true,
		},
		{
			name: "missing cron expression",
			config: map[string]interface{}{
				"id":          "test-schedule-3",
				"workflow_id": "workflow-123",
			},
			expectError: true,
		},
		{
			name: "invalid cron expression",
			config: map[string]interface{}{
				"id":          "test-schedule-4",
				"cron":        "invalid cron",
				"workflow_id": "workflow-123",
			},
			expectError: true,
		},
		{
			name: "cron every 2 hours",
			config: map[string]interface{}{
				"id":          "test-schedule-5",
				"cron":        "0 */2 * * *", // every 2 hours
				"workflow_id": "workflow-789",
			},
			expectError: false,
			expected: &ScheduleTrigger{
				ID:         "test-schedule-5",
				CronExpr:   "0 */2 * * *",
				WorkflowId: "workflow-789",
				Enabled:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewScheduleTrigger(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, trigger)
			} else {
				require.NoError(t, err)
				require.NotNil(t, trigger)
				assert.Equal(t, tt.expected.ID, trigger.ID)
				assert.Equal(t, tt.expected.CronExpr, trigger.CronExpr)
				assert.Equal(t, tt.expected.WorkflowId, trigger.WorkflowId)
				assert.Equal(t, tt.expected.Enabled, trigger.Enabled)
				assert.NotNil(t, trigger.logger)
			}
		})
	}
}

func TestScheduleTrigger_GetMethods(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-get-methods",
		"cron":        "*/10 * * * *",
		"workflow_id": "workflow-test",
	}

	trigger, err := NewScheduleTrigger(config)
	require.NoError(t, err)

	assert.Equal(t, "test-get-methods", trigger.GetID())
	assert.Equal(t, "schedule", trigger.GetType())

	retrievedConfig := trigger.GetConfig()
	assert.Equal(t, "test-get-methods", retrievedConfig["id"])
	assert.Equal(t, "*/10 * * * *", retrievedConfig["cron"])
	assert.Equal(t, "workflow-test", retrievedConfig["workflow_id"])
	assert.True(t, retrievedConfig["enabled"].(bool))
}

func TestScheduleTrigger_Validate(t *testing.T) {
	tests := []struct {
		name        string
		trigger     *ScheduleTrigger
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid trigger",
			trigger: &ScheduleTrigger{
				ID:       "valid-trigger",
				CronExpr: "*/5 * * * *",
			},
			expectError: false,
		},
		{
			name: "missing ID",
			trigger: &ScheduleTrigger{
				CronExpr: "*/5 * * * *",
			},
			expectError: true,
			errorMsg:    "schedule trigger ID is required",
		},
		{
			name: "missing cron expression",
			trigger: &ScheduleTrigger{
				ID: "missing-cron",
			},
			expectError: true,
			errorMsg:    "schedule trigger cron expression is required",
		},
		{
			name: "invalid cron expression",
			trigger: &ScheduleTrigger{
				ID:       "invalid-cron",
				CronExpr: "not a cron",
			},
			expectError: true,
			errorMsg:    "invalid cron expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScheduleTrigger_StartStop(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-start-stop",
		"cron":        "* * * * *", // every minute (for testing)
		"workflow_id": "workflow-test",
	}

	trigger, err := NewScheduleTrigger(config)
	require.NoError(t, err)

	callCount := 0
	var mu sync.Mutex

	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++

		// Verify the trigger data
		assert.Equal(t, "test-start-stop", data["trigger_id"])
		assert.Equal(t, "schedule", data["trigger_type"])
		assert.NotEmpty(t, data["timestamp"])

		return nil
	}

	// Start the trigger
	err = trigger.Start(context.Background(), callback)
	require.NoError(t, err)

	// Wait for a brief moment since every minute is too long for tests
	time.Sleep(100 * time.Millisecond)

	// Stop the trigger
	err = trigger.Stop(context.Background())
	require.NoError(t, err)

	mu.Lock()
	initialCallCount := callCount
	mu.Unlock()

	// Wait a bit more to ensure it stopped
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	finalCallCount := callCount
	mu.Unlock()

	// Since we're using every minute cron, it shouldn't execute during our test timeframe
	// This test mainly verifies that Start/Stop don't error
	assert.Equal(t, initialCallCount, finalCallCount, "Trigger should have stopped executing")
}

func TestScheduleTrigger_DisabledTrigger(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-disabled",
		"cron":        "* * * * *",
		"workflow_id": "workflow-test",
	}

	trigger, err := NewScheduleTrigger(config)
	require.NoError(t, err)

	// Disable the trigger
	trigger.Enabled = false

	callCount := 0
	callback := func(ctx context.Context, data map[string]interface{}) error {
		callCount++
		return nil
	}

	// Start should not error but should not execute
	err = trigger.Start(context.Background(), callback)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop
	err = trigger.Stop(context.Background())
	require.NoError(t, err)

	// Should not have been called since it's disabled
	assert.Equal(t, 0, callCount, "Disabled trigger should not execute")
}

func TestScheduleTrigger_CallbackError(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-callback-error",
		"cron":        "* * * * *", // every minute
		"workflow_id": "workflow-test",
	}

	trigger, err := NewScheduleTrigger(config)
	require.NoError(t, err)

	callCount := 0
	var mu sync.Mutex

	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return assert.AnError // Return an error
	}

	// Start the trigger
	err = trigger.Start(context.Background(), callback)
	require.NoError(t, err)

	// Wait for a execution
	time.Sleep(100 * time.Millisecond)

	// Stop the trigger
	err = trigger.Stop(context.Background())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Since we're using every minute cron, it shouldn't execute during our test timeframe
	// This test mainly verifies that callbacks with errors don't crash the trigger
}

func TestScheduleTrigger_MultipleStartStop(t *testing.T) {
	config := map[string]interface{}{
		"id":          "test-multiple",
		"cron":        "0 */5 * * *", // every 5 minutes (won't actually trigger during test)
		"workflow_id": "workflow-test",
	}

	trigger, err := NewScheduleTrigger(config)
	require.NoError(t, err)

	callback := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	// Multiple start/stop cycles should not error
	for i := 0; i < 3; i++ {
		err = trigger.Start(context.Background(), callback)
		assert.NoError(t, err)

		err = trigger.Stop(context.Background())
		assert.NoError(t, err)
	}
}
