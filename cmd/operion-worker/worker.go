package main

import (
	"context"

	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/workflow"
	log "github.com/sirupsen/logrus"
)

type Worker struct {
	id               string
	repo             *workflow.Repository
	workflowExecutor *workflow.Executor
	eventBus         event_bus.EventBusI
	logger           *log.Entry
}

func NewWorker(
	id string,
	repo *workflow.Repository,
	executor *workflow.Executor,
	eventBus event_bus.EventBusI,
	logger *log.Entry,
) *Worker {
	return &Worker{
		id:               id,
		repo:             repo,
		workflowExecutor: executor,
		eventBus:         eventBus,
		logger:           logger,
	}
}

func (w *Worker) Start(ctx context.Context) error {
	w.logger.Info("Starting worker subscriptions")

	if err := w.eventBus.Subscribe(ctx, string(events.WorkflowTriggeredEvent), w.handleWorkflowTriggered); err != nil {
		return err
	}

	w.logger.Info("Worker started successfully")
	return nil
}

func (w *Worker) handleWorkflowTriggered(ctx context.Context, event interface{}) error {
	triggeredEvent, ok := event.(*events.WorkflowTriggered)
	if !ok {
		w.logger.Error("Invalid event type for WorkflowTriggered")
		return nil
	}

	w.logger.WithFields(log.Fields{
		"workflow_id":  triggeredEvent.WorkflowID,
		"trigger_type": triggeredEvent.TriggerType,
		"event_id":     triggeredEvent.ID,
	}).Info("Processing workflow triggered event")

	_, err := w.repo.FetchByID(triggeredEvent.WorkflowID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get workflow")
		return err
	}

	triggerData := make(map[string]interface{})
	if triggeredEvent.TriggerData != nil {
		triggerData = triggeredEvent.TriggerData
	}

	startedEvent := events.WorkflowStarted{
		BaseEvent:   events.NewBaseEvent(events.WorkflowStartedEvent, triggeredEvent.WorkflowID),
		ExecutionID: triggeredEvent.ID,
	}
	startedEvent.WorkerID = w.id

	if err := w.eventBus.Publish(ctx, startedEvent); err != nil {
		w.logger.WithError(err).Error("Failed to publish workflow started event")
	}

	if err := w.workflowExecutor.Execute(ctx, triggeredEvent.WorkflowID, triggerData); err != nil {
		w.logger.WithError(err).Error("Failed to execute workflow")

		failedEvent := events.WorkflowFailed{
			BaseEvent:   events.NewBaseEvent(events.WorkflowFailedEvent, triggeredEvent.WorkflowID),
			ExecutionID: triggeredEvent.ID,
			Error:       err.Error(),
		}
		failedEvent.WorkerID = w.id

		if publishErr := w.eventBus.Publish(ctx, failedEvent); publishErr != nil {
			w.logger.WithError(publishErr).Error("Failed to publish workflow failed event")
		}

		return err
	}

	finishedEvent := events.WorkflowFinished{
		BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, triggeredEvent.WorkflowID),
		ExecutionID: triggeredEvent.ID,
		Result:      make(map[string]interface{}),
	}
	finishedEvent.WorkerID = w.id

	if err := w.eventBus.Publish(ctx, finishedEvent); err != nil {
		w.logger.WithError(err).Error("Failed to publish workflow finished event")
	}

	w.logger.WithFields(log.Fields{
		"workflow_id":  triggeredEvent.WorkflowID,
		"execution_id": triggeredEvent.ID,
	}).Info("Workflow execution completed")

	return nil
}
