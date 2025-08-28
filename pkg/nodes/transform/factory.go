// Package transform provides data transformation node factory for registry integration.
package transform

import (
	"context"

	"github.com/dukex/operion/pkg/protocol"
)

// TransformNodeFactory creates TransformNode instances.
type TransformNodeFactory struct{}

// Create creates a new TransformNode instance.
func (f *TransformNodeFactory) Create(ctx context.Context, id string, config map[string]any) (protocol.Node, error) {
	return NewTransformNode(id, config)
}

// ID returns the factory ID.
func (f *TransformNodeFactory) ID() string {
	return "transform"
}

// Name returns the factory name.
func (f *TransformNodeFactory) Name() string {
	return "Transform"
}

// Description returns the factory description.
func (f *TransformNodeFactory) Description() string {
	return "Transforms data using Go templates with access to execution context, variables, and node results"
}

// Schema returns the JSON schema for Transform node configuration.
func (f *TransformNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"description": "Go template expression for data transformation. Has access to execution context.",
				"examples": []string{
					`{"user_id": "{{.variables.user_id}}", "status": "active"}`,
					`{{.node_results.api_call.user_name | upper}}`,
					`Processing {{len .trigger_data.items}} items`,
					`{{.variables.first_name}} {{.variables.last_name}}`,
					`{{.node_results.calculate.value | printf "%.2f"}}`,
				},
			},
		},
		"required": []string{"expression"},
		"examples": []map[string]any{
			{
				"expression": `{"full_name": "{{.variables.first_name}} {{.variables.last_name}}", "timestamp": "{{now}}"}`,
			},
			{
				"expression": `{{.node_results.fetch_user.name | title}}`,
			},
			{
				"expression": `{"total": {{add .node_results.sum1.value .node_results.sum2.value}}, "count": {{len .trigger_data.items}}}`,
			},
		},
	}
}

// NewTransformNodeFactory creates a new factory instance.
func NewTransformNodeFactory() protocol.NodeFactory {
	return &TransformNodeFactory{}
}
