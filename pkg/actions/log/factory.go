package log

import (
	"github.com/dukex/operion/pkg/protocol"
)

func NewLogActionFactory() *LogActionFactory {
	return &LogActionFactory{}
}

type LogActionFactory struct{}

func (*LogActionFactory) ID() string {
	return "log"
}

func (*LogActionFactory) Name() string {
	return "Log"
}

func (*LogActionFactory) Description() string {
	return "Logs a message at a specified level. Supports templating for dynamic content."
}

func (f *LogActionFactory) Create(config map[string]any) (protocol.Action, error) {
	if config == nil {
		config = map[string]any{}
	}

	return NewLogAction(config), nil
}

func (f *LogActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to log. Supports templating for dynamic content.",
				"examples": []string{
					"Workflow step completed successfully",
					"Processing user: {{trigger.webhook.user_name}}",
					"HTTP request to {{steps.api_call.url}} returned status {{steps.api_call.status}}",
					"Received {{steps.fetch_data.count}} records at {{now}}",
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
