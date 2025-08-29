package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// ListWorkflowsRequest contains options for listing workflows.
type ListWorkflowsRequest struct {
	// Pagination
	Limit  int `validate:"min=1,max=100"`
	Offset int `validate:"min=0"`

	// Filtering
	OwnerID string
	Status  *models.WorkflowStatus

	// Sorting
	SortBy    string `validate:"oneof=created_at updated_at name"`
	SortOrder string `validate:"oneof=asc desc"`

	// Data Loading Control
	IncludeNodes       bool
	IncludeConnections bool
}

// ListWorkflowsResponse contains the result of listing workflows.
type ListWorkflowsResponse struct {
	Workflows   []*models.Workflow `json:"workflows"`
	TotalCount  int64              `json:"total_count"`
	HasNextPage bool               `json:"has_next_page"`
}

// ListWorkflows retrieves workflows with filtering, sorting, and pagination.
func (w *Workflow) ListWorkflows(ctx context.Context, req ListWorkflowsRequest) (*ListWorkflowsResponse, error) {
	// Validate and set defaults
	if err := w.validateListWorkflowsRequest(&req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Convert to persistence options
	opts := persistence.ListWorkflowsOptions{
		Limit:              req.Limit,
		Offset:             req.Offset,
		OwnerID:            req.OwnerID,
		Status:             req.Status,
		SortBy:             req.SortBy,
		SortOrder:          req.SortOrder,
		IncludeNodes:       req.IncludeNodes,
		IncludeConnections: req.IncludeConnections,
	}

	// Call persistence layer
	result, err := w.persistence.WorkflowRepository().ListWorkflows(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	// Convert to service response
	return &ListWorkflowsResponse{
		Workflows:   result.Workflows,
		TotalCount:  result.TotalCount,
		HasNextPage: result.HasNextPage,
	}, nil
}

// validateListWorkflowsRequest validates and sets defaults for the request.
func (w *Workflow) validateListWorkflowsRequest(req *ListWorkflowsRequest) error {
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 20
	}

	if req.Limit > 100 {
		req.Limit = 100
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	if req.SortBy == "" {
		req.SortBy = "created_at"
	}

	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Validate sort parameters against allowlist
	allowedSorts := []string{"created_at", "updated_at", "name"}
	validSort := false

	for _, allowed := range allowedSorts {
		if req.SortBy == allowed {
			validSort = true

			break
		}
	}

	if !validSort {
		return fmt.Errorf("invalid sort field '%s', allowed: %s", req.SortBy, strings.Join(allowedSorts, ", "))
	}

	// Validate sort order
	if req.SortOrder != "asc" && req.SortOrder != "desc" {
		return fmt.Errorf("invalid sort order '%s', allowed: asc, desc", req.SortOrder)
	}

	// Validate status if provided
	if req.Status != nil {
		allowedStatuses := []models.WorkflowStatus{
			models.WorkflowStatusDraft,
			models.WorkflowStatusPublished,
			models.WorkflowStatusUnpublished,
		}
		validStatus := false

		for _, allowed := range allowedStatuses {
			if *req.Status == allowed {
				validStatus = true

				break
			}
		}

		if !validStatus {
			return fmt.Errorf("invalid status '%s'", *req.Status)
		}
	}

	// Validate OwnerID format if provided (basic validation)
	if req.OwnerID != "" {
		req.OwnerID = strings.TrimSpace(req.OwnerID)
		if req.OwnerID == "" {
			return errors.New("owner ID cannot be empty or whitespace")
		}
	}

	return nil
}

// FetchAll retrieves all workflows (backward compatibility).
// Deprecated: Use ListWorkflows instead.
func (w *Workflow) FetchAll(ctx context.Context) ([]*models.Workflow, error) {
	result, err := w.ListWorkflows(ctx, ListWorkflowsRequest{
		Limit:     100,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}

	return result.Workflows, nil
}

// FetchAllByOwner retrieves all workflows for a specific owner (backward compatibility).
// Deprecated: Use ListWorkflows instead.
func (w *Workflow) FetchAllByOwner(ctx context.Context, ownerID string) ([]*models.Workflow, error) {
	result, err := w.ListWorkflows(ctx, ListWorkflowsRequest{
		OwnerID:   ownerID,
		Limit:     100,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, err
	}

	return result.Workflows, nil
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
