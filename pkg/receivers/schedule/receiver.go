// Package schedule provides schedule-based receiver implementation for the receiver pattern.
package schedule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/robfig/cron/v3"
)

const TriggerTopic = "operion.trigger"

type ScheduleReceiver struct {
	sources  []protocol.SourceConfig
	eventBus eventbus.EventBus
	logger   *slog.Logger
	cron     *cron.Cron
	jobs     map[string]cron.EntryID // maps cron expression to entry ID
	mutex    sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	config   protocol.ReceiverConfig
}

// ScheduleConfig represents configuration for a schedule source
type ScheduleConfig struct {
	Name     string `json:"name"`
	CronExpr string `json:"cron"`
	Enabled  bool   `json:"enabled"`
	Timezone string `json:"timezone"`
}

// NewScheduleReceiver creates a new schedule receiver
func NewScheduleReceiver(eventBus eventbus.EventBus, logger *slog.Logger) *ScheduleReceiver {
	return &ScheduleReceiver{
		eventBus: eventBus,
		logger:   logger.With("module", "schedule_receiver"),
		jobs:     make(map[string]cron.EntryID),
	}
}

func (r *ScheduleReceiver) Configure(config protocol.ReceiverConfig) error {
	r.config = config

	// Filter schedule sources
	r.sources = make([]protocol.SourceConfig, 0)
	for _, source := range config.Sources {
		if source.Type == "schedule" {
			r.sources = append(r.sources, source)
		}
	}

	return r.Validate()
}

func (r *ScheduleReceiver) Validate() error {
	if len(r.sources) == 0 {
		return errors.New("no schedule sources configured")
	}

	for _, source := range r.sources {
		if source.Name == "" {
			return errors.New("schedule source name is required")
		}

		cronExpr, ok := source.Configuration["cron"].(string)
		if !ok || cronExpr == "" {
			return fmt.Errorf("cron expression required for schedule source %s", source.Name)
		}

		// Validate cron expression
		if _, err := cron.ParseStandard(cronExpr); err != nil {
			return fmt.Errorf("invalid cron expression '%s' for source %s: %w", cronExpr, source.Name, err)
		}
	}

	return nil
}

func (r *ScheduleReceiver) Start(ctx context.Context) error {
	r.logger.Info("Starting schedule receiver", "sources_count", len(r.sources))
	r.ctx, r.cancel = context.WithCancel(ctx)

	// Initialize cron scheduler
	r.cron = cron.New(cron.WithChain(
		cron.SkipIfStillRunning(cron.DefaultLogger),
		cron.Recover(cron.DefaultLogger),
	))

	// Start cron jobs for each source
	for _, source := range r.sources {
		if err := r.startScheduleSource(source); err != nil {
			r.logger.Error("Failed to start schedule source", "source", source.Name, "error", err)
			return err
		}
	}

	// Start the cron scheduler
	r.cron.Start()
	r.logger.Info("Schedule receiver started successfully")

	return nil
}

func (r *ScheduleReceiver) startScheduleSource(source protocol.SourceConfig) error {
	logger := r.logger.With("source", source.Name)

	cronExpr := source.Configuration["cron"].(string)
	enabled := getEnabledFlag(source.Configuration)

	if !enabled {
		logger.Info("Schedule source is disabled, skipping")
		return nil
	}

	logger.Info("Starting schedule source", "cron", cronExpr)

	// Create job function that publishes trigger events
	jobFunc := func() {
		r.publishScheduleTriggerEvent(source)
	}

	// Add cron job
	entryID, err := r.cron.AddFunc(cronExpr, jobFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job for source %s: %w", source.Name, err)
	}

	// Store job reference
	r.mutex.Lock()
	r.jobs[source.Name] = entryID
	r.mutex.Unlock()

	logger.Info("Added cron job for schedule source", "cron", cronExpr, "entry_id", entryID)
	return nil
}

func (r *ScheduleReceiver) publishScheduleTriggerEvent(source protocol.SourceConfig) {
	logger := r.logger.With("source", source.Name)
	logger.Debug("Publishing schedule trigger event")

	// Create trigger data
	now := time.Now().UTC()
	triggerData := map[string]interface{}{
		"timestamp": now.Format(time.RFC3339),
		"cron":      source.Configuration["cron"],
		"source":    source.Name,
	}

	// Create original data (same as trigger data for schedules)
	originalData := map[string]interface{}{
		"timestamp": now.Format(time.RFC3339),
		"cron":      source.Configuration["cron"],
		"source":    source.Name,
	}

	// Create and publish trigger event
	triggerEvent := events.NewTriggerEvent("schedule", source.Name, triggerData, originalData)

	// Publish to trigger topic
	go func() {
		if err := r.eventBus.Publish(context.Background(), TriggerTopic, triggerEvent); err != nil {
			logger.Error("Failed to publish schedule trigger event", "error", err)
		} else {
			logger.Debug("Published schedule trigger event successfully")
		}
	}()
}

func (r *ScheduleReceiver) Stop(ctx context.Context) error {
	r.logger.Info("Stopping schedule receiver")

	if r.cancel != nil {
		r.cancel()
	}

	if r.cron != nil {
		r.cron.Stop()
		r.logger.Info("Stopped cron scheduler")
	}

	r.mutex.Lock()
	r.jobs = make(map[string]cron.EntryID)
	r.mutex.Unlock()

	return nil
}

// Helper functions

func getEnabledFlag(config map[string]interface{}) bool {
	if enabled, exists := config["enabled"]; exists {
		if enabledBool, ok := enabled.(bool); ok {
			return enabledBool
		}
	}
	return true // Default to enabled
}