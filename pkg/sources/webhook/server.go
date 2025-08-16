package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/sources/webhook/models"
	"github.com/xeipuuv/gojsonschema"
)

const (
	// Server configuration constants.
	webhookReadTimeout     = 30 * time.Second
	webhookWriteTimeout    = 30 * time.Second
	webhookIdleTimeout     = 60 * time.Second
	webhookShutdownTimeout = 5 * time.Second
	maxRequestBodySize     = 1024 * 1024 // 1MB max request body
)

// WebhookServer manages the HTTP server for webhook requests.
type WebhookServer struct {
	server      *http.Server
	port        int
	persistence WebhookPersistence // Interface for webhook persistence
	callback    protocol.SourceEventCallback
	logger      *slog.Logger
	mu          sync.RWMutex
	started     bool
	done        chan struct{}
	doneOnce    sync.Once
}

// WebhookPersistence defines minimal interface needed by server for webhook operations.
type WebhookPersistence interface {
	WebhookSourceByExternalID(externalID string) (*models.WebhookSource, error)
	WebhookSources() ([]*models.WebhookSource, error)
}

// NewWebhookServer creates a new webhook server instance.
func NewWebhookServer(port int, logger *slog.Logger) *WebhookServer {
	return &WebhookServer{
		port:   port,
		logger: logger.With("module", "webhook_server", "port", port),
		done:   make(chan struct{}),
	}
}

// SetPersistence sets the persistence layer for webhook source lookups.
func (s *WebhookServer) SetPersistence(persistence WebhookPersistence) {
	s.persistence = persistence
}

// RegisterSource logs webhook source registration (sources are now managed via persistence).
func (s *WebhookServer) RegisterSource(source *models.WebhookSource) error {
	s.logger.Info("Webhook source available for requests",
		"source_id", source.ID,
		"external_id", source.ExternalID,
		"url", source.GetWebhookURL())

	return nil
}

// UnregisterSource logs webhook source unregistration (sources are now managed via persistence).
func (s *WebhookServer) UnregisterSource(uuid string) {
	s.logger.Info("Webhook source unregistered", "external_id", uuid)
}

// SetCallback sets the callback function for publishing source events.
func (s *WebhookServer) SetCallback(callback protocol.SourceEventCallback) {
	s.callback = callback
}

// Start starts the HTTP server and begins handling webhook requests.
func (s *WebhookServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/", s.handleWebhook)
	mux.HandleFunc("/health", s.handleHealth)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  webhookReadTimeout,
		WriteTimeout: webhookWriteTimeout,
		IdleTimeout:  webhookIdleTimeout,
	}

	s.started = true
	s.logger.Info("Starting webhook server", "addr", s.server.Addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Webhook server error", "error", err)
		}
	}()

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		s.shutdown(ctx)
	}()

	return nil
}

// Stop gracefully shuts down the webhook server.
func (s *WebhookServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.logger.Info("Stopping webhook server")

	shutdownCtx, cancel := context.WithTimeout(ctx, webhookShutdownTimeout)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Error during server shutdown", "error", err)

		return err
	}

	s.started = false
	s.doneOnce.Do(func() {
		close(s.done)
	})

	s.logger.Info("Webhook server stopped successfully")

	return nil
}

// Done returns a channel that's closed when the server is shut down.
func (s *WebhookServer) Done() <-chan struct{} {
	return s.done
}

// shutdown performs internal shutdown logic.
func (s *WebhookServer) shutdown(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, webhookShutdownTimeout)
	defer cancel()

	if err := s.Stop(shutdownCtx); err != nil {
		s.logger.Error("Error during webhook server shutdown", "error", err)
	}
}

// handleWebhook handles incoming webhook requests.
func (s *WebhookServer) handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Extract UUID from path
	uuid := strings.TrimPrefix(r.URL.Path, "/webhook/")
	if uuid == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Missing webhook UUID in path")

		return
	}

	// Only allow POST requests
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, http.StatusMethodNotAllowed, "Only POST method allowed")

		return
	}

	// Find webhook source by UUID from persistence
	source, err := s.persistence.WebhookSourceByExternalID(uuid)
	if err != nil {
		s.logger.Error("Error checking persistence for webhook UUID", "uuid", uuid, "error", err)
		s.writeErrorResponse(w, http.StatusInternalServerError, "Error processing webhook")

		return
	}

	if source == nil {
		s.logger.Warn("Webhook request for unknown UUID", "uuid", uuid, "remote_addr", r.RemoteAddr)
		s.writeErrorResponse(w, http.StatusNotFound, "Webhook not found")

		return
	}

	// Check if source is active
	if !source.Active {
		s.logger.Warn("Webhook request for inactive source", "source_id", source.ID, "external_id", uuid)
		s.writeErrorResponse(w, http.StatusNotFound, "Webhook not found")

		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	// Read and parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Error reading request body", "source_id", source.ID, "error", err)
		s.writeErrorResponse(w, http.StatusBadRequest, "Error reading request body")

		return
	}

	// Parse JSON body
	var eventData map[string]any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &eventData); err != nil {
			s.logger.Error("Error parsing JSON body", "source_id", source.ID, "error", err)
			s.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON in request body")

			return
		}
	} else {
		eventData = make(map[string]any)
	}

	// Validate against JSON schema if configured
	if source.HasJSONSchema() {
		if err := s.validateJSONSchema(eventData, source.JSONSchema); err != nil {
			s.logger.Warn("JSON schema validation failed", "source_id", source.ID, "error", err)
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Schema validation failed: %v", err))

			return
		}
	}

	// Add request metadata to event data
	enrichedEventData := s.enrichEventData(eventData, r)

	// Publish source event if callback is available
	if s.callback != nil {
		ctx := r.Context()
		if err := s.callback(ctx, source.ID, "webhook", "WebhookReceived", enrichedEventData); err != nil {
			s.logger.Error("Error publishing source event", "source_id", source.ID, "error", err)
			s.writeErrorResponse(w, http.StatusInternalServerError, "Error processing webhook")

			return
		}
	}

	// Log successful webhook processing
	s.logger.Info("Webhook processed successfully",
		"source_id", source.ID,
		"external_id", uuid,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"content_length", r.ContentLength)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Webhook received and processed",
	}); err != nil {
		s.logger.Error("Error encoding success response", "error", err)
	}
}

// handleHealth handles health check requests.
func (s *WebhookServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Get webhook count from persistence for health check
	var hookCount int

	if s.persistence != nil {
		if sources, err := s.persistence.WebhookSources(); err == nil {
			hookCount = len(sources)
		}
	}

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":           "healthy",
		"registered_hooks": hookCount,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		s.logger.Error("Error encoding health response", "error", err)
	}
}

// validateJSONSchema validates event data against the provided JSON schema.
func (s *WebhookServer) validateJSONSchema(eventData map[string]any, schema map[string]any) error {
	schemaLoader := gojsonschema.NewGoLoader(schema)
	dataLoader := gojsonschema.NewGoLoader(eventData)

	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}

		return fmt.Errorf("validation errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// enrichEventData adds request metadata to the event data.
func (s *WebhookServer) enrichEventData(originalData map[string]any, r *http.Request) map[string]any {
	enriched := map[string]any{
		"webhook": map[string]any{
			"method":         r.Method,
			"url":            r.URL.String(),
			"remote_addr":    r.RemoteAddr,
			"user_agent":     r.UserAgent(),
			"content_length": r.ContentLength,
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"headers":        s.extractHeaders(r),
			"query_params":   s.extractQueryParams(r),
		},
		"body": originalData,
	}

	return enriched
}

// extractHeaders extracts HTTP headers from the request.
func (s *WebhookServer) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	for name, values := range r.Header {
		if len(values) > 0 {
			// Join multiple values with comma (standard HTTP behavior)
			headers[name] = strings.Join(values, ", ")
		}
	}

	return headers
}

// extractQueryParams extracts query parameters from the request.
func (s *WebhookServer) extractQueryParams(r *http.Request) map[string]string {
	params := make(map[string]string)

	for name, values := range r.URL.Query() {
		if len(values) > 0 {
			// Use the first value for each parameter
			params[name] = values[0]
		}
	}

	return params
}

// writeErrorResponse writes a JSON error response.
func (s *WebhookServer) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status":  "error",
		"message": message,
		"code":    statusCode,
	}); err != nil {
		s.logger.Error("Error encoding error response", "error", err)
	}
}
