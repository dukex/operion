package registry

import (
	"fmt"

	"github.com/dukex/operion/internal/domain"
)

type ActionFactory func(config map[string]interface{}) (domain.Action, error)

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
