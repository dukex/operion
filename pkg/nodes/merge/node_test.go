package merge

import (
	"testing"

	"github.com/dukex/operion/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMergeNode(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "all",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)
	assert.Equal(t, "test-merge", node.ID())
	assert.Equal(t, "merge", node.Type())
	assert.Equal(t, []string{"left", "right"}, node.inputPorts)
	assert.Equal(t, MergeModeAll, node.mergeMode)
}

func TestNewMergeNode_DefaultValues(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)
	assert.Equal(t, MergeModeAll, node.mergeMode)
}

func TestNewMergeNode_InvalidConfig(t *testing.T) {
	// Missing input_ports
	config := map[string]any{}
	_, err := NewMergeNode("test-merge", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field 'input_ports'")

	// Invalid input port type
	config = map[string]any{
		"input_ports": []any{"left", 123},
	}
	_, err = NewMergeNode("test-merge", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestMergeNode_InputRequirements_All(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right", "center"},
		"merge_mode":  "all",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	requirements := node.InputRequirements()
	assert.Equal(t, []string{"left", "right", "center"}, requirements.RequiredPorts)
	assert.Equal(t, []string{}, requirements.OptionalPorts)
	assert.Equal(t, models.WaitModeAll, requirements.WaitMode)
	assert.Nil(t, requirements.Timeout)
}

func TestMergeNode_InputRequirements_Any(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "any",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	requirements := node.InputRequirements()
	assert.Equal(t, []string{"left", "right"}, requirements.RequiredPorts)
	assert.Equal(t, []string{}, requirements.OptionalPorts)
	assert.Equal(t, models.WaitModeAny, requirements.WaitMode)
	assert.Nil(t, requirements.Timeout)
}

func TestMergeNode_InputRequirements_First(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "first",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	requirements := node.InputRequirements()
	assert.Equal(t, []string{"left", "right"}, requirements.RequiredPorts)
	assert.Equal(t, []string{}, requirements.OptionalPorts)
	assert.Equal(t, models.WaitModeFirst, requirements.WaitMode)
	assert.Nil(t, requirements.Timeout)
}

func TestMergeNode_Execute_All_Mode(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "all",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	ctx := models.ExecutionContext{
		ID: "test-execution",
	}

	inputs := map[string]models.NodeResult{
		"left": {
			NodeID: "node-a",
			Data:   map[string]any{"value": "left-data"},
			Status: string(models.NodeStatusSuccess),
		},
		"right": {
			NodeID: "node-b",
			Data:   map[string]any{"value": "right-data"},
			Status: string(models.NodeStatusSuccess),
		},
	}

	outputs, err := node.Execute(ctx, inputs)
	require.NoError(t, err)

	// Verify merged output
	assert.Contains(t, outputs, OutputPortMerged)
	mergedOutput := outputs[OutputPortMerged]
	assert.Equal(t, node.ID(), mergedOutput.NodeID)
	assert.Equal(t, string(models.NodeStatusSuccess), mergedOutput.Status)

	// Verify merged data
	mergedData, ok := mergedOutput.Data["merged_inputs"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, map[string]any{"value": "left-data"}, mergedData["left"])
	assert.Equal(t, map[string]any{"value": "right-data"}, mergedData["right"])

	// Verify inputs received
	inputsReceived, ok := mergedOutput.Data["inputs_received"].([]string)
	require.True(t, ok)
	assert.Contains(t, inputsReceived, "left")
	assert.Contains(t, inputsReceived, "right")
	assert.Equal(t, "all", mergedOutput.Data["merge_mode"])
}

func TestMergeNode_Execute_First_Mode(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "first",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	ctx := models.ExecutionContext{
		ID: "test-execution",
	}

	inputs := map[string]models.NodeResult{
		"left": {
			NodeID: "node-a",
			Data:   map[string]any{"value": "left-data"},
			Status: string(models.NodeStatusSuccess),
		},
		"right": {
			NodeID: "node-b",
			Data:   map[string]any{"value": "right-data"},
			Status: string(models.NodeStatusSuccess),
		},
	}

	outputs, err := node.Execute(ctx, inputs)
	require.NoError(t, err)

	// Verify merged output
	assert.Contains(t, outputs, OutputPortMerged)
	mergedOutput := outputs[OutputPortMerged]

	// For "first" mode, only the first input should be kept
	mergedData, ok := mergedOutput.Data["merged_inputs"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 1, len(mergedData)) // Only one input should remain

	inputsReceived, ok := mergedOutput.Data["inputs_received"].([]string)
	require.True(t, ok)
	assert.Equal(t, 1, len(inputsReceived)) // Only one input should be recorded
}

func TestMergeNode_Execute_InvalidMergeMode(t *testing.T) {
	// Create node with valid config but set invalid merge mode directly
	node := &MergeNode{
		id:         "test-merge",
		inputPorts: []string{"left", "right"},
		mergeMode:  "invalid-mode",
	}

	ctx := models.ExecutionContext{
		ID: "test-execution",
	}

	inputs := map[string]models.NodeResult{
		"left": {
			NodeID: "node-a",
			Data:   map[string]any{"value": "left-data"},
			Status: string(models.NodeStatusSuccess),
		},
	}

	outputs, err := node.Execute(ctx, inputs)
	require.NoError(t, err)

	// Should return error output
	assert.Contains(t, outputs, OutputPortError)
	errorOutput := outputs[OutputPortError]
	assert.Equal(t, string(models.NodeStatusError), errorOutput.Status)
	assert.Contains(t, errorOutput.Data["error"], "unknown merge mode")
}

func TestMergeNode_GetInputPorts(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right", "center"},
		"merge_mode":  "all",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	inputPorts := node.GetInputPorts()
	assert.Equal(t, 3, len(inputPorts))

	// Check port details
	expectedPortNames := []string{"left", "right", "center"}
	for i, port := range inputPorts {
		expectedPortName := expectedPortNames[i]
		assert.Equal(t, expectedPortName, port.Name)
		assert.Equal(t, node.ID(), port.NodeID)
		assert.Equal(t, models.MakePortID(node.ID(), expectedPortName), port.ID)
		// Required information is now available through InputRequirements()
		assert.Contains(t, port.Description, expectedPortName)
	}
}

func TestMergeNode_GetInputPorts_AnyMode(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "any",
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	inputPorts := node.GetInputPorts()
	assert.Equal(t, 2, len(inputPorts))

	// Required information is now available through InputRequirements()
	// Ports themselves no longer have the Required field
}

func TestMergeNode_GetOutputPorts(t *testing.T) {
	config := map[string]any{
		"input_ports": []any{"left", "right"},
	}

	node, err := NewMergeNode("test-merge", config)
	require.NoError(t, err)

	outputPorts := node.GetOutputPorts()
	assert.Equal(t, 2, len(outputPorts))

	// Check merged port
	var mergedPort, errorPort *models.OutputPort

	for _, port := range outputPorts {
		switch port.Name {
		case OutputPortMerged:
			mergedPort = &port
		case OutputPortError:
			errorPort = &port
		}
	}

	require.NotNil(t, mergedPort)
	assert.Equal(t, models.MakePortID(node.ID(), OutputPortMerged), mergedPort.ID)
	assert.Equal(t, node.ID(), mergedPort.NodeID)
	assert.Contains(t, mergedPort.Description, "Combined data")

	require.NotNil(t, errorPort)
	assert.Equal(t, models.MakePortID(node.ID(), OutputPortError), errorPort.ID)
	assert.Equal(t, node.ID(), errorPort.NodeID)
	assert.Contains(t, errorPort.Description, "Error information")
}

func TestMergeNode_Validate(t *testing.T) {
	node := &MergeNode{}

	// Valid config
	validConfig := map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "all",
	}
	err := node.Validate(validConfig)
	assert.NoError(t, err)

	// Missing input_ports
	invalidConfig := map[string]any{}
	err = node.Validate(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field 'input_ports'")

	// Too few input ports
	invalidConfig = map[string]any{
		"input_ports": []any{"single"},
	}
	err = node.Validate(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 2 input ports")

	// Invalid merge mode
	invalidConfig = map[string]any{
		"input_ports": []any{"left", "right"},
		"merge_mode":  "invalid",
	}
	err = node.Validate(invalidConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid merge_mode")
}
