package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/receivers/kafka"
	"github.com/dukex/operion/pkg/receivers/schedule"
	"github.com/dukex/operion/pkg/receivers/webhook"
	"github.com/dukex/operion/pkg/workflow"
)

// ReceiverManager manages receivers and processes trigger events
type ReceiverManager struct {
	id                string
	eventBus          eventbus.EventBus
	persistence       persistence.Persistence
	logger            *slog.Logger
	ctx               context.Context
	cancel            context.CancelFunc
	restartCount      int
	webhookPort       int
	
	// Receiver management
	receivers         map[string]protocol.Receiver
	receiverFactories map[string]protocol.ReceiverFactory
	receiverMutex     sync.RWMutex
	
	// Workflow processing
	workflowRepo      *workflow.Repository
	triggerMatcher    *workflow.TriggerMatcher
	
	// Configuration
	config            protocol.ReceiverConfig
}

// NewReceiverManager creates a new receiver-based dispatcher manager
func NewReceiverManager(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	webhookPort int,
) *ReceiverManager {
	rm := &ReceiverManager{
		id:                id,
		logger:           logger.With("module", "operion-receiver-manager", "manager_id", id),
		persistence:      persistence,
		eventBus:         eventBus,
		restartCount:     0,
		webhookPort:      webhookPort,
		receivers:        make(map[string]protocol.Receiver),
		receiverFactories: make(map[string]protocol.ReceiverFactory),
		workflowRepo:     workflow.NewRepository(persistence),
		triggerMatcher:   workflow.NewTriggerMatcher(logger),
	}

	// Register receiver factories
	rm.registerReceiverFactories()
	
	return rm
}

// registerReceiverFactories registers all supported receiver types
func (rm *ReceiverManager) registerReceiverFactories() {
	rm.receiverFactories["kafka"] = kafka.NewKafkaReceiverFactory()
	rm.receiverFactories["webhook"] = webhook.NewWebhookReceiverFactory(rm.webhookPort)
	rm.receiverFactories["schedule"] = schedule.NewScheduleReceiverFactory()
	
	rm.logger.Info("Registered receiver factories", "types", []string{"kafka", "webhook", "schedule"})
}

// Configure sets the receiver configuration
func (rm *ReceiverManager) Configure(config protocol.ReceiverConfig) error {
	rm.config = config
	
	if rm.config.TriggerTopic == "" {
		rm.config.TriggerTopic = "operion.trigger"
	}
	
	rm.logger.Info("Configured receiver manager", 
		"trigger_topic", rm.config.TriggerTopic,
		"sources_count", len(rm.config.Sources))
	
	return nil
}

// Start starts the receiver manager
func (rm *ReceiverManager) Start(ctx context.Context) {
	rm.ctx, rm.cancel = context.WithCancel(ctx)
	rm.logger.Info("Starting receiver manager")

	// Setup signal handling
	rm.setupSignals()
	
	// Start receivers
	if err := rm.startReceivers(); err != nil {
		rm.logger.Error("Failed to start receivers", "error", err)
		rm.restart()
		return
	}
	
	// Subscribe to trigger events
	if err := rm.subscribeTriggerEvents(); err != nil {
		rm.logger.Error("Failed to subscribe to trigger events", "error", err)
		rm.restart()
		return
	}
	
	rm.logger.Info("Receiver manager started successfully")
	
	// Wait for context cancellation
	<-rm.ctx.Done()
	rm.logger.Info("Receiver manager stopped")
}

// startReceivers starts all configured receivers
func (rm *ReceiverManager) startReceivers() error {
	rm.logger.Info("Starting receivers", "count", len(rm.config.Sources))
	
	// Group sources by receiver type
	sourcesByType := make(map[string][]protocol.SourceConfig)
	for _, source := range rm.config.Sources {
		sourcesByType[source.Type] = append(sourcesByType[source.Type], source)
	}
	
	// Start receivers for each type
	for receiverType, sources := range sourcesByType {
		if err := rm.startReceiverType(receiverType, sources); err != nil {
			return err
		}
	}
	
	return nil
}

// startReceiverType starts a receiver for a specific type
func (rm *ReceiverManager) startReceiverType(receiverType string, sources []protocol.SourceConfig) error {
	factory, exists := rm.receiverFactories[receiverType]
	if !exists {
		rm.logger.Error("Unknown receiver type", "type", receiverType)
		return nil // Skip unknown types rather than failing completely
	}
	
	// Create receiver configuration for this type
	receiverConfig := protocol.ReceiverConfig{
		Sources:      sources,
		TriggerTopic: rm.config.TriggerTopic,
		Transformers: rm.config.Transformers,
	}
	
	// Create receiver
	receiver, err := factory.Create(receiverConfig, rm.eventBus, rm.logger)
	if err != nil {
		rm.logger.Error("Failed to create receiver", "type", receiverType, "error", err)
		return err
	}
	
	// Start receiver
	if err := receiver.Start(rm.ctx); err != nil {
		rm.logger.Error("Failed to start receiver", "type", receiverType, "error", err)
		return err
	}
	
	// Store receiver
	rm.receiverMutex.Lock()
	rm.receivers[receiverType] = receiver
	rm.receiverMutex.Unlock()
	
	rm.logger.Info("Started receiver", "type", receiverType, "sources_count", len(sources))
	return nil
}

// subscribeTriggerEvents subscribes to trigger events and processes them
func (rm *ReceiverManager) subscribeTriggerEvents() error {
	rm.logger.Info("Subscribing to trigger events", "topic", rm.config.TriggerTopic)
	
	// Register handler for trigger events
	err := rm.eventBus.Handle(events.TriggerDetectedEvent, rm.handleTriggerEvent)
	if err != nil {
		return err
	}
	
	// Start event bus subscription
	return rm.eventBus.Subscribe(rm.ctx)
}

// handleTriggerEvent processes incoming trigger events
func (rm *ReceiverManager) handleTriggerEvent(ctx context.Context, event interface{}) error {
	triggerEvent, ok := event.(*events.TriggerEvent)
	if !ok {
		rm.logger.Error("Invalid trigger event type")
		return nil
	}
	
	rm.logger.Debug("Processing trigger event",
		"event_id", triggerEvent.ID,
		"trigger_type", triggerEvent.TriggerType,
		"source", triggerEvent.Source)
	
	// Get workflows that might match this trigger
	workflows, err := rm.workflowRepo.FetchByTriggerCriteria(
		triggerEvent.TriggerType,
		triggerEvent.Source,
		triggerEvent.TriggerData,
	)
	if err != nil {
		rm.logger.Error("Failed to fetch workflows for trigger", "error", err)
		return err
	}
	
	if len(workflows) == 0 {
		rm.logger.Debug("No workflows found for trigger criteria")
		return nil
	}
	
	// Match workflows against trigger event
	matches := rm.triggerMatcher.MatchWorkflows(*triggerEvent, workflows)
	if len(matches) == 0 {
		rm.logger.Debug("No workflow matches found for trigger event")
		return nil
	}
	
	rm.logger.Info("Found workflow matches for trigger event",
		"matches_count", len(matches),
		"trigger_type", triggerEvent.TriggerType,
		"source", triggerEvent.Source)
	
	// Publish WorkflowTriggered events for each match
	for _, match := range matches {
		if err := rm.publishWorkflowTriggered(ctx, *triggerEvent, match); err != nil {
			rm.logger.Error("Failed to publish workflow triggered event",
				"workflow_id", match.Workflow.ID,
				"error", err)
		}
	}
	
	return nil
}

// publishWorkflowTriggered publishes a WorkflowTriggered event
func (rm *ReceiverManager) publishWorkflowTriggered(ctx context.Context, triggerEvent events.TriggerEvent, match workflow.MatchResult) error {
	// Create WorkflowTriggered event using the original trigger data format
	workflowTriggeredEvent := events.WorkflowTriggered{
		BaseEvent:   events.NewBaseEvent(events.WorkflowTriggeredEvent, match.Workflow.ID),
		TriggerID:   match.MatchedTrigger.TriggerID,
		TriggerData: triggerEvent.OriginalData, // Use original data for backward compatibility
	}
	workflowTriggeredEvent.ID = rm.eventBus.GenerateID()
	
	// Publish to event bus
	if err := rm.eventBus.Publish(ctx, match.Workflow.ID, workflowTriggeredEvent); err != nil {
		return err
	}
	
	rm.logger.Info("Published workflow triggered event",
		"workflow_id", match.Workflow.ID,
		"workflow_name", match.Workflow.Name,
		"trigger_id", match.MatchedTrigger.TriggerID,
		"event_id", workflowTriggeredEvent.ID,
		"match_score", match.MatchScore)
	
	return nil
}

// Stop stops the receiver manager
func (rm *ReceiverManager) Stop() {
	rm.logger.Info("Stopping receiver manager")
	
	if rm.cancel != nil {
		rm.cancel()
	}
	
	// Stop all receivers
	rm.receiverMutex.Lock()
	defer rm.receiverMutex.Unlock()
	
	for receiverType, receiver := range rm.receivers {
		rm.logger.Info("Stopping receiver", "type", receiverType)
		if err := receiver.Stop(context.Background()); err != nil {
			rm.logger.Error("Error stopping receiver", "type", receiverType, "error", err)
		}
	}
	
	rm.receivers = make(map[string]protocol.Receiver)
	rm.logger.Info("All receivers stopped")
}

// restart restarts the receiver manager
func (rm *ReceiverManager) restart() {
	rm.restartCount++
	ctx := context.WithoutCancel(rm.ctx)
	rm.Stop()

	if rm.restartCount > 5 {
		rm.logger.Error("Restart limit reached, exiting...")
		os.Exit(1)
	} else {
		rm.logger.Info("Restarting receiver manager...")
		rm.Start(ctx)
	}
}

// setupSignals sets up signal handling
func (rm *ReceiverManager) setupSignals() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		rm.logger.Info("Received signal", "signal", sig)

		switch sig {
		case syscall.SIGHUP:
			rm.logger.Info("Reloading configuration...")
			rm.restart()
		case syscall.SIGINT, syscall.SIGTERM:
			rm.logger.Info("Shutting down gracefully...")
			rm.Stop()
			os.Exit(0)
		default:
			rm.logger.Warn("Unhandled signal received", "signal", sig)
		}
	}()
}