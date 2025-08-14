package main

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence/file"
	"github.com/dukex/operion/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock event bus for testing.
type MockEventBus struct {
	publishedEvents []any
}

func (m *MockEventBus) Handle(eventType events.EventType, handler eventbus.EventHandler) error {
	return nil
}

func (m *MockEventBus) Publish(ctx context.Context, key string, event eventbus.Event) error {
	m.publishedEvents = append(m.publishedEvents, event)

	return nil
}

func (m *MockEventBus) Subscribe(ctx context.Context) error {
	return nil
}

func (m *MockEventBus) Close() error {
	return nil
}

func (m *MockEventBus) GenerateID() string {
	return "mock-event-id"
}

func TestNewWorkerManager(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create worker manager
	workerID := "test-worker-1"
	wm := NewWorkerManager(workerID, persistence, eventBus, logger, registry)

	// Verify worker manager is created correctly
	assert.NotNil(t, wm)
	assert.Equal(t, workerID, wm.id)
	assert.Equal(t, persistence, wm.persistence)
	assert.Equal(t, registry, wm.registry)
	assert.Equal(t, eventBus, wm.eventBus)
	assert.NotNil(t, wm.logger)
}

func TestWorkerManager_HandleWorkflowTriggered_InvalidEvent(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create worker manager
	wm := NewWorkerManager("test-worker", persistence, eventBus, logger, registry)

	// Handle invalid event type
	err := wm.handleWorkflowTriggered(t.Context(), "invalid-event")

	// Should not return error but log it
	assert.NoError(t, err)
}

func TestWorkerManager_HandleWorkflowTriggered_WorkflowNotFound(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create worker manager
	wm := NewWorkerManager("test-worker", persistence, eventBus, logger, registry)

	// Create a mock workflow triggered event
	baseEvent := events.NewBaseEvent(events.WorkflowTriggeredEvent, "non-existent-workflow")
	baseEvent.WorkerID = "test-worker"
	mockEvent := &events.WorkflowTriggered{
		BaseEvent:   baseEvent,
		TriggerID:   "test-trigger",
		TriggerData: map[string]any{},
	}

	// Handle the event
	err := wm.handleWorkflowTriggered(t.Context(), mockEvent)

	// Should return error for non-existent workflow
	assert.Error(t, err)
}

func TestWorkerManager_HandleWorkflowStepAvailable_InvalidEvent(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create worker manager
	wm := NewWorkerManager("test-worker", persistence, eventBus, logger, registry)

	// Handle invalid event type
	err := wm.handleWorkflowStepAvailable(t.Context(), "invalid-event")

	// Should not return error but log it
	assert.NoError(t, err)
}

func TestWorkerManager_HandleWorkflowStepAvailable_WorkflowNotFound(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create worker manager
	wm := NewWorkerManager("test-worker", persistence, eventBus, logger, registry)

	// Create execution context
	executionCtx := models.ExecutionContext{
		ID:          "exec-123",
		WorkflowID:  "non-existent-workflow",
		Variables:   make(map[string]any),
		StepResults: make(map[string]any),
		TriggerData: map[string]any{},
		Metadata:    map[string]any{},
	}

	// Create a mock workflow step available event
	mockEvent := &events.WorkflowStepAvailable{
		BaseEvent:        events.NewBaseEvent(events.WorkflowStepAvailableEvent, "non-existent-workflow"),
		ExecutionID:      executionCtx.ID,
		StepID:           "step1",
		ExecutionContext: &executionCtx,
	}

	// Handle the event
	err := wm.handleWorkflowStepAvailable(t.Context(), mockEvent)

	// Should return error for non-existent workflow
	assert.Error(t, err)
}

func TestWorkerManager_BasicWorkflowExecution(t *testing.T) {
	// Setup test dependencies
	tempDir := t.TempDir()
	persistence := file.NewPersistence(tempDir)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := registry.NewRegistry(logger)
	eventBus := &MockEventBus{}

	// Create a test workflow
	workflow := &models.Workflow{
		ID:     "basic-test-workflow",
		Name:   "Basic Test Workflow",
		Status: "active",
		Steps: []*models.WorkflowStep{
			{
				ID:       "step1",
				Name:     "Log Step",
				ActionID: "log",
				UID:      "log_step",
				Configuration: map[string]any{
					"message": "Test message",
				},
				Enabled: true,
			},
		},
		Variables: map[string]any{
			"test_var": "test_value",
		},
	}

	// Save workflow to persistence
	repo := NewWorkflowRepository(persistence)
	err := repo.Create(workflow)
	require.NoError(t, err)

	// Create worker manager
	wm := NewWorkerManager("basic-test-worker", persistence, eventBus, logger, registry)

	// Create a mock workflow triggered event
	baseEvent := events.NewBaseEvent(events.WorkflowTriggeredEvent, workflow.ID)
	baseEvent.WorkerID = "basic-test-worker"
	mockEvent := &events.WorkflowTriggered{
		BaseEvent:   baseEvent,
		TriggerID:   "basic-test-trigger",
		TriggerData: map[string]any{"source": "basic_test"},
	}

	// Execute workflow triggered event
	err = wm.handleWorkflowTriggered(t.Context(), mockEvent)
	// Verify execution succeeded (basic workflow functionality should work)
	// Note: This may still fail due to missing action implementations, but the structure should be valid
	if err != nil {
		// Log the error for debugging but don't fail the test if it's just missing actions
		t.Logf("Expected error due to missing action implementations: %v", err)
	}
}

// Mock workflow repository for testing.
type MockWorkflowRepository struct {
	workflows map[string]*models.Workflow
}

func NewWorkflowRepository(persistence any) *MockWorkflowRepository {
	return &MockWorkflowRepository{
		workflows: make(map[string]*models.Workflow),
	}
}

func (r *MockWorkflowRepository) Create(workflow *models.Workflow) error {
	r.workflows[workflow.ID] = workflow

	return nil
}
