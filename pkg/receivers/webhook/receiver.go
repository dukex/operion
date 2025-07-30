// Package webhook provides webhook-based receiver implementation for the receiver pattern.
package webhook

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/protocol"
	existing_webhook "github.com/dukex/operion/pkg/triggers/webhook"
)

const TriggerTopic = "tp_trigger"

type WebhookReceiver struct {
	sources       []protocol.SourceConfig
	eventBus      eventbus.EventBus
	logger        *slog.Logger
	serverManager *existing_webhook.WebhookServerManager
	handlers      map[string]*receiverWebhookHandler
	config        protocol.ReceiverConfig
	port          int
}

// EndpointConfig represents configuration for a webhook endpoint
type EndpointConfig struct {
	Path            string            `json:"path"`
	Method          string            `json:"method"`
	ExpectedHeaders map[string]string `json:"expected_headers"`
}

// NewWebhookReceiver creates a new webhook receiver
func NewWebhookReceiver(eventBus eventbus.EventBus, logger *slog.Logger, port int) *WebhookReceiver {
	return &WebhookReceiver{
		eventBus: eventBus,
		logger:   logger.With("module", "webhook_receiver"),
		handlers: make(map[string]*receiverWebhookHandler),
		port:     port,
	}
}

func (r *WebhookReceiver) Configure(config protocol.ReceiverConfig) error {
	r.config = config

	// Filter webhook sources
	r.sources = make([]protocol.SourceConfig, 0)
	for _, source := range config.Sources {
		if source.Type == "webhook" {
			r.sources = append(r.sources, source)
		}
	}

	return r.Validate()
}

func (r *WebhookReceiver) Validate() error {
	if len(r.sources) == 0 {
		return errors.New("no webhook sources configured")
	}

	for _, source := range r.sources {
		if source.Name == "" {
			return errors.New("webhook source name is required")
		}

		endpoints, ok := source.Configuration["endpoints"]
		if !ok {
			return fmt.Errorf("endpoints configuration required for webhook source %s", source.Name)
		}

		endpointsList, ok := endpoints.([]interface{})
		if !ok {
			return fmt.Errorf("endpoints must be a list for webhook source %s", source.Name)
		}

		if len(endpointsList) == 0 {
			return fmt.Errorf("at least one endpoint required for webhook source %s", source.Name)
		}

		// Validate each endpoint
		for _, ep := range endpointsList {
			epMap, ok := ep.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid endpoint configuration for webhook source %s", source.Name)
			}

			path, ok := epMap["path"].(string)
			if !ok || path == "" {
				return fmt.Errorf("endpoint path is required for webhook source %s", source.Name)
			}

			if path[0] != '/' {
				return fmt.Errorf("endpoint path must start with '/' for webhook source %s", source.Name)
			}
		}
	}

	return nil
}

func (r *WebhookReceiver) Start(ctx context.Context) error {
	r.logger.Info("Starting webhook receiver", "sources_count", len(r.sources), "port", r.port)

	// Get or create webhook server manager
	r.serverManager = existing_webhook.GetWebhookServerManager(r.port, r.logger)
	if r.serverManager == nil {
		return errors.New("failed to get webhook server manager")
	}

	// Start the server manager
	if err := r.serverManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start webhook server manager: %w", err)
	}

	// Register handlers for all configured endpoints
	for _, source := range r.sources {
		if err := r.registerSourceEndpoints(source); err != nil {
			r.logger.Error("Failed to register endpoints for source", "source", source.Name, "error", err)
			return err
		}
	}

	return nil
}

func (r *WebhookReceiver) registerSourceEndpoints(source protocol.SourceConfig) error {
	logger := r.logger.With("source", source.Name)
	logger.Info("Registering webhook endpoints for source")

	endpoints, _ := source.Configuration["endpoints"].([]interface{})

	for _, ep := range endpoints {
		epMap := ep.(map[string]interface{})
		
		path := epMap["path"].(string)
		method := getStringOrDefault(epMap, "method", "POST")
		
		// Convert expected headers
		expectedHeaders := make(map[string]string)
		if headerConfig, exists := epMap["expected_headers"]; exists {
			if headerMap, ok := headerConfig.(map[string]interface{}); ok {
				for k, v := range headerMap {
					expectedHeaders[k] = fmt.Sprintf("%v", v)
				}
			}
		}

		// Create handler
		handler := &receiverWebhookHandler{
			receiver:        r,
			source:          source,
			path:            path,
			method:          method,
			expectedHeaders: expectedHeaders,
			logger:          logger.With("path", path),
		}

		// Create webhook handler compatible with existing system
		webhookHandler := &existing_webhook.WebhookHandler{
			TriggerID: fmt.Sprintf("%s-%s", source.Name, path),
			Callback:  handler.handleRequest,
			Logger:    handler.logger,
		}

		// Register with server manager
		if err := r.serverManager.RegisterWebhook(path, webhookHandler); err != nil {
			return fmt.Errorf("failed to register webhook path %s for source %s: %w", path, source.Name, err)
		}

		r.handlers[path] = handler
		logger.Info("Registered webhook endpoint", "path", path, "method", method)
	}

	return nil
}

func (r *WebhookReceiver) Stop(ctx context.Context) error {
	r.logger.Info("Stopping webhook receiver")

	// Unregister all handlers
	for path := range r.handlers {
		r.serverManager.UnregisterWebhook(path)
	}

	return nil
}

// Helper function
func getStringOrDefault(m map[string]interface{}, key, defaultValue string) string {
	if val, exists := m[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// receiverWebhookHandler handles webhook requests for the receiver
type receiverWebhookHandler struct {
	receiver        *WebhookReceiver
	source          protocol.SourceConfig
	path            string
	method          string
	expectedHeaders map[string]string
	logger          *slog.Logger
}

// handleRequest processes webhook requests and publishes trigger events
func (h *receiverWebhookHandler) handleRequest(ctx context.Context, triggerData map[string]interface{}) error {
	h.logger.Debug("Processing webhook request")

	// Validate method if specified
	if h.method != "" {
		if method, ok := triggerData["method"].(string); ok && method != h.method {
			h.logger.Warn("Request method doesn't match expected method", "expected", h.method, "actual", method)
			return fmt.Errorf("method %s not allowed, expected %s", method, h.method)
		}
	}

	// Validate expected headers if configured
	if len(h.expectedHeaders) > 0 {
		headers, _ := triggerData["headers"].(map[string]interface{})
		for expectedKey, expectedValue := range h.expectedHeaders {
			if actualValue, exists := headers[expectedKey]; !exists || fmt.Sprintf("%v", actualValue) != expectedValue {
				h.logger.Warn("Expected header not found or doesn't match", "header", expectedKey, "expected", expectedValue)
				return fmt.Errorf("header %s doesn't match expected value", expectedKey)
			}
		}
	}

	// Create original data (raw webhook data)
	originalData := make(map[string]interface{})
	for k, v := range triggerData {
		originalData[k] = v
	}

	// Create transformed trigger data for workflow matching
	transformedTriggerData := map[string]interface{}{
		"path":   h.path,
		"method": triggerData["method"],
		"body":   triggerData["body"],
	}

	// Create and publish trigger event
	triggerEvent := events.NewTriggerEvent("webhook", h.source.Name, transformedTriggerData, originalData)

	// Publish to trigger topic
	if err := h.receiver.eventBus.Publish(ctx, TriggerTopic, triggerEvent); err != nil {
		h.logger.Error("Failed to publish trigger event", "error", err)
		return err
	}

	h.logger.Debug("Published webhook trigger event", "source", h.source.Name, "path", h.path)
	return nil
}