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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
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
		logger: log.WithFields(log.Fields{
			"module":     "trigger_manager",
			"trigger_id": id,
		}),
	}
}

func (tm *TriggerManager) Start() error {
	tm.logger.Info("Starting trigger service")

	workflows, err := tm.repository.FetchAll()
	if err != nil {
		return fmt.Errorf("failed to fetch workflows: %w", err)
	}

	if len(workflows) == 0 {
		tm.logger.Info("No workflows found")
		return nil
	}

	tm.logger.Infof("Found %d workflows", len(workflows))

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start triggers for each workflow
	for _, workflow := range workflows {
		if workflow.Status != models.WorkflowStatusActive {
			tm.logger.WithField("workflow_id", workflow.ID).Info("Skipping inactive workflow")
			continue
		}

		wg.Add(1)
		go func(wf *models.Workflow) {
			defer wg.Done()
			if err := tm.startWorkflowTriggers(wf); err != nil {
				tm.logger.Errorf("Failed to start triggers for workflow %s: %v", wf.ID, err)
			}
		}(workflow)
	}

	// Handle shutdown signal
	go func() {
		<-c
		tm.logger.Info("Shutting down trigger service...")
		tm.Stop()
	}()

	wg.Wait()
	tm.logger.Info("Trigger service stopped")
	return nil
}

func (tm *TriggerManager) startWorkflowTriggers(workflow *models.Workflow) error {
	logger := tm.logger.WithFields(log.Fields{
		"workflow_id":   workflow.ID,
		"workflow_name": workflow.Name,
	})
	logger.Info("Starting triggers for workflow")

	for _, triggerItem := range workflow.Triggers {
		logger = logger.WithFields(log.Fields{
			"trigger_id":     triggerItem.ID,
			"trigger_type":   triggerItem.Type,
			"trigger_config": triggerItem.Configuration,
		})

		// Prepare trigger configuration
		config := make(map[string]interface{})
		for k, v := range triggerItem.Configuration {
			config[k] = v
		}
		config["workflow_id"] = workflow.ID
		config["trigger_id"] = triggerItem.ID
		config["id"] = triggerItem.ID

		// Create trigger instance
		trigger, err := tm.registry.CreateTrigger(triggerItem.Type, config)
		if err != nil {
			logger.Errorf("Failed to create trigger: %v", err)
			continue
		}

		// Store running trigger
		tm.triggerMutex.Lock()
		tm.runningTriggers[triggerItem.ID] = trigger
		tm.triggerMutex.Unlock()

		// Create trigger callback that publishes events
		callback := tm.createTriggerCallback(workflow.ID, triggerItem.ID, triggerItem.Type)

		// Start the trigger
		if err := trigger.Start(tm.ctx, callback); err != nil {
			logger.Errorf("Failed to start trigger: %v", err)

			tm.triggerMutex.Lock()
			delete(tm.runningTriggers, triggerItem.ID)
			tm.triggerMutex.Unlock()
			continue
		}

		logger.Info("Started trigger successfully")
	}

	// Wait for context cancellation
	<-tm.ctx.Done()
	return nil
}

func (tm *TriggerManager) createTriggerCallback(workflowID, triggerID, triggerType string) models.TriggerCallback {
	return func(ctx context.Context, data map[string]interface{}) error {
		logger := tm.logger.WithFields(log.Fields{
			"workflow_id":  workflowID,
			"trigger_id":   triggerID,
			"trigger_type": triggerType,
			"trigger_data": data,
		})

		logger.Info("Trigger fired, publishing event")

		event := events.WorkflowTriggered{
			BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, workflowID),
			TriggerID:   triggerID,
			TriggerType: triggerType,
			TriggerData: data,
		}
		event.ID = generateEventID()
		event.TriggerID = triggerID

		// Publish the event
		if err := tm.eventBus.Publish(ctx, event); err != nil {
			logger.Errorf("Failed to publish trigger event: %v", err)
			return err
		}

		logger.WithField("event_id", event.ID).Info("Successfully published trigger event")
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

func generateEventID() string {
	return fmt.Sprintf("event-%s", uuid.New().String()[:8])
}
