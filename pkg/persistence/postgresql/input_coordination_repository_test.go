package postgresql_test

import (
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestNodeInputState(t *testing.T, workflowID, nodeID, executionID string) *models.NodeInputState {
	t.Helper()

	return &models.NodeInputState{
		NodeID:          nodeID,
		ExecutionID:     executionID,
		NodeExecutionID: uuid.New().String(),
		WorkflowID:      workflowID,
		ReceivedInputs: map[string]models.NodeResult{
			"input1": {
				NodeID:    "input1",
				Data:      map[string]any{"data": "test data", "count": 42},
				Status:    "success",
				Timestamp: time.Now().UTC(),
			},
			"input2": {
				NodeID:    "input2",
				Data:      map[string]any{"partial": "data"},
				Status:    "failed",
				Error:     "input validation failed",
				Timestamp: time.Now().UTC(),
			},
		},
		Requirements: models.InputRequirements{
			RequiredPorts: []string{"input1", "input2"},
			OptionalPorts: []string{"input3"},
			WaitMode:      models.WaitModeAll,
			Timeout:       &[]time.Duration{30 * time.Second}[0],
		},
		CreatedAt:     time.Now().UTC(),
		LastUpdatedAt: time.Now().UTC(),
	}
}

func TestInputCoordinationRepository_SaveAndLoadInputState(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create input state
	inputState := createTestNodeInputState(t, workflow.ID, "test_node", execCtx.ID)

	// Test SaveInputState
	err = inputRepo.SaveInputState(ctx, inputState)
	require.NoError(t, err)

	// Test LoadInputState
	retrieved, err := inputRepo.LoadInputState(ctx, inputState.NodeExecutionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, inputState.NodeID, retrieved.NodeID)
	assert.Equal(t, inputState.ExecutionID, retrieved.ExecutionID)
	assert.Equal(t, inputState.NodeExecutionID, retrieved.NodeExecutionID)
	assert.Equal(t, inputState.WorkflowID, retrieved.WorkflowID)

	// Verify ReceivedInputs
	assert.Len(t, retrieved.ReceivedInputs, 2)
	assert.Equal(t, inputState.ReceivedInputs["input1"].Status, retrieved.ReceivedInputs["input1"].Status)
	assert.Equal(t, inputState.ReceivedInputs["input1"].Data["data"], retrieved.ReceivedInputs["input1"].Data["data"])
	assert.Equal(t, float64(42), retrieved.ReceivedInputs["input1"].Data["count"]) // JSON numbers are float64

	assert.Equal(t, inputState.ReceivedInputs["input2"].Status, retrieved.ReceivedInputs["input2"].Status)
	assert.Equal(t, inputState.ReceivedInputs["input2"].Error, retrieved.ReceivedInputs["input2"].Error)

	// Verify Requirements
	assert.Equal(t, inputState.Requirements.RequiredPorts, retrieved.Requirements.RequiredPorts)
	assert.Equal(t, inputState.Requirements.OptionalPorts, retrieved.Requirements.OptionalPorts)
	assert.Equal(t, inputState.Requirements.WaitMode, retrieved.Requirements.WaitMode)
	assert.NotNil(t, retrieved.Requirements.Timeout)
	assert.Equal(t, *inputState.Requirements.Timeout, *retrieved.Requirements.Timeout)
}

func TestInputCoordinationRepository_UpdateInputState(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create and save input state
	inputState := createTestNodeInputState(t, workflow.ID, "test_node", execCtx.ID)
	err = inputRepo.SaveInputState(ctx, inputState)
	require.NoError(t, err)

	// Update input state
	inputState.ReceivedInputs["input3"] = models.NodeResult{
		NodeID:    "input3",
		Data:      map[string]any{"additional": "input"},
		Status:    "success",
		Timestamp: time.Now().UTC(),
	}
	inputState.LastUpdatedAt = time.Now().UTC()

	err = inputRepo.SaveInputState(ctx, inputState) // SaveInputState handles upserts
	require.NoError(t, err)

	// Verify update
	updated, err := inputRepo.LoadInputState(ctx, inputState.NodeExecutionID)
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Len(t, updated.ReceivedInputs, 3)
	assert.Equal(t, "input", updated.ReceivedInputs["input3"].Data["additional"])
}

func TestInputCoordinationRepository_FindPendingNodeExecution(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create multiple input states for the same node and execution (simulating loops)
	baseTime := time.Now().UTC().Add(-1 * time.Hour)

	inputState1 := createTestNodeInputState(t, workflow.ID, "loop_node", execCtx.ID)
	inputState1.CreatedAt = baseTime
	inputState1.LastUpdatedAt = baseTime
	err = inputRepo.SaveInputState(ctx, inputState1)
	require.NoError(t, err)

	inputState2 := createTestNodeInputState(t, workflow.ID, "loop_node", execCtx.ID)
	inputState2.CreatedAt = baseTime.Add(10 * time.Minute)
	inputState2.LastUpdatedAt = baseTime.Add(10 * time.Minute)
	err = inputRepo.SaveInputState(ctx, inputState2)
	require.NoError(t, err)

	inputState3 := createTestNodeInputState(t, workflow.ID, "loop_node", execCtx.ID)
	inputState3.CreatedAt = baseTime.Add(5 * time.Minute) // Middle timestamp
	inputState3.LastUpdatedAt = baseTime.Add(5 * time.Minute)
	err = inputRepo.SaveInputState(ctx, inputState3)
	require.NoError(t, err)

	// Create input state for different node (should not be returned)
	inputState4 := createTestNodeInputState(t, workflow.ID, "other_node", execCtx.ID)
	inputState4.CreatedAt = baseTime.Add(-10 * time.Minute) // Oldest timestamp
	inputState4.LastUpdatedAt = baseTime.Add(-10 * time.Minute)
	err = inputRepo.SaveInputState(ctx, inputState4)
	require.NoError(t, err)

	// Test FindPendingNodeExecution - should return the oldest one for the specific node
	pending, err := inputRepo.FindPendingNodeExecution(ctx, "loop_node", execCtx.ID)
	require.NoError(t, err)
	require.NotNil(t, pending)

	// Should return the oldest one (inputState1)
	assert.Equal(t, inputState1.NodeExecutionID, pending.NodeExecutionID)
	assert.Equal(t, "loop_node", pending.NodeID)

	// Test with node that has no pending executions
	pending, err = inputRepo.FindPendingNodeExecution(ctx, "non_existent_node", execCtx.ID)
	require.NoError(t, err)
	assert.Nil(t, pending)

	// Test with different execution ID
	pending, err = inputRepo.FindPendingNodeExecution(ctx, "loop_node", "different_execution")
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func TestInputCoordinationRepository_DeleteInputState(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create and save input state
	inputState := createTestNodeInputState(t, workflow.ID, "test_node", execCtx.ID)
	err = inputRepo.SaveInputState(ctx, inputState)
	require.NoError(t, err)

	// Verify it exists
	retrieved, err := inputRepo.LoadInputState(ctx, inputState.NodeExecutionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Delete input state
	err = inputRepo.DeleteInputState(ctx, inputState.NodeExecutionID)
	require.NoError(t, err)

	// Verify it's deleted
	deleted, err := inputRepo.LoadInputState(ctx, inputState.NodeExecutionID)
	require.Error(t, err)
	assert.Nil(t, deleted)
	assert.Contains(t, err.Error(), "input state not found")

	// Test deleting non-existent input state (should not error, just warn)
	err = inputRepo.DeleteInputState(ctx, "non_existent_id")
	require.NoError(t, err) // Should not error, just logs a warning
}

func TestInputCoordinationRepository_CleanupExpiredStates(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create input states with different ages
	now := time.Now().UTC()

	// Old state (should be cleaned up)
	oldState := createTestNodeInputState(t, workflow.ID, "old_node", execCtx.ID)
	oldState.CreatedAt = now.Add(-2 * time.Hour)
	oldState.LastUpdatedAt = now.Add(-2 * time.Hour)
	err = inputRepo.SaveInputState(ctx, oldState)
	require.NoError(t, err)

	// Recent state (should NOT be cleaned up)
	recentState := createTestNodeInputState(t, workflow.ID, "recent_node", execCtx.ID)
	recentState.CreatedAt = now.Add(-30 * time.Minute)
	recentState.LastUpdatedAt = now.Add(-30 * time.Minute)
	err = inputRepo.SaveInputState(ctx, recentState)
	require.NoError(t, err)

	// Another old state (should be cleaned up)
	anotherOldState := createTestNodeInputState(t, workflow.ID, "another_old_node", execCtx.ID)
	anotherOldState.CreatedAt = now.Add(-3 * time.Hour)
	anotherOldState.LastUpdatedAt = now.Add(-3 * time.Hour)
	err = inputRepo.SaveInputState(ctx, anotherOldState)
	require.NoError(t, err)

	// Clean up states older than 1 hour
	err = inputRepo.CleanupExpiredStates(ctx, 1*time.Hour)
	require.NoError(t, err)

	// Verify old states are deleted
	_, err = inputRepo.LoadInputState(ctx, oldState.NodeExecutionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input state not found")

	_, err = inputRepo.LoadInputState(ctx, anotherOldState.NodeExecutionID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input state not found")

	// Verify recent state still exists
	retrieved, err := inputRepo.LoadInputState(ctx, recentState.NodeExecutionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	assert.Equal(t, recentState.NodeExecutionID, retrieved.NodeExecutionID)
}

func TestInputCoordinationRepository_ComplexRequirements(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	// Create and save workflow and execution context first
	workflow := createTestWorkflowForNodes(t)
	err := p.WorkflowRepository().Save(ctx, workflow)
	require.NoError(t, err)

	execCtx := createTestExecutionContext(t, workflow.ID)
	err = p.ExecutionContextRepository().SaveExecutionContext(ctx, execCtx)
	require.NoError(t, err)

	inputRepo := p.InputCoordinationRepository()

	// Create input state with complex requirements
	inputState := &models.NodeInputState{
		NodeID:          "complex_node",
		ExecutionID:     execCtx.ID,
		NodeExecutionID: uuid.New().String(),
		WorkflowID:      workflow.ID,
		ReceivedInputs: map[string]models.NodeResult{
			"api_response": {
				NodeID: "api_response",
				Status: "success",
				Data: map[string]any{
					"users": []map[string]any{
						{"id": 1, "name": "Alice", "active": true},
						{"id": 2, "name": "Bob", "active": false},
					},
					"metadata": map[string]any{
						"total_count": 2,
						"has_more":    false,
						"next_cursor": nil,
					},
				},
				Timestamp: time.Now().UTC(),
			},
			"validation_result": {
				NodeID: "validation_result",
				Status: "failed",
				Error:  "Schema validation failed",
				Data: map[string]any{
					"errors": []map[string]any{
						{"field": "email", "message": "Invalid email format"},
						{"field": "age", "message": "Age must be positive"},
					},
					"warnings": []string{"Deprecated field 'legacy_id' used"},
				},
				Timestamp: time.Now().UTC(),
			},
		},
		Requirements: models.InputRequirements{
			RequiredPorts: []string{"api_response", "validation_result", "auth_check"},
			OptionalPorts: []string{"cache_data", "audit_log"},
			WaitMode:      models.WaitModeAll,
			Timeout:       &[]time.Duration{2 * time.Minute}[0],
		},
		CreatedAt:     time.Now().UTC(),
		LastUpdatedAt: time.Now().UTC(),
	}

	// Save input state
	err = inputRepo.SaveInputState(ctx, inputState)
	require.NoError(t, err)

	// Retrieve and verify complex data
	retrieved, err := inputRepo.LoadInputState(ctx, inputState.NodeExecutionID)
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify complex ReceivedInputs
	apiResponse := retrieved.ReceivedInputs["api_response"]
	assert.Equal(t, "success", apiResponse.Status)

	users := apiResponse.Data["users"].([]any)
	assert.Len(t, users, 2)

	firstUser := users[0].(map[string]any)
	assert.Equal(t, float64(1), firstUser["id"])
	assert.Equal(t, "Alice", firstUser["name"])
	assert.Equal(t, true, firstUser["active"])

	metadata := apiResponse.Data["metadata"].(map[string]any)
	assert.Equal(t, float64(2), metadata["total_count"])
	assert.Equal(t, false, metadata["has_more"])
	assert.Nil(t, metadata["next_cursor"])

	// Verify validation result with errors
	validationResult := retrieved.ReceivedInputs["validation_result"]
	assert.Equal(t, "failed", validationResult.Status)
	assert.Equal(t, "Schema validation failed", validationResult.Error)

	errors := validationResult.Data["errors"].([]any)
	assert.Len(t, errors, 2)

	firstError := errors[0].(map[string]any)
	assert.Equal(t, "email", firstError["field"])
	assert.Equal(t, "Invalid email format", firstError["message"])

	warnings := validationResult.Data["warnings"].([]any)
	assert.Contains(t, warnings, "Deprecated field 'legacy_id' used")

	// Verify complex requirements
	assert.Equal(t, inputState.Requirements.RequiredPorts, retrieved.Requirements.RequiredPorts)
	assert.Equal(t, inputState.Requirements.OptionalPorts, retrieved.Requirements.OptionalPorts)
	assert.Equal(t, inputState.Requirements.WaitMode, retrieved.Requirements.WaitMode)
	assert.Equal(t, 2*time.Minute, *retrieved.Requirements.Timeout)
}

func TestInputCoordinationRepository_ErrorCases(t *testing.T) {
	p, ctx, _ := setupTestDB(t)

	inputRepo := p.InputCoordinationRepository()

	// Test loading non-existent input state
	state, err := inputRepo.LoadInputState(ctx, "non-existent-id")
	require.Error(t, err)
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "input state not found")

	// Test saving input state with non-existent execution
	invalidState := &models.NodeInputState{
		NodeID:          "test_node",
		ExecutionID:     "non-existent-execution",
		NodeExecutionID: uuid.New().String(),
		WorkflowID:      uuid.New().String(),
		ReceivedInputs:  make(map[string]models.NodeResult),
		Requirements: models.InputRequirements{
			RequiredPorts: []string{"input"},
			OptionalPorts: []string{},
			WaitMode:      models.WaitModeAny,
		},
		CreatedAt:     time.Now().UTC(),
		LastUpdatedAt: time.Now().UTC(),
	}

	err = inputRepo.SaveInputState(ctx, invalidState)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "violates foreign key constraint")
}
