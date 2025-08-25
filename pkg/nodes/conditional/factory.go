// Package conditional provides conditional branching node factory for registry integration.
package conditional

import (
	"context"

	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/protocol"
)

// ConditionalNodeFactory creates ConditionalNode instances.
type ConditionalNodeFactory struct{}

// Create creates a new ConditionalNode instance.
func (f *ConditionalNodeFactory) Create(ctx context.Context, id string, config map[string]any) (models.Node, error) {
	return NewConditionalNode(id, config)
}

// ID returns the factory ID.
func (f *ConditionalNodeFactory) ID() string {
	return "conditional"
}

// Name returns the factory name.
func (f *ConditionalNodeFactory) Name() string {
	return "Conditional"
}

// Description returns the factory description.
func (f *ConditionalNodeFactory) Description() string {
	return "Evaluates a condition and routes execution to true or false paths. Essential for workflow branching logic."
}

// Schema returns the JSON schema for Conditional node configuration.
func (f *ConditionalNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"condition": map[string]any{
				"type":        "string",
				"description": "Condition expression to evaluate. Supports templating and various data types.",
				"examples": []string{
					`{{.variables.status}} == "active"`,
					`{{.node_results.api_call.status_code}} == 200`,
					`{{gt .variables.count 10}}`,
					`{{.trigger_data.webhook.action}} == "created"`,
					`{{and (.variables.enabled) (ne .variables.mode "test")}}`,
					`true`,
					`false`,
					`{{.variables.user_count}}`, // Non-zero numbers are truthy
				},
			},
		},
		"required": []string{"condition"},
		"examples": []map[string]any{
			{
				"condition": `{{.variables.environment}} == "production"`,
			},
			{
				"condition": `{{gt .node_results.calculate_score.score 75}}`,
			},
			{
				"condition": `{{and (.trigger_data.webhook.verified) (.variables.processing_enabled)}}`,
			},
		},
	}
}

// NewConditionalNodeFactory creates a new factory instance.
func NewConditionalNodeFactory() protocol.NodeFactory {
	return &ConditionalNodeFactory{}
}
