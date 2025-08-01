package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConditional(t *testing.T) {
	tests := []struct {
		name        string
		conditional ConditionalExpression
		expectNil   bool
	}{
		{
			name: "simple language returns SimpleConditionalInterpreter",
			conditional: ConditionalExpression{
				Language:   "simple",
				Expression: "true",
			},
			expectNil: false,
		},
		{
			name: "unsupported language returns nil",
			conditional: ConditionalExpression{
				Language:   "python",
				Expression: "True",
			},
			expectNil: true,
		},
		{
			name: "empty language returns nil",
			conditional: ConditionalExpression{
				Language:   "",
				Expression: "true",
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetConditional(tt.conditional)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.IsType(t, &SimpleConditionalInterpreter{}, result)
			}
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_NilExpression(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	result, err := interpreter.Evaluate(nil)
	require.NoError(t, err)
	assert.True(t, result, "Nil expressions should default to true")
}

func TestSimpleConditionalInterpreter_Evaluate_BooleanValues(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "true boolean",
			input:    true,
			expected: true,
		},
		{
			name:     "false boolean",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_StringValues(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name        string
		input       string
		expected    bool
		expectError bool
	}{
		{
			name:     "empty string defaults to true",
			input:    "",
			expected: true,
		},
		{
			name:     "string true",
			input:    "true",
			expected: true,
		},
		{
			name:     "string false",
			input:    "false",
			expected: false,
		},
		{
			name:     "string 1",
			input:    "1",
			expected: true,
		},
		{
			name:     "string 0",
			input:    "0",
			expected: false,
		},
		{
			name:        "invalid string",
			input:       "maybe",
			expected:    false,
			expectError: true,
		},
		{
			name:        "random string",
			input:       "hello",
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "cannot convert string")
				assert.False(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_IntegerValues(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{
			name:     "zero integer",
			input:    0,
			expected: false,
		},
		{
			name:     "positive integer",
			input:    42,
			expected: true,
		},
		{
			name:     "negative integer",
			input:    -1,
			expected: true,
		},
		{
			name:     "large integer",
			input:    999999,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_Int64Values(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name     string
		input    int64
		expected bool
	}{
		{
			name:     "zero int64",
			input:    int64(0),
			expected: false,
		},
		{
			name:     "positive int64",
			input:    int64(123456789),
			expected: true,
		},
		{
			name:     "negative int64",
			input:    int64(-999),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_Float64Values(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name     string
		input    float64
		expected bool
	}{
		{
			name:     "zero float64",
			input:    0.0,
			expected: false,
		},
		{
			name:     "positive float64",
			input:    3.14,
			expected: true,
		},
		{
			name:     "negative float64",
			input:    -2.5,
			expected: true,
		},
		{
			name:     "very small positive",
			input:    0.0001,
			expected: true,
		},
		{
			name:     "very small negative",
			input:    -0.0001,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSimpleConditionalInterpreter_Evaluate_UnsupportedTypes(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "slice",
			input: []string{"a", "b", "c"},
		},
		{
			name:  "map",
			input: map[string]any{"key": "value"},
		},
		{
			name:  "struct",
			input: struct{ Name string }{Name: "test"},
		},
		{
			name:  "channel",
			input: make(chan int),
		},
		{
			name:  "function",
			input: func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cannot convert")
			assert.False(t, result)
		})
	}
}

func TestConditionalIntegration_WithGetConditional(t *testing.T) {
	tests := []struct {
		name        string
		conditional ConditionalExpression
		input       any
		expected    bool
		expectError bool
	}{
		{
			name: "simple language with true boolean",
			conditional: ConditionalExpression{
				Language:   "simple",
				Expression: "{{eq .user.role \"admin\"}}",
			},
			input:    true,
			expected: true,
		},
		{
			name: "simple language with false boolean",
			conditional: ConditionalExpression{
				Language:   "simple",
				Expression: "{{eq .user.role \"user\"}}",
			},
			input:    false,
			expected: false,
		},
		{
			name: "simple language with integer",
			conditional: ConditionalExpression{
				Language:   "simple",
				Expression: "{{.api_call.status}}",
			},
			input:    200,
			expected: true,
		},
		{
			name: "simple language with zero integer",
			conditional: ConditionalExpression{
				Language:   "simple",
				Expression: "{{.retry_count}}",
			},
			input:    0,
			expected: false,
		},
		{
			name: "unsupported language",
			conditional: ConditionalExpression{
				Language:   "javascript",
				Expression: "user.role === 'admin'",
			},
			input:       true,
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interpreter := GetConditional(tt.conditional)

			if tt.expectError {
				assert.Nil(t, interpreter, "Should return nil for unsupported languages")
			} else {
				require.NotNil(t, interpreter, "Should return valid interpreter for supported languages")

				result, err := interpreter.Evaluate(tt.input)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConditionalTemplateResultScenarios(t *testing.T) {
	interpreter := &SimpleConditionalInterpreter{}

	tests := []struct {
		name           string
		templateResult any
		expected       bool
		expectError    bool
		description    string
	}{
		{
			name:           "template returns boolean true",
			templateResult: true,
			expected:       true,
			description:    "Direct boolean template result",
		},
		{
			name:           "template returns boolean false",
			templateResult: false,
			expected:       false,
			description:    "Direct boolean template result",
		},
		{
			name:           "template returns string 'true'",
			templateResult: "true",
			expected:       true,
			description:    "String boolean template result",
		},
		{
			name:           "template returns string 'false'",
			templateResult: "false",
			expected:       false,
			description:    "String boolean template result",
		},
		{
			name:           "template returns integer 1",
			templateResult: 1,
			expected:       true,
			description:    "Truthy integer template result",
		},
		{
			name:           "template returns integer 0",
			templateResult: 0,
			expected:       false,
			description:    "Falsy integer template result",
		},
		{
			name:           "template returns HTTP status 200",
			templateResult: 200,
			expected:       true,
			description:    "API status code template result",
		},
		{
			name:           "template returns float 3.14",
			templateResult: 3.14,
			expected:       true,
			description:    "Non-zero float template result",
		},
		{
			name:           "template returns float 0.0",
			templateResult: 0.0,
			expected:       false,
			description:    "Zero float template result",
		},
		{
			name:           "template returns empty string",
			templateResult: "",
			expected:       true,
			description:    "Empty string defaults to true",
		},
		{
			name:           "template returns nil",
			templateResult: nil,
			expected:       true,
			description:    "Nil template result defaults to true",
		},
		{
			name:           "template returns complex object",
			templateResult: map[string]any{"status": "success"},
			expected:       false,
			expectError:    true,
			description:    "Unsupported template result type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := interpreter.Evaluate(tt.templateResult)

			if tt.expectError {
				require.Error(t, err, tt.description)
				assert.False(t, result)
			} else {
				require.NoError(t, err, tt.description)
				assert.Equal(t, tt.expected, result, tt.description)
			}
		})
	}
}
