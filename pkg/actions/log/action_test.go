package log_action

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogActionFactory(t *testing.T) {
	factory := NewLogActionFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "log", factory.ID())
}

func TestLogActionFactory_Create(t *testing.T) {
	factory := NewLogActionFactory()

	tests := []struct {
		name   string
		config map[string]interface{}
	}{
		{
			name:   "nil config",
			config: nil,
		},
		{
			name:   "empty config",
			config: map[string]interface{}{},
		},
		{
			name: "config with values",
			config: map[string]interface{}{
				"message": "test message",
				"level":   "info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := factory.Create(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, action)
			assert.IsType(t, &LogAction{}, action)
		})
	}
}

func TestNewLogAction(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
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
			config:        map[string]interface{}{},
			expectedMsg:   "",
			expectedLevel: "info",
		},
		{
			name: "config with message only",
			config: map[string]interface{}{
				"message": "test message",
			},
			expectedMsg:   "test message",
			expectedLevel: "info",
		},
		{
			name: "config with message and level",
			config: map[string]interface{}{
				"message": "debug message",
				"level":   "debug",
			},
			expectedMsg:   "debug message",
			expectedLevel: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := NewLogAction(tt.config)
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
		config        map[string]interface{}
		execCtx       models.ExecutionContext
		expectedMsg   string
		expectedLevel string
		expectError   bool
	}{
		{
			name: "simple message",
			config: map[string]interface{}{
				"message": "Hello, World!",
			},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]interface{}),
			},
			expectedMsg:   "Hello, World!",
			expectedLevel: "info",
			expectError:   false,
		},
		{
			name: "message with debug level",
			config: map[string]interface{}{
				"message": "Debug message",
				"level":   "debug",
			},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]interface{}),
			},
			expectedMsg:   "Debug message",
			expectedLevel: "debug",
			expectError:   false,
		},
		{
			name: "message with templating",
			config: map[string]interface{}{
				"message": "\"Processing workflow: \" & steps.step1.status",
				"level":   "info",
			},
			execCtx: models.ExecutionContext{
				ID:         "exec-123",
				WorkflowID: "workflow-456",
				StepResults: map[string]interface{}{
					"step1": map[string]interface{}{
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
			config: map[string]interface{}{},
			execCtx: models.ExecutionContext{
				StepResults: make(map[string]interface{}),
			},
			expectedMsg:   "",
			expectedLevel: "info",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := NewLogAction(tt.config)
			result, err := action.Execute(context.Background(), tt.execCtx, logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify result contains message and level
				resultMap, ok := result.(map[string]interface{})
				assert.True(t, ok)
				assert.Equal(t, tt.expectedMsg, resultMap["message"])
				assert.Equal(t, tt.expectedLevel, resultMap["level"])
			}
		})
	}
}

func TestLogAction_Execute_WithCancel(t *testing.T) {
	action := NewLogAction(map[string]interface{}{
		"message": "Test message",
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"test": "data",
		},
	}

	result, err := action.Execute(ctx, execCtx, logger)

	// Log action should complete even with cancelled context
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "Test message", resultMap["message"])
	assert.Equal(t, "info", resultMap["level"])
}

func TestLogAction_Execute_LargeStepResults(t *testing.T) {
	action := NewLogAction(map[string]interface{}{})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create large step results to test logging performance
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[string(rune('A'+i%26))+string(rune('a'+i%26))] = map[string]interface{}{
			"index": i,
			"value": "test data " + string(rune('0'+i%10)),
			"nested": map[string]interface{}{
				"level1": map[string]interface{}{
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

	result, err := action.Execute(context.Background(), execCtx, logger)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "", resultMap["message"])
	assert.Equal(t, "info", resultMap["level"])
}
