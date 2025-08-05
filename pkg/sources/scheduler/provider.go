package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
)

// SchedulerSourceProvider implements a centralized cron-based scheduler orchestrator
// that polls the database for due schedules and processes them regardless of their individual cron expressions
type SchedulerSourceProvider struct {
	config      map[string]any
	logger      *slog.Logger
	persistence persistence.Persistence
	callback    protocol.SourceEventCallback
	ticker      *time.Ticker
	done        chan bool
	started     bool
	mu          sync.RWMutex
}

// Start begins the centralized scheduler orchestrator
func (s *SchedulerSourceProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return nil
	}

	s.callback = callback
	s.logger.Info("Starting centralized scheduler orchestrator")

	// Start centralized poller (runs every minute to check all due schedules)
	s.ticker = time.NewTicker(1 * time.Minute)
	s.done = make(chan bool)
	s.started = true

	go s.pollSchedules(ctx)

	s.logger.Info("Centralized scheduler orchestrator started successfully")
	return nil
}

// Stop gracefully shuts down the scheduler orchestrator
func (s *SchedulerSourceProvider) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	s.logger.Info("Stopping scheduler orchestrator")

	if s.ticker != nil {
		s.ticker.Stop()
	}

	select {
	case s.done <- true:
	default:
	}

	s.started = false
	s.logger.Info("Scheduler orchestrator stopped successfully")
	return nil
}

// Validate checks if the scheduler orchestrator configuration is valid
func (s *SchedulerSourceProvider) Validate() error {
	// Orchestrator validation: ensure persistence is available
	if s.persistence == nil {
		return events.ErrInvalidEventData
	}

	// Orchestrator doesn't validate individual schedules - those are validated when created
	// Each schedule in the database has its own cron expression validation
	return nil
}

// pollSchedules is the centralized poller that runs every minute
func (s *SchedulerSourceProvider) pollSchedules(ctx context.Context) {
	for {
		select {
		case <-s.done:
			return
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			s.processDueSchedules()
		}
	}
}

// processDueSchedules queries database for ALL due schedules and publishes events
// This is the core orchestrator method that handles schedules with different cron expressions
func (s *SchedulerSourceProvider) processDueSchedules() {
	now := time.Now().UTC()

	// Query database for ALL schedules that are due, regardless of cron expression
	dueSchedules, err := s.getDueSchedules(now)
	if err != nil {
		s.logger.Error("Failed to get due schedules", "error", err)
		return
	}

	if len(dueSchedules) > 0 {
		s.logger.Info("Processing due schedules", "count", len(dueSchedules))
	}

	for _, schedule := range dueSchedules {
		s.logger.Info("Processing due schedule",
			"source_id", schedule.SourceID,
			"cron_expression", schedule.CronExpression,
			"due_at", schedule.NextDueAt)

		// Publish source event (includes schedule's own cron expression)
		if err := s.publishScheduleEvent(schedule); err != nil {
			s.logger.Error("Failed to publish schedule event",
				"source_id", schedule.SourceID,
				"error", err)

			continue
		}

		// Update next execution time using schedule's own cron expression
		if err := schedule.UpdateNextDueAt(); err != nil {
			s.logger.Error("Failed to update next due at",
				"source_id", schedule.SourceID,
				"error", err)

			continue
		}

		// Save updated schedule back to database
		if err := s.updateSchedule(schedule); err != nil {
			s.logger.Error("Failed to update schedule",
				"source_id", schedule.SourceID,
				"error", err)
		}
	}
}

// getDueSchedules retrieves schedules that are due for execution
func (s *SchedulerSourceProvider) getDueSchedules(now time.Time) ([]*models.Schedule, error) {
	return s.persistence.DueSchedules(now)
}

// updateSchedule saves the updated schedule back to the database
func (s *SchedulerSourceProvider) updateSchedule(schedule *models.Schedule) error {
	if err := s.persistence.SaveSchedule(schedule); err != nil {
		return err
	}

	s.logger.Info("Schedule updated",
		"source_id", schedule.SourceID,
		"next_due_at", schedule.NextDueAt)
	return nil
}

// publishScheduleEvent publishes a source event for a due schedule
func (s *SchedulerSourceProvider) publishScheduleEvent(schedule *models.Schedule) error {
	now := time.Now()

	eventData := map[string]any{
		"cron_expression": schedule.CronExpression,
		"due_at":          schedule.NextDueAt.Format("2006-01-02 15:04"),
		"published_at":    now.Format("2006-01-02 15:04:05.000"),
	}

	return s.callback(context.Background(), schedule.SourceID, "scheduler", "ScheduleDue", eventData)
}
