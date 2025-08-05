package httprequest_test

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

	"github.com/dukex/operion/pkg/actions/httprequest"
	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPRequestAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   map[string]any
		expected *httprequest.Action
	}{
		{
			name: "basic GET request",
			config: map[string]any{
				"id":       "test-1",
				"method":   "GET",
				"host":     "api.example.com",
				"path":     "/data",
				"protocol": "https",
			},
			expected: &httprequest.Action{
				ID:       "test-1",
				Method:   "GET",
				Host:     "api.example.com",
				Path:     "/data",
				Protocol: "https",
				Headers:  map[string]string{},
				Body:     "",
				Timeout:  30 * time.Second,
				Retry:    httprequest.RetryConfig{Attempts: 1, Delay: 0},
			},
		},
		{
			name: "POST request with headers and body",
			config: map[string]any{
				"id":       "test-2",
				"method":   "post",
				"host":     "api.example.com",
				"path":     "/create",
				"protocol": "https",
				"body":     `{"key": "value"}`,
				"headers": map[string]any{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				"retry": map[string]any{
					"attempts": 3.0,
					"delay":    5.0,
				},
			},
			expected: &httprequest.Action{
				ID:       "test-2",
				Method:   "POST",
				Host:     "api.example.com",
				Path:     "/create",
				Protocol: "https",
				Body:     `{"key": "value"}`,
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				Timeout: 30 * time.Second,
				Retry:   httprequest.RetryConfig{Attempts: 3, Delay: 5},
			},
		},
		{
			name: "default method is GET",
			config: map[string]any{
				"id":       "test-3",
				"host":     "api.example.com",
				"path":     "/default",
				"protocol": "https",
			},
			expected: &httprequest.Action{
				ID:       "test-3",
				Method:   "GET",
				Host:     "api.example.com",
				Path:     "/default",
				Protocol: "https",
				Headers:  map[string]string{},
				Body:     "",
				Timeout:  30 * time.Second,
				Retry:    httprequest.RetryConfig{Attempts: 1, Delay: 0},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			action, err := httprequest.NewAction(testCase.config)
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, action)
		})
	}
}

func TestHTTPRequestAction_Execute_Success(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "GET", request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Accept"))

		response := map[string]any{
			"status": "success",
			"data":   "test response",
		}

		writer.Header().Set("Content-Type", "application/json")

		err := json.NewEncoder(writer).Encode(response)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-http",
		Method:   "GET",
		Body:     "",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   httprequest.RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, isMap := result.(map[string]any)
	assert.True(t, isMap, "result should be a map[string]any")
	assert.Equal(t, 200, resultMap["status_code"])
	assert.NotNil(t, resultMap["body"])
	assert.NotNil(t, resultMap["headers"])

	// Check the parsed JSON body
	body, isBodyMap := resultMap["body"].(map[string]any)
	assert.True(t, isBodyMap, "body should be a map[string]any")
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "test response", body["data"])
}

func TestHTTPRequestAction_Execute_POST_WithBody(t *testing.T) {
	t.Parallel()

	// Create test server that expects POST with JSON body
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "POST", request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		// Read and verify body
		var body map[string]any

		err := json.NewDecoder(request.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "test value", body["key"])

		response := map[string]any{
			"created": true,
			"id":      123,
		}

		writer.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(writer).Encode(response)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-http-post",
		Method:   "POST",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Body:     `{"key": "test value"}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   httprequest.RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap, isMap := result.(map[string]any)
	assert.True(t, isMap, "result should be a map[string]any")
	assert.Equal(t, 200, resultMap["status_code"])

	body, isBodyMap := resultMap["body"].(map[string]any)
	assert.True(t, isBodyMap, "body should be a map[string]any")
	assert.Equal(t, true, body["created"])
	assert.InEpsilon(t, 123, body["id"], 0.01)
}

func TestHTTPRequestAction_Execute_WithRetry(t *testing.T) {
	t.Parallel()

	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			writer.WriteHeader(http.StatusInternalServerError)

			return
		}
		// Success on third attempt
		writer.WriteHeader(http.StatusOK)

		err := json.NewEncoder(writer).Encode(map[string]string{"status": "success"})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-retry",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Headers:  make(map[string]string),
		Body:     "",
		Timeout:  5 * time.Second,
		Retry:    httprequest.RetryConfig{Attempts: 3, Delay: 0}, // No delay for faster test
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)

	resultMap, isMap := result.(map[string]any)
	assert.True(t, isMap, "result should be a map[string]any")
	assert.Equal(t, 200, resultMap["status_code"])
}

func TestHTTPRequestAction_Execute_WithTemplating(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body map[string]any

		err := json.NewDecoder(request.Body).Decode(&body)
		assert.NoError(t, err)

		// Verify templated data was used
		assert.Equal(t, "user123", body["user_id"])

		writer.WriteHeader(http.StatusOK)

		err = json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-template",
		Method:   "POST",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Body:     `{"user_id": "{{ .step_results.previous_step.user_id}}" }`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   httprequest.RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		StepResults: map[string]any{
			"previous_step": map[string]any{
				"user_id": "user123",
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)

	resultMap, isMap := result.(map[string]any)
	assert.True(t, isMap, "result should be a map[string]any")
	assert.Equal(t, 200, resultMap["status_code"])
}

func TestHTTPRequestAction_Execute_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-timeout",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Headers:  make(map[string]string),
		Body:     "",
		Timeout:  100 * time.Millisecond, // Very short timeout
		Retry:    httprequest.RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	_, err := action.Execute(context.Background(), execCtx, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http request failed")
}

func TestHTTPRequestAction_Execute_NonJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		writer.WriteHeader(http.StatusOK)

		_, err := writer.Write([]byte("plain text response"))
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	action := &httprequest.Action{
		ID:       "test-text",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Headers:  make(map[string]string),
		Body:     "",
		Timeout:  30 * time.Second,
		Retry:    httprequest.RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		ID:          "test-execution",
		WorkflowID:  "test-workflow",
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		Metadata:    make(map[string]any),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)

	resultMap, isMap := result.(map[string]any)
	assert.True(t, isMap, "result should be a map[string]any")
	assert.Equal(t, 200, resultMap["status_code"])
	assert.Equal(t, "plain text response", resultMap["body"])
}
