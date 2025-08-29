package services

import (
	"context"
	"fmt"
	"time"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/google/uuid"
)

var (
	// ErrWorkflowNotFound is returned when a workflow is not found.
	ErrWorkflowNotFound = persistence.ErrWorkflowNotFound
)

type Workflow struct {
	persistence persistence.Persistence
}

// NewWorkflow creates a new workflow service.
func NewWorkflow(persistence persistence.Persistence) *Workflow {
	return &Workflow{
		persistence: persistence,
	}
}

// HealthCheck checks the health of the persistence layer.
func (w *Workflow) HealthCheck(ctx context.Context) (string, bool) {
	if w.persistence == nil {
		return "Persistence layer not initialized", false
	}

	err := w.persistence.HealthCheck(ctx)
	if err != nil {
		return "Persistence layer is unhealthy: " + err.Error(), false
	}

	return "Persistence layer is healthy", true
}

// FetchAll retrieves all workflows.
func (w *Workflow) FetchAll(ctx context.Context) ([]*models.Workflow, error) {
	workflows, err := w.persistence.WorkflowRepository().GetAll(ctx)
	if err != nil {
		return make([]*models.Workflow, 0), err
	}

	return workflows, nil
}

// FetchAllByOwner retrieves all workflows for a specific owner.
func (w *Workflow) FetchAllByOwner(ctx context.Context, ownerID string) ([]*models.Workflow, error) {
	workflows, err := w.persistence.WorkflowRepository().GetAllByOwner(ctx, ownerID)
	if err != nil {
		return make([]*models.Workflow, 0), err
	}

	return workflows, nil
}

// FetchByID retrieves a workflow by its ID.
func (w *Workflow) FetchByID(ctx context.Context, id string) (*models.Workflow, error) {
	workflow, err := w.persistence.WorkflowRepository().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if workflow == nil {
		return nil, ErrWorkflowNotFound
	}

	return workflow, nil
}

// Create adds a new workflow to the repository.
func (w *Workflow) Create(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	now := time.Now().UTC()
	workflow.ID = uuid.New().String()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	if workflow.Status == "" {
		workflow.Status = models.WorkflowStatusDraft
	}

	err := w.persistence.WorkflowRepository().Save(ctx, workflow)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// Update modifies an existing workflow by its ID.
func (w *Workflow) Update(
	ctx context.Context,
	workflowID string,
	workflow *models.Workflow,
) (*models.Workflow, error) {
	existing, err := w.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if existing == nil {
		return nil, ErrWorkflowNotFound
	}

	workflow.ID = workflowID
	workflow.CreatedAt = existing.CreatedAt
	workflow.UpdatedAt = time.Now().UTC()

	err = w.persistence.WorkflowRepository().Save(ctx, workflow)
	if err != nil {
		return nil, err
	}

	return workflow, nil
}

// Delete removes a workflow by its ID.
func (w *Workflow) Delete(ctx context.Context, workflowID string) error {
	existing, err := w.persistence.WorkflowRepository().GetByID(ctx, workflowID)
	if err != nil {
		return err
	}

	if existing == nil {
		return ErrWorkflowNotFound
	}

	err = w.persistence.WorkflowRepository().Delete(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	return nil
}
