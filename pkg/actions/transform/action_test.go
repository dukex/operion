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
			name: "config with expression",
			config: map[string]interface{}{
				"expression": "$.name",
			},
		},
		{
			name: "config with input and expression",
			config: map[string]interface{}{
				"input":      "$.data",
				"expression": "$.field",
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
		config   map[string]interface{}
		expected *TransformAction
	}{
		{
			name: "basic transform",
			config: map[string]interface{}{
				"id":         "test-1",
				"input":      "$.data",
				"expression": "$.field",
			},
			expected: &TransformAction{
				ID:         "test-1",
				Input:      "$.data",
				Expression: "$.field",
			},
		},
		{
			name:   "empty config",
			config: map[string]interface{}{},
			expected: &TransformAction{
				ID:         "",
				Input:      "",
				Expression: "",
			},
		},
		{
			name: "partial config",
			config: map[string]interface{}{
				"expression": "{ \"name\": $.name, \"age\": $.age }",
			},
			expected: &TransformAction{
				ID:         "",
				Input:      "",
				Expression: "{ \"name\": $.name, \"age\": $.age }",
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
		ID:         "test-transform",
		Input:      "",
		Expression: "$.user.name",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"user": map[string]interface{}{
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
		ID:         "test-input",
		Input:      "$.step1.data",
		Expression: "$.temperature",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"step1": map[string]interface{}{
				"data": map[string]interface{}{
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
		ID:         "test-object",
		Input:      "",
		Expression: `{ "name": $.user.name, "status": "active", "age": $.user.age }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"user": map[string]interface{}{
				"name": "Alice",
				"age":  25,
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, "Alice", resultMap["name"])
	assert.Equal(t, "active", resultMap["status"])
	assert.Equal(t, 25, resultMap["age"])
}

func TestTransformAction_Execute_ArrayTransform(t *testing.T) {
	action := &TransformAction{
		ID:         "test-array",
		Input:      "",
		Expression: "$.users[0].name",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{
					"name": "First User",
					"id":   1,
				},
				map[string]interface{}{
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
		ID:         "test-complex",
		Input:      "$.api_response",
		Expression: `{ "price": $.close ? $.close : $.open, "currency": "USD", "timestamp": $.time }`,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"api_response": map[string]interface{}{
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
	resultMap := result.(map[string]interface{})
	assert.Equal(t, 46000.0, resultMap["price"])
	assert.Equal(t, "USD", resultMap["currency"])
	assert.Equal(t, "2023-10-01T10:00:00Z", resultMap["timestamp"])
}

func TestTransformAction_Execute_EmptyExpression(t *testing.T) {
	action := &TransformAction{
		ID:         "test-empty",
		Input:      "",
		Expression: "",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"data": "test",
		},
	}

	_, err := action.Execute(context.Background(), execCtx, logger)

	// Empty expression should fail
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transformation failed")
}

func TestTransformAction_Execute_InvalidExpression(t *testing.T) {
	action := &TransformAction{
		ID:         "test-invalid",
		Input:      "",
		Expression: "$.invalid..syntax",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"data": "test",
		},
	}

	_, err := action.Execute(context.Background(), execCtx, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "transformation failed")
}

func TestTransformAction_Execute_InputNotFound(t *testing.T) {
	action := &TransformAction{
		ID:         "test-not-found",
		Input:      "$.nonexistent.field",
		Expression: "$.name",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"data": "test",
		},
	}

	_, err := action.Execute(context.Background(), execCtx, logger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get input data")
}

func TestTransformAction_Execute_WithCancel(t *testing.T) {
	action := &TransformAction{
		ID:         "test-cancel",
		Input:      "",
		Expression: "$.data",
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"data": "test value",
		},
	}

	result, err := action.Execute(ctx, execCtx, logger)

	// Transform action should complete even with cancelled context
	require.NoError(t, err)
	assert.Equal(t, "test value", result)
}

func TestTransformAction_Extract(t *testing.T) {
	action := &TransformAction{
		ID:         "test-extract",
		Input:      "",
		Expression: "",
	}

	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"step1": "value1",
			"step2": "value2",
		},
	}

	// Test empty input - should return all step results
	result, err := action.extract(execCtx)
	require.NoError(t, err)
	assert.Equal(t, execCtx.StepResults, result)

	// Test with specific input
	action.Input = "$.step1"
	result, err = action.extract(execCtx)
	require.NoError(t, err)
	assert.Equal(t, "value1", result)
}
