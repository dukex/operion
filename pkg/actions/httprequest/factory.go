package httprequest

import (
	"context"

	"github.com/operion-flow/interfaces"
)

// ActionFactory creates HTTPRequestAction instances.
type ActionFactory struct{}

// NewActionFactory creates a new HTTPRequestActionFactory.
func NewActionFactory() *ActionFactory {
	return &ActionFactory{}
}

// Create creates a new HTTPRequestAction from the given configuration.
func (h *ActionFactory) Create(_ context.Context, config map[string]any) (interfaces.Action, error) {
	return NewAction(config)
}

// ID returns the unique identifier for the action.
func (h *ActionFactory) ID() string {
	return "http_request"
}

// Name returns the name of the action.
func (h *ActionFactory) Name() string {
	return "HTTP Request"
}

// Description returns a brief description of the action.
func (h *ActionFactory) Description() string {
	return "Performs an HTTP request to a specified URL with optional headers and body."
}

// Schema returns the JSON schema for configuring this action.
func (h *ActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"title":       "URL",
				"type":        "string",
				"description": "The URL to send the HTTP request to. Supports templating with step results.",
				"examples": []string{
					"https://api.example.com/users",
					"https://api.example.com/users/{{step_results.get_user_id.user_id}}",
					"{{trigger.webhook.url}}/callback",
				},
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method to use (GET, POST, PUT, DELETE, etc.)",
				"default":     "GET",
				"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "HTTP headers to include in the request. Values support templating.",
				"additionalProperties": map[string]any{
					"type": "string",
				},
				"examples": []map[string]string{
					{
						"Content-Type":  "application/json",
						"Authorization": "Bearer your-token-here",
					},
					{
						"Content-Type":  "application/json",
						"Authorization": "Bearer {{.step_results.auth.token}}",
						"X-User-ID":     "{{.trigger_data.webhook.user_id}}",
					},
				},
			},
			"body": map[string]any{
				"type":        "string",
				"format":      "code",
				"description": "Request body content. Supports templating for dynamic JSON or text content.",
				"examples": []string{
					`{"name": "John Doe", "email": "john@example.com"}`,
					`{"user_id": "{{.step_results.create_user.id}}", "status": "active"}`,
					`{"message": "Hello {{.trigger_data.webhook.name}}", "timestamp": "{{now}}"}`,
				},
			},
			"retries": map[string]any{
				"type":        "object",
				"description": "Retry configuration for failed requests",
				"properties": map[string]any{
					"attempts": map[string]any{
						"type":        "integer",
						"description": "Number of retry attempts on failure",
						"default":     0,
						"minimum":     0,
						"maximum":     5, //nolint:mnd // example value
					},
					"delay": map[string]any{
						"type":        "integer",
						"description": "Delay between retry attempts in milliseconds",
						"default":     1000,  //nolint:mnd // example value
						"minimum":     100,   //nolint:mnd // example value
						"maximum":     30000, //nolint:mnd // example value
					},
				},
				"examples": []map[string]any{
					{
						"attempts": 3,    //nolint:mnd // example value
						"delay":    1000, //nolint:mnd // example value
					},
					{
						"attempts": 2,    //nolint:mnd // example value
						"delay":    2000, //nolint:mnd // example value
					},
				},
			},
		},
		"required":             []string{"url", "method"},
		"additionalProperties": false,
	}
}
