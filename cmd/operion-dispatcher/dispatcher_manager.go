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

	"github.com/dukex/operion/pkg/event_bus"
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
	eventBus        event_bus.EventBus
	runningTriggers map[string]protocol.Trigger
	triggerMutex    sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
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
	eventBus event_bus.EventBus,
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

func (dm *DispatcherManager) restart() {
	dm.restartCount++
	ctx := context.WithoutCancel(dm.ctx)
	dm.stop()

	if dm.restartCount > 5 {
		dm.logger.Error("Restart limit reached, exiting...")
		os.Exit(1)
	} else {
		dm.logger.Info("Restarting dispatcher manager...")
		dm.Start(ctx)
	}
}

func (dm *DispatcherManager) signals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		dm.logger.Info("Received signal", "signal", sig)

		switch sig {
		case syscall.SIGHUP:
			dm.logger.Info("Reloading configuration...")
			dm.restart()
		case syscall.SIGINT, syscall.SIGTERM:
			dm.logger.Info("Shutting down gracefully...")
			os.Exit(0)
		default:
			dm.logger.Warn("Unhandled signal received", "signal", sig)
		}
	}()
}

func (dm *DispatcherManager) run() {
	workflows, err := workflow.NewRepository(dm.persistence).FetchAll()
	if err != nil || len(workflows) == 0 {
		dm.logger.Error("Failed to fetch workflows", "error", err, "workflows_count", len(workflows))
		dm.logger.Info("Retrying in 5 seconds...")

		time.Sleep(5 * time.Second)
		dm.restart()
		return
	}

	var wg sync.WaitGroup

	dm.logger.Info("Fetched workflows", "count", len(workflows))
	for _, workflow := range workflows {
		dm.logger.Info("Processing workflow", "workflow_id", workflow.ID, "workflow_name", workflow.Name)

		if workflow.Status != models.WorkflowStatusActive {
			dm.logger.With("workflow_id", workflow.ID).Info("Skipping inactive workflow")
			continue
		}

		wg.Add(1)
		go func(wf *models.Workflow) {
			defer wg.Done()
			if err := dm.startWorkflowTriggers(wf); err != nil {
				dm.logger.Error("Failed to start triggers for workflow", "workflow_id", wf.ID, "error", err)
			}
		}(workflow)
	}

	wg.Wait()
}

func (dm *DispatcherManager) Start(ctx context.Context) {
	dm.ctx, dm.cancel = context.WithCancel(ctx)
	dm.logger.Info("Starting dispatcher manager")

	if err := dm.webhookManager.Start(dm.ctx); err != nil {
		dm.logger.Error("Failed to start webhook server manager", "error", err)
		return
	}

	dm.signals()
	dm.run()
	dm.logger.Info("Dispatcher manager stopped")
}

func (dm *DispatcherManager) startWorkflowTriggers(workflow *models.Workflow) error {
	logger := dm.logger.With("workflow_id", workflow.ID, "workflow_name", workflow.Name)
	logger.Info("Starting triggers for workflow")

	for _, workflowTrigger := range workflow.WorkflowTriggers {
		wtLogger := logger.With("trigger_id", workflowTrigger.ID)

		config := make(map[string]interface{})
		maps.Copy(config, workflowTrigger.Configuration)
		config["workflow_id"] = workflow.ID
		config["workflow_trigger_id"] = workflowTrigger.ID
		config["trigger_id"] = workflowTrigger.TriggerID
		config["id"] = workflowTrigger.ID

		trigger, err := dm.registry.CreateTrigger(workflowTrigger.TriggerID, config)
		if err != nil {
			wtLogger.Error("Failed to create trigger", "error", err)
			continue
		}

		dm.triggerMutex.Lock()
		dm.runningTriggers[workflowTrigger.TriggerID] = trigger
		dm.triggerMutex.Unlock()

		callback := dm.createTriggerCallback(workflow.ID, workflowTrigger.TriggerID)

		if err := trigger.Start(dm.ctx, callback); err != nil {
			wtLogger.Error("Failed to start trigger", "error", err)

			dm.triggerMutex.Lock()
			delete(dm.runningTriggers, workflowTrigger.TriggerID)
			dm.triggerMutex.Unlock()
			continue
		}

		logger.Info("Started trigger successfully")
	}
	<-dm.ctx.Done()
	return nil
}

func (tm *DispatcherManager) createTriggerCallback(workflowID, triggerID string) protocol.TriggerCallback {
	return func(ctx context.Context, data map[string]interface{}) error {
		logger := tm.logger.With("workflow_id", workflowID, "trigger_id", triggerID)
		logger.Info("Trigger fired, publishing event")

		event := events.WorkflowTriggered{
			BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
			TriggerID:   triggerID,
			TriggerData: data,
		}
		event.ID = tm.eventBus.GenerateID()

		if err := tm.eventBus.Publish(ctx, workflowID, event); err != nil {
			logger.Error("Failed to publish trigger event", "error", err)
			return err
		}

		logger.With("event_id", event.ID).Info("Successfully published trigger event")
		return nil
	}
}

func (dm *DispatcherManager) stop() {
	dm.logger.Info("Stopping dispatcher manager")
	dm.cancel()

	dm.triggerMutex.Lock()
	defer dm.triggerMutex.Unlock()

	for triggerID, trigger := range dm.runningTriggers {
		dm.logger.Info("Stopping trigger", "triggerId", triggerID)
		if err := trigger.Stop(context.Background()); err != nil {
			dm.logger.Error("Error stopping trigger %s: %v", triggerID, err)
		}
	}

	if err := dm.webhookManager.Stop(context.Background()); err != nil {
		dm.logger.Error("Error stopping webhook server manager", "error", err)
	}

	dm.runningTriggers = make(map[string]protocol.Trigger)
	dm.logger.Info("All triggers stopped")
}
