package httprequest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPRequestAction_EnhancedTemplating(t *testing.T) {
	// Set up test environment variables
	if err := os.Setenv("TEST_API_TOKEN", "secret123"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_API_TOKEN"); err != nil {
			t.Error(err)
		}
	}()

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL was templated correctly
		assert.Equal(t, "/users/123/orders", r.URL.Path)

		// Verify headers were templated correctly
		assert.Equal(t, "Bearer secret123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "test-workflow", r.Header.Get("X-Workflow-ID"))

		// Verify body was templated correctly
		if r.Method == "POST" {
			var body map[string]any
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)

			assert.Equal(t, "123", body["user_id"])
			assert.Equal(t, "test-workflow", body["workflow_id"])
			assert.Equal(t, "api.example.com", body["api_endpoint"])
			assert.Equal(t, 100.0, body["order_total"])
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"success":  true,
			"order_id": "ORD-456",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// Create HTTP request action with enhanced templating
	action := &HTTPRequestAction{
		Method:   "POST",
		Host:     strings.Split(server.URL, "://")[1], // Extract host from URL
		Protocol: strings.Split(server.URL, "://")[0], // Extract protocol from URL
		Path:     "/users/{{ .vars.user_id }}/orders",
		Headers: map[string]string{
			"Authorization": "Bearer {{ .env.TEST_API_TOKEN }}",
			"Content-Type":  "application/json",
			"X-Workflow-ID": "{{ .execution.workflow_id }}",
		},
		Body: `{
			"user_id": "{{ .vars.user_id }}",
			"workflow_id": "{{ .execution.workflow_id }}",
			"api_endpoint": "{{ .vars.config.api_endpoint }}",
			"order_total": {{ .step_results.price_calculation.total }}
		}`,
		Timeout: 10 * time.Second, // Increase timeout
		Retry: RetryConfig{
			Attempts: 1,
			Delay:    0,
		},
	}

	// Create execution context with all data types
	executionCtx := models.ExecutionContext{
		ID:         "exec-789",
		WorkflowID: "test-workflow",
		Variables: map[string]any{
			"user_id": 123,
			"config": map[string]any{
				"api_endpoint": "api.example.com",
				"version":      "v1",
			},
		},
		StepResults: map[string]any{
			"price_calculation": map[string]any{
				"total":    100.0,
				"currency": "USD",
			},
			"user_lookup": map[string]any{
				"name":   "john_doe",
				"active": true,
			},
		},
		TriggerData: map[string]any{
			"source": "webhook",
			"event":  "order_created",
		},
		Metadata: map[string]any{
			"version":   "1.0",
			"timestamp": "2023-12-01T10:00:00Z",
		},
	}

	// Execute the action
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	result, err := action.Execute(ctx, executionCtx, logger)

	// Verify execution succeeded
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify response data
	resultMap, ok := result.(map[string]any)
	require.True(t, ok)

	// Check if body was parsed as JSON
	body, bodyExists := resultMap["body"]
	require.True(t, bodyExists, "Response body should exist")

	if bodyMap, ok := body.(map[string]any); ok {
		assert.Equal(t, true, bodyMap["success"])
		assert.Equal(t, "ORD-456", bodyMap["order_id"])
	} else {
		t.Logf("Body is not a map, got: %T %+v", body, body)
		t.Fail()
	}
}

func TestHTTPRequestAction_EnvironmentVariableAccess(t *testing.T) {
	// Simple test just to verify env variable access without complex networking
	if err := os.Setenv("TEST_API_KEY", "test-key-123"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Unsetenv("TEST_API_KEY"); err != nil {
			t.Error(err)
		}
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify header was templated with environment variable
		assert.Equal(t, "Bearer test-key-123", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1], // Extract host from URL
		Protocol: strings.Split(server.URL, "://")[0], // Extract protocol from URL
		Path:     "/",
		Headers: map[string]string{
			"Authorization": "Bearer {{ .env.TEST_API_KEY }}",
		},
		Timeout: 10 * time.Second,
		Retry: RetryConfig{
			Attempts: 1,
			Delay:    0,
		},
	}

	executionCtx := models.ExecutionContext{
		ID:         "env-test",
		WorkflowID: "env-test-workflow",
	}

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	result, err := action.Execute(ctx, executionCtx, logger)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, ok := result.(map[string]any)
	require.True(t, ok)

	// Check if body was parsed as JSON
	body, bodyExists := resultMap["body"]
	require.True(t, bodyExists, "Response body should exist")

	if bodyMap, ok := body.(map[string]any); ok {
		assert.Equal(t, "ok", bodyMap["status"])
	} else {
		t.Logf("Body is not a map, got: %T %+v", body, body)
		t.Fail()
	}
}
