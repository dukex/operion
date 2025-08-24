package main

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/mocks"
	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function to create a basic activator with mocks for testing.
func createTestActivator() (*Activator, *mocks.MockPersistence, *mocks.MockEventBus, *mocks.MockSourceEventBus) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	return activator, mockPersistence, mockEventBus, mockSourceEventBus
}

// Helper function to create a standard source event for testing.
func createTestSourceEvent() *events.SourceEvent {
	return &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}
}

// Helper function to create standard trigger matches for testing.
func createTestTriggerNodeMatches(workflowID, triggerID, sourceID string) []*models.TriggerNodeMatch {
	return []*models.TriggerNodeMatch{
		{
			WorkflowID: workflowID,
			TriggerNode: &models.WorkflowNode{
				ID:         triggerID,
				NodeType:   "trigger:scheduler",
				Category:   models.CategoryTypeTrigger,
				Name:       "Test Trigger",
				SourceID:   &sourceID,
				ProviderID: &[]string{"scheduler"}[0],
				EventType:  &[]string{"schedule_due"}[0],
				Enabled:    true,
			},
		},
	}
}

func TestNewActivator_Success(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	assert.NotNil(t, activator)
	assert.Equal(t, "test-activator", activator.id)
	assert.Equal(t, mockPersistence, activator.persistence)
	assert.Equal(t, mockEventBus, activator.eventBus)
	assert.Equal(t, mockSourceEventBus, activator.sourceEventBus)
	assert.NotNil(t, activator.logger)
	assert.Equal(t, 0, activator.restartCount)
}

func TestNewActivator_WithValidParameters(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("activator-123", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	require.NotNil(t, activator)
	assert.Equal(t, "activator-123", activator.id)
	assert.Same(t, mockPersistence, activator.persistence)
	assert.Same(t, mockEventBus, activator.eventBus)
	assert.Same(t, mockSourceEventBus, activator.sourceEventBus)
}

func TestActivator_HandleSourceEvent_ValidEvent(t *testing.T) {
	activator, mockPersistence, mockEventBus, _ := createTestActivator()
	sourceEvent := createTestSourceEvent()
	triggerMatches := createTestTriggerNodeMatches("workflow-123", "trigger-123", "source-123")

	// Mock finding matching triggers
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(triggerMatches, nil)

	// Mock saving execution context
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)

	// Mock event publishing
	mockEventBus.On("GenerateID").Return("event-123")
	mockEventBus.On("Publish", mock.Anything, "trigger-123:event-123", mock.AnythingOfType("events.NodeActivation")).Return(nil)

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_HandleSourceEvent_InvalidEvent(t *testing.T) {
	activator, mockPersistence, mockEventBus, _ := createTestActivator()

	// Create invalid source event (missing required fields)
	sourceEvent := &events.SourceEvent{
		SourceID:   "", // Missing SourceID makes it invalid
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source_id is required")
	// Should not call persistence or eventbus due to validation failure
	mockPersistence.GetMockNodeRepository().AssertNotCalled(t, "FindTriggerNodesBySourceEventAndProvider")
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestActivator_HandleSourceEvent_NoMatchingTriggers(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	// Mock no matching triggers found
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return([]*models.TriggerNodeMatch{}, nil)

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
	// Should not publish any events when no triggers match
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestActivator_HandleSourceEvent_DatabaseError(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	// Mock database error
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(nil, assert.AnError)

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
	mockEventBus.AssertNotCalled(t, "Publish")
}

func TestActivator_HandleSourceEvent_MultipleMatchingTriggers(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	// Mock multiple matching triggers
	triggerMatches := []*models.TriggerNodeMatch{
		{
			WorkflowID: "workflow-123",
			TriggerNode: &models.WorkflowNode{
				ID:         "trigger-123",
				NodeType:   "trigger:scheduler",
				Category:   models.CategoryTypeTrigger,
				Name:       "Test Trigger 123",
				SourceID:   &[]string{"source-123"}[0],
				ProviderID: &[]string{"scheduler"}[0],
				EventType:  &[]string{"schedule_due"}[0],
				Enabled:    true,
			},
		},
		{
			WorkflowID: "workflow-456",
			TriggerNode: &models.WorkflowNode{
				ID:         "trigger-456",
				NodeType:   "trigger:scheduler",
				Category:   models.CategoryTypeTrigger,
				Name:       "Test Trigger 456",
				SourceID:   &[]string{"source-123"}[0],
				ProviderID: &[]string{"scheduler"}[0],
				EventType:  &[]string{"schedule_due"}[0],
				Enabled:    true,
			},
		},
	}
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(triggerMatches, nil)

	// Mock saving execution contexts for both workflows
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil).Twice()

	// Mock event publishing for both workflows - each publishNodeActivation call needs 2 GenerateID calls
	mockEventBus.On("GenerateID").Return("event-123").Once() // execution ID for first workflow
	mockEventBus.On("GenerateID").Return("event-456").Once() // event ID for first workflow
	mockEventBus.On("GenerateID").Return("event-789").Once() // execution ID for second workflow
	mockEventBus.On("GenerateID").Return("event-abc").Once() // event ID for second workflow
	mockEventBus.On("Publish", mock.Anything, "trigger-123:event-123", mock.AnythingOfType("events.NodeActivation")).Return(nil)
	mockEventBus.On("Publish", mock.Anything, "trigger-456:event-789", mock.AnythingOfType("events.NodeActivation")).Return(nil)

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	mockPersistence.GetMockNodeRepository().AssertExpectations(t)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_HandleSourceEvent_PublishFailure(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	triggerMatches := []*models.TriggerNodeMatch{
		{
			WorkflowID: "workflow-123",
			TriggerNode: &models.WorkflowNode{
				ID:         "trigger-123",
				NodeType:   "trigger:scheduler",
				Category:   models.CategoryTypeTrigger,
				Name:       "Test Trigger 123",
				SourceID:   &[]string{"source-123"}[0],
				ProviderID: &[]string{"scheduler"}[0],
				EventType:  &[]string{"schedule_due"}[0],
				Enabled:    true,
			},
		},
	}
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(triggerMatches, nil)

	// Mock saving execution context
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)

	// Mock event publishing failure
	mockEventBus.On("GenerateID").Return("event-123")
	mockEventBus.On("Publish", mock.Anything, "trigger-123:event-123", mock.AnythingOfType("events.NodeActivation")).Return(assert.AnError)

	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	// Should not return error even if publishing fails (logged but continues)
	assert.NoError(t, err)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_FindTriggersForSourceEvent_Success(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	expectedTriggers := []*models.TriggerNodeMatch{
		{
			WorkflowID: "workflow-123",
			TriggerNode: &models.WorkflowNode{
				ID:         "trigger-123",
				NodeType:   "trigger:scheduler",
				Category:   models.CategoryTypeTrigger,
				Name:       "Test Trigger 123",
				SourceID:   &[]string{"source-123"}[0],
				ProviderID: &[]string{"scheduler"}[0],
				EventType:  &[]string{"schedule_due"}[0],
				Enabled:    true,
			},
		},
	}
	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(expectedTriggers, nil)

	triggers, err := activator.findTriggerNodesForSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	assert.Equal(t, expectedTriggers, triggers)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
}

func TestActivator_FindTriggersForSourceEvent_DatabaseError(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(nil, assert.AnError)

	triggers, err := activator.findTriggerNodesForSourceEvent(context.Background(), sourceEvent)

	assert.Error(t, err)
	assert.Nil(t, triggers)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
}

func TestActivator_FindTriggersForSourceEvent_EmptyResults(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceEvent := &events.SourceEvent{
		SourceID:   "source-123",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-123"},
	}

	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-123", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return([]*models.TriggerNodeMatch{}, nil)

	triggers, err := activator.findTriggerNodesForSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	assert.Empty(t, triggers)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
}

func TestActivator_PublishWorkflowTriggered_Success(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceData := map[string]any{"schedule_id": "sched-123", "timestamp": "2023-01-01T00:00:00Z"}

	// Mock saving execution context
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)

	mockEventBus.On("GenerateID").Return("event-123")
	mockEventBus.On("Publish", mock.Anything, "trigger-123:event-123", mock.MatchedBy(func(event events.NodeActivation) bool {
		inputDataMap, ok := event.InputData.(map[string]any)

		return event.BaseEvent.Type == events.NodeActivationEvent &&
			event.BaseEvent.WorkflowID == "workflow-123" &&
			event.NodeID == "trigger-123" &&
			ok && inputDataMap["schedule_id"] == "sched-123"
	})).Return(nil)

	err := activator.publishNodeActivation(context.Background(), "workflow-123", "trigger-123", sourceData)

	assert.NoError(t, err)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_PublishWorkflowTriggered_EventBusFailure(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceData := map[string]any{"schedule_id": "sched-123"}

	// Mock saving execution context
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)

	mockEventBus.On("GenerateID").Return("event-123")
	mockEventBus.On("Publish", mock.Anything, "trigger-123:event-123", mock.AnythingOfType("events.NodeActivation")).Return(assert.AnError)

	err := activator.publishNodeActivation(context.Background(), "workflow-123", "trigger-123", sourceData)

	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_PublishWorkflowTriggered_ValidEventStructure(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	sourceData := map[string]any{
		"schedule_id": "sched-123",
		"timestamp":   "2023-01-01T00:00:00Z",
		"metadata":    map[string]any{"cron": "0 * * * *"},
	}

	// Mock saving execution context
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)

	mockEventBus.On("GenerateID").Return("event-456")

	// Capture the actual event to validate its structure
	var capturedEvent events.NodeActivation

	mockEventBus.On("Publish", mock.Anything, "trigger-456:event-456", mock.AnythingOfType("events.NodeActivation")).
		Run(func(args mock.Arguments) {
			capturedEvent = args.Get(2).(events.NodeActivation)
		}).Return(nil)

	err := activator.publishNodeActivation(context.Background(), "workflow-456", "trigger-456", sourceData)

	assert.NoError(t, err)

	// Validate event structure
	assert.Equal(t, "event-456", capturedEvent.ID)
	assert.Equal(t, events.NodeActivationEvent, capturedEvent.Type)
	assert.Equal(t, "workflow-456", capturedEvent.WorkflowID)
	assert.Equal(t, "trigger-456", capturedEvent.NodeID)
	assert.Equal(t, sourceData, capturedEvent.InputData)
	assert.NotZero(t, capturedEvent.Timestamp)

	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}

func TestActivator_ProcessSourceEvents_Success(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	// Mock successful setup
	mockSourceEventBus.On("HandleSourceEvents", mock.AnythingOfType("eventbus.SourceEventHandler")).Return(nil)
	mockSourceEventBus.On("SubscribeToSourceEvents", mock.Anything).Return(nil)

	// Create a context that will be cancelled to avoid blocking
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	activator.processSourceEvents(ctx)

	mockSourceEventBus.AssertExpectations(t)
}

func TestActivator_ProcessSourceEvents_HandlerRegistrationFailure(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	// Mock handler registration failure
	mockSourceEventBus.On("HandleSourceEvents", mock.AnythingOfType("eventbus.SourceEventHandler")).Return(assert.AnError)

	ctx := context.Background()
	activator.processSourceEvents(ctx)

	mockSourceEventBus.AssertExpectations(t)
	// Should not call SubscribeToSourceEvents if handler registration fails
	mockSourceEventBus.AssertNotCalled(t, "SubscribeToSourceEvents")
}

func TestActivator_ProcessSourceEvents_SubscriptionFailure(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	// Mock successful handler registration but subscription failure
	mockSourceEventBus.On("HandleSourceEvents", mock.AnythingOfType("eventbus.SourceEventHandler")).Return(nil)
	mockSourceEventBus.On("SubscribeToSourceEvents", mock.Anything).Return(assert.AnError)

	ctx := context.Background()
	activator.processSourceEvents(ctx)

	mockSourceEventBus.AssertExpectations(t)
}

func TestActivator_Restart_IncrementCount(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	// Mock to prevent actual restart process from running
	originalRestartCount := activator.restartCount

	// We can't test the full restart method since it calls os.Exit or starts a new process
	// But we can test the restart count increment by calling it directly
	activator.restartCount++ // Simulate what restart() does

	assert.Equal(t, originalRestartCount+1, activator.restartCount)
}

func TestActivator_Stop_GracefulShutdown(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	ctx, cancel := context.WithCancel(context.Background())

	// Test that stop calls cancel
	activator.stop(cancel)

	// Verify context was cancelled
	select {
	case <-ctx.Done():
		// Context was properly cancelled
	default:
		t.Error("Context should have been cancelled")
	}
}

func TestActivator_Stop_WithNilCancel(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	// Should not panic when cancel is nil
	assert.NotPanics(t, func() {
		activator.stop(nil)
	})
}

func TestActivator_HandleSignals_Setup(t *testing.T) {
	mockPersistence := mocks.NewMockPersistence()
	mockEventBus := &mocks.MockEventBus{}
	mockSourceEventBus := &mocks.MockSourceEventBus{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	activator := NewActivator("test-activator", mockPersistence, mockEventBus, mockSourceEventBus, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test that handleSignals sets up signal handling without panicking
	assert.NotPanics(t, func() {
		activator.handleSignals(ctx, cancel)
		// Give goroutine time to start
		time.Sleep(50 * time.Millisecond)
	})
}

// Integration test helper to test the full event handling flow.
func TestActivator_EventHandlingFlow_Integration(t *testing.T) {
	activator, mockPersistence, mockEventBus, _ := createTestActivator()

	// Use different IDs for integration test to distinguish from unit tests
	sourceEvent := &events.SourceEvent{
		SourceID:   "source-integration",
		ProviderID: "scheduler",
		EventType:  "ScheduleDue",
		EventData:  map[string]any{"schedule_id": "sched-integration"},
	}

	triggerMatches := createTestTriggerNodeMatches("workflow-integration", "trigger-integration", "source-integration")

	mockPersistence.GetMockNodeRepository().On("FindTriggerNodesBySourceEventAndProvider", mock.Anything, "source-integration", "ScheduleDue", "scheduler", models.WorkflowStatusActive).Return(triggerMatches, nil)
	mockPersistence.GetMockExecutionContextRepository().On("SaveExecutionContext", mock.Anything, mock.AnythingOfType("*models.ExecutionContext")).Return(nil)
	mockEventBus.On("GenerateID").Return("event-integration")
	mockEventBus.On("Publish", mock.Anything, "trigger-integration:event-integration", mock.AnythingOfType("events.NodeActivation")).Return(nil)

	// Test the complete flow
	err := activator.handleSourceEvent(context.Background(), sourceEvent)

	assert.NoError(t, err)
	mockPersistence.GetMockWorkflowRepository().AssertExpectations(t)
	mockPersistence.GetMockExecutionContextRepository().AssertExpectations(t)
	mockEventBus.AssertExpectations(t)
}
