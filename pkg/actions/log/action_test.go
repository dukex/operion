package log_action

import (
	"context"
	"fmt"
	"testing"

	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogAction(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *LogAction
	}{
		{
			name: "basic log action",
			config: map[string]interface{}{
				"id":      "test-log-1",
				"message": "Hello, World!",
				"level":   "info",
			},
			expected: &LogAction{
				ID:      "test-log-1",
				Message: "Hello, World!",
				Level:   "info",
			},
		},
		{
			name: "log action without level",
			config: map[string]interface{}{
				"id":      "test-log-2",
				"message": "Debug message",
			},
			expected: &LogAction{
				ID:      "test-log-2",
				Message: "Debug message",
				Level:   "",
			},
		},
		{
			name:   "log action with empty config",
			config: map[string]interface{}{},
			expected: &LogAction{
				ID:      "",
				Message: "",
				Level:   "",
			},
		},
		{
			name: "log action with error level",
			config: map[string]interface{}{
				"id":      "test-log-3",
				"message": "An error occurred",
				"level":   "error",
			},
			expected: &LogAction{
				ID:      "test-log-3",
				Message: "An error occurred",
				Level:   "error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewLogAction(tt.config)

			require.NoError(t, err)
			assert.Equal(t, tt.expected.ID, action.ID)
			assert.Equal(t, tt.expected.Message, action.Message)
			assert.Equal(t, tt.expected.Level, action.Level)
		})
	}
}

func TestLogAction_GetMethods(t *testing.T) {
	action := &LogAction{
		ID:      "test-log",
		Message: "Test message",
		Level:   "info",
	}

	assert.Equal(t, "test-log", action.GetID())
	assert.Equal(t, "log", action.GetType())

	config := action.GetConfig()
	assert.Equal(t, "test-log", config["id"])
	assert.Equal(t, "Test message", config["message"])
	assert.Equal(t, "info", config["level"])

	assert.NoError(t, action.Validate())
}

func TestLogAction_Execute(t *testing.T) {
	tests := []struct {
		name     string
		action   *LogAction
		expected map[string]interface{}
	}{
		{
			name: "info level log",
			action: &LogAction{
				ID:      "test-info",
				Message: "Information message",
				Level:   "info",
			},
			expected: map[string]interface{}{
				"logged_message": "[info] Information message",
				"level":          "info",
			},
		},
		{
			name: "error level log",
			action: &LogAction{
				ID:      "test-error",
				Message: "Error message",
				Level:   "error",
			},
			expected: map[string]interface{}{
				"logged_message": "[error] Error message",
				"level":          "error",
			},
		},
		{
			name: "debug level log",
			action: &LogAction{
				ID:      "test-debug",
				Message: "Debug information",
				Level:   "debug",
			},
			expected: map[string]interface{}{
				"logged_message": "[debug] Debug information",
				"level":          "debug",
			},
		},
		{
			name: "warn level log",
			action: &LogAction{
				ID:      "test-warn",
				Message: "Warning message",
				Level:   "warn",
			},
			expected: map[string]interface{}{
				"logged_message": "[warn] Warning message",
				"level":          "warn",
			},
		},
		{
			name: "no level specified",
			action: &LogAction{
				ID:      "test-no-level",
				Message: "Message without level",
				Level:   "",
			},
			expected: map[string]interface{}{
				"logged_message": "[] Message without level",
				"level":          "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := log.WithField("test", "log_action")
			execCtx := models.ExecutionContext{Logger: logger}

			result, err := tt.action.Execute(context.Background(), execCtx)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogAction_Execute_WithDifferentContexts(t *testing.T) {
	action := &LogAction{
		ID:      "context-test",
		Message: "Context test message",
		Level:   "info",
	}

	// Test with different execution contexts
	contexts := []models.ExecutionContext{
		{Logger: log.WithField("workflow", "test-1")},
		{Logger: log.WithField("workflow", "test-2")},
	}

	for i, execCtx := range contexts {
		t.Run(fmt.Sprintf("context_%d", i), func(t *testing.T) {
			result, err := action.Execute(context.Background(), execCtx)

			require.NoError(t, err)
			expected := map[string]interface{}{
				"logged_message": "[info] Context test message",
				"level":          "info",
			}
			assert.Equal(t, expected, result)
		})
	}
}

func TestLogAction_Execute_WithCancelledContext(t *testing.T) {
	action := &LogAction{
		ID:      "cancelled-test",
		Message: "This should still work",
		Level:   "info",
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	logger := log.WithField("test", "log_action")
	execCtx := models.ExecutionContext{Logger: logger}

	// Log action should still work even with cancelled context
	result, err := action.Execute(ctx, execCtx)

	require.NoError(t, err)
	expected := map[string]interface{}{
		"logged_message": "[info] This should still work",
		"level":          "info",
	}
	assert.Equal(t, expected, result)
}

func TestLogAction_GetConfig_Consistency(t *testing.T) {
	config := map[string]interface{}{
		"id":      "config-test",
		"message": "Original message",
		"level":   "warn",
	}

	action, err := NewLogAction(config)
	require.NoError(t, err)

	retrievedConfig := action.GetConfig()

	// Config should match the original action properties
	assert.Equal(t, action.ID, retrievedConfig["id"])
	assert.Equal(t, action.Message, retrievedConfig["message"])
	assert.Equal(t, action.Level, retrievedConfig["level"])
}
