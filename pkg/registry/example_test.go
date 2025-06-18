package registry_test

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/registry"
)

// Example action implementation
type ExampleAction struct {
	message string
}

func (a *ExampleAction) GetID() string                 { return "example-1" }
func (a *ExampleAction) GetType() string               { return "example" }
func (a *ExampleAction) Validate() error               { return nil }
func (a *ExampleAction) GetConfig() map[string]interface{} { 
	return map[string]interface{}{"message": a.message} 
}

func (a *ExampleAction) Execute(ctx context.Context, ectx models.ExecutionContext) (interface{}, error) {
	return fmt.Sprintf("Hello: %s", a.message), nil
}

// Example trigger implementation  
type ExampleTrigger struct {
	interval string
}

func (t *ExampleTrigger) GetID() string                 { return "example-1" }
func (t *ExampleTrigger) GetType() string               { return "example" }
func (t *ExampleTrigger) Validate() error               { return nil }
func (t *ExampleTrigger) GetConfig() map[string]interface{} { 
	return map[string]interface{}{"interval": t.interval} 
}

func (t *ExampleTrigger) Start(ctx context.Context, callback models.TriggerCallback) error {
	// Start trigger logic here
	return nil
}

func (t *ExampleTrigger) Stop(ctx context.Context) error {
	// Stop trigger logic here
	return nil
}

// Example demonstrating the unified registry usage
func ExampleRegistry() {
	// Create a new unified registry
	reg := registry.NewRegistry()

	// Register an action with schema
	actionComponent := &models.RegisteredComponent{
		Type:        "example-action",
		Name:        "Example Action",
		Description: "An example action that greets with a message",
		Schema: &models.JSONSchema{
			Type: "object",
			Properties: map[string]*models.Property{
				"message": {
					Type:        "string",
					Description: "The message to include in greeting",
				},
			},
			Required: []string{"message"},
		},
	}

	reg.RegisterAction(actionComponent, func(config map[string]interface{}) (models.Action, error) {
		message, ok := config["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message is required")
		}
		return &ExampleAction{message: message}, nil
	})

	// Register a trigger with schema
	triggerComponent := &models.RegisteredComponent{
		Type:        "example-trigger",
		Name:        "Example Trigger", 
		Description: "An example trigger that runs on an interval",
		Schema: &models.JSONSchema{
			Type: "object",
			Properties: map[string]*models.Property{
				"interval": {
					Type:        "string",
					Description: "How often to trigger (e.g., '1m', '5s')",
				},
			},
			Required: []string{"interval"},
		},
	}

	reg.RegisterTrigger(triggerComponent, func(config map[string]interface{}) (models.Trigger, error) {
		interval, ok := config["interval"].(string)
		if !ok {
			return nil, fmt.Errorf("interval is required")
		}
		return &ExampleTrigger{interval: interval}, nil
	})

	// Create instances using the registry
	actionConfig := map[string]interface{}{"message": "World"}
	action, err := reg.CreateAction("example-action", actionConfig)
	if err != nil {
		fmt.Printf("Error creating action: %v\n", err)
		return
	}

	triggerConfig := map[string]interface{}{"interval": "1m"}
	_, err = reg.CreateTrigger("example-trigger", triggerConfig)
	if err != nil {
		fmt.Printf("Error creating trigger: %v\n", err)
		return
	}

	// Use the created instances
	ctx := context.Background()
	executionCtx := models.ExecutionContext{
		ID:          "exec-1",
		WorkflowID:  "workflow-1",
		TriggerData: map[string]interface{}{},
		Variables:   map[string]interface{}{},
		StepResults: map[string]interface{}{},
		Metadata:    map[string]interface{}{},
	}

	result, err := action.Execute(ctx, executionCtx)
	if err != nil {
		fmt.Printf("Error executing action: %v\n", err)
		return
	}

	fmt.Printf("Action result: %v\n", result)
	fmt.Printf("Available actions: %v\n", reg.GetAvailableActions())
	fmt.Printf("Available triggers: %v\n", reg.GetAvailableTriggers())

	// Output:
	// Action result: Hello: World
	// Available actions: [example-action]
	// Available triggers: [example-trigger]
}