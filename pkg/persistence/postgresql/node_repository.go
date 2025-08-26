package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukex/operion/pkg/models"
)

// NodeRepository handles node-related database operations.
type NodeRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewNodeRepository creates a new node repository.
func NewNodeRepository(db *sql.DB, logger *slog.Logger) *NodeRepository {
	return &NodeRepository{db: db, logger: logger}
}

// GetNodesFromPublishedWorkflow retrieves all nodes from a published workflow.
func (nr *NodeRepository) GetNodesFromPublishedWorkflow(ctx context.Context, publishedWorkflowID string) ([]*models.WorkflowNode, error) {
	query := `
		SELECT id, type, category, name, config, enabled, position_x, position_y, source_id, provider_id, event_type
		FROM workflow_nodes
		WHERE workflow_id = $1
		ORDER BY created_at
	`

	rows, err := nr.db.QueryContext(ctx, query, publishedWorkflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow nodes: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			nr.logger.ErrorContext(ctx, "failed to close rows", "error", closeErr)
		}
	}()

	var nodes []*models.WorkflowNode

	for rows.Next() {
		node, err := nr.scanNode(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}

		nodes = append(nodes, node)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating nodes: %w", err)
	}

	return nodes, nil
}

// GetNodeFromPublishedWorkflow retrieves a specific node from a published workflow.
func (nr *NodeRepository) GetNodeFromPublishedWorkflow(ctx context.Context, publishedWorkflowID, nodeID string) (*models.WorkflowNode, error) {
	query := `
		SELECT id, type, category, name, config, enabled, position_x, position_y, source_id, provider_id, event_type
		FROM workflow_nodes
		WHERE workflow_id = $1 AND id = $2
	`

	row := nr.db.QueryRowContext(ctx, query, publishedWorkflowID, nodeID)

	node, err := nr.scanNode(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("node not found: %s in workflow %s", nodeID, publishedWorkflowID)
		}

		return nil, fmt.Errorf("failed to scan node: %w", err)
	}

	return node, nil
}

// SaveNode saves a node to the database (insert or update).
func (nr *NodeRepository) SaveNode(ctx context.Context, workflowID string, node *models.WorkflowNode) error {
	configJSON, err := json.Marshal(node.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal node configuration: %w", err)
	}

	query := `
		INSERT INTO workflow_nodes (id, workflow_id, type, category, name, config, enabled, position_x, position_y, source_id, provider_id, event_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW())
		ON CONFLICT (id, workflow_id) DO UPDATE SET
			type = EXCLUDED.type,
			category = EXCLUDED.category,
			name = EXCLUDED.name,
			config = EXCLUDED.config,
			enabled = EXCLUDED.enabled,
			position_x = EXCLUDED.position_x,
			position_y = EXCLUDED.position_y,
			source_id = EXCLUDED.source_id,
			provider_id = EXCLUDED.provider_id,
			event_type = EXCLUDED.event_type,
			updated_at = EXCLUDED.updated_at
	`

	_, err = nr.db.ExecContext(ctx, query,
		node.ID,
		workflowID,
		node.Type,
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

	return nil
}

// UpdateNode updates an existing node in the database.
func (nr *NodeRepository) UpdateNode(ctx context.Context, workflowID string, node *models.WorkflowNode) error {
	// Check if node exists first
	_, err := nr.GetNodeFromPublishedWorkflow(ctx, workflowID, node.ID)
	if err != nil {
		return err
	}

	// Use SaveNode as it handles upsert logic
	return nr.SaveNode(ctx, workflowID, node)
}

// DeleteNode removes a node from the database.
func (nr *NodeRepository) DeleteNode(ctx context.Context, workflowID, nodeID string) error {
	query := `DELETE FROM workflow_nodes WHERE workflow_id = $1 AND id = $2`

	result, err := nr.db.ExecContext(ctx, query, workflowID, nodeID)
	if err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("node not found: %s in workflow %s", nodeID, workflowID)
	}

	return nil
}

// GetNodesByWorkflow retrieves all nodes for a specific workflow.
func (nr *NodeRepository) GetNodesByWorkflow(ctx context.Context, workflowID string) ([]*models.WorkflowNode, error) {
	return nr.GetNodesFromPublishedWorkflow(ctx, workflowID)
}

// FindTriggerNodesBySourceEventAndProvider finds trigger nodes matching the specified criteria.
func (nr *NodeRepository) FindTriggerNodesBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	query := `
		SELECT 
			w.id as workflow_id,
			n.id,
			n.type,
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
		  AND n.enabled = true
		ORDER BY w.created_at DESC
	`

	rows, err := nr.db.QueryContext(ctx, query, status, sourceID, eventType, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow triggers: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			nr.logger.ErrorContext(ctx, "failed to close rows", "error", closeErr)
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
			&node.Type,
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

// scanNode scans a node from a database row.
func (nr *NodeRepository) scanNode(scanner interface {
	Scan(dest ...any) error
}) (*models.WorkflowNode, error) {
	var (
		node       models.WorkflowNode
		configJSON []byte
	)

	err := scanner.Scan(
		&node.ID,
		&node.Type,
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
		return nil, err
	}

	if configJSON != nil {
		err := json.Unmarshal(configJSON, &node.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal node configuration: %w", err)
		}
	} else {
		node.Config = make(map[string]any)
	}

	return &node, nil
}
