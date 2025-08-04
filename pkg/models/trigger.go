package models

type WorkflowTrigger struct {
	ID          string `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required,min=3"`
	Description string `json:"description" validate:"required"`
	SourceID    string `json:"source_id" validate:"required"`

	// Deprecated: TriggerID is deprecated, the new way of defining
	// a WorkflowTrigger is by defining its Source
	TriggerID string `json:"trigger_id"`

	Configuration map[string]any `json:"configuration"`
}
