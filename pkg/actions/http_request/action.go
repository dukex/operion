package http_request

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/template"
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
	Delay    int
}

func NewHTTPRequestAction(config map[string]interface{}) (*HTTPRequestAction, error) {
	id, _ := config["id"].(string)
	method, _ := config["method"].(string)
	url, _ := config["url"].(string)
	body, _ := config["body"].(string)

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

func (a *HTTPRequestAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (interface{}, error) {
	logger = logger.With(
		"module", "http_request_action",
	)
	logger.Info("Executing HTTPRequestAction")

	var lastErr error
	var resp *http.Response

	for attempt := 1; attempt <= a.Retry.Attempts; attempt++ {
		if attempt > 1 {
			logger.Info(fmt.Sprintf("HTTPRequestAction retry attempt %d/%d", attempt, a.Retry.Attempts))
			time.Sleep(time.Duration(a.Retry.Delay) * time.Second)
		}

		reqCtx, cancel := context.WithTimeout(ctx, a.Timeout)

		var bodyReader io.Reader
		if a.Body != "" {
			body, err := template.Render(a.Body, executionCtx.StepResults)
			if err != nil {
				cancel()
				return nil, fmt.Errorf("failed to render body template: %w", err)
			}

			bodyMarshalled, err := json.Marshal(body)

			if err != nil {
				cancel()
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}

			bodyReader = strings.NewReader(string(bodyMarshalled))
		}

		logger.Debug("Creating HTTP request: %s %s", "method", a.Method, "url", a.URL)
		req, err := http.NewRequestWithContext(reqCtx, a.Method, a.URL, bodyReader)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create http request: %w", err)
			continue
		}

		for key, value := range a.Headers {
			req.Header.Set(key, value)
		}

		client := &http.Client{}
		resp, err = client.Do(req)
		cancel()

		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)
			continue
		}

		if resp.StatusCode >= 500 && attempt < a.Retry.Attempts {
			err = resp.Body.Close()
			if err != nil {
				logger.Error("failed to close response body", "error", err)
			}

			lastErr = fmt.Errorf("server error (status %d), retrying", resp.StatusCode)
			continue
		}

		break
	}

	if resp == nil {
		return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var body interface{}
	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		body = string(bodyBytes)
		logger.Warn("Failed to parse response as JSON, returning as string", "error", err)
	}

	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        body,
		"headers":     resp.Header,
	}

	logger.Info(fmt.Sprintf("HTTPRequestAction completed with status %d, body length: %d", resp.StatusCode, len(bodyBytes)))
	return result, nil
}
