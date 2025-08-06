package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/protocol"
)

var (
	globalServerManager *WebhookServerManager
	once                sync.Once
	resetMu             sync.Mutex
)

// GetGlobalWebhookServerManager returns the global singleton instance of the WebhookServerManager.
func GetGlobalWebhookServerManager() *WebhookServerManager {
	resetMu.Lock()
	defer resetMu.Unlock()

	return globalServerManager
}

type WebhookServerManager struct {
	server   *http.Server
	handlers map[string]*Handler
	mu       sync.RWMutex
	logger   *slog.Logger
	port     int
	started  bool
	done     chan struct{}
	doneOnce sync.Once
}

// GetWebhookServerManager returns the singleton instance of the WebhookServerManager.
func GetWebhookServerManager(port int, logger *slog.Logger) *WebhookServerManager {
	resetMu.Lock()
	defer resetMu.Unlock()

	once.Do(func() {
		globalServerManager = &WebhookServerManager{
			handlers: make(map[string]*Handler),
			logger:   logger.With("module", "webhook_server_manager"),
			port:     port,
			done:     make(chan struct{}),
		}
	})

	return globalServerManager
}

// RegisterWebhook registers a new webhook handler for the specified path.
func (sm *WebhookServerManager) RegisterWebhook(ctx context.Context, path string, handler *Handler) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, exists := sm.handlers[path]
	if exists {
		return fmt.Errorf("webhook path %s already registered", path)
	}

	sm.handlers[path] = handler
	sm.logger.InfoContext(ctx, "Registered webhook handler", "path", path, "trigger_id", handler.TriggerID)

	return nil
}

func (sm *WebhookServerManager) UnregisterWebhook(ctx context.Context, path string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	handler, exists := sm.handlers[path]
	if exists {
		delete(sm.handlers, path)
		sm.logger.InfoContext(ctx, "Unregistered webhook handler", "path", path, "trigger_id", handler.TriggerID)
	}
}

const (
	webhookReadTimeout     = 30 * time.Second
	webhookWriteTimeout    = 30 * time.Second
	webhookIdleTimeout     = 60 * time.Second
	webhookShutdownTimeout = 5 * time.Second
)

func (sm *WebhookServerManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.started {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", sm.handleWebhook)

	sm.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", sm.port),
		Handler:      mux,
		ReadTimeout:  webhookReadTimeout,
		WriteTimeout: webhookWriteTimeout,
		IdleTimeout:  webhookIdleTimeout,
	}

	go func() {
		sm.logger.InfoContext(ctx, "Starting webhook HTTP server", "addr", sm.server.Addr)

		err := sm.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			sm.logger.ErrorContext(ctx, "Failed to start webhook server", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()

		err := sm.Stop(ctx)
		if err != nil {
			sm.logger.ErrorContext(ctx, "Failed to stop webhook server", "error", err)
		}
	}()

	sm.started = true
	sm.logger.InfoContext(ctx, "Webhook server manager started")

	return nil
}

func (sm *WebhookServerManager) handleWebhook(
	response http.ResponseWriter,
	request *http.Request,
) {
	ctx := request.Context()

	sm.mu.RLock()
	handler, exists := sm.handlers[request.URL.Path]
	sm.mu.RUnlock()

	if !exists {
		sm.logger.WarnContext(ctx, "No handler found for webhook path", "path", request.URL.Path)
		http.Error(response, "Webhook path not found", http.StatusNotFound)

		return
	}

	handler.Logger.InfoContext(ctx, "Received webhook request", "method", request.Method, "path", request.URL.Path)

	body, err := io.ReadAll(request.Body)
	if err != nil {
		handler.Logger.ErrorContext(ctx, "Failed to read request body", "error", err)
		http.Error(response, "Failed to read request body", http.StatusBadRequest)

		return
	}

	defer func() {
		err := request.Body.Close()
		if err != nil {
			handler.Logger.ErrorContext(ctx, "Failed to close request body", "error", err)
		}
	}()

	var bodyData any
	if len(body) > 0 {
		err := json.Unmarshal(body, &bodyData)
		if err != nil {
			handler.Logger.WarnContext(ctx, "Failed to parse JSON body, using raw string", "error", err)

			bodyData = string(body)
		}
	}

	headers := make(map[string]any)

	for name, values := range request.Header {
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = values
		}
	}

	query := make(map[string]any)

	for name, values := range request.URL.Query() {
		if len(values) == 1 {
			query[name] = values[0]
		} else {
			query[name] = values
		}
	}

	triggerData := map[string]any{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"method":      request.Method,
		"path":        request.URL.Path,
		"query":       query,
		"headers":     headers,
		"body":        bodyData,
		"remote_addr": request.RemoteAddr,
	}

	go func() {
		err := handler.Callback(ctx, triggerData)
		if err != nil {
			handler.Logger.ErrorContext(ctx, "Error executing workflow for webhook trigger", "error", err)
		}
	}()

	response.Header().Set("Content-Type", "application/json")

	response.WriteHeader(http.StatusOK)

	err = json.NewEncoder(response).Encode(map[string]any{
		"status":  "success",
		"message": "webhook received",
	})
	if err != nil {
		handler.Logger.ErrorContext(ctx, "Failed to encode response", "error", err)
	}
}

func (sm *WebhookServerManager) Stop(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started || sm.server == nil {
		return nil
	}

	sm.logger.InfoContext(ctx, "Stopping webhook server manager")

	shutdownCtx, cancel := context.WithTimeout(ctx, webhookShutdownTimeout)

	defer cancel()

	err := sm.server.Shutdown(shutdownCtx)
	if err != nil {
		sm.logger.ErrorContext(ctx, "Error shutting down webhook server", "error", err)

		return err
	}

	sm.started = false
	sm.doneOnce.Do(func() {
		close(sm.done)
	})
	sm.logger.InfoContext(ctx, "Webhook server manager stopped")

	return nil
}

func (sm *WebhookServerManager) Done() <-chan struct{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.done
}

// ResetGlobalManager resets the global manager (for testing purposes).
func ResetGlobalManager() {
	resetMu.Lock()
	defer resetMu.Unlock()

	once = sync.Once{}
	globalServerManager = nil
}

// Handler handles incoming webhook requests.
type Handler struct {
	TriggerID string
	Callback  protocol.TriggerCallback
	Logger    *slog.Logger
}
