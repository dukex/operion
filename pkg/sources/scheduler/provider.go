package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	schedulerModels "github.com/dukex/operion/pkg/sources/scheduler/models"
	schedulerPersistence "github.com/dukex/operion/pkg/sources/scheduler/persistence"
)

// SchedulerSourceProvider implements a centralized cron-based scheduler orchestrator
// that polls the database for due schedules and processes them regardless of their individual cron expressions.
type SchedulerSourceProvider struct {
	config               map[string]any
	logger               *slog.Logger
	schedulerPersistence schedulerPersistence.SchedulerPersistence
	callback             protocol.SourceEventCallback
	ticker               *time.Ticker
	done                 chan bool
	started              bool
	mu                   sync.RWMutex
}

// Start begins the centralized scheduler orchestrator.
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

// Stop gracefully shuts down the scheduler orchestrator.
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

// Validate checks if the scheduler orchestrator configuration is valid.
func (s *SchedulerSourceProvider) Validate() error {
	// Orchestrator validation: ensure persistence is available
	if s.schedulerPersistence == nil {
		return events.ErrInvalidEventData
	}

	// Orchestrator doesn't validate individual schedules - those are validated when created
	// Each schedule in the database has its own cron expression validation
	return nil
}

// pollSchedules is the centralized poller that runs every minute.
func (s *SchedulerSourceProvider) pollSchedules(ctx context.Context) {
	for {
		select {
		case <-s.done:
			return
		case <-ctx.Done():
			return
		case <-s.ticker.C:
			s.processDueSchedules(ctx)
		}
	}
}

// processDueSchedules queries database for ALL due schedules and publishes events
// This is the core orchestrator method that handles schedules with different cron expressions.
func (s *SchedulerSourceProvider) processDueSchedules(ctx context.Context) {
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
		if err := s.publishScheduleEvent(ctx, schedule); err != nil {
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

// getDueSchedules retrieves schedules that are due for execution.
func (s *SchedulerSourceProvider) getDueSchedules(now time.Time) ([]*schedulerModels.Schedule, error) {
	return s.schedulerPersistence.DueSchedules(now)
}

// updateSchedule saves the updated schedule back to the database.
func (s *SchedulerSourceProvider) updateSchedule(schedule *schedulerModels.Schedule) error {
	if err := s.schedulerPersistence.SaveSchedule(schedule); err != nil {
		return err
	}

	s.logger.Info("Schedule updated",
		"source_id", schedule.SourceID,
		"next_due_at", schedule.NextDueAt)

	return nil
}

// publishScheduleEvent publishes a source event for a due schedule.
func (s *SchedulerSourceProvider) publishScheduleEvent(ctx context.Context, schedule *schedulerModels.Schedule) error {
	now := time.Now()

	eventData := map[string]any{
		"cron_expression": schedule.CronExpression,
		"due_at":          schedule.NextDueAt.Format("2006-01-02 15:04"),
		"published_at":    now.Format("2006-01-02 15:04:05.000"),
	}

	return s.callback(ctx, schedule.SourceID, "scheduler", "ScheduleDue", eventData)
}

// ProviderLifecycle interface implementation

// Initialize sets up the provider with required dependencies.
func (s *SchedulerSourceProvider) Initialize(ctx context.Context, deps protocol.Dependencies) error {
	s.logger = deps.Logger

	// Initialize scheduler-specific persistence based on URL
	persistenceURL := os.Getenv("SCHEDULER_PERSISTENCE_URL")
	if persistenceURL == "" {
		return errors.New("scheduler provider requires SCHEDULER_PERSISTENCE_URL environment variable (e.g., file://./data/scheduler, postgres://...)")
	}
	
	persistence, err := s.createPersistence(persistenceURL)
	if err != nil {
		return err
	}

	s.schedulerPersistence = persistence

	return nil
}

// Configure configures the provider based on current workflow definitions.
func (s *SchedulerSourceProvider) Configure(workflows []*models.Workflow) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Configuring scheduler provider with workflows", "workflow_count", len(workflows))

	scheduleCount := 0

	for _, wf := range workflows {
		if wf.Status != models.WorkflowStatusActive {
			continue
		}

		for _, trigger := range wf.WorkflowTriggers {
			if cronExpr, exists := trigger.Configuration["cron_expression"]; exists {
				if s.processScheduleTrigger(wf.ID, trigger, cronExpr) {
					scheduleCount++
				}
			}
		}
	}

	s.logger.Info("Scheduler configuration completed", "created_schedules", scheduleCount)

	return nil
}

// Prepare performs final preparation before starting the provider.
func (s *SchedulerSourceProvider) Prepare(ctx context.Context) error {
	if s.schedulerPersistence == nil {
		return errors.New("scheduler persistence not initialized")
	}

	s.logger.Info("Scheduler provider prepared and ready")

	return nil
}

// processScheduleTrigger handles the creation of a schedule for a trigger with cron_expression.
// Returns true if a schedule was successfully created, false otherwise.
func (s *SchedulerSourceProvider) processScheduleTrigger(workflowID string, trigger *models.WorkflowTrigger, cronExpr any) bool {
	sourceID := trigger.SourceID
	if sourceID == "" {
		s.logger.Warn("Trigger has cron_expression but no source_id",
			"workflow_id", workflowID,
			"trigger_id", trigger.ID)

		return false
	}

	// Check if schedule already exists
	existingSchedule, err := s.schedulerPersistence.ScheduleBySourceID(sourceID)
	if err != nil {
		s.logger.Error("Failed to check existing schedule",
			"source_id", sourceID,
			"error", err)

		return false
	}

	if existingSchedule != nil {
		s.logger.Debug("Schedule already exists", "source_id", sourceID)

		return false
	}

	// Create new schedule
	cronStr, ok := cronExpr.(string)
	if !ok {
		s.logger.Warn("Invalid cron_expression type",
			"source_id", sourceID,
			"type", cronExpr)

		return false
	}

	schedule, err := schedulerModels.NewSchedule(sourceID, sourceID, cronStr)
	if err != nil {
		s.logger.Error("Failed to create schedule",
			"source_id", sourceID,
			"cron", cronStr,
			"error", err)

		return false
	}

	if err := s.schedulerPersistence.SaveSchedule(schedule); err != nil {
		s.logger.Error("Failed to save schedule",
			"source_id", sourceID,
			"error", err)

		return false
	}

	s.logger.Info("Created schedule",
		"source_id", sourceID,
		"cron", cronStr,
		"next_due_at", schedule.NextDueAt)

	return true
}

// createPersistence creates the appropriate persistence implementation based on URL scheme
func (s *SchedulerSourceProvider) createPersistence(persistenceURL string) (schedulerPersistence.SchedulerPersistence, error) {
	scheme := s.parsePersistenceScheme(persistenceURL)
	
	s.logger.Info("Initializing scheduler persistence", "scheme", scheme, "url", persistenceURL)

	switch scheme {
	case "file":
		// Extract path from file://path
		path := strings.TrimPrefix(persistenceURL, "file://")
		return schedulerPersistence.NewFilePersistence(path)
		
	case "postgres", "postgresql":
		// Future: implement database persistence
		// return schedulerPersistence.NewPostgresPersistence(persistenceURL)
		return nil, errors.New("postgres persistence for scheduler not yet implemented")
		
	case "mysql":
		// Future: implement database persistence  
		// return schedulerPersistence.NewMySQLPersistence(persistenceURL)
		return nil, errors.New("mysql persistence for scheduler not yet implemented")
		
	default:
		return nil, errors.New("unsupported scheduler persistence scheme: " + scheme + " (supported: file, postgres, mysql)")
	}
}

// parsePersistenceScheme extracts the scheme from a persistence URL
func (s *SchedulerSourceProvider) parsePersistenceScheme(persistenceURL string) string {
	parts := strings.Split(persistenceURL, "://")
	if len(parts) < 2 {
		return "file" // default to file if no scheme
	}
	return parts[0]
}
