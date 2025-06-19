package registry

import (
	"context"
	"fmt"

	"github.com/dukex/operion/pkg/models"
	trc "github.com/dukex/operion/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ComponentType defines the type of component (action or trigger)
type ComponentType string

const (
	ComponentTypeAction  ComponentType = "action"
	ComponentTypeTrigger ComponentType = "trigger"
)

// Factory is a generic factory function for creating components
type Factory[T any] func(config map[string]interface{}) (T, error)

// Registry provides a unified registry for actions and triggers with schema support
type Registry struct {
	actionFactories  map[string]Factory[models.Action]
	triggerFactories map[string]Factory[models.Trigger]
	components       map[string]*models.RegisteredComponent
	tracer           trace.Tracer
}

// NewRegistry creates a new unified registry
func NewRegistry() *Registry {
	return &Registry{
		actionFactories:  make(map[string]Factory[models.Action]),
		triggerFactories: make(map[string]Factory[models.Trigger]),
		components:       make(map[string]*models.RegisteredComponent),
		tracer:           trc.GetTracer("registry"),
	}
}

func (r *Registry) RegisterAction(component *models.RegisteredComponent, factory Factory[models.Action]) {
	r.actionFactories[component.Type] = factory
	r.components[component.Type] = component
}

func (r *Registry) RegisterTrigger(component *models.RegisteredComponent, factory Factory[models.Trigger]) {
	r.triggerFactories[component.Type] = factory
	r.components[component.Type] = component
}

// CreateAction creates an action instance by type  
func (r *Registry) CreateAction(actionType string, config map[string]interface{}) (models.Action, error) {
	return r.CreateActionWithContext(context.Background(), actionType, config)
}

// CreateActionWithContext creates an action instance by type with context
func (r *Registry) CreateActionWithContext(ctx context.Context, actionType string, config map[string]interface{}) (models.Action, error) {
	_, span := trc.StartSpan(ctx, r.tracer, "registry.create_action",
		attribute.String(trc.ActionTypeKey, actionType),
	)
	defer span.End()

	span.AddEvent("looking_up_action_factory")
	factory, ok := r.actionFactories[actionType]
	if !ok {
		err := fmt.Errorf("action type '%s' not registered", actionType)
		span.RecordError(err)
		span.SetStatus(codes.Error, "action type not registered")
		return nil, err
	}

	span.AddEvent("creating_action_instance")
	action, err := factory(config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create action instance")
		return nil, err
	}

	span.AddEvent("action_created")
	span.SetStatus(codes.Ok, "action created successfully")
	return action, nil
}

// CreateTrigger creates a trigger instance by type
func (r *Registry) CreateTrigger(triggerType string, config map[string]interface{}) (models.Trigger, error) {
	return r.CreateTriggerWithContext(context.Background(), triggerType, config)
}

// CreateTriggerWithContext creates a trigger instance by type with context
func (r *Registry) CreateTriggerWithContext(ctx context.Context, triggerType string, config map[string]interface{}) (models.Trigger, error) {
	_, span := trc.StartSpan(ctx, r.tracer, "registry.create_trigger",
		attribute.String(trc.TriggerTypeKey, triggerType),
	)
	defer span.End()

	span.AddEvent("looking_up_trigger_factory")
	factory, ok := r.triggerFactories[triggerType]
	if !ok {
		err := fmt.Errorf("trigger type '%s' not registered", triggerType)
		span.RecordError(err)
		span.SetStatus(codes.Error, "trigger type not registered")
		return nil, err
	}

	span.AddEvent("creating_trigger_instance")
	trigger, err := factory(config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create trigger instance")
		return nil, err
	}

	span.AddEvent("trigger_created")
	span.SetStatus(codes.Ok, "trigger created successfully")
	return trigger, nil
}

// GetAvailableActions returns all available action types
func (r *Registry) GetAvailableActions() []string {
	types := make([]string, 0, len(r.actionFactories))
	for actionType := range r.actionFactories {
		types = append(types, actionType)
	}
	return types
}

// GetAvailableTriggers returns all available trigger types
func (r *Registry) GetAvailableTriggers() []string {
	types := make([]string, 0, len(r.triggerFactories))
	for triggerType := range r.triggerFactories {
		types = append(types, triggerType)
	}
	return types
}

// GetComponent retrieves component metadata by type
func (r *Registry) GetComponent(componentType string) (*models.RegisteredComponent, bool) {
	component, exists := r.components[componentType]
	return component, exists
}

// GetAllComponents returns all registered components
func (r *Registry) GetAllComponents() []*models.RegisteredComponent {
	components := make([]*models.RegisteredComponent, 0, len(r.components))
	for _, component := range r.components {
		components = append(components, component)
	}
	return components
}

// GetComponentsByType returns components filtered by type (action or trigger)
func (r *Registry) GetComponentsByType(compType ComponentType) []*models.RegisteredComponent {
	var components []*models.RegisteredComponent

	for _, component := range r.components {
		switch compType {
		case ComponentTypeAction:
			if _, isAction := r.actionFactories[component.Type]; isAction {
				components = append(components, component)
			}
		case ComponentTypeTrigger:
			if _, isTrigger := r.triggerFactories[component.Type]; isTrigger {
				components = append(components, component)
			}
		}
	}

	return components
}

// IsActionRegistered checks if an action type is registered
func (r *Registry) IsActionRegistered(actionType string) bool {
	_, exists := r.actionFactories[actionType]
	return exists
}

// IsTriggerRegistered checks if a trigger type is registered
func (r *Registry) IsTriggerRegistered(triggerType string) bool {
	_, exists := r.triggerFactories[triggerType]
	return exists
}
