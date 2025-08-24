// Package workflow provides workflow publishing functionality for immutable execution.
package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/google/uuid"
)

// PublishingService handles workflow publishing operations for immutable execution.
type PublishingService struct {
	persistence persistence.Persistence
}

// NewPublishingService creates a new workflow publishing service.
func NewPublishingService(persistence persistence.Persistence) *PublishingService {
	return &PublishingService{
		persistence: persistence,
	}
}

// PublishWorkflow creates an immutable published version of a workflow
// This creates a snapshot that can be used for execution without being affected by future edits.
func (s *PublishingService) PublishWorkflow(ctx context.Context, workflowID string) (*models.Workflow, error) {
	// Get the original workflow
	originalWorkflow, err := s.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow for publishing: %w", err)
	}

	if originalWorkflow == nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Validate workflow can be published
	if err := s.validateForPublishing(originalWorkflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Create published version
	publishedWorkflow := s.createPublishedCopy(originalWorkflow)

	// Save the published workflow
	if err := s.persistence.WorkflowRepository().Save(ctx, publishedWorkflow); err != nil {
		return nil, fmt.Errorf("failed to save published workflow: %w", err)
	}

	// Update original workflow to reference the published version
	originalWorkflow.PublishedID = publishedWorkflow.ID

	originalWorkflow.UpdatedAt = time.Now()
	if err := s.persistence.WorkflowRepository().Save(ctx, originalWorkflow); err != nil {
		// Try to rollback the published workflow if updating original fails
		_ = s.persistence.WorkflowRepository().Delete(ctx, publishedWorkflow.ID)

		return nil, fmt.Errorf("failed to update original workflow with published ID: %w", err)
	}

	return publishedWorkflow, nil
}

// GetPublishedWorkflow retrieves a published workflow by its published ID.
func (s *PublishingService) GetPublishedWorkflow(ctx context.Context, publishedID string) (*models.Workflow, error) {
	workflow, err := s.persistence.WorkflowRepository().GetByID(ctx, publishedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get published workflow: %w", err)
	}

	if workflow == nil {
		return nil, fmt.Errorf("published workflow not found: %s", publishedID)
	}

	if workflow.Status != models.WorkflowStatusPublished {
		return nil, fmt.Errorf("workflow %s is not a published version", publishedID)
	}

	return workflow, nil
}

// GetActivePublishedVersion returns the currently published version of a workflow.
func (s *PublishingService) GetActivePublishedVersion(ctx context.Context, workflowID string) (*models.Workflow, error) {
	workflow, err := s.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow == nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	if workflow.PublishedID == "" {
		return nil, fmt.Errorf("workflow %s has no published version", workflowID)
	}

	return s.GetPublishedWorkflow(ctx, workflow.PublishedID)
}

// UnpublishWorkflow removes the published version and resets the workflow to draft state.
func (s *PublishingService) UnpublishWorkflow(ctx context.Context, workflowID string) error {
	workflow, err := s.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow == nil {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	if workflow.PublishedID == "" {
		return fmt.Errorf("workflow %s has no published version to unpublish", workflowID)
	}

	// Delete the published version
	if err := s.persistence.WorkflowRepository().Delete(ctx, workflow.PublishedID); err != nil {
		return fmt.Errorf("failed to delete published workflow: %w", err)
	}

	// Reset original workflow
	workflow.PublishedID = ""
	workflow.Status = models.WorkflowStatusDraft
	workflow.UpdatedAt = time.Now()

	if err := s.persistence.WorkflowRepository().Save(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update workflow after unpublishing: %w", err)
	}

	return nil
}

// IsPublished checks if a workflow has a published version.
func (s *PublishingService) IsPublished(ctx context.Context, workflowID string) (bool, error) {
	workflow, err := s.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return false, fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow == nil {
		return false, fmt.Errorf("workflow not found: %s", workflowID)
	}

	return workflow.PublishedID != "", nil
}

// validateForPublishing checks if a workflow is ready to be published.
func (s *PublishingService) validateForPublishing(workflow *models.Workflow) error {
	if workflow.Status == models.WorkflowStatusPublished {
		return errors.New("cannot publish a workflow that is already a published version")
	}

	if len(workflow.Nodes) == 0 {
		return errors.New("cannot publish workflow with no nodes")
	}

	// Validate all nodes have valid types
	for _, node := range workflow.Nodes {
		if node.NodeType == "" {
			return fmt.Errorf("node %s has no type specified", node.ID)
		}

		if node.ID == "" {
			return errors.New("found node with empty ID")
		}
	}

	// Validate connections reference valid nodes
	nodeMap := make(map[string]bool)
	for _, node := range workflow.Nodes {
		nodeMap[node.ID] = true
	}

	for _, conn := range workflow.Connections {
		// Parse port IDs to extract node IDs for validation
		sourceNodeID, _, sourceOK := models.ParsePortID(conn.SourcePort)
		if !sourceOK {
			return fmt.Errorf("connection has invalid source port ID format: %s", conn.SourcePort)
		}

		targetNodeID, _, targetOK := models.ParsePortID(conn.TargetPort)
		if !targetOK {
			return fmt.Errorf("connection has invalid target port ID format: %s", conn.TargetPort)
		}

		if !nodeMap[sourceNodeID] {
			return fmt.Errorf("connection references non-existent source node: %s", sourceNodeID)
		}

		if !nodeMap[targetNodeID] {
			return fmt.Errorf("connection references non-existent target node: %s", targetNodeID)
		}
	}

	// Must have at least one trigger node to be executable
	hasTriggerNode := false

	for _, node := range workflow.Nodes {
		if node.IsTriggerNode() {
			hasTriggerNode = true

			break
		}
	}

	if !hasTriggerNode {
		return errors.New("cannot publish workflow with no trigger nodes")
	}

	return nil
}

// createPublishedCopy creates an immutable copy of a workflow for execution.
func (s *PublishingService) createPublishedCopy(original *models.Workflow) *models.Workflow {
	now := time.Now()
	publishedID := uuid.New().String()

	// Create deep copy of the workflow
	published := &models.Workflow{
		ID:          publishedID,
		Name:        original.Name,
		Description: original.Description,
		Status:      models.WorkflowStatusPublished,
		ParentID:    original.ID, // Reference to original workflow
		PublishedID: "",          // Published workflows don't have their own published ID
		Variables:   copyMap(original.Variables),
		Metadata:    copyMap(original.Metadata),
		Owner:       original.Owner,
		CreatedAt:   now,
		UpdatedAt:   now,
		PublishedAt: &now,
	}

	// Trigger data is now part of nodes, so no separate triggers to copy

	// Deep copy nodes
	published.Nodes = make([]*models.WorkflowNode, len(original.Nodes))
	for i, node := range original.Nodes {
		published.Nodes[i] = &models.WorkflowNode{
			ID:         node.ID, // Keep same ID for execution consistency
			NodeType:   node.NodeType,
			Category:   node.Category,
			Config:     copyMap(node.Config),
			PositionX:  node.PositionX,
			PositionY:  node.PositionY,
			Name:       node.Name,
			Enabled:    node.Enabled,
			SourceID:   copyStringPointer(node.SourceID),
			ProviderID: copyStringPointer(node.ProviderID),
			EventType:  copyStringPointer(node.EventType),
		}
	}

	// Deep copy connections
	published.Connections = make([]*models.Connection, len(original.Connections))
	for i, conn := range original.Connections {
		published.Connections[i] = &models.Connection{
			ID:         uuid.New().String(),
			SourcePort: conn.SourcePort,
			TargetPort: conn.TargetPort,
		}
	}

	return published
}

// copyMap creates a deep copy of a map[string]any.
func copyMap(original map[string]any) map[string]any {
	if original == nil {
		return nil
	}

	result := make(map[string]any, len(original))
	for k, v := range original {
		result[k] = v // Note: this is a shallow copy of values
	}

	return result
}

// copyStringPointer creates a copy of a string pointer.
func copyStringPointer(original *string) *string {
	if original == nil {
		return nil
	}

	value := *original

	return &value
}
