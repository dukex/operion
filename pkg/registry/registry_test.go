package registry

import (
	"context"
	"testing"

	"github.com/dukex/operion/pkg/models"
)

// Mock action for testing
type mockAction struct {
	id     string
	config map[string]interface{}
}

func (m *mockAction) GetID() string {
	return m.id
}

func (m *mockAction) GetType() string {
	return "mock-action"
}

func (m *mockAction) Execute(ctx context.Context, ectx models.ExecutionContext) (interface{}, error) {
	return "success", nil
}

func (m *mockAction) Validate() error {
	return nil
}

func (m *mockAction) GetConfig() map[string]interface{} {
	return m.config
}

// Mock trigger for testing
type mockTrigger struct {
	id     string
	config map[string]interface{}
}

func (m *mockTrigger) GetID() string {
	return m.id
}

func (m *mockTrigger) GetType() string {
	return "mock-trigger"
}

func (m *mockTrigger) Start(ctx context.Context, callback models.TriggerCallback) error {
	return nil
}

func (m *mockTrigger) Stop(ctx context.Context) error {
	return nil
}

func (m *mockTrigger) Validate() error {
	return nil
}

func (m *mockTrigger) GetConfig() map[string]interface{} {
	return m.config
}

func TestRegistry_RegisterAndCreateAction(t *testing.T) {
	registry := NewRegistry()

	// Create a test action component
	component := &models.RegisteredComponent{
		Type:        "test-action",
		Name:        "Test Action",
		Description: "A test action for unit testing",
		Schema: &models.JSONSchema{
			Type: "object",
			Properties: map[string]*models.Property{
				"message": {
					Type:        "string",
					Description: "Test message",
				},
			},
			Required: []string{"message"},
		},
	}

	// Register the action
	registry.RegisterAction(component, func(config map[string]interface{}) (models.Action, error) {
		return &mockAction{id: "test-1", config: config}, nil
	})

	// Test creation
	config := map[string]interface{}{"message": "hello"}
	action, err := registry.CreateAction("test-action", config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockAct, ok := action.(*mockAction)
	if !ok {
		t.Fatalf("Expected mockAction, got %T", action)
	}

	if mockAct.GetConfig()["message"] != "hello" {
		t.Errorf("Expected message 'hello', got %v", mockAct.GetConfig()["message"])
	}
}

func TestRegistry_RegisterAndCreateTrigger(t *testing.T) {
	registry := NewRegistry()

	// Create a test trigger component
	component := &models.RegisteredComponent{
		Type:        "test-trigger",
		Name:        "Test Trigger",
		Description: "A test trigger for unit testing",
		Schema: &models.JSONSchema{
			Type: "object",
			Properties: map[string]*models.Property{
				"interval": {
					Type:        "string",
					Description: "Trigger interval",
				},
			},
			Required: []string{"interval"},
		},
	}

	// Register the trigger
	registry.RegisterTrigger(component, func(config map[string]interface{}) (models.Trigger, error) {
		return &mockTrigger{id: "test-1", config: config}, nil
	})

	// Test creation
	config := map[string]interface{}{"interval": "1m"}
	trigger, err := registry.CreateTrigger("test-trigger", config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockTrig, ok := trigger.(*mockTrigger)
	if !ok {
		t.Fatalf("Expected mockTrigger, got %T", trigger)
	}

	if mockTrig.GetConfig()["interval"] != "1m" {
		t.Errorf("Expected interval '1m', got %v", mockTrig.GetConfig()["interval"])
	}
}

func TestRegistry_GetComponentsByType(t *testing.T) {
	registry := NewRegistry()

	// Register an action
	actionComponent := &models.RegisteredComponent{
		Type: "test-action",
		Name: "Test Action",
	}
	registry.RegisterAction(actionComponent, func(config map[string]interface{}) (models.Action, error) {
		return &mockAction{id: "test-1"}, nil
	})

	// Register a trigger
	triggerComponent := &models.RegisteredComponent{
		Type: "test-trigger",
		Name: "Test Trigger",
	}
	registry.RegisterTrigger(triggerComponent, func(config map[string]interface{}) (models.Trigger, error) {
		return &mockTrigger{id: "test-1"}, nil
	})

	// Test filtering by action type
	actions := registry.GetComponentsByType(ComponentTypeAction)
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != "test-action" {
		t.Errorf("Expected test-action, got %s", actions[0].Type)
	}

	// Test filtering by trigger type
	triggers := registry.GetComponentsByType(ComponentTypeTrigger)
	if len(triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(triggers))
	}
	if triggers[0].Type != "test-trigger" {
		t.Errorf("Expected test-trigger, got %s", triggers[0].Type)
	}
}

func TestRegistry_GetAvailableActions(t *testing.T) {
	registry := NewRegistry()

	actionComponent := &models.RegisteredComponent{
		Type: "test-action",
		Name: "Test Action",
	}
	registry.RegisterAction(actionComponent, func(config map[string]interface{}) (models.Action, error) {
		return &mockAction{id: "test-1"}, nil
	})

	actions := registry.GetAvailableActions()
	if len(actions) != 1 {
		t.Errorf("Expected 1 action, got %d", len(actions))
	}
	if actions[0] != "test-action" {
		t.Errorf("Expected test-action, got %s", actions[0])
	}
}

func TestRegistry_GetAvailableTriggers(t *testing.T) {
	registry := NewRegistry()

	triggerComponent := &models.RegisteredComponent{
		Type: "test-trigger",
		Name: "Test Trigger",
	}
	registry.RegisterTrigger(triggerComponent, func(config map[string]interface{}) (models.Trigger, error) {
		return &mockTrigger{id: "test-1"}, nil
	})

	triggers := registry.GetAvailableTriggers()
	if len(triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(triggers))
	}
	if triggers[0] != "test-trigger" {
		t.Errorf("Expected test-trigger, got %s", triggers[0])
	}
}

func TestRegistry_ErrorHandling(t *testing.T) {
	registry := NewRegistry()

	// Test creating non-existent action
	_, err := registry.CreateAction("non-existent", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for non-existent action")
	}

	// Test creating non-existent trigger
	_, err = registry.CreateTrigger("non-existent", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for non-existent trigger")
	}
}