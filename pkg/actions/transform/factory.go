package transform

import (
	"context"

	"github.com/operion-flow/interfaces"
)

// ActionFactory is the factory for creating Transform actions.
type ActionFactory struct{}

// NewActionFactory creates a new instance of ActionFactory for the Transform action.
func NewActionFactory() *ActionFactory {
	return &ActionFactory{}
}

// Create creates a new Action instance based on the provided configuration.
func (h *ActionFactory) Create(_ context.Context, config map[string]any) (interfaces.Action, error) {
	return NewAction(config)
}

// ID returns the unique identifier for the Action action factory.
func (h *ActionFactory) ID() string {
	return "transform"
}

// Name returns the name of the Action action factory.
func (h *ActionFactory) Name() string {
	return "Transform"
}

// Description returns a brief description of the Transform action.
func (h *ActionFactory) Description() string {
	return "Transforms data using a specified expression."
}

// Schema returns the JSON schema for the Transform action configuration.
func (h *ActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"expression": map[string]any{
				"type":        "string",
				"format":      "template",
				"description": "Go template expression to transform the data. Use Go template syntax with {{}} delimiters.",
				"examples": []string{
					"{{.name}}",
					"{{.users.0.email}}",
					"{\"fullName\": \"{{.firstName}} {{.lastName}}\", \"isActive\": {{eq .status \"active\"}}}",
					"{{range .data.users}}{\"user_id\": {{.id}}, \"display_name\": \"{{.name}}\"}{{end}}",
					"{{len .items}}",
					"{{with .orders}}{{range .}}{{if gt .total 100.0}}{{.}}{{end}}{{end}}{{end}}",
				},
			},
		},
		"required": []string{"expression"},
	}
}
