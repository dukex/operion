package registry

import (
	"fmt"

	"github.com/dukex/operion/internal/domain"
)

type TriggerFactory func(config map[string]interface{}) (domain.Trigger, error)

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
