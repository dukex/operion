package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
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

// GetAllByOwner returns all workflows for a specific owner from the file system.
func (wr *WorkflowRepository) GetAllByOwner(ctx context.Context, ownerID string) ([]*models.Workflow, error) {
	allWorkflows, err := wr.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all workflows: %w", err)
	}

	workflows := make([]*models.Workflow, 0)

	for _, workflow := range allWorkflows {
		if workflow.Owner == ownerID {
			workflows = append(workflows, workflow)
		}
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

// GetCurrentWorkflow returns the current version (published if exists, otherwise draft).
func (wr *WorkflowRepository) GetCurrentWorkflow(ctx context.Context, workflowGroupID string) (*models.Workflow, error) {
	// Try published first
	published, err := wr.GetPublishedWorkflow(ctx, workflowGroupID)
	if err != nil {
		return nil, err
	}

	if published != nil {
		return published, nil
	}

	// Fall back to draft
	return wr.GetDraftWorkflow(ctx, workflowGroupID)
}

// GetDraftWorkflow returns the draft version of a workflow group.
func (wr *WorkflowRepository) GetDraftWorkflow(ctx context.Context, workflowGroupID string) (*models.Workflow, error) {
	workflows, err := wr.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all workflows: %w", err)
	}

	var latestDraft *models.Workflow
	for _, workflow := range workflows {
		if workflow.WorkflowGroupID == workflowGroupID && workflow.Status == models.WorkflowStatusDraft {
			if latestDraft == nil || workflow.CreatedAt.After(latestDraft.CreatedAt) {
				latestDraft = workflow
			}
		}
	}

	return latestDraft, nil
}

// GetPublishedWorkflow returns the published version of a workflow group.
func (wr *WorkflowRepository) GetPublishedWorkflow(ctx context.Context, workflowGroupID string) (*models.Workflow, error) {
	workflows, err := wr.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all workflows: %w", err)
	}

	var currentPublished *models.Workflow
	for _, workflow := range workflows {
		if workflow.WorkflowGroupID == workflowGroupID && workflow.Status == models.WorkflowStatusPublished {
			if currentPublished == nil || workflow.CreatedAt.After(currentPublished.CreatedAt) {
				currentPublished = workflow
			}
		}
	}

	return currentPublished, nil
}

// PublishWorkflow handles the publish operation.
func (wr *WorkflowRepository) PublishWorkflow(ctx context.Context, workflowID string) error {
	// Get the workflow being published
	workflow, err := wr.GetByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	if workflow == nil {
		return fmt.Errorf("workflow not found: %s", workflowID)
	}

	// Get all workflows to find ones in the same group
	allWorkflows, err := wr.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get all workflows: %w", err)
	}

	// Set all other workflows in group to unpublished
	for _, wf := range allWorkflows {
		if wf.WorkflowGroupID == workflow.WorkflowGroupID && wf.Status == models.WorkflowStatusPublished {
			wf.Status = models.WorkflowStatusUnpublished

			wf.UpdatedAt = time.Now().UTC()
			if err := wr.Save(ctx, wf); err != nil {
				return fmt.Errorf("failed to unpublish workflow %s: %w", wf.ID, err)
			}
		}
	}

	// Set current workflow to published
	workflow.Status = models.WorkflowStatusPublished

	workflow.UpdatedAt = time.Now().UTC()
	if workflow.PublishedAt == nil {
		now := time.Now().UTC()
		workflow.PublishedAt = &now
	}

	return wr.Save(ctx, workflow)
}

// CreateDraftFromPublished creates a draft copy from published version.
func (wr *WorkflowRepository) CreateDraftFromPublished(ctx context.Context, workflowGroupID string) (*models.Workflow, error) {
	// Check if draft already exists
	existingDraft, err := wr.GetDraftWorkflow(ctx, workflowGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing draft: %w", err)
	}

	if existingDraft != nil {
		return existingDraft, nil
	}

	// Get published workflow
	publishedWorkflow, err := wr.GetPublishedWorkflow(ctx, workflowGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get published workflow: %w", err)
	}

	if publishedWorkflow == nil {
		return nil, persistence.NewWorkflowGroupError("CreateDraftFromPublished", workflowGroupID, persistence.ErrPublishedWorkflowNotFound)
	}

	// Create draft copy
	draftWorkflow := *publishedWorkflow

	// Generate new ID (simple approach for file-based)
	draftWorkflow.ID = workflowGroupID + "-draft-" + strconv.FormatInt(time.Now().Unix(), 10)

	// Set as draft
	draftWorkflow.Status = models.WorkflowStatusDraft
	draftWorkflow.CreatedAt = time.Now().UTC()
	draftWorkflow.UpdatedAt = time.Now().UTC()
	draftWorkflow.PublishedAt = nil

	// Save the draft
	if err := wr.Save(ctx, &draftWorkflow); err != nil {
		return nil, fmt.Errorf("failed to save draft workflow: %w", err)
	}

	return &draftWorkflow, nil
}
