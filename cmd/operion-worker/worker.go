package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/events"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/dukex/operion/pkg/workflow"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Worker struct {
	id               string
	repo             *workflow.Repository
	workflowExecutor *workflow.Executor
	eventBus         event_bus.EventBusI
	logger           *log.Entry
	tp               *sdktrace.TracerProvider
	tracer           trace.Tracer
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewWorker(
	id string,
	repo *workflow.Repository,
	executor *workflow.Executor,
	eventBus event_bus.EventBusI,
) *Worker {
	ctx, cancel := context.WithCancel(context.Background())

	tp, err := trc.InitTracer(ctx, "operion-worker")

	if err != nil {
		log.Errorf("Failed to initialize tracer: %v", err)
		os.Exit(1)
	}
	return &Worker{
		id:               id,
		repo:             repo,
		workflowExecutor: executor,
		eventBus:         eventBus,
		tp:               tp,
		tracer:           trc.GetTracer("worker"),
		ctx:              ctx,
		cancel:           cancel,
		logger: log.WithFields(log.Fields{
			"module":    "worker",
			"worker_id": id,
		}),
	}
}

func (w *Worker) Start() error {
	ctx, span := trc.StartSpan(w.ctx, w.tracer, "worker.start",
		attribute.String(trc.WorkerIDKey, w.id),
	)
	defer span.End()

	w.logger.Info("Starting worker subscriptions")
	span.AddEvent("worker_starting")

	if err := w.eventBus.Subscribe(ctx, string(events.WorkflowTriggeredEvent), w.handleWorkflowTriggered); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to subscribe to events")
		return err
	}

	w.logger.Info("Worker started successfully")
	span.AddEvent("worker_subscribed")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.logger.Info("Shutting down worker...")
	span.AddEvent("shutdown_signal_received")
	w.cancel()

	span.SetStatus(codes.Ok, "worker started successfully")
	return nil
}

func (w *Worker) handleWorkflowTriggered(ctx context.Context, event interface{}) error {
	ctx, span := trc.StartSpan(ctx, w.tracer, "worker.handle_workflow_triggered",
		attribute.String(trc.WorkerIDKey, w.id),
	)
	defer span.End()

	span.AddEvent("event_received")

	triggeredEvent, ok := event.(*events.WorkflowTriggered)
	if !ok {
		w.logger.Error("Invalid event type for WorkflowTriggered")
		span.SetStatus(codes.Error, "invalid event type")
		return nil
	}

	// Add workflow and trigger information to span
	span.SetAttributes(
		attribute.String(trc.WorkflowIDKey, triggeredEvent.WorkflowID),
		attribute.String(trc.TriggerTypeKey, triggeredEvent.TriggerType),
		attribute.String(trc.EventIDKey, triggeredEvent.ID),
		attribute.String(trc.TriggerIDKey, triggeredEvent.TriggerID),
	)

	w.logger.WithFields(log.Fields{
		"workflow_id":  triggeredEvent.WorkflowID,
		"trigger_type": triggeredEvent.TriggerType,
		"event_id":     triggeredEvent.ID,
	}).Info("Processing workflow triggered event")

	span.AddEvent("fetching_workflow")
	workflow, err := w.repo.FetchByID(triggeredEvent.WorkflowID)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get workflow")
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch workflow")
		return err
	}

	span.SetAttributes(attribute.String(trc.WorkflowNameKey, workflow.Name))
	span.AddEvent("workflow_fetched")

	triggerData := make(map[string]interface{})
	if triggeredEvent.TriggerData != nil {
		triggerData = triggeredEvent.TriggerData
	}

	span.AddEvent("starting_workflow_execution")
	if err := w.workflowExecutor.Execute(ctx, triggeredEvent.WorkflowID, triggerData); err != nil {
		w.logger.WithError(err).Error("Failed to execute workflow")
		span.RecordError(err)
		span.SetStatus(codes.Error, "workflow execution failed")

		// Publish workflow failed event
		failedEvent := events.WorkflowFailed{
			BaseEvent:   events.NewBaseEvent(events.WorkflowFailedEvent, triggeredEvent.WorkflowID),
			ExecutionID: triggeredEvent.ID,
			Error:       err.Error(),
		}
		failedEvent.WorkerID = w.id

		if publishErr := w.eventBus.Publish(ctx, failedEvent); publishErr != nil {
			w.logger.WithError(publishErr).Error("Failed to publish workflow failed event")
			span.AddEvent("failed_to_publish_failure_event")
		} else {
			span.AddEvent("workflow_failed_event_published")
		}

		return err
	}

	span.AddEvent("workflow_execution_completed")

	// Publish workflow finished event
	finishedEvent := events.WorkflowFinished{
		BaseEvent:   events.NewBaseEvent(events.WorkflowFinishedEvent, triggeredEvent.WorkflowID),
		ExecutionID: triggeredEvent.ID,
		Result:      make(map[string]interface{}),
	}
	finishedEvent.WorkerID = w.id

	if err := w.eventBus.Publish(ctx, finishedEvent); err != nil {
		w.logger.WithError(err).Error("Failed to publish workflow finished event")
		span.AddEvent("failed_to_publish_success_event")
	} else {
		span.AddEvent("workflow_finished_event_published")
	}

	w.logger.WithFields(log.Fields{
		"workflow_id":  triggeredEvent.WorkflowID,
		"execution_id": triggeredEvent.ID,
	}).Info("Workflow execution completed")

	span.SetStatus(codes.Ok, "workflow processed successfully")
	return nil
}
