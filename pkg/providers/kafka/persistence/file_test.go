package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dukex/operion/pkg/providers/kafka/models"
)

func TestNewFilePersistence(t *testing.T) {
	tests := []struct {
		name        string
		setupDir    func(t *testing.T) string
		expectError bool
	}{
		{
			name: "valid directory",
			setupDir: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			expectError: false,
		},
		{
			name: "non-existent directory should be created",
			setupDir: func(t *testing.T) string {
				t.Helper()

				return filepath.Join(t.TempDir(), "new-dir")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := tt.setupDir(t)

			fp, err := NewFilePersistence(dataDir)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, fp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, fp)

				// Verify directory was created
				_, err := os.Stat(dataDir)
				assert.NoError(t, err)

				// Should start with empty sources
				sources, err := fp.KafkaSources()
				require.NoError(t, err)
				assert.Empty(t, sources)

				// Cleanup
				err = fp.Close()
				assert.NoError(t, err)
			}
		})
	}
}

func TestFilePersistence_SaveAndRetrieveKafkaSource(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create test source
	config := map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
		"json_schema": map[string]any{
			"type": "object",
		},
	}
	source, err := models.NewKafkaSource("test-source", config)
	require.NoError(t, err)

	// Save source
	err = fp.SaveKafkaSource(source)
	require.NoError(t, err)

	// Retrieve by ID
	retrievedSource, err := fp.KafkaSourceByID("test-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	// Verify source data
	assert.Equal(t, source.ID, retrievedSource.ID)
	assert.Equal(t, source.ConnectionDetailsID, retrievedSource.ConnectionDetailsID)
	assert.Equal(t, source.ConnectionDetails, retrievedSource.ConnectionDetails)
	assert.Equal(t, source.JSONSchema, retrievedSource.JSONSchema)
	assert.Equal(t, source.Configuration, retrievedSource.Configuration)
	assert.Equal(t, source.Active, retrievedSource.Active)

	// Test non-existent source
	nonExistent, err := fp.KafkaSourceByID("non-existent")
	require.NoError(t, err)
	assert.Nil(t, nonExistent)
}

func TestFilePersistence_KafkaSourceByConnectionDetailsID(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create test sources with same connection details
	config1 := map[string]any{
		"topic":   "orders",
		"brokers": "localhost:9092",
	}
	source1, err := models.NewKafkaSource("source-1", config1)
	require.NoError(t, err)

	source2, err := models.NewKafkaSource("source-2", config1)
	require.NoError(t, err)

	// Create source with different connection details
	config2 := map[string]any{
		"topic":   "events",
		"brokers": "localhost:9092",
	}
	source3, err := models.NewKafkaSource("source-3", config2)
	require.NoError(t, err)

	// Save all sources
	err = fp.SaveKafkaSource(source1)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(source2)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(source3)
	require.NoError(t, err)

	// Retrieve sources by connection details ID
	sources1, err := fp.KafkaSourceByConnectionDetailsID(source1.ConnectionDetailsID)
	require.NoError(t, err)
	assert.Len(t, sources1, 2)

	// Verify correct sources returned
	sourceIDs := []string{sources1[0].ID, sources1[1].ID}
	assert.Contains(t, sourceIDs, "source-1")
	assert.Contains(t, sourceIDs, "source-2")

	// Retrieve source with different connection details
	sources2, err := fp.KafkaSourceByConnectionDetailsID(source3.ConnectionDetailsID)
	require.NoError(t, err)
	assert.Len(t, sources2, 1)
	assert.Equal(t, "source-3", sources2[0].ID)

	// Test non-existent connection details ID
	nonExistent, err := fp.KafkaSourceByConnectionDetailsID("non-existent")
	require.NoError(t, err)
	assert.Empty(t, nonExistent)
}

func TestFilePersistence_KafkaSources(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Initially should be empty
	sources, err := fp.KafkaSources()
	require.NoError(t, err)
	assert.Empty(t, sources)

	// Create and save test sources
	config1 := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	source1, err := models.NewKafkaSource("source-1", config1)
	require.NoError(t, err)

	config2 := map[string]any{"topic": "events", "brokers": "localhost:9092"}
	source2, err := models.NewKafkaSource("source-2", config2)
	require.NoError(t, err)

	err = fp.SaveKafkaSource(source1)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(source2)
	require.NoError(t, err)

	// Retrieve all sources
	sources, err = fp.KafkaSources()
	require.NoError(t, err)
	assert.Len(t, sources, 2)

	// Verify correct sources returned
	sourceIDs := []string{sources[0].ID, sources[1].ID}
	assert.Contains(t, sourceIDs, "source-1")
	assert.Contains(t, sourceIDs, "source-2")
}

func TestFilePersistence_ActiveKafkaSources(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create and save active source
	config1 := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	activeSource, err := models.NewKafkaSource("active-source", config1)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(activeSource)
	require.NoError(t, err)

	// Create and save inactive source
	config2 := map[string]any{"topic": "events", "brokers": "localhost:9092"}
	inactiveSource, err := models.NewKafkaSource("inactive-source", config2)
	require.NoError(t, err)

	inactiveSource.Active = false
	err = fp.SaveKafkaSource(inactiveSource)
	require.NoError(t, err)

	// Retrieve only active sources
	activeSources, err := fp.ActiveKafkaSources()
	require.NoError(t, err)
	assert.Len(t, activeSources, 1)
	assert.Equal(t, "active-source", activeSources[0].ID)
	assert.True(t, activeSources[0].Active)

	// Verify all sources still exist
	allSources, err := fp.KafkaSources()
	require.NoError(t, err)
	assert.Len(t, allSources, 2)
}

func TestFilePersistence_UpdateKafkaSource(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create and save initial source
	config := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	source, err := models.NewKafkaSource("test-source", config)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(source)
	require.NoError(t, err)

	// Update source configuration
	newConfig := map[string]any{
		"topic":          "events",
		"brokers":        "kafka1:9092,kafka2:9092",
		"consumer_group": "operion-events",
	}
	err = source.UpdateConfiguration(newConfig)
	require.NoError(t, err)

	// Save updated source
	err = fp.SaveKafkaSource(source)
	require.NoError(t, err)

	// Retrieve and verify update
	retrievedSource, err := fp.KafkaSourceByID("test-source")
	require.NoError(t, err)
	require.NotNil(t, retrievedSource)

	assert.Equal(t, "events", retrievedSource.ConnectionDetails.Topic)
	assert.Equal(t, "kafka1:9092,kafka2:9092", retrievedSource.ConnectionDetails.Brokers)
	assert.Equal(t, "operion-events", retrievedSource.ConnectionDetails.ConsumerGroup)
	assert.Equal(t, newConfig, retrievedSource.Configuration)
}

func TestFilePersistence_DeleteKafkaSource(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create and save test sources
	config1 := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	source1, err := models.NewKafkaSource("source-1", config1)
	require.NoError(t, err)

	config2 := map[string]any{"topic": "events", "brokers": "localhost:9092"}
	source2, err := models.NewKafkaSource("source-2", config2)
	require.NoError(t, err)

	err = fp.SaveKafkaSource(source1)
	require.NoError(t, err)
	err = fp.SaveKafkaSource(source2)
	require.NoError(t, err)

	// Verify both sources exist
	sources, err := fp.KafkaSources()
	require.NoError(t, err)
	assert.Len(t, sources, 2)

	// Delete one source
	err = fp.DeleteKafkaSource("source-1")
	require.NoError(t, err)

	// Verify source was deleted
	deletedSource, err := fp.KafkaSourceByID("source-1")
	require.NoError(t, err)
	assert.Nil(t, deletedSource)

	// Verify other source still exists
	remainingSource, err := fp.KafkaSourceByID("source-2")
	require.NoError(t, err)
	require.NotNil(t, remainingSource)
	assert.Equal(t, "source-2", remainingSource.ID)

	// Verify total count
	sources, err = fp.KafkaSources()
	require.NoError(t, err)
	assert.Len(t, sources, 1)

	// Delete non-existent source should not error
	err = fp.DeleteKafkaSource("non-existent")
	assert.NoError(t, err)
}

func TestFilePersistence_HealthCheck(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Health check should pass for valid directory
	err = fp.HealthCheck()
	assert.NoError(t, err)

	// Test with non-existent directory
	invalidFp := &FilePersistence{dataDir: "/non/existent/directory"}
	err = invalidFp.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data directory does not exist")
}

func TestFilePersistence_PersistenceAcrossRestarts(t *testing.T) {
	dataDir := t.TempDir()

	// Create first persistence instance
	fp1, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	// Save sources
	config1 := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	source1, err := models.NewKafkaSource("source-1", config1)
	require.NoError(t, err)

	config2 := map[string]any{"topic": "events", "brokers": "localhost:9092"}
	source2, err := models.NewKafkaSource("source-2", config2)
	require.NoError(t, err)

	err = fp1.SaveKafkaSource(source1)
	require.NoError(t, err)
	err = fp1.SaveKafkaSource(source2)
	require.NoError(t, err)

	// Close first instance
	err = fp1.Close()
	require.NoError(t, err)

	// Create second persistence instance (simulating restart)
	fp2, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp2.Close()
	}()

	// Verify sources were loaded from file
	sources, err := fp2.KafkaSources()
	require.NoError(t, err)
	assert.Len(t, sources, 2)

	// Verify source data integrity
	loadedSource1, err := fp2.KafkaSourceByID("source-1")
	require.NoError(t, err)
	require.NotNil(t, loadedSource1)
	assert.Equal(t, source1.ID, loadedSource1.ID)
	assert.Equal(t, source1.ConnectionDetails.Topic, loadedSource1.ConnectionDetails.Topic)

	loadedSource2, err := fp2.KafkaSourceByID("source-2")
	require.NoError(t, err)
	require.NotNil(t, loadedSource2)
	assert.Equal(t, source2.ID, loadedSource2.ID)
	assert.Equal(t, source2.ConnectionDetails.Topic, loadedSource2.ConnectionDetails.Topic)
}

func TestFilePersistence_ConcurrentAccess(t *testing.T) {
	dataDir := t.TempDir()
	fp, err := NewFilePersistence(dataDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create test source
	config := map[string]any{"topic": "orders", "brokers": "localhost:9092"}
	source, err := models.NewKafkaSource("concurrent-test", config)
	require.NoError(t, err)

	// Save initial source
	err = fp.SaveKafkaSource(source)
	require.NoError(t, err)

	// Test concurrent reads (should not cause data races)
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			// Concurrent reads
			sources, readErr := fp.KafkaSources()
			assert.NoError(t, readErr)
			assert.Len(t, sources, 1)

			retrievedSource, readErr := fp.KafkaSourceByID("concurrent-test")
			assert.NoError(t, readErr)
			assert.NotNil(t, retrievedSource)

			activeSource, readErr := fp.ActiveKafkaSources()
			assert.NoError(t, readErr)
			assert.Len(t, activeSource, 1)
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}
}
