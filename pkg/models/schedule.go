package models

import (
	"errors"
	"time"

	"github.com/robfig/cron/v3"
)

// Schedule represents a scheduled task entry stored in the database.
// It contains the cron expression and precomputed next execution time
// to enable efficient centralized scheduling without individual timers.
type Schedule struct {
	// ID uniquely identifies this schedule entry
	ID string `json:"id" validate:"required"`

	// SourceID identifies the source that this schedule belongs to
	SourceID string `json:"source_id" validate:"required"`

	// CronExpression defines when this schedule should trigger
	// Uses standard 5-field cron format (minute hour day month weekday)
	CronExpression string `json:"cron_expression" validate:"required"`

	// NextDueAt is the precomputed next execution time
	// This allows efficient database queries for due schedules
	NextDueAt time.Time `json:"next_due_at" validate:"required"`

	// CreatedAt timestamp when this schedule was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp when this schedule was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Active indicates if this schedule is currently active
	// Inactive schedules are not processed by the poller
	Active bool `json:"active"`
}

// NewSchedule creates a new Schedule with the next execution time calculated.
func NewSchedule(id, sourceID, cronExpression string) (*Schedule, error) {
	now := time.Now().UTC()
	schedule := &Schedule{
		ID:             id,
		SourceID:       sourceID,
		CronExpression: cronExpression,
		CreatedAt:      now,
		UpdatedAt:      now,
		Active:         true,
	}

	// Calculate the first execution time from now (only for initial creation)
	if err := schedule.calculateNextDueAt(now); err != nil {
		return nil, err
	}

	return schedule, nil
}

// UpdateNextDueAt calculates and updates the next execution time based on current time.
func (s *Schedule) UpdateNextDueAt() error {
	// Use current time as reference
	return s.calculateNextDueAt(time.Now().UTC())
}

// calculateNextDueAt is the shared logic for calculating next execution time.
// referenceTime is the time to calculate the next execution from.
func (s *Schedule) calculateNextDueAt(referenceTime time.Time) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(s.CronExpression)

	if err != nil {
		return err
	}

	// Calculate next execution time from the reference time
	s.NextDueAt = cronSchedule.Next(referenceTime)
	s.UpdatedAt = time.Now().UTC()

	return nil
}

// IsDue checks if this schedule is due for execution at the given time.
func (s *Schedule) IsDue(now time.Time) bool {
	return s.Active && !s.NextDueAt.After(now)
}

// Validate performs validation on the schedule fields.
func (s *Schedule) Validate() error {
	if s.ID == "" {
		return ErrInvalidSchedule
	}

	if s.SourceID == "" {
		return ErrInvalidSchedule
	}

	if s.CronExpression == "" {
		return ErrInvalidSchedule
	}

	// Validate cron expression format
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(s.CronExpression)

	return err
}

var (
	// ErrInvalidSchedule is returned when schedule validation fails
	ErrInvalidSchedule = errors.New("invalid schedule configuration")
)
