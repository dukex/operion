// Package registry provides plugin-based system for node factories and providers with dynamic loading capabilities.
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

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

var (
	// ErrProviderNotRegistered is returned when a provider ID is not registered.
	ErrProviderNotRegistered = errors.New("provider ID not registered")
	// ErrNodeNotRegistered is returned when a node type is not registered.
	ErrNodeNotRegistered = errors.New("node type not registered")
)

type Registry struct {
	logger                  *slog.Logger
	sourceProviderFactories map[string]protocol.ProviderFactory
	nodeFactories           map[string]protocol.NodeFactory
}

func NewRegistry(log *slog.Logger) *Registry {
	return &Registry{
		logger:                  log,
		sourceProviderFactories: make(map[string]protocol.ProviderFactory),
		nodeFactories:           make(map[string]protocol.NodeFactory),
	}
}

func (r *Registry) HealthCheck() (string, bool) {
	if len(r.sourceProviderFactories) == 0 && len(r.nodeFactories) == 0 {
		return "No plugins loaded", false
	}

	return "Plugins loaded successfully", true
}

func (r *Registry) LoadProviderPlugins(ctx context.Context, pluginsPath string) ([]protocol.ProviderFactory, error) {
	return loadPlugin[protocol.ProviderFactory](ctx, r.logger, pluginsPath, "Provider")
}

func (r *Registry) RegisterProvider(sourceProviderFactory protocol.ProviderFactory) {
	r.sourceProviderFactories[sourceProviderFactory.ID()] = sourceProviderFactory
}

func (r *Registry) RegisterNode(nodeFactory protocol.NodeFactory) {
	r.nodeFactories[nodeFactory.ID()] = nodeFactory
}

// CreateProvider creates a new source provider instance based on the provided provider ID and configuration.
func (r *Registry) CreateProvider(
	ctx context.Context,
	providerID string,
	config map[string]any,
) (protocol.Provider, error) {
	factory, ok := r.sourceProviderFactories[providerID]
	if !ok {
		return nil, fmt.Errorf("provider ID '%s': %w", providerID, ErrProviderNotRegistered)
	}

	created, err := factory.Create(config, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider '%s': %w", providerID, err)
	}

	return created, nil
}

// CreateNode creates a new node instance based on the provided node type and configuration.
func (r *Registry) CreateNode(
	ctx context.Context,
	nodeType string,
	nodeID string,
	config map[string]any,
) (models.Node, error) {
	factory, ok := r.nodeFactories[nodeType]
	if !ok {
		return nil, fmt.Errorf("node type '%s': %w", nodeType, ErrNodeNotRegistered)
	}

	created, err := factory.Create(ctx, nodeID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create node '%s': %w", nodeType, err)
	}

	return created, nil
}

// GetAvailableProviders returns all available source provider types.
func (r *Registry) GetAvailableProviders() []protocol.ProviderFactory {
	sourceProviders := make([]protocol.ProviderFactory, 0, len(r.sourceProviderFactories))
	for _, sourceProvider := range r.sourceProviderFactories {
		sourceProviders = append(sourceProviders, sourceProvider)
	}

	return sourceProviders
}

func (r *Registry) GetProviders() map[string]protocol.ProviderFactory {
	sourceProviders := make(map[string]protocol.ProviderFactory)
	for id, factory := range r.sourceProviderFactories {
		sourceProviders[id] = factory
	}

	return sourceProviders
}

// GetAvailableNodes returns all available node types.
func (r *Registry) GetAvailableNodes() []protocol.NodeFactory {
	nodes := make([]protocol.NodeFactory, 0, len(r.nodeFactories))
	for _, node := range r.nodeFactories {
		nodes = append(nodes, node)
	}

	return nodes
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

		l.InfoContext(ctx, "Loaded plugin", slog.String("plugin", p))
	}

	return pluginList, nil
}
