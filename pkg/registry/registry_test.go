package registry

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock node for testing.
type mockNode struct {
	id     string
	config map[string]any
}

func (m *mockNode) ID() string {
	return m.id
}

func (m *mockNode) Type() string {
	return "mock-node"
}

func (m *mockNode) Execute(ctx models.ExecutionContext, inputs map[string]models.NodeResult) (map[string]models.NodeResult, error) {
	return map[string]models.NodeResult{
		"output": {
			NodeID: m.id,
			Data: map[string]any{
				"config": m.config,
				"inputs": inputs,
				"result": "mock execution completed",
			},
			Status: "success",
		},
	}, nil
}

func (m *mockNode) GetInputPorts() []models.InputPort {
	return []models.InputPort{{
		Port: models.Port{
			ID:     m.id + ":input",
			NodeID: m.id,
			Name:   "input",
		},
	}}
}

func (m *mockNode) GetOutputPorts() []models.OutputPort {
	return []models.OutputPort{{
		Port: models.Port{
			ID:     m.id + ":output",
			NodeID: m.id,
			Name:   "output",
		},
	}}
}

func (m *mockNode) Validate(config map[string]any) error {
	return nil
}

// Mock node factory for testing.
type mockNodeFactory struct {
	nodeType string
}

func (f *mockNodeFactory) ID() string {
	return f.nodeType
}

func (f *mockNodeFactory) Name() string {
	return "Mock Node"
}

func (f *mockNodeFactory) Description() string {
	return "This is a mock node for testing purposes."
}

func (f *mockNodeFactory) Create(_ context.Context, nodeID string, config map[string]any) (protocol.Node, error) {
	return &mockNode{
		id:     nodeID,
		config: config,
	}, nil
}

func (f *mockNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test_field": map[string]any{
				"type":        "string",
				"description": "Test field for mock node",
			},
		},
	}
}

// Mock provider for testing.
type mockProvider struct {
	id     string
	config map[string]any
}

func (p *mockProvider) ID() string {
	return p.id
}

func (p *mockProvider) Start(ctx context.Context, callback protocol.SourceEventCallback) error {
	return nil
}

func (p *mockProvider) Stop(ctx context.Context) error {
	return nil
}

func (p *mockProvider) Validate() error {
	return nil
}

// Mock provider factory for testing.
type mockProviderFactory struct {
	providerType string
}

func (f *mockProviderFactory) ID() string {
	return f.providerType
}

func (f *mockProviderFactory) Name() string {
	return "Mock Provider"
}

func (f *mockProviderFactory) Description() string {
	return "Mock provider for testing"
}

func (f *mockProviderFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Provider, error) {
	return &mockProvider{
		id:     f.providerType,
		config: config,
	}, nil
}

func (f *mockProviderFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test": map[string]any{
				"type": "string",
			},
		},
	}
}

func (f *mockProviderFactory) EventTypes() []string {
	return []string{"mock_event"}
}

func TestNewRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	assert.NotNil(t, registry)
	assert.Equal(t, logger, registry.logger)
	assert.NotNil(t, registry.sourceProviderFactories)
	assert.NotNil(t, registry.nodeFactories)
	assert.Empty(t, registry.sourceProviderFactories)
	assert.Empty(t, registry.nodeFactories)
}

func TestRegistry_RegisterAndCreateNode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register mock node factory
	nodeFactory := &mockNodeFactory{nodeType: "test-node"}
	registry.RegisterNode(nodeFactory)

	// Verify registration
	assert.Len(t, registry.nodeFactories, 1)
	assert.Contains(t, registry.nodeFactories, "test-node")

	// Create node instance
	config := map[string]any{
		"param1": "value1",
		"param2": 42,
	}

	node, err := registry.CreateNode(context.Background(), "test-node", "node-123", config)
	require.NoError(t, err)
	assert.NotNil(t, node)

	// Verify node can be executed
	execContext := models.ExecutionContext{ID: "exec-123"}
	inputs := map[string]models.NodeResult{}
	result, err := node.Execute(execContext, inputs)
	require.NoError(t, err)

	assert.Contains(t, result, "output")
	output := result["output"]
	assert.Equal(t, "node-123", output.NodeID)
	assert.Equal(t, "success", output.Status)
	assert.Equal(t, config, output.Data["config"])
	assert.Equal(t, inputs, output.Data["inputs"])
	assert.Equal(t, "mock execution completed", output.Data["result"])
}

func TestRegistry_RegisterAndCreateProvider(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register mock provider factory
	providerFactory := &mockProviderFactory{providerType: "test-provider"}
	registry.RegisterProvider(providerFactory)

	// Verify registration
	assert.Len(t, registry.sourceProviderFactories, 1)
	assert.Contains(t, registry.sourceProviderFactories, "test-provider")

	// Create provider instance
	config := map[string]any{
		"setting1": "value1",
		"setting2": true,
	}

	provider, err := registry.CreateProvider(context.Background(), "test-provider", config)
	require.NoError(t, err)
	assert.NotNil(t, provider)

	// Verify provider can be started and stopped
	err = provider.Start(context.Background(), nil)
	assert.NoError(t, err)

	err = provider.Stop(context.Background())
	assert.NoError(t, err)
}

func TestRegistry_CreateNode_NotRegistered(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to create node that's not registered
	node, err := registry.CreateNode(context.Background(), "non-existent-node", "node-123", map[string]any{})

	assert.Error(t, err)
	assert.Nil(t, node)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_CreateProvider_NotRegistered(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to create provider that's not registered
	provider, err := registry.CreateProvider(context.Background(), "non-existent-provider", map[string]any{})

	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_GetAvailableNodes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register multiple nodes
	nodeFactory1 := &mockNodeFactory{nodeType: "node-1"}
	nodeFactory2 := &mockNodeFactory{nodeType: "node-2"}

	registry.RegisterNode(nodeFactory1)
	registry.RegisterNode(nodeFactory2)

	// Get available nodes
	nodes := registry.GetAvailableNodes()
	assert.Len(t, nodes, 2)

	// Check that both factories are present
	nodeTypes := make(map[string]bool)
	for _, factory := range nodes {
		nodeTypes[factory.ID()] = true
	}

	assert.True(t, nodeTypes["node-1"])
	assert.True(t, nodeTypes["node-2"])
}

func TestRegistry_GetAvailableProviders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register multiple providers
	providerFactory1 := &mockProviderFactory{providerType: "provider-1"}
	providerFactory2 := &mockProviderFactory{providerType: "provider-2"}

	registry.RegisterProvider(providerFactory1)
	registry.RegisterProvider(providerFactory2)

	// Get available providers
	providers := registry.GetAvailableProviders()
	assert.Len(t, providers, 2)

	// Check that both factories are present
	providerTypes := make(map[string]bool)
	for _, factory := range providers {
		providerTypes[factory.ID()] = true
	}

	assert.True(t, providerTypes["provider-1"])
	assert.True(t, providerTypes["provider-2"])
}

func TestRegistry_HealthCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Empty registry should fail health check
	message, ok := registry.HealthCheck()
	assert.False(t, ok)
	assert.Equal(t, "No plugins loaded", message)

	// Registry with nodes should pass health check
	nodeFactory := &mockNodeFactory{nodeType: "test-node"}
	registry.RegisterNode(nodeFactory)

	message, ok = registry.HealthCheck()
	assert.True(t, ok)
	assert.Equal(t, "Plugins loaded successfully", message)

	// Registry with providers should pass health check
	registry2 := NewRegistry(logger)
	providerFactory := &mockProviderFactory{providerType: "test-provider"}
	registry2.RegisterProvider(providerFactory)

	message, ok = registry2.HealthCheck()
	assert.True(t, ok)
	assert.Equal(t, "Plugins loaded successfully", message)
}

func TestRegistry_LoadProviderPlugins_NonExistentPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to load plugins from non-existent path
	factories, err := registry.LoadProviderPlugins(context.Background(), "/non/existent/path")

	// Should not fail, but return empty slice
	assert.NoError(t, err)
	assert.Empty(t, factories)
}

func TestRegistry_LoadPlugins_EmptyDirectory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Create temporary empty directory
	tmpDir := t.TempDir()

	// Try to load plugins from empty directory
	providerFactories, err := registry.LoadProviderPlugins(context.Background(), tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, providerFactories)
}
