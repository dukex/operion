package workflow

import (
	"os"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository(t *testing.T) {
	persistence := file.NewFilePersistence("./test-data")
	repo := NewRepository(persistence)

	assert.NotNil(t, repo)
	assert.Equal(t, persistence, repo.persistence)
}

func TestRepository_Create(t *testing.T) {
	testDir := "./test-repo-create"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	workflow := &models.Workflow{
		Name:        "Test Workflow",
		Description: "Test workflow description",
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

	// Create workflow
	created, err := repo.Create(workflow)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Verify ID was generated
	assert.NotEmpty(t, created.ID)

	// Verify timestamps were set
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())

	// Verify status was set
	assert.Equal(t, models.WorkflowStatusInactive, created.Status)

	// Clean up
	err = persistence.DeleteWorkflow(created.ID)
	assert.NoError(t, err)
	cleanupTestDirectory(testDir)
}

func TestRepository_Create_WithExistingID(t *testing.T) {
	testDir := "./test-repo-create-existing"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	workflow := &models.Workflow{
		ID:          "existing-id",
		Name:        "Test Workflow with ID",
		Description: "Test workflow with existing ID",
	}

	// Create workflow
	created, err := repo.Create(workflow)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Verify ID was preserved
	assert.Equal(t, "existing-id", created.ID)

	// Clean up
	err = persistence.DeleteWorkflow(created.ID)
	assert.NoError(t, err)

	cleanupTestDirectory(testDir)
}

func TestRepository_FetchByID(t *testing.T) {
	testDir := "./test-repo-fetch"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	// Create test workflow
	workflow := &models.Workflow{
		ID:          "fetch-test-workflow",
		Name:        "Fetch Test Workflow",
		Description: "Test workflow for fetching",
		Status:      models.WorkflowStatusActive,
	}

	// Save workflow
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Fetch workflow
	fetched, err := repo.FetchByID("fetch-test-workflow")
	require.NoError(t, err)
	require.NotNil(t, fetched)

	// Verify workflow data
	assert.Equal(t, "fetch-test-workflow", fetched.ID)
	assert.Equal(t, "Fetch Test Workflow", fetched.Name)
	assert.Equal(t, "Test workflow for fetching", fetched.Description)

	// Clean up
	err = repo.Delete("fetch-test-workflow")
	assert.NoError(t, err)

	cleanupTestDirectory(testDir)
}

func TestRepository_FetchByID_NotFound(t *testing.T) {
	testDir := "./test-repo-fetch-notfound"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	// Try to fetch non-existent workflow
	workflow, err := repo.FetchByID("non-existent")
	assert.Error(t, err)
	assert.Nil(t, workflow)
	assert.Contains(t, err.Error(), "workflow not found")

	cleanupTestDirectory(testDir)
}

func TestRepository_FetchAll(t *testing.T) {
	testDir := "./test-repo-fetchall"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

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
	}

	// Save all workflows
	for _, workflow := range workflows {
		_, err := repo.Create(workflow)
		require.NoError(t, err)
	}

	// Fetch all workflows
	fetchedWorkflows, err := repo.FetchAll()
	require.NoError(t, err)
	assert.Len(t, fetchedWorkflows, 2)

	// Verify workflows were fetched
	workflowIDs := make([]string, len(fetchedWorkflows))
	for i, workflow := range fetchedWorkflows {
		workflowIDs[i] = workflow.ID
	}

	assert.Contains(t, workflowIDs, "workflow-1")
	assert.Contains(t, workflowIDs, "workflow-2")

	// Clean up
	for _, workflow := range workflows {
		err = repo.Delete(workflow.ID)
		assert.NoError(t, err)
	}

	cleanupTestDirectory(testDir)
}

func TestRepository_Update(t *testing.T) {
	testDir := "./test-repo-update"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	// Create initial workflow
	originalWorkflow := &models.Workflow{
		ID:          "update-workflow",
		Name:        "Original Workflow",
		Description: "Original description",
		Status:      models.WorkflowStatusInactive,
	}

	_, err := repo.Create(originalWorkflow)
	require.NoError(t, err)

	// Update workflow
	updatedWorkflow := &models.Workflow{
		Name:        "Updated Workflow",
		Description: "Updated description",
		Status:      models.WorkflowStatusActive,
	}

	result, err := repo.Update("update-workflow", updatedWorkflow)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify update
	assert.Equal(t, "update-workflow", result.ID)
	assert.Equal(t, "Updated Workflow", result.Name)
	assert.Equal(t, "Updated description", result.Description)
	assert.Equal(t, models.WorkflowStatusActive, result.Status)

	// Verify timestamps (allowing some timing precision differences)
	assert.WithinDuration(t, originalWorkflow.CreatedAt, result.CreatedAt, 0)                                                // CreatedAt preserved
	assert.True(t, result.UpdatedAt.After(originalWorkflow.UpdatedAt) || result.UpdatedAt.Equal(originalWorkflow.UpdatedAt)) // UpdatedAt changed or same

	// Clean up
	err = repo.Delete("update-workflow")
	assert.NoError(t, err)

	cleanupTestDirectory(testDir)
}

func TestRepository_Update_NotFound(t *testing.T) {
	testDir := "./test-repo-update-notfound"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	updatedWorkflow := &models.Workflow{
		Name: "Updated Workflow",
	}

	// Try to update non-existent workflow
	result, err := repo.Update("non-existent", updatedWorkflow)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "workflow not found")

	cleanupTestDirectory(testDir)
}

func TestRepository_Delete(t *testing.T) {
	testDir := "./test-repo-delete"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	// Create test workflow
	workflow := &models.Workflow{
		ID:   "delete-workflow",
		Name: "Delete Test Workflow",
	}

	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Verify workflow exists
	fetched, err := repo.FetchByID("delete-workflow")
	require.NoError(t, err)
	assert.NotNil(t, fetched)

	// Delete workflow
	err = repo.Delete("delete-workflow")
	require.NoError(t, err)

	// Verify workflow was deleted
	fetched, err = repo.FetchByID("delete-workflow")
	assert.Error(t, err)
	assert.Nil(t, fetched)

	cleanupTestDirectory(testDir)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	testDir := "./test-repo-delete-notfound"
	persistence := file.NewFilePersistence(testDir)
	repo := NewRepository(persistence)

	// Try to delete non-existent workflow
	err := repo.Delete("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workflow not found")

	cleanupTestDirectory(testDir)
}

func cleanupTestDirectory(dir string) {
	os.RemoveAll(dir)
}
