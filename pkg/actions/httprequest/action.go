// Package httprequest provides HTTP request action implementation for workflow steps.
package httprequest

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
	ID       string
	Method   string
	Protocol string
	Host     string
	Path     string
	Headers  map[string]string
	Body     string
	Timeout  time.Duration
	Retry    RetryConfig
}

type RetryConfig struct {
	Attempts int
	Delay    int
}

func NewHTTPRequestAction(config map[string]any) (*HTTPRequestAction, error) {
	id, _ := config["id"].(string)
	method, _ := config["method"].(string)
	host, ok := config["host"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'host' in configuration")
	}
	path, _ := config["path"].(string)
	if len(path) == 0 {
		path = "/"
	}
	protocol, _ := config["protocol"].(string)
	if protocol == "" {
		protocol = "http"
	}
	body, _ := config["body"].(string)

	headers := make(map[string]string)
	if headersConfig, exists := config["headers"]; exists {
		if headersMap, ok := headersConfig.(map[string]any); ok {
			for k, v := range headersMap {
				if strVal, ok := v.(string); ok {
					headers[k] = strVal
				}
			}
		}
	}

	retry := RetryConfig{Attempts: 1, Delay: 0}
	if retryConfig, exists := config["retry"]; exists {
		if retryMap, ok := retryConfig.(map[string]any); ok {
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
		ID:       id,
		Method:   strings.ToUpper(method),
		Protocol: protocol,
		Host:     host,
		Path:     path,
		Headers:  headers,
		Body:     body,
		Timeout:  30 * time.Second,
		Retry:    retry,
	}, nil
}

func (a *HTTPRequestAction) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (any, error) {
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
			body, err := template.RenderWithContext(a.Body, &executionCtx)
			if err != nil {
				cancel()
				return nil, fmt.Errorf("failed to render body template: %w", err)
			}

			var bodyBytes []byte
			if str, ok := body.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(body)
				if err != nil {
					cancel()
					return nil, fmt.Errorf("failed to marshal body: %w", err)
				}
			}

			bodyReader = strings.NewReader(string(bodyBytes))
		}

		pathResult, err := template.RenderWithContext(a.Path, &executionCtx)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to render path template: %w", err)
		}
		path := fmt.Sprintf("%v", pathResult)

		url := fmt.Sprintf("%s://%s%s", a.Protocol, a.Host, path)

		logger.Debug("Creating HTTP request: %s %s", "method", a.Method, "url", url)

		req, err := http.NewRequestWithContext(reqCtx, a.Method, url, bodyReader)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("failed to create http request: %w", err)
			continue
		}

		for key, value := range a.Headers {
			headerResult, err := template.RenderWithContext(value, &executionCtx)
			if err != nil {
				cancel()
				lastErr = fmt.Errorf("failed to render header '%s' template: %w", key, err)
				continue
			}
			headerValue := fmt.Sprintf("%v", headerResult)

			req.Header.Set(key, headerValue)
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
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var body any
	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		body = string(bodyBytes)
		logger.Warn("Failed to parse response as JSON, returning as string", "error", err)
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"body":        body,
		"headers":     resp.Header,
	}

	logger.Info(fmt.Sprintf("HTTPRequestAction completed with status %d, body length: %d", resp.StatusCode, len(bodyBytes)))
	return result, nil
}
