// Package registry provides plugin-based system for actions and triggers with dynamic loading capabilities.
package registry

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"plugin"
	"strings"

	"github.com/dukex/operion/pkg/protocol"
)

var (
	// ErrActionNotRegistered is returned when an action type is not registered.
	ErrActionNotRegistered = errors.New("action type not registered")
	// ErrTriggerNotRegistered is returned when a trigger ID is not registered.
	ErrTriggerNotRegistered = errors.New("trigger ID not registered")
	// ErrSourceProviderNotRegistered is returned when a provider ID is not registered.
	ErrSourceProviderNotRegistered = errors.New("provider ID not registered")
)

type Registry struct {
	logger                  *slog.Logger
	actionFactories         map[string]protocol.ActionFactory
	triggerFactories        map[string]protocol.TriggerFactory
	sourceProviderFactories map[string]protocol.SourceProviderFactory
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		logger:                  log,
		actionFactories:         make(map[string]protocol.ActionFactory),
		triggerFactories:        make(map[string]protocol.TriggerFactory),
		sourceProviderFactories: make(map[string]protocol.SourceProviderFactory),
	}
}

func (r *Registry) HealthCheck() (string, bool) {
	if len(r.actionFactories) == 0 && len(r.triggerFactories) == 0 && len(r.sourceProviderFactories) == 0 {
		return "No plugins loaded", false
	}

	return "Plugins loaded successfully", true
}

func (r *Registry) LoadActionPlugins(ctx context.Context, pluginsPath string) ([]protocol.ActionFactory, error) {
	return loadPlugin[protocol.ActionFactory](ctx, r.logger, pluginsPath, "Action")
}

func (r *Registry) LoadTriggerPlugins(ctx context.Context, pluginsPath string) ([]protocol.TriggerFactory, error) {
	return loadPlugin[protocol.TriggerFactory](ctx, r.logger, pluginsPath, "Trigger")
}

func (r *Registry) LoadSourceProviderPlugins(ctx context.Context, pluginsPath string) ([]protocol.SourceProviderFactory, error) {
	return loadPlugin[protocol.SourceProviderFactory](ctx, r.logger, pluginsPath, "SourceProvider")
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

// CreateAction creates a new action instance based on the provided action type and configuration.
func (r *Registry) CreateAction(
	ctx context.Context,
	actionType string,
	config map[string]any,
) (protocol.Action, error) {
	factory, ok := r.actionFactories[actionType]
	if !ok {
		return nil, fmt.Errorf("action type '%s': %w", actionType, ErrActionNotRegistered)
	}

	created, err := factory.Create(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create action '%s': %w", actionType, err)
	}

	return created, nil
}

// CreateTrigger creates a new trigger instance based on the provided trigger ID and configuration.
func (r *Registry) CreateTrigger(
	ctx context.Context,
	triggerID string,
	config map[string]any,
) (protocol.Trigger, error) {
	factory, ok := r.triggerFactories[triggerID]
	if !ok {
		return nil, fmt.Errorf("trigger ID '%s': %w", triggerID, ErrTriggerNotRegistered)
	}

	created, err := factory.Create(ctx, config, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create trigger '%s': %w", triggerID, err)
	}

	return created, nil
}

// CreateSourceProvider creates a new source provider instance based on the provided provider ID and configuration.
func (r *Registry) CreateSourceProvider(
	ctx context.Context,
	providerID string,
	config map[string]any,
) (protocol.SourceProvider, error) {
	factory, ok := r.sourceProviderFactories[providerID]
	if !ok {
		return nil, fmt.Errorf("trigger ID '%s': %w", providerID, ErrTriggerNotRegistered)
	}

	created, err := factory.Create(config, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create trigger '%s': %w", providerID, err)
	}

	return created, nil
}

// GetAvailableActions returns all available action types sorted by ID.
func (r *Registry) GetAvailableActions() []protocol.ActionFactory {
	actions := make([]protocol.ActionFactory, 0, len(r.actionFactories))
	for _, action := range r.actionFactories {
		actions = append(actions, action)
	}

	return actions
}

// GetAvailableTriggers returns all available trigger types.
func (r *Registry) GetAvailableTriggers() []protocol.TriggerFactory {
	triggers := make([]protocol.TriggerFactory, 0, len(r.triggerFactories))
	for _, trigger := range r.triggerFactories {
		triggers = append(triggers, trigger)
	}

	return triggers
}

// GetAvailableProviders returns all available source provider types.
func (r *Registry) GetAvailableSourceProviders() []protocol.SourceProviderFactory {
	sourceProviders := make([]protocol.SourceProviderFactory, 0, len(r.sourceProviderFactories))
	for _, sourceProvider := range r.sourceProviderFactories {
		sourceProviders = append(sourceProviders, sourceProvider)
	}

	return sourceProviders
}

func (r *Registry) GetSourceProviders() map[string]protocol.SourceProviderFactory {
	sourceProviders := make(map[string]protocol.SourceProviderFactory)
	for id, factory := range r.sourceProviderFactories {
		sourceProviders[id] = factory
	}
	return sourceProviders
}

func loadPlugin[T any](ctx context.Context, logger *slog.Logger, pluginsPath string, symbolName string) ([]T, error) {
	rootPath := pluginsPath + "/" + strings.ToLower(symbolName) + "s"
	root := os.DirFS(rootPath)

	pluginPathList, err := fs.Glob(root, "**/*.so")
	if err != nil {
		return nil, err
	}

	l := logger.With(slog.String("path", pluginsPath), slog.String("type", symbolName))
	l.InfoContext(ctx, "Loading plugins")

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

		l.InfoContext(ctx, "Loaded action plugin", slog.String("plugin", p))
	}

	return pluginList, nil
}
