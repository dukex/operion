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
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
)

type WorkerManager struct {
	id          string
	logger      *slog.Logger
	persistence persistence.Persistence
	registry    *registry.Registry
	eventBus    eventbus.EventBus
}

func NewWorkerManager(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	registry *registry.Registry,
) *WorkerManager {
	return &WorkerManager{
		id:          id,
		logger:      logger.With("module", "operion-worker", "worker_id", id),
		persistence: persistence,
		registry:    registry,
		eventBus:    eventBus,
	}
}

func (w *WorkerManager) Start(ctx context.Context) error {
	w.logger.Info("Starting worker manager", "worker_id", w.id)

	if err := w.eventBus.Handle(events.WorkflowTriggeredEvent, w.handleWorkflowTriggered); err != nil {
		return err
	}

	if err := w.eventBus.Handle(events.WorkflowStepAvailableEvent, w.handleWorkflowStepAvailable); err != nil {
		return err
	}

	if err := w.eventBus.Subscribe(ctx); err != nil {
		w.logger.Error("Failed to subscribe to event bus", "error", err)
		return err
	}

	w.logger.Info("Worker started successfully")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.logger.Info("Shutting down worker...")

	return nil
}

func (w *WorkerManager) handleWorkflowTriggered(ctx context.Context, event interface{}) error {
	triggeredEvent, ok := event.(*events.WorkflowTriggered)
	if !ok {
		w.logger.Error("Invalid event type for WorkflowTriggered")
		return nil
	}

	logger := w.logger.With(
		"workflow_id", triggeredEvent.WorkflowID,
		"trigger_id", triggeredEvent.TriggerID,
		"event_id", triggeredEvent.ID,
	)
	logger.Info("Processing workflow triggered event")

	triggerData := make(map[string]interface{})
	if triggeredEvent.TriggerData != nil {
		triggerData = triggeredEvent.TriggerData
	}

	workflowExecutor := workflow.NewExecutor(w.persistence, w.registry)
	eventsToDispatcher, err := workflowExecutor.Start(ctx, logger, triggeredEvent.WorkflowID, triggerData)

	if err != nil {
		w.logger.Error("Failed to execute workflow", "error", err)

		failedEvent := events.WorkflowFailed{
			BaseEvent: events.NewBaseEvent(events.WorkflowFailedEvent, triggeredEvent.WorkflowID),
			Error:     err.Error(),
		}
		failedEvent.WorkerID = triggeredEvent.WorkerID

		if publishErr := w.eventBus.Publish(ctx, triggeredEvent.WorkflowID, failedEvent); publishErr != nil {
			w.logger.Error("Failed to publish workflow failed event", "error", publishErr)
		}

		return err
	}

	for _, event := range eventsToDispatcher {
		if publishErr := w.eventBus.Publish(ctx, triggeredEvent.WorkflowID, event); publishErr != nil {
			w.logger.Error("Failed to publish workflow event", "error", publishErr, "event", event)
			return publishErr
		}
	}

	return nil
}

func (w *WorkerManager) handleWorkflowStepAvailable(ctx context.Context, event interface{}) error {
	workflowExecutor := workflow.NewExecutor(w.persistence, w.registry)
	workflowStepEvent, ok := event.(*events.WorkflowStepAvailable)

	if !ok {
		w.logger.Error("Invalid event type for WorkflowStepAvailable")
		return nil
	}

	logger := w.logger.With(
		"workflow_id", workflowStepEvent.WorkflowID,
		"execution_id", workflowStepEvent.ExecutionID,
		"step_id", workflowStepEvent.StepID,
	)

	logger.Info("Processing workflow step available event")

	workflowItem, err := workflow.NewRepository(w.persistence).FetchByID(workflowStepEvent.WorkflowID)
	if err != nil {
		w.logger.Error("Failed to fetch workflow by ID", "error", err, "workflow_id", workflowStepEvent.WorkflowID)
		return err
	}

	eventsToDispatcher, err := workflowExecutor.ExecuteStep(ctx, logger, workflowItem, workflowStepEvent.ExecutionContext, workflowStepEvent.StepID)

	if err != nil {
		w.logger.Error("Failed to execute workflow step", "error", err)

		// failedEvent := events.WorkflowStepFailed{
		// 	BaseEvent:   events.NewBaseEvent(events.WorkflowStepFailedEvent, workflowStepEvent.WorkflowID),
		// 	ExecutionID: workflowStepEvent.ExecutionID,
		// 	StepID:      workflowStepEvent.StepID,
		// 	Error:       err.Error(),
		// }
		// failedEvent.WorkerID = w.id
		// if publishErr := w.eventBus.Publish(ctx, workflowStepEvent.WorkflowID, failedEvent); publishErr != nil {
		// 	w.logger.Error("Failed to publish workflow step failed event", "error", publishErr)
		// 	return publishErr
		// }
		return nil
	}

	for _, event := range eventsToDispatcher {
		if publishErr := w.eventBus.Publish(ctx, workflowStepEvent.WorkflowID, event); publishErr != nil {
			w.logger.Error("Failed to publish workflow event", "error", publishErr, "event", event)
			return publishErr
		}
	}

	// finishedEvent := events.WorkflowFinished{
	// 	BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, triggeredEvent.WorkflowID),
	// 	ExecutionID: triggeredEvent.ID,
	// 	Result:      make(map[string]interface{}),
	// }
	// finishedEvent.WorkerID = w.id

	// if err := w.eventBus.Publish(ctx, finishedEvent); err != nil {
	// 	w.logger.WithError(err).Error("Failed to publish workflow finished event")
	// }

	// w.logger.WithFields(log.Fields{
	// 	"workflow_id":  triggeredEvent.WorkflowID,
	// 	"execution_id": triggeredEvent.ID,
	// }).Info("Workflow execution completed")

	time.Sleep(1 * time.Second) // Simulate some processing time

	// return errors.New("Workflow execution not implemented yet")
	return nil
}
