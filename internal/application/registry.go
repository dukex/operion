package application

import (
	"fmt"

	"github.com/dukex/operion/internal/domain"
)

// TriggerFactory creates a Trigger instance from a config
type TriggerFactory func(config map[string]interface{}) (domain.Trigger, error)

// ActionFactory creates an Action instance from a config
type ActionFactory func(config map[string]interface{}) (domain.Action, error)

// TriggerRegistry holds factories for creating Triggers
type TriggerRegistry struct {
    factories map[string]TriggerFactory
}

func NewTriggerRegistry() *TriggerRegistry {
    return &TriggerRegistry{factories: make(map[string]TriggerFactory)}
}

func (r *TriggerRegistry) Register(triggerType string, factory TriggerFactory) {
    r.factories[triggerType] = factory
}

func (r *TriggerRegistry) Create(triggerType string, config map[string]interface{}) (domain.Trigger, error) {
    factory, ok := r.factories[triggerType]
    if !ok {
        return nil, fmt.Errorf("trigger type '%s' not registered", triggerType)
    }
    return factory(config)
}


// ActionRegistry holds factories for creating Actions
type ActionRegistry struct {
    factories map[string]ActionFactory
}

func NewActionRegistry() *ActionRegistry {
    return &ActionRegistry{factories: make(map[string]ActionFactory)}
}

func (r *ActionRegistry) Register(actionType string, factory ActionFactory) {
    r.factories[actionType] = factory
}

func (r *ActionRegistry) Create(actionType string, config map[string]interface{}) (domain.Action, error) {
    factory, ok := r.factories[actionType]
    if !ok {
        return nil, fmt.Errorf("action type '%s' not registered", actionType)
    }
    return factory(config)
}