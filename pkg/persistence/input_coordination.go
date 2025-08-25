// Package persistence provides input coordination state management.
package persistence

import (
	"context"
	"time"

	"github.com/dukex/operion/pkg/models"
)

// InputCoordinationRepository manages persistent state for node input coordination.
// This supports the worker's centralized coordination logic for multi-input nodes.
type InputCoordinationRepository interface {
	// SaveInputState persists the current input state for a node execution.
	SaveInputState(ctx context.Context, state *models.NodeInputState) error

	// LoadInputState retrieves input state by node execution ID.
	LoadInputState(ctx context.Context, nodeExecutionID string) (*models.NodeInputState, error)

	// FindPendingNodeExecution finds the first pending execution for a node in a workflow execution.
	// This supports FIFO processing of loops where the same node executes multiple times.
	FindPendingNodeExecution(ctx context.Context, nodeID, executionID string) (*models.NodeInputState, error)

	// DeleteInputState removes input state after successful node execution.
	DeleteInputState(ctx context.Context, nodeExecutionID string) error

	// CleanupExpiredStates removes old input states that exceed the maximum age.
	// This prevents accumulation of abandoned coordination state.
	CleanupExpiredStates(ctx context.Context, maxAge time.Duration) error
}
