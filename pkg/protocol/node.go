// Package protocol defines the interfaces and contracts for pluggable nodes.
package protocol

import (
	"context"

	"github.com/dukex/operion/pkg/models"
)

// NodeFactory creates node instances and provides metadata about the node type.
type NodeFactory interface {
	// Create creates a new node instance with the given configuration
	Create(ctx context.Context, id string, config map[string]any) (models.Node, error)

	// ID returns the unique identifier for this node type
	ID() string

	// Name returns the human-readable name for this node type
	Name() string

	// Description returns a description of what this node does
	Description() string

	// Schema returns the JSON schema for configuring this node
	Schema() map[string]any
}
