package workflow

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/dukex/operion/pkg/actions/http_request"
	log_action "github.com/dukex/operion/pkg/actions/log"
	"github.com/dukex/operion/pkg/actions/transform"
	"github.com/dukex/operion/pkg/channels/gochannel"
	"github.com/dukex/operion/pkg/event_bus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/dukex/operion/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutor(t *testing.T) {
	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()

	executor := NewExecutor(persistence, registry)

	assert.NotNil(t, executor)
	assert.Equal(t, persistence, executor.persistence)
	assert.Equal(t, registry, executor.registry)
}

func TestExecutor_Start_EmptyWorkflow(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with no steps
	workflow := &models.Workflow{
		ID:    "empty-workflow",
		Name:  "Empty Test Workflow",
		Steps: []*models.WorkflowStep{},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Start execution
	events, err := executor.Start(ctx, logger, "empty-workflow", map[string]interface{}{})

	assert.Error(t, err)
	assert.Nil(t, events)
	assert.Contains(t, err.Error(), "has no steps")

	// Clean up
	err = repo.Delete("empty-workflow")
	assert.NoError(t, err)
}

func TestExecutor_Start_Success(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with one step
	workflow := &models.Workflow{
		ID:   "single-step-workflow",
		Name: "Single Step Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step-1",
				Name:     "Log Step",
				ActionID: "log",
				UID:      "log_step",
				Configuration: map[string]interface{}{
					"message": "Test message",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err := repo.Create(workflow)
	require.NoError(t, err)

	// Start execution
	eventList, err := executor.Start(ctx, logger, "single-step-workflow", map[string]interface{}{
		"trigger": "test",
	})

	require.NoError(t, err)
	require.Len(t, eventList, 1)

	// Verify the returned event
	stepAvailableEvent := eventList[0].(*events.WorkflowStepAvailable)
	assert.Equal(t, events.WorkflowStepAvailableEvent, stepAvailableEvent.GetType())
	assert.Equal(t, "single-step-workflow", stepAvailableEvent.WorkflowID)
	assert.Equal(t, "step-1", stepAvailableEvent.StepID)
	assert.NotNil(t, stepAvailableEvent.ExecutionContext)
	assert.Equal(t, "test", stepAvailableEvent.ExecutionContext.TriggerData["trigger"])

	// Clean up
	err = repo.Delete("single-step-workflow")
	assert.NoError(t, err)
}

func TestExecutor_ExecuteStep_LogAction(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with log step
	workflow := &models.Workflow{
		ID:   "log-workflow",
		Name: "Log Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:            "log-step",
				Name:          "Log Action",
				ActionID:      "log",
				UID:           "log_action",
				Configuration: map[string]interface{}{},
				Enabled:       true,
			},
		},
	}

	// Create execution context
	execCtx := &models.ExecutionContext{
		ID:          "exec-123",
		WorkflowID:  "log-workflow",
		TriggerData: map[string]interface{}{"test": "data"},
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Execute step
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, execCtx, "log-step")

	require.NoError(t, err)
	require.Len(t, eventList, 2) // StepFinished + WorkflowFinished (no next step)

	// Events can be in different order, so check by type
	var stepFinishedEvent *events.WorkflowStepFinished
	var workflowFinishedEvent *events.WorkflowFinished

	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFinished:
			stepFinishedEvent = e
		case *events.WorkflowFinished:
			workflowFinishedEvent = e
		}
	}

	// Verify step finished event
	require.NotNil(t, stepFinishedEvent)
	assert.Equal(t, events.WorkflowStepFinishedEvent, stepFinishedEvent.GetType())
	assert.Equal(t, "log-workflow", stepFinishedEvent.WorkflowID)
	assert.Equal(t, "log-step", stepFinishedEvent.StepID)
	assert.Equal(t, "log", stepFinishedEvent.ActionID)
	assert.NotNil(t, stepFinishedEvent.Result)

	// Verify workflow finished event
	require.NotNil(t, workflowFinishedEvent)
	assert.Equal(t, events.WorkflowFinishedEvent, workflowFinishedEvent.GetType())
	assert.Equal(t, "log-workflow", workflowFinishedEvent.WorkflowID)

	// Verify execution context was updated
	assert.Contains(t, execCtx.StepResults, "log_action")
}

func TestExecutor_ExecuteStep_WithNextStep(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create workflow with two steps
	nextStepID := "step-2"
	workflow := &models.Workflow{
		ID:   "two-step-workflow",
		Name: "Two Step Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:            "step-1",
				Name:          "First Step",
				ActionID:      "log",
				UID:           "first_step",
				OnSuccess:     &nextStepID,
				Configuration: map[string]interface{}{},
				Enabled:       true,
			},
			{
				ID:            "step-2",
				Name:          "Second Step",
				ActionID:      "log",
				UID:           "second_step",
				Configuration: map[string]interface{}{},
				Enabled:       true,
			},
		},
	}

	// Create execution context
	execCtx := &models.ExecutionContext{
		ID:          "exec-456",
		WorkflowID:  "two-step-workflow",
		TriggerData: map[string]interface{}{},
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Execute first step
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, execCtx, "step-1")

	require.NoError(t, err)
	require.Len(t, eventList, 2) // StepFinished + StepAvailable

	// Events can be in different order, so check by type
	var stepFinishedEvent *events.WorkflowStepFinished
	var stepAvailableEvent *events.WorkflowStepAvailable

	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFinished:
			stepFinishedEvent = e
		case *events.WorkflowStepAvailable:
			stepAvailableEvent = e
		}
	}

	// Verify step finished event
	require.NotNil(t, stepFinishedEvent)
	assert.Equal(t, "step-1", stepFinishedEvent.StepID)

	// Verify next step available event
	require.NotNil(t, stepAvailableEvent)
	assert.Equal(t, events.WorkflowStepAvailableEvent, stepAvailableEvent.GetType())
	assert.Equal(t, "step-2", stepAvailableEvent.StepID)
	assert.Equal(t, execCtx, stepAvailableEvent.ExecutionContext)
}

func TestExecutor_ExecuteStep_DisabledStep(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	workflow := &models.Workflow{
		ID:   "disabled-step-workflow",
		Name: "Disabled Step Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:            "disabled-step",
				Name:          "Disabled Step",
				ActionID:      "log",
				UID:           "disabled_step",
				Configuration: map[string]interface{}{},
				Enabled:       false, // Disabled
			},
		},
	}

	execCtx := &models.ExecutionContext{
		ID:          "exec-disabled",
		WorkflowID:  "disabled-step-workflow",
		TriggerData: map[string]interface{}{},
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Execute disabled step
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, execCtx, "disabled-step")

	require.NoError(t, err)
	require.Len(t, eventList, 1) // Only WorkflowFinished (step skipped)

	// Verify workflow finished event (disabled step treated as success)
	workflowFinishedEvent := eventList[0].(*events.WorkflowFinished)
	assert.Equal(t, events.WorkflowFinishedEvent, workflowFinishedEvent.GetType())
}

func TestExecutor_ExecuteStep_StepNotFound(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	workflow := &models.Workflow{
		ID:    "test-workflow",
		Name:  "Test Workflow",
		Steps: []*models.WorkflowStep{},
	}

	execCtx := &models.ExecutionContext{
		ID:          "exec-not-found",
		WorkflowID:  "test-workflow",
		TriggerData: map[string]interface{}{},
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Execute non-existent step
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, execCtx, "non-existent-step")

	assert.Error(t, err)
	assert.Nil(t, eventList)
	assert.Contains(t, err.Error(), "step non-existent-step not found")
}

func TestExecutor_ExecuteStep_ActionFailure(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	workflow := &models.Workflow{
		ID:   "failing-workflow",
		Name: "Failing Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:            "failing-step",
				Name:          "Failing Step",
				ActionID:      "non-existent-action", // This will fail
				UID:           "failing_step",
				Configuration: map[string]interface{}{},
				Enabled:       true,
			},
		},
	}

	execCtx := &models.ExecutionContext{
		ID:          "exec-fail",
		WorkflowID:  "failing-workflow",
		TriggerData: map[string]interface{}{},
		StepResults: make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
	}

	// Execute failing step
	eventList, err := executor.ExecuteStep(ctx, logger, workflow, execCtx, "failing-step")

	assert.Error(t, err)
	require.Len(t, eventList, 2) // StepFailed + WorkflowFinished

	// Events can be in different order, so check by type
	var stepFailedEvent *events.WorkflowStepFailed

	for _, event := range eventList {
		switch e := event.(type) {
		case *events.WorkflowStepFailed:
			stepFailedEvent = e
		}
	}

	// Verify step failed event
	require.NotNil(t, stepFailedEvent)
	assert.Equal(t, events.WorkflowStepFailedEvent, stepFailedEvent.GetType())
	assert.Equal(t, "failing-step", stepFailedEvent.StepID)
	assert.Equal(t, "non-existent-action", stepFailedEvent.ActionID)
	assert.Contains(t, stepFailedEvent.Error, "not registered")
}

func TestExecutor_IntegrationWithEventBus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create gochannel event bus for testing
	watermillLogger := watermill.NewSlogLogger(logger)
	pub, sub, err := gochannel.CreateTestChannel(watermillLogger)
	require.NoError(t, err)

	eventBus := event_bus.NewWatermillEventBus(pub, sub)

	// Create test components
	persistence := file.NewFilePersistence("./test-data")
	registry := createTestRegistry()
	executor := NewExecutor(persistence, registry)

	// Create test workflow
	workflow := &models.Workflow{
		ID:   "integration-workflow",
		Name: "Integration Test Workflow",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step-1",
				Name:     "First Step",
				ActionID: "log",
				UID:      "first_step",
				Configuration: map[string]interface{}{
					"message": "Integration test",
				},
				Enabled: true,
			},
		},
	}

	// Save workflow
	repo := NewRepository(persistence)
	_, err = repo.Create(workflow)
	require.NoError(t, err)

	// Set up event handling
	var receivedEvents []event_bus.Event
	var eventsMutex sync.Mutex

	err = eventBus.Handle(events.WorkflowStepAvailableEvent, func(ctx context.Context, event interface{}) error {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		receivedEvents = append(receivedEvents, event.(event_bus.Event))
		return nil
	})
	require.NoError(t, err)

	err = eventBus.Handle(events.WorkflowStepFinishedEvent, func(ctx context.Context, event interface{}) error {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		receivedEvents = append(receivedEvents, event.(event_bus.Event))
		return nil
	})
	require.NoError(t, err)

	err = eventBus.Handle(events.WorkflowFinishedEvent, func(ctx context.Context, event interface{}) error {
		eventsMutex.Lock()
		defer eventsMutex.Unlock()
		receivedEvents = append(receivedEvents, event.(event_bus.Event))
		return nil
	})
	require.NoError(t, err)

	// Start event bus
	err = eventBus.Subscribe(ctx)
	require.NoError(t, err)

	// Start workflow execution
	eventList, err := executor.Start(ctx, logger, "integration-workflow", map[string]interface{}{
		"integration": "test",
	})
	require.NoError(t, err)
	require.Len(t, eventList, 1)

	// Publish the initial event
	for _, event := range eventList {
		err = eventBus.Publish(ctx, "integration-workflow", event)
		require.NoError(t, err)
	}

	// Wait for event processing
	time.Sleep(100 * time.Millisecond)

	// Verify events were received
	eventsMutex.Lock()
	defer eventsMutex.Unlock()

	assert.GreaterOrEqual(t, len(receivedEvents), 1)

	// Clean up
	err = eventBus.Close()
	assert.NoError(t, err)

	err = repo.Delete("integration-workflow")
	assert.NoError(t, err)
}

// Helper function to create a test registry with native actions
func createTestRegistry() *registry.Registry {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	reg := registry.NewRegistry(logger)

	// Register native actions
	reg.RegisterAction(log_action.NewLogActionFactory())
	reg.RegisterAction(transform.NewTransformActionFactory())
	reg.RegisterAction(&HTTPRequestActionFactory{})

	return reg
}

// Simple HTTP request action factory for testing
type HTTPRequestActionFactory struct{}

func (f *HTTPRequestActionFactory) ID() string {
	return "http_request"
}

func (f *HTTPRequestActionFactory) Create(config map[string]interface{}) (protocol.Action, error) {
	return http_request.NewHTTPRequestAction(config)
}
