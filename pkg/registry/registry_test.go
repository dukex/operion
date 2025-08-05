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

// Mock action for testing.
type mockAction struct {
	id     string
	config map[string]any
}

func (m *mockAction) Execute(ctx context.Context, execCtx models.ExecutionContext, logger *slog.Logger) (any, error) {
	return map[string]any{
		"id":     m.id,
		"config": m.config,
		"result": "mock execution completed",
	}, nil
}

// Mock action factory for testing.
type mockActionFactory struct {
	actionType string
}

func (f *mockActionFactory) ID() string {
	return f.actionType
}

func (f *mockActionFactory) Name() string {
	return "Mock Action"
}

func (f *mockActionFactory) Description() string {
	return "This is a mock action for testing purposes."
}

func (f *mockActionFactory) Create(config map[string]any) (protocol.Action, error) {
	return &mockAction{
		id:     f.actionType,
		config: config,
	}, nil
}

func (f *mockActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test_field": map[string]any{
				"type":        "string",
				"description": "Test field for mock action",
			},
		},
	}
}

// Mock trigger for testing.
type mockTrigger struct {
	id       string
	config   map[string]any
	callback protocol.TriggerCallback
}

func (m *mockTrigger) Start(ctx context.Context, callback protocol.TriggerCallback) error {
	m.callback = callback

	return nil
}

func (m *mockTrigger) Stop(ctx context.Context) error {
	return nil
}

func (m *mockTrigger) Validate() error {
	return nil
}

func (m *mockTrigger) TriggerWorkflow() error {
	if m.callback != nil {
		return m.callback(context.Background(), map[string]any{
			"triggered_by": m.id,
		})
	}

	return nil
}

// Mock trigger factory for testing.
type mockTriggerFactory struct {
	triggerType string
}

func (f *mockTriggerFactory) ID() string {
	return f.triggerType
}

func (f *mockTriggerFactory) Name() string {
	return "Mock Trigger"
}

func (f *mockTriggerFactory) Description() string {
	return "Mock trigger for testing"
}

func (f *mockTriggerFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"test": map[string]any{
				"type": "string",
			},
		},
	}
}

func (f *mockTriggerFactory) Create(config map[string]any, logger *slog.Logger) (protocol.Trigger, error) {
	return &mockTrigger{
		id:     f.triggerType,
		config: config,
	}, nil
}

func TestNewRegistry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	assert.NotNil(t, registry)
	assert.Equal(t, logger, registry.logger)
	assert.NotNil(t, registry.actionFactories)
	assert.NotNil(t, registry.triggerFactories)
	assert.Empty(t, registry.actionFactories)
	assert.Empty(t, registry.triggerFactories)
}

func TestRegistry_RegisterAndCreateAction(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register mock action factory
	actionFactory := &mockActionFactory{actionType: "test-action"}
	registry.RegisterAction(actionFactory)

	// Verify registration
	assert.Len(t, registry.actionFactories, 1)
	assert.Contains(t, registry.actionFactories, "test-action")

	// Create action instance
	config := map[string]any{
		"param1": "value1",
		"param2": 42,
	}

	action, err := registry.CreateAction("test-action", config)
	require.NoError(t, err)
	assert.NotNil(t, action)

	// Verify action can be executed
	execCtx := models.ExecutionContext{
		StepResults: make(map[string]any),
	}

	result, err := action.Execute(context.Background(), execCtx, logger)
	require.NoError(t, err)

	resultMap := result.(map[string]any)
	assert.Equal(t, "test-action", resultMap["id"])
	assert.Equal(t, config, resultMap["config"])
	assert.Equal(t, "mock execution completed", resultMap["result"])
}

func TestRegistry_RegisterAndCreateTrigger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register mock trigger factory
	triggerFactory := &mockTriggerFactory{triggerType: "test-trigger"}
	registry.RegisterTrigger(triggerFactory)

	// Verify registration
	assert.Len(t, registry.triggerFactories, 1)
	assert.Contains(t, registry.triggerFactories, "test-trigger")

	// Create trigger instance
	config := map[string]any{
		"schedule": "* * * * *",
		"enabled":  true,
	}

	trigger, err := registry.CreateTrigger("test-trigger", config)
	require.NoError(t, err)
	assert.NotNil(t, trigger)

	// Verify trigger can be started and stopped
	ctx := context.Background()
	callback := func(ctx context.Context, data map[string]any) error {
		return nil
	}

	err = trigger.Start(ctx, callback)
	assert.NoError(t, err)

	err = trigger.Stop(ctx)
	assert.NoError(t, err)
}

func TestRegistry_CreateAction_NotRegistered(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to create action that's not registered
	action, err := registry.CreateAction("non-existent-action", map[string]any{})

	assert.Error(t, err)
	assert.Nil(t, action)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_CreateTrigger_NotRegistered(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to create trigger that's not registered
	trigger, err := registry.CreateTrigger("non-existent-trigger", map[string]any{})

	assert.Error(t, err)
	assert.Nil(t, trigger)
	assert.Contains(t, err.Error(), "not registered")
}

func TestRegistry_MultipleActionsAndTriggers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register multiple actions
	actionFactory1 := &mockActionFactory{actionType: "action-1"}
	actionFactory2 := &mockActionFactory{actionType: "action-2"}

	registry.RegisterAction(actionFactory1)
	registry.RegisterAction(actionFactory2)

	// Register multiple triggers
	triggerFactory1 := &mockTriggerFactory{triggerType: "trigger-1"}
	triggerFactory2 := &mockTriggerFactory{triggerType: "trigger-2"}

	registry.RegisterTrigger(triggerFactory1)
	registry.RegisterTrigger(triggerFactory2)

	// Verify all are registered
	assert.Len(t, registry.actionFactories, 2)
	assert.Len(t, registry.triggerFactories, 2)

	// Create instances of each
	action1, err := registry.CreateAction("action-1", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, action1)

	action2, err := registry.CreateAction("action-2", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, action2)

	trigger1, err := registry.CreateTrigger("trigger-1", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, trigger1)

	trigger2, err := registry.CreateTrigger("trigger-2", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, trigger2)
}

func TestRegistry_OverwriteRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Register action factory
	actionFactory1 := &mockActionFactory{actionType: "same-action"}
	registry.RegisterAction(actionFactory1)

	// Register different factory with same type (should overwrite)
	actionFactory2 := &mockActionFactory{actionType: "same-action"}
	registry.RegisterAction(actionFactory2)

	// Should still have only one entry
	assert.Len(t, registry.actionFactories, 1)

	// The second factory should be used
	action, err := registry.CreateAction("same-action", map[string]any{})
	require.NoError(t, err)
	assert.NotNil(t, action)
}

func TestRegistry_LoadActionPlugins_NonExistentPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to load plugins from non-existent path
	factories, err := registry.LoadActionPlugins("/non/existent/path")

	// Should not fail, but return empty slice
	assert.NoError(t, err)
	assert.Empty(t, factories)
}

func TestRegistry_LoadTriggerPlugins_NonExistentPath(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Try to load plugins from non-existent path
	factories, err := registry.LoadTriggerPlugins("/non/existent/path")

	// Should not fail, but return empty slice
	assert.NoError(t, err)
	assert.Empty(t, factories)
}

func TestRegistry_LoadPlugins_EmptyDirectory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	registry := NewRegistry(logger)

	// Create temporary empty directory
	tmpDir := os.TempDir()

	// Try to load plugins from empty directory
	actionFactories, err := registry.LoadActionPlugins(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, actionFactories)

	triggerFactories, err := registry.LoadTriggerPlugins(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, triggerFactories)
}

func TestMockAction_Execute(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	action := &mockAction{
		id: "test-mock",
		config: map[string]any{
			"test": "value",
		},
	}

	execCtx := models.ExecutionContext{
		ID:         "exec-123",
		WorkflowID: "workflow-456",
		StepResults: map[string]any{
			"previous": "data",
		},
	}

	result, err := action.Execute(context.Background(), execCtx, logger)

	require.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]any)
	assert.Equal(t, "test-mock", resultMap["id"])
	assert.Equal(t, "mock execution completed", resultMap["result"])
}

func TestMockTrigger_StartStopAndCallback(t *testing.T) {
	trigger := &mockTrigger{
		id: "test-mock-trigger",
		config: map[string]any{
			"enabled": true,
		},
	}

	ctx := context.Background()

	var (
		callbackData   map[string]any
		callbackCalled bool
	)

	callback := func(ctx context.Context, data map[string]any) error {
		callbackData = data
		callbackCalled = true

		return nil
	}

	// Start trigger
	err := trigger.Start(ctx, callback)
	assert.NoError(t, err)

	// Trigger workflow manually
	err = trigger.TriggerWorkflow()
	assert.NoError(t, err)
	assert.True(t, callbackCalled)
	assert.Equal(t, "test-mock-trigger", callbackData["triggered_by"])

	// Stop trigger
	err = trigger.Stop(ctx)
	assert.NoError(t, err)
}
