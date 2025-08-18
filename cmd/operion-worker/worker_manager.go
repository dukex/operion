package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/otelhelper"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/registry"
	"github.com/dukex/operion/pkg/workflow"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WorkerManager struct {
	id          string
	logger      *slog.Logger
	persistence persistence.Persistence
	registry    *registry.Registry
	eventBus    eventbus.EventBus
	tracer      trace.Tracer
}

func NewWorkerManager(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	registry *registry.Registry,
	tracer trace.Tracer,
) *WorkerManager {
	return &WorkerManager{
		id:          id,
		logger:      logger.With("module", "operion-worker", "worker_id", id),
		persistence: persistence,
		registry:    registry,
		eventBus:    eventBus,
		tracer:      tracer,
	}
}

func (w *WorkerManager) Start(ctx context.Context) error {
	w.logger.InfoContext(ctx, "Starting worker manager", "worker_id", w.id)

	err := w.eventBus.Handle(ctx, events.WorkflowTriggeredEvent, w.handleWorkflowTriggered)
	if err != nil {
		return err
	}

	err = w.eventBus.Handle(ctx, events.WorkflowStepAvailableEvent, w.handleWorkflowStepAvailable)
	if err != nil {
		return err
	}

	err = w.eventBus.Subscribe(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to subscribe to event bus", "error", err)

		return err
	}

	w.logger.InfoContext(ctx, "Worker started successfully")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.logger.InfoContext(ctx, "Shutting down worker...")

	return nil
}

func (w *WorkerManager) handleWorkflowTriggered(ctx context.Context, event any) error {
	triggeredEvent, ok := event.(*events.WorkflowTriggered)
	if !ok {
		w.logger.ErrorContext(ctx, "Invalid event type for WorkflowTriggered")

		return nil
	}

	traceCtx, span := otelhelper.StartSpan(ctx, w.tracer, "worker.workflow triggered",
		attribute.String(otelhelper.WorkflowIDKey, triggeredEvent.WorkflowID),
		attribute.String(otelhelper.TriggerIDKey, triggeredEvent.TriggerID),
	)
	defer span.End()

	logger := w.logger.With(
		"workflow_id", triggeredEvent.WorkflowID,
		"trigger_id", triggeredEvent.TriggerID,
		"event_id", triggeredEvent.ID,
	)
	logger.InfoContext(traceCtx, "Processing workflow triggered event")

	triggerData := make(map[string]any)
	if triggeredEvent.TriggerData != nil {
		triggerData = triggeredEvent.TriggerData
	}

	workflowExecutor := workflow.NewExecutor(w.persistence, w.registry)

	eventsToDispatcher, err := workflowExecutor.Start(traceCtx, logger, triggeredEvent.WorkflowID, triggerData)
	if err != nil {
		w.logger.ErrorContext(traceCtx, "Failed to execute workflow", "error", err)

		otelhelper.SetError(span, err)

		failedEvent := events.WorkflowFailed{
			BaseEvent: events.NewBaseEvent(events.WorkflowFailedEvent, triggeredEvent.WorkflowID),
			Error:     err.Error(),
		}
		failedEvent.WorkerID = triggeredEvent.WorkerID

		publishErr := w.eventBus.Publish(traceCtx, triggeredEvent.WorkflowID, failedEvent)
		if publishErr != nil {
			w.logger.ErrorContext(traceCtx, "Failed to publish workflow failed event", "error", publishErr)

			otelhelper.SetError(span, errors.New("failed to publish workflow failed event"))
		}

		return err
	}

	for _, event := range eventsToDispatcher {
		publishErr := w.eventBus.Publish(traceCtx, triggeredEvent.WorkflowID, event)

		logger.With("event_id", event.GetType()).InfoContext(traceCtx, "Successfully published trigger event")
		span.AddEvent("event_published")
		span.SetStatus(codes.Ok, "trigger event published successfully")

		if publishErr != nil {
			w.logger.ErrorContext(traceCtx, "Failed to publish workflow event", "error", publishErr, "event", event)

			otelhelper.SetError(span, errors.New("failed to publish workflow event"))

			return publishErr
		}
	}

	return nil
}

func (w *WorkerManager) handleWorkflowStepAvailable(ctx context.Context, event any) error {
	workflowExecutor := workflow.NewExecutor(w.persistence, w.registry)
	workflowStepEvent, ok := event.(*events.WorkflowStepAvailable)

	if !ok {
		w.logger.ErrorContext(ctx, "Invalid event type for WorkflowStepAvailable")

		return nil
	}

	traceCtx, span := otelhelper.StartSpan(ctx, w.tracer, "worker.workflow step available",
		attribute.String(otelhelper.WorkflowIDKey, workflowStepEvent.WorkflowID),
		attribute.String(otelhelper.ExecutionIDKey, workflowStepEvent.ExecutionID),
		attribute.String(otelhelper.StepIDKey, workflowStepEvent.StepID),
		attribute.String(otelhelper.WorkerIDKey, w.id),
		attribute.String("event_id", workflowStepEvent.ID),
	)
	defer span.End()

	logger := w.logger.With(
		"workflow_id", workflowStepEvent.WorkflowID,
		"execution_id", workflowStepEvent.ExecutionID,
		"step_id", workflowStepEvent.StepID,
	)

	logger.InfoContext(traceCtx, "Processing workflow step available event")

	workflowItem, err := workflow.NewRepository(w.persistence).FetchByID(traceCtx, workflowStepEvent.WorkflowID)
	if err != nil {
		w.logger.ErrorContext(traceCtx, "Failed to fetch workflow by ID", "error", err, "workflow_id", workflowStepEvent.WorkflowID)

		return err
	}

	eventsToDispatcher, err := workflowExecutor.ExecuteStep(traceCtx, logger, workflowItem, workflowStepEvent.ExecutionContext, workflowStepEvent.StepID)
	if err != nil {
		w.logger.ErrorContext(traceCtx, "Failed to execute workflow step", "error", err)

		// failedEvent := events.WorkflowStepFailed{
		// 	BaseEvent:   events.NewBaseEvent(events.WorkflowStepFailedEvent, workflowStepEvent.WorkflowID),
		// 	ExecutionID: workflowStepEvent.ExecutionID,
		// 	StepID:      workflowStepEvent.StepID,
		// 	Error:       err.Error(),
		// }
		// failedEvent.WorkerID = w.id
		// if publishErr := w.eventBus.Publish(ctx, workflowStepEvent.WorkflowID, failedEvent); publishErr != nil {
		// 	w.logger.ErrorContext(ctx,"Failed to publish workflow step failed event", "error", publishErr)
		// 	return publishErr
		// }
		return nil
	}

	for _, event := range eventsToDispatcher {
		publishErr := w.eventBus.Publish(ctx, workflowStepEvent.WorkflowID, event)
		if publishErr != nil {
			w.logger.ErrorContext(ctx, "Failed to publish workflow event", "error", publishErr, "event", event)

			return publishErr
		}
	}

	// finishedEvent := events.WorkflowFinished{
	// 	BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, triggeredEvent.WorkflowID),
	// 	ExecutionID: triggeredEvent.ID,
	// 	Result:      make(map[string]any),
	// }
	// finishedEvent.WorkerID = w.id

	// if err := w.eventBus.Publish(ctx, finishedEvent); err != nil {
	// 	w.logger.WithError(err).Error("Failed to publish workflow finished event")
	// }

	// w.logger.WithFields(log.Fields{
	// 	"workflow_id":  triggeredEvent.WorkflowID,
	// 	"execution_id": triggeredEvent.ID,
	// }).Info("Workflow execution completed")

	// return errors.New("Workflow execution not implemented yet")
	return nil
}
