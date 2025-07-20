package template

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_SimpleExpression(t *testing.T) {
	data := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	// Test simple field access
	result, err := Render("name", data)
	require.NoError(t, err)
	assert.Equal(t, "John", result)

	// Test number field
	result, err = Render("age", data)
	require.NoError(t, err)
	assert.Equal(t, 30, result)
}

func TestRender_ComplexExpression(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "Alice",
			"email": "alice@example.com",
		},
		"orders": []interface{}{
			map[string]interface{}{"id": 1, "total": 100.50},
			map[string]interface{}{"id": 2, "total": 75.25},
		},
	}

	// Test nested field access
	result, err := Render("user.name", data)
	require.NoError(t, err)
	assert.Equal(t, "Alice", result)

	// Test array operations
	result, err = Render("$sum(orders.total)", data)
	require.NoError(t, err)
	assert.Equal(t, 175.75, result)

	// Test object construction
	result, err = Render(`{
		"user_name": user.name,
		"total_orders": $count(orders),
		"total_value": $sum(orders.total)
	}`, data)
	require.NoError(t, err)

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Alice", resultMap["user_name"])
	assert.Equal(t, 2, resultMap["total_orders"])
	assert.Equal(t, 175.75, resultMap["total_value"])
}

func TestRender_WithStepResults(t *testing.T) {
	// Simulate execution context step results
	data := map[string]interface{}{
		"api_call": map[string]interface{}{
			"status": 200,
			"body": map[string]interface{}{
				"user_id":  123,
				"username": "testuser",
			},
		},
		"validation": map[string]interface{}{
			"valid":  true,
			"errors": []interface{}{},
		},
	}

	// Test accessing step results
	result, err := Render("api_call.body.username", data)
	require.NoError(t, err)
	assert.Equal(t, "testuser", result)

	// Test conditional expression
	result, err = Render("api_call.status = 200 ? 'success' : 'failed'", data)
	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestRender_ErrorHandling(t *testing.T) {
	data := map[string]interface{}{
		"test": "value",
	}

	// Test invalid JSONata expression
	_, err := Render("invalid..expression", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile input expression")

	// Test reference to non-existent field (actually errors in JSONata)
	_, err = Render("nonexistent.field", data)
	assert.Error(t, err) // JSONata returns "no results found" error
}

func TestRender_EnvironmentVariables(t *testing.T) {
	// Set test environment variable
	if err := os.Setenv("TEST_VAR", "test_value"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_VAR"); err != nil {
			t.Error(err)
		}
	}()

	// Current implementation doesn't support env vars, but let's test what we expect
	data := map[string]interface{}{
		"step_result": "some_value",
	}

	// This should fail with current implementation since env vars aren't supported
	_, err := Render("env.TEST_VAR", data)
	assert.Error(t, err) // Expected to fail with current implementation
}

// Test for workflow variables integration
func TestRender_WorkflowVariables(t *testing.T) {
	// Simulate complete execution context with variables
	data := map[string]interface{}{
		// Step results
		"api_response": map[string]interface{}{
			"data": "response_data",
		},
		// Workflow variables should be accessible but currently aren't in HTTP action
		"workflow_vars": map[string]interface{}{
			"api_endpoint": "https://api.example.com",
			"timeout":      30,
		},
	}

	// Test accessing what should be workflow variables
	result, err := Render("workflow_vars.api_endpoint", data)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", result)
}

func TestRender_StringInterpolation(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name": "John",
			"id":   123,
		},
		"action": "login",
	}

	// Test string construction (JSONata way, not {{ }} syntax)
	result, err := Render(`"User " & user.name & " performed " & action`, data)
	require.NoError(t, err)
	assert.Equal(t, "User John performed login", result)

	// Test URL construction
	result, err = Render(`"https://api.example.com/users/" & $string(user.id)`, data)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users/123", result)
}
