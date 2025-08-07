package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidWebhookSource is returned when webhook source validation fails.
var ErrInvalidWebhookSource = errors.New("invalid webhook source")

// WebhookSource represents a webhook endpoint configuration with UUID-based security mapping.
// Each webhook source maps an external UUID to an internal source ID for security.
type WebhookSource struct {
	// ID is the internal source identifier used in workflows
	ID string `json:"id" validate:"required"`

	// SourceID is the source ID from workflow triggers (same as ID for consistency)
	SourceID string `json:"source_id" validate:"required"`

	// UUID is the external UUID used in webhook URLs for security obfuscation
	UUID string `json:"uuid" validate:"required"`

	// JSONSchema contains optional JSON schema for request body validation
	JSONSchema map[string]any `json:"json_schema,omitempty"`

	// Configuration contains webhook-specific settings from trigger configuration
	Configuration map[string]any `json:"configuration"`

	// CreatedAt is the timestamp when this source was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when this source was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// Active indicates if this webhook source is active and should receive requests
	Active bool `json:"active"`
}

// NewWebhookSource creates a new webhook source with the given parameters.
// Automatically generates a random UUID for external access and sets timestamps.
func NewWebhookSource(sourceID string, configuration map[string]any) (*WebhookSource, error) {
	if sourceID == "" {
		return nil, ErrInvalidWebhookSource
	}

	if configuration == nil {
		configuration = make(map[string]any)
	}

	// Generate random UUID for external access
	webhookUUID := uuid.New().String()
	now := time.Now().UTC()

	source := &WebhookSource{
		ID:            sourceID,
		SourceID:      sourceID,
		UUID:          webhookUUID,
		Configuration: configuration,
		CreatedAt:     now,
		UpdatedAt:     now,
		Active:        true,
	}

	// Extract optional JSON schema from configuration
	if schema, exists := configuration["json_schema"]; exists {
		if schemaMap, ok := schema.(map[string]any); ok {
			source.JSONSchema = schemaMap
		}
	}

	return source, nil
}

// Validate performs validation on the webhook source structure.
func (ws *WebhookSource) Validate() error {
	if ws.ID == "" {
		return ErrInvalidWebhookSource
	}

	if ws.SourceID == "" {
		return ErrInvalidWebhookSource
	}

	if ws.UUID == "" {
		return ErrInvalidWebhookSource
	}

	// Validate UUID format
	if _, err := uuid.Parse(ws.UUID); err != nil {
		return errors.New("invalid UUID format")
	}

	return nil
}

// GetWebhookURL returns the webhook URL path for this source.
func (ws *WebhookSource) GetWebhookURL() string {
	return "/webhook/" + ws.UUID
}

// HasJSONSchema returns true if this webhook source has JSON schema validation configured.
func (ws *WebhookSource) HasJSONSchema() bool {
	return len(ws.JSONSchema) > 0
}

// UpdateConfiguration updates the webhook source configuration and timestamp.
func (ws *WebhookSource) UpdateConfiguration(config map[string]any) {
	ws.Configuration = config
	ws.UpdatedAt = time.Now().UTC()

	// Update JSON schema if present
	if schema, exists := config["json_schema"]; exists {
		if schemaMap, ok := schema.(map[string]any); ok {
			ws.JSONSchema = schemaMap
		}
	} else {
		ws.JSONSchema = nil
	}
}

// MarshalJSON implements the json.Marshaler interface.
func (ws *WebhookSource) MarshalJSON() ([]byte, error) {
	type Alias WebhookSource

	return json.Marshal(&struct {
		*Alias

		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias:     (*Alias)(ws),
		CreatedAt: ws.CreatedAt.Format(time.RFC3339),
		UpdatedAt: ws.UpdatedAt.Format(time.RFC3339),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (ws *WebhookSource) UnmarshalJSON(data []byte) error {
	type Alias WebhookSource

	aux := &struct {
		*Alias

		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		Alias: (*Alias)(ws),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	if aux.CreatedAt != "" {
		ws.CreatedAt, err = time.Parse(time.RFC3339, aux.CreatedAt)
		if err != nil {
			return err
		}
	}

	if aux.UpdatedAt != "" {
		ws.UpdatedAt, err = time.Parse(time.RFC3339, aux.UpdatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}
