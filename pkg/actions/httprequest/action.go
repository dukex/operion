// Package httprequest provides HTTP request action implementation for workflow steps.
package httprequest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/template"
)

const defaultTimeoutSeconds = 30

// Action performs an HTTP request to a specified URL with optional headers, body, and retry logic.
type Action struct {
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

// RetryConfig defines retry behavior for HTTP requests.
type RetryConfig struct {
	Attempts int
	Delay    int
}

// NewAction creates a new HTTPRequestAction from configuration.
func NewAction(config map[string]any) (*Action, error) {
	actionID, _ := config["id"].(string)
	method, _ := config["method"].(string)

	host, ok := config["host"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'host' in configuration: %w", ErrHTTPRequestHostInvalid)
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

	retryConfig, exists := config["retry"]
	if exists {
		retry = parseRetryConfig(retryConfig)
	}

	if method == "" {
		method = http.MethodGet
	}

	return &Action{
		ID:       actionID,
		Method:   strings.ToUpper(method),
		Protocol: protocol,
		Host:     host,
		Path:     path,
		Headers:  headers,
		Body:     body,
		Timeout:  defaultTimeoutSeconds * time.Second,
		Retry:    retry,
	}, nil
}

func parseRetryConfig(retryConfig any) RetryConfig {
	retry := RetryConfig{Attempts: 1, Delay: 0}

	retryMap, ok := retryConfig.(map[string]any)
	if !ok {
		return retry
	}

	if attempts, ok := retryMap["attempts"].(float64); ok {
		retry.Attempts = int(attempts)
	}

	if delay, ok := retryMap["delay"].(float64); ok {
		retry.Delay = int(delay)
	}

	return retry
}

// Execute performs the HTTP request with retry logic and returns the response.
func (a *Action) Execute(ctx context.Context, executionCtx models.ExecutionContext, logger *slog.Logger) (any, error) {
	logger = logger.With(
		"module", "http_request_action",
	)
	logger.InfoContext(ctx, "Executing HTTPRequestAction")

	var (
		lastErr error
		resp    *http.Response
	)

	for attempt := 1; attempt <= a.Retry.Attempts; attempt++ {
		if attempt > 1 {
			logger.InfoContext(ctx, fmt.Sprintf("HTTPRequestAction retry attempt %d/%d", attempt, a.Retry.Attempts))
			time.Sleep(time.Duration(a.Retry.Delay) * time.Second)
		}

		req, err := a.buildRequest(ctx, executionCtx, logger)
		if err != nil {
			lastErr = err

			continue
		}

		client := &http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       a.Timeout,
		}

		resp, err = client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http request failed: %w", err)

			continue
		}

		if resp.StatusCode >= 500 && attempt < a.Retry.Attempts {
			err = resp.Body.Close()
			if err != nil {
				logger.ErrorContext(ctx, "failed to close response body", "error", err)
			}

			lastErr = fmt.Errorf("server error (status %d), retrying: %w", resp.StatusCode, ErrHTTPServerError)

			continue
		}

		break
	}

	if resp == nil {
		return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
	}

	return a.processResponse(ctx, resp, logger)
}

var (
	// ErrHTTPMethodInvalid is returned when the HTTP method is invalid.
	ErrHTTPMethodInvalid = errors.New("invalid HTTP method")
	// ErrHTTPRequestPathInvalid is returned when the HTTP request path is invalid.
	ErrHTTPRequestPathInvalid = errors.New("invalid HTTP request path")
	// ErrHTTPRequestHostInvalid is returned when the HTTP request host is invalid.
	ErrHTTPRequestHostInvalid = errors.New("invalid HTTP request host")
	// ErrHTTPServerError is returned when the server returns an error status code.
	ErrHTTPServerError = errors.New("server error during HTTP request")
	// HTTPRequestRetryAttemptsInvalid = errors.New("invalid HTTP request retry attempts")
	// HTTPRequestRetryDelayInvalid    = errors.New("invalid HTTP request retry delay")
	// HTTPRequestTimeoutInvalid       = errors.New("invalid HTTP request timeout")
	// HTTPRequestBodyInvalid          = errors.New("invalid HTTP request body")
	// HTTPRequestHeadersInvalid       = errors.New("invalid HTTP request headers").
)

// Validate checks if the HTTPRequestAction has valid configuration.
func (a *Action) Validate(_ context.Context) error {
	if a.Method == "" {
		return ErrHTTPMethodInvalid
	}

	if a.Host == "" {
		return ErrHTTPRequestHostInvalid
	}

	_, err := template.Parse(a.Body)
	if err != nil {
		return fmt.Errorf("invalid body template: %w", err)
	}

	for key, value := range a.Headers {
		_, err := template.Parse(value)
		if err != nil {
			return fmt.Errorf("invalid header '%s' template: %w", key, err)
		}
	}

	_, err = template.Parse(a.Path)
	if err != nil {
		return fmt.Errorf("invalid path template: %w", err)
	}

	return nil
}

func (a *Action) buildRequest(
	ctx context.Context,
	executionCtx models.ExecutionContext,
	logger *slog.Logger,
) (*http.Request, error) {
	bodyReader, err := a.buildRequestBody(executionCtx)
	if err != nil {
		return nil, err
	}

	url, err := a.buildURL(executionCtx)
	if err != nil {
		return nil, err
	}

	logger.DebugContext(ctx, "Creating HTTP request: %s %s", "method", a.Method, "url", url)

	req, err := http.NewRequestWithContext(ctx, a.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	err = a.setRequestHeaders(req, executionCtx)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (a *Action) buildRequestBody(executionCtx models.ExecutionContext) (io.Reader, error) {
	if a.Body == "" {
		return strings.NewReader(""), nil
	}

	body, err := template.RenderWithContext(a.Body, &executionCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to render body template: %w", err)
	}

	var bodyBytes []byte
	if str, ok := body.(string); ok {
		bodyBytes = []byte(str)
	} else {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
	}

	return strings.NewReader(string(bodyBytes)), nil
}

func (a *Action) buildURL(executionCtx models.ExecutionContext) (string, error) {
	pathResult, err := template.RenderWithContext(a.Path, &executionCtx)
	if err != nil {
		return "", fmt.Errorf("failed to render path template: %w", err)
	}

	path := fmt.Sprintf("%v", pathResult)

	return fmt.Sprintf("%s://%s%s", a.Protocol, a.Host, path), nil
}

func (a *Action) setRequestHeaders(req *http.Request, executionCtx models.ExecutionContext) error {
	for key, value := range a.Headers {
		headerResult, err := template.RenderWithContext(value, &executionCtx)
		if err != nil {
			return fmt.Errorf("failed to render header '%s' template: %w", key, err)
		}

		headerValue := fmt.Sprintf("%v", headerResult)
		req.Header.Set(key, headerValue)
	}

	return nil
}

func (a *Action) processResponse(ctx context.Context, resp *http.Response, logger *slog.Logger) (any, error) {
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

		logger.WarnContext(ctx, "Failed to parse response as JSON, returning as string", "error", err)
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"body":        body,
		"headers":     resp.Header,
	}

	logger.InfoContext(ctx, fmt.Sprintf("HTTPRequestAction completed with status %d, body length: %d",
		resp.StatusCode, len(bodyBytes)))

	return result, nil
}
