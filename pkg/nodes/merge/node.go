// Package merge provides merge node implementation for joining multiple execution paths.
package merge

import (
	"errors"
	"fmt"

	"github.com/dukex/operion/pkg/models"
)

const (
	OutputPortMerged = "merged"
	OutputPortError  = "error"
	MergeModeAll     = "all"
	MergeModeAny     = "any"
	MergeModeFirst   = "first"
)

// MergeNode implements the Node interface for merging multiple execution paths
// This node waits for all input paths and combines their data.
type MergeNode struct {
	id         string
	inputPorts []string
	mergeMode  string // "all", "any", "first"
}

// NewMergeNode creates a new merge node.
func NewMergeNode(id string, config map[string]any) (*MergeNode, error) {
	// Parse input ports (required)
	inputPortsAny, ok := config["input_ports"].([]any)
	if !ok {
		return nil, errors.New("missing required field 'input_ports'")
	}

	inputPorts := make([]string, len(inputPortsAny))
	for i, port := range inputPortsAny {
		if portStr, ok := port.(string); ok {
			inputPorts[i] = portStr
		} else {
			return nil, fmt.Errorf("input_port %d must be a string", i)
		}
	}

	// Parse merge mode (optional, defaults to "all")
	mergeMode := MergeModeAll
	if mode, ok := config["merge_mode"].(string); ok {
		mergeMode = mode
	}

	return &MergeNode{
		id:         id,
		inputPorts: inputPorts,
		mergeMode:  mergeMode,
	}, nil
}

// ID returns the node ID.
func (n *MergeNode) ID() string {
	return n.id
}

// Type returns the node type.
func (n *MergeNode) Type() string {
	return "merge"
}

// InputRequirements returns the input coordination requirements for this merge node.
func (n *MergeNode) InputRequirements() models.InputRequirements {
	waitMode := models.WaitModeAll

	switch n.mergeMode {
	case MergeModeAny:
		waitMode = models.WaitModeAny
	case MergeModeFirst:
		waitMode = models.WaitModeFirst
	}

	return models.InputRequirements{
		RequiredPorts: n.inputPorts,
		OptionalPorts: []string{},
		WaitMode:      waitMode,
		Timeout:       nil,
	}
}

// Execute merges inputs from multiple execution paths.
// The worker manager now handles input coordination, so this node just processes
// the inputs it receives (which are guaranteed to be ready based on requirements).
func (n *MergeNode) Execute(ctx models.ExecutionContext, inputs map[string]models.NodeResult) (map[string]models.NodeResult, error) {
	// Simple business logic - no coordination needed since worker handles that
	mergedData := make(map[string]any)
	inputsReceived := make([]string, 0, len(inputs))

	// Process all provided inputs (worker already ensured they meet requirements)
	for portName, result := range inputs {
		mergedData[portName] = result.Data
		inputsReceived = append(inputsReceived, portName)
	}

	// Apply merge mode-specific processing if needed
	switch n.mergeMode {
	case MergeModeFirst:
		// For "first" mode, keep only the first input (though worker should have handled this)
		if len(inputsReceived) > 1 {
			firstPort := inputsReceived[0]
			mergedData = map[string]any{firstPort: mergedData[firstPort]}
			inputsReceived = []string{firstPort}
		}
	case MergeModeAll, MergeModeAny:
		// For "all" and "any" modes, use all provided inputs
		// Worker has already ensured the right inputs are provided
	default:
		return n.createErrorResult("unknown merge mode: " + n.mergeMode), nil
	}

	// Return merged result
	return map[string]models.NodeResult{
		OutputPortMerged: {
			NodeID: n.id,
			Data: map[string]any{
				"merged_inputs":   mergedData,
				"inputs_received": inputsReceived,
				"merge_mode":      n.mergeMode,
			},
			Status: string(models.NodeStatusSuccess),
		},
	}, nil
}

// createErrorResult creates a NodeResult for the error output port.
func (n *MergeNode) createErrorResult(errorMessage string) map[string]models.NodeResult {
	return map[string]models.NodeResult{
		OutputPortError: {
			NodeID: n.id,
			Data: map[string]any{
				"error":   errorMessage,
				"success": false,
			},
			Status: string(models.NodeStatusError),
		},
	}
}

// InputPorts returns the input ports for the node (dynamic based on configuration).
func (n *MergeNode) InputPorts() []models.InputPort {
	ports := make([]models.InputPort, 0, len(n.inputPorts))

	for _, port := range n.inputPorts {
		ports = append(ports, models.InputPort{
			Port: models.Port{
				ID:          models.MakePortID(n.id, port),
				NodeID:      n.id,
				Name:        port,
				Description: fmt.Sprintf("Input from execution path '%s'", port),
			},
		})
	}

	return ports
}

// OutputPorts returns the output ports for the node.
func (n *MergeNode) OutputPorts() []models.OutputPort {
	return []models.OutputPort{
		{
			Port: models.Port{
				ID:          models.MakePortID(n.id, OutputPortMerged),
				NodeID:      n.id,
				Name:        OutputPortMerged,
				Description: "Combined data from all input execution paths",
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"merged_inputs":   map[string]any{"type": "object"},
						"inputs_received": map[string]any{"type": "array"},
						"merge_mode":      map[string]any{"type": "string"},
					},
				},
			},
		},
		{
			Port: models.Port{
				ID:          models.MakePortID(n.id, OutputPortError),
				NodeID:      n.id,
				Name:        OutputPortError,
				Description: "Error information when merge operation fails",
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"error":   map[string]any{"type": "string"},
						"success": map[string]any{"type": "boolean"},
					},
				},
			},
		},
	}
}

// Validate validates the node configuration.
func (n *MergeNode) Validate(config map[string]any) error {
	// Validate input_ports
	inputPortsAny, ok := config["input_ports"].([]any)
	if !ok {
		return errors.New("missing required field 'input_ports'")
	}

	if len(inputPortsAny) < 2 {
		return errors.New("merge node requires at least 2 input ports")
	}

	// Validate merge_mode if provided
	if mode, ok := config["merge_mode"].(string); ok {
		validModes := map[string]bool{MergeModeAll: true, MergeModeAny: true, MergeModeFirst: true}
		if !validModes[mode] {
			return fmt.Errorf("invalid merge_mode: %s (must be 'all', 'any', or 'first')", mode)
		}
	}

	return nil
}
