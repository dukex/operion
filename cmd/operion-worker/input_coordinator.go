// Package main provides input coordination logic for the worker manager.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/google/uuid"
)

// InputCoordinator manages input collection and node readiness detection.
// This supports centralized coordination logic in the worker manager,
// allowing nodes to remain purely functional while the worker handles
// multi-input scenarios like merge nodes and loops.
type InputCoordinator struct {
	persistence persistence.Persistence
	logger      *slog.Logger
}

// NewInputCoordinator creates a new input coordinator.
func NewInputCoordinator(persistence persistence.Persistence, logger *slog.Logger) *InputCoordinator {
	return &InputCoordinator{
		persistence: persistence,
		logger:      logger,
	}
}

// GetPendingNodeExecution finds an existing pending execution for a node.
// This supports FIFO processing of loops where the same node may execute
// multiple times in the same workflow execution.
func (ic *InputCoordinator) GetPendingNodeExecution(ctx context.Context, nodeID, executionID string) (*models.NodeInputState, error) {
	repo := ic.persistence.InputCoordinationRepository()

	return repo.FindPendingNodeExecution(ctx, nodeID, executionID)
}

// AddInput adds an input to a node execution's coordination state.
// Returns the updated state and whether the node is ready to execute.
func (ic *InputCoordinator) AddInput(
	ctx context.Context,
	nodeID, executionID, nodeExecutionID, port string,
	result models.NodeResult,
	requirements models.InputRequirements,
) (*models.NodeInputState, bool, error) {
	repo := ic.persistence.InputCoordinationRepository()

	// Load existing state or create new one
	state, err := repo.LoadInputState(ctx, nodeExecutionID)
	if err != nil {
		// State doesn't exist yet, we'll create it
		state = &models.NodeInputState{
			NodeID:          nodeID,
			ExecutionID:     executionID,
			NodeExecutionID: nodeExecutionID,
			WorkflowID:      executionID, // Use execution ID for now, this could be refined later
			ReceivedInputs:  make(map[string]models.NodeResult),
			Requirements:    requirements,
			CreatedAt:       time.Now().UTC(),
			LastUpdatedAt:   time.Now().UTC(),
		}
	}

	// Add the new input
	state.ReceivedInputs[port] = result
	state.LastUpdatedAt = time.Now().UTC()

	// Save updated state
	if err := repo.SaveInputState(ctx, state); err != nil {
		return nil, false, fmt.Errorf("failed to save input state: %w", err)
	}

	// Check if node is ready to execute
	isReady := ic.IsNodeReady(state)

	ic.logger.DebugContext(ctx, "Added input to node coordination state",
		"node_id", nodeID,
		"node_execution_id", nodeExecutionID,
		"port", port,
		"received_ports", len(state.ReceivedInputs),
		"is_ready", isReady,
	)

	return state, isReady, nil
}

// IsNodeReady determines if a node has sufficient inputs to execute
// based on its requirements.
func (ic *InputCoordinator) IsNodeReady(state *models.NodeInputState) bool {
	requirements := state.Requirements

	switch requirements.WaitMode {
	case models.WaitModeAll:
		// All required ports must have inputs
		for _, requiredPort := range requirements.RequiredPorts {
			if _, hasInput := state.ReceivedInputs[requiredPort]; !hasInput {
				return false
			}
		}

		return true

	case models.WaitModeAny:
		// At least one required port must have input
		for _, requiredPort := range requirements.RequiredPorts {
			if _, hasInput := state.ReceivedInputs[requiredPort]; hasInput {
				return true
			}
		}

		return false

	case models.WaitModeFirst:
		// Ready if we have any input at all
		return len(state.ReceivedInputs) > 0

	default:
		// Unknown wait mode, default to any
		return len(state.ReceivedInputs) > 0
	}
}

// CleanupNodeExecution removes input state after successful node execution.
func (ic *InputCoordinator) CleanupNodeExecution(ctx context.Context, nodeExecutionID string) error {
	repo := ic.persistence.InputCoordinationRepository()

	if err := repo.DeleteInputState(ctx, nodeExecutionID); err != nil {
		return fmt.Errorf("failed to cleanup node execution state: %w", err)
	}

	ic.logger.DebugContext(ctx, "Cleaned up node execution state",
		"node_execution_id", nodeExecutionID,
	)

	return nil
}

// GenerateNodeExecutionID creates a unique identifier for a node execution instance.
func GenerateNodeExecutionID() string {
	return "node-exec-" + uuid.New().String()[:8]
}
