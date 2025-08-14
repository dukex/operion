package file

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPersistence(t *testing.T) {
	// Test with regular path
	persistence := NewPersistence("/tmp/test")
	fp := persistence.(*Persistence)
	assert.Equal(t, "/tmp/test", fp.root)

	// Test with file:// prefix
	persistence = NewPersistence("file:///tmp/test")
	fp = persistence.(*Persistence)
	assert.Equal(t, "/tmp/test", fp.root)
}

func TestPersistence_Close(t *testing.T) {
	persistence := NewPersistence("./test-data")
	err := persistence.Close(t.Context())
	assert.NoError(t, err)
}

func TestPersistence_SaveWorkflow(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	workflow := &models.Workflow{
		ID:          "test-workflow",
		Name:        "Test Workflow",
		Description: "Test workflow description",
		Status:      models.WorkflowStatusActive,
		Steps: []*models.WorkflowStep{
			{
				ID:       "step-1",
				Name:     "Test Step",
				ActionID: "log",
				UID:      "test_step",
				Configuration: map[string]any{
					"message": "test",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	err := persistence.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Verify file was created
	filePath := filepath.Join(testDir, "workflows", "test-workflow.json")
	assert.FileExists(t, filePath)

	// Verify timestamps were set
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())
}

func TestPersistence_SaveWorkflow_UpdatesTimestamp(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	workflow := &models.Workflow{
		ID:        "update-workflow",
		Name:      "Update Test Workflow",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Save workflow
	err := persistence.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Verify CreatedAt was preserved and UpdatedAt was set
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), workflow.CreatedAt)
	assert.True(t, workflow.UpdatedAt.After(workflow.CreatedAt))
}

func TestPersistence_WorkflowByID(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Create test workflow
	originalWorkflow := &models.Workflow{
		ID:          "fetch-workflow",
		Name:        "Fetch Test Workflow",
		Description: "Test workflow for fetching",
		Status:      models.WorkflowStatusActive,
		Steps: []*models.WorkflowStep{
			{
				ID:       "step-1",
				Name:     "Test Step",
				ActionID: "log",
				UID:      "test_step",
				Configuration: map[string]any{
					"message": "test",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	err := persistence.SaveWorkflow(t.Context(), originalWorkflow)
	require.NoError(t, err)

	// Fetch workflow
	fetchedWorkflow, err := persistence.WorkflowByID(t.Context(), "fetch-workflow")
	require.NoError(t, err)
	require.NotNil(t, fetchedWorkflow)

	// Verify workflow data
	assert.Equal(t, "fetch-workflow", fetchedWorkflow.ID)
	assert.Equal(t, "Fetch Test Workflow", fetchedWorkflow.Name)
	assert.Equal(t, "Test workflow for fetching", fetchedWorkflow.Description)
	assert.Equal(t, models.WorkflowStatusActive, fetchedWorkflow.Status)
	assert.Len(t, fetchedWorkflow.Steps, 1)
	assert.Equal(t, "step-1", fetchedWorkflow.Steps[0].ID)
}

func TestPersistence_WorkflowByID_NotFound(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Try to fetch non-existent workflow
	workflow, err := persistence.WorkflowByID(t.Context(), "non-existent")
	require.NoError(t, err)
	require.Nil(t, workflow)
}

func TestPersistence_Workflows(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Create multiple test workflows
	workflows := []*models.Workflow{
		{
			ID:     "workflow-1",
			Name:   "First Workflow",
			Status: models.WorkflowStatusActive,
		},
		{
			ID:     "workflow-2",
			Name:   "Second Workflow",
			Status: models.WorkflowStatusInactive,
		},
		{
			ID:     "workflow-3",
			Name:   "Third Workflow",
			Status: models.WorkflowStatusActive,
		},
	}

	// Save all workflows
	for _, workflow := range workflows {
		err := persistence.SaveWorkflow(t.Context(), workflow)
		require.NoError(t, err)
	}

	// Fetch all workflows
	fetchedWorkflows, err := persistence.Workflows(t.Context())
	require.NoError(t, err)
	require.Len(t, fetchedWorkflows, 3)

	// Verify workflows were fetched (order might be different)
	workflowIDs := make([]string, len(fetchedWorkflows))
	for i, workflow := range fetchedWorkflows {
		workflowIDs[i] = workflow.ID
	}

	assert.Contains(t, workflowIDs, "workflow-1")
	assert.Contains(t, workflowIDs, "workflow-2")
	assert.Contains(t, workflowIDs, "workflow-3")
}

func TestPersistence_Workflows_EmptyDirectory(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Fetch workflows from empty directory
	workflows, err := persistence.Workflows(t.Context())
	require.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestPersistence_Workflows_NoDirectory(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Try to fetch workflows without creating directory first
	workflows, err := persistence.Workflows(t.Context())
	// fs.Glob on a non-existent directory returns empty slice with no error
	assert.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestPersistence_DeleteWorkflow(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Create test workflow
	workflow := &models.Workflow{
		ID:   "delete-workflow",
		Name: "Delete Test Workflow",
	}

	// Save workflow
	err := persistence.SaveWorkflow(t.Context(), workflow)
	require.NoError(t, err)

	// Verify file exists
	filePath := filepath.Join(testDir, "workflows", "delete-workflow.json")
	assert.FileExists(t, filePath)

	// Delete workflow
	err = persistence.DeleteWorkflow(t.Context(), "delete-workflow")
	require.NoError(t, err)

	// Verify file was deleted
	assert.NoFileExists(t, filePath)
}

func TestPersistence_DeleteWorkflow_NotFound(t *testing.T) {
	testDir := t.TempDir()

	persistence := NewPersistence(testDir)

	// Try to delete non-existent workflow (should not error)
	err := persistence.DeleteWorkflow(t.Context(), "non-existent")
	assert.NoError(t, err)
}
