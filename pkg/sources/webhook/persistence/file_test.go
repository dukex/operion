package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dukex/operion/pkg/sources/webhook/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilePersistence_SaveAndRetrieve(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "webhook_test")

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create persistence instance
	fp, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create test webhook source
	source, err := models.NewWebhookSource("test-source", map[string]any{
		"json_schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
		},
	})
	require.NoError(t, err)

	// Save source
	err = fp.SaveWebhookSource(source)
	require.NoError(t, err)

	// Retrieve by ID
	retrieved, err := fp.WebhookSourceByID(source.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, source.ID, retrieved.ID)
	assert.Equal(t, source.UUID, retrieved.UUID)
	assert.Equal(t, source.SourceID, retrieved.SourceID)

	// Retrieve by UUID
	retrievedByUUID, err := fp.WebhookSourceByUUID(source.UUID)
	require.NoError(t, err)
	require.NotNil(t, retrievedByUUID)
	assert.Equal(t, source.UUID, retrievedByUUID.UUID)

	// Retrieve by SourceID
	retrievedBySourceID, err := fp.WebhookSourceBySourceID(source.SourceID)
	require.NoError(t, err)
	require.NotNil(t, retrievedBySourceID)
	assert.Equal(t, source.SourceID, retrievedBySourceID.SourceID)
}

func TestFilePersistence_ListSources(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "webhook_list_test")

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create persistence instance
	fp, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create multiple test sources
	source1, err := models.NewWebhookSource("source-1", map[string]any{})
	require.NoError(t, err)

	source2, err := models.NewWebhookSource("source-2", map[string]any{})
	require.NoError(t, err)

	// Set source2 as inactive
	source2.Active = false

	// Save sources
	require.NoError(t, fp.SaveWebhookSource(source1))
	require.NoError(t, fp.SaveWebhookSource(source2))

	// Get all sources
	allSources, err := fp.WebhookSources()
	require.NoError(t, err)
	assert.Len(t, allSources, 2)

	// Get only active sources
	activeSources, err := fp.ActiveWebhookSources()
	require.NoError(t, err)
	assert.Len(t, activeSources, 1)
	assert.Equal(t, "source-1", activeSources[0].SourceID)
}

func TestFilePersistence_DeleteSources(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "webhook_delete_test")

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create persistence instance
	fp, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Create test source
	source, err := models.NewWebhookSource("test-source", map[string]any{})
	require.NoError(t, err)

	// Save source
	require.NoError(t, fp.SaveWebhookSource(source))

	// Verify source exists
	retrieved, err := fp.WebhookSourceByID(source.ID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete by source ID
	err = fp.DeleteWebhookSourceBySourceID(source.SourceID)
	require.NoError(t, err)

	// Verify source is deleted
	deleted, err := fp.WebhookSourceByID(source.ID)
	require.NoError(t, err)
	assert.Nil(t, deleted)
}

func TestFilePersistence_LoadExistingData(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "webhook_load_test")

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create first persistence instance and save data
	fp1, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	source, err := models.NewWebhookSource("persistent-source", map[string]any{
		"test": "data",
	})
	require.NoError(t, err)

	require.NoError(t, fp1.SaveWebhookSource(source))
	_ = fp1.Close()

	// Create second persistence instance - should load existing data
	fp2, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	defer func() {
		_ = fp2.Close()
	}()

	// Verify data was loaded
	retrieved, err := fp2.WebhookSourceBySourceID("persistent-source")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, "persistent-source", retrieved.SourceID)
	assert.Equal(t, source.UUID, retrieved.UUID)
}

func TestFilePersistence_HealthCheck(t *testing.T) {
	// Create temporary directory for test
	tmpDir := filepath.Join(os.TempDir(), "webhook_health_test")

	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create persistence instance
	fp, err := NewFilePersistence(tmpDir)
	require.NoError(t, err)

	defer func() {
		_ = fp.Close()
	}()

	// Health check should pass
	err = fp.HealthCheck()
	assert.NoError(t, err)

	// Remove directory and health check should fail
	_ = fp.Close()
	_ = os.RemoveAll(tmpDir)

	err = fp.HealthCheck()
	assert.Error(t, err)
}
