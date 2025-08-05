package log_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	logaction "github.com/dukex/operion/pkg/actions/log"
	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActionFactory(t *testing.T) {
	t.Parallel()

	factory := logaction.NewActionFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "log", factory.ID())
}

func TestActionFactory_Create(t *testing.T) {
	t.Parallel()

	factory := logaction.NewActionFactory()

	tests := []struct {
		name   string
		config map[string]any
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "empty config",
			config: map[string]any{},
		},
		{
			name: "config with values",
			config: map[string]any{
				"message": "test message",
				"level":   "info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			action, err := factory.Create(t.Context(), tt.config)
			require.NoError(t, err)
			assert.NotNil(t, action)
			assert.IsType(t, &logaction.LogAction{Message: "test message", Level: "info"}, action)
		})
	}
}

func TestNewLogAction(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		expectedMsg   string
		expectedLevel string
	}{
		{
			name:          "nil config",
			config:        nil,
			expectedMsg:   "",
			expectedLevel: "info",
		},
		{
			name:          "empty config",
			config:        map[string]any{},
			expectedMsg:   "",
			expectedLevel: "info",
		},
		{
			name: "config with message only",
			config: map[string]any{
				"message": "test message",
			},
			expectedMsg:   "test message",
			expectedLevel: "info",
		},
		{
			name: "config with message and level",
			config: map[string]any{
				"message": "debug message",
				"level":   "debug",
			},
			expectedMsg:   "debug message",
			expectedLevel: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := logaction.NewLogAction(tt.config)
			assert.NotNil(t, action)
			assert.Equal(t, tt.expectedMsg, action.Message)
			assert.Equal(t, tt.expectedLevel, action.Level)
		})
	}
}

func TestLogAction_Execute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		config        map[string]any
		execCtx       models.ExecutionContext
		expectedMsg   string
		expectedLevel string
		expectError   bool
	}{
		{
			name: "simple message",
			config: map[string]any{
				"message": "Hello, World!",
			},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]any),
			},
			expectedMsg:   "Hello, World!",
			expectedLevel: "info",
			expectError:   false,
		},
		{
			name: "message with debug level",
			config: map[string]any{
				"message": "Debug message",
				"level":   "debug",
			},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]any),
			},
			expectedMsg:   "Debug message",
			expectedLevel: "debug",
			expectError:   false,
		},
		{
			name: "message with templating",
			config: map[string]any{
				"message": "Processing workflow: {{.step_results.step1.status}}",
				"level":   "info",
			},
			execCtx: models.ExecutionContext{
				ID:         "exec-123",
				WorkflowID: "workflow-456",
				StepResults: map[string]any{
					"step1": map[string]any{
						"status": "success",
					},
				},
			},
			expectedMsg:   "Processing workflow: success",
			expectedLevel: "info",
			expectError:   false,
		},
		{
			name:   "empty message",
			config: map[string]any{},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]any),
			},
			expectedMsg:   "",
			expectedLevel: "info",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := logaction.NewLogAction(tt.config)
			result, err := action.Execute(t.Context(), tt.execCtx, logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify result contains message and level
				resultMap, ok := result.(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, tt.expectedMsg, resultMap["message"])
				assert.Equal(t, tt.expectedLevel, resultMap["level"])
			}
		})
	}
}

func TestLogAction_Execute_WithCancel(t *testing.T) {
	action := logaction.NewLogAction(map[string]any{
		"message": "Test message",
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"test": "data",
		},
	}

	result, err := action.Execute(ctx, execCtx, logger)

	// Log action should complete even with cancelled context
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Equal(t, "Test message", resultMap["message"])
	assert.Equal(t, "info", resultMap["level"])
}

func TestLogAction_Execute_LargeStepResults(t *testing.T) {
	action := logaction.NewLogAction(map[string]any{})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create large step results to test logging performance
	largeData := make(map[string]any)
	for i := range 1000 {
		largeData[string(rune('A'+i%26))+string(rune('a'+i%26))] = map[string]any{
			"index": i,
			"value": "test data " + string(rune('0'+i%10)),
			"nested": map[string]any{
				"level1": map[string]any{
					"level2": "deep value",
				},
			},
		}
	}

	execCtx := models.ExecutionContext{
		ID:          "exec-large",
		WorkflowID:  "workflow-large",
		StepResults: largeData,
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Empty(t, resultMap["message"])
	assert.Equal(t, "info", resultMap["level"])
}
