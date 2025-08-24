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

		err = r.loadWorkflowTriggersAndNodes(ctx, workflow)
		if err != nil {
			return nil, fmt.Errorf("failed to load workflow triggers and nodes: %w", err)
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

	if err := r.loadWorkflowTriggersAndNodes(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to load workflow triggers and nodes: %w", err)
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

	// Delete existing nodes and connections (for updates)
	_, err = tx.ExecContext(ctx, "DELETE FROM workflow_connections WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing connections: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM workflow_nodes WHERE workflow_id = $1", workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to delete existing nodes: %w", err)
	}

	// Save nodes and connections
	if err := r.saveWorkflowNodes(ctx, tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow nodes: %w", err)
	}

	if err := r.saveWorkflowConnections(ctx, tx, workflow); err != nil {
		return fmt.Errorf("failed to save workflow connections: %w", err)
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

func (r *WorkflowRepository) loadWorkflowTriggersAndNodes(ctx context.Context, workflow *models.Workflow) error {
	// Triggers are now part of nodes, so we only load nodes with all trigger fields

	// Load nodes with trigger fields
	nodesQuery := `
		SELECT id, node_type, category, name, config, enabled, position_x, position_y, source_id, provider_id, event_type
		FROM workflow_nodes
		WHERE workflow_id = $1
		ORDER BY created_at
	`

	rows, err := r.db.QueryContext(ctx, nodesQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow nodes: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var nodes []*models.WorkflowNode

	for rows.Next() {
		var (
			node       models.WorkflowNode
			configJSON []byte
		)

		err := rows.Scan(
			&node.ID,
			&node.NodeType,
			&node.Category,
			&node.Name,
			&configJSON,
			&node.Enabled,
			&node.PositionX,
			&node.PositionY,
			&node.SourceID,
			&node.ProviderID,
			&node.EventType,
		)
		if err != nil {
			return fmt.Errorf("failed to scan node: %w", err)
		}

		if configJSON != nil {
			err := json.Unmarshal(configJSON, &node.Config)
			if err != nil {
				return fmt.Errorf("failed to unmarshal node configuration: %w", err)
			}
		}

		nodes = append(nodes, &node)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating nodes: %w", err)
	}

	workflow.Nodes = nodes

	// Load connections
	connectionsQuery := `
		SELECT id, source_node_id, source_port, target_node_id, target_port
		FROM workflow_connections
		WHERE workflow_id = $1
		ORDER BY created_at
	`

	rows, err = r.db.QueryContext(ctx, connectionsQuery, workflow.ID)
	if err != nil {
		return fmt.Errorf("failed to query workflow connections: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			r.logger.Error("failed to close rows", "error", err)
		}
	}()

	var connections []*models.Connection

	for rows.Next() {
		var (
			connection                                         models.Connection
			sourceNodeID, sourcePort, targetNodeID, targetPort string
		)

		err := rows.Scan(
			&connection.ID,
			&sourceNodeID,
			&sourcePort,
			&targetNodeID,
			&targetPort,
		)
		if err != nil {
			return fmt.Errorf("failed to scan connection: %w", err)
		}

		// Convert old database format to new Connection struct format
		connection.SourcePort = sourceNodeID + ":" + sourcePort
		connection.TargetPort = targetNodeID + ":" + targetPort

		connections = append(connections, &connection)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating connections: %w", err)
	}

	workflow.Connections = connections

	return nil
}

// Triggers are now saved as part of nodes - no separate trigger saving function needed

// saveWorkflowNodes saves nodes for a workflow.
func (r *WorkflowRepository) saveWorkflowNodes(ctx context.Context, tx *sql.Tx, workflow *models.Workflow) error {
	for _, node := range workflow.Nodes {
		configJSON, err := json.Marshal(node.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal node configuration: %w", err)
		}

		query := `
			INSERT INTO workflow_nodes (id, workflow_id, node_type, category, name, config, enabled, position_x, position_y, source_id, provider_id, event_type)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`

		_, err = tx.ExecContext(ctx, query,
			node.ID,
			workflow.ID,
			node.NodeType,
			node.Category,
			node.Name,
			configJSON,
			node.Enabled,
			node.PositionX,
			node.PositionY,
			node.SourceID,
			node.ProviderID,
			node.EventType,
		)
		if err != nil {
			return fmt.Errorf("failed to save node: %w", err)
		}
	}

	return nil
}

// saveWorkflowConnections saves connections for a workflow.
func (r *WorkflowRepository) saveWorkflowConnections(ctx context.Context, tx *sql.Tx, workflow *models.Workflow) error {
	for _, connection := range workflow.Connections {
		// Parse port IDs to extract node IDs and port names
		sourceNodeID, sourcePortName, sourceOK := models.ParsePortID(connection.SourcePort)
		if !sourceOK {
			return fmt.Errorf("invalid source port ID format: %s", connection.SourcePort)
		}

		targetNodeID, targetPortName, targetOK := models.ParsePortID(connection.TargetPort)
		if !targetOK {
			return fmt.Errorf("invalid target port ID format: %s", connection.TargetPort)
		}

		query := `
			INSERT INTO workflow_connections (id, workflow_id, source_node_id, source_port, target_node_id, target_port)
			VALUES ($1, $2, $3, $4, $5, $6)
		`

		_, err := tx.ExecContext(ctx, query,
			connection.ID,
			workflow.ID,
			sourceNodeID,
			sourcePortName,
			targetNodeID,
			targetPortName,
		)
		if err != nil {
			return fmt.Errorf("failed to save connection: %w", err)
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

// FindTriggersBySourceEventAndProvider returns trigger nodes matching the specified criteria.
func (r *WorkflowRepository) FindTriggersBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	query := `
		SELECT 
			w.id as workflow_id,
			n.id,
			n.node_type,
			n.category,
			n.name,
			n.config,
			n.enabled,
			n.position_x,
			n.position_y,
			n.source_id,
			n.provider_id,
			n.event_type
		FROM workflows w
		JOIN workflow_nodes n ON w.id = n.workflow_id
		WHERE w.deleted_at IS NULL
		  AND w.status = $1
		  AND n.category = 'trigger'
		  AND n.source_id = $2
		  AND n.event_type = $3
		  AND n.provider_id = $4
		ORDER BY w.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status, sourceID, eventType, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow triggers: %w", err)
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to close rows", "error", err)
		}
	}()

	var matches []*models.TriggerNodeMatch

	for rows.Next() {
		var (
			workflowID string
			node       models.WorkflowNode
			configJSON []byte
		)

		err := rows.Scan(
			&workflowID,
			&node.ID,
			&node.NodeType,
			&node.Category,
			&node.Name,
			&configJSON,
			&node.Enabled,
			&node.PositionX,
			&node.PositionY,
			&node.SourceID,
			&node.ProviderID,
			&node.EventType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trigger node match: %w", err)
		}

		if configJSON != nil {
			err := json.Unmarshal(configJSON, &node.Config)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal node configuration: %w", err)
			}
		}

		matches = append(matches, &models.TriggerNodeMatch{
			WorkflowID:  workflowID,
			TriggerNode: &node,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trigger matches: %w", err)
	}

	return matches, nil
}
