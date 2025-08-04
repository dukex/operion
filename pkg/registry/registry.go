// Package registry provides plugin-based system for actions and triggers with dynamic loading capabilities.
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
	logger                 *slog.Logger
	actionFactories        map[string]protocol.ActionFactory
	triggerFactories       map[string]protocol.TriggerFactory
	sourceProviderFactories map[string]protocol.SourceProviderFactory
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		logger:                 log,
		actionFactories:        make(map[string]protocol.ActionFactory),
		triggerFactories:       make(map[string]protocol.TriggerFactory),
		sourceProviderFactories: make(map[string]protocol.SourceProviderFactory),
	}
}

func (r *Registry) HealthCheck() (string, bool) {
	if len(r.actionFactories) == 0 && len(r.triggerFactories) == 0 && len(r.sourceProviderFactories) == 0 {
		return "No plugins loaded", false
	}

	return "Plugins loaded successfully", true
}

func (r *Registry) LoadActionPlugins(pluginsPath string) ([]protocol.ActionFactory, error) {
	return loadPlugin[protocol.ActionFactory](r.logger, pluginsPath, "Action")
}

func (r *Registry) LoadTriggerPlugins(pluginsPath string) ([]protocol.TriggerFactory, error) {
	return loadPlugin[protocol.TriggerFactory](r.logger, pluginsPath, "Trigger")
}

func (r *Registry) LoadSourceProviderPlugins(pluginsPath string) ([]protocol.SourceProviderFactory, error) {
	return loadPlugin[protocol.SourceProviderFactory](r.logger, pluginsPath, "SourceProvider")
}

func (r *Registry) RegisterAction(actionFactory protocol.ActionFactory) {
	r.actionFactories[actionFactory.ID()] = actionFactory
}

func (r *Registry) RegisterTrigger(triggerFactory protocol.TriggerFactory) {
	r.triggerFactories[triggerFactory.ID()] = triggerFactory
}

func (r *Registry) RegisterSourceProvider(sourceProviderFactory protocol.SourceProviderFactory) {
	r.sourceProviderFactories[sourceProviderFactory.ID()] = sourceProviderFactory
}

func (r *Registry) CreateAction(actionType string, config map[string]any) (protocol.Action, error) {
	factory, ok := r.actionFactories[actionType]
	if !ok {
		return nil, fmt.Errorf("action type '%s' not registered", actionType)
	}
	return factory.Create(config)
}

func (r *Registry) CreateTrigger(triggerID string, config map[string]any) (protocol.Trigger, error) {
	factory, ok := r.triggerFactories[triggerID]
	if !ok {
		return nil, fmt.Errorf("trigger ID '%s' not registered", triggerID)
	}
	return factory.Create(config, r.logger)
}

func (r *Registry) CreateSourceProvider(providerID string, config map[string]any) (protocol.SourceProvider, error) {
	factory, ok := r.sourceProviderFactories[providerID]
	if !ok {
		return nil, fmt.Errorf("source provider ID '%s' not registered", providerID)
	}
	return factory.Create(config, r.logger)
}

// GetAvailableActions returns all available action types sorted by ID
func (r *Registry) GetAvailableActions() []protocol.ActionFactory {
	actions := make([]protocol.ActionFactory, 0, len(r.actionFactories))
	for _, action := range r.actionFactories {
		actions = append(actions, action)
	}

	return actions
}

// GetAvailableTriggers returns all available trigger types
func (r *Registry) GetAvailableTriggers() []protocol.TriggerFactory {
	triggers := make([]protocol.TriggerFactory, 0, len(r.triggerFactories))
	for _, trigger := range r.triggerFactories {
		triggers = append(triggers, trigger)
	}
	return triggers
}

// GetAvailableSourceProviders returns all available source provider types
func (r *Registry) GetAvailableSourceProviders() []protocol.SourceProviderFactory {
	sourceProviders := make([]protocol.SourceProviderFactory, 0, len(r.sourceProviderFactories))
	for _, sourceProvider := range r.sourceProviderFactories {
		sourceProviders = append(sourceProviders, sourceProvider)
	}
	return sourceProviders
}

// GetSourceProviders returns all source provider factories as a map
func (r *Registry) GetSourceProviders() map[string]protocol.SourceProviderFactory {
	sourceProviders := make(map[string]protocol.SourceProviderFactory)
	for id, factory := range r.sourceProviderFactories {
		sourceProviders[id] = factory
	}
	return sourceProviders
}

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

func loadPlugin[T any](logger *slog.Logger, pluginsPath string, symbolName string) ([]T, error) {
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
