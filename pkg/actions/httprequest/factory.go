package httprequest

import (
	"github.com/dukex/operion/pkg/protocol"
)

func NewHTTPRequestActionFactory() *HTTPRequestActionFactory {
	return &HTTPRequestActionFactory{}
}

type HTTPRequestActionFactory struct{}

func (h *HTTPRequestActionFactory) Create(config map[string]any) (protocol.Action, error) {
	return NewHTTPRequestAction(config)
}

func (h *HTTPRequestActionFactory) ID() string {
	return "http_request"
}

func (h *HTTPRequestActionFactory) Name() string {
	return "HTTP Request"
}

func (h *HTTPRequestActionFactory) Description() string {
	return "Performs an HTTP request to a specified URL with optional headers and body."
}

func (h *HTTPRequestActionFactory) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"title":       "URL",
				"type":        "string",
				"description": "The URL to send the HTTP request to. Supports templating with step results.",
				"examples": []string{
					"https://api.example.com/users",
					"https://api.example.com/users/{{steps.get_user_id.user_id}}",
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
						"Authorization": "Bearer {{steps.auth.token}}",
						"X-User-ID":     "{{trigger.webhook.user_id}}",
					},
				},
			},
			"body": map[string]any{
				"type":        "string",
				"format":      "code",
				"description": "Request body content. Supports templating for dynamic JSON or text content.",
				"examples": []string{
					`{"name": "John Doe", "email": "john@example.com"}`,
					`{"user_id": "{{steps.create_user.id}}", "status": "active"}`,
					`{"message": "Hello {{trigger.webhook.name}}", "timestamp": "{{now}}"}`,
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
						"maximum":     5,
					},
					"delay": map[string]any{
						"type":        "integer",
						"description": "Delay between retry attempts in milliseconds",
						"default":     1000,
						"minimum":     100,
						"maximum":     30000,
					},
				},
				"examples": []map[string]any{
					{
						"attempts": 3,
						"delay":    1000,
					},
					{
						"attempts": 2,
						"delay":    2000,
					},
				},
			},
		},
		"required": []string{"url", "method"},
		"additionalProperties": false,
	}
}
