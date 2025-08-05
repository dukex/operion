// Package persistence provides data storage abstraction layer for workflows and triggers.
package persistence

import (
	"context"

	"github.com/dukex/operion/pkg/models"
)

type Persistence interface {
	Workflows(ctx context.Context) ([]*models.Workflow, error)
	SaveWorkflow(ctx context.Context, workflow *models.Workflow) error
	WorkflowByID(ctx context.Context, id string) (*models.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) error

	Close(ctx context.Context) error
}
