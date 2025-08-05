package transform

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransformActionFactory(t *testing.T) {
	factory := NewTransformActionFactory()
	assert.NotNil(t, factory)
	assert.Equal(t, "transform", factory.ID())
}

func TestTransformActionFactory_Create(t *testing.T) {
	factory := NewTransformActionFactory()

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
			action, err := factory.Create(tt.config)
			require.NoError(t, err)
			assert.NotNil(t, action)
			assert.IsType(t, &TransformAction{}, action)
		})
	}
}

func TestNewTransformAction(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]any
		expected *TransformAction
	}{
		{
			name: "basic transform",
			config: map[string]any{
				"id":         "test-1",
				"expression": "{{ .field }}",
			},
			expected: &TransformAction{
				Expression: "{{ .field }}",
			},
		},
		{
			name:   "empty config",
			config: map[string]any{},
			expected: &TransformAction{
				Expression: "",
			},
		},
		{
			name: "partial config",
			config: map[string]any{
				"expression": "{ \"name\": {{ .name }}, \"age\": {{ .age }} }",
			},
			expected: &TransformAction{
				Expression: "{ \"name\": {{ .name }}, \"age\": {{ .age }} }",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewTransformAction(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestTransformAction_Execute_SimpleTransform(t *testing.T) {
	action := &TransformAction{
		Expression: "{{.step_results.user.name}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"user": map[string]any{
				"name": "John Doe",
				"age":  30,
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result)
}

func TestTransformAction_Execute_WithInput(t *testing.T) {
	action := &TransformAction{
		Expression: "{{.step_results.step1.data.temperature}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"step1": map[string]any{
				"data": map[string]any{
					"temperature": 25.5,
					"humidity":    60,
				},
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, 25.5, result)
}

func TestTransformAction_Execute_ObjectConstruction(t *testing.T) {
	action := &TransformAction{
		Expression: `{ "name": "{{.step_results.user.name}}", "status": "active", "age": {{.step_results.user.age}} }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"user": map[string]any{
				"name": "Alice",
				"age":  25,
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, "Alice", resultMap["name"])
	assert.Equal(t, "active", resultMap["status"])
	assert.Equal(t, float64(25), resultMap["age"]) // JSON numbers are parsed as float64
}

func TestTransformAction_Execute_ArrayTransform(t *testing.T) {
	action := &TransformAction{
		Expression: "{{index .step_results.users 0 \"name\"}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
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

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, "First User", result)
}

func TestTransformAction_Execute_ComplexTransform(t *testing.T) {
	action := &TransformAction{
		Expression: `{ "price": {{if .step_results.api_response.close}}{{.step_results.api_response.close}}{{else}}{{.step_results.api_response.open}}{{end}}, "currency": "USD", "timestamp": "{{.step_results.api_response.time}}" }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
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

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, float64(46000), resultMap["price"]) // JSON numbers are parsed as float64
	assert.Equal(t, "USD", resultMap["currency"])
	assert.Equal(t, "2023-10-01T10:00:00Z", resultMap["timestamp"])
}

func TestTransformAction_Execute_EmptyExpression(t *testing.T) {
	action := &TransformAction{
		Expression: "",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"data": "test",
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	// Empty expression should return empty string
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestTransformAction_Execute_InvalidExpression(t *testing.T) {
	action := &TransformAction{
		Expression: "{{.invalid..syntax}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"data": "test",
		},
	}

	_, err := action.Execute(context.Background(), execCtx, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transformation failed")
}

func TestTransformAction_Execute_WithCancel(t *testing.T) {
	action := &TransformAction{
		Expression: "{{.step_results.data}}",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	execCtx := models.ExecutionContext{
		StepResults: map[string]any{
			"data": "test value",
		},
	}

	result, err := action.Execute(ctx, execCtx, logger)

	// Transform action should complete even with cancelled context
	require.NoError(t, err)
	assert.Equal(t, "test value", result)
}
