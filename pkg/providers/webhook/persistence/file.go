package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dukex/operion/pkg/providers/webhook/models"
)

// FilePersistence implements WebhookPersistence using JSON files.
type FilePersistence struct {
	dataDir        string
	mu             sync.RWMutex
	webhookSources map[string]*models.WebhookSource // ID -> WebhookSource mapping
}

// NewFilePersistence creates a new file-based webhook persistence.
func NewFilePersistence(dataDir string) (*FilePersistence, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fp := &FilePersistence{
		dataDir:        dataDir,
		webhookSources: make(map[string]*models.WebhookSource),
	}

	// Load existing webhook sources
	if err := fp.loadWebhookSources(); err != nil {
		return nil, fmt.Errorf("failed to load webhook sources: %w", err)
	}

	return fp, nil
}

// SaveWebhookSource saves a webhook source to the file system.
func (fp *FilePersistence) SaveWebhookSource(source *models.WebhookSource) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.webhookSources[source.ID] = source

	return fp.saveWebhookSourcesToFile()
}

// WebhookSourceByID retrieves a webhook source by its ID.
func (fp *FilePersistence) WebhookSourceByID(id string) (*models.WebhookSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	source, exists := fp.webhookSources[id]
	if !exists {
		return nil, nil
	}

	return source, nil
}

// WebhookSourceByExternalID retrieves a webhook source by its external ID.
func (fp *FilePersistence) WebhookSourceByExternalID(externalID string) (*models.WebhookSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	for _, source := range fp.webhookSources {
		if source.ExternalID.String() == externalID {
			return source, nil
		}
	}

	return nil, nil
}

// WebhookSources returns all webhook sources.
func (fp *FilePersistence) WebhookSources() ([]*models.WebhookSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	sources := make([]*models.WebhookSource, 0, len(fp.webhookSources))
	for _, source := range fp.webhookSources {
		sources = append(sources, source)
	}

	return sources, nil
}

// ActiveWebhookSources returns only active webhook sources.
func (fp *FilePersistence) ActiveWebhookSources() ([]*models.WebhookSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	var activeSources []*models.WebhookSource

	for _, source := range fp.webhookSources {
		if source.Active {
			activeSources = append(activeSources, source)
		}
	}

	return activeSources, nil
}

// DeleteWebhookSource removes a webhook source by its ID.
func (fp *FilePersistence) DeleteWebhookSource(id string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	delete(fp.webhookSources, id)

	return fp.saveWebhookSourcesToFile()
}

// HealthCheck verifies that the persistence layer is healthy.
func (fp *FilePersistence) HealthCheck() error {
	// Check if data directory is accessible
	if _, err := os.Stat(fp.dataDir); os.IsNotExist(err) {
		return fmt.Errorf("data directory does not exist: %s", fp.dataDir)
	}

	return nil
}

// Close cleans up resources.
func (fp *FilePersistence) Close() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	return fp.saveWebhookSourcesToFile()
}

// loadWebhookSources loads webhook sources from the file system.
func (fp *FilePersistence) loadWebhookSources() error {
	sourcesFile := filepath.Join(fp.dataDir, "webhook_sources.json")

	if _, err := os.Stat(sourcesFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty sources
		return nil
	}

	data, err := os.ReadFile(sourcesFile) // #nosec G304 -- sourcesFile is constructed from controlled dataDir
	if err != nil {
		return fmt.Errorf("failed to read webhook sources file: %w", err)
	}

	var sources []*models.WebhookSource
	if err := json.Unmarshal(data, &sources); err != nil {
		return fmt.Errorf("failed to unmarshal webhook sources: %w", err)
	}

	// Convert to map
	for _, source := range sources {
		fp.webhookSources[source.ID] = source
	}

	return nil
}

// saveWebhookSourcesToFile saves all webhook sources to the file system.
func (fp *FilePersistence) saveWebhookSourcesToFile() error {
	sourcesFile := filepath.Join(fp.dataDir, "webhook_sources.json")

	// Convert map to slice
	sources := make([]*models.WebhookSource, 0, len(fp.webhookSources))
	for _, source := range fp.webhookSources {
		sources = append(sources, source)
	}

	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal webhook sources: %w", err)
	}

	if err := os.WriteFile(sourcesFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write webhook sources file: %w", err)
	}

	return nil
}
