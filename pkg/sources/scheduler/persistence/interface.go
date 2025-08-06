package persistence

import (
	"time"

	"github.com/dukex/operion/pkg/sources/scheduler/models"
)

// SchedulerPersistence defines the persistence interface for the scheduler provider.
// This interface is specific to scheduler needs and isolated from core persistence.
type SchedulerPersistence interface {
	// Schedule operations
	SaveSchedule(schedule *models.Schedule) error
	ScheduleByID(id string) (*models.Schedule, error)
	ScheduleBySourceID(sourceID string) (*models.Schedule, error)
	Schedules() ([]*models.Schedule, error)
	DueSchedules(before time.Time) ([]*models.Schedule, error)
	DeleteSchedule(id string) error
	DeleteScheduleBySourceID(sourceID string) error

	// Health and lifecycle
	HealthCheck() error
	Close() error
}