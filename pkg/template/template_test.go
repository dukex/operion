package template

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_SimpleExpression(t *testing.T) {
	data := map[string]any{
		"name":  "John",
		"age":   30,
		"isNew": true,
	}

	// Test simple field access
	result, err := Render("{{ .name }}", data)
	require.NoError(t, err)
	assert.Equal(t, "John", result)

	// Test boolean expression
	result, err = Render("{{ .isNew }}", data)
	require.NoError(t, err)
	assert.Equal(t, true, result)

	// Test number field - always map to float
	result, err = Render("{{ .age }}", data)
	require.NoError(t, err)
	assert.Equal(t, 30.0, result)
}

func TestRender_ComplexExpression(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name":  "Alice",
			"email": "alice@example.com",
		},
		"orders": []any{
			map[string]any{"id": 1, "total": 100.50},
			map[string]any{"id": 2, "total": 75.25},
		},
	}

	// Test nested field access
	result, err := Render("{{ .user.name }}", data)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)

	// Test object construction
	result, err = Render(`{
		"user_name": "{{ .user.name }}",
		"total_orders": {{ len .orders }}
	}`, data)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]any)

	require.True(t, ok)
	assert.Equal(t, "Alice", resultMap["user_name"])
	assert.Equal(t, 2.0, resultMap["total_orders"])
}

func TestRender_WithStepResults(t *testing.T) {
	// Simulate execution context step results
	data := map[string]any{
		"api_call": map[string]any{
			"status": 200,
			"body": map[string]any{
				"user_id":  123,
				"username": "testuser",
			},
		},
		"validation": map[string]any{
			"valid":  true,
			"errors": []any{},
		},
	}

	// Test accessing step results
	result, err := Render("{{ .api_call.body.username }}", data)
	require.NoError(t, err)
	assert.Equal(t, "testuser", result)

	// Test conditional expression
	result, err = Render("{{ if eq .api_call.status 200 }}success{{ else }}failed{{ end }}", data)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestRender_ErrorHandling(t *testing.T) {
	data := map[string]any{
		"test": "value",
	}

	// Test invalid template expression
	_, err := Render("{ invalid..expression }}", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse json")

	// Test reference to non-existent field (actually errors in template)
	_, err = Render("{{ nonexistent.field }}", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "function \"nonexistent\" not defined")
}

func TestRender_EnvironmentVariables(t *testing.T) {
	// Set test environment variable
	if err := os.Setenv("TEST_VAR", "test_value"); err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := os.Unsetenv("TEST_VAR")
		if err != nil {
			t.Error(err)
		}
	}()

	// Current implementation doesn't support env vars, but let's test what we expect
	data := map[string]any{
		"step_result": "some_value",
	}

	// This should fail with current implementation since env vars aren't supported
	_, err := Render("{{ env.TEST_VAR }}", data)
	assert.Error(t, err) // Expected to fail with current implementation
}

// Test for workflow variables integration.
func TestRender_WorkflowVariables(t *testing.T) {
	// Simulate complete execution context with variables
	data := map[string]any{
		// Step results
		"api_response": map[string]any{
			"data": "response_data",
		},
		// Workflow variables should be accessible but currently aren't in HTTP action
		"workflow_vars": map[string]any{
			"api_endpoint": "https://api.example.com",
			"timeout":      30,
		},
	}

	// Test accessing what should be workflow variables
	result, err := Render("{{ .workflow_vars.api_endpoint }}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", result)
}

func TestRender_StringInterpolation(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name": "John",
			"id":   123,
		},
		"action": "login",
	}

	// Test string construction
	result, err := Render("User {{.user.name}} performed {{.action}}", data)
	require.NoError(t, err)
	assert.Equal(t, "User John performed login", result)

	// Test URL construction
	result, err = Render("https://api.example.com/users/{{.user.id}}", data)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users/123", result)
}
