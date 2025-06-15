package domain

import "time"

type WorkflowStatus string

type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name" validate:"required,min=3"`
	Description string                 `json:"description" validate:"required"`
	Triggers    []TriggerItem              `json:"triggers"`
	Steps       []WorkflowStep         `json:"steps"`
	Variables   map[string]interface{} `json:"variables"`
	Status      WorkflowStatus         `json:"status"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Owner       string                 `json:"owner"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
