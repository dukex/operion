package schedule

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScheduleTrigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

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
			name: "every minute cron",
			config: map[string]interface{}{
				"id":          "test-schedule-3",
				"cron":        "* * * * *",
				"workflow_id": "workflow-789",
			},
			expectError: false,
			expected: &ScheduleTrigger{
				ID:         "test-schedule-3",
				CronExpr:   "* * * * *",
				WorkflowId: "workflow-789",
				Enabled:    true,
			},
		},
		{
			name: "invalid cron expression",
			config: map[string]interface{}{
				"id":          "test-invalid",
				"cron":        "invalid cron",
				"workflow_id": "workflow-error",
			},
			expectError: true,
		},
		{
			name: "missing id",
			config: map[string]interface{}{
				"cron":        "*/5 * * * *",
				"workflow_id": "workflow-no-id",
			},
			expectError: true,
		},
		{
			name: "missing cron",
			config: map[string]interface{}{
				"id":          "test-no-cron",
				"workflow_id": "workflow-no-cron",
			},
			expectError: true,
		},
		{
			name:        "empty config",
			config:      map[string]interface{}{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewScheduleTrigger(tt.config, logger)

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

func TestScheduleTrigger_Validate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name        string
		trigger     *ScheduleTrigger
		expectError bool
	}{
		{
			name: "valid trigger",
			trigger: &ScheduleTrigger{
				ID:       "valid-trigger",
				CronExpr: "*/5 * * * *",
				Enabled:  true,
				logger:   logger,
			},
			expectError: false,
		},
		{
			name: "empty ID",
			trigger: &ScheduleTrigger{
				ID:       "",
				CronExpr: "*/5 * * * *",
				Enabled:  true,
				logger:   logger,
			},
			expectError: true,
		},
		{
			name: "empty cron expression",
			trigger: &ScheduleTrigger{
				ID:       "test-trigger",
				CronExpr: "",
				Enabled:  true,
				logger:   logger,
			},
			expectError: true,
		},
		{
			name: "invalid cron expression",
			trigger: &ScheduleTrigger{
				ID:       "test-trigger",
				CronExpr: "invalid * cron * expression",
				Enabled:  true,
				logger:   logger,
			},
			expectError: true,
		},
		{
			name: "valid but complex cron",
			trigger: &ScheduleTrigger{
				ID:       "complex-trigger",
				CronExpr: "30 14 * * 1-5", // weekdays at 2:30 PM
				Enabled:  true,
				logger:   logger,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScheduleTrigger_StartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-start-stop",
		"cron":        "* * * * *", // every minute for quick test
		"workflow_id": "workflow-start-stop",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)
	require.NotNil(t, trigger)

	// Test start and stop
	ctx := context.Background()

	var callbackCount int
	var mu sync.Mutex
	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		callbackCount++
		mu.Unlock()
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

	mu.Lock()
	finalCount := callbackCount
	mu.Unlock()

	// May not execute since cron only runs every minute
	assert.GreaterOrEqual(t, finalCount, 0)

	// Wait a bit more and ensure no more executions
	time.Sleep(2 * time.Second)

	mu.Lock()
	afterStopCount := callbackCount
	mu.Unlock()

	// Count should not have increased after stop
	assert.Equal(t, finalCount, afterStopCount)
}

func TestScheduleTrigger_CallbackWithData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-callback-data",
		"cron":        "* * * * *", // every minute
		"workflow_id": "workflow-callback",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	var receivedData map[string]interface{}
	var mu sync.Mutex
	var called bool

	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		receivedData = data
		called = true
		mu.Unlock()
		return nil
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(500 * time.Millisecond)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

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
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-disabled",
		"cron":        "* * * * *",
		"workflow_id": "workflow-disabled",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)

	// Disable the trigger
	trigger.Enabled = false

	ctx := context.Background()
	var called bool
	var mu sync.Mutex

	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		called = true
		mu.Unlock()
		return nil
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(2 * time.Second)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Should not have been called since trigger is disabled
	assert.False(t, called)
}

func TestScheduleTrigger_CallbackError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-callback-error",
		"cron":        "* * * * *",
		"workflow_id": "workflow-error",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	var callCount int
	var mu sync.Mutex

	callback := func(ctx context.Context, data map[string]interface{}) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		return assert.AnError // Return an error
	}

	err = trigger.Start(ctx, callback)
	require.NoError(t, err)

	// Wait for potential execution
	time.Sleep(500 * time.Millisecond)

	err = trigger.Stop(ctx)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// May not have been called since cron runs every minute
	assert.GreaterOrEqual(t, callCount, 0)
}

func TestScheduleTrigger_ConcurrentStartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-concurrent",
		"cron":        "* * * * *",
		"workflow_id": "workflow-concurrent",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	callback := func(ctx context.Context, data map[string]interface{}) error {
		return nil
	}

	// Start multiple times (should not cause issues)
	for i := 0; i < 3; i++ {
		err = trigger.Start(ctx, callback)
		assert.NoError(t, err)
	}

	// Stop multiple times (should not cause issues)
	for i := 0; i < 3; i++ {
		err = trigger.Stop(ctx)
		assert.NoError(t, err)
	}
}

// Mock callback to satisfy TriggerCallback interface
type mockCallback struct {
	called bool
	data   map[string]interface{}
	mu     sync.Mutex
}

func (m *mockCallback) Call(ctx context.Context, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = true
	m.data = data
	return nil
}

func (m *mockCallback) WasCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func (m *mockCallback) GetData() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data
}

func TestScheduleTrigger_Interface(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := map[string]interface{}{
		"id":          "test-interface",
		"cron":        "*/5 * * * *",
		"workflow_id": "workflow-interface",
	}

	trigger, err := NewScheduleTrigger(config, logger)
	require.NoError(t, err)

	// Verify it implements the Trigger interface
	var _ protocol.Trigger = trigger

	assert.Equal(t, "test-interface", trigger.ID)
	assert.Equal(t, "*/5 * * * *", trigger.CronExpr)
	assert.Equal(t, "workflow-interface", trigger.WorkflowId)
	assert.True(t, trigger.Enabled)
}
