// Package persistence provides data storage abstraction layer for workflows and triggers.
package persistence

import (
	"context"
	"time"

	"github.com/dukex/operion/pkg/models"
)

type Persistence interface {
	// Schedule operations
	Schedules() ([]*models.Schedule, error)
	SaveSchedule(schedule *models.Schedule) error
	ScheduleByID(id string) (*models.Schedule, error)
	ScheduleBySourceID(sourceID string) (*models.Schedule, error)
	DeleteSchedule(id string) error
	DeleteScheduleBySourceID(sourceID string) error
	DueSchedules(before time.Time) ([]*models.Schedule, error)

	// Workflows operations
	Workflows(ctx context.Context) ([]*models.Workflow, error)
	SaveWorkflow(ctx context.Context, workflow *models.Workflow) error
	WorkflowByID(ctx context.Context, id string) (*models.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
	HealthCheck(ctx context.Context) error

	Close(ctx context.Context) error
}
