package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/events"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/dukex/operion/pkg/workflow"
	log "github.com/sirupsen/logrus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Worker struct {
	id               string
	repo             *workflow.Repository
	workflowExecutor *workflow.Executor
	eventBus         event_bus.EventBusI
	logger           *log.Entry
	tp               *sdktrace.TracerProvider
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
		ctx:              ctx,
		cancel:           cancel,
		logger: log.WithFields(log.Fields{
			"module":    "trigger_manager",
			"worker_id": id,
		}),
	}
}

func (w *Worker) Start() error {
	w.logger.Info("Starting worker subscriptions")

	if err := w.eventBus.Subscribe(w.ctx, string(events.WorkflowTriggeredEvent), w.handleWorkflowTriggered); err != nil {
		return err
	}

	w.logger.Info("Worker started successfully")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.logger.Info("Shutting down worker...")
	w.cancel()

	return nil
}

func (w *Worker) handleWorkflowTriggered(ctx context.Context, event interface{}) error {
	trace := w.tp.Tracer("handleWorkflowTriggered")
	ctx, span := trace.Start(ctx, "handleWorkflowTriggered")
	defer span.End()

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
	println(triggerData)

	if err := w.workflowExecutor.Execute(ctx, triggeredEvent.WorkflowID, triggerData); err != nil {
		w.logger.WithError(err).Error("Failed to execute workflow")

		// 	failedEvent := events.WorkflowFailed{
		// 		BaseEvent:   events.NewBaseEvent(events.WorkflowFailedEvent, triggeredEvent.WorkflowID),
		// 		ExecutionID: triggeredEvent.ID,
		// 		Error:       err.Error(),
		// 	}
		// 	failedEvent.WorkerID = w.id

		// 	if publishErr := w.eventBus.Publish(ctx, failedEvent); publishErr != nil {
		// 		w.logger.WithError(publishErr).Error("Failed to publish workflow failed event")
		// 	}

		return err
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

	return errors.New("Workflow execution not implemented yet")
}
