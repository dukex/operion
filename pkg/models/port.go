// Package models defines port-based workflow models for node connections.
package models

// Port represents a connection point on a node.
type Port struct {
	ID          string         `json:"id"`      // Globally unique: "{nodeID}:{portName}"
	NodeID      string         `json:"node_id"` // Which node this port belongs to
	Name        string         `json:"name"`    // Port name (unique within node)
	Description string         `json:"description"`
	Schema      map[string]any `json:"schema,omitempty"`
}

// InputPort extends Port with input-specific properties.
type InputPort struct {
	Port
	// Note: Required information is now available through GetInputRequirements()
}

// OutputPort extends Port with output-specific properties.
type OutputPort struct {
	Port
	// Could add output-specific fields like default values, etc.
}

// PortDirection represents the direction of data flow for a port.
type PortDirection string

const (
	PortDirectionInput  PortDirection = "input"
	PortDirectionOutput PortDirection = "output"
)

// GetPortDirection returns the direction of the port based on its type.
func (p InputPort) GetDirection() PortDirection {
	return PortDirectionInput
}

// GetPortDirection returns the direction of the port based on its type.
func (p OutputPort) GetDirection() PortDirection {
	return PortDirectionOutput
}

// ParsePortID parses a port ID in format "{node_id}:{port_name}" into components.
func ParsePortID(portID string) (string, string, bool) {
	for i := range len(portID) {
		if portID[i] == ':' {
			return portID[:i], portID[i+1:], true
		}
	}

	return "", "", false
}

// MakePortID creates a port ID from node ID and port name.
func MakePortID(nodeID, portName string) string {
	return nodeID + ":" + portName
}
