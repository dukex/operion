// Package log provides logging node factory for registry integration.
package log

import (
	"context"

	"github.com/dukex/operion/pkg/protocol"
)

// LogNodeFactory creates LogNode instances.
type LogNodeFactory struct{}

// Create creates a new LogNode instance.
func (f *LogNodeFactory) Create(ctx context.Context, id string, config map[string]any) (protocol.Node, error) {
	return NewLogNode(id, config)
}

// ID returns the factory ID.
func (f *LogNodeFactory) ID() string {
	return "log"
}

// Name returns the factory name.
func (f *LogNodeFactory) Name() string {
	return "Log"
}

// Description returns the factory description.
func (f *LogNodeFactory) Description() string {
	return "Logs messages at different levels (debug, info, warn, error) with template support for dynamic content"
}

// Schema returns the JSON schema for Log node configuration.
func (f *LogNodeFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Message to log. Supports templating with execution context data.",
				"examples": []string{
					"Processing user: {{.variables.user_name}}",
					"Workflow {{.execution_context.published_workflow_id}} started",
					"API call result: {{.node_results.api_call.status}}",
					"Error processing item {{.variables.item_id}}: {{.node_results.validation.error}}",
					"Debug: Current state = {{.variables.current_state}}",
				},
			},
			"level": map[string]any{
				"type":        "string",
				"description": "Log level for the message",
				"enum":        []string{"debug", "info", "warn", "error"},
				"default":     "info",
				"examples":    []string{"info", "warn", "error", "debug"},
			},
		},
		"required": []string{"message"},
		"examples": []map[string]any{
			{
				"message": "Starting workflow execution for user {{.variables.user_id}}",
				"level":   "info",
			},
			{
				"message": "API call failed: {{.node_results.http_request.error}}",
				"level":   "error",
			},
			{
				"message": "Processing {{len .trigger_data.items}} items",
				"level":   "debug",
			},
			{
				"message": "Warning: Rate limit approaching for {{.variables.api_key}}",
				"level":   "warn",
			},
		},
	}
}

// NewLogNodeFactory creates a new factory instance.
func NewLogNodeFactory() protocol.NodeFactory {
	return &LogNodeFactory{}
}
