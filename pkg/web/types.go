// Package web provides HTTP request and response types for the workflow API.
package web

// CreateWorkflowRequest represents the request body for creating a new workflow.
type CreateWorkflowRequest struct {
	Name        string         `json:"name"               validate:"required,min=3"`
	Description string         `json:"description"        validate:"required"`
	Variables   map[string]any `json:"variables"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Owner       string         `json:"owner"              validate:"required"`
}

// UpdateWorkflowRequest represents the request body for updating an existing workflow.
// All fields are optional to support partial updates.
type UpdateWorkflowRequest struct {
	Name        *string        `json:"name,omitempty"        validate:"omitempty,min=3"`
	Description *string        `json:"description,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CreateNodeRequest represents the request body for creating a new workflow node.
type CreateNodeRequest struct {
	Type      string         `json:"type"       validate:"required"`
	Category  string         `json:"category"   validate:"required,oneof=action trigger"`
	Config    map[string]any `json:"config"`
	PositionX int            `json:"position_x"`
	PositionY int            `json:"position_y"`
	Name      string         `json:"name"       validate:"required,min=1"`
	Enabled   bool           `json:"enabled"`
}

// UpdateNodeRequest represents the request body for updating an existing workflow node.
// Type and Category cannot be changed, only config, position, name, and enabled status.
type UpdateNodeRequest struct {
	Config    map[string]any `json:"config"`
	PositionX int            `json:"position_x"`
	PositionY int            `json:"position_y"`
	Name      string         `json:"name"       validate:"required,min=1"`
	Enabled   bool           `json:"enabled"`
}
