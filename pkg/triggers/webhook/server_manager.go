package webhook

import (
	"context"
	"encoding/json"
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
)

type WebhookHandler struct {
	TriggerID string
	Callback  protocol.TriggerCallback
	Logger    *slog.Logger
}

type WebhookServerManager struct {
	server   *http.Server
	handlers map[string]*WebhookHandler
	mu       sync.RWMutex
	logger   *slog.Logger
	port     int
	started  bool
	done     chan struct{}
	doneOnce sync.Once
}

func GetWebhookServerManager(port int, logger *slog.Logger) *WebhookServerManager {
	once.Do(func() {
		globalServerManager = &WebhookServerManager{
			handlers: make(map[string]*WebhookHandler),
			logger:   logger.With("module", "webhook_server_manager"),
			port:     port,
			done:     make(chan struct{}),
		}
	})
	return globalServerManager
}

func SetGlobalWebhookServerManager(manager *WebhookServerManager) {
	globalServerManager = manager
}

func GetGlobalWebhookServerManager() *WebhookServerManager {
	return globalServerManager
}

func (sm *WebhookServerManager) RegisterWebhook(path string, handler *WebhookHandler) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.handlers[path]; exists {
		return fmt.Errorf("webhook path %s already registered", path)
	}

	sm.handlers[path] = handler
	sm.logger.Info("Registered webhook handler", "path", path, "trigger_id", handler.TriggerID)
	return nil
}

func (sm *WebhookServerManager) UnregisterWebhook(path string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if handler, exists := sm.handlers[path]; exists {
		delete(sm.handlers, path)
		sm.logger.Info("Unregistered webhook handler", "path", path, "trigger_id", handler.TriggerID)
	}
}

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
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sm.logger.Info("Starting webhook HTTP server", "addr", sm.server.Addr)
		if err := sm.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sm.logger.Error("Failed to start webhook server", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := sm.Stop(context.Background()); err != nil {
			sm.logger.Error("Failed to stop webhook server", "error", err)
		}
	}()

	sm.started = true
	sm.logger.Info("Webhook server manager started")
	return nil
}

func (sm *WebhookServerManager) handleWebhook(w http.ResponseWriter, r *http.Request) {
	sm.mu.RLock()
	handler, exists := sm.handlers[r.URL.Path]
	sm.mu.RUnlock()

	if !exists {
		sm.logger.Warn("No handler found for webhook path", "path", r.URL.Path)
		http.Error(w, "Webhook path not found", http.StatusNotFound)
		return
	}

	handler.Logger.Info("Received webhook request", "method", r.Method, "path", r.URL.Path)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		handler.Logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			handler.Logger.Error("Failed to close request body", "error", err)
		}
	}()

	var bodyData interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &bodyData); err != nil {
			handler.Logger.Warn("Failed to parse JSON body, using raw string", "error", err)
			bodyData = string(body)
		}
	}

	headers := make(map[string]interface{})
	for name, values := range r.Header {
		if len(values) == 1 {
			headers[name] = values[0]
		} else {
			headers[name] = values
		}
	}

	query := make(map[string]interface{})
	for name, values := range r.URL.Query() {
		if len(values) == 1 {
			query[name] = values[0]
		} else {
			query[name] = values
		}
	}

	triggerData := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"method":      r.Method,
		"path":        r.URL.Path,
		"query":       query,
		"headers":     headers,
		"body":        bodyData,
		"remote_addr": r.RemoteAddr,
	}

	go func() {
		if err := handler.Callback(context.Background(), triggerData); err != nil {
			handler.Logger.Error("Error executing workflow for webhook trigger", "error", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "webhook received",
	}); err != nil {
		handler.Logger.Error("Failed to encode response", "error", err)
	}
}

func (sm *WebhookServerManager) Stop(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started || sm.server == nil {
		return nil
	}

	sm.logger.Info("Stopping webhook server manager")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sm.server.Shutdown(shutdownCtx); err != nil {
		sm.logger.Error("Error shutting down webhook server", "error", err)
		return err
	}

	sm.started = false
	sm.doneOnce.Do(func() {
		close(sm.done)
	})
	sm.logger.Info("Webhook server manager stopped")
	return nil
}

func (sm *WebhookServerManager) GetHandlerCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.handlers)
}

func (sm *WebhookServerManager) Done() <-chan struct{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.done
}

// ResetGlobalManager resets the global manager (for testing purposes)
func ResetGlobalManager() {
	once = sync.Once{}
	globalServerManager = nil
}
