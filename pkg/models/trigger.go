package models

type WorkflowTrigger struct {
	ID          string `json:"id"          validate:"required"`
	Name        string `json:"name"        validate:"required,min=3"`
	Description string `json:"description" validate:"required"`
	SourceID    string `json:"source_id"`
	EventType   string `json:"event_type"  validate:"required"`
	ProviderID  string `json:"provider_id" validate:"required"`

	// TriggerID will be deprecated, the new way of defining
	// a WorkflowTrigger is by defining its Source
	TriggerID string `json:"trigger_id"`

	Configuration map[string]any `json:"configuration"`
}
