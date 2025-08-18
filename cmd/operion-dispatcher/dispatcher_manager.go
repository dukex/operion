package main

import (
	"context"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/triggers/webhook"
	"github.com/dukex/operion/pkg/workflow"
)

type DispatcherManager struct {
	id              string
	eventBus        eventbus.EventBus
	runningTriggers map[string]protocol.Trigger
	triggerMutex    sync.RWMutex
	// tp              *sdktrace.TracerProvider
	logger         *slog.Logger
	persistence    persistence.Persistence
	registry       *registry.Registry
	restartCount   int
	webhookManager *webhook.WebhookServerManager
	webhookPort    int
}

func NewDispatcherManager(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	registry *registry.Registry,
	webhookPort int,
) *DispatcherManager {
	webhookManager := webhook.GetWebhookServerManager(webhookPort, logger)

	return &DispatcherManager{
		id:              id,
		logger:          logger.With("module", "operion-dispatcher", "dispatcher_id", id),
		persistence:     persistence,
		registry:        registry,
		restartCount:    0,
		eventBus:        eventBus,
		runningTriggers: make(map[string]protocol.Trigger),
		webhookManager:  webhookManager,
		webhookPort:     webhookPort,
	}
}

func (dm *DispatcherManager) Start(ctx context.Context) {
	dmCtx, cancel := context.WithCancel(ctx)
	dm.logger.InfoContext(dmCtx, "Starting dispatcher manager")

	err := dm.webhookManager.Start(dmCtx)
	if err != nil {
		dm.logger.ErrorContext(ctx, "Failed to start webhook server manager", "error", err)
		cancel()

		return
	}

	dm.signals(dmCtx, cancel)
	dm.run(dmCtx, cancel)
	dm.logger.InfoContext(dmCtx, "Dispatcher manager stopped")
}

const restartLimit = 5

func (dm *DispatcherManager) restart(ctx context.Context, cancel context.CancelFunc) {
	dm.restartCount++
	dm.stop(ctx, cancel)

	if dm.restartCount > restartLimit {
		dm.logger.ErrorContext(ctx, "Restart limit reached, exiting...")
		os.Exit(1)
	} else {
		dm.logger.InfoContext(ctx, "Restarting dispatcher manager...")
		dm.Start(ctx)
	}
}

func (dm *DispatcherManager) signals(ctx context.Context, cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		dm.logger.InfoContext(ctx, "Received signal", "signal", sig)

		switch sig {
		case syscall.SIGHUP:
			dm.logger.InfoContext(ctx, "Reloading configuration...")
			dm.restart(ctx, cancel)
		case syscall.SIGINT, syscall.SIGTERM:
			dm.logger.InfoContext(ctx, "Shutting down gracefully...")
			os.Exit(0)
		default:
			dm.logger.WarnContext(ctx, "Unhandled signal received", "signal", sig)
		}
	}()
}

func (dm *DispatcherManager) run(ctx context.Context, cancel context.CancelFunc) {
	workflows, err := workflow.NewRepository(dm.persistence).FetchAll(ctx)
	if err != nil || len(workflows) == 0 {
		dm.logger.ErrorContext(ctx, "Failed to fetch workflows", "error", err, "workflows_count", len(workflows))
		dm.logger.InfoContext(ctx, "Retrying in 5 seconds...")

		time.Sleep(5 * time.Second)

		dm.restart(ctx, cancel)

		return
	}

	var wg sync.WaitGroup

	dm.logger.InfoContext(ctx, "Fetched workflows", "count", len(workflows))

	for _, workflow := range workflows {
		dm.logger.InfoContext(ctx, "Processing workflow", "workflow_id", workflow.ID, "workflow_name", workflow.Name)

		if workflow.Status != models.WorkflowStatusActive {
			dm.logger.With("workflow_id", workflow.ID).InfoContext(ctx, "Skipping inactive workflow")

			continue
		}

		wg.Add(1)

		go func(wf *models.Workflow) {
			defer wg.Done()

			dm.startWorkflowTriggers(ctx, wf)
		}(workflow)
	}

	wg.Wait()
}

func (dm *DispatcherManager) startWorkflowTriggers(ctx context.Context, workflow *models.Workflow) {
	logger := dm.logger.With("workflow_id", workflow.ID, "workflow_name", workflow.Name)
	logger.InfoContext(ctx, "Starting triggers for workflow")

	for _, workflowTrigger := range workflow.WorkflowTriggers {
		wtLogger := logger.With("trigger_id", workflowTrigger.ID)

		config := make(map[string]any)
		maps.Copy(config, workflowTrigger.Configuration)
		config["workflow_id"] = workflow.ID
		config["workflow_trigger_id"] = workflowTrigger.ID
		config["trigger_id"] = workflowTrigger.TriggerID
		config["id"] = workflowTrigger.ID

		trigger, err := dm.registry.CreateTrigger(ctx, workflowTrigger.TriggerID, config)
		if err != nil {
			wtLogger.ErrorContext(ctx, "Failed to create trigger", "error", err)

			continue
		}

		dm.triggerMutex.Lock()
		dm.runningTriggers[workflowTrigger.TriggerID] = trigger
		dm.triggerMutex.Unlock()

		callback := dm.createTriggerCallback(workflow.ID, workflowTrigger.TriggerID)

		if err := trigger.Start(ctx, callback); err != nil {
			wtLogger.ErrorContext(ctx, "Failed to start trigger", "error", err)

			dm.triggerMutex.Lock()
			delete(dm.runningTriggers, workflowTrigger.TriggerID)
			dm.triggerMutex.Unlock()

			continue
		}

		logger.InfoContext(ctx, "Started trigger successfully")
	}

	<-ctx.Done()
}

func (tm *DispatcherManager) createTriggerCallback(workflowID, triggerID string) protocol.TriggerCallback {
	return func(ctx context.Context, data map[string]any) error {
		logger := tm.logger.With("workflow_id", workflowID, "trigger_id", triggerID)
		logger.InfoContext(ctx, "Trigger fired, publishing event")

		event := events.WorkflowTriggered{
			BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
			TriggerID:   triggerID,
			TriggerData: data,
		}
		event.ID = tm.eventBus.GenerateID(ctx)

		err := tm.eventBus.Publish(ctx, workflowID, event)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to publish trigger event", "error", err)

			return err
		}

		logger.With("event_id", event.ID).InfoContext(ctx, "Successfully published trigger event")

		return nil
	}
}

func (dm *DispatcherManager) stop(ctx context.Context, cancel context.CancelFunc) {
	dm.logger.InfoContext(ctx, "Stopping dispatcher manager")
	cancel()

	dm.triggerMutex.Lock()
	defer dm.triggerMutex.Unlock()

	for triggerID, trigger := range dm.runningTriggers {
		dm.logger.InfoContext(ctx, "Stopping trigger", "triggerId", triggerID)

		err := trigger.Stop(ctx)
		if err != nil {
			dm.logger.ErrorContext(ctx, "Error stopping trigger %s: %v", triggerID, err)
		}
	}

	err := dm.webhookManager.Stop(ctx)
	if err != nil {
		dm.logger.ErrorContext(ctx, "Error stopping webhook server manager", "error", err)
	}

	dm.runningTriggers = make(map[string]protocol.Trigger)
	dm.logger.InfoContext(ctx, "All triggers stopped")
}
