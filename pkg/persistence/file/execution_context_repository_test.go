package file

import (
	"context"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutionContextRepository_SaveAndGetExecutionContext(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create test execution context
	execCtx := &models.ExecutionContext{
		ID:         "test-execution-123",
		WorkflowID: "workflow-456",
		Status:     models.ExecutionStatusRunning,
		NodeResults: map[string]models.NodeResult{
			"node1::success": {
				NodeID: "node1",
				Data:   map[string]any{"message": "processed"},
				Status: string(models.NodeStatusSuccess),
			},
		},
		TriggerData: map[string]any{
			"event_type": "webhook",
			"payload":    map[string]any{"user_id": "123"},
		},
		Variables: map[string]any{
			"api_url": "https://api.example.com",
			"timeout": 30,
		},
		Metadata: map[string]any{
			"retry_count": 0,
			"priority":    "high",
		},
		ErrorMessage: "",
		CreatedAt:    time.Now(),
		CompletedAt:  nil,
	}

	// Test SaveExecutionContext
	execRepo := persistence.ExecutionContextRepository()
	err := execRepo.SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	// Test GetExecutionContext
	retrieved, err := execRepo.GetExecutionContext(ctx, "test-execution-123")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify all fields
	assert.Equal(t, "test-execution-123", retrieved.ID)
	assert.Equal(t, "workflow-456", retrieved.WorkflowID)
	assert.Equal(t, models.ExecutionStatusRunning, retrieved.Status)
	assert.Equal(t, "processed", retrieved.NodeResults["node1::success"].Data["message"])
	assert.Equal(t, "webhook", retrieved.TriggerData["event_type"])
	assert.Equal(t, "https://api.example.com", retrieved.Variables["api_url"])
	assert.Equal(t, "high", retrieved.Metadata["priority"])
	assert.Empty(t, retrieved.ErrorMessage)
	assert.Nil(t, retrieved.CompletedAt)
}

func TestExecutionContextRepository_UpdateExecutionContext(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create and save initial execution context
	execCtx := &models.ExecutionContext{
		ID:          "test-execution-update",
		WorkflowID:  "workflow-789",
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: map[string]any{"source": "test"},
		Variables:   map[string]any{"counter": 1},
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execRepo := persistence.ExecutionContextRepository()
	err := execRepo.SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	// Update the execution context
	completedAt := time.Now()
	execCtx.Status = models.ExecutionStatusCompleted
	execCtx.NodeResults["final::success"] = models.NodeResult{
		NodeID: "final",
		Data:   map[string]any{"result": "completed"},
		Status: string(models.NodeStatusSuccess),
	}
	execCtx.Variables["counter"] = 5
	execCtx.CompletedAt = &completedAt

	// Test UpdateExecutionContext
	err = execRepo.UpdateExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	// Verify the update
	updated, err := execRepo.GetExecutionContext(ctx, "test-execution-update")
	require.NoError(t, err)
	assert.Equal(t, models.ExecutionStatusCompleted, updated.Status)
	assert.Equal(t, "completed", updated.NodeResults["final::success"].Data["result"])
	assert.Equal(t, float64(5), updated.Variables["counter"]) // JSON unmarshaling converts numbers to float64
	assert.NotNil(t, updated.CompletedAt)
}

func TestExecutionContextRepository_GetExecutionsByWorkflow(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create multiple execution contexts for different workflows
	execCtx1 := &models.ExecutionContext{
		ID:          "exec-1",
		WorkflowID:  "workflow-target",
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execCtx2 := &models.ExecutionContext{
		ID:          "exec-2",
		WorkflowID:  "workflow-target", // Same workflow
		Status:      models.ExecutionStatusCompleted,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execCtx3 := &models.ExecutionContext{
		ID:          "exec-3",
		WorkflowID:  "workflow-different", // Different workflow
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execRepo := persistence.ExecutionContextRepository()

	// Save all executions
	err := execRepo.SaveExecutionContext(ctx, execCtx1)
	require.NoError(t, err)
	err = execRepo.SaveExecutionContext(ctx, execCtx2)
	require.NoError(t, err)
	err = execRepo.SaveExecutionContext(ctx, execCtx3)
	require.NoError(t, err)

	// Test GetExecutionsByWorkflow
	executions, err := execRepo.GetExecutionsByWorkflow(ctx, "workflow-target")
	require.NoError(t, err)

	// Should get 2 executions for workflow-target
	assert.Len(t, executions, 2)

	executionIDs := make([]string, 0, len(executions))
	for _, exec := range executions {
		executionIDs = append(executionIDs, exec.ID)
	}

	assert.Contains(t, executionIDs, "exec-1")
	assert.Contains(t, executionIDs, "exec-2")
	assert.NotContains(t, executionIDs, "exec-3")
}

func TestExecutionContextRepository_GetExecutionsByStatus(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	// Create execution contexts with different statuses
	execCtx1 := &models.ExecutionContext{
		ID:          "exec-running-1",
		WorkflowID:  "workflow-1",
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execCtx2 := &models.ExecutionContext{
		ID:          "exec-running-2",
		WorkflowID:  "workflow-2",
		Status:      models.ExecutionStatusRunning, // Same status
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execCtx3 := &models.ExecutionContext{
		ID:          "exec-completed",
		WorkflowID:  "workflow-3",
		Status:      models.ExecutionStatusCompleted, // Different status
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execRepo := persistence.ExecutionContextRepository()

	// Save all executions
	err := execRepo.SaveExecutionContext(ctx, execCtx1)
	require.NoError(t, err)
	err = execRepo.SaveExecutionContext(ctx, execCtx2)
	require.NoError(t, err)
	err = execRepo.SaveExecutionContext(ctx, execCtx3)
	require.NoError(t, err)

	// Test GetExecutionsByStatus for running executions
	runningExecutions, err := execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusRunning)
	require.NoError(t, err)

	// Should get 2 running executions
	assert.Len(t, runningExecutions, 2)

	runningIDs := make([]string, 0, len(runningExecutions))
	for _, exec := range runningExecutions {
		runningIDs = append(runningIDs, exec.ID)
	}

	assert.Contains(t, runningIDs, "exec-running-1")
	assert.Contains(t, runningIDs, "exec-running-2")
	assert.NotContains(t, runningIDs, "exec-completed")

	// Test GetExecutionsByStatus for completed executions
	completedExecutions, err := execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusCompleted)
	require.NoError(t, err)

	// Should get 1 completed execution
	assert.Len(t, completedExecutions, 1)
	assert.Equal(t, "exec-completed", completedExecutions[0].ID)
}

func TestExecutionContextRepository_GetExecutionContext_NotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	execRepo := persistence.ExecutionContextRepository()

	// Test GetExecutionContext with non-existent ID
	execCtx, err := execRepo.GetExecutionContext(ctx, "non-existent-execution")

	// Verify
	assert.Error(t, err)
	assert.Nil(t, execCtx)
	assert.Contains(t, err.Error(), "execution context not found")
}

func TestExecutionContextRepository_UpdateExecutionContext_NotFound(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	execRepo := persistence.ExecutionContextRepository()

	// Create execution context that hasn't been saved
	execCtx := &models.ExecutionContext{
		ID:          "non-existent-execution",
		WorkflowID:  "workflow-1",
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	// Test UpdateExecutionContext on non-existent execution
	err := execRepo.UpdateExecutionContext(ctx, execCtx)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution context not found")
}

func TestExecutionContextRepository_EmptyRepositories(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := NewPersistence(tempDir)
	ctx := context.Background()

	execRepo := persistence.ExecutionContextRepository()

	// Test operations on empty repository
	executions, err := execRepo.GetExecutionsByWorkflow(ctx, "any-workflow")
	require.NoError(t, err)
	assert.Empty(t, executions)

	executions, err = execRepo.GetExecutionsByStatus(ctx, models.ExecutionStatusRunning)
	require.NoError(t, err)
	assert.Empty(t, executions)
}

func TestExecutionContextRepository_DataIsolation(t *testing.T) {
	// Setup - Create two different persistence instances to ensure isolation
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()
	persistence1 := NewPersistence(tempDir1)
	persistence2 := NewPersistence(tempDir2)
	ctx := context.Background()

	// Create execution context in first persistence
	execCtx := &models.ExecutionContext{
		ID:          "isolated-execution",
		WorkflowID:  "workflow-1",
		Status:      models.ExecutionStatusRunning,
		NodeResults: make(map[string]models.NodeResult),
		TriggerData: make(map[string]any),
		Variables:   make(map[string]any),
		Metadata:    make(map[string]any),
		CreatedAt:   time.Now(),
	}

	execRepo1 := persistence1.ExecutionContextRepository()
	err := execRepo1.SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	// Verify it exists in first repository
	_, err = execRepo1.GetExecutionContext(ctx, "isolated-execution")
	assert.NoError(t, err)

	// Verify it doesn't exist in second repository
	execRepo2 := persistence2.ExecutionContextRepository()
	_, err = execRepo2.GetExecutionContext(ctx, "isolated-execution")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "execution context not found")
}
