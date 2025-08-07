package transform_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/actions/transform"
	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransformActionFactory(t *testing.T) {
	t.Parallel()

	factory := transform.NewActionFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "transform", factory.ID())
}

func TestTransformActionFactory_Create(t *testing.T) {
	t.Parallel()

	factory := transform.NewActionFactory()

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
			name: "config with expression",
			config: map[string]any{
				"expression": "$.name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			action, err := factory.Create(t.Context(), tt.config)
			require.NoError(t, err)
			assert.NotNil(t, action)
			assert.IsType(t, &transform.Action{Expression: ""}, action)
		})
	}
}

func TestNewTransformAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]any
		expected *transform.Action
	}{
		{
			name: "basic transform",
			config: map[string]any{
				"id":         "test-1",
				"expression": "{{ .field }}",
			},
			expected: &transform.Action{
				Expression: "{{ .field }}",
			},
		},
		{
			name:   "empty config",
			config: map[string]any{},
			expected: &transform.Action{
				Expression: "",
			},
		},
		{
			name: "partial config",
			config: map[string]any{
				"expression": "{ \"name\": {{ .name }}, \"age\": {{ .age }} }",
			},
			expected: &transform.Action{
				Expression: "{ \"name\": {{ .name }}, \"age\": {{ .age }} }",
			},
		},
	}

	for _, testingChild := range tests {
		t.Run(testingChild.name, func(t *testing.T) {
			t.Parallel()

			action, err := transform.NewAction(testingChild.config)
			require.NoError(t, err)
			assert.Equal(t, testingChild.expected, action)
		})
	}
}

func TestTransformAction_Execute_SimpleTransform(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: "{{.step_results.user.name}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"user": map[string]any{
				"name": "John Doe",
				"age":  30,
			},
		},
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result)
}

func TestTransformAction_Execute_ObjectConstruction(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: `{ "name": "{{.step_results.user.name}}", "status": "active", "age": {{.step_results.user.age}} }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"age":  25,
			},
		},
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)

	age, ok := resultMap["age"].(float64)
	require.True(t, ok)

	assert.Equal(t, "Alice", resultMap["name"])
	assert.Equal(t, "active", resultMap["status"])
	assert.InEpsilon(t, float64(25), age, 0.001) // JSON numbers are parsed as float64
}

func TestTransformAction_Execute_ArrayTransform(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: "{{index .step_results.users 0 \"name\"}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"users": []any{
				map[string]any{
					"name": "First User",
					"id":   1,
				},
				map[string]any{
					"name": "Second User",
					"id":   2,
				},
			},
		},
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, "First User", result)
}

func TestTransformAction_Execute_ComplexTransform(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: `{ "price": {{if .step_results.api_response.close}}
		{{.step_results.api_response.close}}
		{{else}}
		{{.step_results.api_response.open}}
		{{end}}, "currency": "USD", "timestamp": "{{.step_results.api_response.time}}" }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"api_response": map[string]any{
				"open":  45000.0,
				"close": 46000.0,
				"high":  47000.0,
				"low":   44000.0,
				"time":  "2023-10-01T10:00:00Z",
			},
		},
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)
	assert.True(t, ok)

	price, ok := resultMap["price"].(float64)
	require.True(t, ok)

	assert.InEpsilon(t, float64(46000), price, 0.001) // JSON numbers are parsed as float64
	assert.Equal(t, "USD", resultMap["currency"])
	assert.Equal(t, "2023-10-01T10:00:00Z", resultMap["timestamp"])
}

func TestTransformAction_Execute_EmptyExpression(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: "",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"data": "test",
		},
	}

	result, err := action.Execute(t.Context(), execCtx, logger)

	// Empty expression should return empty string
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestTransformAction_Execute_InvalidExpression(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: "{{.invalid..syntax}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"data": "test",
		},
	}

	_, err := action.Execute(t.Context(), execCtx, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transformation failed")
}

func TestTransformAction_Execute_WithCancel(t *testing.T) {
	t.Parallel()

	action := &transform.Action{
		Expression: "{{.step_results.data}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	execCtx := models.ExecutionContext{
		ID:          "test-exec-1",
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
		Variables:   map[string]any{},
		WorkflowID:  "test-workflow-1",
		StepResults: map[string]any{
			"data": "test value",
		},
	}

	result, err := action.Execute(ctx, execCtx, logger)

	// Transform action should complete even with cancelled context
	require.NoError(t, err)
	assert.Equal(t, "test value", result)
}
