package models

type WorkflowTrigger struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name" validate:"required,min=3"`
	Description   string                 `json:"description" validate:"required"`
	TriggerID     string                 `json:"trigger_id"`
	Configuration map[string]interface{} `json:"configuration"`
}
