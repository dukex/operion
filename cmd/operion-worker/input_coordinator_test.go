package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputCoordinator_AddInput_SingleInput(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "test-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"
	port := "input"

	result := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"value": "test"},
		Status: string(models.NodeStatusSuccess),
	}

	requirements := models.InputRequirements{
		RequiredPorts: []string{"input"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Execute
	state, isReady, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, port, result, requirements)

	// Verify
	require.NoError(t, err)
	assert.True(t, isReady)
	assert.NotNil(t, state)
	assert.Equal(t, nodeID, state.NodeID)
	assert.Equal(t, executionID, state.ExecutionID)
	assert.Equal(t, nodeExecutionID, state.NodeExecutionID)
	assert.Equal(t, 1, len(state.ReceivedInputs))
	assert.Equal(t, result, state.ReceivedInputs[port])
}

func TestInputCoordinator_AddInput_MultipleInputs_WaitAll(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "merge-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"left", "right"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Add first input
	result1 := models.NodeResult{
		NodeID: "node-a",
		Data:   map[string]any{"value": "left-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state1, isReady1, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result1, requirements)
	require.NoError(t, err)
	assert.False(t, isReady1) // Should not be ready with only 1 of 2 inputs
	assert.Equal(t, 1, len(state1.ReceivedInputs))

	// Add second input
	result2 := models.NodeResult{
		NodeID: "node-b",
		Data:   map[string]any{"value": "right-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state2, isReady2, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "right", result2, requirements)
	require.NoError(t, err)
	assert.True(t, isReady2) // Should be ready with both inputs
	assert.Equal(t, 2, len(state2.ReceivedInputs))
	assert.Equal(t, result1, state2.ReceivedInputs["left"])
	assert.Equal(t, result2, state2.ReceivedInputs["right"])
}

func TestInputCoordinator_AddInput_MultipleInputs_WaitAny(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "merge-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"left", "right"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAny,
		Timeout:       nil,
	}

	// Add first input
	result1 := models.NodeResult{
		NodeID: "node-a",
		Data:   map[string]any{"value": "left-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state1, isReady1, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result1, requirements)
	require.NoError(t, err)
	assert.True(t, isReady1) // Should be ready with any input for WaitModeAny
	assert.Equal(t, 1, len(state1.ReceivedInputs))
}

func TestInputCoordinator_AddInput_MultipleInputs_WaitFirst(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "merge-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"left", "right"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeFirst,
		Timeout:       nil,
	}

	// Add first input
	result1 := models.NodeResult{
		NodeID: "node-a",
		Data:   map[string]any{"value": "left-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state1, isReady1, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result1, requirements)
	require.NoError(t, err)
	assert.True(t, isReady1) // Should be ready with first input for WaitModeFirst
	assert.Equal(t, 1, len(state1.ReceivedInputs))

	// Add second input (will be added but node is already ready after first input)
	result2 := models.NodeResult{
		NodeID: "node-b",
		Data:   map[string]any{"value": "right-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state2, isReady2, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "right", result2, requirements)
	require.NoError(t, err)
	assert.True(t, isReady2)
	assert.Equal(t, 2, len(state2.ReceivedInputs)) // Will have both inputs, but node was ready after first
}

func TestInputCoordinator_AddInput_WithOptionalPorts(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "test-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"required"},
		OptionalPorts: []string{"optional"},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Add only required input
	result := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"value": "required-data"},
		Status: string(models.NodeStatusSuccess),
	}

	state, isReady, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "required", result, requirements)
	require.NoError(t, err)
	assert.True(t, isReady) // Should be ready with only required input
	assert.Equal(t, 1, len(state.ReceivedInputs))
}

func TestInputCoordinator_AddInput_WithTimeout(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "timeout-node"
	executionID := "test-execution"
	nodeExecutionID := "test-node-exec-1"

	timeout := 100 * time.Millisecond
	requirements := models.InputRequirements{
		RequiredPorts: []string{"left", "right"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       &timeout,
	}

	// Add first input
	result1 := models.NodeResult{
		NodeID: "node-a",
		Data:   map[string]any{"value": "left-data"},
		Status: string(models.NodeStatusSuccess),
	}

	_, isReady1, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result1, requirements)
	require.NoError(t, err)
	assert.False(t, isReady1) // Should not be ready with only 1 of 2 inputs

	// Wait for timeout to pass
	time.Sleep(150 * time.Millisecond)

	// Check if timed out (this is a basic test - in practice, timeout handling would be more sophisticated)
	state2, isReady2, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result1, requirements)
	require.NoError(t, err)

	// The timeout behavior depends on implementation details, but we verify the structure works
	assert.NotNil(t, state2)
	assert.False(t, isReady2) // Still waiting for second input
}

func TestInputCoordinator_FIFO_Loop_Handling(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "loop-node"
	executionID := "test-execution"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"input"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Simulate multiple node executions (loop scenario)
	nodeExecutionID1 := "node-exec-1"
	nodeExecutionID2 := "node-exec-2"

	result1 := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"iteration": 1},
		Status: string(models.NodeStatusSuccess),
	}

	result2 := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"iteration": 2},
		Status: string(models.NodeStatusSuccess),
	}

	// Add inputs for different node executions
	state1, isReady1, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID1, "input", result1, requirements)
	require.NoError(t, err)
	assert.True(t, isReady1)
	assert.Equal(t, result1, state1.ReceivedInputs["input"])

	state2, isReady2, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID2, "input", result2, requirements)
	require.NoError(t, err)
	assert.True(t, isReady2)
	assert.Equal(t, result2, state2.ReceivedInputs["input"])

	// Verify they're separate executions
	assert.NotEqual(t, state1.NodeExecutionID, state2.NodeExecutionID)
}

func TestInputCoordinator_GetPendingNodeExecution(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "pending-node"
	executionID := "test-execution"
	nodeExecutionID := "node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"left", "right"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Add partial input (node not ready yet)
	result := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"value": "left-data"},
		Status: string(models.NodeStatusSuccess),
	}

	_, isReady, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "left", result, requirements)
	require.NoError(t, err)
	assert.False(t, isReady) // Should not be ready

	// Check for pending execution
	pendingState, err := coordinator.GetPendingNodeExecution(ctx, nodeID, executionID)
	require.NoError(t, err)
	assert.NotNil(t, pendingState)
	assert.Equal(t, nodeExecutionID, pendingState.NodeExecutionID)
	assert.Equal(t, 1, len(pendingState.ReceivedInputs))
}

func TestInputCoordinator_CleanupNodeExecution(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "cleanup-node"
	executionID := "test-execution"
	nodeExecutionID := "node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"input"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Add input
	result := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"value": "test"},
		Status: string(models.NodeStatusSuccess),
	}

	_, isReady, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "input", result, requirements)
	require.NoError(t, err)
	assert.True(t, isReady)

	// Clean up
	err = coordinator.CleanupNodeExecution(ctx, nodeExecutionID)
	require.NoError(t, err)

	// Verify cleanup worked - should not find pending execution
	pendingState, err := coordinator.GetPendingNodeExecution(ctx, nodeID, executionID)
	require.NoError(t, err)
	assert.Nil(t, pendingState) // Should be nil after cleanup
}

func TestInputCoordinator_InvalidInput(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	coordinator := NewInputCoordinator(persistence, logger)

	ctx := context.Background()
	nodeID := "test-node"
	executionID := "test-execution"
	nodeExecutionID := "node-exec-1"

	requirements := models.InputRequirements{
		RequiredPorts: []string{"required"},
		OptionalPorts: []string{},
		WaitMode:      models.WaitModeAll,
		Timeout:       nil,
	}

	// Try to add input for non-required and non-optional port
	result := models.NodeResult{
		NodeID: "source-node",
		Data:   map[string]any{"value": "test"},
		Status: string(models.NodeStatusSuccess),
	}

	_, isReady, err := coordinator.AddInput(ctx, nodeID, executionID, nodeExecutionID, "invalid-port", result, requirements)
	require.NoError(t, err)  // Should handle gracefully
	assert.False(t, isReady) // Should not be ready due to missing required port
}
