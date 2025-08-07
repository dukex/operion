package webhook

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	webhookModels "github.com/dukex/operion/pkg/sources/webhook/models"
	webhookPersistence "github.com/dukex/operion/pkg/sources/webhook/persistence"
)

// WebhookSourceProvider implements a centralized webhook orchestrator that manages
// HTTP webhook endpoints and converts incoming requests to source events.
type WebhookSourceProvider struct {
	config             map[string]any
	logger             *slog.Logger
	callback           protocol.SourceEventCallback
	server             *WebhookServer
	webhookPersistence webhookPersistence.WebhookPersistence
	port               int
	started            bool
	mu                 sync.RWMutex
}

// Start begins the centralized webhook orchestrator.
func (w *WebhookSourceProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.started {
		return nil
	}

	w.callback = callback
	w.logger.Info("Starting centralized webhook orchestrator", "port", w.port)

	// Set callback on server for event publishing
	w.server.SetCallback(callback)

	// Start HTTP server
	if err := w.server.Start(ctx); err != nil {
		return err
	}

	w.started = true

	// Get source count from persistence for logging
	sources, err := w.webhookPersistence.ActiveWebhookSources()
	if err != nil {
		w.logger.Warn("Failed to get active sources count", "error", err)
		w.logger.Info("Centralized webhook orchestrator started successfully", "port", w.port)
	} else {
		w.logger.Info("Centralized webhook orchestrator started successfully",
			"port", w.port,
			"active_sources", len(sources))
	}

	return nil
}

// Stop gracefully shuts down the webhook orchestrator.
func (w *WebhookSourceProvider) Stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.started {
		return nil
	}

	w.logger.Info("Stopping webhook orchestrator")

	// Stop HTTP server
	if err := w.server.Stop(ctx); err != nil {
		w.logger.Error("Error stopping webhook server", "error", err)

		return err
	}

	w.started = false
	w.logger.Info("Webhook orchestrator stopped successfully")

	return nil
}

// Validate checks if the webhook orchestrator configuration is valid.
func (w *WebhookSourceProvider) Validate() error {
	// Orchestrator validation: ensure server is available
	if w.server == nil {
		return errors.New("webhook server not initialized")
	}

	if w.port <= 0 || w.port > 65535 {
		return errors.New("invalid webhook server port")
	}

	// Orchestrator doesn't validate individual webhooks - those are validated when created
	return nil
}

// ProviderLifecycle interface implementation

// Initialize sets up the provider with required dependencies.
func (w *WebhookSourceProvider) Initialize(ctx context.Context, deps protocol.Dependencies) error {
	w.logger = deps.Logger

	// Initialize webhook-specific persistence based on URL
	persistenceURL := os.Getenv("WEBHOOK_PERSISTENCE_URL")
	if persistenceURL == "" {
		return errors.New("webhook provider requires WEBHOOK_PERSISTENCE_URL environment variable (e.g., file://./data/webhook)")
	}

	persistence, err := w.createPersistence(persistenceURL)
	if err != nil {
		return err
	}

	w.webhookPersistence = persistence

	// Get webhook server port from environment or use default
	w.port = w.getWebhookPort()

	// Initialize webhook server
	w.server = NewWebhookServer(w.port, w.logger)
	w.server.SetPersistence(w.webhookPersistence)

	w.logger.Info("Webhook provider initialized", "port", w.port, "persistence", persistenceURL)

	return nil
}

// Configure configures the provider based on current workflow definitions.
func (w *WebhookSourceProvider) Configure(workflows []*models.Workflow) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info("Configuring webhook provider with workflows", "workflow_count", len(workflows))

	sourceCount := 0

	for _, wf := range workflows {
		if wf.Status != models.WorkflowStatusActive {
			continue
		}

		for _, trigger := range wf.WorkflowTriggers {
			if trigger.TriggerID == "webhook" {
				if w.processWebhookTrigger(wf.ID, trigger) {
					sourceCount++
				}
			}
		}
	}

	// Get total sources count from persistence for logging
	totalSources, err := w.webhookPersistence.WebhookSources()
	if err != nil {
		w.logger.Warn("Failed to get total sources count", "error", err)
		w.logger.Info("Webhook configuration completed", "created_sources", sourceCount)
	} else {
		w.logger.Info("Webhook configuration completed",
			"created_sources", sourceCount,
			"total_sources", len(totalSources))
	}

	return nil
}

// Prepare performs final preparation before starting the provider.
func (w *WebhookSourceProvider) Prepare(ctx context.Context) error {
	if w.server == nil {
		return errors.New("webhook server not initialized")
	}

	// Get all active sources from persistence and register with server
	sources, err := w.webhookPersistence.ActiveWebhookSources()
	if err != nil {
		return err
	}

	for _, source := range sources {
		if err := w.server.RegisterSource(source); err != nil {
			w.logger.Error("Failed to register webhook source",
				"source_id", source.SourceID,
				"uuid", source.UUID,
				"error", err)

			return err
		}
	}

	w.logger.Info("Webhook provider prepared and ready",
		"registered_sources", len(sources))

	return nil
}

// processWebhookTrigger handles the creation of a webhook source for a trigger with webhook type.
// Returns true if a source was successfully created, false otherwise.
func (w *WebhookSourceProvider) processWebhookTrigger(workflowID string, trigger *models.WorkflowTrigger) bool {
	sourceID := trigger.SourceID
	if sourceID == "" {
		w.logger.Warn("Trigger has webhook type but no source_id",
			"workflow_id", workflowID,
			"trigger_id", trigger.ID)

		return false
	}

	// Check if source already exists by source ID (not UUID)
	existingSource, err := w.webhookPersistence.WebhookSourceBySourceID(sourceID)
	if err != nil {
		w.logger.Error("Failed to check existing webhook source",
			"source_id", sourceID,
			"error", err)

		return false
	}

	if existingSource != nil {
		w.logger.Debug("Webhook source already exists", "source_id", sourceID)
		// Update configuration if needed
		existingSource.UpdateConfiguration(trigger.Configuration)

		// Save updated source to persistence
		if err := w.webhookPersistence.SaveWebhookSource(existingSource); err != nil {
			w.logger.Error("Failed to update webhook source in persistence",
				"source_id", sourceID,
				"error", err)
		}

		return false
	}

	// Create new webhook source
	source, err := webhookModels.NewWebhookSource(sourceID, trigger.Configuration)
	if err != nil {
		w.logger.Error("Failed to create webhook source",
			"source_id", sourceID,
			"error", err)

		return false
	}

	// Save source to persistence
	if err := w.webhookPersistence.SaveWebhookSource(source); err != nil {
		w.logger.Error("Failed to save webhook source to persistence",
			"source_id", sourceID,
			"error", err)

		return false
	}

	w.logger.Info("Created webhook source",
		"source_id", sourceID,
		"uuid", source.UUID,
		"webhook_url", source.GetWebhookURL())

	return true
}

// getWebhookPort gets the webhook server port from configuration or environment.
func (w *WebhookSourceProvider) getWebhookPort() int {
	// Check configuration first
	if portVal, exists := w.config["port"]; exists {
		if port, ok := portVal.(int); ok && port > 0 && port <= 65535 {
			return port
		}

		if portStr, ok := portVal.(string); ok {
			if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port <= 65535 {
				return port
			}
		}
	}

	// Check environment variable
	if portEnv := os.Getenv("WEBHOOK_PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil && port > 0 && port <= 65535 {
			return port
		}
	}

	// Default port
	return 8085
}

// GetRegisteredSources returns registered sources from persistence for testing/debugging.
func (w *WebhookSourceProvider) GetRegisteredSources() map[string]*webhookModels.WebhookSource {
	w.mu.RLock()
	defer w.mu.RUnlock()

	sources, err := w.webhookPersistence.WebhookSources()
	if err != nil {
		w.logger.Error("Failed to get sources from persistence", "error", err)

		return make(map[string]*webhookModels.WebhookSource)
	}

	// Convert slice to map with UUID as key
	result := make(map[string]*webhookModels.WebhookSource)
	for _, source := range sources {
		result[source.UUID] = source
	}

	return result
}

// GetWebhookURL returns the webhook URL for a given source ID.
func (w *WebhookSourceProvider) GetWebhookURL(sourceID string) string {
	source, err := w.webhookPersistence.WebhookSourceBySourceID(sourceID)
	if err != nil || source == nil {
		return ""
	}

	return source.GetWebhookURL()
}

// createPersistence creates the appropriate persistence implementation based on URL scheme.
func (w *WebhookSourceProvider) createPersistence(persistenceURL string) (webhookPersistence.WebhookPersistence, error) {
	scheme := w.parsePersistenceScheme(persistenceURL)
	w.logger.Info("Initializing webhook persistence", "scheme", scheme, "url", persistenceURL)

	switch scheme {
	case "file":
		// Extract path from file://path
		path := strings.TrimPrefix(persistenceURL, "file://")

		return webhookPersistence.NewFilePersistence(path)
	case "postgres", "postgresql":
		// Future: implement database persistence
		return nil, errors.New("postgres persistence for webhook not yet implemented")
	case "mysql":
		// Future: implement database persistence
		return nil, errors.New("mysql persistence for webhook not yet implemented")
	default:
		return nil, errors.New("unsupported persistence scheme: " + scheme + " (supported: file://)")
	}
}

// parsePersistenceScheme extracts the scheme from a persistence URL.
func (w *WebhookSourceProvider) parsePersistenceScheme(persistenceURL string) string {
	parts := strings.SplitN(persistenceURL, "://", 2)
	if len(parts) < 2 {
		return "unknown"
	}

	return parts[0]
}
