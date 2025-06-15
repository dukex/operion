package http_action

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dukex/operion/internal/domain"
)

// HTTPRequestAction performs an HTTP request
type HTTPRequestAction struct {
	ID      string
	Method  string
	URL     string
	Headers map[string]string
	Body    string
	Timeout time.Duration
	Retry   RetryConfig
}

type RetryConfig struct {
	Attempts int
	Delay    int // seconds
}

func NewHTTPRequestAction(config map[string]interface{}) (*HTTPRequestAction, error) {
	id, _ := config["id"].(string)
	method, _ := config["method"].(string)
	url, _ := config["url"].(string)
	body, _ := config["body"].(string)

	// If no ID is provided in config, it should be passed from the action item
	if id == "" {
		id = "http_request_action"
	}

	// Parse headers
	headers := make(map[string]string)
	if headersConfig, exists := config["headers"]; exists {
		if headersMap, ok := headersConfig.(map[string]interface{}); ok {
			for k, v := range headersMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	// Parse retry config
	retry := RetryConfig{Attempts: 1, Delay: 0}
	if retryConfig, exists := config["retry"]; exists {
		if retryMap, ok := retryConfig.(map[string]interface{}); ok {
			if attempts, ok := retryMap["attempts"].(float64); ok {
				retry.Attempts = int(attempts)
			}
			if delay, ok := retryMap["delay"].(float64); ok {
				retry.Delay = int(delay)
			}
		}
	}

	// Default values
	if method == "" {
		method = http.MethodGet
	}

	return &HTTPRequestAction{
		ID:      id,
		Method:  strings.ToUpper(method),
		URL:     url,
		Headers: headers,
		Body:    body,
		Timeout: 30 * time.Second,
		Retry:   retry,
	}, nil
}

func (a *HTTPRequestAction) GetID() string                     { return a.ID }
func (a *HTTPRequestAction) GetType() string                   { return "http" }
func (a *HTTPRequestAction) GetConfig() map[string]interface{} { /* ... */ return nil }
func (a *HTTPRequestAction) Validate() error                   { /* ... */ return nil }

func (a *HTTPRequestAction) Execute(ctx context.Context, input domain.ExecutionContext) (domain.ExecutionContext, error) {
	log.Printf("Executing HTTPRequestAction '%s' %s to URL '%s'", a.ID, a.Method, a.URL)

	var lastErr error
	var resp *http.Response

	// Retry logic
	for attempt := 1; attempt <= a.Retry.Attempts; attempt++ {
		if attempt > 1 {
			log.Printf("HTTPRequestAction '%s' retry attempt %d/%d", a.ID, attempt, a.Retry.Attempts)
			time.Sleep(time.Duration(a.Retry.Delay) * time.Second)
		}

		// Create request context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, a.Timeout)

		// Create request body
		var bodyReader io.Reader
		if a.Body != "" {
			bodyReader = strings.NewReader(a.Body)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(reqCtx, a.Method, a.URL, bodyReader)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create http request: %w", err)
			continue
		}

		// Add headers
		for key, value := range a.Headers {
			req.Header.Set(key, value)
		}

		// Execute the request
		client := &http.Client{}
		resp, err = client.Do(req)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			continue
		}

		// Check if we should retry based on status code
		if resp.StatusCode >= 500 && attempt < a.Retry.Attempts {
			resp.Body.Close()
			lastErr = fmt.Errorf("server error (status %d), retrying", resp.StatusCode)
			continue
		}

		// Success or non-retryable error
		break
	}

	if resp == nil {
		return input, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return input, fmt.Errorf("failed to read response body: %w", err)
	}

	// Add results to the ExecutionContext
	if input.StepResults == nil {
		input.StepResults = make(map[string]interface{})
	}
	input.StepResults[a.ID] = map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(bodyBytes),
		"headers":     resp.Header,
	}

	log.Printf("HTTPRequestAction '%s' completed with status %d, body length: %d", a.ID, resp.StatusCode, len(bodyBytes))
	return input, nil
}
