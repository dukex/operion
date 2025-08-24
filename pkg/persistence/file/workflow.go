package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/dukex/operion/pkg/models"
)

// WorkflowRepository handles workflow-related file operations.
type WorkflowRepository struct {
	root string // File system root for storing workflows
}

// NewWorkflowRepository creates a new workflow repository.
func NewWorkflowRepository(root string) *WorkflowRepository {
	return &WorkflowRepository{root: root}
}

// GetAll returns all workflows from the file system.
func (wr *WorkflowRepository) GetAll(ctx context.Context) ([]*models.Workflow, error) {
	root := os.DirFS(wr.root + "/workflows")

	jsonFiles, err := fs.Glob(root, "*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow files: %w", err)
	}

	if len(jsonFiles) == 0 {
		return make([]*models.Workflow, 0), nil
	}

	workflows := make([]*models.Workflow, 0, len(jsonFiles))

	for _, file := range jsonFiles {
		workflow, err := wr.GetByID(ctx, file[:len(file)-5])
		if err != nil {
			return nil, err
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// GetByID retrieves a workflow by its ID from the file system.
func (wr *WorkflowRepository) GetByID(_ context.Context, workflowID string) (*models.Workflow, error) {
	filePath := filepath.Clean(path.Join(wr.root, "workflows", workflowID+".json"))

	body, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to fetch workflow %s: %w", workflowID, err)
	}

	var workflow models.Workflow

	err = json.Unmarshal(body, &workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workflow %s: %w", workflowID, err)
	}

	return &workflow, nil
}

// Save saves a workflow to the file system.
func (wr *WorkflowRepository) Save(_ context.Context, workflow *models.Workflow) error {
	err := os.MkdirAll(wr.root+"/workflows", 0750)
	if err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	now := time.Now().UTC()
	if workflow.CreatedAt.IsZero() {
		workflow.CreatedAt = now
	}

	workflow.UpdatedAt = now

	data, err := json.MarshalIndent(workflow, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workflow %s: %w", workflow.ID, err)
	}

	filePath := path.Join(wr.root+"/workflows", workflow.ID+".json")

	return os.WriteFile(filePath, data, 0600)
}

// Delete removes a workflow by its ID.
func (wr *WorkflowRepository) Delete(_ context.Context, id string) error {
	filePath := path.Join(wr.root+"/workflows", id+".json")

	err := os.Remove(filePath)

	if err != nil && os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete workflow %s: %w", id, err)
	}

	return nil
}

// FindTriggersBySourceEventAndProvider returns workflow triggers that match source ID, event type, provider ID, and workflow status.
func (wr *WorkflowRepository) FindTriggersBySourceEventAndProvider(ctx context.Context, sourceID, eventType, providerID string, status models.WorkflowStatus) ([]*models.TriggerNodeMatch, error) {
	// Get all workflows
	workflows, err := wr.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflows: %w", err)
	}

	var matches []*models.TriggerNodeMatch

	for _, workflow := range workflows {
		// Skip workflows that don't match the status filter
		if status != "" && workflow.Status != status {
			continue
		}

		// Check each node for trigger nodes that match
		for _, node := range workflow.Nodes {
			// Check if this is a trigger node matching our criteria
			if node.Category == models.CategoryTypeTrigger &&
				node.SourceID != nil && *node.SourceID == sourceID &&
				node.EventType != nil && *node.EventType == eventType &&
				node.ProviderID != nil && *node.ProviderID == providerID &&
				node.Enabled {
				match := &models.TriggerNodeMatch{
					WorkflowID:  workflow.ID,
					TriggerNode: node,
				}
				matches = append(matches, match)
			}
		}
	}

	return matches, nil
}
