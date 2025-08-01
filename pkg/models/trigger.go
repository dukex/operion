package models

type WorkflowTrigger struct {
	ID            string         `json:"id" validate:"required"`
	Name          string         `json:"name" validate:"required,min=3"`
	Description   string         `json:"description" validate:"required"`
	TriggerID     string         `json:"trigger_id" validate:"required"`
	Configuration map[string]any `json:"configuration"`
}
