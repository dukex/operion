package registry

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"plugin"
	"strings"

	"github.com/dukex/operion/pkg/protocol"
)

type Registry struct {
	logger           *slog.Logger
	actionFactories  map[string]protocol.ActionFactory
	triggerFactories map[string]protocol.TriggerFactory
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		logger:           log,
		actionFactories:  make(map[string]protocol.ActionFactory),
		triggerFactories: make(map[string]protocol.TriggerFactory),
	}
}

func (r *Registry) LoadActionPlugins(pluginsPath string) ([]protocol.ActionFactory, error) {
	return loadPlugin[protocol.ActionFactory](r.logger, pluginsPath, "Action")
}

func (r *Registry) LoadTriggerPlugins(pluginsPath string) ([]protocol.TriggerFactory, error) {
	return loadPlugin[protocol.TriggerFactory](r.logger, pluginsPath, "Trigger")
}

func (r *Registry) RegisterAction(actionFactory protocol.ActionFactory) {
	r.actionFactories[actionFactory.ID()] = actionFactory
}

func (r *Registry) RegisterTrigger(triggerFactory protocol.TriggerFactory) {
	r.triggerFactories[triggerFactory.ID()] = triggerFactory
}

func (r *Registry) CreateAction(actionType string, config map[string]interface{}) (protocol.Action, error) {
	factory, ok := r.actionFactories[actionType]
	if !ok {
		return nil, fmt.Errorf("action type '%s' not registered", actionType)
	}
	return factory.Create(config)
}

func (r *Registry) CreateTrigger(triggerID string, config map[string]interface{}) (protocol.Trigger, error) {
	factory, ok := r.triggerFactories[triggerID]
	if !ok {
		return nil, fmt.Errorf("trigger ID '%s' not registered", triggerID)
	}
	return factory.Create(config, r.logger)
}

// // GetAvailableActions returns all available action types
// func (r *Registry) GetAvailableActions() []string {
// 	types := make([]string, 0, len(r.actionFactories))
// 	for actionType := range r.actionFactories {
// 		types = append(types, actionType)
// 	}
// 	return types
// }

// // GetAvailableTriggers returns all available trigger types
// func (r *Registry) GetAvailableTriggers() []string {
// 	types := make([]string, 0, len(r.triggerFactories))
// 	for triggerType := range r.triggerFactories {
// 		types = append(types, triggerType)
// 	}
// 	return types
// }

// // GetComponent retrieves component metadata by type
// func (r *Registry) GetComponent(componentType string) (*models.RegisteredComponent, bool) {
// 	component, exists := r.components[componentType]
// 	return component, exists
// }

// // GetAllComponents returns all registered components
// func (r *Registry) GetAllComponents() []*models.RegisteredComponent {
// 	components := make([]*models.RegisteredComponent, 0, len(r.components))
// 	for _, component := range r.components {
// 		components = append(components, component)
// 	}
// 	return components
// }

// // GetComponentsByType returns components filtered by type (action or trigger)
// func (r *Registry) GetComponentsByType(compType ComponentType) []*models.RegisteredComponent {
// 	var components []*models.RegisteredComponent

// 	for _, component := range r.components {
// 		switch compType {
// 		case ComponentTypeAction:
// 			if _, isAction := r.actionFactories[component.Type]; isAction {
// 				components = append(components, component)
// 			}
// 		case ComponentTypeTrigger:
// 			if _, isTrigger := r.triggerFactories[component.Type]; isTrigger {
// 				components = append(components, component)
// 			}
// 		}
// 	}

// 	return components
// }

// // IsActionRegistered checks if an action type is registered
// func (r *Registry) IsActionRegistered(actionType string) bool {
// 	_, exists := r.actionFactories[actionType]
// 	return exists
// }

// // IsTriggerRegistered checks if a trigger type is registered
// func (r *Registry) IsTriggerRegistered(triggerType string) bool {
// 	_, exists := r.triggerFactories[triggerType]
// 	return exists
// }

func loadPlugin[T interface{}](logger *slog.Logger, pluginsPath string, symbolName string) ([]T, error) {
	rootPath := pluginsPath + "/" + strings.ToLower(symbolName) + "s"
	root := os.DirFS(rootPath)
	pluginPathList, err := fs.Glob(root, "**/*.so")

	if err != nil {
		return nil, err
	}

	l := logger.With(slog.String("path", pluginsPath), slog.String("type", symbolName))
	l.Info("Loading plugins")

	pluginList := make([]T, 0, len(pluginPathList))
	for _, p := range pluginPathList {
		plg, err := plugin.Open(rootPath + "/" + p)
		if err != nil {
			panic(err)
		}

		v, err := plg.Lookup(symbolName)
		if err != nil {
			panic(err)
		}

		castV, ok := v.(T)
		if !ok {
			panic("Could not cast plugin")
		}

		pluginList = append(pluginList, castV)

		l.Info("Loaded action plugin", slog.String("plugin", p))
	}

	return pluginList, nil
}
