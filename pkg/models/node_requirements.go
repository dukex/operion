// Package models provides core domain models for node input coordination.
package models

import "time"

// NodeInputRequirements interface allows nodes to declare their input coordination needs.
// This is optional - nodes that don't implement this use protocol.GetDefaultInputRequirements().
type NodeInputRequirements interface {
	GetInputRequirements() InputRequirements
}

// InputRequirements defines how a node should wait for and coordinate inputs.
type InputRequirements struct {
	RequiredPorts []string       `json:"required_ports"` // Must receive inputs on all these ports
	OptionalPorts []string       `json:"optional_ports"` // May receive inputs on these ports
	WaitMode      InputWaitMode  `json:"wait_mode"`      // How to handle multiple inputs
	Timeout       *time.Duration `json:"timeout"`        // Optional timeout for input collection
}

// InputWaitMode defines different strategies for waiting for inputs.
type InputWaitMode string

const (
	// WaitModeAll waits for all required ports to have inputs before executing.
	WaitModeAll InputWaitMode = "all"
	// WaitModeAny executes when any required port has input.
	WaitModeAny InputWaitMode = "any"
	// WaitModeFirst executes on first input, ignores subsequent ones.
	WaitModeFirst InputWaitMode = "first"
)

// DefaultInputRequirements returns the standard requirements for single-input nodes.
func DefaultInputRequirements() InputRequirements {
	return InputRequirements{
		RequiredPorts: []string{"main"},
		OptionalPorts: []string{},
		WaitMode:      WaitModeAny,
		Timeout:       nil,
	}
}

// NodeInputState tracks the input collection state for a specific node execution.
// This supports loops by having separate state for each node execution instance.
type NodeInputState struct {
	NodeID          string                `json:"node_id"`           // The workflow node ID
	ExecutionID     string                `json:"execution_id"`      // Workflow execution ID
	NodeExecutionID string                `json:"node_execution_id"` // Individual node execution ID (for loops)
	WorkflowID      string                `json:"workflow_id"`       // Published workflow ID
	ReceivedInputs  map[string]NodeResult `json:"received_inputs"`   // Inputs collected so far
	Requirements    InputRequirements     `json:"requirements"`      // Node's input requirements
	CreatedAt       time.Time             `json:"created_at"`        // When input collection started
	LastUpdatedAt   time.Time             `json:"last_updated_at"`   // When last input was added
}
