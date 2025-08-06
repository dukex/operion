// Package persistence provides data storage abstraction layer for workflows and triggers.
package persistence

import (
	"context"

	"github.com/dukex/operion/pkg/models"
)

type Persistence interface {
	// Workflows operations
	Workflows(ctx context.Context) ([]*models.Workflow, error)
	SaveWorkflow(ctx context.Context, workflow *models.Workflow) error
	WorkflowByID(ctx context.Context, id string) (*models.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) error

	// Trigger operations
	WorkflowTriggersBySourceID(ctx context.Context, sourceID string, status models.WorkflowStatus) ([]*models.TriggerMatch, error)
	WorkflowTriggersBySourceAndEvent(ctx context.Context, sourceID, eventType string, status models.WorkflowStatus) ([]*models.TriggerMatch, error)

	Close(ctx context.Context) error
}
