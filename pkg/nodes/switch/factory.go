// Package switch provides switch node factory for registry integration.
package switchnode

import (
	"context"

	"github.com/dukex/operion/pkg/protocol"
)

// SwitchNodeFactory creates SwitchNode instances.
type SwitchNodeFactory struct{}

// Create creates a new SwitchNode instance.
func (f *SwitchNodeFactory) Create(ctx context.Context, id string, config map[string]any) (protocol.Node, error) {
	return NewSwitchNode(id, config)
}

// ID returns the factory ID.
func (f *SwitchNodeFactory) ID() string {
	return "switch"
}

// Name returns the factory name.
func (f *SwitchNodeFactory) Name() string {
	return "Switch"
}

// Description returns the factory description.
func (f *SwitchNodeFactory) Description() string {
	return "Multi-way branching node that routes execution to different paths based on a value match"
}

// Schema returns the JSON schema for Switch node configuration.
func (f *SwitchNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"value": map[string]any{
				"type":        "string",
				"description": "Expression to evaluate for switch routing. Supports templating.",
				"examples": []string{
					`{{.variables.environment}}`,
					`{{.node_results.api_call.status}}`,
					`{{.trigger_data.webhook.event_type}}`,
					`{{.variables.user_role}}`,
				},
			},
			"cases": map[string]any{
				"type":        "array",
				"description": "Array of case objects defining value-to-output-port mappings",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"value": map[string]any{
							"type":        "string",
							"description": "Value to match against the evaluated expression",
						},
						"output_port": map[string]any{
							"type":        "string",
							"description": "Output port name to route to when this value matches",
						},
					},
					"required": []string{"value", "output_port"},
				},
				"examples": [][]map[string]any{
					{
						{"value": "production", "output_port": "prod_path"},
						{"value": "staging", "output_port": "staging_path"},
						{"value": "development", "output_port": "dev_path"},
					},
					{
						{"value": "success", "output_port": "continue"},
						{"value": "error", "output_port": "error_handler"},
						{"value": "retry", "output_port": "retry_logic"},
					},
				},
			},
		},
		"required": []string{"value"},
		"examples": []map[string]any{
			{
				"value": `{{.variables.deployment_env}}`,
				"cases": []map[string]any{
					{"value": "production", "output_port": "prod_deployment"},
					{"value": "staging", "output_port": "staging_tests"},
					{"value": "development", "output_port": "dev_testing"},
				},
			},
			{
				"value": `{{.node_results.check_status.result}}`,
				"cases": []map[string]any{
					{"value": "healthy", "output_port": "continue_processing"},
					{"value": "warning", "output_port": "alert_team"},
					{"value": "critical", "output_port": "emergency_response"},
				},
			},
		},
	}
}

// NewSwitchNodeFactory creates a new factory instance.
func NewSwitchNodeFactory() protocol.NodeFactory {
	return &SwitchNodeFactory{}
}
