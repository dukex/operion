package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dukex/operion/pkg/models"
)

// WorkflowRepository handles workflow-related database operations
type WorkflowRepository struct {
	db *sql.DB
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
	return &WorkflowRepository{db: db}
}

// GetAll returns all workflows from the database
func (r *WorkflowRepository) GetAll() ([]*models.Workflow, error) {
	query := `
		SELECT id, name, description, variables, 
		       status, metadata, owner, created_at, updated_at, deleted_at
		FROM workflows
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflows: %w", err)
	}
	defer rows.Close()

	workflows := make([]*models.Workflow, 0)
	for rows.Next() {
		workflow, err := r.scanWorkflowBase(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}

		// Load triggers and steps
		if err := r.loadWorkflowTriggersAndSteps(workflow); err != nil {
			return nil, fmt.Errorf("failed to load workflow triggers and steps: %w", err)
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating workflows: %w", err)
	}

	return workflows, nil
}

// GetByID returns a workflow by its ID
func (r *WorkflowRepository) GetByID(id string) (*models.Workflow, error) {
	query := `
		SELECT w.id, w.name, w.description, w.variables, 
		       w.status, w.metadata, w.owner, w.created_at, w.updated_at, w.deleted_at
		FROM workflows w
		WHERE w.id = $1 AND w.deleted_at IS NULL
	`

	row := r.db.QueryRow(query, id)
	workflow, err := r.scanWorkflowBase(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan workflow: %w", err)
	}

	// Load triggers and steps
	if err := r.loadWorkflowTriggersAndSteps(workflow); err != nil {
		return nil, fmt.Errorf("failed to load workflow triggers and steps: %w", err)
	}

	return workflow, nil
}

// Save saves a workflow to the database
func (r *WorkflowRepository) Save(workflow *models.Workflow) error {
	now := time.Now()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}
	workflow.UpdatedAt = now

	// Start transaction
	tx, err := r.db.Begin()
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
		INSERT INTO workflows (id, name, description, variables, status, metadata, owner, created_at, updated_at, deleted_at)
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

	_, err = tx.Exec(workflowQuery,
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
	_, err = tx.Exec("DELETE FROM workflow_triggers WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing triggers: %w", err)
	}

	_, err = tx.Exec("DELETE FROM workflow_steps WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing steps: %w", err)
	}

	// Save triggers
	if err := r.saveWorkflowTriggers(tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow triggers: %w", err)
	}

	// Save steps
	if err := r.saveWorkflowSteps(tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow steps: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete soft deletes a workflow by setting deleted_at timestamp
func (r *WorkflowRepository) Delete(id string) error {
	query := `UPDATE workflows SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(query, id)
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

// loadWorkflowTriggersAndSteps loads triggers and steps for a workflow
func (r *WorkflowRepository) loadWorkflowTriggersAndSteps(workflow *models.Workflow) error {
	// Load triggers
	triggersQuery := `
		SELECT id, name, description, trigger_id, configuration
		FROM workflow_triggers
		WHERE workflow_id = $1
		ORDER BY created_at
	`
	
	rows, err := r.db.Query(triggersQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow triggers: %w", err)
	}
	defer rows.Close()

	var triggers []*models.WorkflowTrigger
	for rows.Next() {
		var trigger models.WorkflowTrigger
		var configJSON []byte
		
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
			if err := json.Unmarshal(configJSON, &trigger.Configuration); err != nil {
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
	
	rows, err = r.db.Query(stepsQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow steps: %w", err)
	}
	defer rows.Close()

	var steps []*models.WorkflowStep
	for rows.Next() {
		var step models.WorkflowStep
		var configJSON []byte
		var conditionalLanguage, conditionalExpression sql.NullString
		
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
			if err := json.Unmarshal(configJSON, &step.Configuration); err != nil {
				return fmt.Errorf("failed to unmarshal step configuration: %w", err)
			}
		}

		if conditionalLanguage.Valid && conditionalExpression.Valid {
			step.Conditional.Language = conditionalLanguage.String
			step.Conditional.Expression = conditionalExpression.String
		}

		steps = append(steps, &step)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating steps: %w", err)
	}
	workflow.Steps = steps

	return nil
}

// saveWorkflowTriggers saves triggers for a workflow
func (r *WorkflowRepository) saveWorkflowTriggers(tx *sql.Tx, workflow *models.Workflow) error {
	for _, trigger := range workflow.WorkflowTriggers {
		configJSON, err := json.Marshal(trigger.Configuration)
		if err != nil {
			return fmt.Errorf("failed to marshal trigger configuration: %w", err)
		}

		query := `
			INSERT INTO workflow_triggers (id, workflow_id, name, description, trigger_id, configuration)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		
		_, err = tx.Exec(query,
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

// saveWorkflowSteps saves steps for a workflow
func (r *WorkflowRepository) saveWorkflowSteps(tx *sql.Tx, workflow *models.Workflow) error {
	for _, step := range workflow.Steps {
		configJSON, err := json.Marshal(step.Configuration)
		if err != nil {
			return fmt.Errorf("failed to marshal step configuration: %w", err)
		}

		var conditionalLanguage, conditionalExpression *string
		if step.Conditional.Language != "" {
			conditionalLanguage = &step.Conditional.Language
			conditionalExpression = &step.Conditional.Expression
		}

		query := `
			INSERT INTO workflow_steps (id, workflow_id, uid, name, action_id, configuration, 
			                           conditional_language, conditional_expression, on_success, on_failure, enabled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		
		_, err = tx.Exec(query,
			step.ID,
			workflow.ID,
			step.UID,
			step.Name,
			step.ActionID,
			configJSON,
			conditionalLanguage,
			conditionalExpression,
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

// scanWorkflowBase scans a database row into a Workflow model (base fields only)
func (r *WorkflowRepository) scanWorkflowBase(scanner interface {
	Scan(dest ...any) error
}) (*models.Workflow, error) {
	var workflow models.Workflow
	var variablesJSON, metadataJSON []byte

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
		if err := json.Unmarshal(variablesJSON, &workflow.Variables); err != nil {
			return nil, fmt.Errorf("failed to unmarshal variables: %w", err)
		}
	}

	if metadataJSON != nil {
		if err := json.Unmarshal(metadataJSON, &workflow.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &workflow, nil
}