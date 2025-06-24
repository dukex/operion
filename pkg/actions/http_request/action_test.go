package http_request

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

func TestNewHTTPRequestAction(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected *HTTPRequestAction
	}{
		{
			name: "basic GET request",
			config: map[string]interface{}{
				"id":       "test-1",
				"method":   "GET",
				"host":     "api.example.com",
				"path":     "/data",
				"protocol": "https",
			},
			expected: &HTTPRequestAction{
				ID:       "test-1",
				Method:   "GET",
				Host:     "api.example.com",
				Path:     "/data",
				Protocol: "https",
				Headers:  map[string]string{},
				Body:     "",
				Timeout:  30 * time.Second,
				Retry:    RetryConfig{Attempts: 1, Delay: 0},
			},
		},
		{
			name: "POST request with headers and body",
			config: map[string]interface{}{
				"id":       "test-2",
				"method":   "post",
				"host":     "api.example.com",
				"path":     "/create",
				"protocol": "https",
				"body":     `{"key": "value"}`,
				"headers": map[string]interface{}{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				"retry": map[string]interface{}{
					"attempts": 3.0,
					"delay":    5.0,
				},
			},
			expected: &HTTPRequestAction{
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
				Retry:   RetryConfig{Attempts: 3, Delay: 5},
			},
		},
		{
			name: "default method is GET",
			config: map[string]interface{}{
				"id":       "test-3",
				"host":     "api.example.com",
				"path":     "/default",
				"protocol": "https",
			},
			expected: &HTTPRequestAction{
				ID:       "test-3",
				Method:   "GET",
				Host:     "api.example.com",
				Path:     "/default",
				Protocol: "https",
				Headers:  map[string]string{},
				Body:     "",
				Timeout:  30 * time.Second,
				Retry:    RetryConfig{Attempts: 1, Delay: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewHTTPRequestAction(tt.config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, action)
		})
	}
}

func TestHTTPRequestAction_Execute_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := map[string]interface{}{
			"status": "success",
			"data":   "test response",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:       "test-http",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]interface{}),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
	assert.NotNil(t, resultMap["body"])
	assert.NotNil(t, resultMap["headers"])

	// Check the parsed JSON body
	body := resultMap["body"].(map[string]interface{})
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, "test response", body["data"])
}

func TestHTTPRequestAction_Execute_POST_WithBody(t *testing.T) {
	// Create test server that expects POST with JSON body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify body
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "test value", body["key"])

		response := map[string]interface{}{
			"created": true,
			"id":      123,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	action := &HTTPRequestAction{
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
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]interface{}),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])

	body := resultMap["body"].(map[string]interface{})
	assert.Equal(t, true, body["created"])
	assert.Equal(t, float64(123), body["id"])
}

func TestHTTPRequestAction_Execute_WithRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Success on third attempt
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:       "test-retry",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Timeout:  5 * time.Second,
		Retry:    RetryConfig{Attempts: 3, Delay: 0}, // No delay for faster test
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]interface{}),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
}

func TestHTTPRequestAction_Execute_WithTemplating(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)

		// Verify templated data was used
		assert.Equal(t, "user123", body["user_id"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:       "test-template",
		Method:   "POST",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Body:     `{"user_id": steps.previous_step.user_id}`,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: map[string]interface{}{
			"previous_step": map[string]interface{}{
				"user_id": "user123",
			},
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
}

func TestHTTPRequestAction_Execute_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:       "test-timeout",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Timeout:  100 * time.Millisecond, // Very short timeout
		Retry:    RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]interface{}),
	}

	_, err := action.Execute(context.Background(), execCtx, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "http request failed")
}

func TestHTTPRequestAction_Execute_NonJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:       "test-text",
		Method:   "GET",
		Host:     strings.Split(server.URL, "://")[1],
		Protocol: strings.Split(server.URL, "://")[0],
		Path:     "/",
		Timeout:  30 * time.Second,
		Retry:    RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]interface{}),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
	assert.Equal(t, "plain text response", resultMap["body"])
}
