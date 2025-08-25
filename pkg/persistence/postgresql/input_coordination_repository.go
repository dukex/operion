package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/models"
)

// InputCoordinationRepository handles input coordination state persistence in PostgreSQL.
type InputCoordinationRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewInputCoordinationRepository creates a new input coordination repository.
func NewInputCoordinationRepository(db *sql.DB, logger *slog.Logger) *InputCoordinationRepository {
	return &InputCoordinationRepository{db: db, logger: logger}
}

// SaveInputState persists the current input state for a node execution.
func (icr *InputCoordinationRepository) SaveInputState(ctx context.Context, state *models.NodeInputState) error {
	// Marshal complex fields to JSON
	receivedInputsJSON, err := json.Marshal(state.ReceivedInputs)
	if err != nil {
		return fmt.Errorf("failed to marshal received inputs: %w", err)
	}

	requirementsJSON, err := json.Marshal(state.Requirements)
	if err != nil {
		return fmt.Errorf("failed to marshal requirements: %w", err)
	}

	query := `
		INSERT INTO input_coordination_states (
			node_id, execution_id, node_execution_id, workflow_id, 
			received_inputs, requirements, created_at, last_updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (node_execution_id) DO UPDATE SET
			received_inputs = EXCLUDED.received_inputs,
			requirements = EXCLUDED.requirements,
			last_updated_at = EXCLUDED.last_updated_at
	`

	_, err = icr.db.ExecContext(ctx, query,
		state.NodeID,
		state.ExecutionID,
		state.NodeExecutionID,
		state.WorkflowID,
		receivedInputsJSON,
		requirementsJSON,
		state.CreatedAt,
		state.LastUpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save input state: %w", err)
	}

	return nil
}

// LoadInputState retrieves input state by node execution ID.
func (icr *InputCoordinationRepository) LoadInputState(ctx context.Context, nodeExecutionID string) (*models.NodeInputState, error) {
	query := `
		SELECT node_id, execution_id, node_execution_id, workflow_id, 
			   received_inputs, requirements, created_at, last_updated_at
		FROM input_coordination_states
		WHERE node_execution_id = $1
	`

	row := icr.db.QueryRowContext(ctx, query, nodeExecutionID)

	state, err := icr.scanInputState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("input state not found for node execution %s", nodeExecutionID)
		}

		return nil, fmt.Errorf("failed to scan input state: %w", err)
	}

	return state, nil
}

// FindPendingNodeExecution finds the first pending execution for a node in a workflow execution.
func (icr *InputCoordinationRepository) FindPendingNodeExecution(ctx context.Context, nodeID, executionID string) (*models.NodeInputState, error) {
	query := `
		SELECT node_id, execution_id, node_execution_id, workflow_id, 
			   received_inputs, requirements, created_at, last_updated_at
		FROM input_coordination_states
		WHERE node_id = $1 AND execution_id = $2
		ORDER BY created_at ASC
		LIMIT 1
	`

	row := icr.db.QueryRowContext(ctx, query, nodeID, executionID)

	state, err := icr.scanInputState(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No pending execution found
		}

		return nil, fmt.Errorf("failed to scan input state: %w", err)
	}

	return state, nil
}

// DeleteInputState removes input state after successful node execution.
func (icr *InputCoordinationRepository) DeleteInputState(ctx context.Context, nodeExecutionID string) error {
	query := `DELETE FROM input_coordination_states WHERE node_execution_id = $1`

	result, err := icr.db.ExecContext(ctx, query, nodeExecutionID)
	if err != nil {
		return fmt.Errorf("failed to delete input state: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		icr.logger.WarnContext(ctx, "input state not found for deletion", "node_execution_id", nodeExecutionID)
	}

	return nil
}

// CleanupExpiredStates removes old input states that exceed the maximum age.
func (icr *InputCoordinationRepository) CleanupExpiredStates(ctx context.Context, maxAge time.Duration) error {
	cutoffTime := time.Now().Add(-maxAge)

	query := `DELETE FROM input_coordination_states WHERE created_at < $1`

	result, err := icr.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired input states: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		icr.logger.InfoContext(ctx, "cleaned up expired input states", "count", rowsAffected, "cutoff_time", cutoffTime)
	}

	return nil
}

// scanInputState scans an input state from a database row.
func (icr *InputCoordinationRepository) scanInputState(scanner interface {
	Scan(dest ...any) error
}) (*models.NodeInputState, error) {
	var (
		state                                models.NodeInputState
		receivedInputsJSON, requirementsJSON []byte
	)

	err := scanner.Scan(
		&state.NodeID,
		&state.ExecutionID,
		&state.NodeExecutionID,
		&state.WorkflowID,
		&receivedInputsJSON,
		&requirementsJSON,
		&state.CreatedAt,
		&state.LastUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Initialize maps to avoid nil pointer dereferences
	state.ReceivedInputs = make(map[string]models.NodeResult)

	// Unmarshal JSON fields
	if receivedInputsJSON != nil {
		err := json.Unmarshal(receivedInputsJSON, &state.ReceivedInputs)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal received inputs: %w", err)
		}
	}

	if requirementsJSON != nil {
		err := json.Unmarshal(requirementsJSON, &state.Requirements)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal requirements: %w", err)
		}
	}

	return &state, nil
}
