package transform

import (
	"context"
	"testing"

	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransformAction(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *TransformAction
	}{
		{
			name: "basic transform action",
			config: map[string]interface{}{
				"id":         "test-transform-1",
				"input":      "data",
				"expression": "$uppercase(name)",
			},
			expected: &TransformAction{
				ID:         "test-transform-1",
				Input:      "data",
				Expression: "$uppercase(name)",
			},
		},
		{
			name: "transform action without input",
			config: map[string]interface{}{
				"id":         "test-transform-2",
				"expression": "$count(items)",
			},
			expected: &TransformAction{
				ID:         "test-transform-2",
				Input:      "",
				Expression: "$count(items)",
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
			name: "complex JSONata expression",
			config: map[string]interface{}{
				"id":         "test-transform-3",
				"input":      "results[0]",
				"expression": "{ \"total\": $sum(amounts), \"count\": $count(items) }",
			},
			expected: &TransformAction{
				ID:         "test-transform-3",
				Input:      "results[0]",
				Expression: "{ \"total\": $sum(amounts), \"count\": $count(items) }",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewTransformAction(tt.config)

			require.NoError(t, err)
			assert.Equal(t, tt.expected.ID, action.ID)
			assert.Equal(t, tt.expected.Input, action.Input)
			assert.Equal(t, tt.expected.Expression, action.Expression)
		})
	}
}

func TestTransformAction_GetMethods(t *testing.T) {
	action := &TransformAction{
		ID:         "test-transform",
		Input:      "data.results",
		Expression: "$uppercase(name)",
	}

	assert.Equal(t, "test-transform", action.GetID())
	assert.Equal(t, "transform", action.GetType())

	config := action.GetConfig()
	assert.Equal(t, "test-transform", config["id"])
	assert.Equal(t, "data.results", config["input"])
	assert.Equal(t, "$uppercase(name)", config["expression"])

	assert.NoError(t, action.Validate())
}

func TestTransformAction_Execute_SimpleTransformation(t *testing.T) {
	action := &TransformAction{
		ID:         "test-simple",
		Input:      "",
		Expression: "name",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"name": "John Doe",
			"age":  30,
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	assert.Equal(t, "John Doe", result)
}

func TestTransformAction_Execute_ArrayTransformation(t *testing.T) {
	action := &TransformAction{
		ID:         "test-array",
		Input:      "",
		Expression: "$count(items)",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"items": []interface{}{"a", "b", "c", "d"},
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	// JSONata can return int or float64 depending on the operation
	switch v := result.(type) {
	case int:
		assert.Equal(t, 4, v)
	case float64:
		assert.Equal(t, float64(4), v)
	default:
		t.Errorf("Expected int or float64, got %T", result)
	}
}

func TestTransformAction_Execute_ObjectConstruction(t *testing.T) {
	action := &TransformAction{
		ID:         "test-object",
		Input:      "",
		Expression: "{ \"fullName\": firstName & \" \" & lastName, \"isAdult\": age >= 18 }",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"firstName": "John",
			"lastName":  "Doe",
			"age":       25,
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, "John Doe", resultMap["fullName"])
	assert.Equal(t, true, resultMap["isAdult"])
}

func TestTransformAction_Execute_WithInputExpression(t *testing.T) {
	action := &TransformAction{
		ID:         "test-input",
		Input:      "data.users[0]",
		Expression: "name",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"data": map[string]interface{}{
				"users": []interface{}{
					map[string]interface{}{
						"name": "Alice",
						"age":  28,
					},
					map[string]interface{}{
						"name": "Bob",
						"age":  32,
					},
				},
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	assert.Equal(t, "Alice", result)
}

func TestTransformAction_Execute_MathOperations(t *testing.T) {
	action := &TransformAction{
		ID:         "test-math",
		Input:      "",
		Expression: "{ \"sum\": $sum(numbers), \"avg\": $sum(numbers) / $count(numbers), \"max\": $max(numbers) }",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"numbers": []interface{}{10, 20, 30, 40, 50},
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, float64(150), resultMap["sum"])
	assert.Equal(t, float64(30), resultMap["avg"])
	assert.Equal(t, float64(50), resultMap["max"])
}

func TestTransformAction_Execute_StringOperations(t *testing.T) {
	action := &TransformAction{
		ID:         "test-string",
		Input:      "",
		Expression: "{ \"upper\": $uppercase(text), \"lower\": $lowercase(text), \"length\": $length(text) }",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"text": "Hello World",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, "HELLO WORLD", resultMap["upper"])
	assert.Equal(t, "hello world", resultMap["lower"])
	// JSONata can return int or float64 for length
	switch v := resultMap["length"].(type) {
	case int:
		assert.Equal(t, 11, v)
	case float64:
		assert.Equal(t, float64(11), v)
	default:
		t.Errorf("Expected int or float64 for length, got %T", resultMap["length"])
	}
}

func TestTransformAction_Execute_InvalidExpression(t *testing.T) {
	action := &TransformAction{
		ID:         "test-invalid",
		Input:      "",
		Expression: "invalid((expression",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger:      logger,
		StepResults: map[string]interface{}{},
	}

	result, err := action.Execute(context.Background(), execCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "transformation failed")
}

func TestTransformAction_Execute_InvalidInputExpression(t *testing.T) {
	action := &TransformAction{
		ID:         "test-invalid-input",
		Input:      "invalid((input",
		Expression: "name",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"name": "John",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get input data")
}

func TestTransformAction_Execute_NonExistentField(t *testing.T) {
	action := &TransformAction{
		ID:         "test-nonexistent",
		Input:      "",
		Expression: "nonexistent.field",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger: logger,
		StepResults: map[string]interface{}{
			"name": "John",
		},
	}

	result, err := action.Execute(context.Background(), execCtx)

	// JSONata should return an error for non-existent fields in this case
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTransformAction_Execute_EmptyStepResults(t *testing.T) {
	action := &TransformAction{
		ID:         "test-empty",
		Input:      "",
		Expression: "$count($keys($))",
	}

	logger := log.WithField("test", "transform_action")
	execCtx := models.ExecutionContext{
		Logger:      logger,
		StepResults: map[string]interface{}{},
	}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	// JSONata can return int or float64 for count
	switch v := result.(type) {
	case int:
		assert.Equal(t, 0, v)
	case float64:
		assert.Equal(t, float64(0), v)
	default:
		t.Errorf("Expected int or float64, got %T", result)
	}
}

func TestTransformAction_GetConfig_Consistency(t *testing.T) {
	config := map[string]interface{}{
		"id":         "config-test",
		"input":      "data.items",
		"expression": "$sum(amounts)",
	}

	action, err := NewTransformAction(config)
	require.NoError(t, err)

	retrievedConfig := action.GetConfig()

	// Config should match the original action properties
	assert.Equal(t, action.ID, retrievedConfig["id"])
	assert.Equal(t, action.Input, retrievedConfig["input"])
	assert.Equal(t, action.Expression, retrievedConfig["expression"])
}
