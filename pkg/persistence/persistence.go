// Package persistence provides data storage abstraction layer for workflows and triggers.
package persistence

import (
	"time"

	"github.com/dukex/operion/pkg/models"
)

type Persistence interface {
	Workflows() ([]*models.Workflow, error)
	SaveWorkflow(workflow *models.Workflow) error
	WorkflowByID(id string) (*models.Workflow, error)
	DeleteWorkflow(id string) error

	// Schedule operations
	Schedules() ([]*models.Schedule, error)
	SaveSchedule(schedule *models.Schedule) error
	ScheduleByID(id string) (*models.Schedule, error)
	ScheduleBySourceID(sourceID string) (*models.Schedule, error)
	DeleteSchedule(id string) error
	DeleteScheduleBySourceID(sourceID string) error
	DueSchedules(before time.Time) ([]*models.Schedule, error)

	HealthCheck() error
	Close() error
}
