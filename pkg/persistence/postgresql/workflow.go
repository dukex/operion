package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/google/uuid"
)

// WorkflowRepository handles workflow-related database operations.
type WorkflowRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewWorkflowRepository creates a new workflow repository.
func NewWorkflowRepository(db *sql.DB, logger *slog.Logger) *WorkflowRepository {
	return &WorkflowRepository{db: db, logger: logger}
}

// GetAll returns all workflows from the database.
func (r *WorkflowRepository) GetAll(ctx context.Context) ([]*models.Workflow, error) {
	query := `
		SELECT
			id
		  , name
		  , description
		  , variables
		  , status
		  , metadata
		  , owner
		  , created_at
		  , updated_at
		  , deleted_at
		FROM workflows
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}

	defer func(ctx context.Context, r *WorkflowRepository) {
		err := rows.Close()
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to close rows", "error", err)
		}
	}(ctx, r)

	workflows := make([]*models.Workflow, 0)

	for rows.Next() {
		workflow, err := r.scanWorkflowBase(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		err = r.loadWorkflowTriggersAndSteps(ctx, workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to load workflow triggers and steps: %w", err)
		}

		workflows = append(workflows, workflow)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating workflows: %w", err)
	}

	return workflows, nil
}

func (r *WorkflowRepository) GetByID(ctx context.Context, id string) (*models.Workflow, error) {
	query := `
		SELECT
			id
		  , name
		  , description
		  , variables
		  , status
		  , metadata
		  , owner
		  , created_at
		  , updated_at
		  , deleted_at
		FROM workflows
		WHERE id = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRowContext(ctx, query, id)

	workflow, err := r.scanWorkflowBase(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to scan workflow: %w", err)
	}

	if err := r.loadWorkflowTriggersAndSteps(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to load workflow triggers and steps: %w", err)
	}

	return workflow, nil
}

// Save saves a workflow to the database.
func (r *WorkflowRepository) Save(ctx context.Context, workflow *models.Workflow) error {
	now := time.Now().UTC()

	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}

	workflow.UpdatedAt = now

	if workflow.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate workflow ID: %w", err)
		}

		workflow.ID = id.String()
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Convert complex fields to JSON
	variablesJSON, err := json.Marshal(workflow.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	metadataJSON, err := json.Marshal(workflow.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Save workflow base data
	workflowQuery := `
		INSERT INTO workflows (id, name, description,
variables, status, metadata, owner, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			variables = EXCLUDED.variables,
			status = EXCLUDED.status,
			metadata = EXCLUDED.metadata,
			owner = EXCLUDED.owner,
			updated_at = EXCLUDED.updated_at,
			deleted_at = EXCLUDED.deleted_at
	`

	_, err = tx.ExecContext(ctx, workflowQuery,
		workflow.ID,
		workflow.Name,
		workflow.Description,
		variablesJSON,
		workflow.Status,
		metadataJSON,
		workflow.Owner,
		workflow.CreatedAt,
		workflow.UpdatedAt,
		workflow.DeletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save workflow base: %w", err)
	}

	// Delete existing triggers and steps (for updates)
	_, err = tx.ExecContext(ctx, "DELETE FROM workflow_triggers WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing triggers: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM workflow_steps WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing steps: %w", err)
	}

	// Save triggers
	if err := r.saveWorkflowTriggers(ctx, tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow triggers: %w", err)
	}

	// Save steps
	if err := r.saveWorkflowSteps(ctx, tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow steps: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete soft deletes a workflow by setting deleted_at timestamp.
func (r *WorkflowRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE workflows SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Workflow doesn't exist or already deleted - this is not an error
		return nil
	}

	return nil
}

func (r *WorkflowRepository) loadWorkflowTriggersAndSteps(ctx context.Context, workflow *models.Workflow) error {
	triggersQuery := `
		SELECT 
			id
		  , name
		  , description
		  , trigger_id
		  , configuration
		FROM workflow_triggers
		WHERE workflow_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, triggersQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow triggers: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to close rows", "error", err)
		}
	}()

	var triggers []*models.WorkflowTrigger

	for rows.Next() {
		var (
			trigger    models.WorkflowTrigger
			configJSON []byte
		)

		err := rows.Scan(
			&trigger.ID,
			&trigger.Name,
			&trigger.Description,
			&trigger.TriggerID,
			&configJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to scan trigger: %w", err)
		}

		if configJSON != nil {
			err := json.Unmarshal(configJSON, &trigger.Configuration)
			if err != nil {
				return fmt.Errorf("failed to unmarshal trigger configuration: %w", err)
			}
		}

		triggers = append(triggers, &trigger)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating triggers: %w", err)
	}

	workflow.WorkflowTriggers = triggers

	// Load steps
	stepsQuery := `
		SELECT id, uid, name, action_id, configuration, conditional_language,
		       conditional_expression, on_success, on_failure, enabled
		FROM workflow_steps
		WHERE workflow_id = $1
		ORDER BY created_at
	`

	rows, err = r.db.QueryContext(ctx, stepsQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow steps: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()

	var steps []*models.WorkflowStep

	for rows.Next() {
		var (
			step                                       models.WorkflowStep
			configJSON                                 []byte
			conditionalLanguage, conditionalExpression sql.NullString
		)

		err := rows.Scan(
			&step.ID,
			&step.UID,
			&step.Name,
			&step.ActionID,
			&configJSON,
			&conditionalLanguage,
			&conditionalExpression,
			&step.OnSuccess,
			&step.OnFailure,
			&step.Enabled,
		)
		if err != nil {
			return fmt.Errorf("failed to scan step: %w", err)
		}

		if configJSON != nil {
			err := json.Unmarshal(configJSON, &step.Configuration)
			if err != nil {
				return fmt.Errorf("failed to unmarshal step configuration: %w", err)
			}
		}

		steps = append(steps, &step)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating steps: %w", err)
	}

	workflow.Steps = steps

	return nil
}

// saveWorkflowTriggers saves triggers for a workflow.
func (r *WorkflowRepository) saveWorkflowTriggers(ctx context.Context, tx *sql.Tx, workflow *models.Workflow) error {
	for _, trigger := range workflow.WorkflowTriggers {
		configJSON, err := json.Marshal(trigger.Configuration)
		if err != nil {
			return fmt.Errorf("failed to marshal trigger configuration: %w", err)
		}

		if len(trigger.ID) == 0 {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("failed to generate trigger ID: %w", err)
			}

			trigger.ID = id.String()
		}

		query := `
			INSERT INTO workflow_triggers (id, workflow_id, name, description, trigger_id, configuration)
			VALUES ($1, $2, $3, $4, $5, $6)
		`

		_, err = tx.ExecContext(ctx, query,
			trigger.ID,
			workflow.ID,
			trigger.Name,
			trigger.Description,
			trigger.TriggerID,
			configJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to save trigger: %w", err)
		}
	}

	return nil
}

// saveWorkflowSteps saves steps for a workflow.
func (r *WorkflowRepository) saveWorkflowSteps(ctx context.Context, tx *sql.Tx, workflow *models.Workflow) error {
	for _, step := range workflow.Steps {
		configJSON, err := json.Marshal(step.Configuration)
		if err != nil {
			return fmt.Errorf("failed to marshal step configuration: %w", err)
		}

		if step.ID == "" {
			id, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("failed to generate step ID: %w", err)
			}

			step.ID = id.String()
		}

		query := `
			INSERT INTO workflow_steps (id, workflow_id, uid, name, action_id, configuration,
			                          on_success, on_failure, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		
		`

		_, err = tx.ExecContext(ctx, query,
			step.ID,
			workflow.ID,
			step.UID,
			step.Name,
			step.ActionID,
			configJSON,
			step.OnSuccess,
			step.OnFailure,
			step.Enabled,
		)
		if err != nil {
			return fmt.Errorf("failed to save step: %w", err)
		}
	}

	return nil
}

func (r *WorkflowRepository) scanWorkflowBase(scanner interface {
	Scan(dest ...any) error
}) (*models.Workflow, error) {
	var (
		workflow                    models.Workflow
		variablesJSON, metadataJSON []byte
	)

	err := scanner.Scan(
		&workflow.ID,
		&workflow.Name,
		&workflow.Description,
		&variablesJSON,
		&workflow.Status,
		&metadataJSON,
		&workflow.Owner,
		&workflow.CreatedAt,
		&workflow.UpdatedAt,
		&workflow.DeletedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if variablesJSON != nil {
		err := json.Unmarshal(variablesJSON, &workflow.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
	}

	if metadataJSON != nil {
		err := json.Unmarshal(metadataJSON, &workflow.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &workflow, nil
}
