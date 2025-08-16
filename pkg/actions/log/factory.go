package log

import (
	"context"

	"github.com/operion-flow/interfaces"
)

// ActionFactory is the factory for creating LogAction instances.
type ActionFactory struct{}

// NewActionFactory creates a new instance of ActionFactory.
func NewActionFactory() *ActionFactory {
	return &ActionFactory{}
}

// ID returns the unique identifier for the action factory.
func (*ActionFactory) ID() string {
	return "log"
}

// Name returns the name of the action factory.
func (*ActionFactory) Name() string {
	return "Log"
}

// Description returns a brief description of the action.
func (*ActionFactory) Description() string {
	return "Logs a message at a specified level. Supports templating for dynamic content."
}

// Create creates a new LogAction instance with the provided configuration.
func (f *ActionFactory) Create(_ context.Context, config map[string]any) (interfaces.Action, error) {
	if config == nil {
		config = map[string]any{}
	}

	return NewLogAction(config), nil
}

// Schema returns the JSON schema for the action configuration.
func (f *ActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to log. Supports templating for dynamic content.",
				"examples": []string{
					"Workflow step completed successfully",
					"Processing user: {{.trigger_data.webhook.user_name}}",
					"HTTP request to {{.step_results.api_call.url}} returned status {{.step_results.api_call.status}}",
					"Received {{.step_results.fetch_data.count}} records at {{now}}",
				},
			},
			"level": map[string]any{
				"type":        "string",
				"description": "Log level for the message",
				"default":     "info",
				"enum":        []string{"debug", "info", "warn", "warning", "error"},
			},
		},
		"required": []string{"message"},
	}
}
