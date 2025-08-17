package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
)

// Activator consumes source events and triggers workflows based on registered triggers.
type Activator struct {
	id             string
	eventBus       eventbus.EventBus
	sourceEventBus eventbus.SourceEventBus
	persistence    persistence.Persistence
	logger         *slog.Logger
	restartCount   int
}

// NewActivator creates a new Activator instance.
func NewActivator(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	sourceEventBus eventbus.SourceEventBus,
	logger *slog.Logger,
) *Activator {
	return &Activator{
		id:             id,
		eventBus:       eventBus,
		sourceEventBus: sourceEventBus,
		persistence:    persistence,
		logger:         logger.With("module", "activator"),
	}
}

// Start begins the activator service.
func (a *Activator) Start(ctx context.Context) {
	aCtx, cancel := context.WithCancel(ctx)

	a.logger.Info("Starting activator")

	a.handleSignals(aCtx, cancel)
	a.run(aCtx)
}

// handleSignals sets up signal handling for graceful shutdown and restart.
func (a *Activator) handleSignals(ctx context.Context, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		a.logger.Info("Received signal", "signal", sig)

		switch sig {
		case syscall.SIGHUP:
			a.logger.Info("Reloading configuration...")
			a.restart(ctx, cancel)
		case syscall.SIGINT, syscall.SIGTERM:
			a.logger.Info("Shutting down gracefully...")
			a.stop(cancel)
			os.Exit(0)
		default:
			a.logger.Warn("Unhandled signal received", "signal", sig)
		}
	}()
}

// restart handles service restart with exponential backoff.
func (a *Activator) restart(ctx context.Context, cancel context.CancelFunc) {
	a.restartCount++
	newCtx := context.WithoutCancel(ctx)

	a.stop(cancel)

	if a.restartCount > 5 {
		a.logger.Error("Restart limit reached, exiting...")
		os.Exit(1)
	}

	backoff := time.Duration(a.restartCount) * time.Second
	a.logger.Info("Restarting activator...", "backoff", backoff)
	time.Sleep(backoff)

	a.Start(newCtx)
}

// run is the main loop that consumes source events.
func (a *Activator) run(ctx context.Context) {
	a.logger.Info("Starting source event consumption")

	// Set up source event subscription
	a.processSourceEvents(ctx)

	// Wait for context cancellation - the subscription runs in background goroutines
	<-ctx.Done()
	a.logger.Info("Activator context cancelled, stopping...")
}

// processSourceEvents handles incoming source events and triggers workflows.
func (a *Activator) processSourceEvents(ctx context.Context) {
	a.logger.Info("Setting up source event subscription")

	// Register handler for source events
	err := a.sourceEventBus.HandleSourceEvents(func(ctx context.Context, sourceEvent *events.SourceEvent) error {
		a.logger.Info("Received source event",
			"source_id", sourceEvent.SourceID,
			"provider_id", sourceEvent.ProviderID,
			"event_type", sourceEvent.EventType)

		return a.handleSourceEvent(ctx, sourceEvent)
	})
	if err != nil {
		a.logger.Error("Failed to register source event handler", "error", err)

		return
	}

	// Start subscribing to source events
	err = a.sourceEventBus.SubscribeToSourceEvents(ctx)
	if err != nil {
		a.logger.Error("Failed to start source event subscription", "error", err)

		return
	}

	a.logger.Info("Successfully subscribed to source events - waiting for events...")
}

// handleSourceEvent processes a single source event and triggers matching workflows.
func (a *Activator) handleSourceEvent(ctx context.Context, sourceEvent *events.SourceEvent) error {
	logger := a.logger.With(
		"source_id", sourceEvent.SourceID,
		"provider_id", sourceEvent.ProviderID,
		"event_type", sourceEvent.EventType,
	)

	logger.Info("Processing source event")

	// Validate the source event
	if err := sourceEvent.Validate(); err != nil {
		logger.Error("Invalid source event", "error", err)

		return err
	}

	// Find triggers that match this source event
	matchingTriggers, err := a.findTriggersForSourceEvent(ctx, sourceEvent)
	if err != nil {
		logger.Error("Failed to find matching triggers", "error", err)

		return err
	}

	logger.Info("Found matching triggers", "count", len(matchingTriggers))

	// Publish WorkflowTriggered event for each matching trigger
	for _, matchInfo := range matchingTriggers {
		if err := a.publishWorkflowTriggered(ctx, matchInfo.WorkflowID, matchInfo.Trigger.ID, sourceEvent.EventData); err != nil {
			logger.Error("Failed to publish WorkflowTriggered event",
				"workflow_id", matchInfo.WorkflowID,
				"trigger_id", matchInfo.Trigger.ID,
				"error", err)
		}
	}

	return nil
}

// findTriggersForSourceEvent queries the database for triggers that match a source event.
func (a *Activator) findTriggersForSourceEvent(ctx context.Context, sourceEvent *events.SourceEvent) ([]*models.TriggerMatch, error) {
	// Use the comprehensive persistence method to find triggers by source ID, event type, and provider ID
	// This enables database implementations to use proper queries and indexes for exact matching
	matchingTriggers, err := a.persistence.WorkflowTriggersBySourceEventAndProvider(
		ctx,
		sourceEvent.SourceID,
		sourceEvent.EventType,
		sourceEvent.ProviderID,
		models.WorkflowStatusActive,
	)
	if err != nil {
		a.logger.Error("Failed to fetch matching triggers", "error", err)

		return nil, err
	}

	// No additional filtering needed - the persistence layer already filtered by all criteria
	return matchingTriggers, nil
}

// publishWorkflowTriggered publishes a WorkflowTriggered event for a specific trigger.
func (a *Activator) publishWorkflowTriggered(ctx context.Context, workflowID, triggerID string, sourceData map[string]any) error {
	logger := a.logger.With("workflow_id", workflowID, "trigger_id", triggerID)
	logger.Info("Publishing WorkflowTriggered event")

	event := events.WorkflowTriggered{
		BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
		TriggerID:   triggerID,
		TriggerData: sourceData,
	}
	event.ID = a.eventBus.GenerateID()

	if err := a.eventBus.Publish(ctx, workflowID, event); err != nil {
		logger.Error("Failed to publish WorkflowTriggered event", "error", err)

		return err
	}

	logger.With("event_id", event.ID).Info("Successfully published WorkflowTriggered event")

	return nil
}

// stop gracefully shuts down the activator.
func (a *Activator) stop(cancel context.CancelFunc) {
	a.logger.Info("Stopping activator")

	if cancel != nil {
		cancel()
	}
}
