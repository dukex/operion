package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/registry"
	trc "github.com/dukex/operion/pkg/tracer"
	"github.com/dukex/operion/pkg/workflow"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type TriggerManager struct {
	id              string
	repository      *workflow.Repository
	registry        *registry.Registry
	eventBus        event_bus.EventBusI
	runningTriggers map[string]models.Trigger
	triggerMutex    sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	tp              *sdktrace.TracerProvider
	tracer          trace.Tracer
	logger          *log.Entry
}

func NewTriggerManager(
	id string,
	repository *workflow.Repository,
	registry *registry.Registry,
	eventBus event_bus.EventBusI,
) *TriggerManager {
	ctx, cancel := context.WithCancel(context.Background())

	tp, err := trc.InitTracer(ctx, "operion-trigger")

	if err != nil {
		log.Errorf("Failed to initialize tracer: %v", err)
		os.Exit(1)
	}

	return &TriggerManager{
		id:              id,
		repository:      repository,
		registry:        registry,
		eventBus:        eventBus,
		runningTriggers: make(map[string]models.Trigger),
		ctx:             ctx,
		cancel:          cancel,
		tp:              tp,
		tracer:          trc.GetTracer("trigger"),
		logger: log.WithFields(log.Fields{
			"module":     "trigger_manager",
			"trigger_id": id,
		}),
	}
}

func (tm *TriggerManager) Start() error {
	ctx, span := trc.StartSpan(tm.ctx, tm.tracer, "trigger_service.start",
		attribute.String(trc.ServiceIDKey, tm.id),
	)
	defer span.End()

	tm.logger.Info("Starting trigger service")
	span.AddEvent("trigger_service_starting")

	workflows, err := tm.repository.FetchAll()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch workflows")
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	workflowCount := len(workflows)
	span.SetAttributes(attribute.Int("workflow.count", workflowCount))

	if workflowCount == 0 {
		tm.logger.Info("No workflows found")
		span.AddEvent("no_workflows_found")
		return nil
	}

	tm.logger.Infof("Found %d workflows", workflowCount)
	span.AddEvent("workflows_loaded", trace.WithAttributes(attribute.Int("count", workflowCount)))

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start triggers for each workflow
	for _, workflow := range workflows {
		if workflow.Status != models.WorkflowStatusActive {
			tm.logger.WithField("workflow_id", workflow.ID).Info("Skipping inactive workflow")
			span.AddEvent("workflow_skipped", trace.WithAttributes(
				attribute.String(trc.WorkflowIDKey, workflow.ID),
				attribute.String("reason", "inactive"),
			))
			continue
		}

		wg.Add(1)
		go func(wf *models.Workflow) {
			defer wg.Done()
			if err := tm.startWorkflowTriggers(ctx, wf); err != nil {
				tm.logger.Errorf("Failed to start triggers for workflow %s: %v", wf.ID, err)
			}
		}(workflow)
	}

	// Handle shutdown signal
	go func() {
		<-c
		tm.logger.Info("Shutting down trigger service...")
		span.AddEvent("shutdown_signal_received")
		tm.Stop()
	}()

	wg.Wait()
	tm.logger.Info("Trigger service stopped")
	span.AddEvent("trigger_service_stopped")
	span.SetStatus(codes.Ok, "trigger service started successfully")
	return nil
}

func (tm *TriggerManager) startWorkflowTriggers(parentCtx context.Context, workflow *models.Workflow) error {
	ctx, span := trc.StartSpan(parentCtx, tm.tracer, "trigger_service.start_workflow_triggers",
		append(trc.WorkflowAttributes(workflow.ID, workflow.Name),
			attribute.Int("trigger.count", len(workflow.Triggers)))...,
	)
	defer span.End()

	logger := tm.logger.WithFields(log.Fields{
		"workflow_id":   workflow.ID,
		"workflow_name": workflow.Name,
	})
	logger.Info("Starting triggers for workflow")
	span.AddEvent("starting_workflow_triggers")

	for _, triggerItem := range workflow.Triggers {
		triggerCtx, triggerSpan := trc.StartSpan(ctx, tm.tracer, "trigger_service.start_trigger",
			append(trc.WorkflowAttributes(workflow.ID, workflow.Name),
				append(trc.TriggerAttributes(triggerItem.ID, triggerItem.Type))...)...,
		)

		logger = logger.WithFields(log.Fields{
			"trigger_id":     triggerItem.ID,
			"trigger_type":   triggerItem.Type,
			"trigger_config": triggerItem.Configuration,
		})

		triggerSpan.AddEvent("preparing_trigger_config")
		// Prepare trigger configuration
		config := make(map[string]interface{})
		for k, v := range triggerItem.Configuration {
			config[k] = v
		}
		config["workflow_id"] = workflow.ID
		config["trigger_id"] = triggerItem.ID
		config["id"] = triggerItem.ID

		triggerSpan.AddEvent("creating_trigger_instance")
		// Create trigger instance
		trigger, err := tm.registry.CreateTriggerWithContext(triggerCtx, triggerItem.Type, config)
		if err != nil {
			logger.Errorf("Failed to create trigger: %v", err)
			triggerSpan.RecordError(err)
			triggerSpan.SetStatus(codes.Error, "failed to create trigger")
			triggerSpan.End()
			continue
		}

		triggerSpan.AddEvent("storing_running_trigger")
		// Store running trigger
		tm.triggerMutex.Lock()
		tm.runningTriggers[triggerItem.ID] = trigger
		tm.triggerMutex.Unlock()

		// Create trigger callback that publishes events
		callback := tm.createTriggerCallback(workflow.ID, triggerItem.ID, triggerItem.Type)

		triggerSpan.AddEvent("starting_trigger")
		// Start the trigger
		if err := trigger.Start(tm.ctx, callback); err != nil {
			logger.Errorf("Failed to start trigger: %v", err)
			triggerSpan.RecordError(err)
			triggerSpan.SetStatus(codes.Error, "failed to start trigger")

			tm.triggerMutex.Lock()
			delete(tm.runningTriggers, triggerItem.ID)
			tm.triggerMutex.Unlock()
			triggerSpan.End()
			continue
		}

		logger.Info("Started trigger successfully")
		triggerSpan.AddEvent("trigger_started_successfully")
		triggerSpan.SetStatus(codes.Ok, "trigger started")
		triggerSpan.End()
	}

	// Wait for context cancellation
	<-tm.ctx.Done()
	return nil
}

func (tm *TriggerManager) createTriggerCallback(workflowID, triggerID, triggerType string) models.TriggerCallback {
	return func(ctx context.Context, data map[string]interface{}) error {
		// Create a new span for the trigger callback using the incoming context
		callbackCtx, span := trc.StartSpan(ctx, tm.tracer, "trigger_service.trigger_fired",
			attribute.String(trc.WorkflowIDKey, workflowID),
			attribute.String(trc.TriggerIDKey, triggerID),
			attribute.String(trc.TriggerTypeKey, triggerType),
		)
		defer span.End()

		logger := tm.logger.WithFields(log.Fields{
			"workflow_id":  workflowID,
			"trigger_id":   triggerID,
			"trigger_type": triggerType,
			"trigger_data": data,
		})

		logger.Info("Trigger fired, publishing event")
		span.AddEvent("trigger_fired", trace.WithAttributes(
			attribute.String("trigger_data", fmt.Sprintf("%+v", data)),
		))

		// Create workflow triggered event
		event := events.WorkflowTriggered{
			BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
			TriggerType: triggerType,
			TriggerData: data,
		}
		event.ID = generateEventID()
		event.TriggerID = triggerID

		span.SetAttributes(attribute.String(trc.EventIDKey, event.ID))
		span.AddEvent("event_created")

		// Publish the event
		if err := tm.eventBus.Publish(callbackCtx, event); err != nil {
			logger.Errorf("Failed to publish trigger event: %v", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to publish event")
			return err
		}

		logger.WithField("event_id", event.ID).Info("Successfully published trigger event")
		span.AddEvent("event_published")
		span.SetStatus(codes.Ok, "trigger event published successfully")
		return nil
	}
}

func (tm *TriggerManager) Stop() {
	tm.tp.Shutdown(tm.ctx)

	tm.cancel()

	tm.triggerMutex.Lock()
	defer tm.triggerMutex.Unlock()

	for triggerID, trigger := range tm.runningTriggers {
		tm.logger.Infof("Stopping trigger %s", triggerID)
		if err := trigger.Stop(context.Background()); err != nil {
			tm.logger.Errorf("Error stopping trigger %s: %v", triggerID, err)
		}
	}

	// Clear running triggers
	tm.runningTriggers = make(map[string]models.Trigger)
	tm.logger.Info("All triggers stopped")
}

func (tm *TriggerManager) GetRunningTriggers() map[string]models.Trigger {
	tm.triggerMutex.RLock()
	defer tm.triggerMutex.RUnlock()

	// Return a copy to avoid race conditions
	triggers := make(map[string]models.Trigger)
	for id, trigger := range tm.runningTriggers {
		triggers[id] = trigger
	}
	return triggers
}

func generateEventID() string {
	return fmt.Sprintf("event-%s", uuid.New().String()[:8])
}
