package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dukex/operion/pkg/providers/kafka/models"
)

// FilePersistence implements KafkaPersistence using JSON files.
type FilePersistence struct {
	dataDir      string
	mu           sync.RWMutex
	kafkaSources map[string]*models.KafkaSource // ID -> KafkaSource mapping
}

// NewFilePersistence creates a new file-based Kafka persistence.
func NewFilePersistence(dataDir string) (*FilePersistence, error) {
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	fp := &FilePersistence{
		dataDir:      dataDir,
		kafkaSources: make(map[string]*models.KafkaSource),
	}

	// Load existing Kafka sources
	if err := fp.loadKafkaSources(); err != nil {
		return nil, fmt.Errorf("failed to load Kafka sources: %w", err)
	}

	return fp, nil
}

// SaveKafkaSource saves a Kafka source to the file system.
func (fp *FilePersistence) SaveKafkaSource(source *models.KafkaSource) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	fp.kafkaSources[source.ID] = source

	return fp.saveKafkaSourcesToFile()
}

// KafkaSourceByID retrieves a Kafka source by its ID.
func (fp *FilePersistence) KafkaSourceByID(id string) (*models.KafkaSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	source, exists := fp.kafkaSources[id]
	if !exists {
		return nil, nil
	}

	return source, nil
}

// KafkaSourceByConnectionDetailsID retrieves Kafka sources by their connection details ID.
// This is used for consumer sharing - sources with the same connection details ID can share consumers.
func (fp *FilePersistence) KafkaSourceByConnectionDetailsID(connectionDetailsID string) ([]*models.KafkaSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	var sources []*models.KafkaSource

	for _, source := range fp.kafkaSources {
		if source.ConnectionDetailsID == connectionDetailsID {
			sources = append(sources, source)
		}
	}

	return sources, nil
}

// KafkaSources returns all Kafka sources.
func (fp *FilePersistence) KafkaSources() ([]*models.KafkaSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	sources := make([]*models.KafkaSource, 0, len(fp.kafkaSources))
	for _, source := range fp.kafkaSources {
		sources = append(sources, source)
	}

	return sources, nil
}

// ActiveKafkaSources returns only active Kafka sources.
func (fp *FilePersistence) ActiveKafkaSources() ([]*models.KafkaSource, error) {
	fp.mu.RLock()
	defer fp.mu.RUnlock()

	var activeSources []*models.KafkaSource

	for _, source := range fp.kafkaSources {
		if source.Active {
			activeSources = append(activeSources, source)
		}
	}

	return activeSources, nil
}

// DeleteKafkaSource removes a Kafka source by its ID.
func (fp *FilePersistence) DeleteKafkaSource(id string) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	delete(fp.kafkaSources, id)

	return fp.saveKafkaSourcesToFile()
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

	return fp.saveKafkaSourcesToFile()
}

// loadKafkaSources loads Kafka sources from the file system.
func (fp *FilePersistence) loadKafkaSources() error {
	sourcesFile := filepath.Join(fp.dataDir, "kafka_sources.json")

	if _, err := os.Stat(sourcesFile); os.IsNotExist(err) {
		// File doesn't exist, start with empty sources
		return nil
	}

	data, err := os.ReadFile(sourcesFile) // #nosec G304 -- sourcesFile is constructed from controlled dataDir
	if err != nil {
		return fmt.Errorf("failed to read Kafka sources file: %w", err)
	}

	var sources []*models.KafkaSource
	if err := json.Unmarshal(data, &sources); err != nil {
		return fmt.Errorf("failed to unmarshal Kafka sources: %w", err)
	}

	// Convert to map
	for _, source := range sources {
		fp.kafkaSources[source.ID] = source
	}

	return nil
}

// saveKafkaSourcesToFile saves all Kafka sources to the file system.
func (fp *FilePersistence) saveKafkaSourcesToFile() error {
	sourcesFile := filepath.Join(fp.dataDir, "kafka_sources.json")

	// Convert map to slice
	sources := make([]*models.KafkaSource, 0, len(fp.kafkaSources))
	for _, source := range fp.kafkaSources {
		sources = append(sources, source)
	}

	data, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal Kafka sources: %w", err)
	}

	if err := os.WriteFile(sourcesFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write Kafka sources file: %w", err)
	}

	return nil
}
