package models

// SchemaProvider defines an interface for components that can provide JSON Schema
type SchemaProvider interface {
	GetSchema() *JSONSchema
}

// JSONSchema represents a JSON Schema for configuration validation
type JSONSchema struct {
	Type        string               `json:"type"`
	Properties  map[string]*Property `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
}

// Property represents a JSON Schema property
type Property struct {
	Type        string               `json:"type"`
	Description string               `json:"description,omitempty"`
	Enum        []any                `json:"enum,omitempty"`
	Default     any                  `json:"default,omitempty"`
	Format      string               `json:"format,omitempty"`
	MinLength   *int                 `json:"minLength,omitempty"`
	MaxLength   *int                 `json:"maxLength,omitempty"`
	Pattern     string               `json:"pattern,omitempty"`
	Items       *Property            `json:"items,omitempty"`
	Properties  map[string]*Property `json:"properties,omitempty"`
	Required    []string             `json:"required,omitempty"`
}

// RegisteredComponent represents a component registered in the system with metadata
type RegisteredComponent struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Schema      *JSONSchema `json:"schema"`
}
