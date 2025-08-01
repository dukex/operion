package file

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilePersistence(t *testing.T) {
	// Test with regular path
	persistence := NewFilePersistence("/tmp/test")
	fp := persistence.(*FilePersistence)
	assert.Equal(t, "/tmp/test", fp.root)

	// Test with file:// prefix
	persistence = NewFilePersistence("file:///tmp/test")
	fp = persistence.(*FilePersistence)
	assert.Equal(t, "/tmp/test", fp.root)
}

func TestFilePersistence_Close(t *testing.T) {
	persistence := NewFilePersistence("./test-data")
	err := persistence.Close()
	assert.NoError(t, err)
}

func TestFilePersistence_SaveWorkflow(t *testing.T) {
	testDir := "./test-persistence"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

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
	err := persistence.SaveWorkflow(workflow)
	require.NoError(t, err)

	// Verify file was created
	filePath := filepath.Join(testDir, "workflows", "test-workflow.json")
	assert.FileExists(t, filePath)

	// Verify timestamps were set
	assert.False(t, workflow.CreatedAt.IsZero())
	assert.False(t, workflow.UpdatedAt.IsZero())
}

func TestFilePersistence_SaveWorkflow_UpdatesTimestamp(t *testing.T) {
	testDir := "./test-persistence-update"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	workflow := &models.Workflow{
		ID:        "update-workflow",
		Name:      "Update Test Workflow",
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// Save workflow
	err := persistence.SaveWorkflow(workflow)
	require.NoError(t, err)

	// Verify CreatedAt was preserved and UpdatedAt was set
	assert.Equal(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), workflow.CreatedAt)
	assert.True(t, workflow.UpdatedAt.After(workflow.CreatedAt))
}

func TestFilePersistence_WorkflowByID(t *testing.T) {
	testDir := "./test-persistence-fetch"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

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
	err := persistence.SaveWorkflow(originalWorkflow)
	require.NoError(t, err)

	// Fetch workflow
	fetchedWorkflow, err := persistence.WorkflowByID("fetch-workflow")
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

func TestFilePersistence_WorkflowByID_NotFound(t *testing.T) {
	testDir := "./test-persistence-notfound"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	// Try to fetch non-existent workflow
	workflow, err := persistence.WorkflowByID("non-existent")
	assert.NoError(t, err)
	assert.Nil(t, workflow)
}

func TestFilePersistence_Workflows(t *testing.T) {
	testDir := "./test-persistence-list"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

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
		err := persistence.SaveWorkflow(workflow)
		require.NoError(t, err)
	}

	// Fetch all workflows
	fetchedWorkflows, err := persistence.Workflows()
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

func TestFilePersistence_Workflows_EmptyDirectory(t *testing.T) {
	testDir := "./test-persistence-empty"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	// Create workflows directory but leave it empty
	err := os.MkdirAll(filepath.Join(testDir, "workflows"), 0755)
	require.NoError(t, err)

	// Fetch workflows from empty directory
	workflows, err := persistence.Workflows()
	require.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestFilePersistence_Workflows_NoDirectory(t *testing.T) {
	testDir := "./test-persistence-nodir"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	// Try to fetch workflows without creating directory first
	workflows, err := persistence.Workflows()
	// fs.Glob on a non-existent directory returns empty slice with no error
	assert.NoError(t, err)
	assert.Empty(t, workflows)
}

func TestFilePersistence_DeleteWorkflow(t *testing.T) {
	testDir := "./test-persistence-delete"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	// Create test workflow
	workflow := &models.Workflow{
		ID:   "delete-workflow",
		Name: "Delete Test Workflow",
	}

	// Save workflow
	err := persistence.SaveWorkflow(workflow)
	require.NoError(t, err)

	// Verify file exists
	filePath := filepath.Join(testDir, "workflows", "delete-workflow.json")
	assert.FileExists(t, filePath)

	// Delete workflow
	err = persistence.DeleteWorkflow("delete-workflow")
	require.NoError(t, err)

	// Verify file was deleted
	assert.NoFileExists(t, filePath)
}

func TestFilePersistence_DeleteWorkflow_NotFound(t *testing.T) {
	testDir := "./test-persistence-delete-notfound"
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}()

	persistence := NewFilePersistence(testDir)

	// Try to delete non-existent workflow (should not error)
	err := persistence.DeleteWorkflow("non-existent")
	assert.NoError(t, err)
}
