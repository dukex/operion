package http_request

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	log "github.com/sirupsen/logrus"
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
				"id":     "test-1",
				"method": "GET",
				"url":    "https://api.example.com/data",
			},
			expected: &HTTPRequestAction{
				ID:      "test-1",
				Method:  "GET",
				URL:     "https://api.example.com/data",
				Headers: map[string]string{},
				Body:    "",
				Timeout: 30 * time.Second,
				Retry:   RetryConfig{Attempts: 1, Delay: 0},
			},
		},
		{
			name: "POST request with headers and body",
			config: map[string]interface{}{
				"id":     "test-2",
				"method": "post",
				"url":    "https://api.example.com/create",
				"body":   `{"key": "value"}`,
				"headers": map[string]interface{}{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
			},
			expected: &HTTPRequestAction{
				ID:     "test-2",
				Method: "POST",
				URL:    "https://api.example.com/create",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				Body:    `{"key": "value"}`,
				Timeout: 30 * time.Second,
				Retry:   RetryConfig{Attempts: 1, Delay: 0},
			},
		},
		{
			name: "request with retry configuration",
			config: map[string]interface{}{
				"id":     "test-3",
				"method": "GET",
				"url":    "https://api.example.com/retry",
				"retry": map[string]interface{}{
					"attempts": float64(3),
					"delay":    float64(5),
				},
			},
			expected: &HTTPRequestAction{
				ID:      "test-3",
				Method:  "GET",
				URL:     "https://api.example.com/retry",
				Headers: map[string]string{},
				Body:    "",
				Timeout: 30 * time.Second,
				Retry:   RetryConfig{Attempts: 3, Delay: 5},
			},
		},
		{
			name: "defaults when method not specified",
			config: map[string]interface{}{
				"id":  "test-4",
				"url": "https://api.example.com/default",
			},
			expected: &HTTPRequestAction{
				ID:      "test-4",
				Method:  "GET",
				URL:     "https://api.example.com/default",
				Headers: map[string]string{},
				Body:    "",
				Timeout: 30 * time.Second,
				Retry:   RetryConfig{Attempts: 1, Delay: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := NewHTTPRequestAction(tt.config)

			require.NoError(t, err)
			assert.Equal(t, tt.expected.ID, action.ID)
			assert.Equal(t, tt.expected.Method, action.Method)
			assert.Equal(t, tt.expected.URL, action.URL)
			assert.Equal(t, tt.expected.Headers, action.Headers)
			assert.Equal(t, tt.expected.Body, action.Body)
			assert.Equal(t, tt.expected.Timeout, action.Timeout)
			assert.Equal(t, tt.expected.Retry, action.Retry)
		})
	}
}

func TestHTTPRequestAction_GetMethods(t *testing.T) {
	action := &HTTPRequestAction{ID: "test", Method: "GET", URL: "https://example.com"}

	assert.Equal(t, "test", action.GetID())
	assert.Equal(t, "http_request", action.GetType())
	assert.Nil(t, action.GetConfig())
	assert.NoError(t, action.Validate())
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
		ID:     "test-http",
		Method: "GET",
		URL:    server.URL,
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

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

func TestHTTPRequestAction_Execute_NonJSONResponse(t *testing.T) {
	// Create test server that returns plain text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:      "test-text",
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
	assert.Equal(t, "Hello, World!", resultMap["body"]) // Should be string since JSON parsing failed
}

func TestHTTPRequestAction_Execute_POST_WithBody(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Read and verify body
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)
		assert.Equal(t, "test value", requestBody["key"])

		response := map[string]interface{}{"created": true}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:     "test-post",
		Method: "POST",
		URL:    server.URL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:    `{"key": "test value"}`,
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 201, resultMap["status_code"])

	body := resultMap["body"].(map[string]interface{})
	assert.True(t, body["created"].(bool))
}

func TestHTTPRequestAction_Execute_ServerError_WithRetry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Server Error"))
			return
		}
		// Third call succeeds
		response := map[string]interface{}{"success": true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:      "test-retry",
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 3, Delay: 0}, // No delay for faster test
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err)
	assert.Equal(t, 3, callCount) // Should have retried 3 times

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 200, resultMap["status_code"])
}

func TestHTTPRequestAction_Execute_AllRetriesFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server Error"))
	}))
	defer server.Close()

	action := &HTTPRequestAction{
		ID:      "test-fail",
		Method:  "GET",
		URL:     server.URL,
		Headers: map[string]string{},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 2, Delay: 0},
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

	require.NoError(t, err) // Should not error, but return the error response

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 500, resultMap["status_code"])
	assert.Equal(t, "Server Error", resultMap["body"])
}

func TestHTTPRequestAction_Execute_InvalidURL(t *testing.T) {
	action := &HTTPRequestAction{
		ID:      "test-invalid",
		Method:  "GET",
		URL:     "://invalid-url",
		Headers: map[string]string{},
		Timeout: 30 * time.Second,
		Retry:   RetryConfig{Attempts: 1, Delay: 0},
	}

	logger := log.WithField("test", "http_action")
	execCtx := models.ExecutionContext{Logger: logger}

	result, err := action.Execute(context.Background(), execCtx)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "all retry attempts failed")
}
